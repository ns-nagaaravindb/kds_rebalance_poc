#!/bin/bash

echo "Waiting for LocalStack to be ready..."
sleep 5

echo "Creating Kinesis stream 'test-stream' with 2 shards..."
awslocal kinesis create-stream \
    --stream-name test-stream \
    --shard-count 2 \
    --region us-east-1

echo "Waiting for stream to become active..."
sleep 3

echo "Describing stream..."
awslocal kinesis describe-stream \
    --stream-name test-stream \
    --region us-east-1

echo "Kinesis stream initialization completed!"
