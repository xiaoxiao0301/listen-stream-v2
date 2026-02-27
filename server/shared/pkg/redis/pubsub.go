package redis

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/redis/go-redis/v9"
)

// PubSub wraps Redis Pub/Sub functionality.
type PubSub struct {
	client  *Client
	pubsub  *redis.PubSub
	mu      sync.RWMutex
	handlers map[string][]MessageHandler
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

// MessageHandler is a function that handles pub/sub messages.
type MessageHandler func(channel string, message string) error

// NewPubSub creates a new Pub/Sub instance.
func NewPubSub(client *Client) *PubSub {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &PubSub{
		client:   client,
		handlers: make(map[string][]MessageHandler),
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Subscribe subscribes to one or more channels.
func (ps *PubSub) Subscribe(channels ...string) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	
	if ps.pubsub == nil {
		ps.pubsub = ps.client.universal.Subscribe(ps.ctx, channels...)
	} else {
		if err := ps.pubsub.Subscribe(ps.ctx, channels...); err != nil {
			return fmt.Errorf("failed to subscribe: %w", err)
		}
	}
	
	return nil
}

// Unsubscribe unsubscribes from one or more channels.
func (ps *PubSub) Unsubscribe(channels ...string) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	
	if ps.pubsub == nil {
		return nil
	}
	
	if err := ps.pubsub.Unsubscribe(ps.ctx, channels...); err != nil {
		return fmt.Errorf("failed to unsubscribe: %w", err)
	}
	
	// Remove handlers for these channels
	for _, channel := range channels {
		delete(ps.handlers, channel)
	}
	
	return nil
}

// PSubscribe subscribes to channels matching a pattern.
func (ps *PubSub) PSubscribe(patterns ...string) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	
	if ps.pubsub == nil {
		ps.pubsub = ps.client.universal.PSubscribe(ps.ctx, patterns...)
	} else {
		if err := ps.pubsub.PSubscribe(ps.ctx, patterns...); err != nil {
			return fmt.Errorf("failed to psubscribe: %w", err)
		}
	}
	
	return nil
}

// PUnsubscribe unsubscribes from patterns.
func (ps *PubSub) PUnsubscribe(patterns ...string) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	
	if ps.pubsub == nil {
		return nil
	}
	
	if err := ps.pubsub.PUnsubscribe(ps.ctx, patterns...); err != nil {
		return fmt.Errorf("failed to punsubscribe: %w", err)
	}
	
	return nil
}

// OnMessage registers a handler for messages on a specific channel.
func (ps *PubSub) OnMessage(channel string, handler MessageHandler) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	
	if ps.handlers[channel] == nil {
		ps.handlers[channel] = make([]MessageHandler, 0)
	}
	
	ps.handlers[channel] = append(ps.handlers[channel], handler)
}

// Start starts listening for messages.
func (ps *PubSub) Start() error {
	ps.mu.Lock()
	if ps.pubsub == nil {
		ps.mu.Unlock()
		return fmt.Errorf("not subscribed to any channels")
	}
	ps.mu.Unlock()
	
	ps.wg.Add(1)
	go ps.receiveMessages()
	
	return nil
}

// receiveMessages receives and dispatches messages to handlers.
func (ps *PubSub) receiveMessages() {
	defer ps.wg.Done()
	
	for {
		select {
		case <-ps.ctx.Done():
			return
		default:
			msg, err := ps.pubsub.ReceiveMessage(ps.ctx)
			if err != nil {
				if ps.ctx.Err() != nil {
					// Context cancelled, exit gracefully
					return
				}
				log.Printf("Error receiving message: %v", err)
				continue
			}
			
			ps.dispatch(msg.Channel, msg.Payload)
		}
	}
}

// dispatch dispatches a message to registered handlers.
func (ps *PubSub) dispatch(channel, message string) {
	ps.mu.RLock()
	handlers := ps.handlers[channel]
	ps.mu.RUnlock()
	
	if len(handlers) == 0 {
		return
	}
	
	// Call all handlers for this channel
	for _, handler := range handlers {
		if err := handler(channel, message); err != nil {
			log.Printf("Error in message handler for channel %s: %v", channel, err)
		}
	}
}

// Publish publishes a message to a channel.
func (ps *PubSub) Publish(ctx context.Context, channel, message string) error {
	if err := ps.client.universal.Publish(ctx, channel, message).Err(); err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}
	return nil
}

// PublishJSON publishes a JSON-encoded message to a channel.
func (ps *PubSub) PublishJSON(ctx context.Context, channel string, data interface{}) error {
	// In a real implementation, you would use json.Marshal here
	// For now, we'll just convert to string
	message := fmt.Sprintf("%v", data)
	return ps.Publish(ctx, channel, message)
}

// Close closes the Pub/Sub connection.
func (ps *PubSub) Close() error {
	// Cancel context to stop receiving messages
	ps.cancel()
	
	// Wait for goroutines to finish
	ps.wg.Wait()
	
	// Close pubsub connection
	ps.mu.Lock()
	defer ps.mu.Unlock()
	
	if ps.pubsub != nil {
		if err := ps.pubsub.Close(); err != nil {
			return fmt.Errorf("failed to close pubsub: %w", err)
		}
	}
	
	return nil
}

// ConfigChangeNotifier is a helper for configuration change notifications.
type ConfigChangeNotifier struct {
	pubsub *PubSub
}

// NewConfigChangeNotifier creates a new config change notifier.
func NewConfigChangeNotifier(client *Client) *ConfigChangeNotifier {
	return &ConfigChangeNotifier{
		pubsub: NewPubSub(client),
	}
}

// Subscribe subscribes to configuration changes.
func (n *ConfigChangeNotifier) Subscribe(handler func(message string) error) error {
	channel := ConfigChangeChannel()
	
	if err := n.pubsub.Subscribe(channel); err != nil {
		return err
	}
	
	n.pubsub.OnMessage(channel, func(ch, msg string) error {
		return handler(msg)
	})
	
	return n.pubsub.Start()
}

// Notify publishes a configuration change notification.
func (n *ConfigChangeNotifier) Notify(ctx context.Context, message string) error {
	channel := ConfigChangeChannel()
	return n.pubsub.Publish(ctx, channel, message)
}

// Close closes the notifier.
func (n *ConfigChangeNotifier) Close() error {
	return n.pubsub.Close()
}

// SyncEventNotifier is a helper for sync event notifications.
type SyncEventNotifier struct {
	pubsub *PubSub
}

// NewSyncEventNotifier creates a new sync event notifier.
func NewSyncEventNotifier(client *Client) *SyncEventNotifier {
	return &SyncEventNotifier{
		pubsub: NewPubSub(client),
	}
}

// Subscribe subscribes to sync events for a specific user.
func (n *SyncEventNotifier) Subscribe(userID string, handler func(message string) error) error {
	channel := SyncEventChannel(userID)
	
	if err := n.pubsub.Subscribe(channel); err != nil {
		return err
	}
	
	n.pubsub.OnMessage(channel, func(ch, msg string) error {
		return handler(msg)
	})
	
	return n.pubsub.Start()
}

// Notify publishes a sync event to a specific user.
func (n *SyncEventNotifier) Notify(ctx context.Context, userID, message string) error {
	channel := SyncEventChannel(userID)
	return n.pubsub.Publish(ctx, channel, message)
}

// NotifyAll publishes a sync event to all users (broadcast).
func (n *SyncEventNotifier) NotifyAll(ctx context.Context, message string) error {
	channel := PubSubChannel("sync:all")
	return n.pubsub.Publish(ctx, channel, message)
}

// Close closes the notifier.
func (n *SyncEventNotifier) Close() error {
	return n.pubsub.Close()
}
