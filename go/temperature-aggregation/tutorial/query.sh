#!/usr/bin/env bash

project_id="$1"

bq query \
  --project_id="$project_id" \
  --use_legacy_sql=false \
  'SELECT * 
   FROM `'$project_id'.raw_data.temperature`
   WHERE publish_time >= "'$(date --utc --iso)'"'
