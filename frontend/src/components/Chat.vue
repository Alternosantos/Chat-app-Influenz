<!--when i was writing this only 3 people knew how it worked me gustavo and god now only god knows good luck!!-->
<template>
  <div class="main-container">
    <div
      class="chat-wrapper"
      :class="{ 'chat-active': isMobileView && chatActive }"
    >
      <!-- USERS LIST -->
      <div class="users-container" v-show="!isMobileView || !chatActive">
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
            <div class="avatar-wrapper">
              <img :src="user.avatar" />
              <span
                v-if="
                  user.hasNewMessage &&
                  (!selectedUser || selectedUser.name !== user.name)
                "
                class="notification-dot"
              ></span>
            </div>
            <div class="user-details">
              <strong>{{ user.name }}</strong>
              <p class="last-message">{{ user.lastMessage }}</p>
            </div>
          </div>
        </div>
      </div>

      <!-- MESSAGES -->
      <div class="messages-container" v-if="!isMobileView || chatActive">
        <template v-if="selectedUser">
          <div class="chat-header">
            <button
              class="back-to-users"
              @click="backToUsers"
              v-if="isMobileView"
              aria-label="Back to user list"
            >
              <svg
                width="24"
                height="24"
                viewBox="0 0 24 24"
                fill="none"
                xmlns="http://www.w3.org/2000/svg"
              >
                <path
                  d="M15 18L9 12L15 6"
                  stroke="white"
                  stroke-width="2"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                />
              </svg>
            </button>
            <div class="user-info">
              <div class="avatar-circle">
                <span class="avatar-initial">{{
                  selectedUser.name.charAt(0)
                }}</span>
              </div>
              <span class="chat-username">{{ selectedUser.name }}</span>
            </div>
          </div>

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

          <div class="input-area">
            <textarea
              v-model="newMessage"
              @keyup.enter.exact="sendMessage"
              placeholder="Type your message..."
            ></textarea>
            <button @click="sendMessage">Send</button>
          </div>
        </template>

        <template v-else>
          <div class="no-chat-selected">
            <img
              src="../assets/InfluenzLogo.png"
              alt="No chat selected"
              class="center-image"
            />
            <h2>Select a user to start chatting</h2>
            <p>Your messages will appear here.</p>
          </div>
        </template>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: "ChatComponent",
  data() {
    const userId = localStorage.getItem("user_id");
    const userName = localStorage.getItem("user_name");

    if (!userId || !userName) {
      window.location.href = "/";
    }

    return {
      searchQuery: "",
      newMessage: "",
      messages: [],
      users: [],
      senderName: userName,
      selectedUser: null,
      newMessageUsers: [],
      ws: null,
      sender: userId,
      showUsers: true,
      isMobileView: window.innerWidth < 992,
      chatActive: false,
    };
  },
  created() {
    localStorage.removeItem("selectedUser");
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

    window.addEventListener("resize", this.handleResize);
  },
  mounted() {
    this.connectWebSocket();
  },
  beforeUnmount() {
    if (this.ws) this.ws.close();
    window.removeEventListener("resize", this.handleResize);
  },
  watch: {
    selectedUser(newUser) {
      if (newUser && this.ws?.readyState === WebSocket.OPEN) {
        this.fetchHistory();
        this.newMessageUsers = this.newMessageUsers.filter(
          (name) => name !== newUser.name
        );
        localStorage.setItem("selectedUser", JSON.stringify(newUser));

        if (this.isMobileView) {
          this.chatActive = true;
        }
      }
    },
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
    handleResize() {
      this.isMobileView = window.innerWidth < 992;
      if (!this.isMobileView) {
        this.chatActive = false;
      }
    },
    toggleUsers() {
      this.showUsers = !this.showUsers;
      if (window.innerWidth <= 768) {
        document.body.style.overflow = this.showUsers ? "hidden" : "auto";
      }
    },
    connectWebSocket() {
      this.ws = new WebSocket(`ws://${window.location.hostname}:8080/ws`);
      this.reconnectAttempts = (this.reconnectAttempts || 0) + 1;
      const delay = Math.min(3000 * this.reconnectAttempts, 30000); // max 30s
      setTimeout(this.connectWebSocket, delay);

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
        const message = JSON.parse(event.data);

        if (message.type === "activeUsers") {
          // 1. Cache the old users before replacing
          const oldUsers = this.users || [];
          this.users = (message.users || [])
            .filter((name) => !!name) // skip empty names
            .filter((name) => name !== this.sender) // exclude self
            .map((name) => {
              const oldUser = oldUsers.find((u) => u.name === name);
              return {
                name,
                avatar: `https://api.dicebear.com/6.x/identicon/svg?seed=${name}`,
                lastMessage: oldUser ? oldUser.lastMessage : "",
                hasNewMessage: oldUser ? oldUser.hasNewMessage : false,
                lastMessageTime: oldUser ? oldUser.lastMessageTime : 0,
              };
            })
            .sort((a, b) => {
              const dateA = new Date(a.lastMessageTime || 0);
              const dateB = new Date(b.lastMessageTime || 0);
              return dateB - dateA;
            });
        } else {
          const exists = this.messages.some(
            (m) => m.sent_at === message.sent_at && m.sender === message.sender
          );
          if (!exists) {
            this.messages.push(message);
            this.updateLastMessage(message);
            const isCurrent = this.selectedUser?.name === message.sender;
            if (!isCurrent && message.recipient === this.sender) {
              if (!this.newMessageUsers.includes(message.sender)) {
                this.newMessageUsers.push(message.sender);
              }
            }
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
        const res = await fetch(
          `http://localhost:8080/messages?sender=${this.sender}&recipient=${this.selectedUser.name}`
        );
        if (!res.ok) return console.error("Server error:", await res.text());
        const data = await res.json();
        this.messages = Array.isArray(data) ? data : [];
        this.scrollToBottom();
      } catch (err) {
        console.error("Fetch error:", err);
      }
    },
    sendMessage() {
      if (!this.newMessage.trim() || !this.selectedUser) return;

      const message = {
        type: "message",
        content: this.newMessage,
        sender: this.sender,
        recipient: this.selectedUser.name,
        sent_at: new Date().toISOString(),
      };

      this.newMessage = "";
      this.updateLastMessage(message);
      if (this.ws?.readyState === WebSocket.OPEN) {
        this.ws.send(JSON.stringify(message));
      }
      this.scrollToBottom();
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
        // NEW: update lastMessageTime
        user.lastMessageTime = message.sent_at || new Date().toISOString();

        if (
          message.sender !== this.sender &&
          (!this.selectedUser || this.selectedUser.name !== message.sender)
        ) {
          user.hasNewMessage = true;
        }
      }
    },
    scrollToBottom() {
      this.$nextTick(() => {
        const el = this.$refs.messages;
        if (el) el.scrollTop = el.scrollHeight;
      });
    },
    selectUser(user) {
      user.hasNewMessage = false;
      this.selectedUser = user;
      this.newMessageUsers = this.newMessageUsers.filter(
        (name) => name !== user.name
      );
      this.fetchHistory();
      this.$nextTick(() => {
        if (this.isMobileView) {
          this.chatActive = true;
        }
      });
    },
    formatTime(timestamp) {
      if (!timestamp) return "";
      const now = new Date();
      const msgDate = new Date(timestamp);
      const diffDays = Math.floor((now - msgDate) / (1000 * 60 * 60 * 24));

      if (diffDays === 0) {
        return msgDate.toLocaleTimeString([], {
          hour: "2-digit",
          minute: "2-digit",
        });
      } else if (diffDays === 1) {
        return "Ontem";
      } else if (diffDays < 7) {
        return msgDate.toLocaleDateString("en-US", { weekday: "long" });
      } else {
        return msgDate.toLocaleDateString([], {
          day: "2-digit",
          month: "2-digit",
          year: "2-digit",
        });
      }
    },
    backToUsers() {
      this.chatActive = false;
      this.selectedUser = null;
      localStorage.removeItem("selectedUser");
    },
  },
};
</script>

<style src="@/components/style.css"></style>
