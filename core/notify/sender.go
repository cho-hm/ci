package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"
)

const sendTimeout = 10 * time.Second

// Sender - 알림 전송 창구. 알림 실패가 CI 결과에 영향을 주지 않도록 에러를 반환하지 않는다.
type Sender interface {
	Send(e Event)
}

// Config - 웹후크 설정. WebhookURL 이 비어 있으면 알림 기능은 비활성화된다.
type Config struct {
	WebhookURL  string
	WebhookType string
}

// New - 설정을 해석해 Sender 를 만든다. 설정이 없거나 잘못되었으면 아무것도 하지 않는 Sender 를 반환한다.
func New(cfg Config) Sender {
	if cfg.WebhookURL == "" {
		log.Println("notification disabled: WEBHOOK_URL is not set")
		return noopSender{}
	}
	endpoint, err := url.Parse(cfg.WebhookURL)
	if err != nil || endpoint.Scheme == "" || endpoint.Host == "" {
		log.Println("WARN notification disabled: WEBHOOK_URL is not a valid absolute url")
		return noopSender{}
	}
	notifier, err := Resolve(cfg.WebhookType, endpoint)
	if err != nil {
		log.Printf("WARN notification disabled: %v", err)
		return noopSender{}
	}
	log.Printf("notification enabled: %s", notifier.Name())
	return &webhookSender{
		notifier: notifier,
		endpoint: endpoint,
		client:   &http.Client{Timeout: sendTimeout},
	}
}

type noopSender struct{}

func (noopSender) Send(Event) {}

type webhookSender struct {
	notifier Notifier
	endpoint *url.URL // 토큰이 포함된 비밀 값이므로 로그에 남기지 않는다.
	client   *http.Client
}

func (s *webhookSender) Send(e Event) {
	if err := s.send(e); err != nil {
		log.Printf("WARN %s notification (%s) failed: %v", s.notifier.Name(), e.Kind, err)
	}
}

func (s *webhookSender) send(e Event) error {
	payload, err := s.notifier.Format(e, s.endpoint)
	if err != nil {
		return err
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), sendTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.endpoint.String(), bytes.NewReader(body))
	if err != nil {
		return redact(err)
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := s.client.Do(req)
	if err != nil {
		return redact(err)
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("unexpected response status: %s", res.Status)
	}
	return nil
}

// redact - net/http 에러 메시지에 포함되는 요청 URL(웹후크 토큰)을 제거한다.
func redact(err error) error {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return fmt.Errorf("%s request: %w", urlErr.Op, urlErr.Err)
	}
	return err
}
