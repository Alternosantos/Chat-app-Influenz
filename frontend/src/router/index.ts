import { createRouter, createWebHistory } from 'vue-router';
import FormComponent from '../components/Form.vue';
import ChatComponent from '../components/Chat.vue';

const routes = [
  { path: '/', component: FormComponent },
  { path: '/chat', component: ChatComponent },
];

const router = createRouter({
  history: createWebHistory(),
  routes,
});

export default router;
