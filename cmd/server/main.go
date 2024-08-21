package main

import (
	"fmt"
	"os"
	"os/signal"

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
	fmt.Printf("Connected successfuly!")
	defer connection.Close()

	// wait for ctrl+c
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	<-signalChan
	fmt.Println("\nClosing the program...")

}
