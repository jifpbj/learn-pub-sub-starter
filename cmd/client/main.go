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

	pubsub.SubscribeJSON(conn, routing.ExchangePerilDirect, routing.PauseKey+"."+gs.GetUsername(), routing.PauseKey, pubsub.SimpleQueueTransient, handlerPause(gs))

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

func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) {
	return func(ps routing.PlayingState) {
		defer fmt.Print("> ")
		gs.HandlePause(ps)
	}
}
