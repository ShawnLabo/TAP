package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"cloud.google.com/go/pubsub"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	projectID := os.Getenv("PUBSUB_PROJECT_ID")
	topicID := os.Getenv("PUBSUB_TOPIC_ID")

	topic := mustPubSubTopic(projectID, topicID)

	controller := newController(topic)
	handler := newHandler(controller)

	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatalf("http.ListenAndServe: %s", err)
	}
}

func mustPubSubTopic(projectID, topicID string) *pubsub.Topic {
	client, err := pubsub.NewClient(context.Background(), projectID)
	if err != nil {
		log.Fatalf("pubsub.NewClient: %s", err)
	}

	return client.Topic(topicID)
}
