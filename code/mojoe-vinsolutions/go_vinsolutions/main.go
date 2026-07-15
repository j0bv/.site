package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// LeadData represents a lead from the website
type LeadData struct {
	FirstName       string `json:"first_name" binding:"required"`
	LastName        string `json:"last_name" binding:"required"`
	Email           string `json:"email" binding:"required,email"`
	Phone           string `json:"phone" binding:"required"`
	VehicleInterest string `json:"vehicle_interest,omitempty"`
	Comments        string `json:"comments,omitempty"`
}

// AppointmentData represents an appointment request
type AppointmentData struct {
	FirstName     string `json:"first_name" binding:"required"`
	LastName      string `json:"last_name" binding:"required"`
	Email         string `json:"email" binding:"required,email"`
	Phone         string `json:"phone" binding:"required"`
	PreferredDate string `json:"preferred_date" binding:"required"`
	PreferredTime string `json:"preferred_time" binding:"required"`
	ServiceType   string `json:"service_type,omitempty"`
	Notes         string `json:"notes,omitempty"`
}

// CreditData represents a credit application
type CreditData struct {
	FirstName        string  `json:"first_name" binding:"required"`
	LastName         string  `json:"last_name" binding:"required"`
	Email            string  `json:"email" binding:"required,email"`
	Phone            string  `json:"phone" binding:"required"`
	SSN              string  `json:"ssn" binding:"required"`
	DateOfBirth      string  `json:"date_of_birth" binding:"required"`
	Address          string  `json:"address" binding:"required"`
	City             string  `json:"city" binding:"required"`
	State            string  `json:"state" binding:"required"`
	ZipCode          string  `json:"zip_code" binding:"required"`
	EmploymentStatus string  `json:"employment_status" binding:"required"`
	Employer         string  `json:"employer,omitempty"`
	MonthlyIncome    float64 `json:"monthly_income" binding:"required"`
	DownPayment      float64 `json:"down_payment,omitempty"`
	VehicleInterest  string  `json:"vehicle_interest,omitempty"`
}

// ApiResponse represents a standard API response
type ApiResponse struct {
	Success bool                   `json:"success"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data,omitempty"`
}

// VinsolutionsClient handles Vinsolutions API interactions
type VinsolutionsClient struct {
	BaseURL  string
	Username string
	Password string
}

// NewVinsolutionsClient creates a new Vinsolutions client
func NewVinsolutionsClient() *VinsolutionsClient {
	return &VinsolutionsClient{
		BaseURL:  "https://vinsolutions.app.coxautoinc.com/vinconnect",
		Username: "jvalentine13156",
		Password: "59febIlikepie!",
	}
}

// CreateLead creates a new lead in Vinsolutions
func (vc *VinsolutionsClient) CreateLead(leadData *LeadData) (*ApiResponse, error) {
	// Simulate Vinsolutions API call
	// In production, this would use actual Vinsolutions API or headless browser
	time.Sleep(100 * time.Millisecond)

	leadID := uuid.New().String()

	return &ApiResponse{
		Success: true,
		Message: "Lead created successfully in Vinsolutions",
		Data: map[string]interface{}{
			"lead_id":       leadID,
			"customer_name": fmt.Sprintf("%s %s", leadData.FirstName, leadData.LastName),
			"email":         leadData.Email,
			"phone":         leadData.Phone,
		},
	}, nil
}

// ScheduleAppointment schedules an appointment in Vinsolutions
func (vc *VinsolutionsClient) ScheduleAppointment(appointmentData *AppointmentData) (*ApiResponse, error) {
	// Simulate appointment scheduling
	time.Sleep(150 * time.Millisecond)

	appointmentID := uuid.New().String()

	return &ApiResponse{
		Success: true,
		Message: "Appointment scheduled successfully",
		Data: map[string]interface{}{
			"appointment_id":   appointmentID,
			"customer_name":    fmt.Sprintf("%s %s", appointmentData.FirstName, appointmentData.LastName),
			"appointment_date": appointmentData.PreferredDate,
			"appointment_time": appointmentData.PreferredTime,
			"service_type":     appointmentData.ServiceType,
		},
	}, nil
}

// ProcessCreditApplication processes a credit application in Vinsolutions
func (vc *VinsolutionsClient) ProcessCreditApplication(creditData *CreditData) (*ApiResponse, error) {
	// Simulate credit application processing
	time.Sleep(200 * time.Millisecond)

	applicationID := uuid.New().String()

	return &ApiResponse{
		Success: true,
		Message: "Credit application processed successfully",
		Data: map[string]interface{}{
			"application_id":    applicationID,
			"customer_name":     fmt.Sprintf("%s %s", creditData.FirstName, creditData.LastName),
			"status":            "Pending Review",
			"monthly_income":    creditData.MonthlyIncome,
			"employment_status": creditData.EmploymentStatus,
		},
	}, nil
}

// createLeadHandler handles lead creation requests
func createLeadHandler(c *gin.Context) {
	var leadData LeadData
	if err := c.ShouldBindJSON(&leadData); err != nil {
		c.JSON(http.StatusBadRequest, ApiResponse{
			Success: false,
			Message: fmt.Sprintf("Invalid request data: %v", err),
		})
		return
	}

	client := NewVinsolutionsClient()
	response, err := client.CreateLead(&leadData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ApiResponse{
			Success: false,
			Message: fmt.Sprintf("Error creating lead: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// scheduleAppointmentHandler handles appointment scheduling requests
func scheduleAppointmentHandler(c *gin.Context) {
	var appointmentData AppointmentData
	if err := c.ShouldBindJSON(&appointmentData); err != nil {
		c.JSON(http.StatusBadRequest, ApiResponse{
			Success: false,
			Message: fmt.Sprintf("Invalid request data: %v", err),
		})
		return
	}

	client := NewVinsolutionsClient()
	response, err := client.ScheduleAppointment(&appointmentData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ApiResponse{
			Success: false,
			Message: fmt.Sprintf("Error scheduling appointment: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// processCreditHandler handles credit application requests
func processCreditHandler(c *gin.Context) {
	var creditData CreditData
	if err := c.ShouldBindJSON(&creditData); err != nil {
		c.JSON(http.StatusBadRequest, ApiResponse{
			Success: false,
			Message: fmt.Sprintf("Invalid request data: %v", err),
		})
		return
	}

	client := NewVinsolutionsClient()
	response, err := client.ProcessCreditApplication(&creditData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ApiResponse{
			Success: false,
			Message: fmt.Sprintf("Error processing credit application: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// healthCheckHandler provides health check endpoint
func healthCheckHandler(c *gin.Context) {
	c.JSON(http.StatusOK, ApiResponse{
		Success: true,
		Message: "MoJoe Auto Vinsolutions API (Go) is healthy",
	})
}

func main() {
	// Set Gin to release mode for production
	gin.SetMode(gin.ReleaseMode)

	// Create Gin router
	r := gin.Default()

	// Add CORS middleware
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Health check endpoint
	r.GET("/health", healthCheckHandler)

	// API routes
	api := r.Group("/api")
	{
		api.POST("/create-lead", createLeadHandler)
		api.POST("/schedule-appointment", scheduleAppointmentHandler)
		api.POST("/process-credit", processCreditHandler)
	}

	// Start server
	fmt.Println("🚀 MoJoe Auto Vinsolutions API (Go) running on :8080")
	log.Fatal(r.Run(":8080"))
}
