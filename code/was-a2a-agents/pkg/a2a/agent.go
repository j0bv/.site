// pkg/a2a/agent.go
// WAS A2A Agent Implementation with Dynamic Expansion

package a2a

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/google/uuid"
)

// Agent represents a WAS A2A agent
type Agent struct {
	ID          string
	Name        string
	Description string
	Platform    string

	// Communication
	client       *pubsub.Client
	topic        *pubsub.Topic
	subscription *pubsub.Subscription

	// Agent state
	tools    map[string]Tool
	handlers map[string]MessageHandler
	mu       sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc

	// Metrics
	messagesProcessed int64
	lastActivity      time.Time
}

// Tool interface for agent capabilities
type Tool interface {
	Name() string
	Execute(ctx context.Context, params map[string]interface{}) (interface{}, error)
	Validate(params map[string]interface{}) error
}

// MessageHandler processes incoming A2A messages
type MessageHandler func(ctx context.Context, msg *A2AMessage) error

// A2AMessage represents communication between agents
type A2AMessage struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	From      string                 `json:"from"`
	To        string                 `json:"to"`
	Timestamp time.Time              `json:"timestamp"`
	Content   map[string]interface{} `json:"content"`
	Priority  int                    `json:"priority"`
	ReplyTo   string                 `json:"reply_to,omitempty"`
}

// NewAgent creates a new A2A agent
func NewAgent(projectID, agentID, name, platform string) (*Agent, error) {
	ctx, cancel := context.WithCancel(context.Background())

	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create pubsub client: %w", err)
	}

	// Create topic for this agent
	topicName := fmt.Sprintf("was-%s-%s", platform, agentID)
	topic := client.Topic(topicName)

	// Create subscription
	subName := fmt.Sprintf("%s-subscription", topicName)
	subscription := client.Subscription(subName)

	agent := &Agent{
		ID:           agentID,
		Name:         name,
		Platform:     platform,
		client:       client,
		topic:        topic,
		subscription: subscription,
		tools:        make(map[string]Tool),
		handlers:     make(map[string]MessageHandler),
		ctx:          ctx,
		cancel:       cancel,
		lastActivity: time.Now(),
	}

	return agent, nil
}

// AddTool registers a new tool with the agent
func (a *Agent) AddTool(tool Tool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.tools[tool.Name()] = tool
}

// AddMessageHandler registers a message handler
func (a *Agent) AddMessageHandler(messageType string, handler MessageHandler) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.handlers[messageType] = handler
}

// SendMessage sends an A2A message to another agent
func (a *Agent) SendMessage(ctx context.Context, to, messageType string, content map[string]interface{}) error {
	msg := &A2AMessage{
		ID:        uuid.New().String(),
		Type:      messageType,
		From:      a.ID,
		To:        to,
		Timestamp: time.Now(),
		Content:   content,
		Priority:  1,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Send to recipient's topic
	topicName := fmt.Sprintf("was-%s", to)
	topic := a.client.Topic(topicName)

	result := topic.Publish(ctx, &pubsub.Message{
		Data: data,
		Attributes: map[string]string{
			"type": messageType,
			"from": a.ID,
		},
	})

	_, err = result.Get(ctx)
	return err
}

// StartListening begins processing incoming messages
func (a *Agent) StartListening() error {
	log.Printf("Agent %s starting to listen for messages", a.ID)

	return a.subscription.Receive(a.ctx, func(ctx context.Context, msg *pubsub.Message) {
		var a2aMsg A2AMessage
		if err := json.Unmarshal(msg.Data, &a2aMsg); err != nil {
			log.Printf("Failed to unmarshal message: %v", err)
			msg.Nack()
			return
		}

		if err := a.handleMessage(ctx, &a2aMsg); err != nil {
			log.Printf("Failed to handle message: %v", err)
			msg.Nack()
			return
		}

		msg.Ack()
		a.messagesProcessed++
		a.lastActivity = time.Now()
	})
}

// handleMessage processes incoming A2A messages
func (a *Agent) handleMessage(ctx context.Context, msg *A2AMessage) error {
	a.mu.RLock()
	handler, exists := a.handlers[msg.Type]
	a.mu.RUnlock()

	if !exists {
		log.Printf("No handler for message type: %s", msg.Type)
		return nil
	}

	return handler(ctx, msg)
}

// Stop stops the agent
func (a *Agent) Stop() {
	a.cancel()
}

// GetMetrics returns agent performance metrics
func (a *Agent) GetMetrics() map[string]interface{} {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return map[string]interface{}{
		"id":                 a.ID,
		"name":               a.Name,
		"platform":           a.Platform,
		"messages_processed": a.messagesProcessed,
		"last_activity":      a.lastActivity,
		"tools_count":        len(a.tools),
		"handlers_count":     len(a.handlers),
	}
}
