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

func getSubscriberID(client *listmonk.Client, email string) (uint, error) {
	service := client.NewGetSubscribersService()
	query := fmt.Sprintf("subscribers.email='%s'", email)
	service.Query(query)
	ctx := context.Background()
	result, err := service.Do(ctx)
	if err != nil {
		return 0, fmt.Errorf("Failed to get subscriber", email)
	}
	log.Println("Got", len(result), "records searching for a subscriber for", email)
	for _, sub := range result {
		if sub.Email == email {
			return sub.Id, nil
		} else {
			log.Println("Record", sub.Id, "has the wrong email", sub.Email)
		}
	}
	return 0, nil
}

func subscribe(client *listmonk.Client, email string) error {
	log.Println("Fake subscription to listmonk for", email)
	return nil
}
func sendTransactional(client *listmonk.Client, templateID uint, email string) error {
	service := client.NewPostTransactionalService()
	service.SubscriberEmail(email)
	service.TemplateId(templateID)
	ctx := context.Background()
	err := service.Do(ctx)
	if err != nil {
		return fmt.Errorf("Failed to send transactional email", err)
	}
	return nil
}
