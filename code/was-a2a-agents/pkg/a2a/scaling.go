// pkg/a2a/scaling.go
// Dynamic Agent Scaling System for WAS A2A Agents

package a2a

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// AgentManager handles dynamic agent creation and scaling
type AgentManager struct {
	agents         map[string]*Agent
	agentFactories map[string]AgentFactory
	mu             sync.RWMutex
	metrics        *MetricsCollector
	scaler         *AutoScaler
	projectID      string
}

// AgentFactory creates new agents of a specific type
type AgentFactory interface {
	CreateAgent(id, name string) (*Agent, error)
	GetAgentType() string
	GetResourceRequirements() ResourceRequirements
}

// ResourceRequirements defines resource needs for an agent
type ResourceRequirements struct {
	CPUMillis    int64    // CPU in millicores
	MemoryMB     int64    // Memory in megabytes
	StorageGB    int64    // Storage in gigabytes
	NetworkMBps  int64    // Network bandwidth
	GPURequired  bool     // Whether GPU is needed
	SpecialTools []string // Special tool requirements
}

// AutoScaler manages automatic agent scaling
type AutoScaler struct {
	manager       *AgentManager
	scalingRules  []ScalingRule
	checkInterval time.Duration
	metrics       *MetricsCollector
}

// ScalingRule defines when and how to scale agents
type ScalingRule struct {
	AgentType      string
	MetricType     MetricType
	Threshold      float64
	ScaleDirection ScaleDirection
	CooldownPeriod time.Duration
	MinInstances   int
	MaxInstances   int
	ResourceLimits ResourceRequirements
}

type MetricType string

const (
	CPUUtilization    MetricType = "cpu_utilization"
	MemoryUtilization MetricType = "memory_utilization"
	MessageQueueDepth MetricType = "message_queue_depth"
	ResponseTime      MetricType = "response_time"
	ErrorRate         MetricType = "error_rate"
)

type ScaleDirection int

const (
	ScaleUp ScaleDirection = iota
	ScaleDown
)

// MetricsCollector collects and manages agent metrics
type MetricsCollector struct {
	metrics map[string]map[string]float64
	mu      sync.RWMutex
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		metrics: make(map[string]map[string]float64),
	}
}

// RecordMetric records a metric for an agent type
func (mc *MetricsCollector) RecordMetric(agentType, metric string, value float64) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if mc.metrics[agentType] == nil {
		mc.metrics[agentType] = make(map[string]float64)
	}
	mc.metrics[agentType][metric] = value
}

// GetMetrics returns metrics for an agent type
func (mc *MetricsCollector) GetMetrics(agentType string) map[string]float64 {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	if metrics, exists := mc.metrics[agentType]; exists {
		return metrics
	}
	return make(map[string]float64)
}

// NewAgentManager creates a new agent manager
func NewAgentManager(projectID string) *AgentManager {
	manager := &AgentManager{
		agents:         make(map[string]*Agent),
		agentFactories: make(map[string]AgentFactory),
		metrics:        NewMetricsCollector(),
		projectID:      projectID,
	}

	manager.scaler = NewAutoScaler(manager)
	return manager
}

// RegisterAgentFactory registers a factory for creating agents
func (am *AgentManager) RegisterAgentFactory(factory AgentFactory) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.agentFactories[factory.GetAgentType()] = factory

	log.Printf("Registered agent factory: %s", factory.GetAgentType())
}

// CreateAgent creates a new agent instance
func (am *AgentManager) CreateAgent(agentType, id, name string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	factory, exists := am.agentFactories[agentType]
	if !exists {
		return fmt.Errorf("no factory registered for agent type: %s", agentType)
	}

	// Check resource requirements
	requirements := factory.GetResourceRequirements()
	if !am.checkResourceAvailability(requirements) {
		return fmt.Errorf("insufficient resources for agent type: %s", agentType)
	}

	agent, err := factory.CreateAgent(id, name)
	if err != nil {
		return fmt.Errorf("failed to create agent: %w", err)
	}

	am.agents[id] = agent

	// Start agent in a goroutine
	go func() {
		if err := agent.StartListening(); err != nil {
			log.Printf("Agent %s stopped with error: %v", id, err)
			am.removeAgent(id)
		}
	}()

	log.Printf("Created and started agent: %s (%s)", id, agentType)
	return nil
}

// ScaleAgents automatically scales agents based on metrics
func (am *AgentManager) ScaleAgents(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	log.Println("Auto-scaler started")

	for {
		select {
		case <-ctx.Done():
			log.Println("Auto-scaler stopped")
			return
		case <-ticker.C:
			am.evaluateScaling()
		}
	}
}

// evaluateScaling checks metrics and scales agents accordingly
func (am *AgentManager) evaluateScaling() {
	for agentType, factory := range am.agentFactories {
		metrics := am.metrics.GetMetrics(agentType)

		// Evaluate scaling rules
		for _, rule := range am.scaler.scalingRules {
			if rule.AgentType != agentType {
				continue
			}

			currentValue := am.getMetricValue(metrics, rule.MetricType)

			if rule.ScaleDirection == ScaleUp && currentValue > rule.Threshold {
				if am.getAgentCount(agentType) < rule.MaxInstances {
					am.scaleUp(agentType, factory, rule)
				}
			} else if rule.ScaleDirection == ScaleDown && currentValue < rule.Threshold {
				if am.getAgentCount(agentType) > rule.MinInstances {
					am.scaleDown(agentType, rule)
				}
			}
		}
	}
}

// scaleUp creates additional agent instances
func (am *AgentManager) scaleUp(agentType string, factory AgentFactory, rule ScalingRule) {
	count := am.getAgentCount(agentType)

	// Generate unique ID
	id := fmt.Sprintf("%s-%s-%d", agentType, generateShortID(), count+1)
	name := fmt.Sprintf("%s Agent %d", agentType, count+1)

	err := am.CreateAgent(agentType, id, name)
	if err != nil {
		log.Printf("Failed to scale up %s: %v", agentType, err)
		return
	}

	log.Printf("Scaled up %s: created %s (total: %d)", agentType, id, count+1)

	// Record scaling event
	am.metrics.RecordMetric(agentType, "scaling_events", float64(count+1))
}

// scaleDown removes agent instances
func (am *AgentManager) scaleDown(agentType string, rule ScalingRule) {
	agents := am.getAgentsByType(agentType)
	if len(agents) <= rule.MinInstances {
		return
	}

	// Find least active agent to remove
	var oldestAgent *Agent
	oldestTime := time.Now()

	for _, agent := range agents {
		if agent.lastActivity.Before(oldestTime) {
			oldestTime = agent.lastActivity
			oldestAgent = agent
		}
	}

	if oldestAgent != nil {
		oldestAgent.Stop()
		am.removeAgent(oldestAgent.ID)

		log.Printf("Scaled down %s: removed %s (total: %d)",
			agentType, oldestAgent.ID, len(agents)-1)

		am.metrics.RecordMetric(agentType, "scaling_events", float64(len(agents)-1))
	}
}

// getAgentCount returns the number of agents of a specific type
func (am *AgentManager) getAgentCount(agentType string) int {
	am.mu.RLock()
	defer am.mu.RUnlock()

	count := 0
	for _, agent := range am.agents {
		if agent.Platform == agentType {
			count++
		}
	}
	return count
}

// getAgentsByType returns all agents of a specific type
func (am *AgentManager) getAgentsByType(agentType string) []*Agent {
	am.mu.RLock()
	defer am.mu.RUnlock()

	var agents []*Agent
	for _, agent := range am.agents {
		if agent.Platform == agentType {
			agents = append(agents, agent)
		}
	}
	return agents
}

// checkResourceAvailability checks if resources are available
func (am *AgentManager) checkResourceAvailability(requirements ResourceRequirements) bool {
	// Simplified resource check - in production, this would check actual resource availability
	return true
}

// removeAgent removes an agent from the manager
func (am *AgentManager) removeAgent(id string) {
	am.mu.Lock()
	defer am.mu.Unlock()

	if agent, exists := am.agents[id]; exists {
		agent.Stop()
		delete(am.agents, id)
	}
}

// getMetricValue gets a metric value from the metrics map
func (am *AgentManager) getMetricValue(metrics map[string]float64, metricType MetricType) float64 {
	if value, exists := metrics[string(metricType)]; exists {
		return value
	}
	return 0.0
}

// NewAutoScaler creates a new auto-scaler
func NewAutoScaler(manager *AgentManager) *AutoScaler {
	return &AutoScaler{
		manager:       manager,
		scalingRules:  []ScalingRule{},
		checkInterval: 30 * time.Second,
		metrics:       manager.metrics,
	}
}

// generateShortID generates a short unique ID
func generateShortID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano()%10000)
}
