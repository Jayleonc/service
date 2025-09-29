// Auth Store: manage tokens, user profile, and auth flows
import {defineStore} from 'pinia'
import router from '@/router'
import {api} from '@/api/client'

const TOKEN_KEY = 'auth.accessToken'
const REFRESH_KEY = 'auth.refreshToken'
const EXPIRES_KEY = 'auth.expiresAt'
const USER_KEY = 'auth.user'

export const useAuthStore = defineStore('auth', {
  state: () => ({
    accessToken: null,
    refreshToken: null,
    expiresAt: null, // epoch ms
    user: null,
    loading: false,
    error: null,
  }),
  getters: {
    isAuthenticated: (s) => !!s.accessToken,
  },
  actions: {
    hydrate() {
      try {
        this.accessToken = localStorage.getItem(TOKEN_KEY)
        this.refreshToken = localStorage.getItem(REFRESH_KEY)
        const exp = localStorage.getItem(EXPIRES_KEY)
        this.expiresAt = exp ? parseInt(exp, 10) : null
        const raw = localStorage.getItem(USER_KEY)
        this.user = raw ? JSON.parse(raw) : null
      } catch (e) {
        // ignore
      }
    },
    persist() {
      if (this.accessToken) localStorage.setItem(TOKEN_KEY, this.accessToken)
      else localStorage.removeItem(TOKEN_KEY)
      if (this.refreshToken) localStorage.setItem(REFRESH_KEY, this.refreshToken)
      else localStorage.removeItem(REFRESH_KEY)
      if (this.expiresAt) localStorage.setItem(EXPIRES_KEY, String(this.expiresAt))
      else localStorage.removeItem(EXPIRES_KEY)
      if (this.user) localStorage.setItem(USER_KEY, JSON.stringify(this.user))
      else localStorage.removeItem(USER_KEY)
    },
    setSession({access_token, refresh_token, expires_in, user}) {
      this.accessToken = access_token
      this.refreshToken = refresh_token
      // expires_in is seconds from backend
      const now = Date.now()
      this.expiresAt = now + (expires_in || 0) * 1000
      if (user) this.user = user
      this.persist()
    },
    clearSession() {
      this.accessToken = null
      this.refreshToken = null
      this.expiresAt = null
      this.user = null
      this.error = null
      this.persist()
    },
    async login(email, password, redirectPath) {
      this.loading = true
      this.error = null
      try {
        const data = await api.post('/v1/user/login', {email, password})
        this.setSession({
          access_token: data.access_token,
          refresh_token: data.refresh_token,
          expires_in: data.expires_in,
          user: data.user,
        })
        const target = typeof redirectPath === 'string' && redirectPath ? redirectPath : '/'
        await router.replace(target)
      } catch (e) {
        this.error = e.message || '登录失败'
        throw e
      } finally {
        this.loading = false
      }
    },
    async register({name, email, password, phone}) {
      this.loading = true
      this.error = null
      try {
        await api.post('/v1/user/register', {name, email, password, phone})
        await router.push('/login')
      } catch (e) {
        this.error = e.message || '注册失败'
        throw e
      } finally {
        this.loading = false
      }
    },
    async fetchMe() {
      const data = await api.post('/v1/user/me/get', {})
      this.user = data
      this.persist()
      return data
    },
    async updateMe(payload) {
      const data = await api.post('/v1/user/me/update', payload)
      this.user = data
      this.persist()
      return data
    },
    async refresh() {
      if (!this.refreshToken) throw new Error('缺少刷新令牌')
      const data = await api.post('/v1/auth/refresh', {refresh_token: this.refreshToken}, {retryOn401: false})
      this.setSession({
        access_token: data.access_token,
        refresh_token: data.refresh_token,
        expires_in: data.expires_in,
      })
    },
    async logout() {
      this.clearSession()
      await router.replace('/login')
    },
  },
})
