// pkg/mcp/server.go
// Model Context Protocol (MCP) Server Implementation for WAS Agents

package mcp

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"google.golang.org/grpc"
)

// MCPServer implements Model Context Protocol for WAS agents
type MCPServer struct {
	contexts    map[string]*ContextStore
	subscribers map[string][]ContextSubscriber
	mu          sync.RWMutex
	grpcServer  *grpc.Server
	tools       map[string]MCPTool
	resources   map[string]MCPResource
}

// ContextStore holds context data for a specific domain/platform
type ContextStore struct {
	ID          string                 `json:"id"`
	Platform    string                 `json:"platform"`
	Agent       string                 `json:"agent"`
	Timestamp   time.Time              `json:"timestamp"`
	Data        map[string]interface{} `json:"data"`
	Schema      *ContextSchema         `json:"schema"`
	TTL         time.Duration          `json:"ttl"`
	AccessLevel AccessLevel            `json:"access_level"`
}

// ContextSchema defines the structure of context data
type ContextSchema struct {
	Version  string           `json:"version"`
	Fields   map[string]Field `json:"fields"`
	Required []string         `json:"required"`
	Optional []string         `json:"optional"`
}

// Field defines a context field specification
type Field struct {
	Type        string      `json:"type"`
	Description string      `json:"description"`
	Validation  []string    `json:"validation"`
	Default     interface{} `json:"default"`
}

// AccessLevel defines who can access context data
type AccessLevel int

const (
	Public AccessLevel = iota
	Platform
	Agent
	Private
)

// ContextSubscriber receives context updates
type ContextSubscriber interface {
	OnContextUpdate(ctx context.Context, update *ContextUpdate) error
	GetSubscriptionID() string
}

// ContextUpdate represents a context change notification
type ContextUpdate struct {
	ContextID  string                 `json:"context_id"`
	UpdateType UpdateType             `json:"update_type"`
	Changes    map[string]interface{} `json:"changes"`
	Timestamp  time.Time              `json:"timestamp"`
	Source     string                 `json:"source"`
}

type UpdateType int

const (
	Create UpdateType = iota
	Update
	Delete
	Merge
)

// MCPTool represents a tool available through MCP
type MCPTool interface {
	Name() string
	Description() string
	Execute(ctx context.Context, params map[string]interface{}) (*ToolResult, error)
	GetSchema() *ToolSchema
}

// MCPResource represents a resource accessible through MCP
type MCPResource interface {
	URI() string
	MimeType() string
	Read(ctx context.Context) ([]byte, error)
	Write(ctx context.Context, data []byte) error
}

// ToolResult represents the result of a tool execution
type ToolResult struct {
	Success  bool                   `json:"success"`
	Data     interface{}            `json:"data"`
	Error    string                 `json:"error,omitempty"`
	Metadata map[string]interface{} `json:"metadata"`
}

// ToolSchema defines the schema for a tool
type ToolSchema struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]Parameter   `json:"parameters"`
	Returns     map[string]interface{} `json:"returns"`
}

// Parameter defines a tool parameter
type Parameter struct {
	Type        string      `json:"type"`
	Description string      `json:"description"`
	Required    bool        `json:"required"`
	Default     interface{} `json:"default"`
}

// NewMCPServer creates a new MCP server
func NewMCPServer() *MCPServer {
	return &MCPServer{
		contexts:    make(map[string]*ContextStore),
		subscribers: make(map[string][]ContextSubscriber),
		tools:       make(map[string]MCPTool),
		resources:   make(map[string]MCPResource),
	}
}

// Start starts the MCP server
func (s *MCPServer) Start(port string) error {
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	s.grpcServer = grpc.NewServer()

	// Register MCP services here
	// RegisterMCPService(s.grpcServer, s)

	log.Printf("MCP Server starting on port %s", port)
	return s.grpcServer.Serve(lis)
}

// Stop stops the MCP server
func (s *MCPServer) Stop() {
	if s.grpcServer != nil {
		s.grpcServer.Stop()
	}
}

// SetContext sets context data for a specific ID
func (s *MCPServer) SetContext(ctx context.Context, contextID string, data *ContextStore) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.contexts[contextID] = data

	// Notify subscribers
	update := &ContextUpdate{
		ContextID:  contextID,
		UpdateType: Create,
		Changes:    data.Data,
		Timestamp:  time.Now(),
		Source:     "mcp_server",
	}

	s.notifySubscribers(ctx, update)
	return nil
}

// GetContext retrieves context data by ID
func (s *MCPServer) GetContext(ctx context.Context, contextID string) (*ContextStore, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if context, exists := s.contexts[contextID]; exists {
		return context, nil
	}

	return nil, fmt.Errorf("context not found: %s", contextID)
}

// UpdateContext updates existing context data
func (s *MCPServer) UpdateContext(ctx context.Context, contextID string, changes map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	context, exists := s.contexts[contextID]
	if !exists {
		return fmt.Errorf("context not found: %s", contextID)
	}

	// Merge changes
	for key, value := range changes {
		context.Data[key] = value
	}
	context.Timestamp = time.Now()

	// Notify subscribers
	update := &ContextUpdate{
		ContextID:  contextID,
		UpdateType: Update,
		Changes:    changes,
		Timestamp:  time.Now(),
		Source:     "mcp_server",
	}

	s.notifySubscribers(ctx, update)
	return nil
}

// RegisterTool registers a tool with the MCP server
func (s *MCPServer) RegisterTool(tool MCPTool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.tools[tool.Name()] = tool
	log.Printf("Registered MCP tool: %s", tool.Name())
}

// ExecuteTool executes a tool through MCP
func (s *MCPServer) ExecuteTool(ctx context.Context, toolName string, params map[string]interface{}) (*ToolResult, error) {
	s.mu.RLock()
	tool, exists := s.tools[toolName]
	s.mu.RUnlock()

	if !exists {
		return &ToolResult{
			Success: false,
			Error:   fmt.Sprintf("tool not found: %s", toolName),
		}, nil
	}

	return tool.Execute(ctx, params)
}

// SubscribeToContext subscribes to context updates
func (s *MCPServer) SubscribeToContext(ctx context.Context, contextID string, subscriber ContextSubscriber) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.subscribers[contextID] = append(s.subscribers[contextID], subscriber)
	log.Printf("Subscribed %s to context %s", subscriber.GetSubscriptionID(), contextID)
}

// notifySubscribers notifies all subscribers of a context update
func (s *MCPServer) notifySubscribers(ctx context.Context, update *ContextUpdate) {
	subscribers, exists := s.subscribers[update.ContextID]
	if !exists {
		return
	}

	for _, subscriber := range subscribers {
		go func(sub ContextSubscriber) {
			if err := sub.OnContextUpdate(ctx, update); err != nil {
				log.Printf("Failed to notify subscriber %s: %v", sub.GetSubscriptionID(), err)
			}
		}(subscriber)
	}
}

// GetAvailableTools returns all available tools
func (s *MCPServer) GetAvailableTools() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var tools []string
	for name := range s.tools {
		tools = append(tools, name)
	}
	return tools
}

// GetContextsByPlatform returns all contexts for a specific platform
func (s *MCPServer) GetContextsByPlatform(platform string) []*ContextStore {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var contexts []*ContextStore
	for _, context := range s.contexts {
		if context.Platform == platform {
			contexts = append(contexts, context)
		}
	}
	return contexts
}
