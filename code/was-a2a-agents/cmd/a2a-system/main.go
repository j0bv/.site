// cmd/a2a-system/main.go
// Main entry point for the comprehensive A2A system

package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/WAS-Repository/was-a2a-agents/pkg/a2a"
	"github.com/WAS-Repository/was-a2a-agents/pkg/agents"
	"github.com/WAS-Repository/was-a2a-agents/pkg/mcp"
	"github.com/WAS-Repository/was-a2a-agents/pkg/mcp/tools"
	"github.com/WAS-Repository/was-a2a-agents/pkg/platforms"
)

func main() {
	log.Println("🚀 Starting WAS A2A System...")

	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		projectID = "axial-studio-470521-b9"
	}

	// Initialize MCP Server
	mcpServer := mcp.NewMCPServer()

	// Register MCP Tools
	mcpServer.RegisterTool(tools.NewNewsContextAnalyzer())
	mcpServer.RegisterTool(tools.NewEstatePropertyAnalyzer())
	mcpServer.RegisterTool(tools.NewGhidraBinaryAnalyzer(nil))
	mcpServer.RegisterTool(tools.NewGhidraRLAgent(nil))

	// Start MCP Server in background
	go func() {
		if err := mcpServer.Start("8081"); err != nil {
			log.Fatalf("Failed to start MCP server: %v", err)
		}
	}()

	// Initialize Agent Manager
	agentManager := a2a.NewAgentManager(projectID)

	// Initialize Platform Agent Factory
	platformFactory := platforms.NewPlatformAgentFactory(projectID, mcpServer)

	// Register Agent Factories for all 13 platforms
	platforms := []string{
		"news", "estate", "eco", "tools", "holdings", "market",
		"institute", "education", "network", "ngo", "world",
		"foundation", "codes",
	}

	for _, platform := range platforms {
		agentManager.RegisterAgentFactory(&PlatformAgentFactory{
			projectID:       projectID,
			mcpServer:       mcpServer,
			platformFactory: platformFactory,
		})
	}

	// Create Tool Analysis Agent
	toolAnalysisAgent, err := agents.NewToolAnalysisAgent(projectID, mcpServer)
	if err != nil {
		log.Fatalf("Failed to create tool analysis agent: %v", err)
	}
	agentManager.RegisterAgentFactory(&ToolAnalysisAgentFactory{agent: toolAnalysisAgent})

	// Create initial agents for all platforms
	createInitialAgents(agentManager, platforms)

	// Start auto-scaling
	ctx, cancel := context.WithCancel(context.Background())
	go agentManager.ScaleAgents(ctx)

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	log.Println("✅ WAS A2A System started successfully")
	log.Println("📊 MCP Server running on port 8081")
	log.Println("🤖 Agent Manager initialized")
	log.Println("🔄 Auto-scaling enabled")

	<-sigChan
	log.Println("🛑 Shutting down WAS A2A System...")

	cancel()
	mcpServer.Stop()

	log.Println("✅ Shutdown complete")
}

func createInitialAgents(manager *a2a.AgentManager, platforms []string) {
	// Create agents for each platform
	for _, platform := range platforms {
		// Create orchestrator agent for each platform
		manager.CreateAgent(platform, platform+"-orchestrator-1", platform+" Orchestrator 1")

		// Create specialized agents based on platform
		switch platform {
		case "news":
			manager.CreateAgent(platform, "news-fact-checker-1", "News Fact Checker 1")
			manager.CreateAgent(platform, "news-translator-1", "News Translator 1")
		case "estate":
			manager.CreateAgent(platform, "estate-analyzer-1", "Estate Analyzer 1")
			manager.CreateAgent(platform, "estate-compliance-1", "Estate Compliance 1")
		case "tools":
			manager.CreateAgent(platform, "tools-analyzer-1", "Tools Analyzer 1")
			manager.CreateAgent(platform, "tools-generator-1", "Tools Generator 1")
		case "eco":
			manager.CreateAgent(platform, "eco-monitor-1", "Eco Monitor 1")
			manager.CreateAgent(platform, "eco-analyzer-1", "Eco Analyzer 1")
		case "education":
			manager.CreateAgent(platform, "education-curriculum-1", "Education Curriculum 1")
			manager.CreateAgent(platform, "education-assessment-1", "Education Assessment 1")
		case "network":
			manager.CreateAgent(platform, "network-allocator-1", "Network Allocator 1")
			manager.CreateAgent(platform, "network-monitor-1", "Network Monitor 1")
		default:
			// Create generic worker for other platforms
			manager.CreateAgent(platform, platform+"-worker-1", platform+" Worker 1")
		}
	}

	// Create tool analysis agent
	manager.CreateAgent("analysis", "tool-analysis-1", "Tool Analysis Agent 1")

	log.Println("✅ Initial agents created for all 13 platforms")
}

// PlatformAgentFactory creates platform-specific agents
type PlatformAgentFactory struct {
	projectID       string
	mcpServer       *mcp.MCPServer
	platformFactory *platforms.PlatformAgentFactory
}

func (f *PlatformAgentFactory) GetAgentType() string {
	return "platform"
}

func (f *PlatformAgentFactory) CreateAgent(id, name string) (*a2a.Agent, error) {
	// Extract platform from agent ID
	platform := extractPlatformFromID(id)
	return f.platformFactory.CreateAgent(platform, id, name)
}

func (f *PlatformAgentFactory) GetResourceRequirements() a2a.ResourceRequirements {
	return a2a.ResourceRequirements{
		CPUMillis:    1000,
		MemoryMB:     1024,
		StorageGB:    2,
		NetworkMBps:  200,
		GPURequired:  false,
		SpecialTools: []string{"platform_specific", "mcp_integration"},
	}
}

// ToolAnalysisAgentFactory creates tool analysis agents
type ToolAnalysisAgentFactory struct {
	agent *agents.ToolAnalysisAgent
}

func (f *ToolAnalysisAgentFactory) GetAgentType() string {
	return "analysis"
}

func (f *ToolAnalysisAgentFactory) CreateAgent(id, name string) (*a2a.Agent, error) {
	return f.agent.Agent, nil
}

func (f *ToolAnalysisAgentFactory) GetResourceRequirements() a2a.ResourceRequirements {
	return a2a.ResourceRequirements{
		CPUMillis:    2000,
		MemoryMB:     2048,
		StorageGB:    4,
		NetworkMBps:  500,
		GPURequired:  true,
		SpecialTools: []string{"ghidra", "rl_engine", "tool_analysis"},
	}
}

// Helper function to extract platform from agent ID
func extractPlatformFromID(id string) string {
	// Simple extraction - assumes format "platform-type-1"
	parts := strings.Split(id, "-")
	if len(parts) > 0 {
		return parts[0]
	}
	return "unknown"
}
