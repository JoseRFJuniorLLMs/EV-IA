package queue

// MessageQueue defines the interface for a message queue adapter
type MessageQueue interface {
	Publish(subject string, data []byte) error
	Subscribe(subject string, handler func(data []byte) error) error
	Close() error
}
