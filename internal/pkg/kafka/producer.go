package kafka

import (
	"log/slog"

	"github.com/segmentio/kafka-go"
)

// NewProducer initializes and returns a new Kafka writer (producer).
func NewProducer(brokers []string, topic string) *kafka.Writer {
	return &kafka.Writer{
		Addr:     kafka.TCP(brokers...),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
		// Additional configurations for robustness:
		RequiredAcks: kafka.RequireOne, // Acknowledge after leader has written.
		Async:        true,             // Asynchronous writes for higher throughput.
		Completion: func(messages []kafka.Message, err error) {
			if err != nil {
				slog.Error("Kafka async write failed", "error", err)
			}
		},
	}
}
