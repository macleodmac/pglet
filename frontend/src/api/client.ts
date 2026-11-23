import { create } from 'zustand'
import { client } from './generated/client.gen'

/** Path pglet is served under: "" at the root, "/pg" with --prefix /pg. */
export const basePath = new URL(document.baseURI).pathname.replace(/\/$/, '')

/** Header the server requires on every API call, which other sites can't send. */
export const csrfHeaders = { 'X-Pglet': '1' }

client.setConfig({ baseUrl: basePath, headers: csrfHeaders })

interface AuthState {
  /** The server rejected our session cookie */
  signedOut: boolean
}

export const useAuthStore = create<AuthState>(() => ({ signedOut: false }))

client.interceptors.response.use((response) => {
  if (response.status === 401) useAuthStore.setState({ signedOut: true })
  return response
})

/** fetch() for API paths, with the base path and CSRF header applied. */
export function apiFetch(path: string, init: RequestInit = {}): Promise<Response> {
  const headers = new Headers(init.headers)
  for (const [k, v] of Object.entries(csrfHeaders)) headers.set(k, v)
  return fetch(basePath + path, { ...init, headers }).then((res) => {
    if (res.status === 401) useAuthStore.setState({ signedOut: true })
    return res
  })
}

export { client }

export * from './generated'
