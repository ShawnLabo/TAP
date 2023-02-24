#!/usr/bin/env bash

curl -i "${RECEIVER_URL}/temperature" -X POST \
  -H "Content-Type: application/json" \
  -d '{
    "data": [
      {"timestamp":"'$(date --utc --iso=sec)'","value":"0.0"},
      {"timestamp":"'$(date --utc --iso=sec)'","value":"1.1"},
      {"timestamp":"'$(date --utc --iso=sec)'","value":"2.2"},
      {"timestamp":"'$(date --utc --iso=sec)'","value":"3.3"},
      {"timestamp":"'$(date --utc --iso=sec)'","value":"4.4"}
    ]
  }'
