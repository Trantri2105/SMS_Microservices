package infra

import "github.com/segmentio/kafka-go"

func NewKafkaWriter(brokers []string, topic string) *kafka.Writer {
	return &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        topic,
		Balancer:     &kafka.Hash{},
		RequiredAcks: kafka.RequireOne,
	}
}

func NewKafkaReader(brokers []string, consumerGroupID string, topic string) *kafka.Reader {
	return kafka.NewReader(kafka.ReaderConfig{
		Brokers:     brokers,
		GroupID:     consumerGroupID,
		Topic:       topic,
	})
}
