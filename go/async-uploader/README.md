# Async Uploader

## Run locally

Run receiver.

```sh
cd receiver
PUBSUB_PROJECT_ID="your-project-id" PUBSUB_TOPIC_ID="your-pubsub-topic-id" go run .
```

Post dummy data.

```sh
curl -i localhost:8080/dataSequence -X POST \
  -d '{
    "data": [
      {"timestamp": "2023-02-22T00:00:00Z", "value": "0"},
      {"timestamp": "2023-02-22T01:00:00Z", "value": "1"},
      {"timestamp": "2023-02-22T02:00:00Z", "value": "2"},
      {"timestamp": "2023-02-22T03:00:00Z", "value": "3"},
      {"timestamp": "2023-02-22T04:00:00Z", "value": "4"}
    ]
  }'
```
