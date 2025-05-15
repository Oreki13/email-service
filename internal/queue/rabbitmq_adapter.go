// filepath: /Users/fandy/project/email-service/internal/queue/rabbitmq_adapter.go
package queue

import (
	"context"
	"email-service/internal/domain"
	"email-service/pkg/telemetry"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	// DefaultExchange adalah nama default exchange
	DefaultExchange = ""
	// EmailExchange adalah nama exchange untuk email
	EmailExchange = "email_exchange"
	// EmailQueue adalah nama queue untuk email
	EmailQueue = "email_queue"
	// HighPriorityQueue adalah nama queue untuk email prioritas tinggi
	HighPriorityQueue = "email_high_priority_queue"
	// NormalPriorityQueue adalah nama queue untuk email prioritas normal
	NormalPriorityQueue = "email_normal_priority_queue"
	// LowPriorityQueue adalah nama queue untuk email prioritas rendah
	LowPriorityQueue = "email_low_priority_queue"
	// DeadLetterExchange adalah nama exchange untuk pesan yang gagal diproses
	DeadLetterExchange = "email_dlx"
	// DeadLetterQueue adalah nama queue untuk pesan yang gagal diproses
	DeadLetterQueue = "email_dlq"
)

// EmailMessage adalah struktur pesan yang dikirim ke queue
type EmailMessage struct {
	ID        string    `json:"id"`
	Priority  string    `json:"priority"`
	Timestamp time.Time `json:"timestamp"`
	TraceID   string    `json:"trace_id,omitempty"`
}

// RabbitMQConfig adalah konfigurasi untuk RabbitMQ
type RabbitMQConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	VHost    string
}

// RabbitMQAdapter adalah adapter untuk RabbitMQ
type RabbitMQAdapter struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	config  RabbitMQConfig
	logger  telemetry.Logger
}

// NewRabbitMQAdapter membuat instance baru RabbitMQAdapter
func NewRabbitMQAdapter(config RabbitMQConfig, logger telemetry.Logger) (*RabbitMQAdapter, error) {
	adapter := &RabbitMQAdapter{
		config: config,
		logger: logger,
	}

	// Connect ke RabbitMQ
	if err := adapter.connect(); err != nil {
		return nil, err
	}

	// Setup exchange dan queue
	if err := adapter.setup(); err != nil {
		return nil, err
	}

	return adapter, nil
}

// connect membuat koneksi ke RabbitMQ
func (a *RabbitMQAdapter) connect() error {
	// Buat URL koneksi RabbitMQ
	url := fmt.Sprintf("amqp://%s:%s@%s:%d/%s",
		a.config.Username,
		a.config.Password,
		a.config.Host,
		a.config.Port,
		a.config.VHost,
	)

	// Connect ke RabbitMQ
	conn, err := amqp.Dial(url)
	if err != nil {
		return domain.ExternalServiceError("Failed to connect to RabbitMQ", err)
	}
	a.conn = conn

	// Buat channel
	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return domain.ExternalServiceError("Failed to open a channel", err)
	}
	a.channel = channel

	return nil
}

// setup membuat exchange dan queue
func (a *RabbitMQAdapter) setup() error {
	// Deklarasi Dead Letter Exchange
	err := a.channel.ExchangeDeclare(
		DeadLetterExchange, // name
		"direct",           // type
		true,               // durable
		false,              // auto-deleted
		false,              // internal
		false,              // no-wait
		nil,                // arguments
	)
	if err != nil {
		return domain.ExternalServiceError("Failed to declare DLX exchange", err)
	}

	// Deklarasi Dead Letter Queue
	_, err = a.channel.QueueDeclare(
		DeadLetterQueue, // name
		true,            // durable
		false,           // delete when unused
		false,           // exclusive
		false,           // no-wait
		nil,             // arguments
	)
	if err != nil {
		return domain.ExternalServiceError("Failed to declare DLQ", err)
	}

	// Bind Dead Letter Queue ke Dead Letter Exchange
	err = a.channel.QueueBind(
		DeadLetterQueue,    // queue name
		"#",                // routing key (all messages)
		DeadLetterExchange, // exchange
		false,              // no-wait
		nil,                // arguments
	)
	if err != nil {
		return domain.ExternalServiceError("Failed to bind DLQ to DLX", err)
	}

	// Deklarasi Exchange untuk email
	err = a.channel.ExchangeDeclare(
		EmailExchange, // name
		"direct",      // type
		true,          // durable
		false,         // auto-deleted
		false,         // internal
		false,         // no-wait
		nil,           // arguments
	)
	if err != nil {
		return domain.ExternalServiceError("Failed to declare email exchange", err)
	}

	// Setup dead letter config
	dlxArgs := amqp.Table{
		"x-dead-letter-exchange": DeadLetterExchange,
	}

	// Deklarasi Queue untuk email prioritas tinggi
	_, err = a.channel.QueueDeclare(
		HighPriorityQueue, // name
		true,              // durable
		false,             // delete when unused
		false,             // exclusive
		false,             // no-wait
		dlxArgs,           // arguments
	)
	if err != nil {
		return domain.ExternalServiceError("Failed to declare high priority queue", err)
	}

	// Bind Queue prioritas tinggi ke Exchange
	err = a.channel.QueueBind(
		HighPriorityQueue,           // queue name
		string(domain.PriorityHigh), // routing key
		EmailExchange,               // exchange
		false,                       // no-wait
		nil,                         // arguments
	)
	if err != nil {
		return domain.ExternalServiceError("Failed to bind high priority queue", err)
	}

	// Deklarasi Queue untuk email prioritas normal
	_, err = a.channel.QueueDeclare(
		NormalPriorityQueue, // name
		true,                // durable
		false,               // delete when unused
		false,               // exclusive
		false,               // no-wait
		dlxArgs,             // arguments
	)
	if err != nil {
		return domain.ExternalServiceError("Failed to declare normal priority queue", err)
	}

	// Bind Queue prioritas normal ke Exchange
	err = a.channel.QueueBind(
		NormalPriorityQueue,           // queue name
		string(domain.PriorityNormal), // routing key
		EmailExchange,                 // exchange
		false,                         // no-wait
		nil,                           // arguments
	)
	if err != nil {
		return domain.ExternalServiceError("Failed to bind normal priority queue", err)
	}

	// Deklarasi Queue untuk email prioritas rendah
	_, err = a.channel.QueueDeclare(
		LowPriorityQueue, // name
		true,             // durable
		false,            // delete when unused
		false,            // exclusive
		false,            // no-wait
		dlxArgs,          // arguments
	)
	if err != nil {
		return domain.ExternalServiceError("Failed to declare low priority queue", err)
	}

	// Bind Queue prioritas rendah ke Exchange
	err = a.channel.QueueBind(
		LowPriorityQueue,           // queue name
		string(domain.PriorityLow), // routing key
		EmailExchange,              // exchange
		false,                      // no-wait
		nil,                        // arguments
	)
	if err != nil {
		return domain.ExternalServiceError("Failed to bind low priority queue", err)
	}

	a.logger.Info(context.Background(), "RabbitMQ setup completed", telemetry.Fields{
		"exchange": EmailExchange,
		"queues": []string{
			HighPriorityQueue,
			NormalPriorityQueue,
			LowPriorityQueue,
		},
	})

	return nil
}

// PublishEmail mengirim ID email ke queue untuk diproses
func (a *RabbitMQAdapter) PublishEmail(ctx context.Context, emailID string, priority domain.Priority) error {
	// Gunakan priority sebagai routing key
	if priority == "" {
		priority = domain.PriorityNormal
	}

	// Siapkan pesan
	message := EmailMessage{
		ID:        emailID,
		Priority:  string(priority),
		Timestamp: time.Now(),
	}

	// Extract trace ID dari context jika ada
	if traceID, ok := ctx.Value("traceID").(string); ok {
		message.TraceID = traceID
	}

	// Marshal message ke JSON
	messageJSON, err := json.Marshal(message)
	if err != nil {
		return domain.InternalError("Failed to marshal email message", err)
	}

	// Publish pesan ke queue
	err = a.channel.Publish(
		EmailExchange,    // exchange
		string(priority), // routing key
		false,            // mandatory
		false,            // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         messageJSON,
			DeliveryMode: amqp.Persistent, // Make message persistent
			MessageId:    uuid.New().String(),
			Timestamp:    time.Now(),
			Headers: amqp.Table{
				"x-trace-id": message.TraceID,
			},
		},
	)

	if err != nil {
		return domain.ExternalServiceError("Failed to publish email to queue", err)
	}

	a.logger.Info(ctx, "Email queued for delivery", telemetry.Fields{
		"email_id": emailID,
		"priority": priority,
		"trace_id": message.TraceID,
	})

	return nil
}

// Close menutup koneksi dan channel
func (a *RabbitMQAdapter) Close() error {
	if a.channel != nil {
		a.channel.Close()
	}
	if a.conn != nil {
		a.conn.Close()
	}
	return nil
}
