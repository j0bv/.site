// agents/dynamic-worker-agent.go
// Dynamic worker agent with intelligent model selection based on task type

package main

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// DynamicWorkerAgent handles various tasks with optimal model selection
type DynamicWorkerAgent struct {
	agentID        string
	name           string
	role           string
	description    string
	status         string
	lastActivity   time.Time
	tasksCompleted int
	errorCount     int

	// Model selection configuration
	modelSelector      *ModelSelector
	taskProcessor      *TaskProcessor
	performanceTracker *PerformanceTracker
}

// ModelSelector handles intelligent model selection
type ModelSelector struct {
	models map[string]*ModelConfig
	rules  []*SelectionRule
}

// ModelConfig defines configuration for each model
type ModelConfig struct {
	Name            string            `json:"name"`
	Provider        string            `json:"provider"`
	ContextLength   int               `json:"context_length"`
	MaxTokens       int               `json:"max_tokens"`
	CostPerToken    float64           `json:"cost_per_token"`
	Specializations []string          `json:"specializations"`
	Performance     *ModelPerformance `json:"performance"`
}

// ModelPerformance tracks model performance metrics
type ModelPerformance struct {
	AverageResponseTime float64 `json:"average_response_time_ms"`
	SuccessRate         float64 `json:"success_rate"`
	QualityScore        float64 `json:"quality_score"`
	UsageCount          int     `json:"usage_count"`
}

// SelectionRule defines rules for model selection
type SelectionRule struct {
	Condition   *TaskCondition `json:"condition"`
	ModelName   string         `json:"model_name"`
	Priority    int            `json:"priority"`
	Description string         `json:"description"`
}

// TaskCondition defines conditions for task matching
type TaskCondition struct {
	TaskType   string  `json:"task_type"`
	Complexity string  `json:"complexity"`
	Deadline   string  `json:"deadline"`
	Language   string  `json:"language"`
	Domain     string  `json:"domain"`
	MinQuality float64 `json:"min_quality"`
}

// TaskProcessor handles task execution with selected models
type TaskProcessor struct {
	agentID        string
	activeTasks    map[string]*Task
	completedTasks map[string]*Task
	mu             sync.RWMutex
}

// Task represents a work item
type Task struct {
	ID            string                 `json:"id"`
	Title         string                 `json:"title"`
	Description   string                 `json:"description"`
	Type          string                 `json:"type"`
	Complexity    string                 `json:"complexity"`
	Language      string                 `json:"language"`
	Domain        string                 `json:"domain"`
	Priority      string                 `json:"priority"`
	Deadline      time.Time              `json:"deadline"`
	Status        string                 `json:"status"`
	AssignedModel string                 `json:"assigned_model"`
	Input         map[string]interface{} `json:"input"`
	Output        map[string]interface{} `json:"output"`
	CreatedAt     time.Time              `json:"created_at"`
	StartedAt     *time.Time             `json:"started_at,omitempty"`
	CompletedAt   *time.Time             `json:"completed_at,omitempty"`
	Error         string                 `json:"error,omitempty"`
}

// PerformanceTracker monitors agent performance
type PerformanceTracker struct {
	agentID         string
	totalTasks      int
	successfulTasks int
	averageTaskTime float64
	qualityScore    float64
	resourceUsage   float64
	lastUpdated     time.Time
}

// NewDynamicWorkerAgent creates a new dynamic worker agent
func NewDynamicWorkerAgent(agentID, name, role, description string) *DynamicWorkerAgent {
	agent := &DynamicWorkerAgent{
		agentID:        agentID,
		name:           name,
		role:           role,
		description:    description,
		status:         "active",
		lastActivity:   time.Now(),
		tasksCompleted: 0,
		errorCount:     0,
	}

	// Initialize model selector with optimized configurations
	agent.modelSelector = NewModelSelector()

	// Initialize task processor
	agent.taskProcessor = &TaskProcessor{
		agentID:        agentID,
		activeTasks:    make(map[string]*Task),
		completedTasks: make(map[string]*Task),
	}

	// Initialize performance tracker
	agent.performanceTracker = &PerformanceTracker{
		agentID: agentID,
	}

	return agent
}

// NewModelSelector creates a new model selector with recommended configurations
func NewModelSelector() *ModelSelector {
	selector := &ModelSelector{
		models: make(map[string]*ModelConfig),
		rules:  make([]*SelectionRule, 0),
	}

	// Configure models based on recommendations
	selector.models["gemini-1.5-pro-002"] = &ModelConfig{
		Name:          "gemini-1.5-pro-002",
		Provider:      "google",
		ContextLength: 128000,
		MaxTokens:     8192,
		CostPerToken:  0.00001,
		Specializations: []string{
			"code_analysis",
			"code_generation",
			"refactoring",
			"architecture",
			"debugging",
		},
		Performance: &ModelPerformance{
			AverageResponseTime: 2500,
			SuccessRate:         0.95,
			QualityScore:        0.92,
			UsageCount:          0,
		},
	}

	selector.models["claude-3.5-sonnet"] = &ModelConfig{
		Name:          "claude-3.5-sonnet",
		Provider:      "anthropic",
		ContextLength: 200000,
		MaxTokens:     8192,
		CostPerToken:  0.000015,
		Specializations: []string{
			"technical_writing",
			"documentation",
			"code_review",
			"explanation",
			"troubleshooting",
		},
		Performance: &ModelPerformance{
			AverageResponseTime: 3000,
			SuccessRate:         0.98,
			QualityScore:        0.96,
			UsageCount:          0,
		},
	}

	selector.models["gemini-1.5-pro"] = &ModelConfig{
		Name:          "gemini-1.5-pro",
		Provider:      "google",
		ContextLength: 128000,
		MaxTokens:     8192,
		CostPerToken:  0.000008,
		Specializations: []string{
			"testing",
			"data_analysis",
			"general_coding",
			"optimization",
			"integration",
		},
		Performance: &ModelPerformance{
			AverageResponseTime: 2000,
			SuccessRate:         0.93,
			QualityScore:        0.89,
			UsageCount:          0,
		},
	}

	selector.models["gemini-1.5-flash"] = &ModelConfig{
		Name:          "gemini-1.5-flash",
		Provider:      "google",
		ContextLength: 128000,
		MaxTokens:     8192,
		CostPerToken:  0.000003,
		Specializations: []string{
			"quick_tasks",
			"data_processing",
			"simple_coding",
			"formatting",
			"validation",
		},
		Performance: &ModelPerformance{
			AverageResponseTime: 800,
			SuccessRate:         0.90,
			QualityScore:        0.85,
			UsageCount:          0,
		},
	}

	selector.models["gemini-1.5-flash-8b"] = &ModelConfig{
		Name:          "gemini-1.5-flash-8b",
		Provider:      "google",
		ContextLength: 128000,
		MaxTokens:     8192,
		CostPerToken:  0.000001,
		Specializations: []string{
			"fast_processing",
			"log_analysis",
			"metric_processing",
			"simple_automation",
			"data_transformation",
		},
		Performance: &ModelPerformance{
			AverageResponseTime: 400,
			SuccessRate:         0.88,
			QualityScore:        0.82,
			UsageCount:          0,
		},
	}

	// Configure selection rules
	selector.rules = []*SelectionRule{
		{
			Condition: &TaskCondition{
				TaskType:   "code_analysis",
				Complexity: "high",
			},
			ModelName:   "gemini-1.5-pro-002",
			Priority:    1,
			Description: "High complexity code analysis requires best code understanding",
		},
		{
			Condition: &TaskCondition{
				TaskType: "documentation",
			},
			ModelName:   "claude-3.5-sonnet",
			Priority:    1,
			Description: "Documentation tasks benefit from superior technical writing",
		},
		{
			Condition: &TaskCondition{
				TaskType:   "testing",
				Complexity: "medium",
			},
			ModelName:   "gemini-1.5-pro",
			Priority:    1,
			Description: "Balanced performance for testing tasks",
		},
		{
			Condition: &TaskCondition{
				TaskType: "data_processing",
				Deadline: "urgent",
			},
			ModelName:   "gemini-1.5-flash-8b",
			Priority:    1,
			Description: "Fast processing for urgent data tasks",
		},
		{
			Condition: &TaskCondition{
				TaskType: "general_coding",
				Deadline: "tight",
			},
			ModelName:   "gemini-1.5-flash",
			Priority:    1,
			Description: "Quick response for tight deadlines",
		},
		{
			Condition: &TaskCondition{
				TaskType: "code_generation",
				Language: "go",
			},
			ModelName:   "gemini-1.5-pro-002",
			Priority:    2,
			Description: "Go development benefits from advanced code understanding",
		},
		{
			Condition: &TaskCondition{
				TaskType: "code_generation",
				Language: "typescript",
			},
			ModelName:   "gemini-1.5-pro",
			Priority:    2,
			Description: "TypeScript development with balanced performance",
		},
		{
			Condition: &TaskCondition{
				Domain: "bigquery",
			},
			ModelName:   "gemini-1.5-pro",
			Priority:    2,
			Description: "BigQuery operations benefit from Google's models",
		},
		{
			Condition: &TaskCondition{
				Domain: "observable",
			},
			ModelName:   "gemini-1.5-pro",
			Priority:    2,
			Description: "Observable Framework development optimization",
		},
		{
			Condition: &TaskCondition{
				MinQuality: 0.95,
			},
			ModelName:   "claude-3.5-sonnet",
			Priority:    1,
			Description: "High quality requirements favor Claude",
		},
	}

	return selector
}

// SelectOptimalModel selects the best model for a given task
func (ms *ModelSelector) SelectOptimalModel(task *Task) string {
	bestModel := "gemini-1.5-pro" // Default fallback
	bestScore := 0.0

	// Score each rule against the task
	for _, rule := range ms.rules {
		score := ms.calculateRuleScore(rule, task)
		if score > bestScore {
			bestScore = score
			bestModel = rule.ModelName
		}
	}

	// If no rules match well, use default based on task type
	if bestScore < 0.5 {
		bestModel = ms.getDefaultModelForTaskType(task.Type)
	}

	return bestModel
}

// calculateRuleScore calculates how well a rule matches a task
func (ms *ModelSelector) calculateRuleScore(rule *SelectionRule, task *Task) float64 {
	score := 0.0
	condition := rule.Condition

	// Task type matching
	if condition.TaskType != "" && strings.EqualFold(condition.TaskType, task.Type) {
		score += 0.4
	}

	// Complexity matching
	if condition.Complexity != "" && strings.EqualFold(condition.Complexity, task.Complexity) {
		score += 0.3
	}

	// Language matching
	if condition.Language != "" && strings.EqualFold(condition.Language, task.Language) {
		score += 0.2
	}

	// Domain matching
	if condition.Domain != "" && strings.Contains(strings.ToLower(task.Description), strings.ToLower(condition.Domain)) {
		score += 0.1
	}

	// Priority weighting
	score *= float64(rule.Priority)

	return score
}

// getDefaultModelForTaskType returns a default model for task type
func (ms *ModelSelector) getDefaultModelForTaskType(taskType string) string {
	switch strings.ToLower(taskType) {
	case "code_analysis", "code_generation", "refactoring":
		return "gemini-1.5-pro-002"
	case "documentation", "technical_writing":
		return "claude-3.5-sonnet"
	case "testing", "validation":
		return "gemini-1.5-pro"
	case "data_processing", "log_analysis":
		return "gemini-1.5-flash-8b"
	default:
		return "gemini-1.5-pro"
	}
}

// ProcessTask processes a task using the optimal model
func (dwa *DynamicWorkerAgent) ProcessTask(task *Task) error {
	// Select optimal model
	selectedModel := dwa.modelSelector.SelectOptimalModel(task)
	task.AssignedModel = selectedModel

	// Update task status
	task.Status = "processing"
	now := time.Now()
	task.StartedAt = &now

	// Add to active tasks
	dwa.taskProcessor.mu.Lock()
	dwa.taskProcessor.activeTasks[task.ID] = task
	dwa.taskProcessor.mu.Unlock()

	// Process the task
	result, err := dwa.executeTaskWithModel(task, selectedModel)
	if err != nil {
		task.Status = "failed"
		task.Error = err.Error()
		dwa.errorCount++
	} else {
		task.Status = "completed"
		task.Output = result
		completedAt := time.Now()
		task.CompletedAt = &completedAt
		dwa.tasksCompleted++
	}

	// Update performance metrics
	dwa.updatePerformanceMetrics(task)

	// Move to completed tasks
	dwa.taskProcessor.mu.Lock()
	delete(dwa.taskProcessor.activeTasks, task.ID)
	dwa.taskProcessor.completedTasks[task.ID] = task
	dwa.taskProcessor.mu.Unlock()

	// Update last activity
	dwa.lastActivity = time.Now()

	return err
}

// executeTaskWithModel executes a task using the specified model
func (dwa *DynamicWorkerAgent) executeTaskWithModel(task *Task, modelName string) (map[string]interface{}, error) {
	// This would integrate with the actual model APIs
	// For now, simulate task execution based on model capabilities

	modelConfig, exists := dwa.modelSelector.models[modelName]
	if !exists {
		return nil, fmt.Errorf("model %s not found", modelName)
	}

	// Simulate processing time based on model performance
	processingTime := time.Duration(modelConfig.Performance.AverageResponseTime) * time.Millisecond
	time.Sleep(processingTime)

	// Generate mock result based on task type
	result := dwa.generateMockResult(task, modelName)

	// Update model usage statistics
	modelConfig.Performance.UsageCount++

	return result, nil
}

// generateMockResult generates a mock result for demonstration
func (dwa *DynamicWorkerAgent) generateMockResult(task *Task, modelName string) map[string]interface{} {
	result := map[string]interface{}{
		"task_id":         task.ID,
		"model_used":      modelName,
		"processing_time": time.Since(*task.StartedAt).Milliseconds(),
		"status":          "completed",
		"timestamp":       time.Now().UTC(),
	}

	// Add model-specific results
	switch modelName {
	case "gemini-1.5-pro-002":
		result["code_analysis"] = map[string]interface{}{
			"quality_score": 0.92,
			"issues_found":  3,
			"recommendations": []string{
				"Optimize database queries",
				"Add error handling",
				"Improve code documentation",
			},
		}
	case "claude-3.5-sonnet":
		result["documentation"] = map[string]interface{}{
			"quality_score":      0.96,
			"sections_generated": 5,
			"readability_score":  0.94,
		}
	case "gemini-1.5-pro":
		result["testing"] = map[string]interface{}{
			"test_cases_generated": 12,
			"coverage_percentage":  85.5,
			"quality_score":        0.89,
		}
	case "gemini-1.5-flash":
		result["quick_processing"] = map[string]interface{}{
			"response_time": "fast",
			"quality_score": 0.85,
		}
	case "gemini-1.5-flash-8b":
		result["data_processing"] = map[string]interface{}{
			"records_processed": 1000,
			"processing_time":   "very_fast",
			"quality_score":     0.82,
		}
	}

	return result
}

// updatePerformanceMetrics updates agent performance metrics
func (dwa *DynamicWorkerAgent) updatePerformanceMetrics(task *Task) {
	dwa.performanceTracker.totalTasks++

	if task.Status == "completed" {
		dwa.performanceTracker.successfulTasks++
	}

	// Calculate success rate
	if dwa.performanceTracker.totalTasks > 0 {
		dwa.performanceTracker.successfulTasks = int(float64(dwa.performanceTracker.successfulTasks) / float64(dwa.performanceTracker.totalTasks) * 100)
	}

	// Calculate average task time
	if task.StartedAt != nil && task.CompletedAt != nil {
		taskTime := task.CompletedAt.Sub(*task.StartedAt).Seconds()
		dwa.performanceTracker.averageTaskTime = (dwa.performanceTracker.averageTaskTime + taskTime) / 2
	}

	// Update quality score based on model performance
	if modelConfig, exists := dwa.modelSelector.models[task.AssignedModel]; exists {
		dwa.performanceTracker.qualityScore = (dwa.performanceTracker.qualityScore + modelConfig.Performance.QualityScore) / 2
	}

	dwa.performanceTracker.lastUpdated = time.Now()
}

// GetStatus returns the current status of the agent
func (dwa *DynamicWorkerAgent) GetStatus() map[string]interface{} {
	dwa.taskProcessor.mu.RLock()
	defer dwa.taskProcessor.mu.RUnlock()

	return map[string]interface{}{
		"agent_id":         dwa.agentID,
		"name":             dwa.name,
		"role":             dwa.role,
		"status":           dwa.status,
		"last_activity":    dwa.lastActivity,
		"tasks_completed":  dwa.tasksCompleted,
		"error_count":      dwa.errorCount,
		"active_tasks":     len(dwa.taskProcessor.activeTasks),
		"completed_tasks":  len(dwa.taskProcessor.completedTasks),
		"performance":      dwa.performanceTracker,
		"available_models": len(dwa.modelSelector.models),
	}
}

// GetAvailableModels returns information about available models
func (dwa *DynamicWorkerAgent) GetAvailableModels() map[string]*ModelConfig {
	return dwa.modelSelector.models
}

// GetModelRecommendations returns model recommendations for a task type
func (dwa *DynamicWorkerAgent) GetModelRecommendations(taskType string) []*ModelRecommendation {
	recommendations := make([]*ModelRecommendation, 0)

	for _, model := range dwa.modelSelector.models {
		if dwa.isModelSuitableForTaskType(model, taskType) {
			recommendation := &ModelRecommendation{
				ModelName:       model.Name,
				Provider:        model.Provider,
				Specializations: model.Specializations,
				Performance:     model.Performance,
				Reasoning:       dwa.getModelReasoning(model, taskType),
			}
			recommendations = append(recommendations, recommendation)
		}
	}

	return recommendations
}

// ModelRecommendation provides model selection guidance
type ModelRecommendation struct {
	ModelName       string            `json:"model_name"`
	Provider        string            `json:"provider"`
	Specializations []string          `json:"specializations"`
	Performance     *ModelPerformance `json:"performance"`
	Reasoning       string            `json:"reasoning"`
}

// isModelSuitableForTaskType checks if a model is suitable for a task type
func (dwa *DynamicWorkerAgent) isModelSuitableForTaskType(model *ModelConfig, taskType string) bool {
	for _, specialization := range model.Specializations {
		if strings.Contains(strings.ToLower(specialization), strings.ToLower(taskType)) {
			return true
		}
	}
	return false
}

// getModelReasoning provides reasoning for model selection
func (dwa *DynamicWorkerAgent) getModelReasoning(model *ModelConfig, taskType string) string {
	switch model.Name {
	case "gemini-1.5-pro-002":
		return "Best choice for complex code analysis and generation with superior understanding"
	case "claude-3.5-sonnet":
		return "Optimal for technical writing and documentation with highest quality output"
	case "gemini-1.5-pro":
		return "Balanced performance for general coding tasks and testing"
	case "gemini-1.5-flash":
		return "Fast response for quick tasks and tight deadlines"
	case "gemini-1.5-flash-8b":
		return "Ultra-fast processing for data-intensive tasks"
	default:
		return "Suitable for general purpose tasks"
	}
}
