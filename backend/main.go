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

var clients = make(map[*websocket.Conn]bool)
var broadcast = make(chan Message)

type Message struct {
    Content string `json:"content"`
    Sender  string `json:"sender"`
    SentAt  string `json:"sent_at"`
}

func main() {
    router := mux.NewRouter()

    // Database connection
    db, err := sql.Open("mysql", "root:password@tcp(127.0.0.1:3306)/Chat_app_db")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    _, err = db.Exec(`CREATE TABLE IF NOT EXISTS messages (
        id INT PRIMARY KEY AUTO_INCREMENT,
        content TEXT,
        sender VARCHAR(255),
        sent_at DATETIME
    )`)
    if err != nil {
        log.Fatal(err)
    }

    router.HandleFunc("/ws", handleConnections)
    router.HandleFunc("/messages", getMessages(db)).Methods("GET")

    go handleMessages(db)

    fmt.Println("Server started on :8080")
    log.Fatal(http.ListenAndServe(":8080", router))
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
    ws, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Fatal(err)
    }
    defer ws.Close()

    clients[ws] = true

    for {
        var msg Message
        err := ws.ReadJSON(&msg)
        if err != nil {
            log.Printf("error: %v", err)
            delete(clients, ws)
            break
        }
        msg.SentAt = time.Now().Format(time.RFC3339)
        broadcast <- msg
    }
}

func handleMessages(db *sql.DB) {
    for {
        msg := <-broadcast
        _, err := db.Exec("INSERT INTO messages (content, sender, sent_at) VALUES (?, ?, ?)",
            msg.Content, msg.Sender, msg.SentAt)
        if err != nil {
            log.Printf("Database error: %v", err)
        }

        for client := range clients {
            err := client.WriteJSON(msg)
            if err != nil {
                log.Printf("error: %v", err)
                client.Close()
                delete(clients, client)
            }
        }
    }
}

func getMessages(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        rows, err := db.Query("SELECT content, sender, sent_at FROM messages ORDER BY sent_at DESC LIMIT 50")
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        defer rows.Close()

        var messages []Message
        for rows.Next() {
            var msg Message
            if err := rows.Scan(&msg.Content, &msg.Sender, &msg.SentAt); err != nil {
                http.Error(w, err.Error(), http.StatusInternalServerError)
                return
            }
            messages = append(messages, msg)
        }

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(messages)
    }
}
