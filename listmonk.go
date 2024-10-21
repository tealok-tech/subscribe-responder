package main

import (
	"context"
	"fmt"
	listmonk "github.com/Exayn/go-listmonk"
	"log"
	"regexp"
	"time"
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

// Create a new subscriber entry given an email address and a name
func createSubscriber(client *listmonk.Client, address string, name string, listID uint) (uint, error) {
	service := client.NewCreateSubscriberService()
	service.Email(address)
	service.Name(name)
	service.ListIds([]uint{listID})
	ctx := context.Background()
	subscriber, err := service.Do(ctx)
	if err != nil {
		return 0, fmt.Errorf("Failed to subscribe", err)
	}
	return subscriber.Id, nil
}

func pollForSubscribers(client *listmonk.Client, sourceListID uint, templateID uint, targetListID uint, emailRegex *regexp.Regexp) error {
	for {
		listService := client.NewGetSubscribersService()
		listService.ListIds([]uint{4}) // sourceListID})
		ctx := context.Background()
		subscribers, err := listService.Do(ctx)
		if err != nil {
			log.Println("Failed to get subscribers", err)
			time.Sleep(time.Duration(3 * 1000000))
			continue
		}
		toUpdate := []uint{}
		for _, sub := range subscribers {
			if emailRegex != nil {
				if !emailRegex.MatchString(sub.Email) {
					// log.Println("Ignoring subscription from", sub.Email, "as it does not match the regex")
					continue
				}
			}
			log.Println("Adding", sub.Email, "(", sub.Id, ") to the addresses to update")
			toUpdate = append(toUpdate, sub.Id)
		}
		updateService := client.NewUpdateSubscribersListsService()
		if len(toUpdate) == 0 {
			goto end
		}
		updateService.Ids(toUpdate)
		// First add to the target list so we don't orphan members if there is a failure
		updateService.Action("add")
		updateService.ListIds([]uint{targetListID})
		_, err = updateService.Do(ctx)
		if err != nil {
			log.Println("Failed to add subscribers to target list", err)
			goto end
		}
		log.Println("Added emails to the target list", targetListID)
		// Then remove from the new subscriber list
		updateService.Action("remove")
		updateService.ListIds([]uint{sourceListID})
		_, err = updateService.Do(ctx)
		if err != nil {
			log.Println("Failed to remove subscribers from source list", err)
			goto end
		}
		log.Println("Removed emails from the source list", sourceListID)
		// Then send the transactional email
		for _, subscriberID := range toUpdate {
			err = sendTransactional(client, templateID, subscriberID)
			if err != nil {
				log.Println("Failed to send transactional email", templateID, "to subscriber", subscriberID, ":", err)
				goto end
			}
		}
	end:
		time.Sleep(time.Duration(10 * 1_000_000_000))
	}
}
func sendTransactional(client *listmonk.Client, templateID uint, subscriberID uint) error {
	service := client.NewPostTransactionalService()
	service.SubscriberId(subscriberID)
	service.ContentType("html")
	service.TemplateId(templateID)
	ctx := context.Background()
	err := service.Do(ctx)
	if err != nil {
		return fmt.Errorf("Failed to send transactional email", err)
	}
	log.Println("Sent transactional email", templateID, "to subscriber", subscriberID)
	return nil
}
