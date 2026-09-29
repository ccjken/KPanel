package backupremote

import (
	"context"
	"encoding/xml"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/kejilion/kejilion-panel/internal/backup"
)

func (c *Client) davURL(name string) string {
	u, _ := url.Parse(c.storage.Endpoint)
	u.Path = path.Join(u.Path, c.storage.Prefix, name)
	if name == "" {
		u.Path = strings.TrimRight(u.Path, "/") + "/"
	}
	return u.String()
}
func (c *Client) davRequest(ctx context.Context, method, name string, body io.Reader, size int64, headers http.Header) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.davURL(name), body)
	if err != nil {
		return nil, ErrInvalid
	}
	req.ContentLength = size
	req.SetBasicAuth(c.storage.Username, c.storage.Secret)
	req.Header.Set("Accept-Encoding", "identity")
	for k, vs := range headers {
		req.Header[k] = vs
	}
	res, err := c.http.Do(req)
	if err != nil {
		return nil, ErrUnavailable
	}
	return res, nil
}
func (c *Client) davPut(ctx context.Context, name string, reader io.Reader, size int64) error {
	// The configured DAV service root must exist. Create only the selected
	// backup prefix, never a caller-provided hierarchy outside that root.
	if c.storage.Prefix != "" {
		copyClient := *c
		copyClient.storage.Prefix = ""
		prefix := ""
		for _, part := range strings.Split(c.storage.Prefix, "/") {
			if part == "" {
				continue
			}
			prefix = path.Join(prefix, part)
			res, err := copyClient.davRequest(ctx, "MKCOL", prefix, nil, 0, nil)
			if err != nil {
				return err
			}
			res.Body.Close()
			if res.StatusCode != 201 && res.StatusCode != 405 {
				return ErrUnavailable
			}
		}
	}
	tmp := name + ".partial-" + backup.NewID()
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = c.remove(cleanup, tmp)
	}()
	res, err := c.davRequest(ctx, "PUT", tmp, reader, size, http.Header{"Content-Type": []string{"application/octet-stream"}})
	if err != nil {
		return err
	}
	res.Body.Close()
	if res.StatusCode != 200 && res.StatusCode != 201 && res.StatusCode != 204 {
		return ErrUnavailable
	}
	res, err = c.davRequest(ctx, "MOVE", tmp, nil, 0, http.Header{"Destination": []string{c.davURL(name)}, "Overwrite": []string{"T"}})
	if err != nil {
		return err
	}
	res.Body.Close()
	if res.StatusCode != 201 && res.StatusCode != 204 {
		return ErrUnavailable
	}
	return nil
}

type davResponse struct {
	Href  string `xml:"href"`
	Props []struct {
		Status string `xml:"status"`
		Prop   struct {
			Length   string `xml:"getcontentlength"`
			Modified string `xml:"getlastmodified"`
			Type     struct {
				Collection *struct{} `xml:"collection"`
			} `xml:"resourcetype"`
		} `xml:"prop"`
	} `xml:"propstat"`
}

func (c *Client) davList(ctx context.Context) ([]Object, error) {
	const body = `<?xml version="1.0"?><d:propfind xmlns:d="DAV:"><d:prop><d:getcontentlength/><d:getlastmodified/><d:resourcetype/></d:prop></d:propfind>`
	res, err := c.davRequest(ctx, "PROPFIND", "", strings.NewReader(body), int64(len(body)), http.Header{"Depth": []string{"1"}, "Content-Type": []string{"application/xml"}})
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != 207 {
		return nil, ErrUnavailable
	}
	limit := &io.LimitedReader{R: res.Body, N: (4 << 20) + 1}
	decoder := xml.NewDecoder(limit)
	base, _ := url.Parse(c.davURL(""))
	out := []Object{}
	count := 0
	seen := map[string]bool{}
	for {
		token, err := decoder.Token()
		if limit.N <= 0 {
			return nil, ErrLimit
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, ErrUnavailable
		}
		start, ok := token.(xml.StartElement)
		if !ok || start.Name.Local != "response" {
			continue
		}
		count++
		if count > MaxObjects+1 {
			return nil, ErrLimit
		}
		var entry davResponse
		if decoder.DecodeElement(&entry, &start) != nil {
			return nil, ErrUnavailable
		}
		u, err := url.Parse(entry.Href)
		if err != nil {
			return nil, ErrUnavailable
		}
		u = base.ResolveReference(u)
		if u.Scheme == base.Scheme && u.Host == base.Host && u.Path == strings.TrimRight(base.Path, "/") {
			continue
		}
		if u.Scheme != base.Scheme || u.Host != base.Host || u.RawQuery != "" || u.Fragment != "" || !strings.HasPrefix(u.Path, base.Path) {
			return nil, ErrUnavailable
		}
		name := strings.TrimPrefix(u.Path, base.Path)
		if !ValidObject(name) || seen[name] {
			continue
		}
		for _, p := range entry.Props {
			if !strings.Contains(p.Status, " 200 ") || p.Prop.Type.Collection != nil {
				continue
			}
			size, e := strconv.ParseInt(p.Prop.Length, 10, 64)
			if e != nil || size <= 0 || size > backup.MaxEncryptedBytes {
				continue
			}
			modified, _ := http.ParseTime(p.Prop.Modified)
			out = append(out, Object{Key: name, Size: size, Modified: modified})
			seen[name] = true
			break
		}
	}
	return out, nil
}
