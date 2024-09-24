package main

import (
	"fmt"
	"os"
)

func main() {
	config, err := readConfig()
	if err != nil {
		fmt.Println("Failed to read config", err)
		os.Exit(1)
	}
	client, err := jmapClient(config.JMAP)
	if err != nil {
		fmt.Println("Failed to create JMAP client", err)
		os.Exit(2)
	}

	// Create a channel over which we'll get various emails we should respond to.
	var toSubscribe chan string
	toSubscribe = make(chan string)
	err = connect(client, toSubscribe)
	if err != nil {
		fmt.Println("Failed to connect", err)
		os.Exit(2)
	}
	// Collect up any waiting emails in our mailbox
	go handleMessages(client, toSubscribe)
	go subscribeToEvents(client, toSubscribe)
	var subscriberEmail string
	for {
		subscriberEmail = <-toSubscribe
		fmt.Println("Pretend response for", subscriberEmail)
	}
}
