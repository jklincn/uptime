package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/bougou/go-ipmi"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// ServerConfig represents a single server configuration
type ServerConfig struct {
	Name         string `toml:"name" json:"name"`
	IP           string `toml:"ip" json:"ip"`
	SSHPort      int    `toml:"ssh_port" json:"ssh_port"`
	IPMIHost     string `toml:"ipmi_host" json:"-"`
	IPMIUser     string `toml:"ipmi_user" json:"-"`
	IPMIPassword string `toml:"ipmi_password" json:"-"`
}

// Config represents the top-level configuration
type Config struct {
	Servers []ServerConfig `toml:"servers"`
}

// Global configuration
var (
	appConfig Config
	configMu  sync.RWMutex
)

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

	data, err := os.ReadFile("server_info.toml")
	if err != nil {
		return err
	}

	if _, err := toml.Decode(string(data), &appConfig); err != nil {
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
		log.Printf("Warning: Could not load config.toml: %v", err)
		// Create a dummy config if file doesn't exist to avoid crash
		appConfig = Config{Servers: []ServerConfig{}}
	} else {
		fmt.Printf("Loaded %d servers from config.toml\n", len(appConfig.Servers))
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
	{
		// Health check
		api.GET("/ping", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "pong"})
		})

		// Get list of servers
		api.GET("/servers", getServers)

		// IPMI Routes
		api.POST("/ipmi/status", getPowerStatus)
		api.POST("/ipmi/control", setPowerState)

		// Network Routes
		api.POST("/network/status", checkNetworkStatus)

		// VPN Status Route
		api.GET("/vpn/status", func(c *gin.Context) {
			targetIP := "10.183.111.1"
			online := checkPing(targetIP)
			status := "offline"
			if online {
				status = "online"
			}
			c.JSON(http.StatusOK, gin.H{
				"status": status,
				"ip":     targetIP,
			})
		})
	}

	fmt.Println("Server starting on :23080")
	r.Run(":23080")
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
