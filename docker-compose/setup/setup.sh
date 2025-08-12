#!/bin/bash

kafka-topics \
  --bootstrap-server kafka:9092 \
  --create \
  --topic health_checks_events \
  --partitions 3 \
  --replication-factor 1

kafka-topics \
  --bootstrap-server kafka:9092 \
  --create \
  --topic server_checks_events \
  --partitions 3 \
  --replication-factor 1

curl -X POST -H "Content-Type: application/json" --data @/setup/source.json http://source-connector:8083/connectors

curl -X PUT "http://es:9200/health_checks" -H "Content-Type: application/json" -d @/setup/health_checks_index.json
