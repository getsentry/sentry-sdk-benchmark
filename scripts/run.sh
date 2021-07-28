#!/bin/bash

# docker compose -f docker-compose.yml -f platform/python/django/instrumented/docker-compose.yml -f tool/database/postgres/docker-compose.yml up -d --build 

bash ./scripts/multi-docker-compose.sh \
    -f ./docker-compose.yml \
    -f ./platform/python/django/baseline/docker-compose.yml \
    -f ./tool/database/postgres/docker-compose.yml \
    -p sentry_sdk_benchmark up
