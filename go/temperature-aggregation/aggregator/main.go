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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/datastore"
	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
)

const (
	datastoreKind = "aggregator"
	datastoreKey  = "lastExecution"
)

var (
	errJobAlreadyFinished = errors.New("job already finished")
)

type lastExecution struct {
	RangeStart    time.Time
	RangeEnd      time.Time
	ExecutionTime time.Time
}

type temperature struct {
	Timestamp   time.Time `json:"timestamp"`
	Temperature float64   `json:"temperature"`
}

type jobExecutor struct {
	bucket    *storage.BucketHandle
	bigquery  *bigquery.Client
	datastore *datastore.Client

	dataset string
	table   string
	key     *datastore.Key
}

func newJobExecutor(ctx context.Context, bucketName, bqProjectID, bqDatasetID, bqTableID, dsProjectID string) *jobExecutor {
	sClient, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatalf("storage.NewClient: %s", err)
	}

	bqClient, err := bigquery.NewClient(ctx, bqProjectID)
	if err != nil {
		log.Fatalf("bigquery.NewClient: %s", err)
	}

	dsClient, err := datastore.NewClient(ctx, dsProjectID)
	if err != nil {
		log.Fatalf("datastore.NewClient: %s", err)
	}

	return &jobExecutor{
		bucket:    sClient.Bucket(bucketName),
		bigquery:  bqClient,
		datastore: dsClient,
		dataset:   bqDatasetID,
		table:     bqTableID,
		key:       datastore.NameKey(datastoreKind, datastoreKey, nil),
	}
}

func (ex *jobExecutor) execute(ctx context.Context, now time.Time) error {
	le, err := ex.getLastExecution(ctx)
	if err != nil {
		return fmt.Errorf("ex.getLastExecutionTime: %w", err)
	}

	log.Printf("last execution: %+v", le)

	start, end, err := ex.calcNextRange(le, now)
	if err != nil {
		return fmt.Errorf("ex.calcNextRange: %w", err)
	}

	log.Printf("range: [%v, %v)", start, end)

	log.Printf("running query")

	data, err := ex.getData(ctx, start, end)
	if err != nil {
		return fmt.Errorf("ex.getData: %w", err)
	}

	objName := end.Format("20060102150405.jsonl")

	log.Printf("uploading data to %s", objName)

	if err := ex.uploadData(ctx, objName, data); err != nil {
		return fmt.Errorf("ex.uploadData: %w", err)
	}

	log.Printf("storing this execution info to datastore")

	if err := ex.storeLastExecution(ctx, start, end, now); err != nil {
		return fmt.Errorf("ex.storeLastExecution: %w", err)
	}

	return nil
}

func (ex *jobExecutor) getLastExecution(ctx context.Context) (*lastExecution, error) {
	le := &lastExecution{}

	if err := ex.datastore.Get(ctx, ex.key, le); err == datastore.ErrNoSuchEntity {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("ex.datastore.Get: %w", err)
	}

	return le, nil
}

func (ex *jobExecutor) calcNextRange(le *lastExecution, now time.Time) (time.Time, time.Time, error) {
	var end time.Time

	if le == nil {
		end = now.Truncate(time.Hour)
	} else {
		end = le.RangeEnd.Add(time.Hour).Truncate(time.Hour)

		if end.After(now) {
			return time.Time{}, time.Time{}, errJobAlreadyFinished
		}
	}

	return end.Add(-time.Hour).UTC(), end.UTC(), nil
}

func (ex *jobExecutor) getData(ctx context.Context, start, end time.Time) ([]temperature, error) {
	q := ex.bigquery.Query(fmt.Sprintf(`
    SELECT timestamp, temperature
    FROM `+"`%s.%s.%s`"+`
    WHERE @start <= publish_time AND publish_time < @end
  `, ex.bigquery.Project(), ex.dataset, ex.table))
	q.Parameters = []bigquery.QueryParameter{
		{
			Name:  "start",
			Value: start,
		},
		{
			Name:  "end",
			Value: end,
		},
	}

	it, err := q.Read(ctx)
	if err != nil {
		return nil, fmt.Errorf("q.Read: %w", err)
	}

	ts := []temperature{}

	for {
		var t temperature

		err := it.Next(&t)
		if err == iterator.Done {
			break
		} else if err != nil {
			return nil, fmt.Errorf("it.Next: %w", err)
		}

		ts = append(ts, t)
	}

	return ts, nil
}

func (ex *jobExecutor) uploadData(ctx context.Context, objName string, data []temperature) error {
	buf := &bytes.Buffer{}

	for _, m := range data {
		b, err := json.Marshal(m)
		if err != nil {
			return fmt.Errorf("json.Marshal: %+v: %w", m, err)
		}

		if _, err := buf.Write(append(b, '\n')); err != nil {
			return fmt.Errorf("buf.Write: %w", err)
		}
	}

	obj := ex.bucket.Object(objName)
	w := obj.NewWriter(ctx)

	if _, err := w.Write(buf.Bytes()); err != nil {
		return fmt.Errorf("w.Write: %s", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("w.Close: %w", err)
	}

	return nil
}

func (ex *jobExecutor) storeLastExecution(ctx context.Context, start, end, now time.Time) error {
	le := &lastExecution{
		RangeStart:    start,
		RangeEnd:      end,
		ExecutionTime: now,
	}

	if _, err := ex.datastore.Put(ctx, ex.key, le); err != nil {
		return fmt.Errorf("ex.datastore.Put: %w", err)
	}

	return nil
}

func main() {
	ctx := context.Background()

	ex := newJobExecutor(ctx,
		os.Getenv("STORAGE_BUCKET_NAME"),
		os.Getenv("BIGQUERY_PROJECT_ID"),
		os.Getenv("BIGQUERY_DATASET_ID"),
		os.Getenv("BIGQUERY_TABLE_ID"),
		os.Getenv("DATASTORE_PROJECT_ID"),
	)

	now := time.Now()

	if err := ex.execute(ctx, now); errors.Is(err, errJobAlreadyFinished) {
		log.Printf("job already finished")
	} else if err != nil {
		log.Fatalf("ex.execute: %s", err)
	} else {
		log.Println("job successfully finished 🥳")
	}
}
