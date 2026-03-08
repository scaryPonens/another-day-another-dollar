export type ConnectionState = 'connecting' | 'connected' | 'disconnected'

export type TabKey = 'dashboard' | 'chat' | 'signals' | 'backtest'

export type PriceSnapshot = {
  symbol: string
  price_usd: number
  volume_24h: number
  change_24h_pct: number
  last_updated_unix: number
}

export type Signal = {
  id: number
  symbol: string
  interval: string
  indicator: string
  timestamp: string
  risk: number
  direction: 'long' | 'short' | 'hold'
  details?: string
  image?: {
    image_id: number
    mime_type: string
    width: number
    height: number
    expires_at: string
  }
}

export type DailyAccuracy = {
  ModelKey?: string
  model_key?: string
  DayUTC?: string
  day_utc?: string
  Total?: number
  total?: number
  Correct?: number
  correct?: number
  Accuracy?: number
  accuracy?: number
}

export type Prediction = {
  Symbol?: string
  symbol?: string
  Interval?: string
  interval?: string
  ModelKey?: string
  model_key?: string
  Direction?: 'long' | 'short' | 'hold'
  direction?: 'long' | 'short' | 'hold'
  Risk?: number
  risk?: number
  IsCorrect?: boolean | null
  is_correct?: boolean | null
  RealizedReturn?: number | null
  realized_return?: number | null
}

export type ChatLine = {
  id: string
  role: 'user' | 'assistant' | 'system'
  text: string
  at: string
}

export type ServerEvent = {
  type: 'ui.status' | 'ui.chat.reply' | 'ui.error' | 'ui.heartbeat' | string
  session_id?: string
  request_id?: string
  seq?: number
  state?: string
  code?: string
  message?: string
  timestamp?: string
}

export type ClientEvent = {
  type: 'ui.command' | 'ui.ping' | 'ui.refresh'
  session_id?: string
  request_id?: string
  command?: string
  message?: string
}
