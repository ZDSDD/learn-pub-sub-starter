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
	durable SimpleQueueType = iota
	transient
)

func DeclareAndBind(
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	simpleQueueType int, // an enum to represent "durable" or "transient"
) (*amqp.Channel, amqp.Queue, error) {

	channel, _ := conn.Channel()
	queue, _ := channel.QueueDeclare(queueName, simpleQueueType == int(durable), simpleQueueType == int(transient), simpleQueueType == int(transient), false, nil)

	err := channel.QueueBind(queueName, key, exchange, false, nil)

	return channel, queue, err
}
