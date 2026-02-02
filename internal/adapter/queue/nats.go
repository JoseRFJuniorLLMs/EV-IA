package queue

import (
	"fmt"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

type NATSQueue struct {
	conn *nats.Conn
	log  *zap.Logger
}

func NewNATSQueue(url string, log *zap.Logger) (MessageQueue, error) {
	nc, err := nats.Connect(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	log.Info("Successfully connected to NATS", zap.String("url", url))
	return &NATSQueue{
		conn: nc,
		log:  log,
	}, nil
}

func (q *NATSQueue) Publish(subject string, data []byte) error {
	return q.conn.Publish(subject, data)
}

func (q *NATSQueue) Subscribe(subject string, handler func(data []byte) error) error {
	_, err := q.conn.Subscribe(subject, func(msg *nats.Msg) {
		if err := handler(msg.Data); err != nil {
			q.log.Error("Error processing message", zap.String("subject", subject), zap.Error(err))
		}
	})
	return err
}

func (q *NATSQueue) Close() error {
	q.conn.Close()
	return nil
}
