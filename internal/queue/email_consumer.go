package queue

import (
	"context"
	"email-service/internal/domain"
	"email-service/pkg/telemetry"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// EmailConsumer adalah consumer untuk memproses email dari queue
type EmailConsumer struct {
	conn            *amqp.Connection
	channel         *amqp.Channel
	config          RabbitMQConfig
	logger          telemetry.Logger
	emailService    domain.EmailService
	shutdownSignal  chan struct{}
	consumerWaitGrp sync.WaitGroup
	consumeTags     []string // untuk menyimpan consumer tags agar bisa dibatalkan
}

// NewEmailConsumer membuat instance baru EmailConsumer
func NewEmailConsumer(
	config RabbitMQConfig,
	logger telemetry.Logger,
	emailService domain.EmailService,
) (*EmailConsumer, error) {
	consumer := &EmailConsumer{
		config:         config,
		logger:         logger,
		emailService:   emailService,
		shutdownSignal: make(chan struct{}),
		consumeTags:    make([]string, 0),
	}

	// Connect ke RabbitMQ
	if err := consumer.connect(); err != nil {
		return nil, err
	}

	return consumer, nil
}

// connect membuat koneksi ke RabbitMQ
func (c *EmailConsumer) connect() error {
	// Buat URL koneksi RabbitMQ
	url := fmt.Sprintf("amqp://%s:%s@%s:%d/%s",
		c.config.Username,
		c.config.Password,
		c.config.Host,
		c.config.Port,
		c.config.VHost,
	)

	// Connect ke RabbitMQ
	conn, err := amqp.Dial(url)
	if err != nil {
		return domain.ExternalServiceError("Failed to connect to RabbitMQ", err)
	}
	c.conn = conn

	// Buat channel
	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return domain.ExternalServiceError("Failed to open a channel", err)
	}
	c.channel = channel

	// Set QoS agar tidak mengambil terlalu banyak message sekaligus
	if err := channel.Qos(
		5,     // prefetch count (jumlah pesan yang bisa diambil sekali)
		0,     // prefetch size (dalam bytes, 0 berarti tidak ada batasan)
		false, // global (false = per consumer)
	); err != nil {
		channel.Close()
		conn.Close()
		return domain.ExternalServiceError("Failed to set QoS", err)
	}

	return nil
}

// Start memulai consumer untuk memproses email dari queue
func (c *EmailConsumer) Start() error {
	// Lakukan reconnect otomatis jika koneksi terputus
	go c.monitorConnection()

	// Mulai consumer untuk setiap prioritas
	if err := c.consumeQueue(HighPriorityQueue); err != nil {
		return err
	}
	if err := c.consumeQueue(NormalPriorityQueue); err != nil {
		return err
	}
	if err := c.consumeQueue(LowPriorityQueue); err != nil {
		return err
	}

	c.logger.Info(context.Background(), "Email consumers started", telemetry.Fields{
		"queues": []string{
			HighPriorityQueue,
			NormalPriorityQueue,
			LowPriorityQueue,
		},
	})

	return nil
}

// monitorConnection memantau koneksi dan melakukan reconnect jika terputus
func (c *EmailConsumer) monitorConnection() {
	connClosed := make(chan *amqp.Error)
	c.conn.NotifyClose(connClosed)

	// Block hingga koneksi tertutup atau shutdown signal diterima
	select {
	case <-connClosed:
		c.logger.Warn(context.Background(), "RabbitMQ connection lost, attempting to reconnect", nil)

		// Tunggu beberapa saat sebelum reconnect
		time.Sleep(5 * time.Second)

		// Coba reconnect hingga berhasil atau shutdown
		for {
			select {
			case <-c.shutdownSignal:
				return
			default:
				if err := c.reconnect(); err != nil {
					c.logger.Error(context.Background(), "Failed to reconnect to RabbitMQ", telemetry.Fields{
						"error": err.Error(),
					})
					time.Sleep(5 * time.Second)
					continue
				}

				// Restart consumers
				if err := c.Start(); err != nil {
					c.logger.Error(context.Background(), "Failed to restart consumers", telemetry.Fields{
						"error": err.Error(),
					})
					time.Sleep(5 * time.Second)
					continue
				}

				c.logger.Info(context.Background(), "Successfully reconnected to RabbitMQ", nil)
				return
			}
		}
	case <-c.shutdownSignal:
		return
	}
}

// reconnect membuat ulang koneksi ke RabbitMQ setelah terputus
func (c *EmailConsumer) reconnect() error {
	// Tutup koneksi dan channel lama jika masih ada
	if c.channel != nil {
		c.channel.Close()
	}
	if c.conn != nil {
		c.conn.Close()
	}

	// Connect kembali
	return c.connect()
}

// consumeQueue memulai consumer untuk memproses email dari queue tertentu
func (c *EmailConsumer) consumeQueue(queueName string) error {
	// Buat consumer tag unik
	consumerTag := fmt.Sprintf("email-consumer-%s-%d", queueName, time.Now().UnixNano())
	c.consumeTags = append(c.consumeTags, consumerTag)

	// Consume messages dari queue
	deliveries, err := c.channel.Consume(
		queueName,   // queue
		consumerTag, // consumer tag
		false,       // auto-ack (false = manual ack)
		false,       // exclusive
		false,       // no-local
		false,       // no-wait
		nil,         // arguments
	)
	if err != nil {
		return domain.ExternalServiceError(fmt.Sprintf("Failed to consume from queue %s", queueName), err)
	}

	// Mulai worker goroutine untuk memproses pesan
	c.consumerWaitGrp.Add(1)
	go func() {
		defer c.consumerWaitGrp.Done()

		for {
			select {
			case delivery, ok := <-deliveries:
				// Channel ditutup
				if !ok {
					c.logger.Warn(context.Background(), fmt.Sprintf("Consumer channel for %s closed", queueName), nil)
					return
				}

				// Proses pesan
				c.handleDelivery(delivery, queueName)

			case <-c.shutdownSignal:
				c.logger.Info(context.Background(), fmt.Sprintf("Shutting down consumer for %s", queueName), nil)
				return
			}
		}
	}()

	return nil
}

// handleDelivery memproses pesan yang diterima dari queue
func (c *EmailConsumer) handleDelivery(delivery amqp.Delivery, queueName string) {
	// Buat context dengan timeout untuk proses pengiriman email
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Extract trace ID dari header jika ada
	var traceID string
	if traceIDHeader, ok := delivery.Headers["x-trace-id"]; ok {
		if traceIDStr, ok := traceIDHeader.(string); ok {
			traceID = traceIDStr
			// Set trace ID ke context
			ctx = context.WithValue(ctx, "traceID", traceID)
		}
	}

	// Parse pesan
	var message EmailMessage
	if err := json.Unmarshal(delivery.Body, &message); err != nil {
		c.logger.Error(ctx, "Failed to unmarshal email message", telemetry.Fields{
			"error":     err.Error(),
			"queue":     queueName,
			"trace_id":  traceID,
			"messageId": delivery.MessageId,
		})

		// Reject pesan tanpa requeue
		delivery.Reject(false)
		return
	}

	c.logger.Info(ctx, "Processing email from queue", telemetry.Fields{
		"email_id":  message.ID,
		"priority":  message.Priority,
		"queue":     queueName,
		"trace_id":  message.TraceID,
		"messageId": delivery.MessageId,
	})

	// Proses email menggunakan email service
	err := c.emailService.ProcessPendingEmails(ctx)
	if err != nil {
		c.logger.Error(ctx, "Failed to process email", telemetry.Fields{
			"error":     err.Error(),
			"email_id":  message.ID,
			"queue":     queueName,
			"trace_id":  message.TraceID,
			"messageId": delivery.MessageId,
		})

		// Requeue pesan jika error bersifat sementara (seperti database error)
		if appErr, ok := err.(*domain.AppError); ok {
			if appErr.Type == domain.ErrorTypeDatabase || appErr.Type == domain.ErrorTypeExternal {
				// Requeue message
				delivery.Reject(true)
				return
			}
		}

		// Reject pesan tanpa requeue untuk error lainnya
		delivery.Reject(false)
		return
	}

	// Proses berhasil, acknowledge pesan
	if err := delivery.Ack(false); err != nil {
		c.logger.Error(ctx, "Failed to acknowledge message", telemetry.Fields{
			"error":     err.Error(),
			"email_id":  message.ID,
			"queue":     queueName,
			"trace_id":  message.TraceID,
			"messageId": delivery.MessageId,
		})
	}

	c.logger.Info(ctx, "Successfully processed email", telemetry.Fields{
		"email_id":  message.ID,
		"priority":  message.Priority,
		"queue":     queueName,
		"trace_id":  message.TraceID,
		"messageId": delivery.MessageId,
	})
}

// Stop menghentikan semua consumer dan menutup koneksi
func (c *EmailConsumer) Stop() error {
	// Kirim sinyal untuk menghentikan semua goroutine
	close(c.shutdownSignal)

	// Cancel semua consumer
	for _, tag := range c.consumeTags {
		if err := c.channel.Cancel(tag, false); err != nil {
			c.logger.Error(context.Background(), "Failed to cancel consumer", telemetry.Fields{
				"consumer_tag": tag,
				"error":        err.Error(),
			})
		}
	}

	// Tunggu semua consumer selesai
	c.consumerWaitGrp.Wait()

	// Tutup koneksi dan channel
	if c.channel != nil {
		c.channel.Close()
	}
	if c.conn != nil {
		c.conn.Close()
	}

	c.logger.Info(context.Background(), "Email consumers stopped", nil)

	return nil
}
