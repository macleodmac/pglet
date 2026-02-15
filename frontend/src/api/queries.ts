import { useQuery, useMutation, keepPreviousData } from '@tanstack/react-query'
import { queryClient } from '../queryClient'
import {
  connect,
  disconnect,
  switchDatabase,
  getConnectionInfo,
  listDatabases,
  listSchemas,
  listObjects,
  getTableColumns,
  getTableRows,
  getTableInfo,
  getTableIndexes,
  getTableConstraints,
  getFunctionDefinition,
  getActivity,
  runQuery,
  explainQuery,
  analyzeQuery,
  cancelQuery,
  listSavedQueries,
  createSavedQuery,
  updateSavedQuery,
  deleteSavedQuery,
  listHistory,
  clearHistory,
  getSettings,
  updateSettings,
  aiGenerate,
  aiTabName,
} from './client'
import type { SavedQueryInput } from './generated'

function unwrap<T>(res: { data?: T; error?: unknown }): T {
  if (res.error) {
    const err = res.error as { error?: string }
    throw new Error(err?.error || 'Request failed')
  }
  return res.data as T
}

// Connection
export function useConnectionInfo() {
  return useQuery({
    queryKey: ['connection'],
    queryFn: async () => unwrap(await getConnectionInfo()),
    enabled: false,
  })
}

export function useConnect() {
  return useMutation({
    mutationFn: async (url: string) => unwrap(await connect({ body: { url } })),
  })
}

export function useDisconnect() {
  return useMutation({
    mutationFn: async () => unwrap(await disconnect()),
  })
}

export function useSwitchDatabase() {
  return useMutation({
    mutationFn: async (database: string) =>
      unwrap(await switchDatabase({ body: { database } })),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['objects'] })
      queryClient.invalidateQueries({ queryKey: ['schemas'] })
    },
  })
}

// Schema
export function useDatabases() {
  return useQuery({
    queryKey: ['databases'],
    queryFn: async () => unwrap(await listDatabases()),
  })
}

export function useSchemas() {
  return useQuery({
    queryKey: ['schemas'],
    queryFn: async () => unwrap(await listSchemas()),
  })
}

export function useObjects() {
  return useQuery({
    queryKey: ['objects'],
    queryFn: async () => unwrap(await listObjects()),
    staleTime: Infinity,
  })
}

export function useTableColumns(table: string) {
  return useQuery({
    queryKey: ['table-columns', table],
    queryFn: async () => unwrap(await getTableColumns({ path: { table } })),
    staleTime: Infinity,
    enabled: !!table,
  })
}

export function useTableRows(
  table: string,
  opts: { limit?: number; offset?: number; sort_column?: string; sort_order?: 'ASC' | 'DESC' },
) {
  return useQuery({
    queryKey: ['table-rows', table, opts],
    queryFn: async () =>
      unwrap(
        await getTableRows({
          path: { table },
          query: {
            limit: opts.limit,
            offset: opts.offset,
            sort_column: opts.sort_column,
            sort_order: opts.sort_order,
          },
        }),
      ),
    placeholderData: keepPreviousData,
    enabled: !!table,
  })
}

export function useTableInfo(table: string) {
  return useQuery({
    queryKey: ['table-info', table],
    queryFn: async () => unwrap(await getTableInfo({ path: { table } })),
    enabled: !!table,
  })
}

export function useTableIndexes(table: string) {
  return useQuery({
    queryKey: ['table-indexes', table],
    queryFn: async () => unwrap(await getTableIndexes({ path: { table } })),
    enabled: !!table,
  })
}

export function useTableConstraints(table: string) {
  return useQuery({
    queryKey: ['table-constraints', table],
    queryFn: async () => unwrap(await getTableConstraints({ path: { table } })),
    enabled: !!table,
  })
}

export function useFunctionDefinition(name: string) {
  return useQuery({
    queryKey: ['function-definition', name],
    queryFn: async () =>
      unwrap(await getFunctionDefinition({ path: { function: name } })),
    staleTime: Infinity,
    enabled: !!name,
  })
}

export function useActivity() {
  return useQuery({
    queryKey: ['activity'],
    queryFn: async () => unwrap(await getActivity()),
    refetchInterval: 5000,
  })
}

// Query execution
export function useRunQuery() {
  return useMutation({
    mutationFn: async (params: { query: string; tab_id: string }) =>
      unwrap(await runQuery({ body: params })),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['history'] })
    },
  })
}

export function useExplainQuery() {
  return useMutation({
    mutationFn: async (params: { query: string; tab_id: string }) =>
      unwrap(await explainQuery({ body: params })),
  })
}

export function useAnalyzeQuery() {
  return useMutation({
    mutationFn: async (params: { query: string; tab_id: string }) =>
      unwrap(await analyzeQuery({ body: params })),
  })
}

export function useCancelQuery() {
  return useMutation({
    mutationFn: async (tabId: string) =>
      unwrap(await cancelQuery({ body: { tab_id: tabId } })),
  })
}

// Saved queries
export function useSavedQueries(database?: string) {
  return useQuery({
    queryKey: ['saved-queries', database],
    queryFn: async () =>
      unwrap(await listSavedQueries({ query: { database } })),
  })
}

export function useCreateSavedQuery() {
  return useMutation({
    mutationFn: async (q: SavedQueryInput) =>
      unwrap(await createSavedQuery({ body: q })),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['saved-queries'] }),
  })
}

export function useUpdateSavedQuery() {
  return useMutation({
    mutationFn: async (q: { id: string } & SavedQueryInput) =>
      unwrap(await updateSavedQuery({ path: { id: q.id }, body: q })),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['saved-queries'] }),
  })
}

export function useDeleteSavedQuery() {
  return useMutation({
    mutationFn: async (id: string) =>
      unwrap(await deleteSavedQuery({ path: { id } })),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['saved-queries'] }),
  })
}

// History
export function useHistory(limit = 50, offset = 0) {
  return useQuery({
    queryKey: ['history', limit, offset],
    queryFn: async () =>
      unwrap(await listHistory({ query: { limit, offset } })),
  })
}

export function useClearHistory() {
  return useMutation({
    mutationFn: async () => unwrap(await clearHistory()),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['history'] }),
  })
}

// Settings
export function useSettings() {
  return useQuery({
    queryKey: ['settings'],
    queryFn: async () => unwrap(await getSettings()),
  })
}

export function useUpdateSettings() {
  return useMutation({
    mutationFn: async (settings: Record<string, string>) =>
      unwrap(await updateSettings({ body: settings })),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['settings'] }),
  })
}

// AI
export function useAiGenerate() {
  return useMutation({
    mutationFn: async (params: {
      prompt: string
      database: string
      messages?: Array<{ role: string; content: string }>
    }) => unwrap(await aiGenerate({ body: params })),
  })
}

/** Fire-and-forget: generate a short tab name from SQL. Returns the name or null on failure. */
export async function autoNameTab(sql: string): Promise<string | null> {
  try {
    const res = await aiTabName({ body: { sql } })
    const name = (res.data as { name?: string })?.name
    return name || null
  } catch {
    return null
  }
}
