// Package cloudwatch menyediakan implementasi adapter untuk AWS CloudWatch
package cloudwatch

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"go.opentelemetry.io/otel/sdk/resource"
)

// CloudWatchExporter adalah exporter untuk AWS CloudWatch Logs
type CloudWatchExporter struct {
	client        *cloudwatchlogs.CloudWatchLogs
	logGroup      string
	logStream     string
	sequenceToken *string
	logEvents     []*cloudwatchlogs.InputLogEvent
	maxBatchSize  int
	maxBatchWait  time.Duration
	mu            sync.Mutex
	done          chan struct{}
	wg            sync.WaitGroup
}

// CloudWatchConfig berisi konfigurasi untuk CloudWatch exporter
type CloudWatchConfig struct {
	// Region adalah AWS region
	Region string
	// AccessKey adalah AWS access key
	AccessKey string
	// SecretKey adalah AWS secret key
	SecretKey string
	// LogGroup adalah CloudWatch log group
	LogGroup string
	// LogStream adalah CloudWatch log stream
	LogStream string
	// RetentionDays adalah jumlah hari untuk menyimpan log (0 = tidak ada batas)
	RetentionDays int32
	// MaxBatchSize adalah jumlah maksimum log events yang akan dikirim dalam satu batch
	MaxBatchSize int
	// MaxBatchWait adalah durasi maksimum untuk menunggu sebelum mengirim batch
	MaxBatchWait time.Duration
	// Environment untuk metadata log
	Environment string
}

// DefaultCloudWatchConfig mengembalikan konfigurasi default untuk CloudWatch exporter
func DefaultCloudWatchConfig() CloudWatchConfig {
	return CloudWatchConfig{
		Region:        "us-west-2",
		LogGroup:      "/email-service/logs",
		LogStream:     fmt.Sprintf("log-stream-%d", time.Now().Unix()),
		RetentionDays: 30,
		MaxBatchSize:  10000,
		MaxBatchWait:  5 * time.Second,
		Environment:   "development",
	}
}

// NewCloudWatchExporter membuat instance baru dari CloudWatch exporter
func NewCloudWatchExporter(cfg CloudWatchConfig) (*CloudWatchExporter, error) {
	// Create AWS config
	awsConfig := aws.Config{
		Region: aws.String(cfg.Region),
	}

	if cfg.AccessKey != "" && cfg.SecretKey != "" {
		awsConfig.Credentials = credentials.NewStaticCredentials(
			cfg.AccessKey,
			cfg.SecretKey,
			"",
		)
	}

	// Create AWS session
	sess, err := session.NewSession(&awsConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS session: %w", err)
	}

	// Create CloudWatch Logs client
	client := cloudwatchlogs.New(sess)

	// Setup log group dan log stream
	if err := setupLogGroupAndStream(client, cfg); err != nil {
		return nil, err
	}

	// Create exporter
	exporter := &CloudWatchExporter{
		client:        client,
		logGroup:      cfg.LogGroup,
		logStream:     cfg.LogStream,
		sequenceToken: nil,
		logEvents:     make([]*cloudwatchlogs.InputLogEvent, 0, cfg.MaxBatchSize),
		maxBatchSize:  cfg.MaxBatchSize,
		maxBatchWait:  cfg.MaxBatchWait,
		done:          make(chan struct{}),
	}

	// Start background worker untuk mengirim logs
	exporter.startWorker()

	return exporter, nil
}

// setupLogGroupAndStream membuat log group dan log stream jika belum ada
func setupLogGroupAndStream(client *cloudwatchlogs.CloudWatchLogs, cfg CloudWatchConfig) error {
	// Check if log group exists
	_, err := client.DescribeLogGroups(&cloudwatchlogs.DescribeLogGroupsInput{
		LogGroupNamePrefix: aws.String(cfg.LogGroup),
	})

	if err != nil {
		// Create log group if it doesn't exist
		_, err = client.CreateLogGroup(&cloudwatchlogs.CreateLogGroupInput{
			LogGroupName: aws.String(cfg.LogGroup),
		})
		if err != nil {
			return fmt.Errorf("failed to create log group: %w", err)
		}

		// Set retention policy
		if cfg.RetentionDays > 0 {
			_, err = client.PutRetentionPolicy(&cloudwatchlogs.PutRetentionPolicyInput{
				LogGroupName:    aws.String(cfg.LogGroup),
				RetentionInDays: aws.Int64(int64(cfg.RetentionDays)),
			})
			if err != nil {
				return fmt.Errorf("failed to set retention policy: %w", err)
			}
		}
	}

	// Create log stream
	_, err = client.CreateLogStream(&cloudwatchlogs.CreateLogStreamInput{
		LogGroupName:  aws.String(cfg.LogGroup),
		LogStreamName: aws.String(cfg.LogStream),
	})
	if err != nil {
		return fmt.Errorf("failed to create log stream: %w", err)
	}

	return nil
}

// startWorker memulai goroutine untuk mengirim batch logs ke CloudWatch
func (e *CloudWatchExporter) startWorker() {
	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		ticker := time.NewTicker(e.maxBatchWait)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				e.flush()
			case <-e.done:
				e.flush()
				return
			}
		}
	}()
}

// LogRecord represents a log record to be exported to CloudWatch
type LogRecord struct {
	Timestamp  time.Time
	Level      string
	Message    string
	Attributes map[string]interface{}
}

// Export exports log records to CloudWatch
func (e *CloudWatchExporter) Export(ctx context.Context, records []LogRecord) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	for _, record := range records {
		// Convert to CloudWatch log event
		event, err := convertToCloudWatchEvent(record)
		if err != nil {
			// Just log error and continue
			fmt.Printf("Failed to convert log record: %v\n", err)
			continue
		}

		e.logEvents = append(e.logEvents, event)

		// Flush jika sudah mencapai batch size
		if len(e.logEvents) >= e.maxBatchSize {
			e.flush()
		}
	}

	return nil
}

// convertToCloudWatchEvent mengonversi log record ke CloudWatch log event
func convertToCloudWatchEvent(record LogRecord) (*cloudwatchlogs.InputLogEvent, error) {
	// Extract timestamp
	timestamp := record.Timestamp.UnixNano() / int64(time.Millisecond)

	// Prepare attributes
	attrs := make(map[string]interface{})
	for k, v := range record.Attributes {
		attrs[k] = v
	}

	// Add standard fields
	attrs["level"] = record.Level
	attrs["timestamp"] = record.Timestamp.Format(time.RFC3339Nano)
	attrs["message"] = record.Message

	// Convert to JSON
	jsonData, err := json.Marshal(attrs)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal log event: %w", err)
	}

	return &cloudwatchlogs.InputLogEvent{
		Timestamp: aws.Int64(timestamp),
		Message:   aws.String(string(jsonData)),
	}, nil
}

// flush mengirim semua log events ke CloudWatch
func (e *CloudWatchExporter) flush() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if len(e.logEvents) == 0 {
		return
	}

	// Sort log events by timestamp
	// Note: AWS requires log events to be in chronological order
	sortLogEvents(e.logEvents)

	// Create input
	input := &cloudwatchlogs.PutLogEventsInput{
		LogGroupName:  aws.String(e.logGroup),
		LogStreamName: aws.String(e.logStream),
		LogEvents:     e.logEvents,
	}

	// Add sequence token if we have one
	if e.sequenceToken != nil {
		input.SequenceToken = e.sequenceToken
	}

	// Send to CloudWatch
	resp, err := e.client.PutLogEvents(input)
	if err != nil {
		fmt.Printf("Failed to send logs to CloudWatch: %v\n", err)
		return
	}

	// Save next sequence token
	e.sequenceToken = resp.NextSequenceToken

	// Clear log events
	e.logEvents = make([]*cloudwatchlogs.InputLogEvent, 0, e.maxBatchSize)
}

// sortLogEvents mengurutkan log events berdasarkan timestamp
func sortLogEvents(events []*cloudwatchlogs.InputLogEvent) {
	// For simplicity, we'll use a simple bubble sort here
	// In a real application, use a more efficient sort algorithm
	n := len(events)
	for i := 0; i < n; i++ {
		for j := 0; j < n-i-1; j++ {
			if *events[j].Timestamp > *events[j+1].Timestamp {
				events[j], events[j+1] = events[j+1], events[j]
			}
		}
	}
}

// Shutdown stops the exporter and flushes any remaining logs
func (e *CloudWatchExporter) Shutdown(ctx context.Context) error {
	close(e.done)

	// Wait for worker to finish
	done := make(chan struct{})
	go func() {
		e.wg.Wait()
		close(done)
	}()

	// Wait for shutdown or context cancellation
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// ForceFlush forces a flush of any buffered logs
func (e *CloudWatchExporter) ForceFlush(ctx context.Context) error {
	e.flush()
	return nil
}

// CloudWatchLogger adalah wrapper untuk logger yang menggunakan CloudWatch
type CloudWatchLogger struct {
	exporter *CloudWatchExporter
	resource *resource.Resource
}

// NewCloudWatchLogger membuat instance baru CloudWatchLogger
func NewCloudWatchLogger(cfg CloudWatchConfig, res *resource.Resource) (*CloudWatchLogger, error) {
	exporter, err := NewCloudWatchExporter(cfg)
	if err != nil {
		return nil, err
	}

	return &CloudWatchLogger{
		exporter: exporter,
		resource: res,
	}, nil
}

// Log sends a log entry to CloudWatch
func (l *CloudWatchLogger) Log(level, msg string, attrs map[string]interface{}) {
	// Create log record
	record := LogRecord{
		Timestamp:  time.Now(),
		Level:      level,
		Message:    msg,
		Attributes: attrs,
	}

	// Add resource attributes
	if l.resource != nil {
		for _, attr := range l.resource.Attributes() {
			record.Attributes[string(attr.Key)] = attr.Value.AsInterface()
		}
	}

	// Export log record
	if err := l.exporter.Export(context.Background(), []LogRecord{record}); err != nil {
		fmt.Printf("Failed to export log: %v\n", err)
	}
}

// Shutdown shuts down the logger
func (l *CloudWatchLogger) Shutdown(ctx context.Context) error {
	return l.exporter.Shutdown(ctx)
}

// ForceFlush forces a flush of any buffered logs
func (l *CloudWatchLogger) ForceFlush(ctx context.Context) error {
	return l.exporter.ForceFlush(ctx)
}
