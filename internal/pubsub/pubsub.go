package pubsub

import (
	"context"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

func PublishJSON[T any](ch *amqp.Channel, exchange, key string, val T) error {
	bytes, err := json.Marshal(val)

	// Fatal error (prints and exits)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return err
	}

	ch.PublishWithContext(context.Background(), exchange, key, false, false, amqp.Publishing{ContentType: "application/json", Body: bytes})
	return nil
}

type SimpleQueueType int

const (
	Durable   SimpleQueueType = iota
	Transient SimpleQueueType = iota
)

func DeclareAndBind(
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	simpleQueueType SimpleQueueType, // an enum to represent "durable" or "transient"
) (*amqp.Channel, amqp.Queue, error) {

	channel, _ := conn.Channel()
	queue, _ := channel.QueueDeclare(queueName, simpleQueueType == Durable, simpleQueueType == Transient, simpleQueueType == Transient, false, nil)

	err := channel.QueueBind(queueName, key, exchange, false, nil)

	return channel, queue, err
}

type MessageProcessor[T any] struct {
	handler func(T)
}

func NewMessageProcessor[T any](handler func(T)) *MessageProcessor[T] {
	return &MessageProcessor[T]{handler: handler}
}

func (mp *MessageProcessor[T]) ProcessMessage(msg amqp.Delivery) {
	fmt.Printf("Received a message: %s\n", msg.Body)
	fmt.Println("Unmarshalling...")
	var msgUnmarshaled T
	if err := json.Unmarshal(msg.Body, &msgUnmarshaled); err != nil {
		fmt.Printf("Error unmarshalling message: %v\n", err)
		// Handle error (maybe reject the message or log it)
		return
	}
	mp.handler(msgUnmarshaled)
	msg.Ack(false)
}

func (mp *MessageProcessor[T]) ProcessDeliveries(deliveries <-chan amqp.Delivery) {
	go func() {
		for msg := range deliveries {
			mp.ProcessMessage(msg)
		}
	}()
}

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange, queueName, key string,
	simpleQueueType SimpleQueueType,
	handler func(T),
) error {
	channel, queue, err := DeclareAndBind(conn, exchange, queueName, key, simpleQueueType)
	if err != nil {
		return err
	}

	deliveryChan, err := channel.Consume(queue.Name, "", false, false, false, false, nil)
	if err != nil {
		return err
	}

	processor := NewMessageProcessor(handler)
	processor.ProcessDeliveries(deliveryChan)

	return nil
}
