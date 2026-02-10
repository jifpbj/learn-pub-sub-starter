package main

import (
	"fmt"
	"log"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	fmt.Println("Starting Peril server...")

	const rabbitConnString = "amqp://guest:guest@127.0.0.1:5672/"

	conn, err := amqp.Dial(rabbitConnString)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()
	fmt.Println("Connected to RabbitMQ:")

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}

	_, _, err = pubsub.DeclareAndBind(
		conn,
		routing.ExchangePerilTopic,
		routing.GameLogSlug,
		routing.GameLogSlug+".*",
		pubsub.SimpleQueueDurable,
	)
	if err != nil {
		log.Fatalf("Failed to declare and bind queue to ExchangePerilTopic, %v", err)
	}

	gamelogic.PrintServerHelp()
	var isPaused bool
	var shouldPublish bool

	for {
		shouldPublish = false
		words := gamelogic.GetInput()
		if len(words) == 0 {
			continue
		}
		switch words[0] {
		case "pause":
			fmt.Println("Sending Pause Message")
			isPaused = true
			shouldPublish = true
		case "resume":
			fmt.Println("Sending Resume Message")
			isPaused = false
			shouldPublish = true
		case "quit":
			fmt.Println("Exiting...")
			return
		default:
			fmt.Println("Command not understood")
		}
		if shouldPublish {
			fmt.Println("Publishing message...")
			err = pubsub.PublishJSON(
				ch,
				routing.ExchangePerilDirect,
				routing.PauseKey,
				routing.PlayingState{
					IsPaused: isPaused,
				},
			)
			if err != nil {
				log.Fatalf("Failed to publish message: %v", err)
			}
		}
	}
	// signalChan := make(chan os.Signal, 1)
	// signal.Notify(signalChan, os.Interrupt)
	// <-signalChan
	// fmt.Println("RabbitMQ connection closed. Exiting...")
}
