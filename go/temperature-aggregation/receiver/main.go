// Copyright 2023 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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

	log.Printf("Listening on %s", port)

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
