#!/bin/bash

echo "🚀 Starting Kinesis POC..."
echo ""

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo "❌ Docker is not running. Please start Docker first."
    exit 1
fi

# Start LocalStack
echo "📦 Starting LocalStack..."
docker-compose up -d

# Wait for LocalStack to be ready
echo "⏳ Waiting for LocalStack to be ready..."
sleep 10

# Check if stream was created
echo "🔍 Verifying Kinesis stream..."
docker exec localstack-kinesis awslocal kinesis describe-stream --stream-name test-stream --region us-east-1 > /dev/null 2>&1

if [ $? -eq 0 ]; then
    echo "✅ Kinesis stream 'test-stream' is ready!"
else
    echo "⚠️  Stream not found. Creating manually..."
    docker exec localstack-kinesis awslocal kinesis create-stream --stream-name test-stream --shard-count 2 --region us-east-1
    sleep 3
fi

echo ""
echo "✅ Setup complete!"
echo ""
echo "📝 Next steps:"
echo "   1. Run producer:  cd producer && go run main.go"
echo "   2. Run consumer:  cd consumer && go run main.go"
echo ""
echo "To stop: ./stop.sh or docker-compose down"
