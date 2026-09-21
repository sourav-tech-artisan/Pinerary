package notifications

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	webpush "github.com/SherClockHolmes/webpush-go"
)

var ErrSubscriptionExpired = errors.New("push subscription expired")

type Sender interface {
	Send(context.Context, json.RawMessage, []byte) error
}

type WebPushSender struct {
	options webpush.Options
}

func NewWebPushSender(subscriber, publicKey, privateKey string) *WebPushSender {
	return &WebPushSender{options: webpush.Options{
		Subscriber: subscriber, VAPIDPublicKey: publicKey, VAPIDPrivateKey: privateKey,
		TTL: 3600, Urgency: webpush.UrgencyNormal,
	}}
}

func (s *WebPushSender) Send(ctx context.Context, rawSubscription json.RawMessage, payload []byte) error {
	if s.options.Subscriber == "" || s.options.VAPIDPublicKey == "" || s.options.VAPIDPrivateKey == "" {
		return fmt.Errorf("Web Push VAPID configuration is incomplete")
	}
	var subscription webpush.Subscription
	if err := json.Unmarshal(rawSubscription, &subscription); err != nil {
		return fmt.Errorf("decode Web Push subscription: %w", err)
	}
	response, err := webpush.SendNotificationWithContext(ctx, payload, &subscription, &s.options)
	if err != nil {
		return fmt.Errorf("send Web Push notification: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusGone || response.StatusCode == http.StatusNotFound {
		return ErrSubscriptionExpired
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("Web Push endpoint returned status %d", response.StatusCode)
	}
	return nil
}
