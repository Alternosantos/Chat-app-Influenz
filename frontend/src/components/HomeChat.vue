<template>
  <div class="main-container">
    <h1 class="chat-title">Messages</h1>
    <div class="chat-wrapper">
      <div class="users-container">
        <h2>Users</h2>
        <div class="search-bar">
          <input type="text" placeholder="Search users" v-model="searchQuery" />
          <button>Search</button>
        </div>
        <div class="user-list">
          <div
            v-for="(user, index) in filteredUsers"
            :key="index"
            class="user-item"
            @click="selectUser(user)"
            :class="{ active: selectedUser && selectedUser.name === user.name }"
          >
            <img :src="user.avatar" alt="User" />
            <div>
              <strong>{{ user.name }}</strong>
              <p>{{ user.lastMessage }}</p>
            </div>
          </div>
        </div>
      </div>

      <div class="messages-container">
        <h2>{{ selectedUser ? selectedUser.name : "Select a user" }}</h2>
        <div class="messages" ref="messages">
          <div
            v-for="(msg, index) in activeMessages"
            :key="index"
            :class="['message', msg.sender === sender ? 'sent' : 'received']"
          >
            <span class="message-content">{{ msg.content }}</span>
            <span class="message-time">{{ formatTime(msg.sent_at) }}</span>
          </div>
        </div>

        <div class="input-area" v-if="selectedUser">
          <textarea
            v-model="newMessage"
            @keyup.enter.exact="sendMessage"
            placeholder="Type your message..."
          ></textarea>
          <button @click="sendMessage">Send</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  data() {
    let userId = localStorage.getItem("user_id");
    if (!userId) {
      userId = "user_" + Math.floor(Math.random() * 1000);
      localStorage.setItem("user_id", userId);
    }
    return {
      searchQuery: "",
      newMessage: "",
      messages: [],
      users: [],
      selectedUser: null,
      ws: null,
      sender: userId,
    };
  },
  created() {
    const savedUser = localStorage.getItem("selectedUser");
    if (savedUser) {
      try {
        this.selectedUser = JSON.parse(savedUser);
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
          this.fetchHistory();
        }
      } catch (e) {
        console.error("Failed to parse saved user", e);
      }
    }
  },
  mounted() {
    this.connectWebSocket();
  },

  watch: {
    selectedUser(newUser) {
      if (newUser && this.ws && this.ws.readyState === WebSocket.OPEN) {
        this.fetchHistory();
        localStorage.setItem("selectedUser", JSON.stringify(newUser));
      }
    },
  },

  beforeUnmount() {
    if (this.ws) {
      this.ws.close();
    }
  },
  computed: {
    filteredUsers() {
      const q = this.searchQuery.toLowerCase();
      return this.users.filter((u) => u.name.toLowerCase().includes(q));
    },
    activeMessages() {
      if (!this.selectedUser) return [];
      return this.messages
        .filter(
          (msg) =>
            (msg.sender === this.sender &&
              msg.recipient === this.selectedUser.name) ||
            (msg.recipient === this.sender &&
              msg.sender === this.selectedUser.name)
        )
        .sort((a, b) => new Date(a.sent_at) - new Date(b.sent_at));
    },
  },
  methods: {
    connectWebSocket() {
      const protocol = window.location.protocol === "https:" ? "wss" : "ws";
      const host =
        process.env.NODE_ENV === "production"
          ? window.location.host
          : "localhost:8080";

      this.ws = new WebSocket(`ws://${window.location.hostname}:8080/ws`);

      this.ws.onopen = () => {
        console.log("WebSocket connected");
        this.ws.send(
          JSON.stringify({
            type: "init",
            userID: this.sender,
          })
        );
      };

      this.ws.onmessage = (event) => {
        console.log("RAW WEBSOCKET MESSAGE:", event.data);
        const message = JSON.parse(event.data);

        if (message.type === "activeUsers") {
          this.users = message.users
            .filter((name) => name !== this.sender)
            .map((name) => ({
              name,
              avatar: `https://api.dicebear.com/6.x/identicon/svg?seed=${name}`,
              lastMessage: "",
            }));
        } else {
          console.log("PROCESSED MESSAGE:", message);
          const exists = this.messages.some(
            (m) => m.sent_at === message.sent_at && m.sender === message.sender
          );

          if (!exists) {
            this.messages.push(message);
            this.updateLastMessage(message);
            if (
              message.sender === this.selectedUser?.name ||
              message.recipient === this.selectedUser?.name
            ) {
              this.scrollToBottom();
            }
          }
        }
      };

      this.ws.onclose = () => {
        console.log("WebSocket closed. Reconnecting...");
        setTimeout(this.connectWebSocket, 3000);
      };

      this.ws.onerror = (error) => {
        console.error("WebSocket error:", error);
      };
    },

    async fetchHistory() {
      if (!this.selectedUser) return;
      try {
        const protocol = window.location.protocol;
        const host =
          process.env.NODE_ENV === "production"
            ? window.location.host
            : "localhost:8080";

        const res = await fetch(
          `${protocol}//${host}/messages?sender=${this.sender}&recipient=${this.selectedUser.name}`
        );

        if (!res.ok) {
          console.error("Server error:", await res.text());
          return;
        }

        const data = await res.json();
        this.messages = Array.isArray(data) ? data : [];
        this.scrollToBottom();
      } catch (err) {
        console.error("Fetch error:", err);
      }
    },

    sendMessage() {
      if (this.newMessage.trim() && this.selectedUser) {
        const message = {
          type: "message",
          content: this.newMessage,
          sender: this.sender,
          recipient: this.selectedUser.name,
          sent_at: new Date().toISOString(),
        };

        this.newMessage = "";
        this.updateLastMessage(message);

        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
          this.ws.send(JSON.stringify(message));
        }
      }
    },

    updateLastMessage(message) {
      const user = this.users.find(
        (u) => u.name === message.sender || u.name === message.recipient
      );
      if (user) {
        user.lastMessage =
          message.content.length > 20
            ? message.content.substring(0, 20) + "..."
            : message.content;
      }
    },

    scrollToBottom() {
      this.$nextTick(() => {
        const el = this.$refs.messages;
        if (el) el.scrollTop = el.scrollHeight;
      });
    },

    selectUser(user) {
      this.selectedUser = user;
      this.fetchHistory();
    },

    formatTime(timestamp) {
      if (!timestamp) return "";
      return new Date(timestamp).toLocaleTimeString([], {
        hour: "2-digit",
        minute: "2-digit",
      });
    },
  },
};
</script>

<style src="@/components/style.css"></style>
