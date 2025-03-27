package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
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

	// Handle preflight OPTIONS request
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status": "ok", "server": "DevDeck"}`)
}

func websocketHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Error upgrading:", err)
		return
	}
	defer conn.Close()

	defer func() {
		if r := recover(); r != nil {
			log.Println("Recovered from panic:", r)
		}
	}()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Websocket closed unexpectedly: %v\n", err)
			} else {
				log.Println("Websocket closed by the client")
			}
			break
		}

		var request map[string]any
		if err := json.Unmarshal(message, &request); err != nil {
			log.Println("Invalid JSON received:", err)
			services.SendMessage(conn, []byte(`{"error": "Invalid JSON format", "success": false}`))
			continue
		}

		eventType, ok := request["type"].(string)
		if !ok {
			services.SendMessage(conn, []byte(`{"error": "Missing or invalid 'type' field"}`))
			continue
		}

		switch eventType {
		case "init":
			var rootCommands []commands.Command
			for _, c := range cmds {
				if c.Main {
					rootCommands = append(rootCommands, c)
				}
			}
			response, _ := json.Marshal(map[string]any{
				"layout":   layout,
				"commands": rootCommands,
			})
			services.SendMessage(conn, response)
		case "switch_context":
			contextName := request["context"].(string)

			contextCommands := []commands.Command{
				{
					UUID:    "back",
					Icon:    "arrow-back-outline",
					Type:    "context",
					Context: "main",
				},
			}
			for _, cmd := range cmds {
				if cmd.Context == contextName && !cmd.Main {
					contextCommands = append(contextCommands, cmd)
				}
			}

			response, _ := json.Marshal(map[string]any{
				"commands": contextCommands,
			})
			services.SendMessage(conn, response)
		case "fetch_commands":
			var rootCommands []commands.Command
			for _, c := range cmds {
				if c.Main {
					rootCommands = append(rootCommands, c)
				}
			}
			response, _ := json.Marshal(map[string]any{
				"commands": rootCommands,
			})
			services.SendMessage(conn, response)
		case "run":
			contextName, _ := request["context"].(string)

			uuid, _ := request["uuid"].(string)

			var command commands.Command
			for _, c := range cmds {
				if c.Context == contextName && c.UUID == uuid {
					command = c
				}
			}

			switch command.Action {
			case "open":
				err := command.OpenApplication()
				if err != nil {
					log.Printf("Error opening app: %s\n%s\n:", command.App, err)
					services.SendMessage(conn, fmt.Appendf(nil, `{"error": "Command failed: %v", "success": false}`, err))
					continue
				}
			default:
				err := command.Execute()
				if err != nil {
					log.Println("Error executing command:", err)
					services.SendMessage(conn, fmt.Appendf(nil, `{"error": "Command failed: %v", "success": false}`, err))
					continue
				}
			}

			services.SendMessage(conn, []byte(`{"success": true}`))
		default:
			services.SendMessage(conn, fmt.Appendf(nil, `{"error": "Unknown event type: %s", "success": false}`, eventType))
		}
	}
}

func main() {
	log.SetOutput(os.Stdout)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	services.LoadConfig(&cmds, &layout)

	viper.WatchConfig()

	viper.OnConfigChange(func(in fsnotify.Event) {
		services.LoadConfig(&cmds, &layout)
	})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	srv := &http.Server{Addr: ":8080"}

	go services.RegisterMDNS()

	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/ws", websocketHandler)

	port := viper.GetString("server.port")
	if port == "" {
		port = "8080"
	}

	go func() {
		log.Println("Websocket server started on :8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe failed: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutting down...")

	stop()

	ctxShutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctxShutdown); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited properly")
}
