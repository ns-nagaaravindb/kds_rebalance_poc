#!/bin/bash

STREAM_NAME="test-stream"
NEW_SHARD_COUNT=$1

if [ -z "$NEW_SHARD_COUNT" ]; then
    echo "Usage: ./scripts/reshard-stream.sh <new_shard_count>"
    echo "Example: ./scripts/reshard-stream.sh 3"
    exit 1
fi

echo "Updating stream '$STREAM_NAME' to $NEW_SHARD_COUNT shards..."

# Update shard count
docker exec localstack-kinesis awslocal kinesis update-shard-count \
    --stream-name $STREAM_NAME \
    --target-shard-count $NEW_SHARD_COUNT \
    --scaling-type UNIFORM_SCALING \
    --region us-east-1

echo ""
echo "Waiting for stream to update..."
sleep 5

echo ""
echo "Current stream status:"
docker exec localstack-kinesis awslocal kinesis describe-stream \
    --stream-name $STREAM_NAME \
    --region us-east-1 | grep -A 2 "Shards"

echo ""
echo "✅ Stream resharding complete!"
