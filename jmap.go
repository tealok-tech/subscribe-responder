package main

import (
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"

	"git.sr.ht/~rockorager/go-jmap"
	"git.sr.ht/~rockorager/go-jmap/core"
	"git.sr.ht/~rockorager/go-jmap/core/push"
	"git.sr.ht/~rockorager/go-jmap/mail"
	"git.sr.ht/~rockorager/go-jmap/mail/email"
	"git.sr.ht/~rockorager/go-jmap/mail/mailbox"
)

type JMAPClient struct {
	client *jmap.Client

	EmailState     string
	MailboxIDs     []jmap.ID
	TrashMailboxID jmap.ID
}

func jmapClient(config JMAPConfig) (*JMAPClient, error) {
	if config.Password != "" && config.Username == "" {
		return nil, errors.New("You must specify both the username and password, not just the password")
	}
	if config.Password == "" && config.Username != "" {
		return nil, errors.New("You must specify both the username and password, not just the username")
	}

	// If we don't have an endpoint, and the username looks like an email address, try autodetection
	var username_parts = strings.Split(config.Username, "@")
	var endpoint = config.Endpoint
	if len(username_parts) > 1 {
		domain := username_parts[1]
		log.Print("Looking up endpoint for domain", domain)
		var err error
		endpoint, err = core.Discover(domain)
		log.Println("Discovered endpoint", endpoint)
		if err != nil {
			return nil, fmt.Errorf("Failed to detect endpoint: %w", err)
		}
	}
	// Create a new client. The SessionEndpoint must be specified for
	// initial connections.
	if endpoint == "" {
		return nil, errors.New("No endpoint specified and unable to detect endpoint")
	}
	log.Println("Using endpoint", endpoint)
	c := &jmap.Client{
		SessionEndpoint: endpoint,
	}
	result := JMAPClient{c, "", []jmap.ID{}, ""}

	// Set the authentication mechanism. This also sets the HttpClient of
	// the jmap client
	// client.WithAccessToken("my-access-token")
	if config.Password != "" && config.Username != "" {
		log.Print("Using basic auth")
		c.WithBasicAuth(config.Username, config.Password)
	} else {
		return nil, errors.New("No authentication provided")
	}
	return &result, nil
}

func connectJMAP(client *JMAPClient) error {
	// Authenticate the client. This gets a Session object. Session objects
	// are cacheable, and have their own state string clients can use to
	// decide when to refresh. The client can be initialized with a cached
	// Session object. If one isn't available, the first request will also
	// authenticate the client
	if err := client.client.Authenticate(); err != nil {
		return fmt.Errorf("Failed to authenticate: %w", err)
	}
	//log.Println("Got a client. Session", client.client.Session)

	// Get the various mailboxes we'll need
	id := client.client.Session.PrimaryAccounts[mail.URI]
	req := &jmap.Request{}

	req.Invoke(&mailbox.Get{
		Account: id,
	})

	resp, err := client.client.Do(req)
	if err != nil {
		return fmt.Errorf("Failed to get mailboxes", err)
	}

	// Enqueue a handler for any emails sitting around
	for _, inv := range resp.Responses {
		switch r := inv.Args.(type) {
		case *mailbox.GetResponse:
			for _, m := range r.List {
				client.MailboxIDs = append(client.MailboxIDs, m.ID)
				if m.Role == mailbox.RoleTrash {
					client.TrashMailboxID = m.ID
				}
			}
		}
	}
	return nil
}

// Goroutine for deleting messages
func deleteMessages(client *JMAPClient, toDelete chan Request) {
	accountID := client.client.Session.PrimaryAccounts[mail.URI]
	var request Request
	for {
		request = <-toDelete

		req := &jmap.Request{}
		mailboxIDs := trashMailboxIDs(client)
		req.Invoke(&email.Set{
			Account: accountID,
			Update: map[jmap.ID]jmap.Patch{
				request.EmailID: {
					"mailboxIds": mailboxIDs,
				},
			},
		})
		resp, err := client.client.Do(req)
		if err != nil {
			log.Println("Failed to delete email", request.EmailID, resp, err)
		} else {
			log.Println("Deleted email", request.EmailID)
		}
	}
}

// Make a map of mailbox IDs to booleans where all mailbox IDs are mapped to 'false'
// except the trash mailbox. This will effectively delete an email.
func trashMailboxIDs(client *JMAPClient) map[jmap.ID]bool {
	result := make(map[jmap.ID]bool, len(client.MailboxIDs)+1)
	for _, m := range client.MailboxIDs {
		result[m] = false
	}
	result[client.TrashMailboxID] = true
	return result
}

// Search for existing email messages to our subscribe mailbox.
// For any messages we find we'll put a subscription request into the toSubscribe channel.
func handleMessages(client *JMAPClient, toSubscribe chan<- Request, emailFilterRegex *regexp.Regexp) error {
	// Get the account ID of the primary mail account
	id := client.client.Session.PrimaryAccounts[mail.URI]

	// Create a new request
	req := &jmap.Request{}

	// Invoke a result reference call
	req.Invoke(&email.Get{
		Account: id,
	})

	// Make the request
	resp, err := client.client.Do(req)
	if err != nil {
		// Handle the error
		return fmt.Errorf("Failed to handle messages", err)
	}

	// Enqueue a handler for any emails sitting around
	for _, inv := range resp.Responses {
		// Our result to individual calls is in the Args field of the
		// invocation
		switch r := inv.Args.(type) {
		case *email.GetResponse:
			client.EmailState = r.State
			for _, eml := range r.List {
				if eml.MailboxIDs[client.TrashMailboxID] {
					continue
				}
				log.Println("Subject:", eml.Subject)
				for _, f := range eml.From {
					log.Println("Email from:", f.Email)
					if emailFilterRegex != nil {
						if !emailFilterRegex.MatchString(f.Email) {
							log.Println("Email does not match filter, ignoring.")
							continue
						}
					}
					toSubscribe <- Request{
						f.Email,
						eml.ID,
						f.Name,
					}
				}
			}
		}
	}
	return nil
}

func subscribeToEvents(client *JMAPClient, toSubscribe chan Request) error {
	var eventSource push.EventSource
	eventSource.Client = client.client
	eventSource.Handler = func(change *jmap.StateChange) {
		for accountId, state := range change.Changed {
			for key, value := range state {
				log.Println("Account", accountId, key, "has new state", value)
				if key == "Email" {
					if value != client.EmailState {
						GetEmailChanges(client, toSubscribe)
					}
				}
			}
		}
	}
	eventSource.Ping = 10
	eventSource.CloseAfterState = false
	log.Println("Listening for events from JMAP server")
	eventSource.Listen()
	log.Println("Exiting JMAP listen event loop")
	return nil
}

// Get all of the email chainges since a particular
func GetEmailChanges(client *JMAPClient, toSubscribe chan Request) error {
	id := client.client.Session.PrimaryAccounts[mail.URI]
	req := &jmap.Request{}
	log.Println("Getting email changes since", client.EmailState)
	callID := req.Invoke(&email.Changes{
		Account:    id,
		SinceState: client.EmailState,
	})
	req.Invoke(&email.Get{
		Account: id,
		ReferenceIDs: &jmap.ResultReference{
			ResultOf: callID,
			Name:     "Email/changes",
			Path:     "/created",
		},
	})

	resp, err := client.client.Do(req)
	if err != nil {
		return fmt.Errorf("Failed to get changes", err)
	}
	// Enqueue a handler for any emails sitting around
	for _, inv := range resp.Responses {
		// Our result to individual calls is in the Args field of the
		// invocation
		switch r := inv.Args.(type) {
		// We don't care about the changes response
		//case *email.ChangesResponse:
		case *email.GetResponse:
			for _, eml := range r.List {
				log.Println("Email subject:", eml.Subject)
				for _, f := range eml.From {
					toSubscribe <- Request{
						f.Email,
						eml.ID,
						f.Name,
					}
				}
			}
		}

	}
	return nil
}

func subscribeSender(sender *mail.Address) {
	fmt.Println("Subscribing ", sender.Email)
}

func moveToTrash(client *JMAPClient, email *email.Email) {
	fmt.Println("Moving", email.ID, "to trash")

}
