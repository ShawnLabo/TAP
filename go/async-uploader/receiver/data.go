package main

import (
	"time"
)

type dataSequence struct {
	Data []*data `json:"data"`
}

type data struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}
