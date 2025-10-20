import { createRouter, createWebHistory } from 'vue-router';

import { useAuthStore } from '@/store/auth';

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: () => import('@/views/LoginView.vue'),
      meta: { public: true },
    },
    {
      path: '/register',
      name: 'register',
      component: () => import('@/views/RegisterView.vue'),
      meta: { public: true },
    },
    {
      path: '/',
      redirect: '/me',
    },
    {
      path: '/me',
      name: 'me',
      component: () => import('@/views/MeView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/admin',
      component: () => import('@/views/AdminLayout.vue'),
      meta: { requiresAuth: true, requiresAdmin: true },
      children: [
        {
          path: 'users',
          name: 'admin-users',
          component: () => import('@/views/AdminUsers.vue'),
        },
        {
          path: 'roles',
          name: 'admin-roles',
          component: () => import('@/views/AdminRoles.vue'),
        },
      ],
    },
    {
      path: '/:pathMatch(.*)*',
      redirect: '/me',
    },
  ],
});

let profileLoaded = false;

router.beforeEach(async (to, from, next) => {
  const auth = useAuthStore();

  if (!auth.isAuthenticated) {
    profileLoaded = false;
  }

  if (auth.isAuthenticated && !profileLoaded) {
    try {
      await auth.loadProfile();
    } catch (error) {
      console.error('Failed to load profile', error);
      auth.logout();
    }
    profileLoaded = true;
  }

  if (to.meta.requiresAuth && !auth.isAuthenticated) {
    return next({ name: 'login', query: { redirect: to.fullPath } });
  }

  if (to.meta.requiresAdmin && !auth.isAdmin) {
    return next({ name: 'me' });
  }

  if (to.name === 'login' && auth.isAuthenticated) {
    return next({ name: 'me' });
  }

  return next();
});

export default router;
