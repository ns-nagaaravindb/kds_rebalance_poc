package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/sirupsen/logrus"
	"github.com/vmware/vmware-go-kcl/clientlibrary/config"
	"github.com/vmware/vmware-go-kcl/clientlibrary/interfaces"
	"github.com/vmware/vmware-go-kcl/clientlibrary/worker"
	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	AWS struct {
		Region    string `yaml:"region"`
		Endpoint  string `yaml:"endpoint"`
		AccessKey string `yaml:"access_key"`
		SecretKey string `yaml:"secret_key"`
	} `yaml:"aws"`
	Kinesis struct {
		StreamName string `yaml:"stream_name"`
	} `yaml:"kinesis"`
	Consumer struct {
		AssignmentMode                           string   `yaml:"assignment_mode"` // "kcl" or "manual"
		ApplicationName                          string   `yaml:"application_name"`
		WorkerID                                 string   `yaml:"worker_id"`
		MaxRecords                               int      `yaml:"max_records"`
		CallProcessRecordsEvenForEmptyRecordList bool     `yaml:"call_process_records_even_for_empty_list"`
		AssignedShards                           []string `yaml:"assigned_shards"`
		PollIntervalMs                           int      `yaml:"poll_interval_ms"`
	} `yaml:"consumer"`
}

// Event represents a sample data event
type Event struct {
	EventID   string                 `json:"event_id"`
	UserID    string                 `json:"user_id"`
	Timestamp time.Time              `json:"timestamp"`
	Action    string                 `json:"action"`
	Value     float64                `json:"value"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// RecordProcessor implements the KCL RecordProcessor interface
type RecordProcessor struct {
	shardID     string
	recordCount int
	startTime   time.Time
}

// Initialize is called once when the processor starts processing a shard
func (rp *RecordProcessor) Initialize(input *interfaces.InitializationInput) {
	rp.shardID = input.ShardId
	rp.recordCount = 0
	rp.startTime = time.Now()
	log.Printf("[%s] Initializing record processor", rp.shardID)
}

// ProcessRecords is called to process a batch of records from the shard
func (rp *RecordProcessor) ProcessRecords(input *interfaces.ProcessRecordsInput) {
	// Process each record
	for _, record := range input.Records {
		var event Event
		if err := json.Unmarshal(record.Data, &event); err != nil {
			log.Printf("[%s] Failed to unmarshal record: %v", rp.shardID, err)
			continue
		}

		rp.recordCount++
		log.Printf("[%s] Record #%d | EventID: %s | UserID: %s | Action: %s | Value: %.2f | SeqNum: %s",
			rp.shardID, rp.recordCount, event.EventID, event.UserID, event.Action, event.Value, *record.SequenceNumber)
	}

	// Checkpoint after processing records
	if len(input.Records) > 0 {
		lastRecord := input.Records[len(input.Records)-1]
		if err := input.Checkpointer.Checkpoint(lastRecord.SequenceNumber); err != nil {
			log.Printf("[%s] Failed to checkpoint: %v", rp.shardID, err)
		}
	}
}

// Shutdown is called when the processor is shutting down
func (rp *RecordProcessor) Shutdown(input *interfaces.ShutdownInput) {
	elapsed := time.Since(rp.startTime).Seconds()
	log.Printf("[%s] Shutting down. Reason: %v. Processed %d records in %.2f seconds",
		rp.shardID, input.ShutdownReason, rp.recordCount, elapsed)

	// Checkpoint on graceful shutdown
	if input.ShutdownReason == interfaces.TERMINATE {
		if err := input.Checkpointer.Checkpoint(nil); err != nil {
			log.Printf("[%s] Failed to checkpoint on shutdown: %v", rp.shardID, err)
		}
	}
}

// RecordProcessorFactory creates new RecordProcessor instances
type RecordProcessorFactory struct{}

// CreateProcessor creates a new RecordProcessor for a shard
func (f *RecordProcessorFactory) CreateProcessor() interfaces.IRecordProcessor {
	return &RecordProcessor{}
}

// ManualShardProcessor processes records from a specific shard
type ManualShardProcessor struct {
	shardID       string
	streamName    string
	kinesisClient *kinesis.Kinesis
	maxRecords    int64
	pollInterval  time.Duration
	recordCount   int
	startTime     time.Time
}

// ProcessShard processes records from the assigned shard in a loop
func (msp *ManualShardProcessor) ProcessShard(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	msp.startTime = time.Now()
	log.Printf("[%s] [Goroutine] Starting manual processor for shard", msp.shardID)

	// Get shard iterator
	iteratorOutput, err := msp.kinesisClient.GetShardIterator(&kinesis.GetShardIteratorInput{
		StreamName:        aws.String(msp.streamName),
		ShardId:           aws.String(msp.shardID),
		ShardIteratorType: aws.String("TRIM_HORIZON"), // Start from beginning
	})
	if err != nil {
		log.Printf("[%s] Failed to get shard iterator: %v", msp.shardID, err)
		return
	}

	shardIterator := iteratorOutput.ShardIterator

	for {
		select {
		case <-ctx.Done():
			elapsed := time.Since(msp.startTime).Seconds()
			log.Printf("[%s] [Goroutine] Stopping. Processed %d records in %.2f seconds",
				msp.shardID, msp.recordCount, elapsed)
			return
		default:
			if shardIterator == nil {
				log.Printf("[%s] Shard iterator is nil, shard might be closed", msp.shardID)
				return
			}

			// Get records
			getRecordsOutput, err := msp.kinesisClient.GetRecords(&kinesis.GetRecordsInput{
				ShardIterator: shardIterator,
				Limit:         aws.Int64(msp.maxRecords),
			})
			if err != nil {
				log.Printf("[%s] Failed to get records: %v", msp.shardID, err)
				time.Sleep(msp.pollInterval)
				continue
			}

			// Process records
			for _, record := range getRecordsOutput.Records {
				var event Event
				if err := json.Unmarshal(record.Data, &event); err != nil {
					log.Printf("[%s] Failed to unmarshal record: %v", msp.shardID, err)
					continue
				}

				msp.recordCount++
				log.Printf("[%s] [Goroutine] Record #%d | EventID: %s | UserID: %s | Action: %s | Value: %.2f | SeqNum: %s",
					msp.shardID, msp.recordCount, event.EventID, event.UserID, event.Action, event.Value, *record.SequenceNumber)
			}

			// Update iterator for next fetch
			shardIterator = getRecordsOutput.NextShardIterator

			// Wait before next poll
			time.Sleep(msp.pollInterval)
		}
	}
}

func loadConfig() (*Config, error) {
	// Check for custom config file path from environment variable
	configFile := os.Getenv("CONFIG_FILE")
	if configFile == "" {
		configFile = "../config.yaml"
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configFile, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	log.Printf("Loaded configuration from: %s", configFile)
	return &cfg, nil
}

func runManualMode(cfg *Config) error {
	log.Println("Running in MANUAL assignment mode")
	log.Printf("Worker ID: %s, Assigned Shards: %v", cfg.Consumer.WorkerID, cfg.Consumer.AssignedShards)

	// Create AWS session
	awsConfig := &aws.Config{
		Region:      aws.String(cfg.AWS.Region),
		Endpoint:    aws.String(cfg.AWS.Endpoint),
		Credentials: credentials.NewStaticCredentials(cfg.AWS.AccessKey, cfg.AWS.SecretKey, ""),
	}
	sess, err := session.NewSession(awsConfig)
	if err != nil {
		return fmt.Errorf("failed to create AWS session: %w", err)
	}

	kinesisClient := kinesis.New(sess)

	// Validate assigned shards exist
	describeOutput, err := kinesisClient.DescribeStream(&kinesis.DescribeStreamInput{
		StreamName: aws.String(cfg.Kinesis.StreamName),
	})
	if err != nil {
		return fmt.Errorf("failed to describe stream: %w", err)
	}

	availableShards := make(map[string]bool)
	for _, shard := range describeOutput.StreamDescription.Shards {
		availableShards[*shard.ShardId] = true
	}

	// Validate configuration
	for _, shardID := range cfg.Consumer.AssignedShards {
		if !availableShards[shardID] {
			return fmt.Errorf("assigned shard %s does not exist in stream", shardID)
		}
	}

	log.Printf("Validated %d assigned shards against stream", len(cfg.Consumer.AssignedShards))

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Received shutdown signal...")
		cancel()
	}()

	// Start a goroutine for each assigned shard
	var wg sync.WaitGroup
	pollInterval := time.Duration(cfg.Consumer.PollIntervalMs) * time.Millisecond

	for _, shardID := range cfg.Consumer.AssignedShards {
		wg.Add(1)
		processor := &ManualShardProcessor{
			shardID:       shardID,
			streamName:    cfg.Kinesis.StreamName,
			kinesisClient: kinesisClient,
			maxRecords:    int64(cfg.Consumer.MaxRecords),
			pollInterval:  pollInterval,
		}
		go processor.ProcessShard(ctx, &wg)
	}

	log.Printf("Started %d goroutines (one per assigned shard)", len(cfg.Consumer.AssignedShards))
	log.Println("Consumer is running. Press Ctrl+C to stop.")

	// Wait for all goroutines to finish
	wg.Wait()
	log.Println("All shard processors stopped.")
	return nil
}

func runKCLMode(cfg *Config) error {
	log.Println("Running in KCL assignment mode (automatic rebalancing)")

	// Enable debug logging for KCL library
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	// Configure KCL
	kclConfig := config.NewKinesisClientLibConfig(
		cfg.Consumer.ApplicationName,
		cfg.Kinesis.StreamName,
		cfg.AWS.Region,
		cfg.Consumer.WorkerID,
	)

	// Set LocalStack endpoints
	kclConfig.KinesisEndpoint = cfg.AWS.Endpoint
	kclConfig.DynamoDBEndpoint = cfg.AWS.Endpoint

	// Set credentials for LocalStack
	kclConfig.KinesisCredentials = credentials.NewStaticCredentials(cfg.AWS.AccessKey, cfg.AWS.SecretKey, "")
	kclConfig.DynamoDBCredentials = credentials.NewStaticCredentials(cfg.AWS.AccessKey, cfg.AWS.SecretKey, "")

	// Set other configuration options
	kclConfig.InitialPositionInStream = config.TRIM_HORIZON // Read from beginning of stream
	kclConfig.MaxRecords = cfg.Consumer.MaxRecords
	kclConfig.CallProcessRecordsEvenForEmptyRecordList = cfg.Consumer.CallProcessRecordsEvenForEmptyRecordList
	kclConfig.EnableManualShardMapping = true
	kclConfig.WithManualShardMapping(map[string]string{
		"shardId-000000000001": "worker-2",
		"shardId-000000000002": "worker-1",
		"shardId-000000000003": "worker-3",
	})

	log.Printf("Application: %s, Worker ID: %s", cfg.Consumer.ApplicationName, cfg.Consumer.WorkerID)
	log.Printf("Configuration: MaxRecords=%d", cfg.Consumer.MaxRecords)

	// Create worker
	recordProcessorFactory := &RecordProcessorFactory{}
	kclWorker := worker.NewWorker(recordProcessorFactory, kclConfig)

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start the worker in a goroutine
	log.Println("Consumer is running. Press Ctrl+C to stop.")

	errChan := make(chan error, 1)
	go func() {
		if err := kclWorker.Start(); err != nil {
			errChan <- err
		}
	}()

	// Wait for either shutdown signal or error
	select {
	case <-sigChan:
		log.Println("Received shutdown signal...")
		kclWorker.Shutdown()
	case err := <-errChan:
		return fmt.Errorf("worker failed: %w", err)
	}

	return nil
}

func main() {
	log.Println("Starting Kinesis Consumer...")

	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Printf("Connected to Kinesis stream: %s", cfg.Kinesis.StreamName)

	// Run in the configured assignment mode
	var runErr error
	switch cfg.Consumer.AssignmentMode {
	case "manual":
		runErr = runManualMode(cfg)
	case "kcl":
		runErr = runKCLMode(cfg)
	default:
		log.Fatalf("Invalid assignment_mode: %s. Must be 'manual' or 'kcl'", cfg.Consumer.AssignmentMode)
	}

	if runErr != nil {
		log.Fatalf("Consumer failed: %v", runErr)
	}

	log.Println("Consumer stopped.")
}
