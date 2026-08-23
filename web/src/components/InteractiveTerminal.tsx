import React, { useEffect, useRef, useState, useCallback } from 'react'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { WebLinksAddon } from '@xterm/addon-web-links'
import '@xterm/xterm/css/xterm.css'
import {
  Terminal as TerminalIcon,
  RotateCcw,
  Sparkles,
  Zap,
  AlertCircle,
  GitBranch,
  Code2,
} from 'lucide-react'
import type { Task } from '../types'
import { useApp } from '../context/AppContext'

interface InteractiveTerminalProps {
  task: Task
  isExpanded?: boolean
}

export const InteractiveTerminal: React.FC<InteractiveTerminalProps> = ({ task, isExpanded }) => {
  const { settings, openInEditor } = useApp()
  const terminalContainerRef = useRef<HTMLDivElement>(null)
  const termRef = useRef<Terminal | null>(null)
  const fitAddonRef = useRef<FitAddon | null>(null)
  const wsRef = useRef<WebSocket | null>(null)

  const [status, setStatus] = useState<'connecting' | 'connected' | 'disconnected' | 'error'>('connecting')

  // Connect WebSocket to Go backend PTY
  const connectWs = useCallback(() => {
    if (wsRef.current) {
      wsRef.current.close()
    }

    setStatus('connecting')

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    // In dev mode (Vite at :5173), connect to backend port :8090, or fallback to current host
    const host = window.location.port === '5173' ? `${window.location.hostname}:8090` : window.location.host
    const wsUrl = `${protocol}//${host}/ws/terminal?taskId=${encodeURIComponent(task.id)}`

    try {
      const ws = new WebSocket(wsUrl)
      ws.binaryType = 'arraybuffer'
      wsRef.current = ws

      ws.onopen = () => {
        setStatus('connected')
        // Send initial resize
        if (fitAddonRef.current && termRef.current) {
          fitAddonRef.current.fit()
          const { cols, rows } = termRef.current
          ws.send(JSON.stringify({ type: 'resize', cols, rows }))
        }
      }

      ws.onmessage = (event) => {
        if (termRef.current) {
          if (event.data instanceof ArrayBuffer) {
            const bytes = new Uint8Array(event.data)
            termRef.current.write(bytes)
          } else if (typeof event.data === 'string') {
            try {
              const msg = JSON.parse(event.data)
              if (msg.type === 'data') {
                termRef.current.write(msg.data)
              }
            } catch {
              termRef.current.write(event.data)
            }
          }
        }
      }

      ws.onclose = () => {
        setStatus('disconnected')
      }

      ws.onerror = (err) => {
        console.error('Terminal WebSocket error:', err)
        setStatus('error')
      }
    } catch (err: any) {
      console.error('WebSocket connection error:', err)
      setStatus('error')
    }
  }, [task.id])

  // Initialize Xterm.js
  useEffect(() => {
    if (!terminalContainerRef.current) return

    const term = new Terminal({
      cursorBlink: true,
      cursorStyle: 'bar',
      fontSize: 12,
      lineHeight: 1.25,
      fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
      theme: {
        background: '#0a0f1d',
        foreground: '#e2e8f0',
        cursor: '#818cf8',
        cursorAccent: '#0a0f1d',
        selectionBackground: '#4338ca55',
        black: '#0f172a',
        red: '#f87171',
        green: '#4ade80',
        yellow: '#facc15',
        blue: '#60a5fa',
        magenta: '#c084fc',
        cyan: '#38bdf8',
        white: '#f1f5f9',
        brightBlack: '#475569',
        brightRed: '#ef4444',
        brightGreen: '#22c55e',
        brightYellow: '#eab308',
        brightBlue: '#3b82f6',
        brightMagenta: '#a855f7',
        brightCyan: '#06b6d4',
        brightWhite: '#ffffff',
      },
      convertEol: true,
      allowProposedApi: true,
    })

    const fitAddon = new FitAddon()
    const webLinksAddon = new WebLinksAddon()

    term.loadAddon(fitAddon)
    term.loadAddon(webLinksAddon)

    term.open(terminalContainerRef.current)
    fitAddon.fit()

    termRef.current = term
    fitAddonRef.current = fitAddon

    // Handle user keystrokes in xterm
    term.onData((data) => {
      if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
        wsRef.current.send(JSON.stringify({ type: 'input', data }))
      }
    })

    // Resize observer to auto fit when container dimensions change
    const resizeObserver = new ResizeObserver(() => {
      if (fitAddonRef.current && termRef.current && wsRef.current?.readyState === WebSocket.OPEN) {
        try {
          fitAddonRef.current.fit()
          const { cols, rows } = termRef.current
          wsRef.current.send(JSON.stringify({ type: 'resize', cols, rows }))
        } catch {
          // ignore transient resize errors
        }
      }
    })

    resizeObserver.observe(terminalContainerRef.current)

    // Connect WebSocket
    connectWs()

    return () => {
      resizeObserver.disconnect()
      if (wsRef.current) {
        wsRef.current.close()
      }
      term.dispose()
    }
  }, [connectWs])

  // Handle expand resize
  useEffect(() => {
    const timer = setTimeout(() => {
      if (fitAddonRef.current && termRef.current && wsRef.current?.readyState === WebSocket.OPEN) {
        fitAddonRef.current.fit()
        const { cols, rows } = termRef.current
        wsRef.current.send(JSON.stringify({ type: 'resize', cols, rows }))
      }
    }, 200)
    return () => clearTimeout(timer)
  }, [isExpanded])

  // Helper to send text / command into terminal
  const sendCommand = (cmd: string) => {
    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify({ type: 'input', data: cmd }))
      termRef.current?.focus()
    }
  }

  // Quick Action commands
  const aiCmd = settings.aiProvider || 'agy'
  const handleLaunchAgent = () => {
    sendCommand(`${aiCmd}\n`)
  }

  const handleRunSkill = (skill: string) => {
    switch (skill) {
      case 'clarify':
        sendCommand(`${aiCmd} -p "/clarify-issue"\n`)
        break
      case 'specify':
        sendCommand(`${aiCmd} -p "/specify-issue"\n`)
        break
      case 'implement':
        sendCommand(`${aiCmd} -p "/code-issue"\n`)
        break
      case 'pr':
        sendCommand(`${aiCmd} -p "/create-pr"\n`)
        break
      case 'test':
        sendCommand(`npm test || go test ./...\n`)
        break
      case 'git_status':
        sendCommand(`git status -s\n`)
        break
    }
  }

  const handleInterrupt = () => {
    sendCommand('\x03') // Ctrl+C
  }

  const handleClear = () => {
    termRef.current?.clear()
    sendCommand('clear\n')
  }

  const handleResetSession = async () => {
    try {
      await fetch('/api/terminal/reset', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ taskId: task.id }),
      })
      termRef.current?.clear()
      connectWs()
    } catch (e) {
      console.error('Reset session error:', e)
    }
  }

  return (
    <div className="flex flex-col h-full bg-[#070b14] border border-slate-800/80 rounded-xl overflow-hidden shadow-2xl">
      {/* Top Bar: Session Info & Status */}
      <div className="flex items-center justify-between px-3 py-2 bg-slate-900/90 border-b border-slate-800/80 text-xs select-none">
        <div className="flex items-center gap-2 overflow-hidden">
          <div className="flex items-center gap-1.5 font-mono text-[11px] text-slate-300 font-semibold">
            <TerminalIcon size={14} className="text-indigo-400" />
            <span>ZSH Session</span>
          </div>

          <span className="text-slate-600">•</span>

          {/* Status Indicator */}
          {status === 'connected' && (
            <div className="flex items-center gap-1 text-[10px] text-emerald-400 bg-emerald-500/10 border border-emerald-500/20 px-2 py-0.5 rounded-full font-medium">
              <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse" />
              <span>Connecté (Live PTY)</span>
            </div>
          )}
          {status === 'connecting' && (
            <div className="flex items-center gap-1 text-[10px] text-amber-400 bg-amber-500/10 border border-amber-500/20 px-2 py-0.5 rounded-full font-medium">
              <span className="w-1.5 h-1.5 rounded-full bg-amber-400 animate-ping" />
              <span>Connexion...</span>
            </div>
          )}
          {(status === 'disconnected' || status === 'error') && (
            <div className="flex items-center gap-1.5">
              <span className="flex items-center gap-1 text-[10px] text-red-400 bg-red-500/10 border border-red-500/20 px-2 py-0.5 rounded-full font-medium">
                <AlertCircle size={10} />
                <span>Déconnecté</span>
              </span>
              <button
                type="button"
                onClick={connectWs}
                className="text-[10px] text-indigo-400 hover:text-indigo-300 underline cursor-pointer"
              >
                Reconnecter
              </button>
            </div>
          )}
        </div>

        {/* Task Worktree & Branch badges */}
        <div className="flex items-center gap-2">
          {task.branchName && (
            <div className="hidden sm:flex items-center gap-1 text-[10px] font-mono text-purple-300 bg-purple-500/10 border border-purple-500/20 px-2 py-0.5 rounded-md">
              <GitBranch size={10} />
              <span className="truncate max-w-[140px]">{task.branchName}</span>
            </div>
          )}

          <div className="flex items-center gap-1">
            <button
              type="button"
              onClick={handleInterrupt}
              title="Envoyer Ctrl+C (Interrompre le processus en cours)"
              className="px-2 py-1 bg-red-500/10 hover:bg-red-500/20 border border-red-500/30 text-red-300 text-[10px] rounded font-mono font-medium transition-colors cursor-pointer"
            >
              Ctrl+C
            </button>
            <button
              type="button"
              onClick={handleClear}
              title="Effacer l'écran"
              className="px-2 py-1 bg-slate-800 hover:bg-slate-700 border border-slate-700 text-slate-300 text-[10px] rounded font-mono transition-colors cursor-pointer"
            >
              Clear
            </button>
            <button
              type="button"
              onClick={handleResetSession}
              title="Redémarrer une nouvelle session shell vierge"
              className="flex items-center gap-1 px-2 py-1 bg-slate-800 hover:bg-slate-700 border border-slate-700 text-slate-300 text-[10px] rounded font-mono transition-colors cursor-pointer"
            >
              <RotateCcw size={10} />
              <span>Reset</span>
            </button>
          </div>
        </div>
      </div>

      {/* Quick Agent Actions Bar */}
      <div className="flex items-center gap-1.5 px-3 py-1.5 bg-slate-950/80 border-b border-slate-800/60 overflow-x-auto text-[11px] scrollbar-none">
        <span className="text-slate-400 font-mono text-[10px] flex items-center gap-1 shrink-0 mr-1">
          <Zap size={11} className="text-amber-400" />
          <span>Actions rapides :</span>
        </span>

        <button
          type="button"
          onClick={handleLaunchAgent}
          className="flex items-center gap-1 px-2.5 py-1 bg-indigo-600/20 hover:bg-indigo-600/30 border border-indigo-500/40 text-indigo-300 rounded-md font-mono text-[11px] transition-all shrink-0 cursor-pointer shadow-sm hover:scale-[1.02]"
        >
          <Sparkles size={11} className="text-indigo-400" />
          <span className="font-semibold">Lancer {aiCmd}</span>
        </button>

        <button
          type="button"
          onClick={() => handleRunSkill('clarify')}
          className="flex items-center gap-1 px-2 py-1 bg-slate-900 hover:bg-slate-800 border border-slate-800 text-slate-300 hover:text-white rounded-md font-mono text-[10.5px] transition-all shrink-0 cursor-pointer"
        >
          <span>/clarify</span>
        </button>

        <button
          type="button"
          onClick={() => handleRunSkill('specify')}
          className="flex items-center gap-1 px-2 py-1 bg-slate-900 hover:bg-slate-800 border border-slate-800 text-slate-300 hover:text-white rounded-md font-mono text-[10.5px] transition-all shrink-0 cursor-pointer"
        >
          <span>/specify</span>
        </button>

        <button
          type="button"
          onClick={() => handleRunSkill('implement')}
          className="flex items-center gap-1 px-2 py-1 bg-emerald-500/10 hover:bg-emerald-500/20 border border-emerald-500/30 text-emerald-300 rounded-md font-mono text-[10.5px] transition-all shrink-0 cursor-pointer"
        >
          <span>/code</span>
        </button>

        <button
          type="button"
          onClick={() => handleRunSkill('pr')}
          className="flex items-center gap-1 px-2 py-1 bg-purple-500/10 hover:bg-purple-500/20 border border-purple-500/30 text-purple-300 rounded-md font-mono text-[10.5px] transition-all shrink-0 cursor-pointer"
        >
          <span>/create-pr</span>
        </button>

        <button
          type="button"
          onClick={() => handleRunSkill('git_status')}
          className="flex items-center gap-1 px-2 py-1 bg-slate-900 hover:bg-slate-800 border border-slate-800 text-slate-400 hover:text-slate-200 rounded-md font-mono text-[10.5px] transition-all shrink-0 cursor-pointer"
        >
          <span>git status</span>
        </button>

        <button
          type="button"
          onClick={() => openInEditor({ taskId: task.id })}
          className="flex items-center gap-1 px-2 py-1 bg-cyan-500/10 hover:bg-cyan-500/20 border border-cyan-500/30 text-cyan-300 rounded-md font-mono text-[10.5px] transition-all shrink-0 cursor-pointer"
          title={`Ouvrir le dossier dans ${settings.editorCommand || 'VS Code'}`}
        >
          <Code2 size={11} className="text-cyan-400" />
          <span>{settings.editorCommand || 'code'}</span>
        </button>
      </div>

      {/* Terminal Viewport */}
      <div
        ref={terminalContainerRef}
        className="flex-1 w-full h-full p-2 overflow-hidden bg-[#0a0f1d]"
        style={{ minHeight: '320px' }}
      />
    </div>
  )
}
