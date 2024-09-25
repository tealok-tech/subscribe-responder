package main

import (
	"git.sr.ht/~rockorager/go-jmap"
)

type Request struct {
	// The email address for the request
	EmailAddress string
	// The ID of the email message this request came frome
	EmailID jmap.ID
	// The name of the person, if provided
	EmailName string
}
