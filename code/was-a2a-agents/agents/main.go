// agents/main.go
// Main entry point for the WAS Agent System

package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	log.Printf("🤖 WAS Agent System Starting...")
	log.Printf("📅 %s", time.Now().Format("2006-01-02 15:04:05"))

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Printf("🛑 Received shutdown signal")
	}()

	// Initialize project
	projectID := "axial-studio-470521-b9"

	// Create integrated agent system with optimized model configurations
	integratedSystem, err := NewIntegratedAgentSystem(projectID)
	if err != nil {
		log.Fatalf("❌ Failed to initialize integrated agent system: %v", err)
	}

	// Start the integrated agent system
	if err := integratedSystem.Start(); err != nil {
		log.Fatalf("❌ Integrated agent system failed: %v", err)
	}

	log.Printf("🚀 WAS Integrated Agent System started successfully")
	log.Printf("📊 Enhanced Orchestrator: Gemini 2.5 Pro")
	log.Printf("🤖 Dynamic Worker: Multi-model selection")
	log.Printf("📈 Monitoring Agent: Gemini 1.5 Pro")
	log.Printf("🌐 Webhook server running on :8080")

	log.Printf("✅ WAS Agent System shutdown complete")
}
