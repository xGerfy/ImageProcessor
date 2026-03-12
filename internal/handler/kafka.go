package handler

import (
	"context"
	"encoding/json"

	"image-processor/internal/model"

	kafka "github.com/segmentio/kafka-go"
	wbfkafka "github.com/wb-go/wbf/kafka"
	"github.com/wb-go/wbf/zlog"
)

// KafkaProducer - обёртка над Kafka producer
type KafkaProducer struct {
	producer *wbfkafka.Producer
	topic    string
}

// NewKafkaProducer создаёт producer
func NewKafkaProducer(brokers []string, topic string, _ zlog.Zerolog) *KafkaProducer {
	return &KafkaProducer{
		producer: wbfkafka.NewProducer(brokers, topic),
		topic:    topic,
	}
}

// SendTask отправляет задачу на обработку
func (p *KafkaProducer) SendTask(ctx context.Context, task *model.ImageTask) error {
	data, err := json.Marshal(task)
	if err != nil {
		return err
	}

	zlog.Logger.Info().Str("id", task.ID).Msg("sending task to Kafka")

	err = p.producer.Send(ctx, []byte(task.ID), data)
	if err != nil {
		return err
	}

	return nil
}

// Close закрывает producer
func (p *KafkaProducer) Close() {
	p.producer.Close()
}

// KafkaConsumer - обёртка над Kafka consumer
type KafkaConsumer struct {
	consumer *wbfkafka.Consumer
	topic    string
	groupID  string
	lastMsg  kafka.Message // Последнее полученное сообщение
}

// NewKafkaConsumer создаёт consumer
func NewKafkaConsumer(brokers []string, topic, groupID string, _ zlog.Zerolog) *KafkaConsumer {
	return &KafkaConsumer{
		consumer: wbfkafka.NewConsumer(brokers, topic, groupID),
		topic:    topic,
		groupID:  groupID,
	}
}

// FetchTask получает задачу
func (c *KafkaConsumer) FetchTask(ctx context.Context) (*model.ImageTask, error) {
	msg, err := c.consumer.Fetch(ctx)
	if err != nil {
		return nil, err
	}

	// Сохраняем сообщение для последующего коммита
	c.lastMsg = msg

	var task model.ImageTask
	if err := json.Unmarshal(msg.Value, &task); err != nil {
		return nil, err
	}

	zlog.Logger.Info().Str("id", task.ID).Msg("received task from Kafka")

	return &task, nil
}

// CommitTask подтверждает обработку задачи
func (c *KafkaConsumer) CommitTask(ctx context.Context, task *model.ImageTask) error {
	if c.lastMsg.Topic == "" {
		return nil
	}

	err := c.consumer.Commit(ctx, c.lastMsg)
	c.lastMsg = kafka.Message{} // Очищаем после коммита
	return err
}

// Close закрывает consumer
func (c *KafkaConsumer) Close() {
	c.consumer.Close()
}
