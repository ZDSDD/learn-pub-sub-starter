package main

import (
	"fmt"
	"strings"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	fmt.Println("Starting Peril client...")
	var connectionString string = "amqp://guest:guest@localhost:5672/"
	connection, err := amqp.Dial(connectionString)
	if err != nil {
		fmt.Printf("There was an error creating connection")
		return
	}
	fmt.Printf("Connected successfuly!")

	defer connection.Close()
	username, _ := gamelogic.ClientWelcome()
	fmt.Printf("Welcome %s! Nice to see you :)\n", username)

	pubsub.DeclareAndBind(connection, routing.ExchangePerilDirect, strings.Join([]string{routing.PauseKey, username}, "."), routing.PauseKey, pubsub.Transient)
	gamestate := gamelogic.NewGameState(username)
	gamelogic.PrintClientHelp()
	for {
		words := gamelogic.GetInput()
		if len(words) == 0 {
			continue
		}

		switch words[0] {
		case "spawn":
			err := gamestate.CommandSpawn(words)
			if err != nil {
				fmt.Println(err.Error())
			}
		case "exit":
			fmt.Println("Exiting the game...")
			return
		default:
			fmt.Printf("I don't know what %s means\n", words[0])
		}
	}
}
