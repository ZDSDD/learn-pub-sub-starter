package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	fmt.Println("Starting Peril server...")
	var connectionString string = "amqp://guest:guest@localhost:5672/"

	connection, err := amqp.Dial(connectionString)
	if err != nil {
		fmt.Printf("There was an error creating connection")
		return
	}
	fmt.Println("Connected successfuly!")

	defer connection.Close()

	channel, err := connection.Channel()
	if err != nil {
		fmt.Printf("\nthere was en arror creating connection channel %v\n", err)
		return
	}
	pubsub.PublishJSON(channel, routing.ExchangePerilDirect, routing.PauseKey, routing.PlayingState{IsPaused: true})
	gamelogic.PrintServerHelp()
	// wait for ctrl+c
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	done := make(chan struct{})
	pubsub.DeclareAndBind(connection, routing.ExchangePerilTopic, routing.GameLogSlug, "game_logs.*", pubsub.Durable, nil)
	pubsub.Subscribe(connection, routing.ExchangePerilTopic, "game_logs", "game_logs.*", pubsub.Durable, handleLogs, pubsub.DecodeGob, nil)
	go func() {
		for {
			select {
			case <-ctx.Done():
				fmt.Println("Received interrupt signal. Exiting...")
				close(done)
				return
			default:
				var words = []string{}
				// words := gamelogic.GetInput()
				if len(words) == 0 {
					continue
				}

				switch words[0] {
				case "pause":
					fmt.Println("pause command detected. Sending message...")
					err := pubsub.PublishJSON(channel, routing.ExchangePerilDirect, routing.PauseKey, routing.PlayingState{IsPaused: true})
					if err != nil {
						fmt.Printf("Error publishing pause: %v\n", err)
					}
				case "resume":
					fmt.Println("resume command detected. Sending message...")
					err := pubsub.PublishJSON(channel, routing.ExchangePerilDirect, routing.PauseKey, routing.PlayingState{IsPaused: false})
					if err != nil {
						fmt.Printf("Error publishing resume: %v\n", err)
					}
				case "exit":
					fmt.Println("Exiting the game...")
					close(done)
					return
				default:
					fmt.Printf("I don't know what %s means\n", words[0])
				}
			}
		}
	}()

	<-done
	fmt.Println("Closing the program...")
}

func handleLogs(gamelog routing.GameLog) pubsub.Acktype {
	err := gamelogic.WriteLog(gamelog)
	if err != nil {
		return pubsub.NackRequeue
	}
	return pubsub.Ack
}
