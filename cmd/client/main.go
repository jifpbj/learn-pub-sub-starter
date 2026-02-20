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
	fmt.Println("Starting Peril client...")

	const rabbitConnString = "amqp://guest:guest@127.0.0.1:5672/"

	conn, err := amqp.Dial(rabbitConnString)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()
	fmt.Println("Connected to RabbitMQ:")

	username, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Fatalf("Failed to get username from ClientWelcome, %v", err)
	}

	gs := gamelogic.NewGameState(username)

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to create channel: %v", err)
	}

	err = pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilTopic,
		routing.ArmyMovesPrefix+"."+username,
		routing.ArmyMovesPrefix+".*",
		pubsub.SimpleQueueTransient,
		handlerMove(gs),
	)
	if err != nil {
		log.Fatalf("Failed to subscribe to army moves: %v", err)
	}

	err = pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilDirect,
		routing.PauseKey+"."+gs.GetUsername(),
		routing.PauseKey,
		pubsub.SimpleQueueTransient,
		handlerPause(gs),
	)
	if err != nil {
		log.Fatalf("Failed to subscribe to pause messages: %v", err)
	}

	for {
		words := gamelogic.GetInput()
		if len(words) == 0 {
			fmt.Println("No command entered")
			continue
		}
		switch words[0] {
		case "spawn":
			fmt.Println("Spawning unit...")
			err := gs.CommandSpawn(words)
			if err != nil {
				fmt.Printf("Error processing spawn command: %v\n", err)
			}
		case "move":
			armyMove, err := gs.CommandMove(words)
			if err != nil {
				fmt.Printf("Error processing move command: %v\n", err)
			}
			err = pubsub.PublishJSON(
				ch,
				routing.ExchangePerilTopic,
				routing.ArmyMovesPrefix+"."+username,
				armyMove,
			)
			if err != nil {
				fmt.Printf("Failed to publish move command: %v\n", err)
			}
			fmt.Printf("Published move command to RabbitMQ: %v\n", armyMove)
			fmt.Printf("%v moved to %v with units %v", armyMove.Player.Username, armyMove.ToLocation, armyMove.Units)
		case "status":
			gs.CommandStatus()
		case "help":
			gamelogic.PrintClientHelp()
		case "spam":
			fmt.Println("Spamming not allowed yet!")
		case "quit":
			gamelogic.PrintQuit()
			return
		default:
			fmt.Println("Command not understood, 'help' for list of commands")
			continue
		}
	}
}

func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) pubsub.Acktype {
	return func(ps routing.PlayingState) pubsub.Acktype {
		defer fmt.Print("> ")
		gs.HandlePause(ps)
		return pubsub.Ack
	}
}

func handlerMove(gs *gamelogic.GameState) func(gamelogic.ArmyMove) pubsub.Acktype {
	return func(am gamelogic.ArmyMove) pubsub.Acktype {
		defer fmt.Print("> ")
		move := gs.HandleMove(am)
		switch move {
		case gamelogic.MoveOutcomeSafe:
			return pubsub.Ack
		case gamelogic.MoveOutcomeMakeWar:
			return pubsub.Ack
		case gamelogic.MoveOutcomeSamePlayer:
			return pubsub.NackDiscard
		}
		fmt.Printf("Unknown move outcome: %v\n", move)
		return pubsub.NackDiscard
	}
}
