package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"

	"github.com/bougou/go-ipmi"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// ServerConfig represents a single server configuration
type ServerConfig struct {
	Name         string `json:"name"`
	IP           string `json:"ip"`
	SSHPort      int    `json:"ssh_port"`
	IPMIHost     string `json:"ipmi_host"`
	IPMIUser     string `json:"ipmi_user"`
	IPMIPassword string `json:"ipmi_password"`
}

// UserConfig represents a user configuration
type UserConfig struct {
	Name  string `json:"name"`
	Phone string `json:"phone"`
}

// VPNConfig represents VPN configuration
type VPNConfig struct {
	Name string `json:"name"`
	IP   string `json:"ip"`
}

// SMSConfig represents SMS configuration
type SMSConfig struct {
	SignName     string `json:"sign_name"`
	TemplateCode string `json:"template_code"`
}

// AliyunConfig represents Aliyun credentials
type AliyunConfig struct {
	AccessKeyID     string    `json:"access_key_id"`
	AccessKeySecret string    `json:"access_key_secret"`
	SMS             SMSConfig `json:"sms"`
}

// Config represents the top-level configuration
type Config struct {
	VPN     VPNConfig      `json:"vpn"`
	Aliyun  AliyunConfig   `json:"aliyun"`
	Users   []UserConfig   `json:"users"`
	Servers []ServerConfig `json:"servers"`
}

// Global configuration
var (
	appConfig Config
	configMu  sync.RWMutex
)

// TokenData stores token information
type TokenData struct {
	Phone     string
	ExpiresAt time.Time
}

// Auth storage
var (
	authTokens = make(map[string]TokenData) // token -> TokenData
	authMu     sync.RWMutex
)

// AuthRequest defines the payload for authentication
type AuthRequest struct {
	Phone string `json:"phone" binding:"required"`
	Code  string `json:"code"`
}

// IPMIRequest defines the payload for IPMI operations
type IPMIRequest struct {
	ServerName string `json:"server_name" binding:"required"`
	Action     string `json:"action"` // "on", "off", "cycle", "reset", "soft" (for control)
}

// NetworkRequest defines the payload for Network checks
type NetworkRequest struct {
	ServerName string `json:"server_name" binding:"required"`
}

func loadConfig() error {
	configMu.Lock()
	defer configMu.Unlock()

	data, err := os.ReadFile("server_info.json")
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, &appConfig); err != nil {
		return err
	}
	return nil
}

func getServerByName(name string) (ServerConfig, bool) {
	configMu.RLock()
	defer configMu.RUnlock()

	for _, s := range appConfig.Servers {
		if s.Name == name {
			return s, true
		}
	}
	return ServerConfig{}, false
}

func checkPing(ip string) bool {
	// Use system ping command
	// -c 1: send 1 packet
	// -W 1: wait 1 second for response
	cmd := exec.Command("ping", "-c", "1", "-W", "1", ip)
	err := cmd.Run()
	return err == nil
}

func main() {
	// Load configuration
	if err := loadConfig(); err != nil {
		log.Printf("Warning: Could not load server_info.json: %v", err)
		// Create a dummy config if file doesn't exist to avoid crash
		appConfig = Config{Servers: []ServerConfig{}}
	} else {
		fmt.Printf("Loaded %d servers from server_info.json\n", len(appConfig.Servers))
	}

	r := gin.Default()

	// Configure CORS to allow frontend access
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"}, // In production, replace with specific frontend origin
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	api := r.Group("/api")

	// Public Auth Routes
	api.POST("/auth/send-code", sendCodeHandler)
	api.POST("/auth/login", loginHandler)

	// Protected Routes
	protected := api.Group("/")
	protected.Use(authMiddleware())
	{
		// Health check
		protected.GET("/ping", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "pong"})
		})

		// Get list of servers
		protected.GET("/servers", getServers)

		// IPMI Routes
		protected.POST("/ipmi/status", getPowerStatus)
		protected.POST("/ipmi/control", setPowerState)

		// Network Routes
		protected.POST("/network/status", checkNetworkStatus)

		// VPN Status Route
		protected.GET("/vpn/status", func(c *gin.Context) {
			configMu.RLock()
			targetIP := appConfig.VPN.IP
			targetName := appConfig.VPN.Name
			configMu.RUnlock()

			if targetIP == "" {
				c.JSON(http.StatusOK, gin.H{
					"status": "unknown",
					"error":  "VPN not configured",
				})
				return
			}

			online := checkPing(targetIP)
			status := "offline"
			if online {
				status = "online"
			}
			c.JSON(http.StatusOK, gin.H{
				"status": status,
				"ip":     targetIP,
				"name":   targetName,
			})
		})
	}

	fmt.Println("Server starting on :23080")
	r.Run(":23080")
}

// Auth Handlers and Middleware

func sendCodeHandler(c *gin.Context) {
	var req AuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	configMu.RLock()
	allowed := false
	var userName string
	for _, u := range appConfig.Users {
		if u.Phone == req.Phone {
			allowed = true
			userName = u.Name
			break
		}
	}
	configMu.RUnlock()

	if !allowed {
		c.JSON(http.StatusForbidden, gin.H{"error": "无效的手机号"})
		return
	}

	// Send SMS via Aliyun
	if err := SendSmsCode(req.Phone); err != nil {
		log.Printf("Failed to send SMS to %s: %v", req.Phone, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send verification code"})
		return
	}

	fmt.Printf(">>> SMS sent to %s (%s) <<<\n", userName, req.Phone)

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Verification code sent"})
}

func loginHandler(c *gin.Context) {
	var req AuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify code via Aliyun
	valid, err := CheckSmsCode(req.Phone, req.Code)
	if err != nil {
		log.Printf("Verification error for %s: %v", req.Phone, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Verification failed"})
		return
	}

	if !valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid verification code"})
		return
	}

	// Generate token
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}
	token := hex.EncodeToString(b)

	authMu.Lock()
	authTokens[token] = TokenData{
		Phone:     req.Phone,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour), // Valid for 7 days
	}
	authMu.Unlock()

	c.JSON(http.StatusOK, gin.H{"success": true, "token": token})
}

func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			return
		}

		authMu.RLock()
		tokenData, exists := authTokens[token]
		authMu.RUnlock()

		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}

		if time.Now().After(tokenData.ExpiresAt) {
			authMu.Lock()
			delete(authTokens, token)
			authMu.Unlock()
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token expired"})
			return
		}

		c.Next()
	}
}

// getServers returns the list of configured servers (without sensitive info)
func getServers(c *gin.Context) {
	configMu.RLock()
	defer configMu.RUnlock()
	c.JSON(http.StatusOK, appConfig.Servers)
}

// Helper to create and connect an IPMI client
func getIPMIClient(host, user, password string) (*ipmi.Client, error) {
	h, p, err := net.SplitHostPort(host)
	port := 623
	if err == nil {
		host = h
		if portNum, err := strconv.Atoi(p); err == nil {
			port = portNum
		}
	}

	client, err := ipmi.NewClient(host, port, user, password)
	if err != nil {
		return nil, err
	}

	if err := client.Connect(context.Background()); err != nil {
		return nil, err
	}
	return client, nil
}

// getPowerStatus checks the power status of a server via IPMI
func getPowerStatus(c *gin.Context) {
	var req IPMIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	server, found := getServerByName(req.ServerName)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Server not found"})
		return
	}

	if server.IPMIHost == "" {
		c.JSON(http.StatusOK, gin.H{
			"status": "unknown",
			"error":  "IPMI not configured",
		})
		return
	}

	client, err := getIPMIClient(server.IPMIHost, server.IPMIUser, server.IPMIPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status": "unknown",
			"error":  fmt.Sprintf("Failed to connect to IPMI: %v", err),
		})
		return
	}
	defer client.Close(context.Background())

	status, err := client.GetChassisStatus(context.Background())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status": "unknown",
			"error":  fmt.Sprintf("Failed to get chassis status: %v", err),
		})
		return
	}

	// PowerState bit 0 indicates power status (1=on, 0=off)
	finalStatus := "off"
	if status.PowerIsOn {
		finalStatus = "on"
	}

	c.JSON(http.StatusOK, gin.H{
		"status": finalStatus,
		"raw":    status,
	})
}

// setPowerState controls the power state of a server via IPMI
func setPowerState(c *gin.Context) {
	var req IPMIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Action == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "action is required (on, off, cycle, reset, soft)"})
		return
	}

	server, found := getServerByName(req.ServerName)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Server not found"})
		return
	}

	if server.IPMIHost == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "IPMI not configured for this server",
		})
		return
	}

	// Map action to IPMI control command
	var controlType ipmi.ChassisControl
	switch req.Action {
	case "off":
		controlType = ipmi.ChassisControlPowerDown
	case "on":
		controlType = ipmi.ChassisControlPowerUp
	case "cycle":
		controlType = ipmi.ChassisControlPowerCycle
	case "reset":
		controlType = ipmi.ChassisControlHardReset
	case "soft":
		// 0x05: Initiate Soft-shutdown of OS via ACPI
		controlType = ipmi.ChassisControl(5)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid action"})
		return
	}

	client, err := getIPMIClient(server.IPMIHost, server.IPMIUser, server.IPMIPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Failed to connect to IPMI: %v", err),
		})
		return
	}
	defer client.Close(context.Background())

	// Execute command
	if _, err := client.ChassisControl(context.Background(), controlType); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Failed to execute power command: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("Power command '%s' sent successfully", req.Action),
	})
}

// checkNetworkStatus checks if the server is reachable via SSH (TCP)
func checkNetworkStatus(c *gin.Context) {
	var req NetworkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	server, found := getServerByName(req.ServerName)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Server not found"})
		return
	}

	// Determine port (default to 22 if not specified)
	port := "22"
	if server.SSHPort != 0 {
		port = strconv.Itoa(server.SSHPort)
	}

	// Check TCP connection to port
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(server.IP, port), 2*time.Second)
	status := "offline"
	if err == nil {
		status = "online"
		conn.Close()
	}

	c.JSON(http.StatusOK, gin.H{
		"status": status,
	})
}
