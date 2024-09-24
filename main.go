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

	err = connectJMAP(client)
	if err != nil {
		fmt.Println("Failed to connect", err)
		os.Exit(2)
	}
	listmonk, err := connectListmonk(config.Listmonk)
	if err != nil {
		fmt.Println("Failed with listmonk", err)
		os.Exit(3)
	}
	// Create a channel over which we'll get various emails we should respond to.
	toSubscribe := make(chan Request)
	go handleMessages(client, toSubscribe)
	// Create a channel over which we'll delete messages we've properly handled
	toDelete := make(chan Request)
	go deleteMessages(client, toDelete)
	go subscribeToEvents(client, toSubscribe)
	var request Request
	for {
		request = <-toSubscribe
		subscriberID, err := getSubscriberID(listmonk, request.EmailAddress)
		// If they don't have a subscriber ID
		if subscriberID == 0 && err == nil {
			subscriberID, err = subscribe(listmonk, request.EmailAddress, request.EmailName)
			if err != nil {
				fmt.Println("Failed to subscribe", err)
				continue
			}
		}
		err = sendTransactional(listmonk, config.Listmonk.TransactionalTemplateID, subscriberID)
		if err != nil {
			fmt.Println("Failed to send transactional email to", subscriberID, err)
		}
		toDelete <- request
	}
}
