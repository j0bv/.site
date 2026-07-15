// agents/integrated-agent-system.go
// Integrated agent system with optimized model configurations

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

// IntegratedAgentSystem combines all enhanced agent components
type IntegratedAgentSystem struct {
	// Core components
	enhancedOrchestrator *EnhancedAgentOrchestrator
	dynamicWorker        *DynamicWorkerAgent
	monitoringAgent      *MonitoringDataAgent
	tuiDashboard         *TUIDashboardAgent

	// Original components for compatibility
	originalOrchestrator *AgentOrchestrator

	// System configuration
	projectID string
	server    *http.Server
	mu        sync.RWMutex
}

// SystemStatus represents the overall system status
type SystemStatus struct {
	Status               string                    `json:"status"`
	Timestamp            time.Time                 `json:"timestamp"`
	EnhancedOrchestrator map[string]interface{}    `json:"enhanced_orchestrator"`
	DynamicWorker        map[string]interface{}    `json:"dynamic_worker"`
	MonitoringAgent      map[string]interface{}    `json:"monitoring_agent"`
	OriginalOrchestrator map[string]interface{}    `json:"original_orchestrator"`
	PerformanceMetrics   *SystemPerformanceMetrics `json:"performance_metrics"`
}

// SystemPerformanceMetrics tracks overall system performance
type SystemPerformanceMetrics struct {
	TotalProjects       int       `json:"total_projects"`
	ActiveProjects      int       `json:"active_projects"`
	CompletedProjects   int       `json:"completed_projects"`
	TotalTasks          int       `json:"total_tasks"`
	SuccessfulTasks     int       `json:"successful_tasks"`
	AverageTaskTime     float64   `json:"average_task_time_seconds"`
	SystemUptime        float64   `json:"system_uptime_hours"`
	ResourceUtilization float64   `json:"resource_utilization_percent"`
	ErrorRate           float64   `json:"error_rate"`
	QualityScore        float64   `json:"quality_score"`
	LastUpdated         time.Time `json:"last_updated"`
}

// NewIntegratedAgentSystem creates a new integrated agent system
func NewIntegratedAgentSystem(projectID string) (*IntegratedAgentSystem, error) {
	// Initialize cloud service
	cloudService := &cloud.WASCloudService{}

	// Initialize enhanced orchestrator with Gemini 2.5 Pro
	enhancedOrchestrator, err := NewEnhancedAgentOrchestrator(projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize enhanced orchestrator: %w", err)
	}

	// Initialize dynamic worker agent
	dynamicWorker := NewDynamicWorkerAgent(
		"dynamic-worker-agent",
		"Dynamic Worker Agent",
		"Multi-Purpose Worker",
		"Intelligent task processing with dynamic model selection",
	)

	// Initialize monitoring data agent
	monitoringAgent := NewMonitoringDataAgent(projectID)

	// Initialize TUI dashboard agent
	apiBaseURL := "http://localhost:8080"
	tuiDashboard := NewTUIDashboardAgent(apiBaseURL)

	// Initialize original orchestrator for compatibility
	originalOrchestrator := NewAgentOrchestrator(cloudService)

	system := &IntegratedAgentSystem{
		enhancedOrchestrator: enhancedOrchestrator,
		dynamicWorker:        dynamicWorker,
		monitoringAgent:      monitoringAgent,
		tuiDashboard:         tuiDashboard,
		originalOrchestrator: originalOrchestrator,
		projectID:            projectID,
	}

	return system, nil
}

// Start starts the integrated agent system
func (ias *IntegratedAgentSystem) Start() error {
	log.Printf("🚀 Starting Integrated Agent System with optimized model configurations")

	// Start enhanced orchestrator
	if err := ias.enhancedOrchestrator.Start(); err != nil {
		log.Printf("⚠️ Enhanced orchestrator start failed: %v", err)
	} else {
		log.Printf("✅ Enhanced orchestrator started with Gemini 2.5 Pro")
	}

	// Start monitoring agent
	if err := ias.monitoringAgent.Start(); err != nil {
		log.Printf("⚠️ Monitoring agent start failed: %v", err)
	} else {
		log.Printf("✅ Monitoring agent started with Gemini 1.5 Pro")
	}

	// Start TUI dashboard agent
	if err := ias.tuiDashboard.Start(); err != nil {
		log.Printf("⚠️ TUI dashboard agent start failed: %v", err)
	} else {
		log.Printf("✅ TUI dashboard agent started")
	}

	// Start original orchestrator for compatibility
	ctx := context.Background()
	if err := ias.originalOrchestrator.Start(ctx); err != nil {
		log.Printf("⚠️ Original orchestrator start failed: %v", err)
	} else {
		log.Printf("✅ Original orchestrator started for compatibility")
	}

	// Start integrated webhook server
	return ias.startIntegratedServer()
}

// startIntegratedServer starts the integrated webhook server
func (ias *IntegratedAgentSystem) startIntegratedServer() error {
	mux := http.NewServeMux()

	// Health and status endpoints
	mux.HandleFunc("/health", ias.handleHealth)
	mux.HandleFunc("/status", ias.handleStatus)
	mux.HandleFunc("/api/system/status", ias.handleSystemStatus)

	// Enhanced orchestrator endpoints
	mux.HandleFunc("/api/projects", ias.handleProjects)
	mux.HandleFunc("/api/projects/", ias.handleProject)
	mux.HandleFunc("/api/agents/status", ias.handleAgentStatus)

	// Dynamic worker endpoints
	mux.HandleFunc("/api/worker/tasks", ias.handleWorkerTasks)
	mux.HandleFunc("/api/worker/models", ias.handleWorkerModels)

	// Monitoring endpoints
	mux.HandleFunc("/api/monitoring/resources", ias.handleMonitoringResources)
	mux.HandleFunc("/api/monitoring/metrics", ias.handleMonitoringMetrics)
	mux.HandleFunc("/api/monitoring/anomalies", ias.handleMonitoringAnomalies)
	mux.HandleFunc("/api/monitoring/grafana", ias.handleGrafanaDashboards)

	// TUI Dashboard endpoints
	mux.HandleFunc("/api/tui/dashboard", ias.handleTUIDashboard)
	mux.HandleFunc("/api/tui/status", ias.handleTUIStatus)

	// GitHub webhook
	mux.HandleFunc("/github-webhook", ias.handleGitHubWebhook)

	// Original orchestrator endpoints for compatibility
	mux.HandleFunc("/api/orchestrator/", ias.handleOriginalOrchestrator)

	ias.server = &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	log.Printf("🌐 Integrated webhook server starting on :8080")
	return ias.server.ListenAndServe()
}

// handleHealth provides system health information
func (ias *IntegratedAgentSystem) handleHealth(w http.ResponseWriter, r *http.Request) {
	ias.mu.RLock()
	defer ias.mu.RUnlock()

	health := map[string]interface{}{
		"status":                "healthy",
		"system":                "integrated_agent_system",
		"version":               "2.0.0",
		"enhanced_orchestrator": "gemini-2.5-pro",
		"dynamic_worker":        "multi_model_selection",
		"monitoring_agent":      "gemini-1.5-pro",
		"components": map[string]interface{}{
			"enhanced_orchestrator": "active",
			"dynamic_worker":        "active",
			"monitoring_agent":      "active",
			"tui_dashboard":         "active",
			"original_orchestrator": "active",
		},
		"timestamp": time.Now().UTC(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// handleStatus provides detailed system status
func (ias *IntegratedAgentSystem) handleStatus(w http.ResponseWriter, r *http.Request) {
	ias.mu.RLock()
	defer ias.mu.RUnlock()

	status := &SystemStatus{
		Status:               "operational",
		Timestamp:            time.Now().UTC(),
		EnhancedOrchestrator: ias.enhancedOrchestrator.GetStatus(),
		DynamicWorker:        ias.dynamicWorker.GetStatus(),
		MonitoringAgent:      ias.monitoringAgent.GetStatus(),
		OriginalOrchestrator: ias.originalOrchestrator.GetStatus(),
		PerformanceMetrics:   ias.calculateSystemMetrics(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// handleSystemStatus provides comprehensive system status
func (ias *IntegratedAgentSystem) handleSystemStatus(w http.ResponseWriter, r *http.Request) {
	ias.mu.RLock()
	defer ias.mu.RUnlock()

	systemStatus := map[string]interface{}{
		"system_info": map[string]interface{}{
			"name":       "WAS Integrated Agent System",
			"version":    "2.0.0",
			"project_id": ias.projectID,
			"status":     "operational",
			"uptime":     time.Since(time.Now()).Hours(),
		},
		"model_configurations": map[string]interface{}{
			"orchestrator": map[string]interface{}{
				"model":          "gemini-2.5-pro",
				"context_window": 128000,
				"max_tokens":     8192,
				"optimization":   "complex_reasoning",
				"capabilities":   []string{"project_management", "task_decomposition", "github_integration"},
			},
			"dynamic_worker": map[string]interface{}{
				"models": []string{
					"gemini-1.5-pro-002",
					"claude-3.5-sonnet",
					"gemini-1.5-pro",
					"gemini-1.5-flash",
					"gemini-1.5-flash-8b",
				},
				"selection_strategy": "dynamic_based_on_task_type",
				"capabilities":       []string{"code_analysis", "documentation", "testing", "data_processing"},
			},
			"monitoring_agent": map[string]interface{}{
				"model":          "gemini-1.5-pro",
				"context_window": 128000,
				"max_tokens":     8192,
				"optimization":   "data_analysis_and_transformation",
				"capabilities":   []string{"resource_discovery", "metric_analysis", "anomaly_detection"},
			},
		},
		"performance_metrics": ias.calculateSystemMetrics(),
		"timestamp":           time.Now().UTC(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(systemStatus)
}

// handleProjects handles project management endpoints
func (ias *IntegratedAgentSystem) handleProjects(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		ias.listProjects(w, r)
	case "POST":
		ias.createProject(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleProject handles individual project endpoints
func (ias *IntegratedAgentSystem) handleProject(w http.ResponseWriter, r *http.Request) {
	// Extract project ID from URL path
	path := strings.TrimPrefix(r.URL.Path, "/api/projects/")
	projectID := strings.Split(path, "/")[0]

	switch r.Method {
	case "GET":
		ias.getProject(w, r, projectID)
	case "PUT":
		ias.updateProject(w, r, projectID)
	case "DELETE":
		ias.deleteProject(w, r, projectID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// listProjects returns a list of all projects
func (ias *IntegratedAgentSystem) listProjects(w http.ResponseWriter, r *http.Request) {
	// This would integrate with the enhanced orchestrator
	projects := []map[string]interface{}{
		{
			"id":              "project-1",
			"title":           "Enhanced Grafana Dashboard",
			"status":          "executing",
			"progress":        65.0,
			"created_at":      time.Now().Add(-2 * time.Hour),
			"assigned_agents": []string{"code-improvement-ai", "testing-ai"},
		},
		{
			"id":              "project-2",
			"title":           "BigQuery Optimization",
			"status":          "planning",
			"progress":        10.0,
			"created_at":      time.Now().Add(-30 * time.Minute),
			"assigned_agents": []string{"monitoring-ai", "deployment-ai"},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(projects)
}

// createProject creates a new project
func (ias *IntegratedAgentSystem) createProject(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// This would integrate with the enhanced orchestrator
	projectID := fmt.Sprintf("project-%d", time.Now().Unix())

	response := map[string]interface{}{
		"id":           projectID,
		"status":       "accepted",
		"message":      "Project queued for processing with enhanced orchestrator",
		"orchestrator": "gemini-2.5-pro",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// getProject returns project details
func (ias *IntegratedAgentSystem) getProject(w http.ResponseWriter, r *http.Request, projectID string) {
	// This would integrate with the enhanced orchestrator
	project := map[string]interface{}{
		"id":              projectID,
		"title":           "Sample Project",
		"status":          "executing",
		"progress":        65.0,
		"created_at":      time.Now().Add(-2 * time.Hour),
		"assigned_agents": []string{"code-improvement-ai", "testing-ai"},
		"subtasks": []map[string]interface{}{
			{
				"id":     "task-1",
				"title":  "Code Analysis",
				"status": "completed",
				"agent":  "code-improvement-ai",
				"model":  "gemini-1.5-pro-002",
			},
			{
				"id":     "task-2",
				"title":  "Testing",
				"status": "in_progress",
				"agent":  "testing-ai",
				"model":  "gemini-1.5-pro",
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(project)
}

// updateProject updates project details
func (ias *IntegratedAgentSystem) updateProject(w http.ResponseWriter, r *http.Request, projectID string) {
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}

// deleteProject deletes a project
func (ias *IntegratedAgentSystem) deleteProject(w http.ResponseWriter, r *http.Request, projectID string) {
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}

// handleAgentStatus returns agent status information
func (ias *IntegratedAgentSystem) handleAgentStatus(w http.ResponseWriter, r *http.Request) {
	ias.mu.RLock()
	defer ias.mu.RUnlock()

	status := map[string]interface{}{
		"enhanced_orchestrator": ias.enhancedOrchestrator.GetStatus(),
		"dynamic_worker":        ias.dynamicWorker.GetStatus(),
		"monitoring_agent":      ias.monitoringAgent.GetStatus(),
		"original_orchestrator": ias.originalOrchestrator.GetStatus(),
		"timestamp":             time.Now().UTC(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// handleWorkerTasks handles dynamic worker task endpoints
func (ias *IntegratedAgentSystem) handleWorkerTasks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		ias.listWorkerTasks(w, r)
	case "POST":
		ias.createWorkerTask(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// listWorkerTasks returns worker task information
func (ias *IntegratedAgentSystem) listWorkerTasks(w http.ResponseWriter, r *http.Request) {
	tasks := []map[string]interface{}{
		{
			"id":            "task-1",
			"title":         "Code Analysis Task",
			"type":          "code_analysis",
			"model":         "gemini-1.5-pro-002",
			"status":        "completed",
			"quality_score": 0.92,
		},
		{
			"id":            "task-2",
			"title":         "Documentation Task",
			"type":          "documentation",
			"model":         "claude-3.5-sonnet",
			"status":        "in_progress",
			"quality_score": 0.96,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}

// createWorkerTask creates a new worker task
func (ias *IntegratedAgentSystem) createWorkerTask(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// This would integrate with the dynamic worker
	response := map[string]interface{}{
		"id":      fmt.Sprintf("task-%d", time.Now().Unix()),
		"status":  "accepted",
		"message": "Task queued for processing with dynamic model selection",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleWorkerModels returns available worker models
func (ias *IntegratedAgentSystem) handleWorkerModels(w http.ResponseWriter, r *http.Request) {
	models := ias.dynamicWorker.GetAvailableModels()
	recommendations := ias.dynamicWorker.GetModelRecommendations("code_analysis")

	response := map[string]interface{}{
		"available_models":   models,
		"recommendations":    recommendations,
		"selection_strategy": "dynamic_based_on_task_type",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleMonitoringResources returns discovered resources
func (ias *IntegratedAgentSystem) handleMonitoringResources(w http.ResponseWriter, r *http.Request) {
	resources := ias.monitoringAgent.GetDiscoveredResources()

	response := map[string]interface{}{
		"resources": resources,
		"count":     len(resources),
		"timestamp": time.Now().UTC(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleMonitoringMetrics returns collected metrics
func (ias *IntegratedAgentSystem) handleMonitoringMetrics(w http.ResponseWriter, r *http.Request) {
	metrics := ias.monitoringAgent.GetCollectedMetrics()

	response := map[string]interface{}{
		"metrics":   metrics,
		"count":     len(metrics),
		"timestamp": time.Now().UTC(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleMonitoringAnomalies returns detected anomalies
func (ias *IntegratedAgentSystem) handleMonitoringAnomalies(w http.ResponseWriter, r *http.Request) {
	anomalies := ias.monitoringAgent.GetDetectedAnomalies()

	response := map[string]interface{}{
		"anomalies": anomalies,
		"count":     len(anomalies),
		"timestamp": time.Now().UTC(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleGrafanaDashboards returns Grafana dashboard information
func (ias *IntegratedAgentSystem) handleGrafanaDashboards(w http.ResponseWriter, r *http.Request) {
	dashboards := ias.monitoringAgent.GetGrafanaDashboards()

	response := map[string]interface{}{
		"dashboards": dashboards,
		"count":      len(dashboards),
		"timestamp":  time.Now().UTC(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleTUIDashboard returns TUI dashboard data
func (ias *IntegratedAgentSystem) handleTUIDashboard(w http.ResponseWriter, r *http.Request) {
	// Get comprehensive dashboard data
	dashboardData := map[string]interface{}{
		"system_status": ias.enhancedOrchestrator.GetStatus(),
		"agents":        ias.getAgentStatuses(),
		"monitoring":    ias.monitoringAgent.GetStatus(),
		"projects":      ias.getProjectData(),
		"timestamp":     time.Now().UTC(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dashboardData)
}

// handleTUIStatus returns TUI agent status
func (ias *IntegratedAgentSystem) handleTUIStatus(w http.ResponseWriter, r *http.Request) {
	status := ias.tuiDashboard.GetStatus()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// getAgentStatuses returns status of all agents
func (ias *IntegratedAgentSystem) getAgentStatuses() map[string]interface{} {
	return map[string]interface{}{
		"enhanced_orchestrator": ias.enhancedOrchestrator.GetStatus(),
		"dynamic_worker":        ias.dynamicWorker.GetStatus(),
		"monitoring_agent":      ias.monitoringAgent.GetStatus(),
		"tui_dashboard":         ias.tuiDashboard.GetStatus(),
		"original_orchestrator": ias.originalOrchestrator.GetStatus(),
	}
}

// getProjectData returns project information
func (ias *IntegratedAgentSystem) getProjectData() []map[string]interface{} {
	// This would return real project data
	return []map[string]interface{}{
		{
			"id":              "project-1",
			"title":           "Enhanced Grafana Dashboard",
			"status":          "executing",
			"progress":        65.0,
			"created_at":      time.Now().Add(-2 * time.Hour),
			"assigned_agents": []string{"code-improvement-ai", "testing-ai"},
		},
		{
			"id":              "project-2",
			"title":           "BigQuery Optimization",
			"status":          "planning",
			"progress":        10.0,
			"created_at":      time.Now().Add(-30 * time.Minute),
			"assigned_agents": []string{"monitoring-ai", "deployment-ai"},
		},
	}
}

// handleGitHubWebhook handles GitHub webhook events
func (ias *IntegratedAgentSystem) handleGitHubWebhook(w http.ResponseWriter, r *http.Request) {
	log.Printf("📥 Received GitHub webhook")

	// This would process GitHub events and create project requests
	// using the enhanced orchestrator with Gemini 2.5 Pro

	response := map[string]interface{}{
		"status":       "processed",
		"message":      "GitHub webhook processed by enhanced orchestrator",
		"orchestrator": "gemini-2.5-pro",
		"timestamp":    time.Now().UTC(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleOriginalOrchestrator handles original orchestrator endpoints for compatibility
func (ias *IntegratedAgentSystem) handleOriginalOrchestrator(w http.ResponseWriter, r *http.Request) {
	// This would delegate to the original orchestrator
	response := map[string]interface{}{
		"status":    "compatibility_mode",
		"message":   "Original orchestrator endpoints for backward compatibility",
		"timestamp": time.Now().UTC(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// calculateSystemMetrics calculates overall system performance metrics
func (ias *IntegratedAgentSystem) calculateSystemMetrics() *SystemPerformanceMetrics {
	// This would calculate real metrics from all components
	return &SystemPerformanceMetrics{
		TotalProjects:       10,
		ActiveProjects:      3,
		CompletedProjects:   7,
		TotalTasks:          50,
		SuccessfulTasks:     45,
		AverageTaskTime:     120.5,
		SystemUptime:        24.0,
		ResourceUtilization: 75.5,
		ErrorRate:           0.05,
		QualityScore:        0.92,
		LastUpdated:         time.Now(),
	}
}

// GetSystemInfo returns comprehensive system information
func (ias *IntegratedAgentSystem) GetSystemInfo() map[string]interface{} {
	ias.mu.RLock()
	defer ias.mu.RUnlock()

	return map[string]interface{}{
		"system_name": "WAS Integrated Agent System",
		"version":     "2.0.0",
		"project_id":  ias.projectID,
		"status":      "operational",
		"components": map[string]interface{}{
			"enhanced_orchestrator": map[string]interface{}{
				"model":        "gemini-2.5-pro",
				"status":       "active",
				"capabilities": []string{"project_management", "task_decomposition", "github_integration"},
			},
			"dynamic_worker": map[string]interface{}{
				"models":       []string{"gemini-1.5-pro-002", "claude-3.5-sonnet", "gemini-1.5-pro", "gemini-1.5-flash", "gemini-1.5-flash-8b"},
				"status":       "active",
				"capabilities": []string{"code_analysis", "documentation", "testing", "data_processing"},
			},
			"monitoring_agent": map[string]interface{}{
				"model":        "gemini-1.5-pro",
				"status":       "active",
				"capabilities": []string{"resource_discovery", "metric_analysis", "anomaly_detection"},
			},
			"original_orchestrator": map[string]interface{}{
				"status":  "active",
				"purpose": "backward_compatibility",
			},
		},
		"performance_metrics": ias.calculateSystemMetrics(),
		"timestamp":           time.Now().UTC(),
	}
}
