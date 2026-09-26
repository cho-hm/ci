package notify

import "net/url"

// Message - Notifier 에 전달되는 입력. 원본 데이터(Event)와 공통 표현(Card)을 함께 담는다.
type Message struct {
	Event Event
	Card  Card
}

// Notifier - 하나의 웹후크 플랫폼에 대한 메시지 형식 전략.
// 전송(HTTP)은 Sender 가 담당하고, Notifier 는 Card 를 플랫폼 꾸밈 요소로 렌더링하는 것만 책임진다.
type Notifier interface {
	// Name - WEBHOOK_TYPE 으로 지정할 때 쓰는 식별자.
	Name() string
	// Matches - WEBHOOK_TYPE 미지정 시, 웹후크 URL 만으로 이 플랫폼인지 판별.
	Matches(endpoint *url.URL) bool
	// Format - JSON 으로 직렬화될 요청 본문을 만든다.
	Format(m Message, endpoint *url.URL) (any, error)
}
