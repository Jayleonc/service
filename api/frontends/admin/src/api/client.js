// Lightweight fetch-based API client with auth and refresh handling
import { useAuthStore } from '@/store/auth'

const defaultHeaders = {
  'Content-Type': 'application/json',
}

async function parseEnvelope(response) {
  const text = await response.text()
  let json
  try {
    json = text ? JSON.parse(text) : {}
  } catch (e) {
    throw new Error('响应解析失败')
  }
  const { code, message, data } = json ?? {}
  if (!response.ok) {
    const msg = message || `HTTP ${response.status}`
    const err = new Error(msg)
    err.status = response.status
    err.code = code
    err.raw = json
    throw err
  }
  if (typeof code === 'number' && code !== 0) {
    const err = new Error(message || '业务错误')
    err.status = response.status
    err.code = code
    err.raw = json
    throw err
  }
  return data
}

async function doFetch(url, options = {}, { retryOn401 = true } = {}) {
  const auth = useAuthStore()

  const headers = { ...defaultHeaders, ...(options.headers || {}) }
  if (auth.accessToken) {
    headers['Authorization'] = `Bearer ${auth.accessToken}`
  }

  const res = await fetch(url, { ...options, headers })
  if (res.status === 401 && retryOn401 && auth.refreshToken) {
    try {
      await auth.refresh()
      const headers2 = { ...defaultHeaders, ...(options.headers || {}) }
      if (auth.accessToken) headers2['Authorization'] = `Bearer ${auth.accessToken}`
      const res2 = await fetch(url, { ...options, headers: headers2 })
      return await parseEnvelope(res2)
    } catch (e) {
      auth.logout()
      throw e
    }
  }
  return await parseEnvelope(res)
}

export const api = {
  get: (url, opts) => doFetch(url, { method: 'GET' }, opts),
  post: (url, body, opts) => doFetch(url, { method: 'POST', body: JSON.stringify(body) }, opts),
  put: (url, body, opts) => doFetch(url, { method: 'PUT', body: JSON.stringify(body) }, opts),
  del: (url, opts) => doFetch(url, { method: 'DELETE' }, opts),
}
