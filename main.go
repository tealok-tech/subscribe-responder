package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"git.sr.ht/~rockorager/go-jmap"
	"git.sr.ht/~rockorager/go-jmap/core"
	"git.sr.ht/~rockorager/go-jmap/core/push"
	"git.sr.ht/~rockorager/go-jmap/mail"
	"git.sr.ht/~rockorager/go-jmap/mail/email"
	"git.sr.ht/~rockorager/go-jmap/mail/mailbox"
)

func main() {
	var endpoint = ""
	var password = ""
	var username = ""

	flag.StringVar(&endpoint, "endpoint", "https://api.fastmail.com/jmap/session", "The endpoint to use when communicating with the JMAP server")
	flag.StringVar(&password, "password", "", "Use basic authentication, this is the password")
	flag.StringVar(&username, "username", "", "Use basic authentication, this is the username")
	flag.Parse()

	if password != "" && username == "" {
		fmt.Printf("You must specify both the username and password, not just the password")
		os.Exit(1)
	}
	if password == "" && username != "" {
		fmt.Printf("You must specify both the username and password, not just the username")
		os.Exit(1)
	}

	// If we don't have an endpoint, and the username looks like an email address, try autodetection
	var username_parts = strings.Split(username, "@")
	if len(username_parts) > 1 {
		domain := username_parts[1]
		log.Print("Looking up endpoint for domain", domain)
		var err error
		endpoint, err = core.Discover(domain)
		if err != nil {
			fmt.Printf("Failed to detect endpoint: ", err)
			os.Exit(2)
		}
	}
	// Create a new client. The SessionEndpoint must be specified for
	// initial connections.
	if endpoint == "" {
		fmt.Printf("No endpoint specified and unable to detect endpoint")
		os.Exit(2)
	}
	client := &jmap.Client{
		SessionEndpoint: endpoint,
	}

	// Set the authentication mechanism. This also sets the HttpClient of
	// the jmap client
	// client.WithAccessToken("my-access-token")
	if password != "" && username != "" {
		log.Print("Using basic auth")
		client.WithBasicAuth(username, password)
	} else {
		log.Fatal("No authentication provided")
		os.Exit(3)
	}

	// Authenticate the client. This gets a Session object. Session objects
	// are cacheable, and have their own state string clients can use to
	// decide when to refresh. The client can be initialized with a cached
	// Session object. If one isn't available, the first request will also
	// authenticate the client
	if err := client.Authenticate(); err != nil {
		fmt.Printf("Failed to authenticate: ", err)
		os.Exit(4)
	}

	// Get the account ID of the primary mail account
	id := client.Session.PrimaryAccounts[mail.URI]

	// Create a new request
	req := &jmap.Request{}

	// Invoke a method. The CallID of this method will be returned to be
	// used when chaining calls
	req.Invoke(&mailbox.Get{
		Account: id,
	})

	// Invoke a changes call, let's save the callID and pass it to a Get
	// method
	callID := req.Invoke(&email.Changes{
		Account:    id,
		SinceState: "some-known-state",
	})

	// Invoke a result reference call
	req.Invoke(&email.Get{
		Account: id,
		ReferenceIDs: &jmap.ResultReference{
			ResultOf: callID,          // The CallID of the referenced method
			Name:     "Email/changes", // The name of the referenced method
			Path:     "/created",      // JSON pointer to the location of the reference
		},
	})

	// Make the request
	resp, err := client.Do(req)
	if err != nil {
		// Handle the error
	}

	// Loop through the responses to invidividual invocations
	for _, inv := range resp.Responses {
		// Our result to individual calls is in the Args field of the
		// invocation
		switch r := inv.Args.(type) {
		case *mailbox.GetResponse:
			// A GetResponse contains a List of the objects
			// retrieved
			for _, mbox := range r.List {
				fmt.Println("Mailbox name:", mbox.Name)
				fmt.Println("Total email:", mbox.TotalEmails)
				fmt.Println("Unread email:", mbox.UnreadEmails)
			}
		case *email.GetResponse:
			for _, eml := range r.List {
				fmt.Println("Email subject:", eml.Subject)
			}
		}
		// There is a response in here to the Email/changes call, but we
		// don't care about the results since we passed them to the
		// Email/get call
	}

	var eventSource push.EventSource
	eventSource.Client = client
	eventSource.Handler = handler
	eventSource.Ping = 10
	eventSource.CloseAfterState = false
	fmt.Println("Listening")
	eventSource.Listen()
	fmt.Println("Exiting")
}

func handler(change *jmap.StateChange) {
	fmt.Println("Go state change", change)
}
