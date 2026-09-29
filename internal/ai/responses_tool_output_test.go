package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type responsesBatchClient struct {
	started  bool
	endpoint string
}

func (*responsesBatchClient) Models(context.Context, Provider, string) ([]Model, error) {
	return nil, nil
}

func (c *responsesBatchClient) Stream(ctx context.Context, provider Provider, key string, request CompletionRequest, emit func(CompletionEvent) error) error {
	if strings.HasPrefix(request.System, "你负责 KPanel 后台学习评估") {
		return emit(CompletionEvent{Delta: `{"decision":"skip"}`, Done: true})
	}
	if !c.started {
		c.started = true
		parts := map[int]*ToolCall{
			0: {ID: "call_first", Name: "host_action", Arguments: json.RawMessage(`{"target":"first"}`)},
			1: {ID: "call_second", Name: "host_action", Arguments: json.RawMessage(`{"target":"second"}`)},
		}
		items := []json.RawMessage{
			json.RawMessage(`{"type":"reasoning","id":"rs_batch","encrypted_content":"opaque","summary":[]}`),
			json.RawMessage(`{"type":"function_call","id":"fc_first","call_id":"call_first","name":"host_action","arguments":"{\"target\":\"first\"}"}`),
			json.RawMessage(`{"type":"function_call","id":"fc_second","call_id":"call_second","name":"host_action","arguments":"{\"target\":\"second\"}"}`),
		}
		return emit(responsesCompletionEvent(provider.ID, request.Model, parts, items, true, Usage{}))
	}
	provider.Protocol, provider.APIMode = ProtocolOpenAICompatible, OpenAIResponses
	provider.BaseURL, provider.EndpointScope = c.endpoint, EndpointPrivate
	return NewHTTPModelClient().Stream(ctx, provider, key, request, emit)
}

func TestResponsesIncompleteToolBatchCanContinue(t *testing.T) {
	for _, mode := range []string{"approve", "reject", "resource-conflict", "all-completed"} {
		t.Run(mode, func(t *testing.T) {
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var body struct {
					Input []map[string]any `json:"input"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
					w.WriteHeader(500)
					return
				}
				requests++
				calls, outputs := map[string]int{}, map[string][]string{}
				for _, item := range body.Input {
					id, _ := item["call_id"].(string)
					if item["type"] == "function_call" {
						calls[id]++
					}
					if item["type"] == "function_call_output" {
						output, _ := item["output"].(string)
						outputs[id] = append(outputs[id], output)
					}
				}
				for _, id := range []string{"call_first", "call_second"} {
					if calls[id] != 1 || len(outputs[id]) != 1 {
						t.Errorf("request %d: call %s count=%d outputs=%v", requests, id, calls[id], outputs[id])
						w.WriteHeader(400)
						fmt.Fprint(w, `{"error":{"message":"No tool output found for function call"}}`)
						return
					}
				}
				first, second := outputs["call_first"][0], outputs["call_second"][0]
				if strings.Contains(first, "tool_output_unavailable") {
					t.Error("recorded first result was replaced")
				}
				if mode == "all-completed" {
					if !strings.Contains(second, `"status":"ok"`) {
						t.Errorf("completed result changed: %s", second)
					}
				} else if !strings.Contains(second, "tool_output_unavailable") || !strings.Contains(second, "unknown") {
					t.Errorf("missing result must be explicitly unknown: %s", second)
				}
				w.Header().Set("Content-Type", "text/event-stream")
				fmt.Fprint(w, "data: {\"type\":\"response.output_text.delta\",\"delta\":\"继续核对实际状态\"}\n\ndata: {\"type\":\"response.completed\",\"response\":{}}\n\n")
			}))
			defer server.Close()
			s, providers, provider, model := runtimeFixture(t)
			defer s.Close()
			ctx := context.Background()
			session, err := s.CreateSession(ctx, Session{UserID: "admin", ProviderID: provider.ID, ModelID: model.ID})
			if err != nil {
				t.Fatal(err)
			}
			newRun := func(content string) Run {
				run, err := s.CreateRun(ctx, Run{SessionID: session.ID, UserID: "admin", ProviderID: provider.ID, ModelID: model.ID, ApprovalMode: ApprovalManual})
				if err != nil {
					t.Fatal(err)
				}
				if _, err := s.AddMessage(ctx, Message{SessionID: session.ID, RunID: run.ID, Role: RoleUser, Content: content}); err != nil {
					t.Fatal(err)
				}
				return run
			}
			run := newRun("处理两个测试资源")
			tools := &fakeTools{}
			if mode == "resource-conflict" || mode == "all-completed" {
				tools.readOnly = true
			}
			if mode == "resource-conflict" {
				tools.executeErr = ErrToolConflict
			}
			runtime, err := NewNativeRuntime(s, providers, &responsesBatchClient{endpoint: server.URL}, tools, NewEventHub())
			if err != nil {
				t.Fatal(err)
			}
			defer runtime.Close()
			if err := runtime.Run(ctx, run.ID); err != nil {
				t.Fatal(err)
			}
			if mode == "approve" || mode == "reject" {
				if tools.executed != 0 {
					t.Fatal("tool executed before approval")
				}
				if err := runtime.Resume(ctx, run.ID, Decision{ToolCallID: "call_first", Approve: mode == "approve"}); err != nil {
					t.Fatal(err)
				}
			}
			// Old partial batches stay in session history: a later user turn must also work.
			next := newRun("继续")
			if err := runtime.Run(ctx, next.ID); err != nil {
				t.Fatal(err)
			}
			wantExecuted := map[string]int{"approve": 1, "reject": 0, "resource-conflict": 1, "all-completed": 2}[mode]
			if tools.executed != wantExecuted || requests != 2 {
				t.Fatalf("executed=%d want=%d provider requests=%d want=2", tools.executed, wantExecuted, requests)
			}
			for _, id := range []string{run.ID, next.ID} {
				stored, err := s.RunByID(ctx, id)
				if err != nil || stored.Status != RunCompleted {
					t.Fatalf("run=%s status=%s err=%v", id, stored.Status, err)
				}
			}
		})
	}
}

func TestResponsesNativeBatchReplayReconciliation(t *testing.T) {
	for _, mode := range []string{"missing", "missing-with-ids", "complete", "split", "reversed", "other-model", "other-provider"} {
		t.Run(mode, func(t *testing.T) {
			items := []json.RawMessage{
				json.RawMessage(`{"type":"reasoning","id":"rs_batch","encrypted_content":"opaque","summary":[]}`),
				json.RawMessage(`{"type":"function_call","id":"fc_first","call_id":"call_first","name":"host_action","arguments":"{}"}`),
				json.RawMessage(`{"type":"function_call","id":"fc_second","call_id":"call_second","name":"host_action","arguments":"{}"}`),
			}
			raw, err := json.Marshal(responsesNativeContext{Type: "openai_responses_output", ProviderID: "provider", Model: "model", Complete: true, Items: items})
			if err != nil {
				t.Fatal(err)
			}
			first := ToolCall{ID: "call_first", Name: "host_action", Arguments: json.RawMessage(`{}`), ProviderData: raw}
			second := ToolCall{ID: "call_second", Name: "host_action", Arguments: json.RawMessage(`{}`)}
			messages := []ChatMessage{
				{Role: "assistant", ToolCalls: []ToolCall{first}},
				{Role: "tool", ToolCallID: first.ID, Content: "first real output"},
			}
			secondResult := ChatMessage{Role: "tool", ToolCallID: second.ID, Content: "second real output"}
			switch mode {
			case "complete":
				messages[0].ToolCalls = append(messages[0].ToolCalls, second)
				messages = append(messages, secondResult)
			case "split", "reversed":
				other := []ChatMessage{{Role: "assistant", ToolCalls: []ToolCall{second}}, secondResult}
				if mode == "split" {
					messages = append(messages, other...)
				} else {
					messages = append(other, messages...)
				}
			}
			foreign := mode == "other-model" || mode == "other-provider"
			missing := mode == "missing" || mode == "missing-with-ids"
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var body struct {
					Input []map[string]any `json:"input"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				calls, outputs, reasoning := map[string]int{}, map[string]string{}, 0
				for _, item := range body.Input {
					id, _ := item["call_id"].(string)
					switch item["type"] {
					case "reasoning":
						reasoning++
						if item["encrypted_content"] != "opaque" {
							t.Error("native reasoning changed")
						}
					case "function_call":
						calls[id]++
					case "function_call_output":
						if _, exists := outputs[id]; exists {
							t.Errorf("duplicate output for %s", id)
						}
						outputs[id], _ = item["output"].(string)
						if mode == "missing-with-ids" && item["id"] == nil {
							t.Errorf("compatibility output lacks id: %#v", item)
						}
					}
				}
				wantCalls, wantReasoning := 2, 1
				if foreign {
					wantCalls, wantReasoning = 1, 0
				}
				if len(calls) != wantCalls || len(outputs) != wantCalls || reasoning != wantReasoning {
					t.Errorf("calls=%v outputs=%v reasoning=%d", calls, outputs, reasoning)
				}
				for id, count := range calls {
					if count != 1 || outputs[id] == "" {
						t.Errorf("unpaired or duplicate call: %s count=%d", id, count)
					}
				}
				if outputs[first.ID] != "first real output" {
					t.Error("recorded result changed")
				}
				if missing && outputs[second.ID] != responsesMissingToolOutput {
					t.Error("missing result must be marked unknown")
				}
				if !foreign && !missing && outputs[second.ID] != "second real output" {
					t.Error("later recorded result replaced by unknown marker")
				}
				w.Header().Set("Content-Type", "text/event-stream")
				fmt.Fprint(w, "data: {\"type\":\"response.completed\",\"response\":{}}\n\n")
			}))
			defer server.Close()
			provider := Provider{ID: "provider", Protocol: ProtocolOpenAICompatible, APIMode: OpenAIResponses, BaseURL: server.URL, EndpointScope: EndpointPrivate}
			request := CompletionRequest{Model: "model", Messages: messages}
			if mode == "other-model" {
				request.Model = "other"
			}
			if mode == "other-provider" {
				provider.ID = "other"
			}
			if err := NewHTTPModelClient().streamOpenAIResponsesAttempt(context.Background(), provider, "key", request, mode == "missing-with-ids", func(CompletionEvent) error { return nil }); err != nil {
				t.Fatal(err)
			}
		})
	}
}
