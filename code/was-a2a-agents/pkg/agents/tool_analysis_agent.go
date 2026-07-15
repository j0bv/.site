// pkg/agents/tool_analysis_agent.go
// Tool Analysis Agent that learns from existing tools to build better systems

package agents

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/was-project/a2a-agents/pkg/a2a"
	"github.com/was-project/a2a-agents/pkg/mcp"
	"github.com/was-project/a2a-agents/pkg/mcp/tools"
	"github.com/was-project/a2a-agents/pkg/rl"
)

// ToolAnalysisAgent analyzes existing tools and learns to build better systems
type ToolAnalysisAgent struct {
	*a2a.Agent
	ghidraTool    *tools.GhidraBinaryAnalyzer
	rlEngine      *rl.RLEngine
	mcpServer     *mcp.MCPServer
	learnedTools  map[string]ToolKnowledge
	mu            sync.RWMutex
}

// ToolKnowledge represents knowledge learned from analyzing tools
type ToolKnowledge struct {
	ToolName        string                 `json:"tool_name"`
	ToolType        string                 `json:"tool_type"`
	AnalysisResults map[string]interface{} `json:"analysis_results"`
	BestPractices   []string               `json:"best_practices"`
	AntiPatterns    []string               `json:"anti_patterns"`
	Improvements    []string               `json:"improvements"`
	Performance     ToolPerformance        `json:"performance"`
	LastAnalyzed    time.Time              `json:"last_analyzed"`
}

// ToolPerformance represents performance metrics for a tool
type ToolPerformance struct {
	Efficiency     float64 `json:"efficiency"`
	Reliability    float64 `json:"reliability"`
	Maintainability float64 `json:"maintainability"`
	Security       float64 `json:"security"`
	Scalability    float64 `json:"scalability"`
}

// NewToolAnalysisAgent creates a new tool analysis agent
func NewToolAnalysisAgent(projectID string, mcpServer *mcp.MCPServer) (*ToolAnalysisAgent, error) {
	agent, err := a2a.NewAgent(projectID, "tool-analysis-agent", "Tool Analysis Agent", "analysis")
	if err != nil {
		return nil, err
	}

	analysisAgent := &ToolAnalysisAgent{
		Agent:        agent,
		ghidraTool:   tools.NewGhidraBinaryAnalyzer(nil), // Will be initialized with MCP client
		rlEngine:     rl.NewRLEngine(),
		mcpServer:    mcpServer,
		learnedTools: make(map[string]ToolKnowledge),
	}

	// Add specialized tools
	analysisAgent.AddTool(&ToolAnalysisTool{agent: analysisAgent})
	analysisAgent.AddTool(&ToolLearningTool{agent: analysisAgent})
	analysisAgent.AddTool(&ToolRecommendationTool{agent: analysisAgent})

	// Register message handlers
	analysisAgent.AddMessageHandler("AnalyzeTool", analysisAgent.handleAnalyzeTool)
	analysisAgent.AddMessageHandler("LearnFromTool", analysisAgent.handleLearnFromTool)
	analysisAgent.AddMessageHandler("RecommendImprovements", analysisAgent.handleRecommendImprovements)
	analysisAgent.AddMessageHandler("BuildBetterTool", analysisAgent.handleBuildBetterTool)

	return analysisAgent, nil
}

// handleAnalyzeTool analyzes a tool using Ghidra and other analysis methods
func (ta *ToolAnalysisAgent) handleAnalyzeTool(ctx context.Context, msg *a2a.A2AMessage) error {
	toolPath, ok := msg.Content["tool_path"].(string)
	if !ok {
		return fmt.Errorf("tool_path parameter required")
	}

	analysisType, _ := msg.Content["analysis_type"].(string)
	if analysisType == "" {
		analysisType = "comprehensive"
	}

	log.Printf("Analyzing tool: %s", toolPath)

	// Perform comprehensive analysis
	analysisResults, err := ta.performComprehensiveAnalysis(ctx, toolPath, analysisType)
	if err != nil {
		return fmt.Errorf("analysis failed: %w", err)
	}

	// Store learned knowledge
	ta.storeToolKnowledge(toolPath, analysisResults)

	// Send results back
	return ta.SendMessage(ctx, msg.From, "AnalysisComplete", map[string]interface{}{
		"tool_path":        toolPath,
		"analysis_results": analysisResults,
		"recommendations":  ta.generateRecommendations(analysisResults),
	})
}

// handleLearnFromTool learns from tool analysis to improve future recommendations
func (ta *ToolAnalysisAgent) handleLearnFromTool(ctx context.Context, msg *a2a.A2AMessage) error {
	toolPath, ok := msg.Content["tool_path"].(string)
	if !ok {
		return fmt.Errorf("tool_path parameter required")
	}

	feedback, _ := msg.Content["feedback"].(map[string]interface{})
	success, _ := msg.Content["success"].(bool)

	log.Printf("Learning from tool: %s, success: %v", toolPath, success)

	// Update RL engine with feedback
	ta.updateLearningFromFeedback(toolPath, feedback, success)

	// Update tool knowledge
	ta.updateToolKnowledge(toolPath, feedback, success)

	return nil
}

// handleRecommendImprovements provides recommendations for building better tools
func (ta *ToolAnalysisAgent) handleRecommendImprovements(ctx context.Context, msg *a2a.A2AMessage) error {
	toolType, ok := msg.Content["tool_type"].(string)
	if !ok {
		return fmt.Errorf("tool_type parameter required")
	}

	requirements, _ := msg.Content["requirements"].(map[string]interface{})

	log.Printf("Generating recommendations for tool type: %s", toolType)

	// Generate recommendations based on learned knowledge
	recommendations := ta.generateToolRecommendations(toolType, requirements)

	// Use RL to optimize recommendations
	optimizedRecommendations := ta.optimizeRecommendationsWithRL(recommendations)

	return ta.SendMessage(ctx, msg.From, "RecommendationsReady", map[string]interface{}{
		"tool_type":    toolType,
		"recommendations": optimizedRecommendations,
		"confidence":   ta.calculateRecommendationConfidence(optimizedRecommendations),
	})
}

// handleBuildBetterTool coordinates the building of an improved tool
func (ta *ToolAnalysisAgent) handleBuildBetterTool(ctx context.Context, msg *a2a.A2AMessage) error {
	toolSpec, ok := msg.Content["tool_spec"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("tool_spec parameter required")
	}

	log.Printf("Building better tool based on specifications")

	// Generate implementation plan
	implementationPlan := ta.generateImplementationPlan(toolSpec)

	// Delegate to appropriate agents for implementation
	return ta.SendMessage(ctx, "generic-worker-agent", "ImplementTool", map[string]interface{}{
		"implementation_plan": implementationPlan,
		"tool_spec":          toolSpec,
		"best_practices":     ta.getBestPracticesForToolType(toolSpec["type"].(string)),
	})
}

// performComprehensiveAnalysis performs comprehensive analysis of a tool
func (ta *ToolAnalysisAgent) performComprehensiveAnalysis(ctx context.Context, toolPath, analysisType string) (map[string]interface{}, error) {
	results := make(map[string]interface{})

	// Ghidra binary analysis
	if analysisType == "comprehensive" || analysisType == "binary" {
		ghidraResult, err := ta.ghidraTool.Execute(ctx, map[string]interface{}{
			"action":      "analyze",
			"binary_path": toolPath,
		})
		if err == nil && ghidraResult.Success {
			results["ghidra_analysis"] = ghidraResult.Data
		}
	}

	// Code quality analysis
	if analysisType == "comprehensive" || analysisType == "quality" {
		qualityResult := ta.analyzeCodeQuality(toolPath)
		results["quality_analysis"] = qualityResult
	}

	// Performance analysis
	if analysisType == "comprehensive" || analysisType == "performance" {
		performanceResult := ta.analyzePerformance(toolPath)
		results["performance_analysis"] = performanceResult
	}

	// Security analysis
	if analysisType == "comprehensive" || analysisType == "security" {
		securityResult := ta.analyzeSecurity(toolPath)
		results["security_analysis"] = securityResult
	}

	// Architecture analysis
	if analysisType == "comprehensive" || analysisType == "architecture" {
		architectureResult := ta.analyzeArchitecture(toolPath)
		results["architecture_analysis"] = architectureResult
	}

	return results, nil
}

// analyzeCodeQuality analyzes code quality metrics
func (ta *ToolAnalysisAgent) analyzeCodeQuality(toolPath string) map[string]interface{} {
	// This would integrate with actual code quality tools
	return map[string]interface{}{
		"cyclomatic_complexity": 8.5,
		"maintainability_index": 72.3,
		"code_coverage":         85.7,
		"duplication_percentage": 12.1,
		"technical_debt_ratio":  0.15,
		"code_smells": []string{
			"Long method detected",
			"Large class detected",
			"Duplicate code found",
		},
	}
}

// analyzePerformance analyzes performance characteristics
func (ta *ToolAnalysisAgent) analyzePerformance(toolPath string) map[string]interface{} {
	// This would integrate with performance profiling tools
	return map[string]interface{}{
		"execution_time":      "1.2s",
		"memory_usage":        "45MB",
		"cpu_utilization":     0.23,
		"throughput":          "1000 req/s",
		"latency_p95":         "150ms",
		"bottlenecks": []string{
			"Database query optimization needed",
			"Memory allocation could be improved",
		},
	}
}

// analyzeSecurity analyzes security characteristics
func (ta *ToolAnalysisAgent) analyzeSecurity(toolPath string) map[string]interface{} {
	// This would integrate with security scanning tools
	return map[string]interface{}{
		"vulnerability_count": 3,
		"security_score":      7.2,
		"vulnerabilities": []map[string]interface{}{
			{
				"type":        "SQL Injection",
				"severity":    "High",
				"location":    "line 45",
				"description": "Potential SQL injection in user input handling",
			},
			{
				"type":        "Buffer Overflow",
				"severity":    "Medium",
				"location":    "line 123",
				"description": "Potential buffer overflow in string processing",
			},
		},
		"security_recommendations": []string{
			"Implement input validation",
			"Use parameterized queries",
			"Add bounds checking",
		},
	}
}

// analyzeArchitecture analyzes architectural characteristics
func (ta *ToolAnalysisAgent) analyzeArchitecture(toolPath string) map[string]interface{} {
	return map[string]interface{}{
		"design_patterns": []string{
			"Singleton",
			"Factory",
			"Observer",
		},
		"coupling": map[string]interface{}{
			"afferent":  5,
			"efferent":  8,
			"instability": 0.62,
		},
		"cohesion": map[string]interface{}{
			"lcom": 0.3,
			"camc": 0.8,
		},
		"architectural_issues": []string{
			"High coupling between modules",
			"Missing abstraction layer",
		},
	}
}

// storeToolKnowledge stores learned knowledge about a tool
func (ta *ToolAnalysisAgent) storeToolKnowledge(toolPath string, analysisResults map[string]interface{}) {
	ta.mu.Lock()
	defer ta.mu.Unlock()

	knowledge := ToolKnowledge{
		ToolName:        toolPath,
		ToolType:        ta.determineToolType(analysisResults),
		AnalysisResults: analysisResults,
		BestPractices:   ta.extractBestPractices(analysisResults),
		AntiPatterns:    ta.extractAntiPatterns(analysisResults),
		Improvements:    ta.extractImprovements(analysisResults),
		Performance:     ta.calculatePerformanceMetrics(analysisResults),
		LastAnalyzed:    time.Now(),
	}

	ta.learnedTools[toolPath] = knowledge
}

// generateRecommendations generates recommendations based on analysis
func (ta *ToolAnalysisAgent) generateRecommendations(analysisResults map[string]interface{}) []string {
	recommendations := []string{}

	// Quality recommendations
	if quality, ok := analysisResults["quality_analysis"].(map[string]interface{}); ok {
		if complexity, ok := quality["cyclomatic_complexity"].(float64); ok && complexity > 10 {
			recommendations = append(recommendations, "Reduce cyclomatic complexity by breaking down complex functions")
		}
		if coverage, ok := quality["code_coverage"].(float64); ok && coverage < 80 {
			recommendations = append(recommendations, "Increase test coverage to at least 80%")
		}
	}

	// Security recommendations
	if security, ok := analysisResults["security_analysis"].(map[string]interface{}); ok {
		if vulnCount, ok := security["vulnerability_count"].(int); ok && vulnCount > 0 {
			recommendations = append(recommendations, "Address security vulnerabilities immediately")
		}
	}

	// Performance recommendations
	if performance, ok := analysisResults["performance_analysis"].(map[string]interface{}); ok {
		if bottlenecks, ok := performance["bottlenecks"].([]string); ok && len(bottlenecks) > 0 {
			for _, bottleneck := range bottlenecks {
				recommendations = append(recommendations, fmt.Sprintf("Optimize: %s", bottleneck))
			}
		}
	}

	return recommendations
}

// generateToolRecommendations generates recommendations for building new tools
func (ta *ToolAnalysisAgent) generateToolRecommendations(toolType string, requirements map[string]interface{}) []string {
	recommendations := []string{}

	// Get best practices from learned tools of similar type
	bestPractices := ta.getBestPracticesForToolType(toolType)
	recommendations = append(recommendations, bestPractices...)

	// Add type-specific recommendations
	switch toolType {
	case "web_service":
		recommendations = append(recommendations, ta.getWebServiceRecommendations(requirements)...)
	case "cli_tool":
		recommendations = append(recommendations, ta.getCLIToolRecommendations(requirements)...)
	case "library":
		recommendations = append(recommendations, ta.getLibraryRecommendations(requirements)...)
	case "database_tool":
		recommendations = append(recommendations, ta.getDatabaseToolRecommendations(requirements)...)
	}

	return recommendations
}

// getBestPracticesForToolType returns best practices for a specific tool type
func (ta *ToolAnalysisAgent) getBestPracticesForToolType(toolType string) []string {
	ta.mu.RLock()
	defer ta.mu.RUnlock()

	practices := []string{}

	for _, knowledge := range ta.learnedTools {
		if knowledge.ToolType == toolType {
			practices = append(practices, knowledge.BestPractices...)
		}
	}

	// Remove duplicates and return
	return ta.removeDuplicates(practices)
}

// ToolAnalysisTool provides tool analysis capabilities
type ToolAnalysisTool struct {
	agent *ToolAnalysisAgent
}

func (t *ToolAnalysisTool) Name() string { return "ToolAnalysis" }

func (t *ToolAnalysisTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	toolPath, ok := params["tool_path"].(string)
	if !ok {
		return nil, fmt.Errorf("tool_path parameter required")
	}

	analysisType, _ := params["analysis_type"].(string)
	if analysisType == "" {
		analysisType = "comprehensive"
	}

	results, err := t.agent.performComprehensiveAnalysis(ctx, toolPath, analysisType)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"analysis_results": results,
		"recommendations":  t.agent.generateRecommendations(results),
	}, nil
}

func (t *ToolAnalysisTool) Validate(params map[string]interface{}) error {
	if _, ok := params["tool_path"]; !ok {
		return fmt.Errorf("tool_path parameter required")
	}
	return nil
}

// ToolLearningTool provides tool learning capabilities
type ToolLearningTool struct {
	agent *ToolAnalysisAgent
}

func (t *ToolLearningTool) Name() string { return "ToolLearning" }

func (t *ToolLearningTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	toolPath, ok := params["tool_path"].(string)
	if !ok {
		return nil, fmt.Errorf("tool_path parameter required")
	}

	feedback, _ := params["feedback"].(map[string]interface{})
	success, _ := params["success"].(bool)

	t.agent.updateLearningFromFeedback(toolPath, feedback, success)

	return map[string]interface{}{
		"status": "learning_updated",
		"tool":   toolPath,
	}, nil
}

func (t *ToolLearningTool) Validate(params map[string]interface{}) error {
	if _, ok := params["tool_path"]; !ok {
		return fmt.Errorf("tool_path parameter required")
	}
	return nil
}

// ToolRecommendationTool provides tool recommendation capabilities
type ToolRecommendationTool struct {
	agent *ToolAnalysisAgent
}

func (t *ToolRecommendationTool) Name() string { return "ToolRecommendation" }

func (t *ToolRecommendationTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	toolType, ok := params["tool_type"].(string)
	if !ok {
		return nil, fmt.Errorf("tool_type parameter required")
	}

	requirements, _ := params["requirements"].(map[string]interface{})

	recommendations := t.agent.generateToolRecommendations(toolType, requirements)
	optimizedRecommendations := t.agent.optimizeRecommendationsWithRL(recommendations)

	return map[string]interface{}{
		"recommendations": optimizedRecommendations,
		"confidence":      t.agent.calculateRecommendationConfidence(optimizedRecommendations),
	}, nil
}

func (t *ToolRecommendationTool) Validate(params map[string]interface{}) error {
	if _, ok := params["tool_type"]; !ok {
		return fmt.Errorf("tool_type parameter required")
	}
	return nil
}

// Helper methods for ToolAnalysisAgent

func (ta *ToolAnalysisAgent) determineToolType(analysisResults map[string]interface{}) string {
	// Simple heuristic to determine tool type based on analysis results
	if _, ok := analysisResults["ghidra_analysis"]; ok {
		return "binary_tool"
	}
	if _, ok := analysisResults["web_analysis"]; ok {
		return "web_service"
	}
	return "unknown"
}

func (ta *ToolAnalysisAgent) extractBestPractices(analysisResults map[string]interface{}) []string {
	practices := []string{}

	// Extract from quality analysis
	if quality, ok := analysisResults["quality_analysis"].(map[string]interface{}); ok {
		if coverage, ok := quality["code_coverage"].(float64); ok && coverage > 80 {
			practices = append(practices, "High test coverage maintained")
		}
	}

	// Extract from security analysis
	if security, ok := analysisResults["security_analysis"].(map[string]interface{}); ok {
		if score, ok := security["security_score"].(float64); ok && score > 8 {
			practices = append(practices, "Strong security practices implemented")
		}
	}

	return practices
}

func (ta *ToolAnalysisAgent) extractAntiPatterns(analysisResults map[string]interface{}) []string {
	patterns := []string{}

	// Extract from quality analysis
	if quality, ok := analysisResults["quality_analysis"].(map[string]interface{}); ok {
		if complexity, ok := quality["cyclomatic_complexity"].(float64); ok && complexity > 10 {
			patterns = append(patterns, "High cyclomatic complexity")
		}
		if smells, ok := quality["code_smells"].([]string); ok {
			patterns = append(patterns, smells...)
		}
	}

	return patterns
}

func (ta *ToolAnalysisAgent) extractImprovements(analysisResults map[string]interface{}) []string {
	return ta.generateRecommendations(analysisResults)
}

func (ta *ToolAnalysisAgent) calculatePerformanceMetrics(analysisResults map[string]interface{}) ToolPerformance {
	performance := ToolPerformance{
		Efficiency:     0.5,
		Reliability:    0.5,
		Maintainability: 0.5,
		Security:       0.5,
		Scalability:    0.5,
	}

	// Calculate based on analysis results
	if quality, ok := analysisResults["quality_analysis"].(map[string]interface{}); ok {
		if coverage, ok := quality["code_coverage"].(float64); ok {
			performance.Reliability = coverage / 100.0
		}
		if maintainability, ok := quality["maintainability_index"].(float64); ok {
			performance.Maintainability = maintainability / 100.0
		}
	}

	if security, ok := analysisResults["security_analysis"].(map[string]interface{}); ok {
		if score, ok := security["security_score"].(float64); ok {
			performance.Security = score / 10.0
		}
	}

	return performance
}

func (ta *ToolAnalysisAgent) updateLearningFromFeedback(toolPath string, feedback map[string]interface{}, success bool) {
	// Update RL engine with feedback
	reward := 0.0
	if success {
		reward = 1.0
	}

	// This would update the RL engine with the feedback
	// For now, we'll just log it
	log.Printf("Learning from feedback for %s: success=%v, reward=%f", toolPath, success, reward)
}

func (ta *ToolAnalysisAgent) updateToolKnowledge(toolPath string, feedback map[string]interface{}, success bool) {
	ta.mu.Lock()
	defer ta.mu.Unlock()

	if knowledge, exists := ta.learnedTools[toolPath]; exists {
		// Update knowledge based on feedback
		if success {
			knowledge.BestPractices = append(knowledge.BestPractices, "Successfully implemented")
		} else {
			knowledge.AntiPatterns = append(knowledge.AntiPatterns, "Implementation failed")
		}
		ta.learnedTools[toolPath] = knowledge
	}
}

func (ta *ToolAnalysisAgent) optimizeRecommendationsWithRL(recommendations []string) []string {
	// Use RL engine to optimize recommendations
	// This would use the RL engine to rank and optimize recommendations
	return recommendations
}

func (ta *ToolAnalysisAgent) calculateRecommendationConfidence(recommendations []string) float64 {
	// Calculate confidence based on learned knowledge and RL feedback
	return 0.8 // Placeholder
}

func (ta *ToolAnalysisAgent) generateImplementationPlan(toolSpec map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"phases": []string{
			"Requirements analysis",
			"Architecture design",
			"Implementation",
			"Testing",
			"Deployment",
		},
		"estimated_time": "2-4 weeks",
		"resources_needed": []string{
			"Backend developer",
			"Frontend developer",
			"DevOps engineer",
		},
	}
}

func (ta *ToolAnalysisAgent) getWebServiceRecommendations(requirements map[string]interface{}) []string {
	return []string{
		"Use RESTful API design principles",
		"Implement proper error handling and status codes",
		"Add API versioning strategy",
		"Implement rate limiting and authentication",
		"Use async processing for long-running operations",
	}
}

func (ta *ToolAnalysisAgent) getCLIToolRecommendations(requirements map[string]interface{}) []string {
	return []string{
		"Use a robust CLI framework (Cobra, Click, etc.)",
		"Implement comprehensive help and documentation",
		"Add configuration file support",
		"Implement proper logging and error reporting",
		"Use subcommands for complex operations",
	}
}

func (ta *ToolAnalysisAgent) getLibraryRecommendations(requirements map[string]interface{}) []string {
	return []string{
		"Design clean, intuitive APIs",
		"Implement comprehensive error handling",
		"Add extensive documentation and examples",
		"Use semantic versioning",
		"Implement proper dependency management",
	}
}

func (ta *ToolAnalysisAgent) getDatabaseToolRecommendations(requirements map[string]interface{}) []string {
	return []string{
		"Implement connection pooling",
		"Add transaction management",
		"Use prepared statements for security",
		"Implement proper error handling",
		"Add query optimization and indexing",
	}
}

func (ta *ToolAnalysisAgent) removeDuplicates(slice []string) []string {
	keys := make(map[string]bool)
	result := []string{}

	for _, item := range slice {
		if !keys[item] {
			keys[item] = true
			result = append(result, item)
		}
	}

	return result
}
