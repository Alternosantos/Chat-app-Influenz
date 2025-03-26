<style src="@/components/style.css"></style>
<template>
  <div class="chat-container">
    <div class="users-panel">
      <input type="text" placeholder="Search">
      <div class="user-list">
        <div v-for="(user, index) in users" :key="index" class="user-item">
          <input type="checkbox" :checked="user.active">
          <span>{{ user.name }}</span>
        </div>
      </div>
    </div>
    
    <div class="messages-panel">
      <div class="messages" ref="messages">
        <div v-for="(msg, index) in messages" :key="index" class="message">
          <div class="message-content">
            <strong>{{ msg.sender }}:</strong> {{ msg.content }}
          </div>
          <div class="message-meta">
            <small>{{ formatDate(msg.sent_at) }}</small>
          </div>
        </div>
      </div>
      
      <div class="input-area">
        <input v-model="newMessage" @keyup.enter="sendMessage" placeholder="Type your message...">
        <button @click="sendMessage">Send</button>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  data() {
    return {
      newMessage: '',
      messages: [],
      users: [
        { name: 'Username this is a sample text that was ...', active: false },
        { name: 'Username this is the lost sent text over here...', active: false },
        { name: 'Username this is the lost sent text over here...', active: false },
      ],
      ws: null,
      sender: 'Anonymous' + Math.floor(Math.random() * 1000)
    }
  },
  mounted() {
    this.connectWebSocket()
    this.fetchHistory()
  },
  methods: {
    formatDate(dateString) {
      return new Date(dateString).toLocaleTimeString()
    },
    connectWebSocket() {
      this.ws = new WebSocket('ws://localhost:8080/ws')
      
      this.ws.onerror = (error) => {
        console.error('WebSocket error:', error)
      }
      
      this.ws.onmessage = (event) => {
        const message = JSON.parse(event.data)
        this.messages.push(message)
        this.scrollToBottom()
      }
    },
    async fetchHistory() {
      try {
        const response = await fetch('http://localhost:8080/messages')
        this.messages = await response.json()
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
          sent_at: new Date().toISOString()
        }
        this.ws.send(JSON.stringify(message))
        this.newMessage = ''
      }
    },
    scrollToBottom() {
      this.$nextTick(() => {
        const container = this.$refs.messages
        container.scrollTop = container.scrollHeight
      })
    }
  }
}
</script>
