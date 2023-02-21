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

type postDataSequenceRequest struct {
	Data []struct {
		Timestamp time.Time `json:"timestamp"`
		Value     string    `json:"value"`
	} `json:"data"`
}

func (c *controller) postDataSequence(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		handleError(w, http.StatusBadRequest, fmt.Errorf("%s not allowed", r.Method))
		return
	}

	req := &postDataSequenceRequest{}

	if err := decodeJSONBody(r, req); err != nil {
		handleError(w, http.StatusInternalServerError, fmt.Errorf("decodeJSONBody: %w", err))
		return
	}

	seq, err := dataSequenceFromRequest(req)
	if err != nil {
		handleError(w, http.StatusBadRequest, fmt.Errorf("dataSequenceFromRequest: %w", err))
	}

	if err := c.publish(r.Context(), seq); err != nil {
		handleError(w, http.StatusInternalServerError, fmt.Errorf("c.publish: %w", err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (c *controller) publish(ctx context.Context, seq *dataSequence) error {
	bytes, err := json.Marshal(seq)
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
}

func dataSequenceFromRequest(req *postDataSequenceRequest) (*dataSequence, error) {
	seq := &dataSequence{Data: []*data{}}

	for _, rd := range req.Data {
		d := &data{Timestamp: rd.Timestamp}

		v, err := strconv.ParseFloat(rd.Value, 64)
		if err != nil {
			return nil, fmt.Errorf("strconv.ParseFloat: %w", err)
		}

		d.Value = v

		seq.Data = append(seq.Data, d)
	}

	return seq, nil
}
