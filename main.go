package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/devdeck-app/devdeck-server/commands"
	"github.com/devdeck-app/devdeck-server/services"
	"github.com/fsnotify/fsnotify"
	"github.com/gorilla/websocket"
	"github.com/spf13/viper"
)

type EventType string

const (
	FetchCommands = EventType("fetch_commands")
	RunCommand    = EventType("run")
)

type CommandMessage struct {
	UUID   string `json:"uuid"`
	Action string `json:"action"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var layout services.Layout
var cmds []commands.Command

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept")

	clientIP := r.RemoteAddr
	services.Info("Health check from %s", clientIP)

	// Log user agent to help identify connection types
	userAgent := r.Header.Get("User-Agent")
	if userAgent != "" {
		services.Debug("User-Agent: %s", userAgent)
	}

	// Handle preflight OPTIONS request
	if r.Method == "OPTIONS" {
		services.Debug("Received OPTIONS request from %s", clientIP)
		w.WriteHeader(http.StatusOK)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status": "ok", "server": "DevDeck"}`)

	services.Info("Health check successful for %s", clientIP)
}

func websocketHandler(w http.ResponseWriter, r *http.Request) {
	clientIP := r.RemoteAddr
	services.Info("WebSocket connection request from %s", clientIP)

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		services.Error("Error upgrading connection from %s: %v", clientIP, err)
		return
	}
	defer conn.Close()

	// Log connection details
	services.Info("WebSocket client connected from %s", clientIP)
	if ua := r.Header.Get("User-Agent"); ua != "" {
		services.Debug("WebSocket client User-Agent: %s", ua)
	}

	defer func() {
		if r := recover(); r != nil {
			services.Error("Recovered from panic in websocket handler: %v", r)
		}
	}()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				services.Warn("WebSocket closed unexpectedly: %v", err)
			} else {
				services.Info("WebSocket closed by client %s", clientIP)
			}
			break
		}

		// Log received message
		msgStr := string(message)
		if len(msgStr) > 200 {
			services.Debug("Received message from %s: %s...(truncated)", clientIP, msgStr[:200])
		} else {
			services.Debug("Received message from %s: %s", clientIP, msgStr)
		}

		var request map[string]any
		if err := json.Unmarshal(message, &request); err != nil {
			services.Error("Invalid JSON received from %s: %v", clientIP, err)
			services.SendMessage(conn, []byte(`{"error": "Invalid JSON format", "success": false}`))
			continue
		}

		eventType, ok := request["type"].(string)
		if !ok {
			services.Error("Missing or invalid 'type' field in request from %s", clientIP)
			services.SendMessage(conn, []byte(`{"error": "Missing or invalid 'type' field"}`))
			continue
		}

		services.Info("Processing event type '%s' from %s", eventType, clientIP)

		switch eventType {
		case "init":
			services.Info("Processing 'init' request from %s", clientIP)
			var rootCommands []commands.Command
			for _, c := range cmds {
				if c.Main {
					rootCommands = append(rootCommands, c)
				}
			}
			services.Debug("Sending %d root commands and layout config to client", len(rootCommands))
			response, _ := json.Marshal(map[string]any{
				"layout":   layout,
				"commands": rootCommands,
			})
			services.SendMessage(conn, response)
			services.Info("Init response sent to client %s", clientIP)
		case "switch_context":
			contextName, ok := request["context"].(string)
			if !ok {
				services.Error("Invalid context name in switch_context request from %s", clientIP)
				services.SendMessage(conn, []byte(`{"error": "Invalid context name", "success": false}`))
				continue
			}

			services.Info("Switching to context '%s' for client %s", contextName, clientIP)

			contextCommands := []commands.Command{
				{
					UUID:    "back",
					Icon:    "arrow-back-outline",
					Type:    "context",
					Context: "main",
				},
			}

			count := 0
			for _, cmd := range cmds {
				if cmd.Context == contextName && !cmd.Main {
					contextCommands = append(contextCommands, cmd)
					count++
				}
			}

			services.Debug("Found %d commands for context '%s'", count, contextName)

			response, _ := json.Marshal(map[string]any{
				"commands": contextCommands,
			})
			services.SendMessage(conn, response)
			services.Info("Context '%s' commands sent to client %s", contextName, clientIP)
		case "fetch_commands":
			services.Info("Fetching root commands for client %s", clientIP)
			var rootCommands []commands.Command
			for _, c := range cmds {
				if c.Main {
					rootCommands = append(rootCommands, c)
				}
			}

			services.Debug("Sending %d root commands to client", len(rootCommands))
			response, _ := json.Marshal(map[string]any{
				"commands": rootCommands,
			})
			services.SendMessage(conn, response)
			services.Info("Root commands sent to client %s", clientIP)
		case "run":
			contextName, contextOk := request["context"].(string)
			uuid, uuidOk := request["uuid"].(string)

			if !contextOk || !uuidOk {
				services.Error("Missing context or UUID in run command from %s", clientIP)
				services.SendMessage(conn, []byte(`{"error": "Missing required parameters", "success": false}`))
				continue
			}

			services.Info("Running command UUID '%s' in context '%s' for client %s", uuid, contextName, clientIP)

			var command commands.Command
			found := false
			for _, c := range cmds {
				if c.Context == contextName && c.UUID == uuid {
					command = c
					found = true
					break
				}
			}

			if !found {
				services.Error("Command not found - UUID: %s, Context: %s", uuid, contextName)
				services.SendMessage(conn, []byte(`{"error": "Command not found", "success": false}`))
				continue
			}

			services.Info("Executing command: %s (App: %s, Action: %s)",
				command.Description, command.App, command.Action)

			switch command.Action {
			case "open":
				services.Debug("Opening application: %s", command.App)
				err := command.OpenApplication()
				if err != nil {
					services.Error("Error opening app %s: %v", command.App, err)
					services.SendMessage(conn, fmt.Appendf(nil, `{"error": "Command failed: %v", "success": false}`, err))
					continue
				}
				services.Info("Successfully opened app: %s", command.App)
			default:
				actionParts := strings.Split(command.Action, " ")
				if len(actionParts) > 0 {
					services.Debug("Executing command: %s with %d arguments", actionParts[0], len(actionParts)-1)
				}

				err := command.Execute()
				if err != nil {
					services.Error("Error executing command: %v", err)
					services.SendMessage(conn, fmt.Appendf(nil, `{"error": "Command failed: %v", "success": false}`, err))
					continue
				}
				services.Info("Command executed successfully")
			}

			services.SendMessage(conn, []byte(`{"success": true}`))
			services.Info("Command execution response sent to client %s", clientIP)
		default:
			services.Warn("Unknown event type '%s' from client %s", eventType, clientIP)
			services.SendMessage(conn, fmt.Appendf(nil, `{"error": "Unknown event type: %s", "success": false}`, eventType))
		}
	}
}

func main() {
	// Set up standard logger for backward compatibility
	log.SetOutput(os.Stdout)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	// Initialize our new logger system
	logLevel := services.INFO
	if logLevelStr := viper.GetString("log.level"); logLevelStr != "" {
		logLevel = services.GetLogLevel(logLevelStr)
	}

	enableFileLogging := viper.GetBool("log.file_enabled")
	services.SetupLogging(enableFileLogging, logLevel)
	defer services.CloseLogger()

	services.Info("DevDeck server starting up...")
	services.Info("Log level set to: %v, File logging: %v", logLevel, enableFileLogging)

	// Load configuration
	services.LoadConfig(&cmds, &layout)

	viper.WatchConfig()

	viper.OnConfigChange(func(in fsnotify.Event) {
		services.LoadConfig(&cmds, &layout)
	})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go services.RegisterMDNS()

	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/ws", websocketHandler)

	port := viper.GetString("server.port")
	if port == "" {
		port = "4242"
	}

	srv := &http.Server{Addr: fmt.Sprintf(":%s", port)}

	// Log all available network interfaces for debugging
	interfaces, _ := net.Interfaces()
	for _, intf := range interfaces {
		addrs, err := intf.Addrs()
		if err != nil {
			services.Error("Error getting addresses for interface %s: %v", intf.Name, err)
			continue
		}

		if len(addrs) > 0 {
			addrStrings := make([]string, len(addrs))
			for i, addr := range addrs {
				addrStrings[i] = addr.String()
			}
			services.Debug("Network interface %s: %s", intf.Name, strings.Join(addrStrings, ", "))
		}
	}

	// Start the server
	services.Info("Starting DevDeck server on port %s", port)
	services.Info("Health endpoint: http://localhost:%s/health", port)
	services.Info("WebSocket endpoint: ws://localhost:%s/ws", port)

	go func() {
		services.Info("Server ready to accept connections")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			services.Fatal("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal
	<-ctx.Done()
	services.Info("Received shutdown signal, gracefully terminating...")

	stop()

	// Give the server time to finish ongoing connections
	services.Info("Allowing 5 seconds for connections to close")
	ctxShutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := srv.Shutdown(ctxShutdown); err != nil {
		services.Error("Server shutdown had an error: %v", err)
		services.Fatal("Server forced to exit")
	}

	services.Info("Thank you for using DevDeck - server exited gracefully")
}
