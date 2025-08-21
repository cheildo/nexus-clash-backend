package kafka

import (
	"time"

	"github.com/segmentio/kafka-go"
)

// NewConsumer initializes and returns a new Kafka reader (consumer).
func NewConsumer(brokers []string, topic, groupID string) *kafka.Reader {
	return kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		Topic:          topic,
		GroupID:        groupID, // Consumers in the same group share the load.
		MinBytes:       10e3,    // 10KB
		MaxBytes:       10e6,    // 10MB
		CommitInterval: time.Second,
	})
}
