import React, { useState, useEffect, useRef, useCallback } from 'react'
import {
  X,
  Send,
  Terminal,
  MessageSquare,
  Sparkles,
  Bot,
  User,
  Trash2,
  Copy,
  Check,
  Maximize2,
  Minimize2,
  Loader2,
  Square,
  GitBranch,
  FolderGit2,
  Folder,
  Code2,
  Flame,
  HelpCircle,
  FileCode,
  ShieldCheck,
  Zap,
} from 'lucide-react'
import { useApp } from '../context/AppContext'
import type { TaskMessage } from '../types'
import { InteractiveTerminal } from './InteractiveTerminal'

// Simple Markdown Renderer with Code Blocks and Copy Button
const MarkdownContent: React.FC<{ content: string }> = ({ content }) => {
  const [copiedIndex, setCopiedIndex] = useState<number | null>(null)

  const copyToClipboard = (text: string, index: number) => {
    navigator.clipboard.writeText(text)
    setCopiedIndex(index)
    setTimeout(() => setCopiedIndex(null), 2000)
  }

  // Split content by code blocks
  const parts = content.split(/(```[\s\S]*?```)/g)

  return (
    <div className="space-y-2 text-xs leading-relaxed text-[var(--text-primary)] break-words">
      {parts.map((part, idx) => {
        if (part.startsWith('```') && part.endsWith('```')) {
          const firstLineEnd = part.indexOf('\n')
          const lang = firstLineEnd !== -1 ? part.slice(3, firstLineEnd).trim() : ''
          const code = firstLineEnd !== -1 ? part.slice(firstLineEnd + 1, -3) : part.slice(3, -3)

          return (
            <div key={idx} className="my-2 rounded-xl overflow-hidden border border-slate-800 bg-slate-950 font-mono text-[11px] shadow-sm">
              <div className="flex items-center justify-between px-3 py-1.5 bg-slate-900/90 border-b border-slate-800 text-slate-400 text-[10px]">
                <span className="font-semibold uppercase tracking-wider">{lang || 'text'}</span>
                <button
                  type="button"
                  onClick={() => copyToClipboard(code, idx)}
                  className="flex items-center gap-1 hover:text-white transition-colors cursor-pointer"
                >
                  {copiedIndex === idx ? (
                    <>
                      <Check size={11} className="text-emerald-400" />
                      <span className="text-emerald-400">Copié !</span>
                    </>
                  ) : (
                    <>
                      <Copy size={11} />
                      <span>Copier</span>
                    </>
                  )}
                </button>
              </div>
              <pre className="p-3 overflow-x-auto text-slate-200 whitespace-pre leading-relaxed">
                <code>{code}</code>
              </pre>
            </div>
          )
        }

        // Render standard Markdown lines
        const lines = part.split('\n')
        return (
          <div key={idx} className="space-y-1">
            {lines.map((line, lineIdx) => {
              const trimmed = line.trim()
              if (!trimmed) return <div key={lineIdx} className="h-1.5" />

              // Headers
              if (trimmed.startsWith('### ')) {
                return <h4 key={lineIdx} className="font-bold text-sm text-[var(--text-primary)] mt-3 mb-1 text-[var(--accent-color)]">{trimmed.slice(4)}</h4>
              }
              if (trimmed.startsWith('## ')) {
                return <h3 key={lineIdx} className="font-bold text-base text-[var(--text-primary)] mt-3 mb-1.5">{trimmed.slice(3)}</h3>
              }
              if (trimmed.startsWith('# ')) {
                return <h2 key={lineIdx} className="font-bold text-lg text-[var(--text-primary)] mt-4 mb-2">{trimmed.slice(2)}</h2>
              }

              // Blockquote
              if (trimmed.startsWith('> ')) {
                return (
                  <blockquote key={lineIdx} className="pl-3 border-l-2 border-[var(--accent-color)] italic text-[var(--text-secondary)] my-1">
                    {trimmed.slice(2)}
                  </blockquote>
                )
              }

              // List items
              if (trimmed.startsWith('- ') || trimmed.startsWith('* ')) {
                return (
                  <div key={lineIdx} className="flex items-start gap-2 ml-2 my-0.5">
                    <span className="text-[var(--accent-color)] mt-1">•</span>
                    <span>{renderInlineFormatting(trimmed.slice(2))}</span>
                  </div>
                )
              }

              // Numbered list
              const numMatch = trimmed.match(/^(\d+)\.\s+(.*)/)
              if (numMatch) {
                return (
                  <div key={lineIdx} className="flex items-start gap-2 ml-2 my-0.5">
                    <span className="font-mono font-bold text-[var(--accent-color)] text-[10px] mt-0.5">{numMatch[1]}.</span>
                    <span>{renderInlineFormatting(numMatch[2])}</span>
                  </div>
                )
              }

              return <p key={lineIdx} className="my-0.5 leading-relaxed">{renderInlineFormatting(line)}</p>
            })}
          </div>
        )
      })}
    </div>
  )
}

function renderInlineFormatting(text: string): React.ReactNode {
  // Bold & inline code
  const tokens = text.split(/(\*\*.*?\*\*|`.*?`)/g)
  return tokens.map((token, i) => {
    if (token.startsWith('**') && token.endsWith('**')) {
      return <strong key={i} className="font-bold text-[var(--text-primary)]">{token.slice(2, -2)}</strong>
    }
    if (token.startsWith('`') && token.endsWith('`')) {
      return (
        <code key={i} className="px-1.5 py-0.5 rounded bg-[var(--bg-tertiary)] text-[var(--accent-color)] font-mono text-[11px] border border-[var(--border-color)]">
          {token.slice(1, -1)}
        </code>
      )
    }
    return token
  })
}

export const TaskChatDrawer: React.FC = () => {
  const {
    chatTask,
    setChatTask,
    getTaskMessages,
    sendTaskMessageStream,
    clearTaskMessages,
    settings,
    projects,
    setDiffTask,
    t,
  } = useApp()

  const [messages, setMessages] = useState<TaskMessage[]>([])
  const [inputText, setInputText] = useState('')
  const [isLoadingMessages, setIsLoadingMessages] = useState(false)
  const [isStreaming, setIsStreaming] = useState(false)
  const [streamingChunk, setStreamingChunk] = useState('')
  const [streamingSteps, setStreamingSteps] = useState<string[]>([])
  const [activeTab, setActiveTab] = useState<'chat' | 'terminal'>('chat')
  const [isFullscreen, setIsFullscreen] = useState(false)
  const [selectedSkillId, setSelectedSkillId] = useState<string>('')
  const [terminalLogs, setTerminalLogs] = useState<string[]>([])

  const messagesEndRef = useRef<HTMLDivElement>(null)
  const terminalEndRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLTextAreaElement>(null)

  // Fetch conversation history when chatTask changes
  useEffect(() => {
    if (!chatTask) {
      setMessages([])
      setTerminalLogs([])
      setStreamingChunk('')
      setStreamingSteps([])
      return
    }

    let isMounted = true
    setIsLoadingMessages(true)
    getTaskMessages(chatTask.id).then(msgs => {
      if (isMounted) {
        setMessages(msgs)
        setIsLoadingMessages(false)
        // Focus input
        setTimeout(() => inputRef.current?.focus(), 150)
      }
    })

    return () => {
      isMounted = false
    }
  }, [chatTask, getTaskMessages])

  // Scroll to bottom
  const scrollToBottom = useCallback(() => {
    if (activeTab === 'chat') {
      messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
    } else {
      terminalEndRef.current?.scrollIntoView({ behavior: 'smooth' })
    }
  }, [activeTab])

  useEffect(() => {
    scrollToBottom()
  }, [messages, streamingChunk, streamingSteps, terminalLogs, scrollToBottom])

  if (!chatTask) return null

  const handleClose = () => {
    setChatTask(null)
  }

  const handleSend = async (customPrompt?: string, skillOverride?: string) => {
    const textToSend = (customPrompt !== undefined ? customPrompt : inputText).trim()
    if (!textToSend || isStreaming) return

    const skillIdToUse = skillOverride !== undefined ? skillOverride : selectedSkillId

    // Optimistically add user message to list
    const optimisticUserMsg: TaskMessage = {
      id: 'temp-' + Date.now(),
      taskId: chatTask.id,
      role: 'user',
      content: textToSend,
      skillId: skillIdToUse,
      createdAt: new Date().toISOString(),
    }

    setMessages(prev => [...prev, optimisticUserMsg])
    setInputText('')
    setSelectedSkillId('')
    setIsStreaming(true)
    setStreamingChunk('')
    setStreamingSteps([])
    setTerminalLogs(prev => [
      ...prev,
      `[${new Date().toLocaleTimeString()}] > ${textToSend}`,
    ])

    const assistantMsg = await sendTaskMessageStream(
      chatTask.id,
      textToSend,
      skillIdToUse || undefined,
      chunk => {
        setStreamingChunk(prev => prev + chunk)
        setTerminalLogs(prev => [...prev, chunk])
      },
      step => {
        setStreamingSteps(prev => [...prev, step])
        setTerminalLogs(prev => [...prev, `[STEP] ${step}`])
      }
    )

    if (assistantMsg) {
      setMessages(prev => [...prev.filter(m => m.id !== optimisticUserMsg.id), optimisticUserMsg, assistantMsg])
    }

    setIsStreaming(false)
    setStreamingChunk('')
    setStreamingSteps([])
  }

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
      e.preventDefault()
      handleSend()
    }
  }

  const handleClearHistory = async () => {
    if (window.confirm(t.chat.clearHistory + ' ?')) {
      await clearTaskMessages(chatTask.id)
      setMessages([])
      setTerminalLogs([])
    }
  }

  const project = projects.find(p => p.id === chatTask.projectId)

  return (
    <div className="fixed inset-0 z-50 overflow-hidden select-none">
      {/* Backdrop */}
      <div
        onClick={handleClose}
        className="absolute inset-0 bg-black/50 backdrop-blur-2xs animate-in fade-in duration-200"
      />

      {/* Drawer */}
      <div
        className={`absolute inset-y-0 right-0 max-w-full flex transition-all duration-200 ${
          isFullscreen ? 'w-full' : 'w-full sm:w-[680px] md:w-[750px] lg:w-[840px]'
        }`}
      >
        <div className="w-full bg-[var(--bg-primary)] border-l border-[var(--border-color)] shadow-2xl flex flex-col h-full animate-in slide-in-from-right duration-200">
          
          {/* Top Header */}
          <div className="px-5 py-3 border-b border-[var(--border-color)] bg-[var(--bg-secondary)]/80 backdrop-blur-md shrink-0 flex items-center justify-between">
            <div className="flex items-center gap-3 min-w-0">
              {/* Task key badge */}
              <span className="font-mono text-xs font-bold text-[var(--accent-color)] bg-[var(--accent-light)] px-2.5 py-1 rounded-lg flex items-center gap-1.5 shrink-0 border border-[var(--accent-color)]/20 shadow-2xs">
                {chatTask.source === 'linear' && <span className="text-indigo-400 font-bold font-mono">◆</span>}
                {chatTask.source === 'github' && <FolderGit2 size={13} className="text-purple-400" />}
                {(!chatTask.source || chatTask.source === 'local') && <Folder size={13} className="text-emerald-400" />}
                {chatTask.key}
              </span>

              <div className="min-w-0">
                <h3 className="text-sm font-bold text-[var(--text-primary)] truncate max-w-md" title={chatTask.title}>
                  {chatTask.title}
                </h3>
                <div className="flex items-center gap-2 text-[10px] text-[var(--text-muted)] font-mono">
                  {chatTask.branchName && (
                    <span className="flex items-center gap-1 text-indigo-400 bg-indigo-500/10 px-1.5 py-0.2 rounded border border-indigo-500/20">
                      <GitBranch size={10} />
                      {chatTask.branchName}
                    </span>
                  )}
                  <span>• CWD: {project?.repoPath ? project.repoPath.split('/').pop() : 'fretzee-studio'}</span>
                  <span>• {settings.aiProvider.toUpperCase()}</span>
                </div>
              </div>
            </div>

            {/* Header Right Actions */}
            <div className="flex items-center gap-2 shrink-0">
              {/* View Switcher (Chat vs Terminal) */}
              <div className="flex items-center p-0.5 rounded-lg bg-[var(--bg-tertiary)] border border-[var(--border-color)]">
                <button
                  type="button"
                  onClick={() => setActiveTab('chat')}
                  className={`flex items-center gap-1.5 px-2.5 py-1 rounded-md text-[11px] font-semibold transition-all cursor-pointer ${
                    activeTab === 'chat'
                      ? 'bg-[var(--accent-color)] text-white shadow-xs'
                      : 'text-[var(--text-secondary)] hover:text-[var(--text-primary)]'
                  }`}
                >
                  <MessageSquare size={12} />
                  <span>{t.chat.viewChat}</span>
                </button>
                <button
                  type="button"
                  onClick={() => setActiveTab('terminal')}
                  className={`flex items-center gap-1.5 px-2.5 py-1 rounded-md text-[11px] font-semibold transition-all cursor-pointer ${
                    activeTab === 'terminal'
                      ? 'bg-slate-900 text-emerald-400 border border-slate-800 shadow-xs'
                      : 'text-[var(--text-secondary)] hover:text-[var(--text-primary)]'
                  }`}
                >
                  <Terminal size={12} className={isStreaming ? 'animate-pulse text-emerald-400' : ''} />
                  <span>{t.chat.viewTerminal}</span>
                </button>
              </div>

              {/* View Git Diff */}
              <button
                type="button"
                onClick={() => setDiffTask(chatTask)}
                className="p-1.5 rounded-lg text-indigo-400 hover:bg-indigo-500/10 transition-colors cursor-pointer border border-indigo-500/20"
                title="Inspecter le diff Git"
              >
                <Code2 size={15} />
              </button>

              {/* Clear History */}
              {messages.length > 0 && (
                <button
                  type="button"
                  onClick={handleClearHistory}
                  className="p-1.5 rounded-lg text-[var(--text-muted)] hover:text-rose-400 hover:bg-rose-500/10 transition-colors cursor-pointer"
                  title={t.chat.clearHistory}
                >
                  <Trash2 size={15} />
                </button>
              )}

              {/* Fullscreen Toggle */}
              <button
                type="button"
                onClick={() => setIsFullscreen(!isFullscreen)}
                className="p-1.5 rounded-lg text-[var(--text-muted)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-tertiary)] transition-colors cursor-pointer hidden sm:block"
                title={isFullscreen ? 'Réduire' : 'Plein écran'}
              >
                {isFullscreen ? <Minimize2 size={15} /> : <Maximize2 size={15} />}
              </button>

              {/* Close */}
              <button
                type="button"
                onClick={handleClose}
                className="p-1.5 rounded-lg text-[var(--text-muted)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-tertiary)] transition-colors cursor-pointer"
              >
                <X size={17} />
              </button>
            </div>
          </div>

          {/* Main Body */}
          <div className="flex-1 overflow-y-auto p-4 sm:p-6 space-y-4">
            
            {/* VIEW MODE A: CHAT CONVERSATION */}
            {activeTab === 'chat' && (
              <>
                {/* Initial Context Card */}
                <div className="p-3.5 rounded-2xl bg-[var(--bg-secondary)] border border-[var(--border-color)] shadow-2xs space-y-2 text-xs">
                  <div className="flex items-center justify-between text-[11px] font-bold text-[var(--text-secondary)]">
                    <span className="flex items-center gap-1.5">
                      <Bot size={13} className="text-[var(--accent-color)]" />
                      <span>Agent Copilot ({settings.aiProvider.toUpperCase()}) connecté au dépôt</span>
                    </span>
                    <span className="text-[10px] text-emerald-400 bg-emerald-500/10 px-2 py-0.5 rounded-full font-mono border border-emerald-500/20">
                      ● Prêt
                    </span>
                  </div>
                  {chatTask.description && (
                    <p className="text-[11px] text-[var(--text-secondary)] line-clamp-2 italic bg-[var(--bg-tertiary)]/50 p-2 rounded-lg border border-[var(--border-color)]/50 font-sans">
                      "{chatTask.description}"
                    </p>
                  )}
                </div>

                {/* Messages List */}
                {isLoadingMessages ? (
                  <div className="flex flex-col items-center justify-center py-12 text-[var(--text-muted)] space-y-2">
                    <Loader2 size={24} className="animate-spin text-[var(--accent-color)]" />
                    <span className="text-xs">Chargement de la discussion...</span>
                  </div>
                ) : messages.length === 0 && !isStreaming ? (
                  <div className="flex flex-col items-center justify-center py-10 px-4 text-center space-y-4 text-[var(--text-secondary)]">
                    <div className="w-12 h-12 rounded-2xl bg-[var(--accent-light)] flex items-center justify-center text-[var(--accent-color)] shadow-inner">
                      <Sparkles size={24} />
                    </div>
                    <div className="space-y-1 max-w-sm">
                      <h4 className="font-bold text-sm text-[var(--text-primary)]">Discuter de cette tâche avec l'agent</h4>
                      <p className="text-xs text-[var(--text-muted)] leading-relaxed">
                        L'agent possède le contexte complet du ticket, de sa branche Git et de son code source. Vous pouvez lui poser des questions ou lancer des compétences directes.
                      </p>
                    </div>

                    {/* Quick Suggestions Chips */}
                    <div className="grid grid-cols-1 sm:grid-cols-2 gap-2 w-full max-w-md pt-2 text-left">
                      <button
                        type="button"
                        onClick={() => handleSend('Clarifie cette tâche, identifie les ambiguïtés et formule les questions de cadrage.', 'clarify')}
                        className="p-2.5 rounded-xl bg-[var(--bg-secondary)] hover:bg-[var(--bg-tertiary)] border border-[var(--border-color)] hover:border-[var(--accent-color)]/50 text-xs transition-all flex items-center gap-2 group cursor-pointer"
                      >
                        <HelpCircle size={14} className="text-amber-400 shrink-0" />
                        <span className="font-medium text-[11px] group-hover:text-[var(--accent-color)]">🔍 Clarifier la tâche</span>
                      </button>

                      <button
                        type="button"
                        onClick={() => handleSend('Rédige une spécification technique Speckit détaillée pour implémenter cette tâche.', 'specify')}
                        className="p-2.5 rounded-xl bg-[var(--bg-secondary)] hover:bg-[var(--bg-tertiary)] border border-[var(--border-color)] hover:border-[var(--accent-color)]/50 text-xs transition-all flex items-center gap-2 group cursor-pointer"
                      >
                        <FileCode size={14} className="text-blue-400 shrink-0" />
                        <span className="font-medium text-[11px] group-hover:text-[var(--accent-color)]">📝 Rédiger la Spec Speckit</span>
                      </button>

                      <button
                        type="button"
                        onClick={() => handleSend('Implémente et écris concrètement les modifications de code pour cette tâche.', 'implement')}
                        className="p-2.5 rounded-xl bg-[var(--bg-secondary)] hover:bg-[var(--bg-tertiary)] border border-[var(--border-color)] hover:border-[var(--accent-color)]/50 text-xs transition-all flex items-center gap-2 group cursor-pointer"
                      >
                        <Flame size={14} className="text-indigo-400 shrink-0" />
                        <span className="font-medium text-[11px] group-hover:text-[var(--accent-color)]">💻 Coder la tâche</span>
                      </button>

                      <button
                        type="button"
                        onClick={() => handleSend('Exécute les tests unitaires et le build du projet pour valider le code.')}
                        className="p-2.5 rounded-xl bg-[var(--bg-secondary)] hover:bg-[var(--bg-tertiary)] border border-[var(--border-color)] hover:border-[var(--accent-color)]/50 text-xs transition-all flex items-center gap-2 group cursor-pointer"
                      >
                        <ShieldCheck size={14} className="text-emerald-400 shrink-0" />
                        <span className="font-medium text-[11px] group-hover:text-[var(--accent-color)]">🧪 Tester & Valider le build</span>
                      </button>
                    </div>
                  </div>
                ) : (
                  messages.map(msg => {
                    const isUser = msg.role === 'user'
                    return (
                      <div
                        key={msg.id}
                        className={`flex gap-3 ${isUser ? 'justify-end' : 'justify-start'} animate-in fade-in duration-150`}
                      >
                        {/* Agent Avatar */}
                        {!isUser && (
                          <div className="w-8 h-8 rounded-xl bg-[var(--accent-light)] flex items-center justify-center text-[var(--accent-color)] shrink-0 shadow-2xs mt-0.5 border border-[var(--accent-color)]/30">
                            <Bot size={16} />
                          </div>
                        )}

                        <div className={`space-y-1.5 max-w-[85%] sm:max-w-[80%]`}>
                          {/* Role and Time */}
                          <div className={`flex items-center gap-2 text-[10px] text-[var(--text-muted)] font-mono ${isUser ? 'justify-end' : 'justify-start'}`}>
                            <span className="font-bold">{isUser ? 'Vous' : `Agent Copilot (${settings.aiProvider.toUpperCase()})`}</span>
                            {msg.skillId && (
                              <span className="px-1.5 py-0.2 rounded bg-[var(--accent-light)] text-[var(--accent-color)] font-sans font-semibold">
                                #{msg.skillId}
                              </span>
                            )}
                            <span>{new Date(msg.createdAt).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}</span>
                          </div>

                          {/* Steps badges if any */}
                          {msg.steps && msg.steps.length > 0 && (
                            <div className="space-y-1 py-1">
                              {msg.steps.map((step, sIdx) => (
                                <div
                                  key={sIdx}
                                  className="text-[10px] font-mono px-2 py-1 rounded-md bg-[var(--bg-tertiary)]/70 text-[var(--text-secondary)] border border-[var(--border-color)]/70 flex items-center gap-1.5"
                                >
                                  <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 shrink-0" />
                                  <span className="truncate">{step}</span>
                                </div>
                              ))}
                            </div>
                          )}

                          {/* Message Bubble */}
                          <div
                            className={`p-3.5 rounded-2xl shadow-xs text-xs ${
                              isUser
                                ? 'bg-[var(--accent-color)] text-white rounded-tr-xs'
                                : 'bg-[var(--bg-secondary)] text-[var(--text-primary)] border border-[var(--border-color)] rounded-tl-xs'
                            }`}
                          >
                            {isUser ? (
                              <p className="whitespace-pre-wrap leading-relaxed">{msg.content}</p>
                            ) : (
                              <MarkdownContent content={msg.content} />
                            )}
                          </div>
                        </div>

                        {/* User Avatar */}
                        {isUser && (
                          <div className="w-8 h-8 rounded-xl bg-[var(--accent-color)] text-white flex items-center justify-center shrink-0 shadow-2xs mt-0.5 font-bold text-xs">
                            <User size={15} />
                          </div>
                        )}
                      </div>
                    )
                  })
                )}

                {/* Live Streaming Bubble */}
                {isStreaming && (
                  <div className="flex gap-3 justify-start animate-in fade-in duration-150">
                    <div className="w-8 h-8 rounded-xl bg-[var(--accent-light)] flex items-center justify-center text-[var(--accent-color)] shrink-0 shadow-2xs mt-0.5 border border-[var(--accent-color)]/30 animate-pulse">
                      <Bot size={16} />
                    </div>
                    <div className="space-y-2 max-w-[85%] sm:max-w-[80%]">
                      <div className="flex items-center gap-2 text-[10px] text-[var(--text-muted)] font-mono">
                        <span className="font-bold">Agent Copilot en direct...</span>
                        <Loader2 size={11} className="animate-spin text-[var(--accent-color)]" />
                      </div>

                      {/* Live Steps arriving */}
                      {streamingSteps.length > 0 && (
                        <div className="space-y-1">
                          {streamingSteps.map((step, sIdx) => (
                            <div
                              key={sIdx}
                              className="text-[10px] font-mono px-2 py-1 rounded-md bg-[var(--bg-tertiary)] text-[var(--text-secondary)] border border-[var(--border-color)] flex items-center gap-1.5 animate-in slide-in-from-left duration-150"
                            >
                              <span className="w-1.5 h-1.5 rounded-full bg-cyan-400 animate-pulse shrink-0" />
                              <span className="truncate">{step}</span>
                            </div>
                          ))}
                        </div>
                      )}

                      {/* Streaming Content */}
                      <div className="p-3.5 rounded-2xl shadow-xs text-xs bg-[var(--bg-secondary)] text-[var(--text-primary)] border border-[var(--border-color)] rounded-tl-xs">
                        {streamingChunk ? (
                          <MarkdownContent content={streamingChunk} />
                        ) : (
                          <div className="flex items-center gap-2 text-xs text-[var(--text-muted)] py-1">
                            <Loader2 size={13} className="animate-spin text-[var(--accent-color)]" />
                            <span>{t.chat.agentTyping}</span>
                          </div>
                        )}
                      </div>
                    </div>
                  </div>
                )}

                <div ref={messagesEndRef} />
              </>
            )}

            {/* VIEW MODE B: INTERACTIVE ZSH TERMINAL (PTY) */}
            {activeTab === 'terminal' && (
              <div className="h-full flex flex-col -m-2">
                <InteractiveTerminal task={chatTask} isExpanded={isFullscreen} />
              </div>
            )}
          </div>

          {/* Input & Skills Bar (Active in Chat mode) */}
          {activeTab === 'chat' && (
            <div className="p-3 sm:p-4 border-t border-[var(--border-color)] bg-[var(--bg-secondary)]/90 backdrop-blur-md shrink-0 space-y-2.5">
              
              {/* Quick Skills / Slash Actions */}
              <div className="flex items-center gap-1.5 overflow-x-auto pb-1 text-[11px] no-scrollbar">
                <span className="text-[10px] font-bold uppercase tracking-wider text-[var(--text-muted)] shrink-0 mr-1 flex items-center gap-1">
                  <Zap size={11} className="text-[var(--accent-color)]" />
                  Actions :
                </span>

                <button
                  type="button"
                  onClick={() => handleSend('Analyse la tâche et pose les questions clés de clarification.', 'clarify')}
                  className="px-2 py-0.5 rounded-lg bg-[var(--bg-tertiary)] hover:bg-amber-500/20 text-amber-300 border border-[var(--border-color)] hover:border-amber-500/40 text-[10px] font-bold shrink-0 transition-colors cursor-pointer"
                >
                  🔍 /clarify
                </button>

                <button
                  type="button"
                  onClick={() => handleSend('Rédige la spécification technique Speckit pour ce ticket.', 'specify')}
                  className="px-2 py-0.5 rounded-lg bg-[var(--bg-tertiary)] hover:bg-blue-500/20 text-blue-300 border border-[var(--border-color)] hover:border-blue-500/40 text-[10px] font-bold shrink-0 transition-colors cursor-pointer"
                >
                  📝 /specify
                </button>

                <button
                  type="button"
                  onClick={() => handleSend('Implémente et écris directement les modifications de code pour ce ticket.', 'implement')}
                  className="px-2 py-0.5 rounded-lg bg-[var(--bg-tertiary)] hover:bg-indigo-500/20 text-indigo-300 border border-[var(--border-color)] hover:border-indigo-500/40 text-[10px] font-bold shrink-0 transition-colors cursor-pointer"
                >
                  💻 /code
                </button>

                <button
                  type="button"
                  onClick={() => handleSend('Lance les tests et vérifie la compilation du projet.')}
                  className="px-2 py-0.5 rounded-lg bg-[var(--bg-tertiary)] hover:bg-emerald-500/20 text-emerald-300 border border-[var(--border-color)] hover:border-emerald-500/40 text-[10px] font-bold shrink-0 transition-colors cursor-pointer"
                >
                  🧪 /test
                </button>

                <button
                  type="button"
                  onClick={() => handleSend('Vérifie les fichiers modifiés, commite et crée la Pull Request.', 'create_pr')}
                  className="px-2 py-0.5 rounded-lg bg-[var(--bg-tertiary)] hover:bg-purple-500/20 text-purple-300 border border-[var(--border-color)] hover:border-purple-500/40 text-[10px] font-bold shrink-0 transition-colors cursor-pointer"
                >
                  🚀 /pr
                </button>
              </div>

              {/* Input form */}
              <div className="relative flex items-end gap-2">
                <div className="relative flex-1">
                  <textarea
                    ref={inputRef}
                    rows={2}
                    value={inputText}
                    onChange={e => setInputText(e.target.value)}
                    onKeyDown={handleKeyDown}
                    placeholder={t.chat.sendPlaceholder}
                    disabled={isStreaming}
                    className="w-full px-3.5 py-2.5 text-xs rounded-2xl bg-[var(--bg-tertiary)] border border-[var(--border-color)] text-[var(--text-primary)] placeholder-[var(--text-muted)] focus:outline-none focus:border-[var(--accent-color)] resize-none leading-relaxed shadow-inner"
                  />
                </div>

                {isStreaming ? (
                  <button
                    type="button"
                    onClick={() => setIsStreaming(false)}
                    className="h-10 px-4 rounded-xl bg-rose-500 hover:bg-rose-600 text-white font-bold text-xs flex items-center gap-1.5 shadow-md transition-all cursor-pointer shrink-0"
                  >
                    <Square size={13} fill="currentColor" />
                    <span>{t.chat.cancelStream}</span>
                  </button>
                ) : (
                  <button
                    type="button"
                    disabled={!inputText.trim()}
                    onClick={() => handleSend()}
                    className="h-10 px-4 rounded-xl bg-[var(--accent-color)] hover:opacity-90 disabled:opacity-40 text-white font-bold text-xs flex items-center gap-1.5 shadow-md transition-all cursor-pointer shrink-0"
                  >
                    <span>{t.chat.send}</span>
                    <Send size={13} />
                  </button>
                )}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
