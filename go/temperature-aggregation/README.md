# Temperature Aggregation

## Run locally

Run receiver.

```sh
cd receiver
PUBSUB_PROJECT_ID="your-project-id" PUBSUB_TOPIC_ID="your-pubsub-topic-id" go run .
```

Post dummy data.

```sh
curl -i localhost:8080/temperature -X POST \
  -d '{
    "data": [
      {"timestamp": "2023-02-22T00:00:00Z", "value": "0.0"},
      {"timestamp": "2023-02-22T01:00:00Z", "value": "1.0"},
      {"timestamp": "2023-02-22T02:00:00Z", "value": "2.0"},
      {"timestamp": "2023-02-22T03:00:00Z", "value": "3.0"},
      {"timestamp": "2023-02-22T04:00:00Z", "value": "4.0"}
    ]
  }'
```
