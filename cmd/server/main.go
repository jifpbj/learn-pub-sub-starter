package main

import (
	"fmt"
	"os"
	"os/signal"

	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	fmt.Println("Starting Peril server...")
	const connectionString = "amqp://guest:guest@localhost:5672/"
	conn := amqp.Dial(connectionString)
	defer conn.Close()
	fmt.Println("Connected to RabbitMQ:", conn.RemoteAddr())

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	<-signalChan
}
