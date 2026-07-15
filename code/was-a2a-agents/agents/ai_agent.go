package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"cloud.google.com/go/aiplatform/apiv1"
	"cloud.google.com/go/pubsub"
	"google.golang.org/api/option"
)

// A2AProtocol defines the Agent-to-Agent communication protocol
type A2AProtocol struct {
	MessageType string                 `json:"message_type"`
	SenderID    string                 `json:"sender_id"`
	ReceiverID  string                 `json:"receiver_id,omitempty"` // Empty for broadcast
	TaskID      string                 `json:"task_id"`
	Payload     map[string]interface{} `json:"payload"`
	Timestamp   time.Time              `json:"timestamp"`
	Priority    int                    `json:"priority"` // 1-10, higher = more urgent
}

// AgentTask represents a task for an AI agent
type AgentTask struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Context     map[string]interface{} `json:"context"`
	Priority    int                    `json:"priority"`
	CreatedAt   time.Time             `json:"created_at"`
	AssignedTo  string                `json:"assigned_to,omitempty"`
	Status      string                `json:"status"` // pending, in_progress, completed, failed
}

// AgentResult represents the result of an agent's work
type AgentResult struct {
	TaskID    string                 `json:"task_id"`
	AgentID   string                 `json:"agent_id"`
	Status    string                 `json:"status"`
	Result    map[string]interface{} `json:"result"`
	Error     string                 `json:"error,omitempty"`
	Timestamp time.Time             `json:"timestamp"`
}

// AIAgent represents an AI-powered agent using Vertex AI
type AIAgent struct {
	ID          string
	Role        string
	Description string
	Model       string
	Client      *aiplatform.PredictionClient
	PubSub      *pubsub.Client
	Context     context.Context
	Capabilities []string
	IsActive    bool
}

// NewAIAgent creates a new AI agent with Vertex AI integration
func NewAIAgent(id, role, description string, projectID, location string) (*AIAgent, error) {
	ctx := context.Background()
	
	// Initialize Vertex AI client
	client, err := aiplatform.NewPredictionClient(ctx, option.WithQuotaProject(projectID))
	if err != nil {
		return nil, fmt.Errorf("failed to create Vertex AI client: %v", err)
	}

	// Initialize Pub/Sub client
	pubsubClient, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to create Pub/Sub client: %v", err)
	}

	agent := &AIAgent{
		ID:          id,
		Role:        role,
		Description: description,
		Model:       "text-bison@001", // Using PaLM 2 model
		Client:      client,
		PubSub:      pubsubClient,
		Context:     ctx,
		Capabilities: []string{},
		IsActive:    true,
	}

	return agent, nil
}

// ProcessTask processes a task using Vertex AI
func (a *AIAgent) ProcessTask(task *AgentTask) (*AgentResult, error) {
	log.Printf("Agent %s processing task %s: %s", a.ID, task.ID, task.Description)

	// Prepare the prompt for the AI model
	prompt := a.buildPrompt(task)
	
	// Call Vertex AI
	response, err := a.callVertexAI(prompt)
	if err != nil {
		return &AgentResult{
			TaskID:    task.ID,
			AgentID:   a.ID,
			Status:    "failed",
			Error:     err.Error(),
			Timestamp: time.Now(),
		}, err
	}

	// Parse the response
	result := a.parseResponse(response, task)

	return &AgentResult{
		TaskID:    task.ID,
		AgentID:   a.ID,
		Status:    "completed",
		Result:    result,
		Timestamp: time.Now(),
	}, nil
}

// buildPrompt creates a structured prompt for the AI model
func (a *AIAgent) buildPrompt(task *AgentTask) string {
	return fmt.Sprintf(`
You are %s, an AI agent specialized in %s.

Task: %s
Description: %s

Context: %s

Please provide a detailed response with:
1. Analysis of the task
2. Step-by-step approach
3. Expected outcomes
4. Any recommendations or concerns

Format your response as JSON with the following structure:
{
  "analysis": "your analysis here",
  "approach": ["step1", "step2", "step3"],
  "outcomes": ["outcome1", "outcome2"],
  "recommendations": "your recommendations here",
  "confidence": 0.85
}
`, a.Role, a.Description, task.Type, task.Description, a.formatContext(task.Context))
}

// callVertexAI calls the Vertex AI API
func (a *AIAgent) callVertexAI(prompt string) (string, error) {
	// This is a simplified version - in production you'd use the actual Vertex AI API
	// For now, we'll simulate the response
	return fmt.Sprintf(`{
		"analysis": "Task analyzed by %s agent",
		"approach": ["Analyze requirements", "Plan implementation", "Execute solution"],
		"outcomes": ["Improved code quality", "Better performance", "Enhanced maintainability"],
		"recommendations": "Consider implementing automated testing",
		"confidence": 0.9
	}`, a.Role), nil
}

// parseResponse parses the AI response into a structured format
func (a *AIAgent) parseResponse(response string, task *AgentTask) map[string]interface{} {
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		log.Printf("Failed to parse AI response: %v", err)
		return map[string]interface{}{
			"raw_response": response,
			"error":        "Failed to parse response",
		}
	}
	return result
}

// formatContext formats the task context for the prompt
func (a *AIAgent) formatContext(context map[string]interface{}) string {
	contextJSON, err := json.MarshalIndent(context, "", "  ")
	if err != nil {
		return "Unable to format context"
	}
	return string(contextJSON)
}

// SendMessage sends a message to other agents via Pub/Sub
func (a *AIAgent) SendMessage(messageType, receiverID, taskID string, payload map[string]interface{}) error {
	msg := &A2AProtocol{
		MessageType: messageType,
		SenderID:    a.ID,
		ReceiverID:  receiverID,
		TaskID:      taskID,
		Payload:     payload,
		Timestamp:   time.Now(),
		Priority:    5, // Default priority
	}

	messageData, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %v", err)
	}

	topic := a.PubSub.Topic("agent-communication")
	result := topic.Publish(a.Context, &pubsub.Message{
		Data: messageData,
		Attributes: map[string]string{
			"message_type": messageType,
			"sender_id":    a.ID,
			"receiver_id":  receiverID,
			"priority":     fmt.Sprintf("%d", msg.Priority),
		},
	})

	_, err = result.Get(a.Context)
	return err
}

// ListenForMessages listens for incoming messages from other agents
func (a *AIAgent) ListenForMessages(callback func(*A2AProtocol)) error {
	sub := a.PubSub.Subscription("agent-communication-sub")
	
	return sub.Receive(a.Context, func(ctx context.Context, msg *pubsub.Message) {
		var protocol A2AProtocol
		if err := json.Unmarshal(msg.Data, &protocol); err != nil {
			log.Printf("Failed to unmarshal message: %v", err)
			msg.Nack()
			return
		}

		// Check if message is for this agent or broadcast
		if protocol.ReceiverID != "" && protocol.ReceiverID != a.ID {
			msg.Ack()
			return
		}

		// Process the message
		callback(&protocol)
		msg.Ack()
	})
}

// StartAgent starts the agent's main processing loop
func (a *AIAgent) StartAgent() error {
	log.Printf("Starting AI Agent: %s (%s)", a.ID, a.Role)
	
	// Start listening for messages
	go func() {
		if err := a.ListenForMessages(a.handleMessage); err != nil {
			log.Printf("Error listening for messages: %v", err)
		}
	}()

	// Start task processing loop
	go a.taskProcessingLoop()

	return nil
}

// handleMessage handles incoming messages from other agents
func (a *AIAgent) handleMessage(msg *A2AProtocol) {
	log.Printf("Agent %s received message: %s from %s", a.ID, msg.MessageType, msg.SenderID)
	
	switch msg.MessageType {
	case "task_assignment":
		a.handleTaskAssignment(msg)
	case "collaboration_request":
		a.handleCollaborationRequest(msg)
	case "status_update":
		a.handleStatusUpdate(msg)
	default:
		log.Printf("Unknown message type: %s", msg.MessageType)
	}
}

// handleTaskAssignment handles task assignment messages
func (a *AIAgent) handleTaskAssignment(msg *A2AProtocol) {
	taskData, ok := msg.Payload["task"].(map[string]interface{})
	if !ok {
		log.Printf("Invalid task data in message")
		return
	}

	// Convert to AgentTask
	task := &AgentTask{
		ID:          taskData["id"].(string),
		Type:        taskData["type"].(string),
		Description: taskData["description"].(string),
		Context:     taskData["context"].(map[string]interface{}),
		Priority:    int(taskData["priority"].(float64)),
		CreatedAt:   time.Now(),
		AssignedTo:  a.ID,
		Status:      "pending",
	}

	// Process the task
	result, err := a.ProcessTask(task)
	if err != nil {
		log.Printf("Failed to process task: %v", err)
		return
	}

	// Send result back
	a.SendMessage("task_result", msg.SenderID, task.ID, map[string]interface{}{
		"result": result,
	})
}

// handleCollaborationRequest handles collaboration requests
func (a *AIAgent) handleCollaborationRequest(msg *A2AProtocol) {
	log.Printf("Agent %s received collaboration request from %s", a.ID, msg.SenderID)
	
	// Send collaboration response
	a.SendMessage("collaboration_response", msg.SenderID, msg.TaskID, map[string]interface{}{
		"available": true,
		"capabilities": a.Capabilities,
		"estimated_time": "5 minutes",
	})
}

// handleStatusUpdate handles status updates from other agents
func (a *AIAgent) handleStatusUpdate(msg *A2AProtocol) {
	log.Printf("Agent %s received status update from %s: %s", a.ID, msg.SenderID, msg.Payload["status"])
}

// taskProcessingLoop continuously processes tasks from the task queue
func (a *AIAgent) taskProcessingLoop() {
	sub := a.PubSub.Subscription("agent-task-queue-sub")
	
	for {
		err := sub.Receive(a.Context, func(ctx context.Context, msg *pubsub.Message) {
			var task AgentTask
			if err := json.Unmarshal(msg.Data, &task); err != nil {
				log.Printf("Failed to unmarshal task: %v", err)
				msg.Nack()
				return
			}

			// Process the task
			result, err := a.ProcessTask(&task)
			if err != nil {
				log.Printf("Failed to process task: %v", err)
				msg.Nack()
				return
			}

			// Publish result
			resultData, _ := json.Marshal(result)
			topic := a.PubSub.Topic("agent-results")
			topic.Publish(ctx, &pubsub.Message{
				Data: resultData,
				Attributes: map[string]string{
					"agent_id": a.ID,
					"task_id":  task.ID,
					"status":   result.Status,
				},
			})

			msg.Ack()
		})

		if err != nil {
			log.Printf("Error in task processing loop: %v", err)
			time.Sleep(5 * time.Second)
		}
	}
}

// StopAgent stops the agent
func (a *AIAgent) StopAgent() {
	a.IsActive = false
	log.Printf("Agent %s stopped", a.ID)
}
