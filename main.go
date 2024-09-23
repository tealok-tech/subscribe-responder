package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	var endpoint = ""
	var password = ""
	var username = ""

	flag.StringVar(&endpoint, "endpoint", "https://api.fastmail.com/jmap/session", "The endpoint to use when communicating with the JMAP server")
	flag.StringVar(&password, "password", "", "Use basic authentication, this is the password")
	flag.StringVar(&username, "username", "", "Use basic authentication, this is the username")
	flag.Parse()

	client, err := jmapClient(password, username, endpoint)
	if err != nil {
		fmt.Println("Failed to create JMAP client", err)
		os.Exit(1)
	}

	// Create a channel over which we'll get various emails we should respond to.
	var toSubscribe chan string
	toSubscribe = make(chan string)
	err = connect(client, toSubscribe)
	if err != nil {
		fmt.Println("Failed to connect", err)
		os.Exit(2)
	}

}
