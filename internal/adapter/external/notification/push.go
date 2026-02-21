package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"go.uber.org/zap"
)

// PushAdapter sends push notifications via Firebase Cloud Messaging (FCM) HTTP v1 API
type PushAdapter struct {
	serverKey  string
	projectID  string
	httpClient *http.Client
	log        *zap.Logger
}

// NewPushAdapter creates a new Firebase push notification adapter
func NewPushAdapter(serverKey, projectID string, log *zap.Logger) *PushAdapter {
	return &PushAdapter{
		serverKey:  serverKey,
		projectID:  projectID,
		httpClient: &http.Client{},
		log:        log,
	}
}

// fcmMessage represents an FCM message payload
type fcmMessage struct {
	To           string            `json:"to,omitempty"`
	Topic        string            `json:"topic,omitempty"`
	Notification *fcmNotification  `json:"notification"`
	Data         map[string]string `json:"data,omitempty"`
}

type fcmNotification struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

// SendPush sends a push notification to a specific device token
func (a *PushAdapter) SendPush(ctx context.Context, deviceToken, title, body string, data map[string]string) error {
	if a.serverKey == "" {
		a.log.Warn("Push adapter not configured, skipping send", zap.String("token", deviceToken))
		return nil
	}

	msg := fcmMessage{
		To:           deviceToken,
		Notification: &fcmNotification{Title: title, Body: body},
		Data:         data,
	}

	return a.send(ctx, msg)
}

// SendToTopic sends a push notification to all devices subscribed to a topic
func (a *PushAdapter) SendToTopic(ctx context.Context, topic, title, body string) error {
	if a.serverKey == "" {
		a.log.Warn("Push adapter not configured, skipping topic send", zap.String("topic", topic))
		return nil
	}

	msg := fcmMessage{
		Topic:        topic,
		Notification: &fcmNotification{Title: title, Body: body},
	}

	return a.send(ctx, msg)
}

func (a *PushAdapter) send(ctx context.Context, msg fcmMessage) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("push: marshal message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://fcm.googleapis.com/fcm/send", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("push: create request: %w", err)
	}

	req.Header.Set("Authorization", "key="+a.serverKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		a.log.Error("Failed to send push notification", zap.Error(err))
		return fmt.Errorf("push: send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		a.log.Error("FCM API error", zap.Int("status", resp.StatusCode))
		return fmt.Errorf("push: FCM error status %d", resp.StatusCode)
	}

	a.log.Info("Push notification sent",
		zap.String("to", msg.To),
		zap.String("topic", msg.Topic),
	)
	return nil
}
