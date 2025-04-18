package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	_ "github.com/go-sql-driver/mysql"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type Client struct {
	Conn *websocket.Conn
	Mu   sync.Mutex
}

type Clients struct {
	sync.RWMutex
	m map[string]*Client
}

var clients = Clients{m: make(map[string]*Client)}
var broadcast = make(chan Message)
var db *sql.DB

type Message struct {
	Type      string    `json:"type,omitempty"`
	Content   string    `json:"content,omitempty"`
	Sender    string    `json:"sender"`
	Recipient string    `json:"recipient,omitempty"`
	SentAt    time.Time `json:"sent_at,omitempty"`
}

func main() {
	var err error
	db, err = sql.Open("mysql", "root:secret@tcp(db:3306)/chatdb?parseTime=true")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS users (
		id INT AUTO_INCREMENT PRIMARY KEY,
		username VARCHAR(255) UNIQUE NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS messages (
		id INT AUTO_INCREMENT PRIMARY KEY,
		content TEXT NOT NULL,
		sender VARCHAR(255) NOT NULL,
		recipient VARCHAR(255) NOT NULL,
		sent_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		INDEX (sender),
		INDEX (recipient)
	)`)
	if err != nil {
		log.Fatal(err)
	}

	router := mux.NewRouter()
	router.Use(corsMiddleware)
	router.HandleFunc("/ws", handleConnections)
	router.HandleFunc("/messages", getMessages).Methods("GET")

	go handleMessages()

	server := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	log.Println("Server started on :8080")
	log.Fatal(server.ListenAndServe())
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
		
		if r.Header.Get("Upgrade") == "websocket" {
			next.ServeHTTP(w, r)
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
	
	if err := ws.ReadJSON(&initMsg); err != nil || initMsg.Type != "init" {
		log.Println("Failed to read init message:", err)
		ws.Close()
		return
	}

	
	clients.Lock()
	clients.m[initMsg.UserID] = &Client{Conn: ws}
	clients.Unlock()

	log.Printf("New connection from: %s", initMsg.UserID)
	broadcastUserList()

	defer func() {
		ws.Close()
		clients.Lock()
		delete(clients.m, initMsg.UserID)
		clients.Unlock()
		log.Printf("Connection closed for: %s", initMsg.UserID)
		broadcastUserList()
	}()

	for {
		var msg Message
		if err := ws.ReadJSON(&msg); err != nil {
			log.Printf("Read error: %v", err)
			break
		}
		msg.SentAt = time.Now()
		broadcast <- msg
	}
}

func handleMessages() {
	for msg := range broadcast {
		log.Printf("Processing message: %+v", msg)

		if msg.Type == "message" {
			// Store message in database
			_, err := db.Exec(
				"INSERT INTO messages (content, sender, recipient, sent_at) VALUES (?, ?, ?, ?)",
				msg.Content, msg.Sender, msg.Recipient, msg.SentAt,
			)
			if err != nil {
				log.Printf("Database error: %v", err)
				continue
			}

			
			clients.RLock()
			recipientClient, ok := clients.m[msg.Recipient]
			clients.RUnlock()
			
			if ok {
				recipientClient.Mu.Lock()
				if err := recipientClient.Conn.WriteJSON(msg); err != nil {
					log.Printf("Error sending to recipient %s: %v", msg.Recipient, err)
				}
				recipientClient.Mu.Unlock()
			}

			
			clients.RLock()
			senderClient, ok := clients.m[msg.Sender]
			clients.RUnlock()
			
			if ok && msg.Sender != msg.Recipient {
				senderClient.Mu.Lock()
				if err := senderClient.Conn.WriteJSON(msg); err != nil {
					log.Printf("Error sending to sender %s: %v", msg.Sender, err)
				}
				senderClient.Mu.Unlock()
			}
		}
	}
}

func broadcastUserList() {
	clients.RLock()
	defer clients.RUnlock()

	userList := make([]string, 0, len(clients.m))
	for userID := range clients.m {
		userList = append(userList, userID)
	}

	update := map[string]interface{}{
		"type":  "activeUsers",
		"users": userList,
	}

	for _, client := range clients.m {
		client.Mu.Lock()
		if err := client.Conn.WriteJSON(update); err != nil {
			log.Printf("Error broadcasting user list: %v", err)
		}
		client.Mu.Unlock()
	}
}

func getMessages(w http.ResponseWriter, r *http.Request) {
	sender := r.URL.Query().Get("sender")
	recipient := r.URL.Query().Get("recipient")

	if sender == "" || recipient == "" {
		http.Error(w, "Missing sender or recipient", http.StatusBadRequest)
		return
	}

	rows, err := db.Query(`
		SELECT sender, recipient, content, sent_at
		FROM messages
		WHERE (sender = ? AND recipient = ?)
		   OR (sender = ? AND recipient = ?)
		ORDER BY sent_at ASC
	`, sender, recipient, recipient, sender)

	if err != nil {
		log.Printf("Error querying messages: %v", err)
		http.Error(w, "Error fetching messages", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		if err := rows.Scan(&msg.Sender, &msg.Recipient, &msg.Content, &msg.SentAt); err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}
		messages = append(messages, msg)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(messages); err != nil {
		log.Printf("Error encoding messages: %v", err)
		http.Error(w, "Error encoding messages", http.StatusInternalServerError)
	}
}