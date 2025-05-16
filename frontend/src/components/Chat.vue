<!--when i was writing this only 3 people knew how it worked me gustavo and god now only god know good luck!! -->

<template>
  <div class="main-container">
    <h1 class="chat-title">Messages</h1>
    <button
      @click="toggleUsers"
      class="toggle-users-btn"
      :class="{ collapsed: !showUsers }"
      :title="showUsers ? 'Hide users' : 'Show users'"
    ></button>
    <div class="chat-wrapper">
      <div class="users-container" v-show="showUsers">
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
              <img :src="user.avatar" alt="User Avatar" />
              <span
                v-if="
                  user.hasNewMessage &&
                  (!selectedUser || selectedUser.name !== user.name)
                "
                class="notification-dot"
              ></span>
            </div>
            <div>
              <strong>{{ user.name }}</strong>
              <p>{{ user.lastMessage }}</p>
            </div>
          </div>
        </div>
      </div>

      <div class="messages-container">
        <template v-if="selectedUser">
          <h2>{{ selectedUser.name }}</h2>
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
    let userId = localStorage.getItem("user_id");
    let userName = localStorage.getItem("user_name");

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
  },
  mounted() {
    this.connectWebSocket();
  },
  watch: {
    selectedUser(newUser) {
      if (newUser && this.ws && this.ws.readyState === WebSocket.OPEN) {
        this.fetchHistory();
        this.newMessageUsers = this.newMessageUsers.filter(
          (name) => name !== newUser.name
        );
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
    toggleUsers() {
      this.showUsers = !this.showUsers;

      if (window.innerWidth <= 768) {
        document.body.style.overflow = this.showUsers ? "hidden" : "auto";
      }
    },
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
        const days = [
          "Sunday",
          "Monday",
          "Tuesday",
          "Wednesday",
          "Thursday",
          "Friday",
          "Saturday",
        ];
        return days[msgDate.getDay()];
      } else {
        return msgDate.toLocaleDateString([], {
          day: "2-digit",
          month: "2-digit",
          year: "2-digit",
        });
      }
    },
  },
};
</script>

<style src="@/components/style.css"></style>
