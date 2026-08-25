import React, { useCallback, useEffect, useRef, useState } from 'react'
import { Layers, X } from 'lucide-react'
import { InteractiveTerminal } from './InteractiveTerminal'
import type { TerminalSession } from '../types'
import { useApp } from '../context/AppContext'

const WIDTH_KEY = 'taskacao_terminal_panel_width'
const MIN_WIDTH = 340
const MAX_WIDTH = 1000
const DEFAULT_WIDTH = 480

const readStoredWidth = (): number => {
  try {
    const raw = Number(localStorage.getItem(WIDTH_KEY))
    if (Number.isFinite(raw) && raw >= MIN_WIDTH) return Math.min(raw, MAX_WIDTH)
  } catch {
    // blocked storage: fall back to the default width
  }
  return DEFAULT_WIDTH
}

/**
 * The workspace CLI, docked to the right of the app. It is a real side panel,
 * not an overlay: the views shrink next to it so the board and the shell stay
 * usable at the same time.
 *
 * The PTY session lives server-side and is keyed on the project, so closing the
 * panel does not kill the shell — reopening replays its screen. One session per
 * project, plus a global one when no project is selected.
 */
export const WorkspaceTerminalPanel: React.FC = () => {
  const {
    isTerminalPanelOpen,
    setIsTerminalPanelOpen,
    currentProject,
    settings,
    chatTask,
    setChatTask,
    listTerminalSessions,
    resetTerminalSession,
    projects,
    tasks,
  } = useApp()

  const [width, setWidth] = useState<number>(readStoredWidth)
  const [isResizing, setIsResizing] = useState(false)
  const [sessions, setSessions] = useState<TerminalSession[]>([])
  const [showSessions, setShowSessions] = useState(false)
  const widthRef = useRef(width)

  useEffect(() => {
    widthRef.current = width
  }, [width])

  // La liste n'est relue qu'à l'ouverture du volet : c'est une information de
  // diagnostic, pas un flux à rafraîchir en continu.
  useEffect(() => {
    if (!showSessions) return
    listTerminalSessions().then(setSessions)
  }, [showSessions, listTerminalSessions])

  const onPointerDown = useCallback((e: React.PointerEvent) => {
    e.preventDefault()
    setIsResizing(true)
  }, [])

  useEffect(() => {
    if (!isResizing) return

    const onMove = (e: PointerEvent) => {
      const next = Math.min(MAX_WIDTH, Math.max(MIN_WIDTH, window.innerWidth - e.clientX))
      setWidth(next)
    }
    const onUp = () => {
      setIsResizing(false)
      try {
        localStorage.setItem(WIDTH_KEY, String(widthRef.current))
      } catch {
        // ignore
      }
    }

    window.addEventListener('pointermove', onMove)
    window.addEventListener('pointerup', onUp)
    return () => {
      window.removeEventListener('pointermove', onMove)
      window.removeEventListener('pointerup', onUp)
    }
  }, [isResizing])

  if (!isTerminalPanelOpen) return null

  // Une tâche ouverte prend la main sur le panneau : son PTY vit ici, ancré à
  // côté du board, plutôt que dans une modale qui recouvre tout.
  const sessionId = currentProject ? `project-${currentProject.id}` : 'global-workspace'
  const cwd = currentProject?.repoPath || settings.repoPath || ''
  const label = chatTask
    ? `CLI · ${chatTask.key}`
    : currentProject
      ? `CLI · ${currentProject.name}`
      : 'CLI · Workspace'

  const sessionLabel = (id: string): string => {
    if (id === 'global-workspace') return 'Workspace'
    if (id.startsWith('project-')) {
      const proj = projects.find(p => `project-${p.id}` === id)
      return proj ? `Projet ${proj.name}` : id
    }
    if (id.startsWith('task-')) {
      const task = tasks.find(t => `task-${t.id}` === id)
      return task ? `Tâche ${task.key}` : id
    }
    return id
  }

  return (
    <aside
      className="relative shrink-0 h-full flex bg-[var(--bg-primary)] border-l border-[var(--border-color)]"
      style={{ width }}
    >
      {/* Resize handle */}
      <div
        onPointerDown={onPointerDown}
        title="Redimensionner le panneau"
        className={`absolute left-0 top-0 h-full w-1.5 -ml-0.5 z-10 cursor-col-resize transition-colors ${
          isResizing ? 'bg-[var(--accent-color)]' : 'hover:bg-[var(--accent-color)]/60'
        }`}
      />

      <div className="flex-1 min-w-0 p-2 flex flex-col gap-2">
        {/* Barre des sessions : ce qui tourne encore, et de quoi y revenir. */}
        <div className="shrink-0 flex items-center gap-1.5 text-[10px]">
          <button
            type="button"
            onClick={() => setShowSessions(v => !v)}
            className={`flex items-center gap-1 px-2 py-1 rounded-md font-bold border transition-colors cursor-pointer ${
              showSessions
                ? 'bg-[var(--accent-light)] accent-text border-[var(--accent-color)]/40'
                : 'bg-[var(--bg-secondary)] text-[var(--text-secondary)] border-[var(--border-color)] hover:text-[var(--text-primary)]'
            }`}
            title="Sessions PTY en cours"
          >
            <Layers size={11} />
            <span>Sessions</span>
          </button>

          {chatTask && (
            <button
              type="button"
              onClick={() => setChatTask(null)}
              className="flex items-center gap-1 px-2 py-1 rounded-md font-bold bg-cyan-500/15 text-cyan-300 border border-cyan-500/40 hover:bg-cyan-500/25 transition-colors cursor-pointer"
              title="Revenir au terminal du projet"
            >
              <X size={10} />
              <span>{chatTask.key}</span>
            </button>
          )}
        </div>

        {showSessions && (
          <div className="shrink-0 max-h-40 overflow-y-auto rounded-xl border border-[var(--border-color)] bg-[var(--bg-secondary)] divide-y divide-[var(--border-color)]/60">
            {sessions.length === 0 ? (
              <p className="p-2 text-[10px] text-[var(--text-muted)]">Aucune session active.</p>
            ) : (
              sessions.map(sess => (
                <div key={sess.id} className="flex items-center gap-2 px-2 py-1.5">
                  <span className={`w-1.5 h-1.5 rounded-full shrink-0 ${sess.clients > 0 ? 'bg-emerald-400' : 'bg-[var(--text-muted)]'}`}
                    title={sess.clients > 0 ? `${sess.clients} vue(s) attachée(s)` : 'Aucune vue attachée, le shell tourne toujours'} />
                  <div className="min-w-0 flex-1">
                    <span className="block text-[10px] font-bold text-[var(--text-primary)] truncate">
                      {sessionLabel(sess.id)}
                    </span>
                    <span className="block text-[9px] font-mono text-[var(--text-muted)] truncate" title={sess.cwd}>
                      {sess.cwd}
                    </span>
                  </div>
                  {sess.id.startsWith('task-') && (
                    <button
                      type="button"
                      onClick={() => {
                        const task = tasks.find(t => `task-${t.id}` === sess.id)
                        if (task) setChatTask(task)
                      }}
                      className="px-1.5 py-0.5 rounded text-[9px] font-bold text-[var(--accent-color)] hover:underline cursor-pointer shrink-0"
                      title="Afficher cette session"
                    >
                      Ouvrir
                    </button>
                  )}
                  <button
                    type="button"
                    onClick={async () => {
                      await resetTerminalSession(sess.id)
                      listTerminalSessions().then(setSessions)
                    }}
                    className="p-0.5 rounded text-[var(--text-muted)] hover:text-rose-400 cursor-pointer shrink-0"
                    title="Terminer cette session"
                  >
                    <X size={10} />
                  </button>
                </div>
              ))
            )}
          </div>
        )}

        <div className="flex-1 min-h-0">
          {chatTask ? (
            <InteractiveTerminal
              key={`task-${chatTask.id}`}
              task={chatTask}
              label={label}
              onClose={() => setChatTask(null)}
            />
          ) : (
            <InteractiveTerminal
              key={sessionId}
              sessionId={sessionId}
              cwd={cwd}
              projectId={currentProject?.id}
              label={label}
              onClose={() => setIsTerminalPanelOpen(false)}
            />
          )}
        </div>
      </div>
    </aside>
  )
}
