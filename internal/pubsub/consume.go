package pubsub

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type SimpleQueueType int

const (
	SimpleQueueDurable SimpleQueueType = iota
	SimpleQueueTransient
)

func DeclareAndBind(
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType,
) (*amqp.Channel, amqp.Queue, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, amqp.Queue{}, fmt.Errorf("could not create channel: %v", err)
	}

	queue, err := ch.QueueDeclare(
		queueName,                       // name
		queueType == SimpleQueueDurable, // durable
		queueType != SimpleQueueDurable, // delete when unused
		queueType != SimpleQueueDurable, // exclusive
		false,                           // no-wait
		amqp.Table{"x-dead-letter-exchange": "peril_dlx"},
	)
	if err != nil {
		return nil, amqp.Queue{}, fmt.Errorf("could not declare queue: %v", err)
	}

	err = ch.QueueBind(
		queue.Name, // queue name
		key,        // routing key
		exchange,   // SimpleQueueDurable
		false,      // no-wait
		nil,        // args
	)
	if err != nil {
		return nil, amqp.Queue{}, fmt.Errorf("could not bind queue: %v", err)
	}
	return ch, queue, nil
}

type Acktype int

const (
	Ack Acktype = iota
	NackRequeue
	NackDiscard
)

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType, // an enum to represent "durable" or "transient"
	handler func(T) Acktype,
) error {
	ch, _, err := DeclareAndBind(conn, exchange, queueName, key, queueType)
	if err != nil {
		return err
	}
	deliveryChannel, err := ch.Consume(queueName, "", false, false, false, false, nil)
	if err != nil {
		return err
	}

	go func() {
		for d := range deliveryChannel {
			var msg T
			err := json.Unmarshal(d.Body, &msg)
			if err != nil {
				fmt.Printf("could not unmarshal message: %v\n", err)
				continue
			}
			switch handler(msg) {
			case Ack:
				d.Ack(false)
				fmt.Printf("Handler returned Ack for message: %v\n", msg)
			case NackRequeue:
				d.Nack(false, true)
				fmt.Printf("Handler returned Nack Requeue for message: %v\n", msg)
			case NackDiscard:
				d.Nack(false, false)
				fmt.Printf("Handler returned Nack Discard  for message: %v\n", msg)
			}
		}
	}()
	return nil
}

func SubscribeGob[T any](conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType, // an enum to represent "durable" or "transient"
	handler func(T) Acktype,
) error {
	return subscribe(conn, exchange, queueName, key, queueType, handler,
		func(data []byte) (T, error) {
			var gobMsg T
			reader := bytes.NewBuffer(data)
			decoder := gob.NewDecoder(reader)
			err := decoder.Decode(&gobMsg)
			return gobMsg, err
		})
}

func subscribe[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType,
	handler func(T) Acktype,
	unmarshaller func([]byte) (T, error),
) error {
	ch, _, err := DeclareAndBind(conn, exchange, queueName, key, queueType)
	if err != nil {
		return err
	}
	deliveryChannel, err := ch.Consume(queueName, "", false, false, false, false, nil)
	if err != nil {
		return err
	}

	go func() {
		for d := range deliveryChannel {
			msg, err := unmarshaller(d.Body)
			if err != nil {
				fmt.Printf("could not unmarshal message: %v\n", err)
				continue
			}
			switch handler(msg) {
			case Ack:
				d.Ack(false)
				fmt.Printf("Handler returned Ack for message: %v\n", msg)
			case NackRequeue:
				d.Nack(false, true)
				fmt.Printf("Handler returned Nack Requeue for message: %v\n", msg)
			case NackDiscard:
				d.Nack(false, false)
				fmt.Printf("Handler returned Nack Discard  for message: %v\n", msg)
			}
		}
	}()
	return nil
}
