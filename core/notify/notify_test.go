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
		Kind:          kind,
		Outcome:       outcome,
		Workflow:      "CI",
		ServerURL:     "https://github.com",
		Repository:    "owner/repo",
		RefName:       "v1.2.0",
		RefType:       "tag",
		EventName:     "push",
		Sha:           "0123456789abcdef0123456789abcdef01234567",
		CommitMessage: "feat: <payment> & more",
		Actor:         "someone",
		RunURL:        "https://github.com/owner/repo/actions/runs/1",
		RunNumber:     "42",
		RunAttempt:    "2",
		Env:           "gradle",
		Tasks:         []string{"publish", "build"},
		Runner:        "Linux X64",
		Version:       "v9.9.9",
		Elapsed:       192 * time.Second,
		Phases: []constant.PhaseResult{
			{Phase: "publish", Status: constant.PhaseSuccess, Elapsed: 48 * time.Second},
			{Phase: "build", Status: constant.PhaseFailure, Cause: errors.New("docker exited"), Elapsed: 24 * time.Second,
				Artifacts: []string{"ghcr.io/owner/repo:v1.2.0"}},
			{Phase: "extra", Status: constant.PhaseSkipped, Reason: "check not passed"},
		},
	}
}

func allEvents() []Event {
	return []Event{sampleEvent(Started, Running), sampleEvent(Finished, Succeeded), sampleEvent(Finished, Failed)}
}

func sampleMessage(e Event, lang string) Message {
	t, _ := textsOf(lang)
	return Message{Event: e, Card: NewCard(e, t)}
}

func mustURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("invalid test url %q: %v", raw, err)
	}
	return u
}

func marshal(t *testing.T, payload any) string {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	return string(body)
}

// ---- 플랫폼 판별 ----

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
	n, err := Resolve(" JSON ", mustURL(t, "https://discord.com/api/webhooks/1/abc"))
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

// ---- Card (공통 표현) ----

// 제목은 상태별로 서로 달라야 하고, 저장소·실행 번호·재시도 정보를 담아야 한다.
func TestCard_TitleCarriesIdentityAndDiffersByOutcome(t *testing.T) {
	seen := map[string]bool{}
	for _, e := range allEvents() {
		title := sampleMessage(e, "").Card.Title
		for _, want := range []string{e.Repository, "#" + e.RunNumber, e.RunAttempt} {
			if !strings.Contains(title, want) {
				t.Errorf("title %q does not contain %q", title, want)
			}
		}
		seen[title] = true
	}
	if len(seen) != len(allEvents()) {
		t.Error("titles are not distinguishable by outcome")
	}
}

// 첫 시도는 재시도 표기를 붙이지 않는다.
func TestCard_FirstAttemptHasNoAttemptSuffix(t *testing.T) {
	e := sampleEvent(Started, Running)
	withRetry := sampleMessage(e, "").Card.Title
	e.RunAttempt = "1"
	firstTry := sampleMessage(e, "").Card.Title
	if len(firstTry) >= len(withRetry) {
		t.Errorf("first attempt title should be shorter than retry title: %q vs %q", firstTry, withRetry)
	}
}

// 종료 카드는 phase 별 결과·결과물·실패 원인·링크를 모두 노출해야 한다.
func TestCard_FinishedExposesPhasesArtifactsErrorsAndLinks(t *testing.T) {
	e := sampleEvent(Finished, Failed)
	c := sampleMessage(e, "").Card
	if len(c.Phases) != len(e.Phases) {
		t.Error("phases not carried over")
	}
	if len(c.Artifacts) == 0 {
		t.Error("artifacts missing")
	}
	if len(c.Errors) == 0 {
		t.Error("failure causes missing")
	}
	if len(c.Links) == 0 || c.TitleURL == "" {
		t.Error("run/commit links missing")
	}
	if c.Commit == nil || c.Commit.URL == "" {
		t.Error("commit view or link missing")
	}
}

// 값이 없는 정보는 카드에 빈 항목으로 남지 않아야 한다.
func TestCard_OmitsEmptyFacts(t *testing.T) {
	c := sampleMessage(Event{Kind: Started, Repository: "owner/repo"}, "").Card
	for _, f := range c.Facts {
		if f.Value == "" {
			t.Errorf("empty fact rendered: %+v", f)
		}
	}
	if c.Commit != nil || len(c.Links) != 0 {
		t.Error("commit/links should be omitted without sha/server url")
	}
}

// 언어 설정에 따라 문구가 달라지고, 지원하지 않는 언어는 기본 언어로 폴백한다.
func TestCard_LanguageSelection(t *testing.T) {
	e := sampleEvent(Finished, Succeeded)
	en := sampleMessage(e, "en").Card.Title
	ko := sampleMessage(e, "ko").Card.Title
	if en == ko {
		t.Error("ko title should differ from en title")
	}
	fallback, ok := textsOf("xx")
	if ok {
		t.Error("unsupported language should be reported")
	}
	if NewCard(e, fallback).Title != en {
		t.Error("unsupported language should fall back to default")
	}
}

// ---- 플랫폼 렌더링 ----

// 모든 Notifier 는 모든 이벤트/언어에 대해 직렬화 가능한 본문을 만들고,
// 저장소 이름과 실행 링크를 드러내야 한다.
func TestFormat_AllNotifiersRenderIdentityAndRunLink(t *testing.T) {
	endpoint := mustURL(t, "https://api.telegram.org/bot1:x/sendMessage?chat_id=42")
	for name, n := range registry {
		for _, lang := range []string{"en", "ko"} {
			for _, e := range allEvents() {
				payload, err := n.Format(sampleMessage(e, lang), endpoint)
				if err != nil {
					t.Fatalf("%s/%s/%s: format error: %v", name, lang, e.Outcome, err)
				}
				body := marshal(t, payload)
				for _, want := range []string{e.Repository, e.RunURL} {
					if !strings.Contains(body, want) {
						t.Errorf("%s/%s/%s: payload does not contain %q", name, lang, e.Outcome, want)
					}
				}
			}
		}
	}
}

// HTML 기반 플랫폼은 사용자 입력(커밋 메시지)을 이스케이프해야 한다.
func TestFormat_HTMLPlatformsEscapeUserText(t *testing.T) {
	endpoint := mustURL(t, "https://api.telegram.org/bot1:x/sendMessage?chat_id=42")
	for _, n := range []Notifier{Telegram{}, GoogleChat{}} {
		e := sampleEvent(Finished, Failed)
		e.Phases = append(e.Phases, constant.PhaseResult{Phase: "fast", Status: constant.PhaseSuccess, Elapsed: 100 * time.Millisecond})
		payload, _ := n.Format(sampleMessage(e, "en"), endpoint)
		body := rawJSON(t, payload)
		if strings.Contains(body, "<1s") {
			t.Errorf("%s: sub-second duration was not escaped", n.Name())
		}
		if strings.Contains(body, "<payment>") {
			t.Errorf("%s: raw user text was not escaped", n.Name())
		}
		if !strings.Contains(body, "&lt;payment&gt;") {
			t.Errorf("%s: escaped user text missing", n.Name())
		}
	}
}

// rawJSON - json.Marshal 의 기본 HTML 이스케이프(\u003c) 없이 직렬화해 실제 전송 텍스트를 검사한다.
func rawJSON(t *testing.T, payload any) string {
	t.Helper()
	var buf strings.Builder
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(payload); err != nil {
		t.Fatalf("encode error: %v", err)
	}
	return buf.String()
}

// Slack mrkdwn 제어 문자(<, >, &)는 링크 문법 밖에서 이스케이프되어야 한다.
func TestSlack_EscapesControlCharacters(t *testing.T) {
	e := sampleEvent(Finished, Failed)
	e.Phases = append(e.Phases, constant.PhaseResult{Phase: "fast", Status: constant.PhaseSuccess, Elapsed: 100 * time.Millisecond})
	body := rawJSON(t, must(Slack{}.Format(sampleMessage(e, "en"), nil)))
	for _, raw := range []string{"<1s", "<payment>"} {
		if strings.Contains(body, raw) {
			t.Errorf("unescaped %q in slack payload", raw)
		}
	}
}

func must(payload any, err error) any {
	if err != nil {
		panic(err)
	}
	return payload
}

// 긴 실패 메시지도 Discord field 길이 제한을 넘지 않아야 한다.
func TestDiscord_RespectsFieldLimit(t *testing.T) {
	e := sampleEvent(Finished, Failed)
	e.Cause = errors.New(strings.Repeat("x", 5000))
	payload, _ := Discord{}.Format(sampleMessage(e, "en"), nil)
	for _, f := range payload.(discordPayload).Embeds[0].Fields {
		if len([]rune(f.Value)) > discordFieldLimit {
			t.Errorf("field %q exceeds limit: %d", f.Name, len([]rune(f.Value)))
		}
	}
}

// Telegram 메시지는 길이 제한을 넘지 않아야 한다.
func TestTelegram_RespectsTextLimit(t *testing.T) {
	e := sampleEvent(Finished, Failed)
	for i := 0; i < 200; i++ {
		e.Phases = append(e.Phases, constant.PhaseResult{Phase: strings.Repeat("p", 30), Status: constant.PhaseSkipped, Reason: "check not passed"})
	}
	payload, _ := Telegram{}.Format(sampleMessage(e, "en"), mustURL(t, "https://api.telegram.org/bot1:x/sendMessage?chat_id=1"))
	if n := len([]rune(payload.(telegramPayload).Text)); n > telegramTextLimit {
		t.Errorf("telegram text exceeds limit: %d", n)
	}
}

func TestTelegram_RequiresChatID(t *testing.T) {
	_, err := Telegram{}.Format(sampleMessage(sampleEvent(Started, Running), "en"), mustURL(t, "https://api.telegram.org/bot1:x/sendMessage"))
	if err == nil {
		t.Error("expected error when chat_id is missing")
	}
}

// json 형식은 외부 계약이므로 상태·phase·결과물이 이벤트와 일치해야 한다.
func TestJSON_ContractReflectsEvent(t *testing.T) {
	for _, e := range allEvents() {
		payload, _ := JSON{}.Format(sampleMessage(e, "en"), nil)
		var decoded struct {
			Version int    `json:"version"`
			Status  string `json:"status"`
			Event   string `json:"event"`
			Phases  []struct {
				Status    string   `json:"status"`
				Artifacts []string `json:"artifacts"`
			} `json:"phases"`
		}
		if err := json.Unmarshal([]byte(marshal(t, payload)), &decoded); err != nil {
			t.Fatalf("decode error: %v", err)
		}
		if decoded.Version < 2 {
			t.Errorf("version should be >= 2, got %d", decoded.Version)
		}
		if decoded.Status != e.Outcome.String() || decoded.Event != e.Kind.String() {
			t.Errorf("status/event mismatch: %+v", decoded)
		}
		if len(decoded.Phases) != len(e.Phases) {
			t.Error("phases not carried over")
		}
	}
}

// ---- 전송 ----

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

// 지원하지 않는 언어는 알림을 끄지 않고 기본 언어로 계속 동작한다.
func TestNew_UnknownLangStillSends(t *testing.T) {
	if _, ok := New(Config{WebhookURL: "https://example.com/hook", Lang: "xx"}).(noopSender); ok {
		t.Error("unsupported language should not disable notifications")
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
