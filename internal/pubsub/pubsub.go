package pubsub

import (
	"context"
	"encoding/json"
	"fmt"

	ampq "github.com/rabbitmq/amqp091-go"
)

func PublishJSON[T any](ch *ampq.Channel, exchange, key string, val T) error {
	bytes, err := json.Marshal(val)

	// Fatal error (prints and exits)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return err
	}

	ch.PublishWithContext(context.Background(), exchange, key, false, false, ampq.Publishing{ContentType: "application/json", Body: bytes})
	return nil
}
