// Composables
import { createRouter, createWebHistory } from "vue-router";

const routes = [
  // Auth pages
  {
    path: "/login",
    name: "Login",
    component: () => import("@/views/Login.vue"),
    meta: { transition: "fade" },
  },
  {
    path: "/register",
    name: "Register",
    component: () => import("@/views/Register.vue"),
    meta: { transition: "fade" },
  },

  // App pages (require auth)
  {
    path: "/",
    component: () => import("@/layouts/default/Default.vue"),
    meta: { transition: "slide-right", requiresAuth: true },
    children: [
      {
        path: "",
        name: "Users",
        component: () => import(/* webpackChunkName: "users" */ "@/views/Users.vue"),
        meta: { requiresAuth: true },
      },
      {
        path: "me",
        name: "Me",
        component: () => import("@/views/Me.vue"),
        meta: { requiresAuth: true },
      },
    ],
  },
  {
    path: "/user-profile/:id",
    component: () => import("@/layouts/default/Default.vue"),
    meta: { transition: "slide-right", requiresAuth: true },
    children: [
      {
        path: "",
        name: "UserProfile",
        component: () => import(/* webpackChunkName: "userProfile" */ "@/views/UserProfile.vue"),
        meta: { requiresAuth: true },
      },
    ],
  },
];

const router = createRouter({
  history: createWebHistory(process.env.BASE_URL),
  routes,
});

export default router;
