package main

import (
	"fmt"
	"os"
	"regexp"
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

	emailRegex, err := regexp.Compile(config.SubscriptionResponder.EmailFilterRegex)
	if err != nil {
		fmt.Println("Failed to parse email filter", config.SubscriptionResponder.EmailFilterRegex)
		os.Exit(4)
	}

	// handle messages that are already on the server
	go handleMessages(jmap, toSubscribe, emailRegex)
	// Create a channel over which we'll delete messages we've properly handled
	toDelete := make(chan Request)
	// Start a goroutine for deleting messages after they are handled
	go deleteMessages(jmap, toDelete)
	// Subscribe to incoming messages
	go subscribeToEvents(jmap, toSubscribe)
	// Poll regularly for anyone that has added themselves to the temporary mailing list
	go pollForSubscribers(listmonk, config.Listmonk.NewSubscriberList, config.Listmonk.TransactionalTemplateID, config.Listmonk.TargetList, emailRegex)
	var request Request
	for {
		request = <-toSubscribe
		subscriberID, err := getSubscriberID(listmonk, request.EmailAddress)
		// If they don't have a subscriber ID
		if subscriberID == 0 && err == nil {
			subscriberID, err = createSubscriber(listmonk, request.EmailAddress, request.EmailName, config.Listmonk.NewSubscriberList)
			if err != nil {
				fmt.Println("Failed to create subscriber", err)
				continue
			}
		}
		toDelete <- request
	}
}
