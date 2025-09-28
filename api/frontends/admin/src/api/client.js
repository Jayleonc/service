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

function normaliseBody(payload) {
  if (payload === undefined || payload === null) {
    return '{}'
  }
  if (typeof payload === 'string') {
    return payload
  }
  return JSON.stringify(payload)
}

async function doFetch(url, options = {}, { retryOn401 = true } = {}) {
  const auth = useAuthStore()

  const headers = { ...defaultHeaders, ...(options.headers || {}) }
  if (auth.accessToken) {
    headers.Authorization = `Bearer ${auth.accessToken}`
  }

  const requestInit = {
    method: options.method || 'POST',
    headers,
    body: normaliseBody(options.body),
  }

  const res = await fetch(url, requestInit)
  if (res.status === 401 && retryOn401 && auth.refreshToken) {
    try {
      await auth.refresh()
      const headers2 = { ...defaultHeaders, ...(options.headers || {}) }
      if (auth.accessToken) headers2.Authorization = `Bearer ${auth.accessToken}`
      const res2 = await fetch(url, {
        method: requestInit.method,
        headers: headers2,
        body: requestInit.body,
      })
      return await parseEnvelope(res2)
    } catch (e) {
      await auth.logout()
      throw e
    }
  }
  return await parseEnvelope(res)
}

export const api = {
  post: (url, body = {}, opts) => doFetch(url, { method: 'POST', body }, opts ?? {}),
}
