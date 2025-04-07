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
            v-for="(msg, index) in [...activeMessages].reverse()"
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
    return {
      searchQuery: "",
      newMessage: "",
      messages: [],
      users: [],
      selectedUser: null,
      ws: null,
      sender: "user_" + Math.floor(Math.random() * 1000),
    };
  },
  mounted() {
    this.connectWebSocket();
  },
  computed: {
    filteredUsers() {
      return this.users.filter((user) =>
        user.name.toLowerCase().includes(this.searchQuery.toLowerCase())
      );
    },
    activeMessages() {
      if (!this.selectedUser) {
        console.log("No user selected");
        return [];
      }

      const filtered = this.messages.filter(
        (msg) =>
          (msg.sender === this.sender &&
            msg.recipient === this.selectedUser.name) ||
          (msg.recipient === this.sender &&
            msg.sender === this.selectedUser.name)
      );

      console.log("Filtered messages:", filtered);
      return filtered.sort((a, b) => new Date(a.sent_at) - new Date(b.sent_at));
    },
  },
  methods: {
    connectWebSocket() {
      this.ws = new WebSocket("ws://localhost:8080/ws");

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
          if (
            (message.sender === this.selectedUser?.name &&
              message.recipient === this.sender) ||
            (message.recipient === this.selectedUser?.name &&
              message.sender === this.sender)
          ) {
            this.messages = [...this.messages, message];
            this.updateLastMessage(message);
            this.scrollToBottom();
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
        const res = await fetch(
          `http://localhost:8080/messages?sender=${this.sender}&recipient=${this.selectedUser.name}`
        );

        if (!res.ok) {
          const text = await res.text();
          console.error("Server error:", text);
          return;
        }

        const data = await res.json();
        this.messages = Array.isArray(data) ? data : [];
      } catch (err) {
        console.error("Error fetching history:", err);
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

        this.updateLastMessage(message);
        this.newMessage = "";
        this.scrollToBottom();

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
        const container = this.$refs.messages;
        if (container) {
          container.scrollTop = container.scrollHeight;
        }
      });
    },

    selectUser(user) {
      this.selectedUser = user;
      this.fetchHistory();
    },

    formatTime(timestamp) {
      if (!timestamp) return "";
      const date = new Date(timestamp);
      return date.toLocaleTimeString([], {
        hour: "2-digit",
        minute: "2-digit",
      });
    },
  },
};
</script>

<style src="@/components/style.css"></style>
