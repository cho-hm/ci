package notify

import (
	"fmt"
	"net/url"
	"strings"
)

var registry = map[string]Notifier{}

// fallback - URL 로 판별되지 않을 때 사용하는 플랫폼 중립 형식.
var fallback Notifier = JSON{}

func Register(n Notifier) {
	registry[n.Name()] = n
}

// Resolve - 명시된 타입이 있으면 그것을, 없으면 URL 로 판별한 Notifier 를 반환한다.
func Resolve(webhookType string, endpoint *url.URL) (Notifier, error) {
	if webhookType != "" {
		return byName(webhookType)
	}
	return detect(endpoint), nil
}

func byName(webhookType string) (Notifier, error) {
	n, ok := registry[strings.ToLower(strings.TrimSpace(webhookType))]
	if !ok {
		return nil, fmt.Errorf("unsupported webhook type: %s", webhookType)
	}
	return n, nil
}

func detect(endpoint *url.URL) Notifier {
	for _, n := range registry {
		if n != fallback && n.Matches(endpoint) {
			return n
		}
	}
	return fallback
}

func hostIs(endpoint *url.URL, hosts ...string) bool {
	host := strings.ToLower(endpoint.Hostname())
	for _, h := range hosts {
		if host == h {
			return true
		}
	}
	return false
}

func hostEndsWith(endpoint *url.URL, suffixes ...string) bool {
	host := strings.ToLower(endpoint.Hostname())
	for _, s := range suffixes {
		if strings.HasSuffix(host, s) {
			return true
		}
	}
	return false
}
