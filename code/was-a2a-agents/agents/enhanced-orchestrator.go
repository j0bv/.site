// agents/enhanced-orchestrator.go
// Enhanced orchestrator with Gemini 2.5 Pro for complex project management

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"was-cloud-dashboard/internal/cloud"
)

// EnhancedAgentOrchestrator manages complex project decomposition and multi-agent coordination
type EnhancedAgentOrchestrator struct {
	config       *EnhancedAgentConfig
	cloudService *cloud.WASCloudService
	github       *GitHubIntegration
	aiAgents     map[string]*AIAgent
	a2aManager   *A2ACommunicationManager
	projectID    string
	server       *http.Server
	workflows    map[string]*Workflow
	agentStatus  map[string]map[string]interface{}
	mu           sync.RWMutex

	// Enhanced project management capabilities
	projectQueue       chan *ProjectRequest
	activeProjects     map[string]*ProjectStatus
	resourceManager    *ResourceManager
	performanceMonitor *PerformanceMonitor

	// Gemini 2.5 Pro configuration
	orchestratorModel string
	contextWindow     int
	maxTokens         int
}

// EnhancedAgentConfig with Gemini 2.5 Pro optimization
type EnhancedAgentConfig struct {
	Name       string `yaml:"name"`
	Version    string `yaml:"version"`
	Mode       string `yaml:"mode"`
	RepoPath   string `yaml:"repository.path"`
	MainBranch string `yaml:"repository.main_branch"`
	AutoCommit bool   `yaml:"repository.auto_commit"`
	AutoPush   bool   `yaml:"repository.auto_push"`

	// Gemini 2.5 Pro Configuration
	OrchestratorModel string `yaml:"orchestrator_model"`
	ContextWindow     int    `yaml:"context_window"`
	MaxTokens         int    `yaml:"max_tokens"`
	OptimizationFocus string `yaml:"optimization_focus"`

	// Project Management Settings
	MaxConcurrentProjects int                       `yaml:"max_concurrent_projects"`
	ProjectTimeout        int                       `yaml:"project_timeout_minutes"`
	ResourceAllocation    map[string]ResourceConfig `yaml:"resource_allocation"`
}

// ResourceConfig defines resource allocation for different agent types
type ResourceConfig struct {
	CPU          string `yaml:"cpu"`
	Memory       string `yaml:"memory"`
	Priority     string `yaml:"priority"`
	MinInstances int    `yaml:"min_instances"`
	MaxInstances int    `yaml:"max_instances"`
}

// ProjectRequest represents a complex project that needs orchestration
type ProjectRequest struct {
	ID              string                 `json:"id"`
	Title           string                 `json:"title"`
	Description     string                 `json:"description"`
	Priority        string                 `json:"priority"`
	Deadline        time.Time              `json:"deadline"`
	Requirements    map[string]interface{} `json:"requirements"`
	TechnologyStack []string               `json:"technology_stack"`
	Complexity      string                 `json:"complexity"`
	Source          string                 `json:"source"` // github, tui, webui, api
	CreatedAt       time.Time              `json:"created_at"`
}

// ProjectStatus tracks the progress of a project
type ProjectStatus struct {
	Project             *ProjectRequest `json:"project"`
	Status              string          `json:"status"` // planning, executing, testing, deploying, completed, failed
	Progress            float64         `json:"progress"`
	AssignedAgents      []string        `json:"assigned_agents"`
	Subtasks            []*SubTask      `json:"subtasks"`
	Milestones          []*Milestone    `json:"milestones"`
	Issues              []*ProjectIssue `json:"issues"`
	StartTime           time.Time       `json:"start_time"`
	LastUpdate          time.Time       `json:"last_update"`
	EstimatedCompletion time.Time       `json:"estimated_completion"`
}

// SubTask represents a decomposed task within a project
type SubTask struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	AgentType   string     `json:"agent_type"`
	Model       string     `json:"model"`
	Priority    string     `json:"priority"`
	Status      string     `json:"status"`
	AssignedTo  string     `json:"assigned_to"`
	CreatedAt   time.Time  `json:"created_at"`
	DueDate     time.Time  `json:"due_date"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// Milestone represents a significant checkpoint in project completion
type Milestone struct {
	ID           string     `json:"id"`
	Title        string     `json:"title"`
	Description  string     `json:"description"`
	Status       string     `json:"status"`
	DueDate      time.Time  `json:"due_date"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	Dependencies []string   `json:"dependencies"`
}

// ProjectIssue represents problems or blockers in project execution
type ProjectIssue struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Severity    string     `json:"severity"` // low, medium, high, critical
	Status      string     `json:"status"`   // open, investigating, resolved
	AssignedTo  string     `json:"assigned_to"`
	CreatedAt   time.Time  `json:"created_at"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
}

// ResourceManager handles resource allocation and optimization
type ResourceManager struct {
	availableResources map[string]ResourceConfig
	allocatedResources map[string]string
	mu                 sync.RWMutex
}

// PerformanceMonitor tracks agent and project performance metrics
type PerformanceMonitor struct {
	agentMetrics   map[string]*AgentMetrics
	projectMetrics map[string]*ProjectMetrics
	systemMetrics  *SystemMetrics
	mu             sync.RWMutex
}

// AgentMetrics tracks performance of individual agents
type AgentMetrics struct {
	AgentID         string    `json:"agent_id"`
	TasksCompleted  int       `json:"tasks_completed"`
	AverageTaskTime float64   `json:"average_task_time_seconds"`
	SuccessRate     float64   `json:"success_rate"`
	ResourceUsage   float64   `json:"resource_usage_percent"`
	LastActivity    time.Time `json:"last_activity"`
	ErrorCount      int       `json:"error_count"`
	QualityScore    float64   `json:"quality_score"`
}

// ProjectMetrics tracks project completion performance
type ProjectMetrics struct {
	ProjectID          string             `json:"project_id"`
	CompletionTime     float64            `json:"completion_time_hours"`
	QualityScore       float64            `json:"quality_score"`
	ResourceEfficiency float64            `json:"resource_efficiency"`
	AgentUtilization   map[string]float64 `json:"agent_utilization"`
	IssueResolution    float64            `json:"issue_resolution_rate"`
}

// SystemMetrics tracks overall system performance
type SystemMetrics struct {
	TotalProjects       int       `json:"total_projects"`
	ActiveProjects      int       `json:"active_projects"`
	CompletedProjects   int       `json:"completed_projects"`
	AverageProjectTime  float64   `json:"average_project_time_hours"`
	SystemUptime        float64   `json:"system_uptime_hours"`
	ResourceUtilization float64   `json:"resource_utilization_percent"`
	LastUpdated         time.Time `json:"last_updated"`
}

// NewEnhancedAgentOrchestrator creates a new orchestrator with Gemini 2.5 Pro
func NewEnhancedAgentOrchestrator(projectID string) (*EnhancedAgentOrchestrator, error) {
	config := &EnhancedAgentConfig{
		Name:       "WAS Enhanced Orchestrator",
		Version:    "2.0.0",
		Mode:       "production",
		RepoPath:   "/workspace",
		MainBranch: "main",
		AutoCommit: true,
		AutoPush:   true,

		// Gemini 2.5 Pro Configuration
		OrchestratorModel: "gemini-2.5-pro",
		ContextWindow:     128000, // 128K context window
		MaxTokens:         8192,
		OptimizationFocus: "complex_reasoning",

		// Project Management Settings
		MaxConcurrentProjects: 10,
		ProjectTimeout:        60, // 60 minutes
		ResourceAllocation: map[string]ResourceConfig{
			"orchestrator": {
				CPU:          "4 vCPU",
				Memory:       "16 GB",
				Priority:     "highest",
				MinInstances: 1,
				MaxInstances: 10,
			},
			"generic_worker": {
				CPU:          "2-8 vCPU",
				Memory:       "8-32 GB",
				Priority:     "high",
				MinInstances: 2,
				MaxInstances: 50,
			},
			"tui_agent": {
				CPU:          "1 vCPU",
				Memory:       "4 GB",
				Priority:     "low_latency",
				MinInstances: 1,
				MaxInstances: 5,
			},
			"webui_agent": {
				CPU:          "2 vCPU",
				Memory:       "8 GB",
				Priority:     "medium",
				MinInstances: 1,
				MaxInstances: 20,
			},
			"monitoring_agent": {
				CPU:          "2 vCPU",
				Memory:       "8 GB",
				Priority:     "background",
				MinInstances: 1,
				MaxInstances: 3,
			},
		},
	}

	// Initialize cloud service
	cloudService := &cloud.WASCloudService{}

	// Initialize GitHub integration (with fallback)
	ctx := context.Background()
	github, err := NewGitHubIntegration(ctx, projectID)
	if err != nil {
		log.Printf("⚠️ GitHub integration failed, continuing without it: %v", err)
		// Continue without GitHub integration for now
		github = nil
	}

	// Initialize A2A communication manager (with fallback)
	a2aManager, err := NewA2ACommunicationManager(projectID)
	if err != nil {
		log.Printf("⚠️ A2A communication manager failed, continuing without it: %v", err)
		// Continue without A2A manager for now
		a2aManager = nil
	}

	orchestrator := &EnhancedAgentOrchestrator{
		config:             config,
		cloudService:       cloudService,
		github:             github,
		aiAgents:           make(map[string]*AIAgent),
		a2aManager:         a2aManager,
		projectID:          projectID,
		workflows:          make(map[string]*Workflow),
		agentStatus:        make(map[string]map[string]interface{}),
		projectQueue:       make(chan *ProjectRequest, 100),
		activeProjects:     make(map[string]*ProjectStatus),
		resourceManager:    NewResourceManager(config.ResourceAllocation),
		performanceMonitor: NewPerformanceMonitor(),
		orchestratorModel:  config.OrchestratorModel,
		contextWindow:      config.ContextWindow,
		maxTokens:          config.MaxTokens,
	}

	// Initialize AI agents with optimized models
	if err := orchestrator.initializeOptimizedAIAgents(); err != nil {
		return nil, fmt.Errorf("failed to initialize AI agents: %w", err)
	}

	return orchestrator, nil
}

// initializeOptimizedAIAgents sets up agents with recommended model configurations
func (ao *EnhancedAgentOrchestrator) initializeOptimizedAIAgents() error {
	// Code Improvement Agent - Gemini 1.5 Pro-002 for superior code understanding
	codeImprovementAgent := &AIAgent{
		ID:          "code-improvement-ai",
		Role:        "Code Enhancement",
		Description: "AI-powered code improvement and optimization using Gemini 1.5 Pro-002",
		Model:       "gemini-1.5-pro-002",
		Capabilities: []string{
			"code_analysis",
			"performance_optimization",
			"refactoring",
			"code_review",
			"architecture_improvement",
			"security_analysis",
		},
		IsActive: true,
	}

	// Testing Agent - Gemini 1.5 Pro for balanced testing capabilities
	testingAgent := &AIAgent{
		ID:          "testing-ai",
		Role:        "Quality Assurance",
		Description: "Comprehensive testing automation using Gemini 1.5 Pro",
		Model:       "gemini-1.5-pro",
		Capabilities: []string{
			"unit_test_generation",
			"integration_testing",
			"e2e_testing",
			"performance_testing",
			"security_testing",
			"test_optimization",
		},
		IsActive: true,
	}

	// Documentation Agent - Claude 3.5 Sonnet for superior technical writing
	documentationAgent := &AIAgent{
		ID:          "documentation-ai",
		Role:        "Technical Writing",
		Description: "High-quality documentation generation using Claude 3.5 Sonnet",
		Model:       "claude-3.5-sonnet",
		Capabilities: []string{
			"technical_writing",
			"api_documentation",
			"user_guides",
			"readme_generation",
			"architecture_docs",
			"troubleshooting_guides",
		},
		IsActive: true,
	}

	// Deployment Agent - Gemini 1.5 Pro for deployment orchestration
	deploymentAgent := &AIAgent{
		ID:          "deployment-ai",
		Role:        "Deployment Automation",
		Description: "Automated deployment and infrastructure management using Gemini 1.5 Pro",
		Model:       "gemini-1.5-pro",
		Capabilities: []string{
			"deployment_planning",
			"infrastructure_management",
			"rollback_strategies",
			"monitoring_setup",
			"scaling_optimization",
			"security_hardening",
		},
		IsActive: true,
	}

	// Monitoring Agent - Gemini 1.5 Pro for comprehensive monitoring
	monitoringAgent := &AIAgent{
		ID:          "monitoring-ai",
		Role:        "System Monitoring",
		Description: "Intelligent monitoring and performance optimization using Gemini 1.5 Pro",
		Model:       "gemini-1.5-pro",
		Capabilities: []string{
			"resource_discovery",
			"metric_analysis",
			"anomaly_detection",
			"performance_optimization",
			"alert_management",
			"capacity_planning",
		},
		IsActive: true,
	}

	// Register all agents
	ao.aiAgents["code-improvement-ai"] = codeImprovementAgent
	ao.aiAgents["testing-ai"] = testingAgent
	ao.aiAgents["documentation-ai"] = documentationAgent
	ao.aiAgents["deployment-ai"] = deploymentAgent
	ao.aiAgents["monitoring-ai"] = monitoringAgent

	// Initialize agent status
	for _, agent := range ao.aiAgents {
		ao.agentStatus[agent.ID] = map[string]interface{}{
			"agent_id":        agent.ID,
			"role":            agent.Role,
			"status":          "active",
			"last_seen":       time.Now(),
			"tasks_active":    0,
			"tasks_completed": 0,
		}
	}

	return nil
}

// ProcessProjectRequest handles complex project decomposition and delegation
func (ao *EnhancedAgentOrchestrator) ProcessProjectRequest(req *ProjectRequest) error {
	ao.mu.Lock()
	defer ao.mu.Unlock()

	// Add to project queue
	select {
	case ao.projectQueue <- req:
		// Project queued successfully
	default:
		return fmt.Errorf("project queue is full, cannot process request")
	}

	// Create project status
	projectStatus := &ProjectStatus{
		Project:    req,
		Status:     "planning",
		Progress:   0.0,
		StartTime:  time.Now(),
		LastUpdate: time.Now(),
	}

	ao.activeProjects[req.ID] = projectStatus

	// Start project processing in background
	go ao.processProject(req)

	return nil
}

// processProject handles the complete project lifecycle
func (ao *EnhancedAgentOrchestrator) processProject(req *ProjectRequest) {
	log.Printf("Starting project processing: %s", req.Title)

	// Phase 1: Project Analysis and Decomposition using Gemini 2.5 Pro
	subtasks, err := ao.decomposeProject(req)
	if err != nil {
		log.Printf("Failed to decompose project %s: %v", req.ID, err)
		ao.updateProjectStatus(req.ID, "failed", 0.0)
		return
	}

	// Phase 2: Resource Allocation
	if err := ao.allocateResources(req, subtasks); err != nil {
		log.Printf("Failed to allocate resources for project %s: %v", req.ID, err)
		ao.updateProjectStatus(req.ID, "failed", 0.0)
		return
	}

	// Phase 3: Task Execution
	ao.updateProjectStatus(req.ID, "executing", 10.0)
	if err := ao.executeProjectTasks(req, subtasks); err != nil {
		log.Printf("Failed to execute project %s: %v", req.ID, err)
		ao.updateProjectStatus(req.ID, "failed", 0.0)
		return
	}

	// Phase 4: Testing and Validation
	ao.updateProjectStatus(req.ID, "testing", 80.0)
	if err := ao.validateProject(req); err != nil {
		log.Printf("Project validation failed for %s: %v", req.ID, err)
		ao.updateProjectStatus(req.ID, "failed", 0.0)
		return
	}

	// Phase 5: Deployment
	ao.updateProjectStatus(req.ID, "deploying", 90.0)
	if err := ao.deployProject(req); err != nil {
		log.Printf("Project deployment failed for %s: %v", req.ID, err)
		ao.updateProjectStatus(req.ID, "failed", 0.0)
		return
	}

	// Phase 6: Completion
	ao.updateProjectStatus(req.ID, "completed", 100.0)
	log.Printf("Project completed successfully: %s", req.Title)
}

// decomposeProject uses Gemini 2.5 Pro to break down complex projects
func (ao *EnhancedAgentOrchestrator) decomposeProject(req *ProjectRequest) ([]*SubTask, error) {
	// This would integrate with Vertex AI Gemini 2.5 Pro API
	// For now, return a placeholder implementation

	subtasks := []*SubTask{
		{
			ID:          fmt.Sprintf("%s-analysis", req.ID),
			Title:       "Project Analysis",
			Description: "Analyze project requirements and technology stack",
			AgentType:   "generic_worker",
			Model:       "gemini-1.5-pro",
			Priority:    "high",
			Status:      "pending",
			CreatedAt:   time.Now(),
			DueDate:     time.Now().Add(30 * time.Minute),
		},
		{
			ID:          fmt.Sprintf("%s-development", req.ID),
			Title:       "Code Development",
			Description: "Implement core functionality",
			AgentType:   "code-improvement-ai",
			Model:       "gemini-1.5-pro-002",
			Priority:    "high",
			Status:      "pending",
			CreatedAt:   time.Now(),
			DueDate:     time.Now().Add(2 * time.Hour),
		},
		{
			ID:          fmt.Sprintf("%s-testing", req.ID),
			Title:       "Testing and Validation",
			Description: "Comprehensive testing of implemented features",
			AgentType:   "testing-ai",
			Model:       "gemini-1.5-pro",
			Priority:    "high",
			Status:      "pending",
			CreatedAt:   time.Now(),
			DueDate:     time.Now().Add(1 * time.Hour),
		},
		{
			ID:          fmt.Sprintf("%s-documentation", req.ID),
			Title:       "Documentation",
			Description: "Generate comprehensive documentation",
			AgentType:   "documentation-ai",
			Model:       "claude-3.5-sonnet",
			Priority:    "medium",
			Status:      "pending",
			CreatedAt:   time.Now(),
			DueDate:     time.Now().Add(45 * time.Minute),
		},
		{
			ID:          fmt.Sprintf("%s-deployment", req.ID),
			Title:       "Deployment",
			Description: "Deploy to production environment",
			AgentType:   "deployment-ai",
			Model:       "gemini-1.5-pro",
			Priority:    "high",
			Status:      "pending",
			CreatedAt:   time.Now(),
			DueDate:     time.Now().Add(30 * time.Minute),
		},
	}

	return subtasks, nil
}

// allocateResources assigns optimal resources to project tasks
func (ao *EnhancedAgentOrchestrator) allocateResources(req *ProjectRequest, subtasks []*SubTask) error {
	// Use resource manager to allocate optimal resources
	for _, subtask := range subtasks {
		resourceConfig, exists := ao.config.ResourceAllocation[subtask.AgentType]
		if !exists {
			return fmt.Errorf("no resource configuration for agent type: %s", subtask.AgentType)
		}

		// Allocate resources based on task priority and complexity
		if err := ao.resourceManager.AllocateResource(subtask.ID, resourceConfig); err != nil {
			return fmt.Errorf("failed to allocate resources for subtask %s: %w", subtask.ID, err)
		}
	}

	return nil
}

// executeProjectTasks coordinates the execution of all project subtasks
func (ao *EnhancedAgentOrchestrator) executeProjectTasks(req *ProjectRequest, subtasks []*SubTask) error {
	// Execute tasks in parallel with proper coordination
	for _, subtask := range subtasks {
		go ao.executeSubTask(req.ID, subtask)
	}

	// Wait for all tasks to complete
	// This would implement proper task coordination and dependency management
	time.Sleep(5 * time.Second) // Placeholder

	return nil
}

// executeSubTask executes an individual subtask using the appropriate agent
func (ao *EnhancedAgentOrchestrator) executeSubTask(projectID string, subtask *SubTask) {
	log.Printf("Executing subtask: %s", subtask.Title)

	// Update subtask status
	subtask.Status = "executing"

	// This would integrate with the actual agent execution
	// For now, simulate task execution
	time.Sleep(2 * time.Second)

	// Mark as completed
	now := time.Now()
	subtask.Status = "completed"
	subtask.CompletedAt = &now

	log.Printf("Completed subtask: %s", subtask.Title)
}

// validateProject performs comprehensive project validation
func (ao *EnhancedAgentOrchestrator) validateProject(req *ProjectRequest) error {
	// Use testing agent to validate project
	log.Printf("Validating project: %s", req.Title)

	// This would integrate with testing agent
	time.Sleep(2 * time.Second)

	return nil
}

// deployProject handles project deployment
func (ao *EnhancedAgentOrchestrator) deployProject(req *ProjectRequest) error {
	// Use deployment agent to deploy project
	log.Printf("Deploying project: %s", req.Title)

	// This would integrate with deployment agent
	time.Sleep(2 * time.Second)

	return nil
}

// updateProjectStatus updates the status of a project
func (ao *EnhancedAgentOrchestrator) updateProjectStatus(projectID, status string, progress float64) {
	ao.mu.Lock()
	defer ao.mu.Unlock()

	if project, exists := ao.activeProjects[projectID]; exists {
		project.Status = status
		project.Progress = progress
		project.LastUpdate = time.Now()
	}
}

// NewResourceManager creates a new resource manager
func NewResourceManager(resourceConfigs map[string]ResourceConfig) *ResourceManager {
	return &ResourceManager{
		availableResources: resourceConfigs,
		allocatedResources: make(map[string]string),
	}
}

// AllocateResource allocates resources for a task
func (rm *ResourceManager) AllocateResource(taskID string, config ResourceConfig) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.allocatedResources[taskID] = fmt.Sprintf("%s-%s", config.CPU, config.Memory)
	return nil
}

// NewPerformanceMonitor creates a new performance monitor
func NewPerformanceMonitor() *PerformanceMonitor {
	return &PerformanceMonitor{
		agentMetrics:   make(map[string]*AgentMetrics),
		projectMetrics: make(map[string]*ProjectMetrics),
		systemMetrics: &SystemMetrics{
			LastUpdated: time.Now(),
		},
	}
}

// Start starts the enhanced orchestrator
func (ao *EnhancedAgentOrchestrator) Start() error {
	// Start project processing loop
	go ao.projectProcessingLoop()

	// Start performance monitoring
	go ao.performanceMonitoringLoop()

	// Start webhook server
	return ao.startWebhookServer()
}

// projectProcessingLoop continuously processes projects from the queue
func (ao *EnhancedAgentOrchestrator) projectProcessingLoop() {
	for {
		select {
		case project := <-ao.projectQueue:
			go ao.processProject(project)
		case <-time.After(1 * time.Minute):
			// Periodic cleanup and status updates
			ao.cleanupCompletedProjects()
		}
	}
}

// performanceMonitoringLoop continuously monitors system performance
func (ao *EnhancedAgentOrchestrator) performanceMonitoringLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ao.updateSystemMetrics()
		}
	}
}

// cleanupCompletedProjects removes completed projects from active tracking
func (ao *EnhancedAgentOrchestrator) cleanupCompletedProjects() {
	ao.mu.Lock()
	defer ao.mu.Unlock()

	for projectID, project := range ao.activeProjects {
		if project.Status == "completed" || project.Status == "failed" {
			if time.Since(project.LastUpdate) > 1*time.Hour {
				delete(ao.activeProjects, projectID)
			}
		}
	}
}

// updateSystemMetrics updates overall system performance metrics
func (ao *EnhancedAgentOrchestrator) updateSystemMetrics() {
	ao.mu.RLock()
	defer ao.mu.RUnlock()

	// Update system metrics
	ao.performanceMonitor.systemMetrics.TotalProjects = len(ao.activeProjects)
	ao.performanceMonitor.systemMetrics.ActiveProjects = 0
	ao.performanceMonitor.systemMetrics.CompletedProjects = 0

	for _, project := range ao.activeProjects {
		switch project.Status {
		case "executing", "testing", "deploying":
			ao.performanceMonitor.systemMetrics.ActiveProjects++
		case "completed":
			ao.performanceMonitor.systemMetrics.CompletedProjects++
		}
	}

	ao.performanceMonitor.systemMetrics.LastUpdated = time.Now()
}

// startWebhookServer starts the webhook server for GitHub integration
func (ao *EnhancedAgentOrchestrator) startWebhookServer() error {
	mux := http.NewServeMux()

	// Health endpoint
	mux.HandleFunc("/health", ao.handleHealth)

	// Project management endpoints
	mux.HandleFunc("/api/projects", ao.handleProjects)
	mux.HandleFunc("/api/projects/", ao.handleProject)
	mux.HandleFunc("/api/agents/status", ao.handleAgentStatus)

	// GitHub webhook
	mux.HandleFunc("/github-webhook", ao.handleGitHubWebhook)

	ao.server = &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	return ao.server.ListenAndServe()
}

// handleHealth provides system health information
func (ao *EnhancedAgentOrchestrator) handleHealth(w http.ResponseWriter, r *http.Request) {
	ao.mu.RLock()
	defer ao.mu.RUnlock()

	health := map[string]interface{}{
		"status":          "healthy",
		"orchestrator":    "gemini-2.5-pro",
		"agents":          len(ao.aiAgents),
		"active_projects": len(ao.activeProjects),
		"ai_agents":       ao.aiAgents,
		"system_metrics":  ao.performanceMonitor.systemMetrics,
		"timestamp":       time.Now().UTC(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// handleProjects handles project management endpoints
func (ao *EnhancedAgentOrchestrator) handleProjects(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		ao.listProjects(w, r)
	case "POST":
		ao.createProject(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleProject handles individual project endpoints
func (ao *EnhancedAgentOrchestrator) handleProject(w http.ResponseWriter, r *http.Request) {
	// Extract project ID from URL path
	path := strings.TrimPrefix(r.URL.Path, "/api/projects/")
	projectID := strings.Split(path, "/")[0]

	switch r.Method {
	case "GET":
		ao.getProject(w, r, projectID)
	case "PUT":
		ao.updateProject(w, r, projectID)
	case "DELETE":
		ao.deleteProject(w, r, projectID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// listProjects returns a list of all projects
func (ao *EnhancedAgentOrchestrator) listProjects(w http.ResponseWriter, r *http.Request) {
	ao.mu.RLock()
	defer ao.mu.RUnlock()

	projects := make([]*ProjectStatus, 0, len(ao.activeProjects))
	for _, project := range ao.activeProjects {
		projects = append(projects, project)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(projects)
}

// createProject creates a new project
func (ao *EnhancedAgentOrchestrator) createProject(w http.ResponseWriter, r *http.Request) {
	var req ProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	req.ID = fmt.Sprintf("project-%d", time.Now().Unix())
	req.CreatedAt = time.Now()

	if err := ao.ProcessProjectRequest(&req); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"id":      req.ID,
		"status":  "accepted",
		"message": "Project queued for processing",
	})
}

// getProject returns project details
func (ao *EnhancedAgentOrchestrator) getProject(w http.ResponseWriter, r *http.Request, projectID string) {
	ao.mu.RLock()
	defer ao.mu.RUnlock()

	project, exists := ao.activeProjects[projectID]
	if !exists {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(project)
}

// updateProject updates project details
func (ao *EnhancedAgentOrchestrator) updateProject(w http.ResponseWriter, r *http.Request, projectID string) {
	// Implementation for project updates
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}

// deleteProject deletes a project
func (ao *EnhancedAgentOrchestrator) deleteProject(w http.ResponseWriter, r *http.Request, projectID string) {
	ao.mu.Lock()
	defer ao.mu.Unlock()

	if _, exists := ao.activeProjects[projectID]; exists {
		delete(ao.activeProjects, projectID)
		w.WriteHeader(http.StatusNoContent)
	} else {
		http.Error(w, "Project not found", http.StatusNotFound)
	}
}

// handleAgentStatus returns agent status information
func (ao *EnhancedAgentOrchestrator) handleAgentStatus(w http.ResponseWriter, r *http.Request) {
	ao.mu.RLock()
	defer ao.mu.RUnlock()

	status := map[string]interface{}{
		"agents":    ao.agentStatus,
		"ai_agents": ao.aiAgents,
		"timestamp": time.Now().UTC(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// handleGitHubWebhook handles GitHub webhook events
func (ao *EnhancedAgentOrchestrator) handleGitHubWebhook(w http.ResponseWriter, r *http.Request) {
	// Implementation for GitHub webhook handling
	// This would process GitHub events and create project requests
	log.Printf("Received GitHub webhook")
	w.WriteHeader(http.StatusOK)
}

// GetStatus returns the current status of the enhanced orchestrator
func (ao *EnhancedAgentOrchestrator) GetStatus() map[string]interface{} {
	ao.mu.RLock()
	defer ao.mu.RUnlock()

	return map[string]interface{}{
		"agent_id":        "enhanced-orchestrator",
		"name":            "Enhanced Orchestrator",
		"role":            "Project Management",
		"status":          "active",
		"model":           ao.orchestratorModel,
		"context_window":  ao.contextWindow,
		"max_tokens":      ao.maxTokens,
		"active_projects": len(ao.activeProjects),
		"ai_agents":       len(ao.aiAgents),
		"agent_status":    ao.agentStatus,
		"last_activity":   time.Now(),
	}
}
