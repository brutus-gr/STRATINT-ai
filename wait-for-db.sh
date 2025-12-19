#!/bin/bash

# Wait for Cloud SQL instance to be ready

# Configure these for your environment
PROJECT_ID="${GCP_PROJECT_ID:?Error: Set GCP_PROJECT_ID environment variable}"
INSTANCE_NAME="${CLOUD_SQL_INSTANCE:-osint-db}"

echo "Waiting for Cloud SQL instance '$INSTANCE_NAME' to be ready..."
echo ""

while true; do
    STATE=$(gcloud sql instances describe $INSTANCE_NAME \
        --project=$PROJECT_ID \
        --format="value(state)" 2>/dev/null)

    if [ "$STATE" = "RUNNABLE" ]; then
        echo ""
        echo "✓ Instance is RUNNABLE and ready!"
        echo ""
        gcloud sql instances describe $INSTANCE_NAME \
            --project=$PROJECT_ID \
            --format="table(name,databaseVersion,state,ipAddresses[0].ipAddress)"
        echo ""
        echo "You can now run: ./setup-db.sh"
        exit 0
    else
        echo -n "."
        sleep 5
    fi
done
