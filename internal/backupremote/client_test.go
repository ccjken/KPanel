package backupremote

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"github.com/kejilion/kejilion-panel/internal/backup"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// Both adapters speak their real wire protocols to a bounded local fixture.
// Production cannot inject this transport or connect to loopback.
func remoteFixture(t *testing.T, kind string, denyMove bool) (*Client, *sync.Map) {
	t.Helper()
	objects := &sync.Map{}
	parts := &sync.Map{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if kind == "webdav" {
			user, pass, ok := r.BasicAuth()
			if !ok || user != "backup" || pass != "secret-never-returned" {
				t.Error("DAV authorization")
				w.WriteHeader(401)
				return
			}
		} else if !strings.HasPrefix(r.Header.Get("Authorization"), "AWS4-HMAC-SHA256 ") {
			t.Error("S3 not signed")
			w.WriteHeader(401)
			return
		}
		key := r.URL.Path
		switch r.Method {
		case "POST":
			if r.URL.Query().Has("uploads") {
				fmt.Fprint(w, `<InitiateMultipartUploadResult><Bucket>bucket</Bucket><Key>kpanel/large.kpb</Key><UploadId>fixture-upload</UploadId></InitiateMultipartUploadResult>`)
			} else if r.URL.Query().Get("uploadId") == "fixture-upload" {
				var completed struct {
					Parts []struct {
						Number string `xml:"PartNumber"`
					} `xml:"Part"`
				}
				if xml.NewDecoder(r.Body).Decode(&completed) != nil {
					t.Error("multipart completion XML")
				}
				var data []byte
				for _, part := range completed.Parts {
					v, ok := parts.LoadAndDelete(part.Number)
					if !ok {
						t.Error("missing uploaded part", part.Number)
						return
					}
					data = append(data, v.([]byte)...)
				}
				objects.Store(key, data)
				fmt.Fprint(w, `<CompleteMultipartUploadResult><Bucket>bucket</Bucket><Key>kpanel/large.kpb</Key><ETag>"fixture-2"</ETag></CompleteMultipartUploadResult>`)
			} else {
				t.Error("unexpected multipart operation")
				w.WriteHeader(400)
			}
		case "MKCOL":
			w.WriteHeader(201)
		case "PUT":
			var reader io.Reader = r.Body
			if r.Header.Get("Content-Encoding") == "aws-chunked" {
				reader = httputil.NewChunkedReader(r.Body)
			}
			b, err := io.ReadAll(io.LimitReader(reader, 32<<20))
			if err != nil {
				t.Error(err)
			}
			if part := r.URL.Query().Get("partNumber"); part != "" {
				parts.Store(part, b)
			} else {
				objects.Store(key, b)
			}
			w.Header().Set("ETag", `"fixture"`)
			w.WriteHeader(200)
		case "MOVE":
			if denyMove {
				w.WriteHeader(403)
				return
			}
			u, _ := url.Parse(r.Header.Get("Destination"))
			v, ok := objects.Load(key)
			if !ok {
				w.WriteHeader(404)
				return
			}
			objects.Store(u.Path, v)
			objects.Delete(key)
			w.WriteHeader(201)
		case "DELETE":
			if r.URL.Query().Get("uploadId") != "" {
				parts.Clear()
			}
			objects.Delete(key)
			w.WriteHeader(204)
		case "HEAD":
			v, ok := objects.Load(key)
			if !ok {
				w.WriteHeader(404)
				return
			}
			w.Header().Set("Content-Length", fmt.Sprint(len(v.([]byte))))
			w.Header().Set("ETag", `"fixture"`)
			w.Header().Set("Last-Modified", "Mon, 28 Sep 2026 03:00:00 GMT")
		case "PROPFIND":
			if r.Header.Get("Depth") != "1" {
				t.Error("DAV depth")
			}
			w.WriteHeader(207)
			fmt.Fprint(w, `<d:multistatus xmlns:d="DAV:"><d:response><d:href>/bucket/kpanel</d:href><d:propstat><d:prop><d:resourcetype><d:collection/></d:resourcetype></d:prop><d:status>HTTP/1.1 200 OK</d:status></d:propstat></d:response>`)
			objects.Range(func(k, v any) bool {
				fmt.Fprintf(w, `<d:response><d:href>%s</d:href><d:propstat><d:prop><d:getcontentlength>%d</d:getcontentlength></d:prop><d:status>HTTP/1.1 200 OK</d:status></d:propstat></d:response>`, k, len(v.([]byte)))
				return true
			})
			fmt.Fprint(w, `</d:multistatus>`)
		case "GET":
			if r.URL.Query().Get("list-type") == "2" {
				fmt.Fprint(w, `<ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><Name>bucket</Name><IsTruncated>false</IsTruncated>`)
				objects.Range(func(k, v any) bool {
					fmt.Fprintf(w, `<Contents><Key>%s</Key><Size>%d</Size><LastModified>2026-09-28T03:00:00.000Z</LastModified></Contents>`, strings.TrimPrefix(k.(string), "/bucket/"), len(v.([]byte)))
					return true
				})
				fmt.Fprint(w, `</ListBucketResult>`)
				return
			}
			v, ok := objects.Load(key)
			if !ok {
				w.WriteHeader(404)
				return
			}
			w.Header().Set("Content-Length", fmt.Sprint(len(v.([]byte))))
			w.Header().Set("ETag", `"fixture"`)
			w.Header().Set("Last-Modified", "Mon, 28 Sep 2026 03:00:00 GMT")
			w.Write(v.([]byte))
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL)
			w.WriteHeader(400)
		}
	}))
	t.Cleanup(server.Close)
	storage := testStorage()
	storage.Kind = kind
	storage.Endpoint = server.URL + "/bucket"
	if kind == "s3" {
		storage.Endpoint = server.URL
		storage.Bucket = "bucket"
		storage.PathStyle = true
		storage.AccessKey = "fixture-key"
	}
	client, err := newClient(storage, server.Client().Transport)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(client.Close)
	return client, objects
}

func TestRemoteAdaptersEncryptedRoundTrip(t *testing.T) {
	for _, kind := range []string{"s3", "webdav"} {
		t.Run(kind, func(t *testing.T) {
			client, objects := remoteFixture(t, kind, false)
			ctx := context.Background()
			if err := client.Test(ctx); err != nil {
				t.Fatal("connection test", err)
			}
			objects.Range(func(k, v any) bool { t.Error("test object leaked", k); return true })
			dir := t.TempDir()
			payload := filepath.Join(dir, "panel.payload")
			os.WriteFile(payload, []byte(`{"fixture":true}`), 0600)
			f, err := os.Create(filepath.Join(dir, "source.kpb"))
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()
			if _, err = backup.Write(f, "fixture-password", "test", []backup.Source{{Module: "panel", Path: payload}}); err != nil {
				t.Fatal(err)
			}
			size, _ := f.Seek(0, io.SeekEnd)
			f.Seek(0, 0)
			if err = client.Upload(ctx, "archive.kpb", f, size); err != nil {
				t.Fatal("upload", err)
			}
			list, err := client.List(ctx)
			if err != nil || len(list) != 1 || list[0].Key != "archive.kpb" {
				t.Fatal("list", list, err)
			}
			target := filepath.Join(dir, "download.kpb")
			if _, err = client.Download(ctx, "archive.kpb", target); err != nil {
				t.Fatal("download", err)
			}
			imported, err := os.Open(target)
			if err != nil {
				t.Fatal(err)
			}
			defer imported.Close()
			if _, err = backup.Read(imported, "fixture-password", t.TempDir()); err != nil {
				t.Fatal("restore compatibility", err)
			}
			if err = client.Delete(ctx, "archive.kpb"); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestWebDAVFailedMoveCleansPartial(t *testing.T) {
	client, objects := remoteFixture(t, "webdav", true)
	if client.put(context.Background(), "archive.kpb", bytes.NewReader([]byte("abc")), 3) == nil {
		t.Fatal("denied move succeeded")
	}
	objects.Range(func(k, v any) bool { t.Error("partial leaked", k); return true })
}
func TestRemoteNetworkBoundaries(t *testing.T) {
	for _, address := range []string{"127.0.0.1", "::1", "169.254.169.254", "100.100.100.200", "168.63.129.16", "fd00:ec2::254", "::ffff:127.0.0.1", "64:ff9b::7f00:1", "2002:7f00:1::", "224.0.0.1", "0.0.0.0"} {
		if allowedIP(netip.MustParseAddr(address)) {
			t.Error("unsafe address", address)
		}
	}
	for _, address := range []string{"192.168.1.5", "10.0.0.4", "8.8.8.8", "fd00::5"} {
		if !allowedIP(netip.MustParseAddr(address)) {
			t.Error("valid target rejected", address)
		}
	}
	client, _ := remoteFixture(t, "webdav", false)
	if _, err := client.http.Get("http://unrelated.invalid/secret"); err == nil {
		t.Fatal("cross-origin credentials")
	}
	storage := testStorage()
	storage.Endpoint = "http://127.0.0.1:1"
	prod, _ := NewClient(storage)
	defer prod.Close()
	if prod.Test(context.Background()) == nil {
		t.Fatal("production loopback allowed")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestS3ConfiguredEndpointNormalization(t *testing.T) {
	for _, endpoint := range []string{"https://s3.amazonaws.com", "https://s3.us-east-1.amazonaws.com", "https://s3.dualstack.us-east-1.amazonaws.com", "https://storage.example:443", "http://storage.example:80"} {
		t.Run(endpoint, func(t *testing.T) {
			calls := 0
			storage := testStorage()
			storage.Kind = "s3"
			storage.Endpoint = endpoint
			storage.Bucket = "bucket"
			storage.AccessKey = "fixture"
			client, err := newClient(storage, roundTripFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`<ListBucketResult><IsTruncated>false</IsTruncated></ListBucketResult>`))}, nil
			}))
			if err != nil {
				t.Fatal(err)
			}
			defer client.Close()
			if _, err = client.List(context.Background()); err != nil || calls != 1 {
				t.Fatal("configured SDK endpoint blocked", calls, err)
			}
			if _, err = client.http.Get("https://attacker.amazonaws.com/"); err == nil {
				t.Fatal("unconfigured host allowed")
			}
		})
	}
}
func TestS3MultipartStreaming(t *testing.T) {
	client, objects := remoteFixture(t, "s3", false)
	data := bytes.Repeat([]byte("encrypted-backup-fixture"), 800000)
	f, err := os.Create(filepath.Join(t.TempDir(), "large.kpb"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	f.Write(data)
	f.Seek(0, 0)
	if err = client.Upload(context.Background(), "large.kpb", f, int64(len(data))); err != nil {
		t.Fatal(err)
	}
	got, ok := objects.Load("/bucket/kpanel/large.kpb")
	if !ok || !bytes.Equal(got.([]byte), data) {
		t.Fatal("multipart changed the package")
	}
}
func TestWebDAVRejectsInvalidListingAndTruncatedDownload(t *testing.T) {
	storage := testStorage()
	client, err := newClient(storage, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		res := &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("abc")), ContentLength: 3}
		if r.Method == "HEAD" {
			res.ContentLength = 4
		}
		if r.Method == "PROPFIND" {
			res.StatusCode = 207
			res.Body = io.NopCloser(strings.NewReader("<html>login</html>"))
		}
		return res, nil
	}))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if _, err = client.List(context.Background()); err == nil {
		t.Fatal("invalid DAV document accepted")
	}
	target := filepath.Join(t.TempDir(), "upload.kpb")
	if _, err = client.Download(context.Background(), "archive.kpb", target); err == nil {
		t.Fatal("short download accepted")
	}
	if _, err = os.Stat(target); !os.IsNotExist(err) {
		t.Fatal("partial download retained")
	}
}
