package mocks

// MockMessageQueue is a mock implementation of MessageQueue interface
type MockMessageQueue struct {
	PublishedMessages map[string][][]byte
	Subscribers       map[string][]func([]byte) error
	PublishFunc       func(topic string, data []byte) error
	SubscribeFunc     func(topic string, handler func([]byte) error) error
	CloseFunc         func() error
}

func NewMockMessageQueue() *MockMessageQueue {
	return &MockMessageQueue{
		PublishedMessages: make(map[string][][]byte),
		Subscribers:       make(map[string][]func([]byte) error),
	}
}

func (m *MockMessageQueue) Publish(topic string, data []byte) error {
	if m.PublishFunc != nil {
		return m.PublishFunc(topic, data)
	}
	m.PublishedMessages[topic] = append(m.PublishedMessages[topic], data)
	return nil
}

func (m *MockMessageQueue) Subscribe(topic string, handler func([]byte) error) error {
	if m.SubscribeFunc != nil {
		return m.SubscribeFunc(topic, handler)
	}
	m.Subscribers[topic] = append(m.Subscribers[topic], handler)
	return nil
}

func (m *MockMessageQueue) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}

// GetPublishedMessages returns all messages published to a topic
func (m *MockMessageQueue) GetPublishedMessages(topic string) [][]byte {
	return m.PublishedMessages[topic]
}

// ClearMessages clears all published messages
func (m *MockMessageQueue) ClearMessages() {
	m.PublishedMessages = make(map[string][][]byte)
}
