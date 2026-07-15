// pkg/rl/engine.go
// Reinforcement Learning Engine for Tool Analysis and System Improvement

package rl

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"sync"
	"time"
)

// RLEngine implements reinforcement learning for tool analysis
type RLEngine struct {
	stateSpace    map[string]State
	actionSpace   map[string]Action
	qTable        map[string]map[string]float64
	learningRate  float64
	discountFactor float64
	epsilon       float64
	mu            sync.RWMutex
	experienceBuffer []Experience
	maxBufferSize int
}

// State represents the current state of a system or tool
type State struct {
	ID          string                 `json:"id"`
	Features    map[string]float64     `json:"features"`
	Context     map[string]interface{} `json:"context"`
	Timestamp   time.Time              `json:"timestamp"`
	Quality     float64                `json:"quality"`
	Performance float64                `json:"performance"`
}

// Action represents an action that can be taken to improve a system
type Action struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Parameters  map[string]interface{} `json:"parameters"`
	ExpectedReward float64             `json:"expected_reward"`
	Confidence  float64                `json:"confidence"`
}

// Experience represents a learning experience
type Experience struct {
	State      State  `json:"state"`
	Action     Action `json:"action"`
	Reward     float64 `json:"reward"`
	NextState  State  `json:"next_state"`
	Timestamp  time.Time `json:"timestamp"`
	Success    bool   `json:"success"`
}

// Reward represents the reward for an action
type Reward struct {
	Value       float64 `json:"value"`
	Components  map[string]float64 `json:"components"`
	Timestamp   time.Time `json:"timestamp"`
	Context     string `json:"context"`
}

// NewRLEngine creates a new RL engine
func NewRLEngine() *RLEngine {
	return &RLEngine{
		stateSpace:     make(map[string]State),
		actionSpace:    make(map[string]Action),
		qTable:         make(map[string]map[string]float64),
		learningRate:   0.1,
		discountFactor: 0.9,
		epsilon:        0.1,
		experienceBuffer: make([]Experience, 0),
		maxBufferSize:  10000,
	}
}

// AnalyzeAndImprove analyzes a system and suggests improvements using RL
func (rl *RLEngine) AnalyzeAndImprove(analysisData map[string]interface{}) []string {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Extract state from analysis data
	state := rl.extractState(analysisData)
	
	// Get best action for current state
	action := rl.selectAction(state)
	
	// Generate improvements based on action
	improvements := rl.generateImprovements(state, action)
	
	// Store experience for learning
	rl.storeExperience(state, action, 0.0, state, true)
	
	return improvements
}

// extractState extracts a state from analysis data
func (rl *RLEngine) extractState(analysisData map[string]interface{}) State {
	features := make(map[string]float64)
	context := make(map[string]interface{})
	
	// Extract features from analysis
	if functions, ok := analysisData["functions"].(map[string]interface{}); ok {
		if count, ok := functions["count"].(float64); ok {
			features["function_count"] = count
		}
		if complexity, ok := functions["complexity"].(map[string]interface{}); ok {
			if avgComplexity, ok := complexity["average"].(float64); ok {
				features["avg_complexity"] = avgComplexity
			}
		}
	}
	
	if security, ok := analysisData["security"].(map[string]interface{}); ok {
		if vulns, ok := security["vulnerabilities"].([]interface{}); ok {
			features["vulnerability_count"] = float64(len(vulns))
		}
	}
	
	if patterns, ok := analysisData["patterns"].(map[string]interface{}); ok {
		if crypto, ok := patterns["crypto"].([]interface{}); ok {
			features["crypto_patterns"] = float64(len(crypto))
		}
		if networking, ok := patterns["networking"].([]interface{}); ok {
			features["networking_patterns"] = float64(len(networking))
		}
	}
	
	// Calculate quality and performance scores
	quality := rl.calculateQuality(features)
	performance := rl.calculatePerformance(features)
	
	stateID := rl.generateStateID(features)
	
	return State{
		ID:          stateID,
		Features:    features,
		Context:     context,
		Timestamp:   time.Now(),
		Quality:     quality,
		Performance: performance,
	}
}

// selectAction selects the best action for a given state using epsilon-greedy
func (rl *RLEngine) selectAction(state State) Action {
	stateID := state.ID
	
	// Initialize Q-table for this state if needed
	if _, exists := rl.qTable[stateID]; !exists {
		rl.qTable[stateID] = make(map[string]float64)
	}
	
	// Epsilon-greedy action selection
	if rand.Float64() < rl.epsilon {
		// Explore: choose random action
		return rl.getRandomAction()
	} else {
		// Exploit: choose best known action
		return rl.getBestAction(stateID)
	}
}

// getBestAction returns the action with highest Q-value for a state
func (rl *RLEngine) getBestAction(stateID string) Action {
	bestActionID := ""
	bestValue := math.Inf(-1)
	
	for actionID, value := range rl.qTable[stateID] {
		if value > bestValue {
			bestValue = value
			bestActionID = actionID
		}
	}
	
	if bestActionID == "" {
		return rl.getRandomAction()
	}
	
	return rl.actionSpace[bestActionID]
}

// getRandomAction returns a random action
func (rl *RLEngine) getRandomAction() Action {
	actions := make([]Action, 0, len(rl.actionSpace))
	for _, action := range rl.actionSpace {
		actions = append(actions, action)
	}
	
	if len(actions) == 0 {
		// Initialize with default actions if empty
		rl.initializeDefaultActions()
		actions = make([]Action, 0, len(rl.actionSpace))
		for _, action := range rl.actionSpace {
			actions = append(actions, action)
		}
	}
	
	return actions[rand.Intn(len(actions))]
}

// initializeDefaultActions initializes default actions for the RL engine
func (rl *RLEngine) initializeDefaultActions() {
	defaultActions := []Action{
		{
			ID:   "optimize_performance",
			Type: "optimization",
			Parameters: map[string]interface{}{
				"target": "performance",
				"method": "algorithm_optimization",
			},
			ExpectedReward: 0.8,
			Confidence:     0.7,
		},
		{
			ID:   "improve_security",
			Type: "security",
			Parameters: map[string]interface{}{
				"target": "vulnerabilities",
				"method": "secure_coding_practices",
			},
			ExpectedReward: 0.9,
			Confidence:     0.8,
		},
		{
			ID:   "reduce_complexity",
			Type: "refactoring",
			Parameters: map[string]interface{}{
				"target": "code_complexity",
				"method": "modularization",
			},
			ExpectedReward: 0.7,
			Confidence:     0.6,
		},
		{
			ID:   "enhance_readability",
			Type: "maintainability",
			Parameters: map[string]interface{}{
				"target": "code_readability",
				"method": "documentation_and_naming",
			},
			ExpectedReward: 0.6,
			Confidence:     0.7,
		},
		{
			ID:   "add_monitoring",
			Type: "observability",
			Parameters: map[string]interface{}{
				"target": "system_observability",
				"method": "logging_and_metrics",
			},
			ExpectedReward: 0.8,
			Confidence:     0.8,
		},
	}
	
	for _, action := range defaultActions {
		rl.actionSpace[action.ID] = action
	}
}

// generateImprovements generates improvement suggestions based on state and action
func (rl *RLEngine) generateImprovements(state State, action Action) []string {
	improvements := []string{}
	
	switch action.Type {
	case "optimization":
		improvements = append(improvements, rl.generateOptimizationImprovements(state, action)...)
	case "security":
		improvements = append(improvements, rl.generateSecurityImprovements(state, action)...)
	case "refactoring":
		improvements = append(improvements, rl.generateRefactoringImprovements(state, action)...)
	case "maintainability":
		improvements = append(improvements, rl.generateMaintainabilityImprovements(state, action)...)
	case "observability":
		improvements = append(improvements, rl.generateObservabilityImprovements(state, action)...)
	}
	
	return improvements
}

// generateOptimizationImprovements generates performance optimization suggestions
func (rl *RLEngine) generateOptimizationImprovements(state State, action Action) []string {
	improvements := []string{}
	
	if state.Features["function_count"] > 100 {
		improvements = append(improvements, "Consider breaking down large functions into smaller, more focused functions")
	}
	
	if state.Features["avg_complexity"] > 10 {
		improvements = append(improvements, "Reduce cyclomatic complexity by simplifying conditional logic")
	}
	
	if state.Features["crypto_patterns"] > 0 {
		improvements = append(improvements, "Optimize cryptographic operations by using hardware acceleration where available")
	}
	
	if state.Features["networking_patterns"] > 0 {
		improvements = append(improvements, "Implement connection pooling and async I/O for better network performance")
	}
	
	return improvements
}

// generateSecurityImprovements generates security improvement suggestions
func (rl *RLEngine) generateSecurityImprovements(state State, action Action) []string {
	improvements := []string{}
	
	if state.Features["vulnerability_count"] > 0 {
		improvements = append(improvements, "Address identified security vulnerabilities immediately")
		improvements = append(improvements, "Implement automated security testing in CI/CD pipeline")
	}
	
	improvements = append(improvements, "Add input validation and sanitization for all user inputs")
	improvements = append(improvements, "Implement proper error handling to avoid information disclosure")
	improvements = append(improvements, "Use secure coding practices and follow OWASP guidelines")
	
	return improvements
}

// generateRefactoringImprovements generates refactoring suggestions
func (rl *RLEngine) generateRefactoringImprovements(state State, action Action) []string {
	improvements := []string{}
	
	if state.Features["function_count"] > 50 {
		improvements = append(improvements, "Break down monolithic code into smaller, cohesive modules")
	}
	
	improvements = append(improvements, "Extract common functionality into reusable components")
	improvements = append(improvements, "Remove code duplication and consolidate similar functions")
	improvements = append(improvements, "Apply design patterns to improve code structure")
	
	return improvements
}

// generateMaintainabilityImprovements generates maintainability suggestions
func (rl *RLEngine) generateMaintainabilityImprovements(state State, action Action) []string {
	improvements := []string{}
	
	improvements = append(improvements, "Add comprehensive documentation and code comments")
	improvements = append(improvements, "Use descriptive variable and function names")
	improvements = append(improvements, "Implement consistent coding standards and formatting")
	improvements = append(improvements, "Add unit tests for critical functionality")
	
	return improvements
}

// generateObservabilityImprovements generates observability suggestions
func (rl *RLEngine) generateObservabilityImprovements(state State, action Action) []string {
	improvements := []string{}
	
	improvements = append(improvements, "Add structured logging throughout the application")
	improvements = append(improvements, "Implement metrics collection for key performance indicators")
	improvements = append(improvements, "Add health checks and monitoring endpoints")
	improvements = append(improvements, "Implement distributed tracing for complex workflows")
	
	return improvements
}

// calculateQuality calculates a quality score based on features
func (rl *RLEngine) calculateQuality(features map[string]float64) float64 {
	quality := 1.0
	
	// Penalize high complexity
	if complexity, ok := features["avg_complexity"]; ok {
		quality -= complexity * 0.1
	}
	
	// Penalize vulnerabilities
	if vulns, ok := features["vulnerability_count"]; ok {
		quality -= vulns * 0.2
	}
	
	// Reward good patterns
	if crypto, ok := features["crypto_patterns"]; ok && crypto > 0 {
		quality += 0.1
	}
	
	return math.Max(0.0, math.Min(1.0, quality))
}

// calculatePerformance calculates a performance score based on features
func (rl *RLEngine) calculatePerformance(features map[string]float64) float64 {
	performance := 1.0
	
	// Penalize high function count (indicates potential performance issues)
	if funcCount, ok := features["function_count"]; ok {
		performance -= funcCount * 0.001
	}
	
	// Reward optimization patterns
	if networking, ok := features["networking_patterns"]; ok && networking > 0 {
		performance += 0.1
	}
	
	return math.Max(0.0, math.Min(1.0, performance))
}

// generateStateID generates a unique ID for a state based on its features
func (rl *RLEngine) generateStateID(features map[string]float64) string {
	// Simple hash of features for state ID
	hash := 0
	for key, value := range features {
		hash += len(key) + int(value*1000)
	}
	return fmt.Sprintf("state_%d", hash)
}

// storeExperience stores a learning experience
func (rl *RLEngine) storeExperience(state State, action Action, reward float64, nextState State, success bool) {
	experience := Experience{
		State:     state,
		Action:    action,
		Reward:    reward,
		NextState: nextState,
		Timestamp: time.Now(),
		Success:   success,
	}
	
	rl.experienceBuffer = append(rl.experienceBuffer, experience)
	
	// Maintain buffer size
	if len(rl.experienceBuffer) > rl.maxBufferSize {
		rl.experienceBuffer = rl.experienceBuffer[1:]
	}
	
	// Update Q-table
	rl.updateQTable(state, action, reward, nextState)
}

// updateQTable updates the Q-table using Q-learning
func (rl *RLEngine) updateQTable(state State, action Action, reward float64, nextState State) {
	stateID := state.ID
	actionID := action.ID
	
	// Initialize Q-values if needed
	if _, exists := rl.qTable[stateID]; !exists {
		rl.qTable[stateID] = make(map[string]float64)
	}
	if _, exists := rl.qTable[stateID][actionID]; !exists {
		rl.qTable[stateID][actionID] = 0.0
	}
	
	// Find max Q-value for next state
	maxNextQ := 0.0
	if nextStateQ, exists := rl.qTable[nextState.ID]; exists {
		for _, qValue := range nextStateQ {
			if qValue > maxNextQ {
				maxNextQ = qValue
			}
		}
	}
	
	// Q-learning update
	currentQ := rl.qTable[stateID][actionID]
	newQ := currentQ + rl.learningRate*(reward+rl.discountFactor*maxNextQ-currentQ)
	rl.qTable[stateID][actionID] = newQ
}

// LearnFromExperience learns from stored experiences
func (rl *RLEngine) LearnFromExperience() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	// Sample random experiences for learning
	sampleSize := min(100, len(rl.experienceBuffer))
	if sampleSize == 0 {
		return
	}
	
	// Randomly sample experiences
	indices := make([]int, sampleSize)
	for i := range indices {
		indices[i] = rand.Intn(len(rl.experienceBuffer))
	}
	
	// Learn from sampled experiences
	for _, idx := range indices {
		exp := rl.experienceBuffer[idx]
		rl.updateQTable(exp.State, exp.Action, exp.Reward, exp.NextState)
	}
}

// GetStatistics returns learning statistics
func (rl *RLEngine) GetStatistics() map[string]interface{} {
	rl.mu.RLock()
	defer rl.mu.RUnlock()
	
	return map[string]interface{}{
		"states_learned":     len(rl.stateSpace),
		"actions_available":  len(rl.actionSpace),
		"experiences_stored": len(rl.experienceBuffer),
		"learning_rate":      rl.learningRate,
		"epsilon":           rl.epsilon,
	}
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
