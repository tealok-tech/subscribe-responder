package main

import (
	"context"
	"fmt"
	listmonk "github.com/Exayn/go-listmonk"
	"log"
)

func connectListmonk(config ListmonkConfig) (*listmonk.Client, error) {
	log.Println("Endpoint", config.BaseURL)
	client := listmonk.NewClient(
		config.BaseURL,
		&config.Username,
		&config.Password,
	)
	ctx := context.Background()
	service := client.NewGetHealthService()
	result, err := service.Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("Failed in listmonk", err)
	}
	log.Println("Listmonk health", *result)
	return client, nil
}

func sendTransactional(client *listmonk.Client, email string) {
	log.Println("Fake send transactional")
}
