# Kinesis Data Streams POC with LocalStack

A proof-of-concept implementation demonstrating AWS Kinesis Data Streams using LocalStack and Go, featuring a **customized VMware KCL library** with manual shard assignment capabilities for precise control over shard-to-worker mapping.

## Features

- **LocalStack Integration**: Run Kinesis and DynamoDB locally using Docker
- **Go Producer**: Generates and publishes test data to Kinesis streams
- **Go Consumer with VMware KCL**: Uses VMware's Kinesis Client Library (v1.5.1) for reliable stream processing
- **Customized KCL Library**: Fork of VMware KCL with manual shard assignment support
- **Dual Assignment Modes**: 
  - **KCL Mode**: Automatic shard rebalancing across workers
  - **Manual Mode**: Explicit shard-to-worker mapping without automatic rebalancing


### Assignment Modes

####  KCL Mode (manual Rebalancing)

```yaml
consumer:
  assignment_mode: kcl
  application_name: kds-rebalance-consumer
  worker_id: worker-1
```
**Implementation:**
```go
kclConfig.EnableManualShardMapping = true
kclConfig.WithManualShardMapping(map[string]string{
    "shardId-000000000000": "worker-1",
    "shardId-000000000001": "worker-2",
    "shardId-000000000002": "worker-3",
    "shardId-000000000003": "worker-3",  // worker-3 handles 2 shards
})
```

### Consumer Output (KCL Manual Mode)

```
INFO[2025-11-13T19:24:31+05:30] Starting worker event loop.                  
INFO[2025-11-13T19:24:31+05:30] Found new shard with id shardId-000000000000 
INFO[2025-11-13T19:24:31+05:30] Found new shard with id shardId-000000000001 
INFO[2025-11-13T19:24:31+05:30] Found new shard with id shardId-000000000002 
INFO[2025-11-13T19:24:31+05:30] Found new shard with id shardId-000000000003 
INFO[2025-11-13T19:24:31+05:30] Found 4 shards                               
DEBU[2025-11-13T19:24:31+05:30] Shard shardId-000000000001 is mapped to worker worker-2, but this is worker worker-1, skipping 
DEBU[2025-11-13T19:24:31+05:30] Shard shardId-000000000002 is mapped to this worker worker-1, attempting to acquire lease 
DEBU[2025-11-13T19:24:31+05:30] Retrieved Shard Iterator 49668922566113071865993009184509502859981837721493569570 
DEBU[2025-11-13T19:24:31+05:30] Attempting to get a lock for shard: shardId-000000000002, leaseTimeout: 2025-11-13 13:54:37 +0000 UTC, assignedTo: worker-1, newAssignedTo: worker-1 
INFO[2025-11-13T19:24:31+05:30] Start polling shard consumer for shard: shardId-000000000002 
DEBU[2025-11-13T19:24:31+05:30] Retrieved Shard Iterator SHARD_END           
...
```


### Go module issues with customized KCL

```bash
# Clean and reinstall
rm -f go.sum
go clean -modcache
go mod download
go mod tidy

# Verify replace directive exists
grep "replace github.com/vmware/vmware-go-kcl" go.mod
```

Expected output:
```
replace github.com/vmware/vmware-go-kcl => github.com/ns-nagaaravindb/vmware-go-kcl v1.5.1
```


## Available Make Commands

```bash
make help           # Show all available commands
make start          # Start LocalStack
make stop           # Stop LocalStack
make build          # Build producer and consumer binaries
make produce        # Run producer
make consumer       # Run consumer (default config)
make reshard        # Reshard stream (usage: make reshard SHARDS=4)
make clean          # Clean build artifacts and data
make test           # Test compilation and Docker config
```





## Key Learnings





**Use Manual Mode :**
- Running in Kubernetes with fixed deployments
- Need precise resource allocation per shard
- Want to avoid rebalancing disruptions
- Prefer explicit control over automatic behavior

### Customizing VMware KCL
- Fork required for manual shard assignment feature
- Replace directive in go.mod enables transparent usage
- Custom logic validates shard assignment before lease acquisition
- Maintains compatibility with standard KCL features


## References


- **VMware KCL Customized Fork**: [github.com/ns-nagaaravindb/vmware-go-kcl](https://github.com/ns-nagaaravindb/vmware-go-kcl)
- **PR** [https://github.com/ns-nagaaravindb/vmware-go-kcl/pull/1](https://github.com/ns-nagaaravindb/vmware-go-kcl/pull/1)
- **customized KCL library Release** [https://github.com/ns-nagaaravindb/vmware-go-kcl/releases/tag/v1.5.1](https://github.com/ns-nagaaravindb/vmware-go-kcl/releases/tag/v1.5.1)



### Customized KCL Library

This POC uses a **forked version** of VMware's KCL library with manual shard assignment support:

- **Original**: `github.com/vmware/vmware-go-kcl@v1.5.1`
- **Customized**: `github.com/ns-nagaaravindb/vmware-go-kcl@v1.5.1`

**Key Enhancements:**
1. `EnableManualShardMapping` flag to enable/disable manual assignment
2. `WithManualShardMapping(map[string]string)` method for explicit shard-to-worker mapping
3. Shard assignment validation before lease acquisition
4. Worker skips shards not assigned to it

**Implementation:**
```go
kclConfig.EnableManualShardMapping = true
kclConfig.WithManualShardMapping(map[string]string{
    "shardId-000000000000": "worker-1",
    "shardId-000000000001": "worker-2",
    "shardId-000000000002": "worker-3",
    "shardId-000000000003": "worker-3",  // worker-3 handles 2 shards
})
```


## Quick Start

### 1. Start LocalStack

```bash
# Start LocalStack with Kinesis and DynamoDB
make start
# OR
docker-compose up -d

# Verify services are running
docker-compose logs localstack | grep -i "kinesis\|dynamodb"
```

You should see output indicating that the `test-stream` with 2 shards was created successfully.

### 2. Install Go Dependencies

```bash
go mod download
go mod tidy
```

**Note**: This project uses a customized fork of VMware KCL library (`github.com/ns-nagaaravindb/vmware-go-kcl`) with manual shard assignment support. The `go.mod` includes a replace directive:

```go
replace github.com/vmware/vmware-go-kcl => github.com/ns-nagaaravindb/vmware-go-kcl v1.5.1
```

### 3. Build the Applications

```bash
make build
```

This creates binaries in the `bin/` directory.

### 4. Run the Producer

In one terminal, start the producer to continuously send data to Kinesis:

```bash
make produce
```

The producer will generate random events and distribute them across all shards using partition keys.

### 5. Run the Consumer

In another terminal, start the consumer:

**Option A: Single Consumer (KCL Mode - Automatic Rebalancing)**
```bash
make consumer
```


## Configuration

### Configuration Files

The project uses YAML configuration files for different deployment scenarios:

- `config.yaml` - Default configuration (KCL mode or manual with all shards)


### Configuration Structure

```yaml
aws:
  region: us-east-1
  endpoint: http://localhost:4566  # LocalStack endpoint
  access_key: test
  secret_key: test

kinesis:
  stream_name: test-stream

producer:
  batch_size: 10              # Messages per batch
  batch_delay_ms: 1000        # Delay between batches (ms)
  total_messages: 0           # 0 for infinite

consumer:
  assignment_mode: kcl       
  application_name: kds-rebalance-consumer
  worker_id: worker-1
  max_records: 10
  call_process_records_even_for_empty_list: false
  poll_interval_ms: 1000      # Only used in "manual" mode
```


### Kubernetes Deployment Example

When deploying to Kubernetes, you can use environment variables or ConfigMaps to configure shard mapping:

```go
package main

import (
    "os"
    "strings"
    "github.com/vmware/vmware-go-kcl/clientlibrary/config"
    wk "github.com/vmware/vmware-go-kcl/clientlibrary/worker"
)

func parseShardMapping(envVar string) map[string]string {
    mapping := make(map[string]string)
    // Format: "shardId-000000000000:worker-1,shardId-000000000001:worker-2"
    pairs := strings.Split(envVar, ",")
    for _, pair := range pairs {
        parts := strings.Split(pair, ":")
        if len(parts) == 2 {
            mapping[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
        }
    }
    return mapping
}

func main() {
    // Get worker ID from pod name (e.g., worker-1, worker-2)
    workerID := os.Getenv("POD_NAME")
    if workerID == "" {
        workerID = "worker-1"
    }

    // Get shard mapping from environment variable or ConfigMap
    shardMappingEnv := os.Getenv("SHARD_MAPPING")
    shardMapping := parseShardMapping(shardMappingEnv)

    kclConfig := config.NewKinesisClientLibConfig(
        os.Getenv("APP_NAME"),
        os.Getenv("STREAM_NAME"),
        os.Getenv("AWS_REGION"),
        workerID,
    ).
        WithManualShardMapping(shardMapping).
        WithInitialPositionInStream(config.LATEST).
        WithLogger(logger.GetDefaultLogger())

    factory := &MyRecordProcessorFactory{}
    worker := wk.NewWorker(factory, kclConfig)
    
    if err := worker.Start(); err != nil {
        panic(err)
    }
    defer worker.Shutdown()

    // ... handle shutdown
}
```

**Kubernetes ConfigMap example:**

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: kcl-shard-mapping
data:
  shard-mapping: |
    shardId-000000000000:worker-1
    shardId-000000000001:worker-2
    shardId-000000000002:worker-1
    shardId-000000000003:worker-2
```

**Kubernetes Deployment example:**

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kcl-worker
spec:
  replicas: 2
  template:
    spec:
      containers:
      - name: worker
        image: my-kcl-app:latest
        env:
        - name: POD_NAME
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        - name: APP_NAME
          value: "my-app"
        - name: STREAM_NAME
          value: "my-stream"
        - name: AWS_REGION
          value: "us-east-1"
        - name: SHARD_MAPPING
          valueFrom:
            configMapKeyRef:
              name: kcl-shard-mapping
              key: shard-mapping
```


### Stream Resharding

The POC supports dynamic shard addition without automatic rebalancing:

```bash
# Add shards to the stream (e.g., 2 → 4 shards)
make reshard SHARDS=4

# Or use the script directly
./scripts/reshard-stream.sh 4

# Verify new shard count
docker exec localstack-kinesis awslocal kinesis describe-stream \
  --stream-name test-stream --query 'StreamDescription.Shards[].ShardId'
```




