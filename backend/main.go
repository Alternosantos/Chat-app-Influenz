package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	_ "github.com/go-sql-driver/mysql"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Changed: now maps userID to connection
var clients = make(map[string]*websocket.Conn)
var broadcast = make(chan Message)
var db *sql.DB

type Message struct {
	Type      string `json:"type,omitempty"`
	Content   string `json:"content,omitempty"`
	Sender    string `json:"sender"`
	Recipient string `json:"recipient,omitempty"`
	SentAt    string `json:"sent_at,omitempty"`
}

func main() {
	var err error
	db, err = sql.Open("mysql", "root:password@tcp(127.0.0.1:3306)/Chat_app_db?parseTime=true")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS messages (
		id INT PRIMARY KEY AUTO_INCREMENT,
		content TEXT,
		sender VARCHAR(255),
		recipient VARCHAR(255),
		sent_at DATETIME
	)`)
	if err != nil {
		log.Fatal(err)
	}

	router := mux.NewRouter()

	router.Use(corsMiddleware)
	router.HandleFunc("/ws", handleConnections)
	router.HandleFunc("/messages", getMessages).Methods("GET")

	go handleMessages()

	fmt.Println("Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	var initMsg struct {
		Type   string `json:"type"`
		UserID string `json:"userID"`
	}
	err = ws.ReadJSON(&initMsg)
	if err != nil || initMsg.Type != "init" {
		log.Println("Failed to read init message")
		ws.Close()
		return
	}

	// Save connection
	clients[initMsg.UserID] = ws
	broadcastUserList()

	defer func() {
		ws.Close()
		delete(clients, initMsg.UserID)
		broadcastUserList()
	}()

	for {
		var msg Message
		err := ws.ReadJSON(&msg)
		if err != nil {
			log.Printf("Read error: %v", err)
			break
		}
		msg.SentAt = time.Now().Format(time.RFC3339)
		broadcast <- msg
	}
}

func handleMessages() {
	for msg := range broadcast {
		if msg.Type == "message" {
			_, err := db.Exec(
				"INSERT INTO messages (content, sender, recipient, sent_at) VALUES (?, ?, ?, ?)",
				msg.Content, msg.Sender, msg.Recipient, msg.SentAt,
			)
			if err != nil {
				log.Printf("Database error: %v", err)
				continue
			}
		}

		// Send to recipient and sender if they're connected
		if conn, ok := clients[msg.Recipient]; ok {
			conn.WriteJSON(msg)
		}
		if conn, ok := clients[msg.Sender]; ok && msg.Recipient != msg.Sender {
			conn.WriteJSON(msg)
		}
	}
}

func broadcastUserList() {
	userList := []string{}
	for userID := range clients {
		userList = append(userList, userID)
	}

	update := map[string]interface{}{
		"type":  "activeUsers",
		"users": userList,
	}

	for _, conn := range clients {
		conn.WriteJSON(update)
	}
}

func getMessages(w http.ResponseWriter, r *http.Request) {
	sender := r.URL.Query().Get("sender")
	recipient := r.URL.Query().Get("recipient")

	query := `
		SELECT content, sender, recipient, sent_at 
		FROM messages 
		WHERE (sender = ? AND recipient = ?) OR (sender = ? AND recipient = ?)
		ORDER BY sent_at ASC
	`

	rows, err := db.Query(query, sender, recipient, recipient, sender)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		if err := rows.Scan(&msg.Content, &msg.Sender, &msg.Recipient, &msg.SentAt); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		messages = append(messages, msg)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}
