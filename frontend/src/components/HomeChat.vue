<template>
  <div class="chat-container">
    <div class="messages" ref="messages">
      <div v-for="(msg, index) in messages" :key="index" class="message">
        <strong>{{ msg.sender }}:</strong> {{ msg.content }}
        <small>{{ formatDate(msg.sent_at) }}</small>
      </div>
    </div>
    <div class="input-area">
      <input v-model="newMessage" @keyup.enter="sendMessage" placeholder="Type your message..." />
      <button @click="sendMessage">Send</button>
    </div>
  </div>
</template>

<script>
export default {
  data() {
    return {
      newMessage: '',
      messages: [],
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
      
      this.ws.onmessage = (event) => {
        const message = JSON.parse(event.data)
        this.messages.push(message)
        this.scrollToBottom()
      }
    },
    async fetchHistory() {
      const response = await fetch('http://localhost:8081/messages')
      this.messages = await response.json()
      this.scrollToBottom()
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

<style scoped>
.chat-container {
  max-width: 600px;
  margin: 20px auto;
  border: 1px solid #ccc;
  border-radius: 8px;
  padding: 20px;
}

.messages {
  height: 400px;
  overflow-y: auto;
  margin-bottom: 20px;
  padding: 10px;
  border: 1px solid #eee;
  border-radius: 4px;
}

.message {
  margin-bottom: 10px;
  padding: 8px;
  border-bottom: 1px solid #eee;
}

.input-area {
  display: flex;
  gap: 10px;
}

input {
  flex: 1;
  padding: 8px;
  border: 1px solid #ccc;
  border-radius: 4px;
}

button {
  padding: 8px 16px;
  background-color: #007bff;
  color: white;
  border: none;
  border-radius: 4px;
  cursor: pointer;
}

button:hover {
  background-color: #0056b3;
}

small {
  color: #666;
  margin-left: 10px;
}
</style>
