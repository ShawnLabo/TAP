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
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"cloud.google.com/go/pubsub"
	"golang.org/x/sync/errgroup"
)

type controller struct {
	topic *pubsub.Topic
}

func newController(topic *pubsub.Topic) *controller {
	return &controller{topic}
}

type rootResponse struct {
	OK bool `json:"ok"`
}

func (c *controller) root(w http.ResponseWriter, r *http.Request) {
	res := &rootResponse{OK: true}
	respondJSON(w, http.StatusOK, res)
}

type postTemperatureRequest struct {
	Data []struct {
		Timestamp time.Time `json:"timestamp"`
		Value     string    `json:"value"`
	} `json:"data"`
}

func (c *controller) postTemperature(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		handleError(w, http.StatusBadRequest, fmt.Errorf("%s not allowed", r.Method))
		return
	}

	req := &postTemperatureRequest{}

	if err := decodeJSONBody(r, req); err != nil {
		handleError(w, http.StatusInternalServerError, fmt.Errorf("decodeJSONBody: %w", err))
		return
	}

	tl, err := temperatureListFromRequest(req)
	if err != nil {
		handleError(w, http.StatusBadRequest, fmt.Errorf("temperatureListFromRequest: %w", err))
	}

	if err := c.publish(r.Context(), tl); err != nil {
		handleError(w, http.StatusInternalServerError, fmt.Errorf("c.publish: %w", err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (c *controller) publish(ctx context.Context, tl []temperature) error {
	eg, ctx := errgroup.WithContext(ctx)

	for _, t := range tl {
		t := t
		eg.Go(func() error {
			bytes, err := json.Marshal(t)
			if err != nil {
				return fmt.Errorf("json.Marshal: %w", err)
			}

			msg := &pubsub.Message{Data: bytes}
			result := c.topic.Publish(ctx, msg)

			id, err := result.Get(ctx)
			if err != nil {
				return fmt.Errorf("result.Get: %w", err)
			}

			log.Printf("Published message: id=%s", id)

			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return fmt.Errorf("eg.Wait: %w", err)
	}

	return nil
}

func temperatureListFromRequest(req *postTemperatureRequest) ([]temperature, error) {
	tl := []temperature{}

	for _, rd := range req.Data {
		t := temperature{Timestamp: rd.Timestamp}

		v, err := strconv.ParseFloat(rd.Value, 64)
		if err != nil {
			return nil, fmt.Errorf("strconv.ParseFloat: %w", err)
		}

		t.Temperature = v

		tl = append(tl, t)
	}

	return tl, nil
}
