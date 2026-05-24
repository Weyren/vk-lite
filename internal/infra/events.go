package infra

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/Weyren/vk-lite/pkg/utils"
	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitPublisher struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

func NewRabbitPublisher(cfg *utils.Config) *RabbitPublisher {
	var lastErr error
	for attempt := 1; attempt <= 15; attempt++ {
		conn, err := amqp.Dial(cfg.AMQPURL)
		if err != nil {
			lastErr = err
			log.Printf("rabbitmq is not ready yet, attempt %d/15: %v", attempt, err)
			time.Sleep(time.Second)
			continue
		}

		channel, err := conn.Channel()
		if err != nil {
			lastErr = err
			_ = conn.Close()
			time.Sleep(time.Second)
			continue
		}

		if err := channel.ExchangeDeclare("vk_lite.events", "topic", true, false, false, false, nil); err != nil {
			lastErr = err
			_ = channel.Close()
			_ = conn.Close()
			time.Sleep(time.Second)
			continue
		}

		return &RabbitPublisher{conn: conn, channel: channel}
	}

	log.Printf("rabbitmq publisher disabled: %v", lastErr)
	return nil
}

func (p *RabbitPublisher) Publish(ctx context.Context, eventType string, payload any) {
	if p == nil || p.channel == nil {
		return
	}

	body, err := json.Marshal(map[string]any{
		"type":       eventType,
		"occurredAt": time.Now().UTC(),
		"payload":    payload,
	})
	if err != nil {
		log.Printf("cannot marshal event %s: %v", eventType, err)
		return
	}

	err = p.channel.PublishWithContext(ctx, "vk_lite.events", eventType, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
	})
	if err != nil {
		log.Printf("cannot publish event %s: %v", eventType, err)
	}
}
