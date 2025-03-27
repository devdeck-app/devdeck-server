package services

import (
	"log"

	"github.com/gorilla/websocket"
)

func SendMessage(conn *websocket.Conn, message []byte) error {
	err := conn.WriteMessage(websocket.TextMessage, message)

	if err != nil {
		log.Println("Failed to send message:", err)
	}
	return err
}
