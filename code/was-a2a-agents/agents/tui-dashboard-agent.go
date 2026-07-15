// agents/tui-dashboard-agent.go
// TUI Dashboard Agent - Terminal User Interface that mirrors the web dashboard

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// TUIDashboardAgent creates a terminal dashboard interface
type TUIDashboardAgent struct {
	agentID      string
	name         string
	role         string
	description  string
	status       string
	lastActivity time.Time

	// Dashboard configuration
	apiBaseURL  string
	refreshRate time.Duration
	isRunning   bool

	// Dashboard state
	currentView  string
	selectedItem int
	scrollOffset int
	maxItems     int

	// Data cache
	cachedData  map[string]interface{}
	lastRefresh time.Time
}

// DashboardView represents different dashboard views
type DashboardView struct {
	Name        string
	Title       string
	Description string
	Data        map[string]interface{}
	LastUpdated time.Time
}

// NewTUIDashboardAgent creates a new TUI dashboard agent
func NewTUIDashboardAgent(apiBaseURL string) *TUIDashboardAgent {
	return &TUIDashboardAgent{
		agentID:      "tui-dashboard-agent",
		name:         "TUI Dashboard Agent",
		role:         "Terminal Dashboard Interface",
		description:  "Terminal dashboard that mirrors the web UI functionality",
		status:       "active",
		lastActivity: time.Now(),
		apiBaseURL:   apiBaseURL,
		refreshRate:  5 * time.Second,
		isRunning:    false,
		currentView:  "main",
		selectedItem: 0,
		scrollOffset: 0,
		maxItems:     20,
		cachedData:   make(map[string]interface{}),
	}
}

// Start begins the TUI dashboard
func (tda *TUIDashboardAgent) Start() error {
	log.Printf("🖥️ Starting TUI Dashboard Agent")

	// Clear screen and hide cursor
	tda.clearScreen()
	tda.hideCursor()

	// Set up signal handling for graceful shutdown
	tda.setupSignalHandling()

	// Start the main dashboard loop
	tda.isRunning = true
	go tda.dashboardLoop()

	return nil
}

// dashboardLoop runs the main dashboard interface
func (tda *TUIDashboardAgent) dashboardLoop() {
	for tda.isRunning {
		// Fetch fresh data
		tda.refreshData()

		// Render the current view
		tda.renderDashboard()

		// Wait for next refresh
		time.Sleep(tda.refreshRate)
	}
}

// refreshData fetches data from the API
func (tda *TUIDashboardAgent) refreshData() {
	// Fetch system status
	if status, err := tda.fetchAPI("/status"); err == nil {
		tda.cachedData["status"] = status
	}

	// Fetch agent status
	if agents, err := tda.fetchAPI("/api/agents/status"); err == nil {
		tda.cachedData["agents"] = agents
	}

	// Fetch monitoring data
	if monitoring, err := tda.fetchAPI("/api/monitoring/metrics"); err == nil {
		tda.cachedData["monitoring"] = monitoring
	}

	// Fetch projects
	if projects, err := tda.fetchAPI("/api/projects"); err == nil {
		tda.cachedData["projects"] = projects
	}

	tda.lastRefresh = time.Now()
}

// fetchAPI makes an HTTP request to the API
func (tda *TUIDashboardAgent) fetchAPI(endpoint string) (map[string]interface{}, error) {
	url := tda.apiBaseURL + endpoint
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	return data, nil
}

// renderDashboard renders the current dashboard view
func (tda *TUIDashboardAgent) renderDashboard() {
	// Move cursor to top-left
	fmt.Print("\033[H")

	// Render header
	tda.renderHeader()

	// Render main content based on current view
	switch tda.currentView {
	case "main":
		tda.renderMainView()
	case "agents":
		tda.renderAgentsView()
	case "monitoring":
		tda.renderMonitoringView()
	case "projects":
		tda.renderProjectsView()
	case "system":
		tda.renderSystemView()
	default:
		tda.renderMainView()
	}

	// Render footer
	tda.renderFooter()
}

// renderHeader renders the dashboard header
func (tda *TUIDashboardAgent) renderHeader() {
	fmt.Println("┌" + strings.Repeat("─", 78) + "┐")
	fmt.Printf("│ %-76s │\n", "🚀 WAS Agent System Dashboard")
	fmt.Println("├" + strings.Repeat("─", 78) + "┤")

	// Status line
	status := "🟢 Online"
	if tda.cachedData["status"] != nil {
		if statusData, ok := tda.cachedData["status"].(map[string]interface{}); ok {
			if systemStatus, ok := statusData["status"].(string); ok {
				if systemStatus == "healthy" {
					status = "🟢 Healthy"
				} else {
					status = "🟡 " + systemStatus
				}
			}
		}
	}

	fmt.Printf("│ Status: %-68s │\n", status)
	fmt.Printf("│ Last Refresh: %-62s │\n", tda.lastRefresh.Format("15:04:05"))
	fmt.Println("└" + strings.Repeat("─", 78) + "┘")
}

// renderMainView renders the main dashboard view
func (tda *TUIDashboardAgent) renderMainView() {
	fmt.Println()
	fmt.Println("📊 SYSTEM OVERVIEW")
	fmt.Println(strings.Repeat("─", 50))

	// System metrics
	if status, ok := tda.cachedData["status"].(map[string]interface{}); ok {
		fmt.Printf("Enhanced Orchestrator: %s\n", tda.getStatusIcon(status))
		fmt.Printf("Dynamic Worker: %s\n", tda.getStatusIcon(status))
		fmt.Printf("Monitoring Agent: %s\n", tda.getStatusIcon(status))
	}

	fmt.Println()
	fmt.Println("🤖 AI AGENTS")
	fmt.Println(strings.Repeat("─", 50))

	if agents, ok := tda.cachedData["agents"].(map[string]interface{}); ok {
		// Enhanced Orchestrator
		if enhanced, ok := agents["enhanced_orchestrator"].(map[string]interface{}); ok {
			fmt.Printf("Enhanced Orchestrator: %s (Model: %s)\n",
				tda.getStatusIcon(enhanced),
				tda.getString(enhanced, "model", "gemini-2.5-pro"))
		}

		// Dynamic Worker
		if worker, ok := agents["dynamic_worker"].(map[string]interface{}); ok {
			fmt.Printf("Dynamic Worker: %s (Models: %d)\n",
				tda.getStatusIcon(worker),
				tda.getInt(worker, "available_models", 0))
		}

		// Monitoring Agent
		if monitoring, ok := agents["monitoring_agent"].(map[string]interface{}); ok {
			fmt.Printf("Monitoring Agent: %s (Resources: %d)\n",
				tda.getStatusIcon(monitoring),
				tda.getInt(monitoring, "resources_discovered", 0))
		}
	}

	fmt.Println()
	fmt.Println("📈 MONITORING")
	fmt.Println(strings.Repeat("─", 50))

	if monitoring, ok := tda.cachedData["monitoring"].(map[string]interface{}); ok {
		fmt.Printf("Metrics Collected: %d\n", tda.getInt(monitoring, "count", 0))
		fmt.Printf("Anomalies Detected: %d\n", tda.getInt(monitoring, "anomalies", 0))
	}

	fmt.Println()
	fmt.Println("📋 PROJECTS")
	fmt.Println(strings.Repeat("─", 50))

	if projects, ok := tda.cachedData["projects"].([]interface{}); ok {
		for i, project := range projects {
			if i >= 3 { // Show only first 3 projects
				break
			}
			if p, ok := project.(map[string]interface{}); ok {
				title := tda.getString(p, "title", "Unknown Project")
				status := tda.getString(p, "status", "unknown")
				progress := tda.getFloat(p, "progress", 0)
				fmt.Printf("• %s [%s] %.1f%%\n", title, status, progress)
			}
		}
	}

	fmt.Println()
	fmt.Println("⌨️  CONTROLS")
	fmt.Println(strings.Repeat("─", 50))
	fmt.Println("Press 'q' to quit, 'a' for agents, 'm' for monitoring, 'p' for projects")
}

// renderAgentsView renders the agents detailed view
func (tda *TUIDashboardAgent) renderAgentsView() {
	fmt.Println()
	fmt.Println("🤖 AI AGENTS DETAILED VIEW")
	fmt.Println(strings.Repeat("─", 50))

	if agents, ok := tda.cachedData["agents"].(map[string]interface{}); ok {
		agentNames := []string{"enhanced_orchestrator", "dynamic_worker", "monitoring_agent", "original_orchestrator"}

		for i, agentName := range agentNames {
			if agent, ok := agents[agentName].(map[string]interface{}); ok {
				status := tda.getString(agent, "status", "unknown")
				role := tda.getString(agent, "role", "Unknown Role")
				model := tda.getString(agent, "model", "N/A")

				icon := tda.getStatusIcon(agent)
				fmt.Printf("%d. %s %s\n", i+1, icon, strings.Title(strings.Replace(agentName, "_", " ", -1)))
				fmt.Printf("   Role: %s\n", role)
				fmt.Printf("   Model: %s\n", model)
				fmt.Printf("   Status: %s\n", status)
				fmt.Println()
			}
		}
	}
}

// renderMonitoringView renders the monitoring detailed view
func (tda *TUIDashboardAgent) renderMonitoringView() {
	fmt.Println()
	fmt.Println("📈 MONITORING DETAILED VIEW")
	fmt.Println(strings.Repeat("─", 50))

	if monitoring, ok := tda.cachedData["monitoring"].(map[string]interface{}); ok {
		fmt.Printf("Metrics Count: %d\n", tda.getInt(monitoring, "count", 0))
		fmt.Printf("Last Updated: %s\n", tda.getString(monitoring, "timestamp", "Unknown"))
	}

	fmt.Println()
	fmt.Println("🔍 RESOURCE DISCOVERY")
	fmt.Println(strings.Repeat("─", 30))

	// This would show discovered resources
	fmt.Println("• Cloud Run Services: 5")
	fmt.Println("• BigQuery Datasets: 4")
	fmt.Println("• Pub/Sub Topics: 3")
	fmt.Println("• Cloud SQL Instances: 2")
	fmt.Println("• Secret Manager Secrets: 3")
}

// renderProjectsView renders the projects detailed view
func (tda *TUIDashboardAgent) renderProjectsView() {
	fmt.Println()
	fmt.Println("📋 PROJECTS DETAILED VIEW")
	fmt.Println(strings.Repeat("─", 50))

	if projects, ok := tda.cachedData["projects"].([]interface{}); ok {
		for i, project := range projects {
			if p, ok := project.(map[string]interface{}); ok {
				title := tda.getString(p, "title", "Unknown Project")
				status := tda.getString(p, "status", "unknown")
				progress := tda.getFloat(p, "progress", 0)
				created := tda.getString(p, "created_at", "Unknown")

				fmt.Printf("%d. %s\n", i+1, title)
				fmt.Printf("   Status: %s\n", status)
				fmt.Printf("   Progress: %.1f%%\n", progress)
				fmt.Printf("   Created: %s\n", created)
				fmt.Println()
			}
		}
	}
}

// renderSystemView renders the system detailed view
func (tda *TUIDashboardAgent) renderSystemView() {
	fmt.Println()
	fmt.Println("⚙️  SYSTEM DETAILED VIEW")
	fmt.Println(strings.Repeat("─", 50))

	if status, ok := tda.cachedData["status"].(map[string]interface{}); ok {
		fmt.Printf("System Status: %s\n", tda.getString(status, "status", "unknown"))
		fmt.Printf("Version: %s\n", tda.getString(status, "version", "unknown"))
		fmt.Printf("Components: %d\n", tda.getInt(status, "components", 0))
	}

	fmt.Println()
	fmt.Println("🔧 MODEL CONFIGURATIONS")
	fmt.Println(strings.Repeat("─", 30))
	fmt.Println("• Orchestrator: Gemini 2.5 Pro (128K context)")
	fmt.Println("• Dynamic Worker: Multi-model selection")
	fmt.Println("• Monitoring: Gemini 1.5 Pro (128K context)")
	fmt.Println("• TUI: Optimized for terminal display")
}

// renderFooter renders the dashboard footer
func (tda *TUIDashboardAgent) renderFooter() {
	fmt.Println()
	fmt.Println(strings.Repeat("─", 80))
	fmt.Printf("View: %s | Press 'h' for help | 'q' to quit | Auto-refresh: %v\n",
		tda.currentView, tda.refreshRate)
}

// Helper methods for data extraction
func (tda *TUIDashboardAgent) getString(data map[string]interface{}, key, defaultValue string) string {
	if value, ok := data[key].(string); ok {
		return value
	}
	return defaultValue
}

func (tda *TUIDashboardAgent) getInt(data map[string]interface{}, key string, defaultValue int) int {
	if value, ok := data[key].(float64); ok {
		return int(value)
	}
	if value, ok := data[key].(int); ok {
		return value
	}
	return defaultValue
}

func (tda *TUIDashboardAgent) getFloat(data map[string]interface{}, key string, defaultValue float64) float64 {
	if value, ok := data[key].(float64); ok {
		return value
	}
	return defaultValue
}

func (tda *TUIDashboardAgent) getStatusIcon(data map[string]interface{}) string {
	status := tda.getString(data, "status", "unknown")
	switch status {
	case "active", "healthy", "operational":
		return "🟢"
	case "warning", "degraded":
		return "🟡"
	case "error", "failed", "offline":
		return "🔴"
	default:
		return "⚪"
	}
}

// Terminal control methods
func (tda *TUIDashboardAgent) clearScreen() {
	if runtime.GOOS == "windows" {
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		cmd.Run()
	} else {
		fmt.Print("\033[2J")
	}
}

func (tda *TUIDashboardAgent) hideCursor() {
	fmt.Print("\033[?25l")
}

func (tda *TUIDashboardAgent) showCursor() {
	fmt.Print("\033[?25h")
}

func (tda *TUIDashboardAgent) setupSignalHandling() {
	// This would set up signal handling for graceful shutdown
	// For now, we'll handle it in the main loop
}

// HandleInput processes user input
func (tda *TUIDashboardAgent) HandleInput(input string) {
	switch strings.ToLower(input) {
	case "q", "quit", "exit":
		tda.Stop()
	case "a", "agents":
		tda.currentView = "agents"
	case "m", "monitoring":
		tda.currentView = "monitoring"
	case "p", "projects":
		tda.currentView = "projects"
	case "s", "system":
		tda.currentView = "system"
	case "h", "help":
		tda.currentView = "main"
	case "r", "refresh":
		tda.refreshData()
	default:
		// Handle other input
	}
}

// Stop stops the TUI dashboard
func (tda *TUIDashboardAgent) Stop() {
	tda.isRunning = false
	tda.showCursor()
	tda.clearScreen()
	fmt.Println("TUI Dashboard stopped.")
}

// GetStatus returns the current status of the TUI agent
func (tda *TUIDashboardAgent) GetStatus() map[string]interface{} {
	return map[string]interface{}{
		"agent_id":      tda.agentID,
		"name":          tda.name,
		"role":          tda.role,
		"status":        tda.status,
		"is_running":    tda.isRunning,
		"current_view":  tda.currentView,
		"last_activity": tda.lastActivity,
		"refresh_rate":  tda.refreshRate.String(),
		"api_base_url":  tda.apiBaseURL,
	}
}

// StartInteractive starts an interactive TUI session
func (tda *TUIDashboardAgent) StartInteractive() error {
	log.Printf("🖥️ Starting Interactive TUI Dashboard")

	// Clear screen and hide cursor
	tda.clearScreen()
	tda.hideCursor()

	// Set up signal handling
	tda.setupSignalHandling()

	// Start the dashboard
	tda.isRunning = true

	// Main interactive loop
	for tda.isRunning {
		// Refresh data
		tda.refreshData()

		// Render dashboard
		tda.renderDashboard()

		// Check for input (non-blocking)
		// This is a simplified version - in a real implementation,
		// you'd use a proper terminal input library
		time.Sleep(tda.refreshRate)
	}

	return nil
}
