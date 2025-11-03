package pubsub

import (
	"encoding/json"
	"encoding/gob"
	"bytes"
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

type SimpleQueueType int

const (
	SimpleQueueDurable SimpleQueueType = iota
	SimpleQueueTransient
)

type Acktype int

const (
	Ack Acktype = iota
	NackDiscard
	NackRequeue
)

func SubscribeJSON[T any](conn *amqp.Connection, exchange, queueName, key string, simpleQueueType SimpleQueueType, handler func(T) Acktype) error {
	unmarshaller := func(data []byte) (T, error) {
		var target T
		err := json.Unmarshal(data, &target)
		return target, err
	}

	return subscribe[T](
		conn,
		exchange,
		queueName,
		key,
		simpleQueueType,
		handler,
		unmarshaller,
	)
}

func SubscribeGob[T any](conn *amqp.Connection, exchange, queueName, key string, simpleQueueType SimpleQueueType, handler func(T) Acktype) error {
	unmarshaller := func(data []byte) (T, error) {
		buffer := bytes.NewBuffer(data)
		decoder := gob.NewDecoder(buffer)
		var target T
		err := decoder.Decode(&target)
		return target, err
	}
	
	return subscribe[T](
		conn,
		exchange,
		queueName,
		key,
		simpleQueueType,
		handler,
		unmarshaller,
	)
}


func subscribe[T any](
	conn *amqp.Connection, 
	exchange, 
	queueName, 
	key string, 
	simpleQueueType SimpleQueueType, 
	handler func(T) Acktype,
	unmarshaller func([]byte)(T, error),
) error {
	ch, queue, err := DeclareAndBind(
		conn,
		exchange,
		queueName,
		key,
		simpleQueueType,
	)
	if err != nil {
		return fmt.Errorf("could not declare and bind queue: %v", err)
	}
	msgs, err := ch.Consume(queue.Name, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("could not consume messages: %v", err)
	}

	go func() {
		defer ch.Close()
		for msg := range msgs {
			target, err := unmarshaller(msg.Body)
			if err != nil {
				fmt.Printf("could not unmarshal message: %v", err)
				continue
			}
			switch handler(target) {
			case Ack:
				msg.Ack(false)
				log.Println("Ack")
			case NackDiscard:
				msg.Nack(false, false)
				log.Println("NackDiscard")
			case NackRequeue:
				msg.Nack(false, true)
				log.Println("NackRequeue")
			}
		}
	}()
	return nil
}

func DeclareAndBind(conn *amqp.Connection, exchange, queueName, key string, queueType SimpleQueueType) (*amqp.Channel, amqp.Queue, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, amqp.Queue{}, fmt.Errorf("could not create channel: %v", err)
	}

	queue, err := ch.QueueDeclare(
		queueName,
		queueType == SimpleQueueDurable,
		queueType != SimpleQueueDurable,
		queueType != SimpleQueueDurable,
		false,
		amqp.Table{
			"x-dead-letter-exchange": "peril_dlx",
		},
	)
	if err != nil {
		return nil, amqp.Queue{}, fmt.Errorf("could not create queue: %v", err)
	}

	err = ch.QueueBind(
		queue.Name,
		key,
		exchange,
		false,
		nil,
	)
	if err != nil {
		return nil, amqp.Queue{}, fmt.Errorf("could not bind queue: %v", err)
	}
	return ch, queue, nil
}
