.PHONY: help start stop build clean producer consumer consumer-w1 consumer-w2 consumer-w3 reshard test

help:
	@echo "Available commands:"
	@echo "  make start        - Start LocalStack"
	@echo "  make stop         - Stop LocalStack"
	@echo "  make build        - Build producer and consumer"
	@echo "  make produce      - Run the producer"
	@echo "  make consume      - Run the consumer (default config)"
	@echo "  make consumer-w1  - Run consumer worker-1 (shard 0)"
	@echo "  make consumer-w2  - Run consumer worker-2 (shard 1)"
	@echo "  make consumer-w3  - Run consumer worker-3 (shards 2,3)"
	@echo "  make reshard      - Add shards to stream (usage: make reshard SHARDS=3)"
	@echo "  make clean        - Clean up build artifacts"
	@echo "  make test         - Test the setup"

start:
	@./scripts/start.sh

stop:
	@./scripts/stop.sh

build:
	@echo "Building producer..."
	@cd producer && go build -o ../bin/producer main.go
	@echo "Building consumer..."
	@cd consumer && go build -o ../bin/consumer main.go
	@echo "✅ Build complete!"

produce:
	@cd producer && go run main.go

consumer:
	@cd consumer && go run main.go

consumer1:
	@echo "Starting Consumer Worker 1 (shardId-000000000000)..."
	@cd consumer && CONFIG_FILE=../config-worker1.yaml go run main.go

consumer2:
	@echo "Starting Consumer Worker 2 (shardId-000000000001)..."
	@cd consumer && CONFIG_FILE=../config-worker2.yaml go run main.go

consumer3:
	@echo "Starting Consumer Worker 3 (shardId-000000000002, shardId-000000000003)..."
	@cd consumer && CONFIG_FILE=../config-worker3.yaml go run main.go

reshard:
	@./scripts/reshard-stream.sh $(SHARDS)

clean:
	@echo "Cleaning up..."
	@rm -rf bin/
	@rm -rf localstack-data/
	@echo "✅ Cleanup complete!"

test:
	@echo "Testing Go compilation..."
	@cd producer && go build -o /dev/null main.go && echo "✅ Producer compiles"
	@cd consumer && go build -o /dev/null main.go && echo "✅ Consumer compiles"
	@echo "Testing Docker setup..."
	@docker-compose config > /dev/null && echo "✅ Docker Compose config valid"
	@echo ""
	@echo "All tests passed! ✅"
