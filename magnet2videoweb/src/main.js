import { createApp } from 'vue'
import App from './App.vue'

import './assets/main.css'

createApp(App).mount('#app')

import Vue from 'vue';
import VueRouter from 'vue-router';
import Login from './components/loginpage.vue';
import UserInfo from './components/userinfo.vue';

Vue.use(VueRouter);

const router = new VueRouter({
  mode: 'history',
  routes: [
    { path: '/login', component: Login },
    { path: '/userinfo', component: UserInfo }
  ]
});

new Vue({
  router
}).$mount('#app');
