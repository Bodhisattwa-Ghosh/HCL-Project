package realtime

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/Bodhisattwa-Ghosh/HCL-Project/backend/internal/domain"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Broker fans a single poll result update out to every API instance. Redis
// Pub/Sub means Server-Sent Event clients work even when the app is scaled.
type Broker struct {
	client *redis.Client
}

func New(client *redis.Client) *Broker {
	return &Broker{client: client}
}

func (b *Broker) Publish(ctx context.Context, poll domain.PublicPoll) error {
	payload, err := json.Marshal(poll)
	if err != nil {
		return fmt.Errorf("marshal poll update: %w", err)
	}
	return b.client.Publish(ctx, Channel(poll.ID), payload).Err()
}

func (b *Broker) Subscribe(ctx context.Context, pollID string) (*redis.PubSub, error) {
	subscription := b.client.Subscribe(ctx, Channel(pollID))
	if _, err := subscription.Receive(ctx); err != nil {
		_ = subscription.Close()
		return nil, err
	}
	return subscription, nil
}

func Channel(pollID string) string {
	return "pulsepoll:poll:" + pollID + ":events"
}

func CounterKey(pollID primitive.ObjectID) string {
	return "pulsepoll:poll:" + pollID.Hex() + ":votes"
}
