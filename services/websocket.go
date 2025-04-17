package services

import (
	"github.com/gorilla/websocket"
)

// SendMessage sends a message over the websocket connection
func SendMessage(conn *websocket.Conn, message []byte) error {
	// Log the outgoing message (truncate if too long)
	msgStr := string(message)
	if len(msgStr) > 200 {
		Debug("Sending websocket message: %s...(truncated)", msgStr[:200])
	} else {
		Debug("Sending websocket message: %s", msgStr)
	}
	
	err := conn.WriteMessage(websocket.TextMessage, message)
	
	if err != nil {
		Error("Failed to send websocket message: %v", err)
	}
	return err
}