import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  getBacktestDaily,
  getBacktestPredictions,
  getBacktestSummary,
  getPrices,
  getSignalImage,
  getSignals,
  login,
  logout,
} from './lib/api'
import {
  SIGNAL_INDICATOR_OPTIONS,
  SIGNAL_RISK_OPTIONS,
  SIGNAL_SYMBOL_OPTIONS,
  cycleIndex,
  isEditableTarget,
  nextTab,
  tabFromKey,
} from './lib/keyboard'
import { ConsoleSocket } from './lib/ws'
import type { ChatLine, ConnectionState, DailyAccuracy, Prediction, ServerEvent, Signal, TabKey } from './types/events'

const SIGNAL_VISIBLE_ROWS = 16

function uid(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID()
  }
  return `id-${Date.now()}-${Math.random().toString(16).slice(2)}`
}

function formatMoney(value: number): string {
  return new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD', maximumFractionDigits: 2 }).format(value)
}

function formatPrice(value: number): string {
  return new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD', minimumFractionDigits: 2, maximumFractionDigits: 4 }).format(value)
}

function clamp(value: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, value))
}

function heatTileStyle(change24h: number) {
  const normalized = clamp(Math.abs(change24h) / 10, 0, 1)
  const alpha = 0.2 + normalized * 0.55
  if (change24h >= 0) {
    return { backgroundColor: `rgba(49, 194, 129, ${alpha})` }
  }
  return { backgroundColor: `rgba(240, 114, 114, ${alpha})` }
}

function readAccuracy(item: DailyAccuracy): number {
  return item.accuracy ?? item.Accuracy ?? 0
}

function readTotal(item: DailyAccuracy): number {
  return item.total ?? item.Total ?? 0
}

function readCorrect(item: DailyAccuracy): number {
  return item.correct ?? item.Correct ?? 0
}

function readModelKey(item: DailyAccuracy): string {
  return item.model_key ?? item.ModelKey ?? 'unknown'
}

function readDay(item: DailyAccuracy): string {
  return item.day_utc ?? item.DayUTC ?? ''
}

function readPredictionString(pred: Prediction, key: 'symbol' | 'interval' | 'model_key'): string {
  if (key === 'symbol') {
    return pred.symbol ?? pred.Symbol ?? ''
  }
  if (key === 'interval') {
    return pred.interval ?? pred.Interval ?? ''
  }
  return pred.model_key ?? pred.ModelKey ?? ''
}

function readPredictionDirection(pred: Prediction): string {
  return (pred.direction ?? pred.Direction ?? 'hold').toUpperCase()
}

function readPredictionRisk(pred: Prediction): number {
  return pred.risk ?? pred.Risk ?? 0
}

function readPredictionCorrect(pred: Prediction): boolean | null {
  const value = pred.is_correct ?? pred.IsCorrect
  if (value === undefined) {
    return null
  }
  return value
}

function readPredictionReturn(pred: Prediction): number | null {
  const value = pred.realized_return ?? pred.RealizedReturn
  if (value === undefined) {
    return null
  }
  return value
}

function hasSignalChart(signal: Signal): boolean {
  return Boolean(signal.image?.image_id)
}

function isMobileViewport(): boolean {
  return typeof window !== 'undefined' && window.matchMedia('(max-width: 720px)').matches
}

export default function App() {
  const [authenticated, setAuthenticated] = useState(false)
  const [apiKeyInput, setAPIKeyInput] = useState('')
  const [apiKey, setAPIKey] = useState('')
  const [authError, setAuthError] = useState('')
  const [sessionId, setSessionID] = useState('')
  const [activeTab, setActiveTab] = useState<TabKey>('dashboard')
  const [connection, setConnection] = useState<ConnectionState>('disconnected')
  const [degradedRealtime, setDegradedRealtime] = useState(false)
  const [statusText, setStatusText] = useState('ready')
  const [chatInput, setChatInput] = useState('')
  const [chatLines, setChatLines] = useState<ChatLine[]>([])
  const [chatWaiting, setChatWaiting] = useState(false)
  const [signalSymbolIndex, setSignalSymbolIndex] = useState(0)
  const [signalRiskIndex, setSignalRiskIndex] = useState(0)
  const [signalIndicatorIndex, setSignalIndicatorIndex] = useState(0)
  const [signalsScrollOffset, setSignalsScrollOffset] = useState(0)
  const [selectedSignalID, setSelectedSignalID] = useState<number | null>(null)
  const [selectedSignalChartURL, setSelectedSignalChartURL] = useState<string | null>(null)
  const [signalChartModalOpen, setSignalChartModalOpen] = useState(false)
  const [backtestView, setBacktestView] = useState<'accuracy' | 'predictions'>('accuracy')

  const socketRef = useRef<ConsoleSocket | null>(null)
  const reconnectRef = useRef<number | null>(null)
  const pingRef = useRef<number | null>(null)
  const activeRequestRef = useRef<string | null>(null)

  useEffect(() => {
    const stored = sessionStorage.getItem('web_console_api_key')
    if (stored) {
      setAPIKeyInput(stored)
    }
  }, [])

  const healthQuery = useQuery({
    queryKey: ['health'],
    queryFn: async () => {
      const response = await fetch('/health')
      return response.ok
    },
    refetchInterval: 30000,
  })

  const dashboardPricesQuery = useQuery({
    queryKey: ['prices', apiKey],
    queryFn: () => getPrices(apiKey),
    enabled: authenticated,
    refetchInterval: activeTab === 'dashboard' ? 10000 : false,
  })

  const dashboardSignalsQuery = useQuery({
    queryKey: ['dashboard-signals', apiKey],
    queryFn: () => getSignals(apiKey, { limit: 10 }),
    enabled: authenticated,
    refetchInterval: activeTab === 'dashboard' ? 10000 : false,
  })

  const signalsQuery = useQuery({
    queryKey: ['signals', apiKey, signalSymbolIndex, signalRiskIndex, signalIndicatorIndex],
    queryFn: () =>
      getSignals(apiKey, {
        limit: 100,
        symbol: signalSymbolIndex > 0 ? SIGNAL_SYMBOL_OPTIONS[signalSymbolIndex] : undefined,
        risk: signalRiskIndex > 0 ? Number(SIGNAL_RISK_OPTIONS[signalRiskIndex]) : undefined,
        indicator: signalIndicatorIndex > 0 ? SIGNAL_INDICATOR_OPTIONS[signalIndicatorIndex] : undefined,
      }),
    enabled: authenticated,
  })

  const signalChartQuery = useQuery({
    queryKey: ['signal-chart', apiKey, selectedSignalID],
    queryFn: () => getSignalImage(apiKey, selectedSignalID ?? 0),
    enabled: authenticated && selectedSignalID !== null,
  })

  const backtestSummaryQuery = useQuery({
    queryKey: ['backtest-summary', apiKey],
    queryFn: () => getBacktestSummary(apiKey),
    enabled: authenticated,
  })

  const backtestDailyQuery = useQuery({
    queryKey: ['backtest-daily', apiKey],
    queryFn: () => getBacktestDaily(apiKey, 30),
    enabled: authenticated,
  })

  const backtestPredictionsQuery = useQuery({
    queryKey: ['backtest-predictions', apiKey],
    queryFn: () => getBacktestPredictions(apiKey, 50),
    enabled: authenticated,
  })

  useEffect(() => {
    setSignalsScrollOffset(0)
  }, [signalSymbolIndex, signalRiskIndex, signalIndicatorIndex, signalsQuery.data])

  useEffect(() => {
    if (!selectedSignalID) {
      setSelectedSignalChartURL(null)
      setSignalChartModalOpen(false)
      return
    }

    const selectedStillVisible = (signalsQuery.data ?? []).some((signal) => signal.id === selectedSignalID && hasSignalChart(signal))
    if (!selectedStillVisible) {
      setSelectedSignalID(null)
      setSelectedSignalChartURL(null)
    }
  }, [signalsQuery.data, selectedSignalID])

  useEffect(() => {
    if (!signalChartQuery.data) {
      setSelectedSignalChartURL(null)
      return
    }
    const nextURL = URL.createObjectURL(signalChartQuery.data)
    setSelectedSignalChartURL(nextURL)
    return () => {
      URL.revokeObjectURL(nextURL)
    }
  }, [signalChartQuery.data])

  useEffect(() => {
    if (activeTab !== 'signals') {
      setSignalChartModalOpen(false)
    }
  }, [activeTab])

  const appendChatLine = useCallback((line: Omit<ChatLine, 'id' | 'at'>) => {
    setChatLines((prev) => [...prev, { id: uid(), at: new Date().toISOString(), ...line }])
  }, [])

  const handleWSEvent = useCallback(
    (event: ServerEvent) => {
      if (event.type === 'ui.status') {
        const state = event.state ?? 'status'
        if (state === 'thinking') {
          setChatWaiting(true)
          setStatusText('advisor thinking')
        } else if (state === 'idle') {
          setChatWaiting(false)
          setStatusText('ready')
          activeRequestRef.current = null
        } else if (state === 'connected') {
          setStatusText('realtime connected')
        } else {
          setStatusText(state)
        }
      } else if (event.type === 'ui.chat.reply') {
        setChatWaiting(false)
        appendChatLine({ role: 'assistant', text: event.message ?? '' })
        activeRequestRef.current = null
      } else if (event.type === 'ui.error') {
        setChatWaiting(false)
        appendChatLine({ role: 'system', text: `error: ${event.message ?? 'unknown error'}` })
        activeRequestRef.current = null
      } else if (event.type === 'ui.heartbeat') {
        setStatusText('realtime alive')
      }
    },
    [appendChatLine],
  )

  useEffect(() => {
    if (!authenticated || !sessionId) {
      return
    }

    let canceled = false

    const connect = () => {
      if (canceled) {
        return
      }
      const socket = new ConsoleSocket()
      socketRef.current = socket
      setConnection('connecting')
      socket.connect(sessionId, {
        onOpen: () => {
          if (canceled) {
            return
          }
          setConnection('connected')
          setDegradedRealtime(false)
          setStatusText('realtime connected')
        },
        onClose: () => {
          if (canceled) {
            return
          }
          setConnection('disconnected')
          setDegradedRealtime(true)
          setStatusText('realtime degraded')
          if (reconnectRef.current === null) {
            reconnectRef.current = window.setTimeout(() => {
              reconnectRef.current = null
              connect()
            }, 3000)
          }
        },
        onError: () => {
          if (canceled) {
            return
          }
          setStatusText('websocket error')
        },
        onEvent: handleWSEvent,
      })
    }

    connect()

    pingRef.current = window.setInterval(() => {
      const socket = socketRef.current
      if (socket?.isOpen()) {
        socket.send({ type: 'ui.ping', session_id: sessionId })
      }
    }, 10000)

    return () => {
      canceled = true
      if (reconnectRef.current !== null) {
        window.clearTimeout(reconnectRef.current)
        reconnectRef.current = null
      }
      if (pingRef.current !== null) {
        window.clearInterval(pingRef.current)
        pingRef.current = null
      }
      socketRef.current?.disconnect()
      socketRef.current = null
    }
  }, [authenticated, sessionId, handleWSEvent])

  const refreshActiveTab = useCallback(() => {
    if (activeTab === 'dashboard') {
      void dashboardPricesQuery.refetch()
      void dashboardSignalsQuery.refetch()
    } else if (activeTab === 'signals') {
      void signalsQuery.refetch()
    } else if (activeTab === 'backtest') {
      void backtestSummaryQuery.refetch()
      void backtestDailyQuery.refetch()
      void backtestPredictionsQuery.refetch()
    }
    socketRef.current?.send({ type: 'ui.refresh', session_id: sessionId })
  }, [
    activeTab,
    dashboardPricesQuery,
    dashboardSignalsQuery,
    signalsQuery,
    backtestSummaryQuery,
    backtestDailyQuery,
    backtestPredictionsQuery,
    sessionId,
  ])

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape' && signalChartModalOpen) {
        event.preventDefault()
        setSignalChartModalOpen(false)
        return
      }

      const tab = tabFromKey(event.key)
      if (tab) {
        event.preventDefault()
        setActiveTab(tab)
        return
      }
      if (event.key === 'Tab') {
        event.preventDefault()
        setActiveTab((prev) => nextTab(prev, event.shiftKey ? -1 : 1))
        return
      }

      const editable = isEditableTarget(event.target)
      if (activeTab === 'chat' && editable) {
        return
      }

      if (event.shiftKey && event.key.toLowerCase() === 'r') {
        event.preventDefault()
        refreshActiveTab()
        return
      }

      if (activeTab === 'signals') {
        if (event.key === 's' || event.key === 'S') {
          event.preventDefault()
          setSignalSymbolIndex((prev) => cycleIndex(prev, SIGNAL_SYMBOL_OPTIONS.length))
          return
        }
        if (event.key === 'r' || event.key === 'R') {
          if (event.shiftKey) {
            return
          }
          event.preventDefault()
          setSignalRiskIndex((prev) => cycleIndex(prev, SIGNAL_RISK_OPTIONS.length))
          return
        }
        if (event.key === 'i' || event.key === 'I') {
          event.preventDefault()
          setSignalIndicatorIndex((prev) => cycleIndex(prev, SIGNAL_INDICATOR_OPTIONS.length))
          return
        }
        if (event.key === 'j' || event.key === 'ArrowDown') {
          event.preventDefault()
          const size = signalsQuery.data?.length ?? 0
          setSignalsScrollOffset((prev) => {
            const max = Math.max(0, size - SIGNAL_VISIBLE_ROWS)
            return Math.min(max, prev + 1)
          })
          return
        }
        if (event.key === 'k' || event.key === 'ArrowUp') {
          event.preventDefault()
          setSignalsScrollOffset((prev) => Math.max(0, prev - 1))
          return
        }
      }

      if (activeTab === 'backtest' && (event.key === 'v' || event.key === 'V')) {
        event.preventDefault()
        setBacktestView((prev) => (prev === 'accuracy' ? 'predictions' : 'accuracy'))
      }
    }

    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [activeTab, refreshActiveTab, signalChartModalOpen, signalsQuery.data])

  async function handleLogin(event: React.FormEvent) {
    event.preventDefault()
    setAuthError('')
    try {
      const key = apiKeyInput.trim()
      const payload = await login(key)
      setSessionID(payload.session.session_id)
      setAPIKey(key)
      sessionStorage.setItem('web_console_api_key', key)
      setAuthenticated(true)
      setChatLines([{ id: uid(), role: 'system', text: 'Connected. Ask the advisor in the Chat tab.', at: new Date().toISOString() }])
    } catch (err) {
      setAuthError(err instanceof Error ? err.message : 'login failed')
    }
  }

  async function handleLogout() {
    await logout()
    socketRef.current?.disconnect()
    setAuthenticated(false)
    setSessionID('')
    setAPIKey('')
    setConnection('disconnected')
    setDegradedRealtime(false)
    setStatusText('signed out')
    setChatLines([])
    setChatInput('')
    setChatWaiting(false)
    activeRequestRef.current = null
  }

  function submitChat() {
    const text = chatInput.trim()
    if (!text || !sessionId) {
      return
    }
    appendChatLine({ role: 'user', text })
    setChatInput('')

    const socket = socketRef.current
    if (!socket?.isOpen()) {
      appendChatLine({ role: 'system', text: 'Realtime connection unavailable. Chat is disabled until websocket reconnects.' })
      return
    }

    const requestID = uid()
    activeRequestRef.current = requestID
    setChatWaiting(true)
    socket.send({
      type: 'ui.command',
      session_id: sessionId,
      request_id: requestID,
      command: 'ask',
      message: text,
    })
  }

  const visibleSignals = useMemo(() => {
    const all = signalsQuery.data ?? []
    return all.slice(signalsScrollOffset, signalsScrollOffset + SIGNAL_VISIBLE_ROWS)
  }, [signalsQuery.data, signalsScrollOffset])

  if (!authenticated) {
    return (
      <main className="console-shell console-shell--auth">
        <section className="auth-card">
          <h1>Web Operator Console</h1>
          <p>Use your REST API key to open a browser session for the TUI workflow.</p>
          <form onSubmit={handleLogin}>
            <input
              type="password"
              placeholder="X-API-Key"
              value={apiKeyInput}
              onChange={(event) => setAPIKeyInput(event.target.value)}
            />
            <button type="submit">Connect</button>
            {authError ? <div className="auth-error">{authError}</div> : null}
          </form>
        </section>
      </main>
    )
  }

  return (
    <main className="console-shell">
      <header className="topbar">
        <h1>Umbrella Operator Console</h1>
        <div className="topbar-right">
          <span className={`chip chip--${connection}`}>{connection}</span>
          <span className={`chip ${degradedRealtime ? 'chip--warn' : 'chip--ok'}`}>
            {degradedRealtime ? 'realtime degraded' : 'realtime live'}
          </span>
          <span className={`chip ${healthQuery.data ? 'chip--ok' : 'chip--warn'}`}>api {healthQuery.data ? 'ok' : 'degraded'}</span>
          <button type="button" onClick={handleLogout}>
            Logout
          </button>
        </div>
      </header>

      <nav className="tabs" aria-label="Console tabs">
        <button type="button" className={activeTab === 'dashboard' ? 'active' : ''} onClick={() => setActiveTab('dashboard')}>
          1:Dashboard
        </button>
        <button type="button" className={activeTab === 'chat' ? 'active' : ''} onClick={() => setActiveTab('chat')}>
          2:Chat
        </button>
        <button type="button" className={activeTab === 'signals' ? 'active' : ''} onClick={() => setActiveTab('signals')}>
          3:Signals
        </button>
        <button type="button" className={activeTab === 'backtest' ? 'active' : ''} onClick={() => setActiveTab('backtest')}>
          4:Backtest
        </button>
      </nav>

      <section className="workspace">
        <section className="main-pane">
          {activeTab === 'dashboard' ? (
            <div className="panel">
              <div className="panel-header">
                <h2>Dashboard</h2>
                <p>Auto-refresh every 10s. Press Shift+R to refresh now.</p>
              </div>
              <div className="grid two-up">
                <article className="card">
                  <h3>Live Prices</h3>
                  <div className="table-scroll">
                    <table>
                      <thead>
                        <tr>
                          <th>Symbol</th>
                          <th>Price</th>
                          <th>24h</th>
                          <th className="hide-mobile">Volume</th>
                        </tr>
                      </thead>
                      <tbody>
                        {(dashboardPricesQuery.data ?? []).map((p) => (
                          <tr key={p.symbol}>
                            <td>{p.symbol}</td>
                            <td>{formatPrice(p.price_usd)}</td>
                            <td className={p.change_24h_pct >= 0 ? 'pos' : 'neg'}>{p.change_24h_pct.toFixed(2)}%</td>
                            <td className="hide-mobile">{formatMoney(p.volume_24h)}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                </article>
                <article className="card">
                  <h3>24h Heat Map</h3>
                  <div className="heat-grid">
                    {(dashboardPricesQuery.data ?? []).map((price) => (
                      <article key={price.symbol} className="heat-tile" style={heatTileStyle(price.change_24h_pct)}>
                        <span className="heat-tile__symbol">{price.symbol}</span>
                        <span className={`heat-tile__change ${price.change_24h_pct >= 0 ? 'pos' : 'neg'}`}>
                          {price.change_24h_pct > 0 ? '+' : ''}
                          {price.change_24h_pct.toFixed(2)}%
                        </span>
                      </article>
                    ))}
                  </div>
                </article>
              </div>
              <article className="card">
                  <h3>Active Signals</h3>
                  <ul className="signal-list">
                    {(dashboardSignalsQuery.data ?? []).map((signal) => (
                      <li key={signal.id}>
                        <span>{signal.symbol}</span>
                        <span>{signal.indicator}</span>
                        <span className={`dir dir--${signal.direction}`}>{signal.direction.toUpperCase()}</span>
                        <span>R{signal.risk}</span>
                      </li>
                    ))}
                  </ul>
              </article>
            </div>
          ) : null}

          {activeTab === 'chat' ? (
            <div className="panel">
              <div className="panel-header">
                <h2>Chat</h2>
                <p>Enter submits. Realtime advisor status comes from WebSocket events.</p>
              </div>
              <div className="chat-log" aria-live="polite">
                {chatLines.map((line) => (
                  <article key={line.id} className={`chat-line chat-line--${line.role}`}>
                    <span>{new Date(line.at).toLocaleTimeString()}</span>
                    <p>{line.text}</p>
                  </article>
                ))}
                {chatWaiting ? (
                  <article className="chat-line chat-line--system">
                    <span>{new Date().toLocaleTimeString()}</span>
                    <p>Advisor is thinking...</p>
                  </article>
                ) : null}
              </div>
              <form
                className="chat-input"
                onSubmit={(event) => {
                  event.preventDefault()
                  submitChat()
                }}
              >
                <input
                  value={chatInput}
                  onChange={(event) => setChatInput(event.target.value)}
                  placeholder="Ask about markets, signals, or risk posture..."
                  autoComplete="off"
                />
                <button type="submit" disabled={chatWaiting}>
                  Send
                </button>
              </form>
            </div>
          ) : null}

          {activeTab === 'signals' ? (
            <div className="panel">
              <div className="panel-header">
                <h2>Signals</h2>
                <p>
                  Filters: [s] symbol {SIGNAL_SYMBOL_OPTIONS[signalSymbolIndex]} | [r] risk {SIGNAL_RISK_OPTIONS[signalRiskIndex]} | [i] indicator{' '}
                  {SIGNAL_INDICATOR_OPTIONS[signalIndicatorIndex]} | [j/k] scroll
                </p>
                <div className="touch-controls touch-controls--signals">
                  <button type="button" className="touch-pill" onClick={() => setSignalSymbolIndex((prev) => cycleIndex(prev, SIGNAL_SYMBOL_OPTIONS.length))}>
                    Symbol: {SIGNAL_SYMBOL_OPTIONS[signalSymbolIndex]}
                  </button>
                  <button type="button" className="touch-pill" onClick={() => setSignalRiskIndex((prev) => cycleIndex(prev, SIGNAL_RISK_OPTIONS.length))}>
                    Risk: {SIGNAL_RISK_OPTIONS[signalRiskIndex]}
                  </button>
                  <button
                    type="button"
                    className="touch-pill"
                    onClick={() => setSignalIndicatorIndex((prev) => cycleIndex(prev, SIGNAL_INDICATOR_OPTIONS.length))}
                  >
                    Indicator: {SIGNAL_INDICATOR_OPTIONS[signalIndicatorIndex]}
                  </button>
                </div>
              </div>
              <article className="card">
                <div className="table-scroll">
                  <table>
                    <thead>
                      <tr>
                        <th>ID</th>
                        <th>Symbol</th>
                        <th className="hide-mobile">Int</th>
                        <th className="hide-mobile">Indicator</th>
                        <th>Dir</th>
                        <th>Risk</th>
                        <th className="hide-mobile">Time</th>
                      </tr>
                    </thead>
                    <tbody>
                      {visibleSignals.map((signal: Signal) => (
                        <tr
                          key={signal.id}
                          className={hasSignalChart(signal) ? 'row-clickable' : undefined}
                          onClick={() => {
                            if (!hasSignalChart(signal)) {
                              return
                            }
                            if (isMobileViewport()) {
                              setSelectedSignalID(signal.id)
                              setSignalChartModalOpen(true)
                              return
                            }
                            setSelectedSignalID((current) => (current === signal.id ? null : signal.id))
                          }}
                        >
                          <td className={hasSignalChart(signal) ? 'chart-link' : undefined}>
                            {signal.id}
                            {hasSignalChart(signal) ? ' [chart]' : ''}
                          </td>
                          <td>{signal.symbol}</td>
                          <td className="hide-mobile">{signal.interval}</td>
                          <td className="hide-mobile">{signal.indicator}</td>
                          <td className={`dir dir--${signal.direction}`}>{signal.direction.toUpperCase()}</td>
                          <td>{signal.risk}</td>
                          <td className="hide-mobile">{new Date(signal.timestamp).toLocaleString()}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
                <p className="footnote">
                  Showing {Math.min(signalsScrollOffset + 1, signalsQuery.data?.length ?? 0)}-
                  {Math.min(signalsScrollOffset + SIGNAL_VISIBLE_ROWS, signalsQuery.data?.length ?? 0)} of {signalsQuery.data?.length ?? 0}
                </p>
                <section className="signal-chart-panel">
                  <h4>Signal Chart</h4>
                  {selectedSignalID === null ? (
                    <p className="footnote">Click a signal row marked with `[chart]` to view its chart.</p>
                  ) : null}
                  {selectedSignalID !== null && signalChartQuery.isLoading ? <p className="footnote">Loading chart...</p> : null}
                  {selectedSignalID !== null && signalChartQuery.isError ? (
                    <p className="footnote">Chart unavailable for signal #{selectedSignalID}.</p>
                  ) : null}
                  {selectedSignalID !== null && selectedSignalChartURL ? (
                    <img src={selectedSignalChartURL} alt={`Signal ${selectedSignalID} chart`} className="signal-chart-image" />
                  ) : null}
                </section>
              </article>
            </div>
          ) : null}

          {activeTab === 'backtest' ? (
            <div className="panel">
              <div className="panel-header">
                <h2>Backtest</h2>
                <p>View: {backtestView} | [v] toggle view | Shift+R refresh.</p>
                <div className="touch-controls touch-controls--views">
                  <button
                    type="button"
                    className={`touch-pill ${backtestView === 'accuracy' ? 'touch-pill--active' : ''}`}
                    aria-pressed={backtestView === 'accuracy'}
                    onClick={() => setBacktestView('accuracy')}
                  >
                    Accuracy
                  </button>
                  <button
                    type="button"
                    className={`touch-pill ${backtestView === 'predictions' ? 'touch-pill--active' : ''}`}
                    aria-pressed={backtestView === 'predictions'}
                    onClick={() => setBacktestView('predictions')}
                  >
                    Predictions
                  </button>
                </div>
              </div>
              {backtestView === 'accuracy' ? (
                <div className="grid two-up">
                  <article className="card">
                    <h3>Model Accuracy (All-Time)</h3>
                    {(backtestSummaryQuery.data ?? []).length === 0 ? (
                      <p className="footnote">No backtest summary data available.</p>
                    ) : (
                      <div className="accuracy-bars">
                        {(backtestSummaryQuery.data ?? []).map((item, idx) => {
                          const accuracy = clamp(readAccuracy(item), 0, 1)
                          return (
                            <div key={`${readModelKey(item)}-${idx}`} className="accuracy-row">
                              <span className="accuracy-label">{readModelKey(item)}</span>
                              <div className="accuracy-track">
                                <div className="accuracy-fill" style={{ width: `${Math.max(accuracy * 100, 2)}%` }} />
                              </div>
                              <span className="accuracy-meta">
                                {(accuracy * 100).toFixed(1)}% ({readTotal(item)})
                              </span>
                            </div>
                          )
                        })}
                      </div>
                    )}
                  </article>
                  <article className="card">
                    <h3>Daily Accuracy (30d)</h3>
                    <div className="table-scroll">
                      <table>
                        <thead>
                          <tr>
                            <th>Day</th>
                            <th>Accuracy</th>
                            <th>Correct</th>
                          </tr>
                        </thead>
                        <tbody>
                          {(backtestDailyQuery.data ?? []).slice(0, 30).map((item, idx) => (
                            <tr key={`${readDay(item)}-${idx}`}>
                              <td>{readDay(item).slice(0, 10)}</td>
                              <td>{(readAccuracy(item) * 100).toFixed(1)}%</td>
                              <td>
                                {readCorrect(item)}/{readTotal(item)}
                              </td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  </article>
                </div>
              ) : (
                <article className="card">
                  <h3>Recent Resolved Predictions</h3>
                  <div className="table-scroll">
                    <table>
                      <thead>
                        <tr>
                          <th>Symbol</th>
                          <th className="hide-mobile">Int</th>
                          <th className="hide-mobile">Model</th>
                          <th>Dir</th>
                          <th>Risk</th>
                          <th>Correct</th>
                          <th>Return</th>
                        </tr>
                      </thead>
                      <tbody>
                        {(backtestPredictionsQuery.data ?? []).map((pred, idx) => {
                          const correct = readPredictionCorrect(pred)
                          const ret = readPredictionReturn(pred)
                          return (
                            <tr key={`${readPredictionString(pred, 'symbol')}-${idx}`}>
                              <td>{readPredictionString(pred, 'symbol')}</td>
                              <td className="hide-mobile">{readPredictionString(pred, 'interval')}</td>
                              <td className="hide-mobile">{readPredictionString(pred, 'model_key')}</td>
                              <td>{readPredictionDirection(pred)}</td>
                              <td>{readPredictionRisk(pred)}</td>
                              <td>{correct === null ? '?' : correct ? 'YES' : 'NO'}</td>
                              <td>{ret === null ? 'n/a' : `${ret > 0 ? '+' : ''}${(ret * 100).toFixed(2)}%`}</td>
                            </tr>
                          )
                        })}
                      </tbody>
                    </table>
                  </div>
                </article>
              )}
            </div>
          ) : null}
        </section>
        <aside className="side-pane">
          <h3>Telemetry</h3>
          <p>Reserved for live status panes.</p>
          <div className="telemetry-item">
            <span>Session</span>
            <strong>{sessionId.slice(0, 12)}...</strong>
          </div>
          <div className="telemetry-item">
            <span>Status</span>
            <strong>{statusText}</strong>
          </div>
          <div className="telemetry-item">
            <span>Mode</span>
            <strong>{degradedRealtime ? 'REST fallback' : 'WS + REST hybrid'}</strong>
          </div>
        </aside>
      </section>

      {signalChartModalOpen ? (
        <section
          className="signal-chart-modal"
          role="dialog"
          aria-modal="true"
          aria-label="Signal chart"
          onClick={() => setSignalChartModalOpen(false)}
        >
          <article className="signal-chart-modal__content" onClick={(event) => event.stopPropagation()}>
            <header className="signal-chart-modal__header">
              <h3>Signal Chart</h3>
              <button type="button" onClick={() => setSignalChartModalOpen(false)}>
                Close
              </button>
            </header>
            {selectedSignalID !== null && signalChartQuery.isLoading ? <p className="footnote">Loading chart...</p> : null}
            {selectedSignalID !== null && signalChartQuery.isError ? <p className="footnote">Chart unavailable for signal #{selectedSignalID}.</p> : null}
            {selectedSignalID !== null && selectedSignalChartURL ? (
              <img src={selectedSignalChartURL} alt={`Signal ${selectedSignalID} chart`} className="signal-chart-image signal-chart-image--modal" />
            ) : null}
          </article>
        </section>
      ) : null}

      <footer className="statusline">
        <span>tab:{activeTab}</span>
        <span>session:{sessionId}</span>
        <span className="hide-mobile">hint: 1-4 tabs | Tab/Shift+Tab cycle | Shift+R refresh</span>
      </footer>
    </main>
  )
}
