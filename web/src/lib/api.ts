import type { DailyAccuracy, Prediction, PriceSnapshot, ServerEvent, Signal } from '../types/events'

export type SessionPayload = {
  session: {
    session_id: string
    created_at: string
    last_seen: string
    last_seq: number
    ttl_seconds: number
  }
  events?: ServerEvent[]
  history?: string[]
}

export async function login(apiKey: string): Promise<SessionPayload> {
  const response = await fetch('/api/web-console/login', {
    method: 'POST',
    credentials: 'include',
    headers: {
      'X-API-Key': apiKey,
    },
  })
  if (!response.ok) {
    throw new Error('login failed')
  }
  return response.json() as Promise<SessionPayload>
}

export async function getSession(sessionId?: string, since = 0): Promise<SessionPayload> {
  const params = new URLSearchParams()
  if (sessionId) {
    params.set('session_id', sessionId)
  }
  if (since > 0) {
    params.set('since', String(since))
  }
  const response = await fetch(`/api/web-console/session?${params.toString()}`, {
    credentials: 'include',
  })
  if (!response.ok) {
    throw new Error('session fetch failed')
  }
  return response.json() as Promise<SessionPayload>
}

export async function logout(): Promise<void> {
  await fetch('/api/web-console/logout', {
    method: 'POST',
    credentials: 'include',
  })
}

function authHeaders(apiKey: string): HeadersInit {
  const headers: Record<string, string> = {}
  if (apiKey.trim()) {
    headers['X-API-Key'] = apiKey.trim()
  }
  return headers
}

export async function getPrices(apiKey: string): Promise<PriceSnapshot[]> {
  const response = await fetch('/api/prices', {
    credentials: 'include',
    headers: authHeaders(apiKey),
  })
  if (!response.ok) {
    throw new Error('failed to fetch prices')
  }
  const payload = (await response.json()) as { prices?: PriceSnapshot[] }
  return payload.prices ?? []
}

export type SignalQuery = {
  symbol?: string
  risk?: number
  indicator?: string
  limit?: number
}

export async function getSignals(apiKey: string, query: SignalQuery): Promise<Signal[]> {
  const params = new URLSearchParams()
  if (query.symbol) {
    params.set('symbol', query.symbol)
  }
  if (query.risk) {
    params.set('risk', String(query.risk))
  }
  if (query.indicator) {
    params.set('indicator', query.indicator)
  }
  if (query.limit) {
    params.set('limit', String(query.limit))
  }
  const response = await fetch(`/api/signals?${params.toString()}`, {
    credentials: 'include',
    headers: authHeaders(apiKey),
  })
  if (!response.ok) {
    throw new Error('failed to fetch signals')
  }
  const payload = (await response.json()) as { signals?: Signal[] }
  return payload.signals ?? []
}

export async function getSignalImage(apiKey: string, signalID: number): Promise<Blob> {
  const response = await fetch(`/api/signals/${signalID}/image`, {
    credentials: 'include',
    headers: authHeaders(apiKey),
  })
  if (!response.ok) {
    throw new Error('failed to fetch signal image')
  }
  return response.blob()
}

export async function getBacktestSummary(apiKey: string): Promise<DailyAccuracy[]> {
  const response = await fetch('/api/backtest/summary', {
    credentials: 'include',
    headers: authHeaders(apiKey),
  })
  if (!response.ok) {
    throw new Error('failed to fetch backtest summary')
  }
  const payload = (await response.json()) as { summary?: DailyAccuracy[] }
  return payload.summary ?? []
}

export async function getBacktestDaily(apiKey: string, days = 30): Promise<DailyAccuracy[]> {
  const params = new URLSearchParams({ days: String(days) })
  const response = await fetch(`/api/backtest/daily?${params.toString()}`, {
    credentials: 'include',
    headers: authHeaders(apiKey),
  })
  if (!response.ok) {
    throw new Error('failed to fetch backtest daily')
  }
  const payload = (await response.json()) as { daily?: DailyAccuracy[] }
  return payload.daily ?? []
}

export async function getBacktestPredictions(apiKey: string, limit = 50): Promise<Prediction[]> {
  const params = new URLSearchParams({ limit: String(limit) })
  const response = await fetch(`/api/backtest/predictions?${params.toString()}`, {
    credentials: 'include',
    headers: authHeaders(apiKey),
  })
  if (!response.ok) {
    throw new Error('failed to fetch backtest predictions')
  }
  const payload = (await response.json()) as { predictions?: Prediction[] }
  return payload.predictions ?? []
}
