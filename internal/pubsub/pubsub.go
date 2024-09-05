package pubsub

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func EncodeGob(gl routing.GameLog) ([]byte, error) {
	var network bytes.Buffer
	enc := gob.NewEncoder(&network)
	err := enc.Encode(gl)
	if err != nil {
		return nil, err
	}
	return network.Bytes(), nil
}

func DecodeGob(data []byte) (routing.GameLog, error) {
	var gl routing.GameLog
	dec := gob.NewDecoder(bytes.NewBuffer(data))
	err := dec.Decode(&gl)
	if err != nil {
		return routing.GameLog{}, err
	}
	return gl, nil
}
func DecodeJSON[T any](data []byte) (T, error) {
	var msgUnmarshaled T
	if err := json.Unmarshal(data, &msgUnmarshaled); err != nil {
		fmt.Printf("Error unmarshalling message: %v\n", err)
		// Handle error (maybe reject the message or log it)
		return msgUnmarshaled, err
	}
	return msgUnmarshaled, nil
}

func PublishGob(ch *amqp.Channel, exchange, key string, val routing.GameLog) error {

	bytes, err := EncodeGob(val)
	if err != nil {
		return err
	}
	ch.PublishWithContext(context.Background(), exchange, key, false, false, amqp.Publishing{ContentType: "application/gob", Body: bytes})
	return nil
}
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
	table amqp.Table) (*amqp.Channel, amqp.Queue, error) {

	channel, _ := conn.Channel()

	queue, err := channel.QueueDeclare(queueName, simpleQueueType == Durable, simpleQueueType == Transient, simpleQueueType == Transient, false, table)
	if err != nil {
		return nil, amqp.Queue{}, fmt.Errorf("error ocured when declaring a queue. quename: %s,\n%v", queueName, err)
	}
	err = channel.QueueBind(queueName, key, exchange, false, nil)

	return channel, queue, err
}

type MessageProcessor[T any] struct {
	handler       func(T) Acktype
	decodeHandler func([]byte) (T, error)
}

func NewMessageProcessor[T any](handler func(T) Acktype,
	decodeHandler func([]byte) (T, error)) *MessageProcessor[T] {
	return &MessageProcessor[T]{handler: handler, decodeHandler: decodeHandler}
}

type Acktype int

const (
	Ack         Acktype = iota
	NackRequeue Acktype = iota
	NackDiscard Acktype = iota
)

func (mp *MessageProcessor[T]) ProcessMessage(msg amqp.Delivery) {
	fmt.Printf("Received a message\n")
	fmt.Println("Unmarshalling...")
	msgUnmarshaled, err := mp.decodeHandler(msg.Body)
	if err != nil {
		fmt.Printf("Error unmarshalling message: %v\n", err)
		// Handle error (maybe reject the message or log it)
		msg.Nack(false, false)
		return
	}
	ack := mp.handler(msgUnmarshaled)
	switch ack {
	case Ack:
		err := msg.Ack(false)
		if err != nil {
			fmt.Println("error!", err)
		} else {
			fmt.Println("Ack fired.")
		}
	case NackRequeue:
		err := msg.Nack(false, true)
		if err != nil {
			fmt.Println("error!", err)
		} else {
			fmt.Println("NackR fired.")
		}
	case NackDiscard:
		err := msg.Nack(false, false)

		if err != nil {
			fmt.Println("error!", err)
		} else {
			fmt.Println("NackD fired.")
		}
	}
	// msg.Ack(false)
}

func (mp *MessageProcessor[T]) ProcessDeliveries(deliveries <-chan amqp.Delivery) {
	go func() {
		for msg := range deliveries {
			mp.ProcessMessage(msg)
		}
	}()
}

func Subscribe[T any](
	conn *amqp.Connection,
	exchange, queueName, key string,
	simpleQueueType SimpleQueueType,
	handler func(T) Acktype,
	decodeHandler func([]byte) (T, error),
	table amqp.Table) error {
	channel, queue, err := DeclareAndBind(conn, exchange, queueName, key, simpleQueueType, table)
	if err != nil {
		return fmt.Errorf("error occured when declare and bind\n%v", err)
	}

	// Do prefetch here
	err = channel.Qos(10, 0, false)
	if err != nil {
		fmt.Printf("Eror when declaring Qos\nerr:%v\n", err)
	}
	deliveryChan, err := channel.Consume(queue.Name, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("error when chanel was created\n%v", err)
	}

	processor := NewMessageProcessor(handler, decodeHandler)
	processor.ProcessDeliveries(deliveryChan)

	return nil
}
