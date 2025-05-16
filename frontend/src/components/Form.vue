<!--when i was writing this only 3 people knew how it worked me gustavo and god now only god know good luck!! -->

<template>
  <div class="form-container">
    <h1>Login</h1>
    <form @submit.prevent="submitForm">
      <input v-model="id" type="text" placeholder="Enter your ID" required />
      <input
        v-model="name"
        type="text"
        placeholder="Enter your name"
        required
      />
      <button type="submit">Enter Chat</button>
    </form>
  </div>
</template>

<script>
export default {
  name: "FormComponent",
  data() {
    return {
      id: "",
      name: "",
    };
  },
  methods: {
    async submitForm() {
      localStorage.setItem("user_id", this.id);
      localStorage.setItem("user_name", this.name);

      
      await fetch("http://localhost:8080/users", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ id: this.id, name: this.name }),
      });

      
      const ws = new WebSocket(`ws://${window.location.hostname}:8080/ws`);
      ws.onopen = () => {
        ws.send(
          JSON.stringify({
            type: "init",
            userID: this.id,
            userName: this.name,
          })
        );
      };

      this.$router.push("/chat");
    },
  },
};
</script>

<style scoped>
.form-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  margin-top: 100px;
}
form {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}
input,
button {
  padding: 10px;
  font-size: 16px;
}
</style>