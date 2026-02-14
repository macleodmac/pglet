import { client } from './generated/client.gen'

// Configure the hey-api client with base URL.
// In dev mode, Vite proxies /api to localhost:8081.
// In production, relative URLs work since frontend is served from the same origin.
client.setConfig({ baseUrl: '' })

export { client }

// Re-export all generated SDK functions and types for convenience
export * from './generated'
