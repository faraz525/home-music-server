#!/bin/bash
# Helper script to run the backend locally with the correct data directory
# This ensures we use the same data location as docker-compose.override.yml

cd "$(dirname "$0")"
DATA_DIR=../data/cratedrop go run .

