#!/bin/bash

# Set minimal environment untuk testing template
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=postgres
export DB_PASS=1234
export DB_NAME=email_service

export WEBUI_ENABLED=true
export WEBUI_USERNAME=admin
export WEBUI_PASSWORD=admin123
export WEBUI_SESSION_DURATION=60

export SERVER_PORT=8080

echo "Starting email service with template debugging..."
./bin/email-service server
