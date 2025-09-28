import { useAuthStore } from '@/store/auth'

export function setupRouterGuards(router) {
  router.beforeEach(async (to, from, next) => {
    const auth = useAuthStore()
    // Ensure state is hydrated
    if (auth.accessToken === null && typeof window !== 'undefined') {
      auth.hydrate()
    }

    // If already authenticated, prevent visiting login/register
    if ((to.name === 'Login' || to.name === 'Register') && auth.isAuthenticated) {
      return next({ path: '/' })
    }

    if (to.matched.some(r => r.meta && r.meta.requiresAuth)) {
      if (!auth.isAuthenticated) {
        return next({ path: '/login', query: { redirect: to.fullPath } })
      }
    }

    return next()
  })
}
