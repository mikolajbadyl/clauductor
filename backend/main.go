package main

import (
	"crypto/rand"
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var buildVersion = "dev"

//go:embed all:frontend
var frontendFiles embed.FS

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Message struct {
	Type     string `json:"type"`
	Data     any    `json:"data,omitempty"`
	ClientID string `json:"clientId,omitempty"`
}

type Client struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

type ClientManager struct {
	clients map[string]*Client
	mu      sync.RWMutex
}

var manager = ClientManager{
	clients: make(map[string]*Client),
}

func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "service" {
		runServiceCommand(os.Args[2:])
		return
	}

	host := flag.String("host", "localhost", "Host to bind to (use 0.0.0.0 for all interfaces)")
	port := flag.String("port", "8080", "Port to listen on")
	mcpMode := flag.Bool("mcp", false, "Run as MCP stdio server (for Claude MCP integration)")
	flag.Parse()

	if *mcpMode {
		runMCPServer()
		return
	}

	r := gin.Default()
	r.Use(corsMiddleware())
	r.GET("/ws", wsHandler)
	r.POST("/run", runHandler)
	r.POST("/stop", stopHandler)
	r.POST("/api/permissions/notify", permissionNotifyHandler)
	r.GET("/api/permissions/:id", permissionPollHandler)
	r.POST("/api/permissions/:id/decision", permissionDecideHandler)
	r.GET("/projects", projectsHandler)
	r.POST("/projects/check", checkPathHandler)
	r.POST("/projects/create", createPathHandler)
	r.GET("/config", getConfigHandler)
	r.POST("/config", saveConfigHandler)
	r.GET("/projects/:encoded/sessions", sessionsHandler)
	r.GET("/projects/:encoded/sessions/:id", sessionHandler)
	r.POST("/api/sessions/:id/claim", sessionClaimHandler)
	r.GET("/api/sessions/:id/pending-permissions", pendingPermissionsHandler)
	r.GET("/api/version", func(c *gin.Context) {
		c.JSON(200, gin.H{"version": buildVersion})
	})
	r.GET("/api/active-sessions", activeSessionsHandler)
	r.GET("/api/claude-version", claudeVersionHandler)
	r.POST("/api/claude-update", claudeUpdateHandler)
	r.GET("/api/usage", usageHandler)
	r.GET("/api/profiles", getProfilesHandler)
	r.PUT("/api/profiles", saveProfilesHandler)
	r.PUT("/api/profiles/active", setActiveProfileHandler)
	r.GET("/api/prefs", getPrefsHandler)
	r.PUT("/api/prefs", savePrefsHandler)
	r.GET("/api/files", listFilesHandler)
	r.POST("/api/upload", uploadFileHandler)
	r.GET("/api/project-settings", getProjectSettingsHandler)
	r.POST("/api/project-settings", saveProjectSettingsHandler)

	frontendFS, err := fs.Sub(frontendFiles, "frontend")
	if err != nil {
		log.Println("No embedded frontend, running in API-only mode")
	} else {
		fileServer := http.FileServer(http.FS(frontendFS))
		r.NoRoute(func(c *gin.Context) {
			path := strings.TrimPrefix(c.Request.URL.Path, "/")

			if path != "" {
				if f, err := frontendFS.Open(path); err == nil {
					f.Close()
					fileServer.ServeHTTP(c.Writer, c.Request)
					return
				}
			}

			c.Request.URL.Path = "/"
			fileServer.ServeHTTP(c.Writer, c.Request)
		})
	}

	addr := *host + ":" + *port
	writePortFile(*port)
	log.Printf("Clauductor running on http://%s", addr)
	r.Run(addr)
}

func clauductorDir() string {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".clauductor")
	os.MkdirAll(dir, 0755)
	return dir
}

func writePortFile(port string) {
	path := filepath.Join(clauductorDir(), "port")
	os.WriteFile(path, []byte(port), 0644)
}

func readPortFile() string {
	home, _ := os.UserHomeDir()
	data, err := os.ReadFile(filepath.Join(home, ".clauductor", "port"))
	if err != nil {
		return "8080"
	}
	return strings.TrimSpace(string(data))
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, PUT, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

func wsHandler(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("WS upgrade error:", err)
		return
	}

	clientID := generateID()
	manager.Register(clientID, conn)
	log.Printf("Client connected: %s", clientID)

	client := manager.Get(clientID)
	if client != nil {
		client.mu.Lock()
		conn.WriteJSON(Message{Type: "connected", ClientID: clientID})
		client.mu.Unlock()
	}

	go func() {
		defer func() {
			manager.Unregister(clientID)
			conn.Close()
			log.Printf("Client disconnected: %s", clientID)
		}()
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	}()
}

func (m *ClientManager) Register(id string, conn *websocket.Conn) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.clients[id] = &Client{conn: conn}
}

func (m *ClientManager) Unregister(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.clients, id)
}

func (m *ClientManager) Get(id string) *Client {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.clients[id]
}

func (m *ClientManager) Send(clientID string, msg Message) {
	client := m.Get(clientID)
	if client == nil {
		return
	}
	client.mu.Lock()
	defer client.mu.Unlock()
	if err := client.conn.WriteJSON(msg); err != nil {
		log.Printf("Send error to %s: %v", clientID, err)
	}
}
