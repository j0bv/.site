package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"cloud.google.com/go/pubsub"
)

// A2ACommunicationManager manages Agent-to-Agent communication
type A2ACommunicationManager struct {
	projectID     string
	pubsubClient  *pubsub.Client
	context       context.Context
	agents        map[string]*AIAgent
	messageQueue  chan *A2AProtocol
	mu            sync.RWMutex
	isRunning     bool
}

// NewA2ACommunicationManager creates a new A2A communication manager
func NewA2ACommunicationManager(projectID string) (*A2ACommunicationManager, error) {
	ctx := context.Background()
	
	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to create Pub/Sub client: %v", err)
	}

	manager := &A2ACommunicationManager{
		projectID:    projectID,
		pubsubClient: client,
		context:      ctx,
		agents:       make(map[string]*AIAgent),
		messageQueue: make(chan *A2AProtocol, 1000),
		isRunning:    false,
	}

	return manager, nil
}

// RegisterAgent registers an agent with the communication manager
func (m *A2ACommunicationManager) RegisterAgent(agent *AIAgent) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.agents[agent.ID] = agent
	log.Printf("Registered agent: %s (%s)", agent.ID, agent.Role)
	return nil
}

// UnregisterAgent removes an agent from the communication manager
func (m *A2ACommunicationManager) UnregisterAgent(agentID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.agents, agentID)
	log.Printf("Unregistered agent: %s", agentID)
}

// Start starts the A2A communication manager
func (m *A2ACommunicationManager) Start() error {
	if m.isRunning {
		return fmt.Errorf("communication manager is already running")
	}

	m.isRunning = true
	log.Println("Starting A2A Communication Manager")

	// Start message processing
	go m.processMessages()

	// Start heartbeat monitoring
	go m.heartbeatMonitor()

	return nil
}

// Stop stops the A2A communication manager
func (m *A2ACommunicationManager) Stop() {
	m.isRunning = false
	log.Println("Stopping A2A Communication Manager")
}

// SendMessage sends a message between agents
func (m *A2ACommunicationManager) SendMessage(senderID, receiverID, messageType, taskID string, payload map[string]interface{}) error {
	message := &A2AProtocol{
		MessageType: messageType,
		SenderID:    senderID,
		ReceiverID:  receiverID,
		TaskID:      taskID,
		Payload:     payload,
		Timestamp:   time.Now(),
		Priority:    5,
	}

	// Add to message queue for processing
	select {
	case m.messageQueue <- message:
		return nil
	default:
		return fmt.Errorf("message queue is full")
	}
}

// BroadcastMessage broadcasts a message to all agents
func (m *A2ACommunicationManager) BroadcastMessage(senderID, messageType, taskID string, payload map[string]interface{}) error {
	return m.SendMessage(senderID, "", messageType, taskID, payload)
}

// processMessages processes messages from the queue
func (m *A2ACommunicationManager) processMessages() {
	for message := range m.messageQueue {
		if !m.isRunning {
			break
		}

		// Publish to Pub/Sub
		if err := m.publishMessage(message); err != nil {
			log.Printf("Failed to publish message: %v", err)
		}
	}
}

// publishMessage publishes a message to Pub/Sub
func (m *A2ACommunicationManager) publishMessage(message *A2AProtocol) error {
	messageData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %v", err)
	}

	topic := m.pubsubClient.Topic("agent-communication")
	result := topic.Publish(m.context, &pubsub.Message{
		Data: messageData,
		Attributes: map[string]string{
			"message_type": message.MessageType,
			"sender_id":    message.SenderID,
			"receiver_id":  message.ReceiverID,
			"priority":     fmt.Sprintf("%d", message.Priority),
			"timestamp":    message.Timestamp.Format(time.RFC3339),
		},
	})

	_, err = result.Get(m.context)
	return err
}

// heartbeatMonitor monitors agent health
func (m *A2ACommunicationManager) heartbeatMonitor() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if !m.isRunning {
			break
		}

		m.mu.RLock()
		agentCount := len(m.agents)
		m.mu.RUnlock()

		log.Printf("A2A Communication Manager heartbeat - Active agents: %d", agentCount)
	}
}

// GetAgentStatus returns the status of all registered agents
func (m *A2ACommunicationManager) GetAgentStatus() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := make(map[string]interface{})
	status["total_agents"] = len(m.agents)
	status["agents"] = make(map[string]interface{})

	for id, agent := range m.agents {
		status["agents"].(map[string]interface{})[id] = map[string]interface{}{
			"id":           agent.ID,
			"role":         agent.Role,
			"description":  agent.Description,
			"is_active":    agent.IsActive,
			"capabilities": agent.Capabilities,
		}
	}

	return status
}

// DistributeTask distributes a task to the most appropriate agent
func (m *A2ACommunicationManager) DistributeTask(task *AgentTask) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Find the best agent for the task
	bestAgent := m.findBestAgentForTask(task)
	if bestAgent == nil {
		return fmt.Errorf("no suitable agent found for task: %s", task.Type)
	}

	// Send task to the agent
	payload := map[string]interface{}{
		"task": task,
	}

	return m.SendMessage("orchestrator", bestAgent.ID, "task_assignment", task.ID, payload)
}

// findBestAgentForTask finds the best agent for a given task
func (m *A2ACommunicationManager) findBestAgentForTask(task *AgentTask) *AIAgent {
	var bestAgent *AIAgent
	bestScore := 0

	for _, agent := range m.agents {
		if !agent.IsActive {
			continue
		}

		score := m.calculateAgentScore(agent, task)
		if score > bestScore {
			bestScore = score
			bestAgent = agent
		}
	}

	return bestAgent
}

// calculateAgentScore calculates how well an agent matches a task
func (m *A2ACommunicationManager) calculateAgentScore(agent *AIAgent, task *AgentTask) int {
	score := 0

	// Role-based scoring
	switch task.Type {
	case "code_improvement":
		if agent.Role == "Code Enhancement" {
			score += 10
		}
	case "testing_validation":
		if agent.Role == "Testing & Validation" {
			score += 10
		}
	case "deployment_pipeline":
		if agent.Role == "Deployment & Operations" {
			score += 10
		}
	case "monitoring":
		if agent.Role == "Monitoring & Analytics" {
			score += 10
		}
	}

	// Capability-based scoring
	for _, capability := range agent.Capabilities {
		if capability == task.Type {
			score += 5
		}
	}

	// Priority-based scoring
	score += task.Priority

	return score
}

// CreateTask creates a new task and distributes it
func (m *A2ACommunicationManager) CreateTask(taskType, description string, context map[string]interface{}, priority int) (*AgentTask, error) {
	task := &AgentTask{
		ID:          fmt.Sprintf("task_%d", time.Now().UnixNano()),
		Type:        taskType,
		Description: description,
		Context:     context,
		Priority:    priority,
		CreatedAt:   time.Now(),
		Status:      "pending",
	}

	// Publish task to task queue
	taskData, err := json.Marshal(task)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal task: %v", err)
	}

	topic := m.pubsubClient.Topic("agent-task-queue")
	result := topic.Publish(m.context, &pubsub.Message{
		Data: taskData,
		Attributes: map[string]string{
			"task_type": task.Type,
			"priority":  fmt.Sprintf("%d", task.Priority),
			"status":    task.Status,
		},
	})

	_, err = result.Get(m.context)
	if err != nil {
		return nil, fmt.Errorf("failed to publish task: %v", err)
	}

	// Also try to distribute directly
	go m.DistributeTask(task)

	return task, nil
}

// GetTaskResults retrieves results from the results topic
func (m *A2ACommunicationManager) GetTaskResults(callback func(*AgentResult)) error {
	sub := m.pubsubClient.Subscription("agent-results-sub")
	
	return sub.Receive(m.context, func(ctx context.Context, msg *pubsub.Message) {
		var result AgentResult
		if err := json.Unmarshal(msg.Data, &result); err != nil {
			log.Printf("Failed to unmarshal result: %v", err)
			msg.Nack()
			return
		}

		callback(&result)
		msg.Ack()
	})
}

// Close closes the communication manager
func (m *A2ACommunicationManager) Close() error {
	m.Stop()
	return m.pubsubClient.Close()
}
