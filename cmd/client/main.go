package main

import (
	"fmt"
	"log"
	"strings"
	"time"

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

	channel, err := connection.Channel()
	if err != nil {
		log.Fatal(err)
	}
	var table = make(map[string]interface{})
	table["x-dead-letter-exchange"] = "peril_dlx"
	defer connection.Close()
	username, _ := gamelogic.ClientWelcome()
	fmt.Printf("Welcome %s! Nice to see you :)\n", username)
	pubsub.DeclareAndBind(connection, routing.ExchangePerilDirect, strings.Join([]string{routing.PauseKey, username}, "."), routing.PauseKey, pubsub.Transient, table)
	gamestate := gamelogic.NewGameState(username)
	err = pubsub.SubscribeJSON(connection, routing.ExchangePerilTopic, strings.Join([]string{"army_moves", username}, "."), "army_moves.*", pubsub.Transient, handleMove(gamestate, channel), table)

	if err != nil {
		log.Fatal(err)
	}

	err = pubsub.SubscribeJSON(connection, routing.ExchangePerilTopic, "war", "war.*", pubsub.Durable, handleWar(gamestate, channel), nil)
	if err != nil {
		log.Fatal("error creating war subscription\nerr: ", err)
	}

	err = pubsub.SubscribeJSON(connection, routing.ExchangePerilDirect,
		fmt.Sprintf("%s.%s", "pause", username), routing.PauseKey, pubsub.Transient, handlerPause(gamestate), table)
	if err != nil {
		log.Fatal(err)
	}

gameLoop:
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
		case "move":
			armyMove, err := gamestate.CommandMove(words)
			if err != nil {
				println(err.Error())
			}

			err = pubsub.PublishJSON(channel, routing.ExchangePerilTopic, strings.Join([]string{"army_moves", username}, "."), armyMove)
			if err == nil {
				fmt.Println("Move was published successfully")
			} else {
				fmt.Println("Ohh myy dear... Move wasn't pubslihed at all.", err)
			}
		case "status":
			gamelogic.PrintClientHelp()
		case "spam":
			fmt.Println("Spamming not allowed yet!")
		case "quit":
			gamelogic.PrintQuit()
			break gameLoop
		default:
			fmt.Printf("I don't know what %s means\n", words[0])
		}
	}
	fmt.Println("Closing the game")
}

func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) pubsub.Acktype {
	return func(rps routing.PlayingState) pubsub.Acktype {
		defer fmt.Print("> ")
		gs.HandlePause(rps)
		return pubsub.Ack
	}
}

func handleMove(gs *gamelogic.GameState, ch *amqp.Channel) func(gamelogic.ArmyMove) pubsub.Acktype {
	return func(am gamelogic.ArmyMove) pubsub.Acktype {
		defer fmt.Print("> ")
		moveOutcome := gs.HandleMove(am)
		if moveOutcome == gamelogic.MoveOutcomeMakeWar {
			fmt.Printf("This dude: %s, attacked this dude: %s", gs.Player.Username, am.Player.Username)
			rofMsg := gamelogic.RecognitionOfWar{
				Attacker: am.Player,
				Defender: gs.Player,
			}
			// Publish the JSON message
			routingKey := strings.Join([]string{routing.WarRecognitionsPrefix, gs.Player.Username}, ".")
			err := pubsub.PublishJSON(ch, routing.ExchangePerilTopic, routingKey, rofMsg)
			if err != nil {
				fmt.Println("Error publishing JSON message:", err)
				return pubsub.NackRequeue
			}

			return pubsub.Ack
		}
		if moveOutcome == gamelogic.MoveOutComeSafe {
			return pubsub.Ack
		} else if moveOutcome == gamelogic.MoveOutcomeSamePlayer {
			return pubsub.NackDiscard
		} else {
			return pubsub.NackDiscard
		}
	}
}

func handleWar(gs *gamelogic.GameState, ch *amqp.Channel) func(gamelogic.RecognitionOfWar) pubsub.Acktype {
	return func(rof gamelogic.RecognitionOfWar) pubsub.Acktype {
		defer fmt.Print("> ")
		fmt.Println("Handling war in progress...")
		var logs = routing.GameLog{
			CurrentTime: time.Now(),
			Message:     getLogs(gs.HandleWar(rof)),
			Username:    gs.GetUsername(),
		}
		username := gs.Player.Username
		err := pubsub.PublishGob(ch, routing.ExchangePerilTopic, strings.Join([]string{routing.GameLogSlug, username}, "."), logs)
		if err != nil {
			return pubsub.NackRequeue
		}
		return pubsub.Ack
	}
}

func getLogs(warOutcome gamelogic.WarOutcome, winner, loser string) string {
	switch warOutcome {
	case gamelogic.WarOutcomeOpponentWon, gamelogic.WarOutcomeYouWon:
		return fmt.Sprintf("%s won a war against %s", winner, loser)
	case gamelogic.WarOutcomeDraw:
		return fmt.Sprintf("A war between %s and %s resulted in a draw", winner, loser)
	default:
		return string(warOutcome)
	}
}
