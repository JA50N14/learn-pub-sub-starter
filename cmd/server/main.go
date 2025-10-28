package main

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/JA50N14/learn-pub-sub-starter/internal/pubsub"
	amqp "github.com/rabbitmq/amqp091-go"

)

func main() {
	const connStr = "amqp://guest:guest@localhost:5672/"
	conn, err := amqp.Dial(connStr)
	if err != nil {
		fmt.Println("error creating ampq connection: %v", err)
		os.Exit(1)
	}
	conn.Close()
	fmt.Println("Starting Peril server...")
	
	ch, err := conn.Channel()
	if err != nil {
		fmt.Println("error opening channel on amqp connection: %v", err)
		os.Exit(1)
	}
	err = pubsub.PublishJSON(ch, )

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	<-signalChan
	fmt.Println("Program is shutting down and connections closed")



}
