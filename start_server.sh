#!/bin/bash

# Load environment variables from .env file
set -a
source .env
set +a

# Set playwright browsers path to permanent directory
export PLAYWRIGHT_BROWSERS_PATH=~/playwright-browsers

# Start the server
./server
