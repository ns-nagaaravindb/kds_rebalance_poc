# export env 

```bash
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test
export AWS_DEFAULT_REGION=us-east-1
export ENDPOINT_URL=http://localhost:4566
```


# AWS Kinesis LocalStack CLI Commands

Below is a quick reference table for managing and testing AWS Kinesis Data Streams (KDS) in a LocalStack environment.

| Action             | Command                                                                                                                                    |
| ------------------ | ------------------------------------------------------------------------------------------------------------------------------------------ |
| List streams       | `aws kinesis list-streams --endpoint-url $ENDPOINT_URL`                                                                                    |
| List shards        | `aws kinesis describe-stream --stream-name test-stream --endpoint-url $ENDPOINT_URL`                                                       |
| Get shard iterator | `aws kinesis get-shard-iterator --stream-name test-stream --shard-id <id> --shard-iterator-type TRIM_HORIZON --endpoint-url $ENDPOINT_URL` |
| Read data          | `aws kinesis get-records --shard-iterator <iterator> --endpoint-url $ENDPOINT_URL`                                                         |
| Write data         | `aws kinesis put-record --stream-name test-stream --partition-key key1 --data '{"msg":"hi"}' --endpoint-url $ENDPOINT_URL`                 |

---

### 🧩 Notes
- Replace `<id>` with the actual Shard ID (from the describe-stream command).
- Replace `<iterator>` with the actual Shard Iterator (from the get-shard-iterator command).
- `$ENDPOINT_URL` typically equals `http://localhost:4566` for LocalStack.



docker exec localstack-kinesis awslocal dynamodb list-tables --region us-east-1
{
    "TableNames": [
        "kds-rebalance-consumer"
    ]
}


### Manually Create/Manage Streams

```bash
# Access LocalStack container
docker exec -it localstack-kinesis bash

# List streams
awslocal kinesis list-streams

# Describe a stream
awslocal kinesis describe-stream --stream-name test-stream

# Get shard details
awslocal kinesis list-shards --stream-name test-stream

# Delete a stream
awslocal kinesis delete-stream --stream-name test-stream

# Create a new stream with 4 shards
awslocal kinesis create-stream --stream-name my-stream --shard-count 4
```

### DynamoDB Lease Table Management

When using KCL mode, check the lease table:

```bash
# List DynamoDB tables
docker exec localstack-kinesis awslocal dynamodb list-tables

# Scan lease table
docker exec localstack-kinesis awslocal dynamodb scan \
  --table-name kds-rebalance-consumer

# Delete lease table (to reset checkpoint)
docker exec localstack-kinesis awslocal dynamodb delete-table \
  --table-name kds-rebalance-consumer
```



