package delivery

import (
	"context"
	"email-service/internal/domain"
	"email-service/pkg/telemetry"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/ses/types"
)

// SESAdapter adalah adapter untuk mengirim email menggunakan AWS SES
type SESAdapter struct {
	client *ses.Client
	region string
	logger telemetry.Logger
}

// NewSESAdapter membuat instance baru SES adapter
func NewSESAdapter(config SESConfig, logger telemetry.Logger) domain.EmailDelivery {
	// Buat konfigurasi AWS
	cfg := aws.Config{
		Region:      config.Region,
		Credentials: credentials.NewStaticCredentialsProvider(config.AccessKey, config.SecretKey, ""),
	}

	// Buat client SES
	client := ses.NewFromConfig(cfg)

	return &SESAdapter{
		client: client,
		region: config.Region,
		logger: logger,
	}
}

// Send mengirim email menggunakan AWS SES
func (a *SESAdapter) Send(ctx context.Context, mail *domain.Email) error {
	// Siapkan input untuk SendEmail
	input := &ses.SendEmailInput{
		Source: aws.String(mail.From),
		Destination: &types.Destination{
			ToAddresses:  mail.To,
			CcAddresses:  mail.Cc,
			BccAddresses: mail.Bcc,
		},
		Message: &types.Message{
			Subject: &types.Content{
				Data:    aws.String(mail.Subject),
				Charset: aws.String("UTF-8"),
			},
			Body: &types.Body{},
		},
	}

	// Set konten email
	if mail.PlainBody != "" {
		input.Message.Body.Text = &types.Content{
			Data:    aws.String(mail.PlainBody),
			Charset: aws.String("UTF-8"),
		}
	}

	if mail.HTMLBody != "" {
		input.Message.Body.Html = &types.Content{
			Data:    aws.String(mail.HTMLBody),
			Charset: aws.String("UTF-8"),
		}
	}

	// Tambahkan headers dari metadata
	if len(mail.Metadata) > 0 {
		headers := make([]types.MessageTag, 0, len(mail.Metadata))
		for key, value := range mail.Metadata {
			headers = append(headers, types.MessageTag{
				Name:  aws.String(key),
				Value: aws.String(value),
			})
		}
		input.Tags = headers
	}

	// Kirim email
	_, err := a.client.SendEmail(ctx, input)
	if err != nil {
		return domain.ExternalServiceError(fmt.Sprintf("Failed to send email via AWS SES: %v", err), err)
	}

	return nil
}

// Name mengembalikan nama provider
func (a *SESAdapter) Name() string {
	return string(domain.ProviderSES)
}
