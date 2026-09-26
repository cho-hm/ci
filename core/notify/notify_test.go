package notify

import (
	"ci/core/constant"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

func sampleEvent(kind Kind, outcome Outcome) Event {
	return Event{
		Kind:       kind,
		Outcome:    outcome,
		Workflow:   "CI",
		Repository: "owner/repo",
		RefName:    "master",
		Sha:        "0123456789abcdef0123456789abcdef01234567",
		Actor:      "someone",
		RunURL:     "https://github.com/owner/repo/actions/runs/1",
		Elapsed:    90 * time.Second,
		Phases: []constant.PhaseResult{
			{Phase: "publish", Status: constant.PhaseSuccess},
			{Phase: "build", Status: constant.PhaseFailure, Cause: errors.New("docker exited")},
		},
	}
}

func mustURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("invalid test url %q: %v", raw, err)
	}
	return u
}

// 알려진 플랫폼 URL 은 해당 Notifier 로, 그 외는 json 폴백으로 판별되어야 한다.
func TestResolve_DetectsPlatformFromURL(t *testing.T) {
	cases := map[string]string{
		"https://discord.com/api/webhooks/1/abc":                             "discord",
		"https://hooks.slack.com/services/T/B/X":                             "slack",
		"https://prod-00.westus.logic.azure.com/workflows/x/triggers/manual": "teams",
		"https://chat.googleapis.com/v1/spaces/X/messages?key=k&token=t":     "googlechat",
		"https://api.telegram.org/bot123:abc/sendMessage?chat_id=42":         "telegram",
		"https://example.internal/hooks/ci":                                  "json",
	}
	for raw, want := range cases {
		n, err := Resolve("", mustURL(t, raw))
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", raw, err)
		}
		if n.Name() != want {
			t.Errorf("%s: detected %q, want %q", raw, n.Name(), want)
		}
	}
}

// 명시된 WEBHOOK_TYPE 은 URL 판별보다 우선하며, 대소문자/공백에 관대해야 한다.
func TestResolve_ExplicitTypeOverridesDetection(t *testing.T) {
	endpoint := mustURL(t, "https://discord.com/api/webhooks/1/abc")
	n, err := Resolve(" JSON ", endpoint)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n.Name() != "json" {
		t.Errorf("explicit type ignored: got %q", n.Name())
	}
}

func TestResolve_UnknownTypeIsError(t *testing.T) {
	if _, err := Resolve("no-such-platform", mustURL(t, "https://example.com")); err == nil {
		t.Error("expected error for unsupported webhook type")
	}
}

// 모든 등록된 Notifier 는 모든 이벤트 종류에 대해 JSON 직렬화 가능한 본문을 만들어야 하며,
// 본문에 저장소 이름이 드러나야 한다.
func TestFormat_AllNotifiersProduceSerializablePayload(t *testing.T) {
	endpoint := mustURL(t, "https://api.telegram.org/bot1:x/sendMessage?chat_id=42")
	events := []Event{sampleEvent(Started, Running), sampleEvent(Finished, Succeeded), sampleEvent(Finished, Failed)}
	for name, n := range registry {
		for _, e := range events {
			payload, err := n.Format(e, endpoint)
			if err != nil {
				t.Fatalf("%s/%s: format error: %v", name, e.Outcome, err)
			}
			body, err := json.Marshal(payload)
			if err != nil {
				t.Fatalf("%s/%s: marshal error: %v", name, e.Outcome, err)
			}
			if !strings.Contains(string(body), e.Repository) {
				t.Errorf("%s/%s: payload does not mention repository", name, e.Outcome)
			}
		}
	}
}

func TestTelegram_RequiresChatID(t *testing.T) {
	_, err := Telegram{}.Format(sampleEvent(Started, Running), mustURL(t, "https://api.telegram.org/bot1:x/sendMessage"))
	if err == nil {
		t.Error("expected error when chat_id is missing")
	}
}

// json 형식은 외부 계약이므로 상태 필드가 이벤트와 일치해야 한다.
func TestJSON_StatusReflectsOutcome(t *testing.T) {
	for _, outcome := range []Outcome{Running, Succeeded, Failed} {
		payload, _ := JSON{}.Format(sampleEvent(Finished, outcome), nil)
		body, _ := json.Marshal(payload)
		var decoded struct {
			Status string `json:"status"`
			Phases []struct {
				Status string `json:"status"`
			} `json:"phases"`
		}
		if err := json.Unmarshal(body, &decoded); err != nil {
			t.Fatalf("decode error: %v", err)
		}
		if decoded.Status != outcome.String() {
			t.Errorf("status = %q, want %q", decoded.Status, outcome.String())
		}
		if len(decoded.Phases) != len(sampleEvent(Finished, outcome).Phases) {
			t.Errorf("phases not carried over")
		}
	}
}

type recorder struct {
	mu     sync.Mutex
	bodies []string
}

func (r *recorder) handler(status int) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		body, _ := io.ReadAll(req.Body)
		r.mu.Lock()
		r.bodies = append(r.bodies, string(body))
		r.mu.Unlock()
		w.WriteHeader(status)
	}
}

func TestNew_WithoutURLSendsNothing(t *testing.T) {
	if _, ok := New(Config{}).(noopSender); !ok {
		t.Error("expected noop sender when WEBHOOK_URL is empty")
	}
}

func TestNew_UnknownTypeSendsNothing(t *testing.T) {
	if _, ok := New(Config{WebhookURL: "https://example.com/hook", WebhookType: "nope"}).(noopSender); !ok {
		t.Error("expected noop sender for unsupported type")
	}
}

func TestSender_PostsFormattedPayload(t *testing.T) {
	rec := &recorder{}
	server := httptest.NewServer(rec.handler(http.StatusNoContent))
	defer server.Close()

	New(Config{WebhookURL: server.URL, WebhookType: "json"}).Send(sampleEvent(Started, Running))

	if len(rec.bodies) == 0 {
		t.Fatal("webhook was not called")
	}
	if !json.Valid([]byte(rec.bodies[0])) {
		t.Error("request body is not valid json")
	}
}

// 수신측 오류나 네트워크 오류가 나도 Send 는 panic 하지 않고, 로그에 웹후크 URL 을 남기지 않아야 한다.
func TestSender_FailureIsSwallowedAndURLRedacted(t *testing.T) {
	var logs strings.Builder
	log.SetOutput(&logs)
	defer log.SetOutput(io.Discard)

	server := httptest.NewServer((&recorder{}).handler(http.StatusInternalServerError))
	secretPath := "/hooks/super-secret-token"
	New(Config{WebhookURL: server.URL + secretPath, WebhookType: "json"}).Send(sampleEvent(Finished, Failed))
	server.Close()
	// 서버 종료 후 전송 → 네트워크 오류 경로
	New(Config{WebhookURL: server.URL + secretPath, WebhookType: "json"}).Send(sampleEvent(Finished, Failed))

	if !strings.Contains(logs.String(), "WARN") {
		t.Error("expected failure to be reported as warning")
	}
	if strings.Contains(logs.String(), "super-secret-token") {
		t.Error("webhook url leaked into logs")
	}
}
