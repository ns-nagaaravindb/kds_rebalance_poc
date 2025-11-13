package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
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
	Producer struct {
		BatchSize     int `yaml:"batch_size"`
		BatchDelayMs  int `yaml:"batch_delay_ms"`
		TotalMessages int `yaml:"total_messages"`
	} `yaml:"producer"`
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

var actions = []string{"login", "purchase", "view", "click", "logout", "search", "add_to_cart", "checkout"}

func loadConfig() (*Config, error) {
	data, err := os.ReadFile("../config.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &cfg, nil
}

func generateEvent() *Event {
	return &Event{
		EventID:   fmt.Sprintf("evt_%d", time.Now().UnixNano()),
		UserID:    fmt.Sprintf("user_%d", rand.Intn(1000)),
		Timestamp: time.Now(),
		Action:    actions[rand.Intn(len(actions))],
		Value:     rand.Float64() * 1000,
		Metadata: map[string]interface{}{
			"source":  "web",
			"version": "1.0",
			"session": fmt.Sprintf("sess_%d", rand.Intn(100)),
		},
	}
}

func main() {
	log.Println("Starting Kinesis Producer...")

	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize AWS Config
	ctx := context.Background()
	awsCfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(cfg.AWS.Region),
		config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{
					URL:               cfg.AWS.Endpoint,
					HostnameImmutable: true,
				}, nil
			})),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AWS.AccessKey,
			cfg.AWS.SecretKey,
			"",
		)),
	)
	if err != nil {
		log.Fatalf("Failed to load AWS config: %v", err)
	}

	// Create Kinesis client
	client := kinesis.NewFromConfig(awsCfg)

	log.Printf("Connected to Kinesis stream: %s", cfg.Kinesis.StreamName)
	log.Printf("Configuration: BatchSize=%d, BatchDelay=%dms, TotalMessages=%d",
		cfg.Producer.BatchSize, cfg.Producer.BatchDelayMs, cfg.Producer.TotalMessages)

	messageCount := 0
	startTime := time.Now()

	for {
		// Check if we've reached the total message limit
		if cfg.Producer.TotalMessages > 0 && messageCount >= cfg.Producer.TotalMessages {
			log.Printf("Reached total message limit: %d messages", cfg.Producer.TotalMessages)
			break
		}

		// Send batch of messages
		for i := 0; i < cfg.Producer.BatchSize; i++ {
			event := generateEvent()
			data, err := json.Marshal(event)
			if err != nil {
				log.Printf("Failed to marshal event: %v", err)
				continue
			}

			// Use UserID as partition key for consistent shard assignment
			input := &kinesis.PutRecordInput{
				StreamName:   aws.String(cfg.Kinesis.StreamName),
				Data:         data,
				PartitionKey: aws.String(event.UserID),
			}

			output, err := client.PutRecord(ctx, input)
			if err != nil {
				log.Printf("Failed to put record: %v", err)
				continue
			}

			messageCount++
			log.Printf("[%d] Sent event %s | UserID: %s | Action: %s | ShardID: %s | SequenceNumber: %s",
				messageCount, event.EventID, event.UserID, event.Action, *output.ShardId, *output.SequenceNumber)

			// Break if we've reached the limit mid-batch
			if cfg.Producer.TotalMessages > 0 && messageCount >= cfg.Producer.TotalMessages {
				break
			}
		}

		// Calculate and display stats
		elapsed := time.Since(startTime).Seconds()
		rate := float64(messageCount) / elapsed
		log.Printf("Stats: Total=%d, Rate=%.2f msgs/sec, Elapsed=%.2fs", messageCount, rate, elapsed)

		// Wait before next batch
		if cfg.Producer.TotalMessages == 0 || messageCount < cfg.Producer.TotalMessages {
			time.Sleep(time.Duration(cfg.Producer.BatchDelayMs) * time.Millisecond)
		}
	}

	elapsed := time.Since(startTime).Seconds()
	log.Printf("Producer completed: %d messages in %.2f seconds (%.2f msgs/sec)",
		messageCount, elapsed, float64(messageCount)/elapsed)
}
