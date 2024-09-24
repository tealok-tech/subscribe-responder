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
	err = connectJMAP(client, toSubscribe)
	if err != nil {
		fmt.Println("Failed to connect", err)
		os.Exit(2)
	}
	listmonk, err := connectListmonk(config.Listmonk)
	if err != nil {
		fmt.Println("Failed with listmonk", err)
		os.Exit(3)
	}
	go handleMessages(client, toSubscribe)
	go subscribeToEvents(client, toSubscribe)
	var subscriberEmail string
	for {
		subscriberEmail = <-toSubscribe
		subscriberId, err := getSubscriberID(listmonk, subscriberEmail)
		// If they don't have a subscriber ID
		if subscriberId == 0 && err == nil {
			subscribe(listmonk, subscriberEmail)
			continue
		}
		err = sendTransactional(listmonk, config.Listmonk.TransactionalTemplateID, subscriberEmail)
		if err != nil {
			fmt.Println("Failed to send transactional email to", subscriberEmail, err)
		}
	}
}
