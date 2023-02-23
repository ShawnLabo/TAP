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

## References

* BigQuery
  * [Specifying a schema  |  BigQuery  |  Google Cloud](https://cloud.google.com/bigquery/docs/schemas#specifying_a_json_schema_file)
* Pub/Sub
  * [Creating and managing schemas  |  Cloud Pub/Sub Documentation  |  Google Cloud](https://cloud.google.com/pubsub/docs/schemas)
  * [BigQuery subscriptions  |  Cloud Pub/Sub Documentation  |  Google Cloud](https://cloud.google.com/pubsub/docs/bigquery)
  * [Specification | Apache Avro](https://avro.apache.org/docs/1.11.1/specification/_print/#schema-declaration)
  * [Create and use subscriptions  |  Cloud Pub/Sub Documentation  |  Google Cloud](https://cloud.google.com/pubsub/docs/create-subscription#assign_bigquery_service_account)
