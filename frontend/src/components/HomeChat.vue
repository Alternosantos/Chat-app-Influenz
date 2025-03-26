<style src="@/components/style.css"></style>
<template>
  <div class="chat-wrapper">
    <!-- Painel de Usuários -->
    <div class="users-container">
      <h2>Users</h2>
      <div class="search-bar">
        <input type="text" placeholder="Search users" v-model="searchQuery">
        <button>Search</button>
      </div>
  <!--Lista de usuarios-->
      <div class="user-list">
        <div v-for="(user, index) in filteredUsers" :key="index" class="user-item" @click="selectUser(user)">
          <img :src="user.avatar" alt="User">
          <div>
            <strong>{{ user.name }}</strong>
            <p>{{ user.lastMessage }}</p>
          </div>
        </div>
      </div>
    </div>

    <!-- Painel de Mensagens -->
    <div class="messages-container">
      <h2>{{ selectedUser ? selectedUser.name : "Username" }}</h2>
      <div class="messages" ref="messages">
        <div v-for="(msg, index) in messages" :key="index" :class="['message', msg.sent ? 'sent' : 'received']">
          {{ msg.content }}
        </div>
      </div>

      <!-- Input de mensagem -->
      <div class="input-area">
        <input v-model="newMessage" @keyup.enter="sendMessage" placeholder="Placeholder text">
        <button @click="sendMessage">Send</button>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  data() {
    return {
      searchQuery: '',
      newMessage: '',
      messages: [],
      users: [
        { name: 'Alberto', avatar: 'https://i.imgur.com/6VBx3io.png', lastMessage: 'this is a sample text that was ...' },
        { name: 'Username', avatar: 'https://i.imgur.com/6VBx3io.png', lastMessage: 'this is the last sent text over here...' },
        { name: 'Username', avatar: 'https://i.imgur.com/6VBx3io.png', lastMessage: 'this is the last sent text over here...' },
        { name: 'Username', avatar: 'https://i.imgur.com/6VBx3io.png', lastMessage: 'this is the last sent text over here...' },
        { name: 'Username', avatar: 'https://i.imgur.com/6VBx3io.png', lastMessage: 'this is the last sent text over here...' },
        { name: 'Username', avatar: 'https://i.imgur.com/6VBx3io.png', lastMessage: 'this is the last sent text over here...' }
      ],
      selectedUser: null,
      ws: null,
      sender: 'Anonymous' + Math.floor(Math.random() * 1000)
    }
  },
  mounted() {
    this.connectWebSocket()
    this.fetchHistory()
  },
  computed: {
    filteredUsers() {
      return this.users.filter(user =>
        user.name.toLowerCase().includes(this.searchQuery.toLowerCase())
      )
    }
  },
  methods: {
    connectWebSocket() {
      this.ws = new WebSocket('ws://localhost:8080/ws')

      this.ws.onmessage = (event) => {
        const message = JSON.parse(event.data)
        message.sent = message.sender === this.sender
        this.messages.push(message)
        this.scrollToBottom()
      }
    },
    async fetchHistory() {
      try {
        const response = await fetch('http://localhost:8080/messages')
        this.messages = await response.json().map(msg => ({
          ...msg,
          sent: msg.sender === this.sender
        }))
        this.scrollToBottom()
      } catch (error) {
        console.error('Failed to fetch history:', error)
      }
    },
    sendMessage() {
      if (this.newMessage.trim()) {
        const message = {
          content: this.newMessage,
          sender: this.sender,
          sent_at: new Date().toISOString(),
          sent: true
        }
        this.ws.send(JSON.stringify(message))
        this.messages.push(message)
        this.newMessage = ''
      }
    },
    scrollToBottom() {
      this.$nextTick(() => {
        const container = this.$refs.messages
        if (container) {
          container.scrollTop = container.scrollHeight
        }
      })
    },
    selectUser(user) {
      this.selectedUser = user
    }
  }
}
</script>

