import { create } from 'zustand'

interface ConnectionInfo {
  host: string
  port: number
  user: string
  database: string
  version: string
}

interface ConnectionState {
  connected: boolean
  info: ConnectionInfo | null
  setConnected: (info: ConnectionInfo) => void
  setDisconnected: () => void
}

export const useConnectionStore = create<ConnectionState>((set) => ({
  connected: false,
  info: null,
  setConnected: (info) => set({ connected: true, info }),
  setDisconnected: () => set({ connected: false, info: null }),
}))
