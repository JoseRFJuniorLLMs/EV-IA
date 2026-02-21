package queue

import (
	"fmt"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

// RabbitMQQueue implements the MessageQueue interface using RabbitMQ
type RabbitMQQueue struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	url     string
	mu      sync.RWMutex
	log     *zap.Logger
}

// NewRabbitMQQueue creates a new RabbitMQ message queue adapter
func NewRabbitMQQueue(url string, log *zap.Logger) (MessageQueue, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open RabbitMQ channel: %w", err)
	}

	q := &RabbitMQQueue{
		conn:    conn,
		channel: ch,
		url:     url,
		log:     log,
	}

	go q.monitorConnection()

	log.Info("Successfully connected to RabbitMQ", zap.String("url", url))
	return q, nil
}

func (q *RabbitMQQueue) Publish(subject string, data []byte) error {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if q.channel == nil {
		return fmt.Errorf("rabbitmq: channel not available")
	}

	err := q.channel.ExchangeDeclare(subject, "fanout", true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("rabbitmq: declare exchange: %w", err)
	}

	err = q.channel.Publish(
		subject, "", false, false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        data,
			Timestamp:   time.Now(),
		},
	)
	if err != nil {
		return fmt.Errorf("rabbitmq: publish: %w", err)
	}

	return nil
}

func (q *RabbitMQQueue) Subscribe(subject string, handler func(data []byte) error) error {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if q.channel == nil {
		return fmt.Errorf("rabbitmq: channel not available")
	}

	err := q.channel.ExchangeDeclare(subject, "fanout", true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("rabbitmq: declare exchange: %w", err)
	}

	queue, err := q.channel.QueueDeclare("", false, true, true, false, nil)
	if err != nil {
		return fmt.Errorf("rabbitmq: declare queue: %w", err)
	}

	err = q.channel.QueueBind(queue.Name, "", subject, false, nil)
	if err != nil {
		return fmt.Errorf("rabbitmq: bind queue: %w", err)
	}

	msgs, err := q.channel.Consume(queue.Name, "", true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("rabbitmq: consume: %w", err)
	}

	go func() {
		for msg := range msgs {
			if err := handler(msg.Body); err != nil {
				q.log.Error("Error processing RabbitMQ message",
					zap.String("exchange", subject),
					zap.Error(err),
				)
			}
		}
	}()

	q.log.Info("Subscribed to RabbitMQ exchange", zap.String("exchange", subject))
	return nil
}

func (q *RabbitMQQueue) Close() error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.channel != nil {
		q.channel.Close()
	}
	if q.conn != nil {
		return q.conn.Close()
	}
	return nil
}

func (q *RabbitMQQueue) monitorConnection() {
	for {
		reason, ok := <-q.conn.NotifyClose(make(chan *amqp.Error))
		if !ok {
			return
		}
		q.log.Warn("RabbitMQ connection lost, reconnecting...", zap.String("reason", reason.Reason))

		for {
			time.Sleep(5 * time.Second)
			conn, err := amqp.Dial(q.url)
			if err != nil {
				q.log.Error("Failed to reconnect to RabbitMQ", zap.Error(err))
				continue
			}
			ch, err := conn.Channel()
			if err != nil {
				conn.Close()
				continue
			}

			q.mu.Lock()
			q.conn = conn
			q.channel = ch
			q.mu.Unlock()

			q.log.Info("Successfully reconnected to RabbitMQ")
			break
		}
	}
}
