import React, { useCallback, useEffect, useRef, useState } from 'react'
import { Check, Layers, X } from 'lucide-react'
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
    terminalSessionOverride,
    openTerminalSession,
    setSelectedTask,
    pendingInteractive,
    confirmInteractiveStep,
    dismissInteractiveStep,
    projects,
    tasks,
  } = useApp()

  const [width, setWidth] = useState<number>(readStoredWidth)
  const [isResizing, setIsResizing] = useState(false)
  const [sessions, setSessions] = useState<TerminalSession[]>([])
  // Plein écran : le panneau ancré couvre toute la fenêtre. Le terminal de la
  // fiche tâche avait déjà ce bouton, le panneau ne l'avait jamais eu, alors
  // qu'il est devenu la maison du CLI.
  const [isFullscreen, setIsFullscreen] = useState(false)
  const widthRef = useRef(width)

  useEffect(() => {
    widthRef.current = width
  }, [width])

  // La liste n'est relue qu'à l'ouverture du volet : c'est une information de
  // diagnostic, pas un flux à rafraîchir en continu.
  useEffect(() => {
    // Interrogée tant que le panneau est ouvert : c'est cette liste qui alimente
    // le sélecteur de session et qui dit si un agent tourne déjà dans la console
    // affichée, donc l'état du bouton de démarrage.
    listTerminalSessions().then(setSessions)
    const timer = window.setInterval(() => {
      listTerminalSessions().then(setSessions)
    }, 5000)
    return () => window.clearInterval(timer)
  }, [isTerminalPanelOpen, chatTask?.id, listTerminalSessions])

  useEffect(() => {
    if (!isFullscreen) return
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setIsFullscreen(false)
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [isFullscreen])

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
  // L'état de l'agent est porté par la liste des sessions, rafraîchie toutes les
  // cinq secondes : elle dit si un agent tourne déjà dans cette console.
  const agentRunning = chatTask
    ? Boolean(sessions.find(sess => sess.id === `task-${chatTask.id}`)?.agentRunning)
    : false

  const sessionId = currentProject ? `project-${currentProject.id}` : 'global-workspace'
  const cwd = currentProject?.repoPath || settings.repoPath || ''
  const label = chatTask
    ? `CLI · ${chatTask.key}`
    : currentProject
      ? `CLI · ${currentProject.name}`
      : 'CLI · Workspace'

  // La session réellement montrée : celle d'une tâche, celle ouverte par
  // identifiant, ou celle du projet à défaut. C'est la valeur du sélecteur.
  const displayedSessionId = chatTask
    ? `task-${chatTask.id}`
    : terminalSessionOverride || sessionId

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

  // Trié sur le libellé affiché, pas sur l'ordre reçu : l'ordre doit être celui
  // qu'on lit, et il ne doit pas changer parce qu'une console a écrit une ligne.
  const sortedSessions = [...sessions].sort((a, b) =>
    sessionLabel(a.id).localeCompare(sessionLabel(b.id), undefined, { numeric: true })
  )

  const sessionsList = isFullscreen ? (
          <div
            className={`shrink-0 overflow-y-auto rounded-xl border border-[var(--border-color)] bg-[var(--bg-secondary)] divide-y divide-[var(--border-color)]/60 ${
              isFullscreen ? 'w-60 h-full' : 'mt-2 max-h-36'
            }`}
          >
            {sessions.length === 0 ? (
              <p className="p-2 text-[10px] text-[var(--text-muted)]">Aucune session active.</p>
            ) : (
              sortedSessions.map(sess => (
                <div
                  key={sess.id}
                  role="button"
                  tabIndex={0}
                  onClick={() => {
                    // La ligne entière ouvre la session : viser un lien de trois
                    // lettres pour changer de console n'avait aucun intérêt.
                    const task = tasks.find(t => `task-${t.id}` === sess.id)
                    if (task) setChatTask(task)
                    else openTerminalSession(sess.id)
                  }}
                  onKeyDown={e => {
                    if (e.key !== 'Enter' && e.key !== ' ') return
                    e.preventDefault()
                    const task = tasks.find(t => `task-${t.id}` === sess.id)
                    if (task) setChatTask(task)
                    else openTerminalSession(sess.id)
                  }}
                  className="flex items-center gap-2 px-2 py-1.5 cursor-pointer hover:bg-[var(--bg-tertiary)] transition-colors"
                  title="Afficher cette session"
                >
                  <span className={`w-1.5 h-1.5 rounded-full shrink-0 ${sess.clients > 0 ? 'bg-emerald-400' : 'bg-[var(--text-muted)]'}`}
                    title={sess.clients > 0 ? `${sess.clients} vue(s) attachée(s)` : 'Aucune vue attachée, le shell tourne toujours'} />
                  <div className="min-w-0 flex-1">
                    <span className="flex items-center gap-1.5 text-[10px] font-bold text-[var(--text-primary)] truncate">
                      <span className="truncate">{sessionLabel(sess.id)}</span>
                      {sess.agentRunning && (
                        <span
                          className="shrink-0 text-[8px] font-mono px-1 rounded accent-text bg-[var(--accent-light)]"
                          title="Un agent tourne dans cette session"
                        >
                          agent
                        </span>
                      )}
                      {chatTask && sess.id === `task-${chatTask.id}` && (
                        <span className="shrink-0 text-[8px] font-mono px-1 rounded bg-[var(--bg-tertiary)] text-[var(--text-muted)]">
                          affichée
                        </span>
                      )}
                    </span>
                    <span className="block text-[9px] font-mono text-[var(--text-muted)] truncate" title={sess.cwd}>
                      {sess.cwd}
                    </span>
                  </div>
                  <button
                    type="button"
                    onClick={async e => {
                      e.stopPropagation()
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
  ) : null

  return (
    <aside
      className={
        isFullscreen
          ? 'fixed inset-0 z-50 flex bg-[var(--bg-primary)]'
          : 'relative shrink-0 h-full flex bg-[var(--bg-primary)] border-l border-[var(--border-color)]'
      }
      style={isFullscreen ? undefined : { width }}
    >
      {/* Poignée de redimensionnement, sans objet en plein écran */}
      {!isFullscreen && (
        <div
          onPointerDown={onPointerDown}
          title="Redimensionner le panneau"
          className={`absolute left-0 top-0 h-full w-1.5 -ml-0.5 z-10 cursor-col-resize transition-colors ${
            isResizing ? 'bg-[var(--accent-color)]' : 'hover:bg-[var(--accent-color)]/60'
          }`}
        />
      )}

      <div className="flex-1 min-w-0 p-2 flex flex-col gap-2">
        {/* Sessions : un sélecteur en haut quand la place manque, la liste en
            colonne à gauche seulement en plein écran. Une liste dépliée dans un
            panneau étroit mangeait la hauteur du terminal. */}
        <div className="shrink-0 flex items-center gap-1.5 text-[10px]">
          {!isFullscreen && sessions.length > 0 && (
            <>
              <Layers size={11} className="text-[var(--text-muted)] shrink-0" />
              <select
                value={displayedSessionId}
                onChange={e => {
                  const id = e.target.value
                  const task = tasks.find(t => `task-${t.id}` === id)
                  if (task) setChatTask(task)
                  else openTerminalSession(id)
                }}
                className="min-w-0 flex-1 px-1.5 py-1 rounded-md font-bold bg-[var(--bg-secondary)] text-[var(--text-secondary)] border border-[var(--border-color)] focus:outline-none focus:border-[var(--accent-color)] cursor-pointer"
                title="Session affichée dans ce panneau"
              >
                {sessions.every(sess => sess.id !== displayedSessionId) && (
                  <option value={displayedSessionId}>{sessionLabel(displayedSessionId)}</option>
                )}
                {sortedSessions.map(sess => (
                  <option key={sess.id} value={sess.id}>
                    {sessionLabel(sess.id)}
                    {sess.agentRunning ? ' · agent' : ''}
                  </option>
                ))}
              </select>
              <button
                type="button"
                onClick={async () => {
                  await resetTerminalSession(displayedSessionId)
                  listTerminalSessions().then(setSessions)
                }}
                className="p-1 rounded-md text-[var(--text-muted)] hover:text-rose-400 border border-transparent hover:border-[var(--border-color)] cursor-pointer shrink-0"
                title="Terminer la session affichée"
              >
                <X size={11} />
              </button>
            </>
          )}

          {chatTask && (
            <span className="flex items-center gap-1 pl-2 pr-1 py-1 rounded-md font-bold bg-cyan-500/15 text-cyan-300 border border-cyan-500/40">
              {/* La clé ouvre la fiche, la croix revient au terminal du projet.
                  Les deux étaient sur le même bouton : cliquer la clé fermait la
                  console de la tâche, ce qui est l'inverse de l'attendu. */}
              <button
                type="button"
                onClick={() => setSelectedTask(chatTask)}
                className="hover:underline cursor-pointer"
                title={`Ouvrir la fiche de ${chatTask.key}`}
              >
                {chatTask.key}
              </button>
              <button
                type="button"
                onClick={() => setChatTask(null)}
                className="p-0.5 rounded hover:bg-cyan-500/25 cursor-pointer"
                title="Revenir au terminal du projet"
              >
                <X size={10} />
              </button>
            </span>
          )}
        </div>

        {pendingInteractive && chatTask && pendingInteractive.taskId === chatTask.id && (
          <InteractiveStepBar
            label={pendingInteractive.label}
            taskKey={pendingInteractive.taskKey}
            onConfirm={confirmInteractiveStep}
            onDismiss={dismissInteractiveStep}
          />
        )}

        {/* En plein écran, les sessions passent à gauche en colonne : la largeur
            ne manque plus, et la liste se lit comme une barre latérale. Sinon
            elles restent en pied, sous le terminal. */}
        <div className={`flex-1 min-h-0 flex gap-2 ${isFullscreen ? 'flex-row' : 'flex-col'}`}>
          {sessionsList}

          <div className="flex-1 min-w-0 min-h-0">
          {chatTask ? (
            <InteractiveTerminal
              key={`task-${chatTask.id}`}
              task={chatTask}
              label={label}
              agentRunning={agentRunning}
              isExpanded={isFullscreen}
              onToggleExpand={() => setIsFullscreen(v => !v)}
              onClose={() => setChatTask(null)}
            />
          ) : terminalSessionOverride ? (
            <InteractiveTerminal
              key={terminalSessionOverride}
              sessionId={terminalSessionOverride}
              label={sessionLabel(terminalSessionOverride)}
              isExpanded={isFullscreen}
              onToggleExpand={() => setIsFullscreen(v => !v)}
              onClose={() => openTerminalSession('')}
            />
          ) : (
            <InteractiveTerminal
              key={sessionId}
              sessionId={sessionId}
              cwd={cwd}
              projectId={currentProject?.id}
              label={label}
              isExpanded={isFullscreen}
              onToggleExpand={() => setIsFullscreen(v => !v)}
              onClose={() => setIsTerminalPanelOpen(false)}
            />
          )}
          </div>

        </div>
      </div>
    </aside>
  )
}

/**
 * Fin d'un pas interactif. La skill du dépôt ne fait que produire du texte dans
 * le terminal : elle ne pose aucun label et ne transitionne pas le ticket. Cette
 * barre est le seul endroit où l'utilisateur dit que la session a abouti, et
 * c'est cette confirmation qui fait avancer le ticket sur le tracker.
 */
const InteractiveStepBar: React.FC<{
  label: string
  taskKey: string
  onConfirm: (note?: string) => Promise<void>
  onDismiss: () => void
}> = ({ label, taskKey, onConfirm, onDismiss }) => {
  const [note, setNote] = useState('')
  const [isBusy, setIsBusy] = useState(false)

  const confirm = async () => {
    if (isBusy) return
    setIsBusy(true)
    await onConfirm(note.trim() || undefined)
    setIsBusy(false)
  }

  return (
    <div className="px-3 py-2 border-b border-[var(--border-color)] bg-[var(--accent-light)]/40 space-y-1.5">
      <div className="flex items-center gap-2">
        <span className="text-[10px] font-bold uppercase tracking-wider text-[var(--accent-color)]">
          {label} · {taskKey}
        </span>
        <button
          type="button"
          onClick={onDismiss}
          className="ml-auto p-0.5 rounded text-[var(--text-muted)] hover:text-rose-400 cursor-pointer"
          title="Ne pas faire avancer le ticket"
        >
          <X size={11} />
        </button>
      </div>
      <p className="text-[10px] text-[var(--text-secondary)] leading-snug">
        Quand la session est terminée, confirme pour poser le label d'étape, changer le statut et
        transitionner le ticket sur le tracker.
      </p>
      <div className="flex items-center gap-1.5">
        <input
          value={note}
          onChange={e => setNote(e.target.value)}
          onKeyDown={e => {
            if (e.key === 'Enter') confirm()
          }}
          placeholder="Compte-rendu à publier en commentaire (facultatif)"
          className="flex-1 min-w-0 px-2 py-1 text-[11px] rounded-lg bg-[var(--bg-tertiary)] border border-[var(--border-color)] text-[var(--text-primary)] focus:outline-none focus:border-[var(--accent-color)]"
        />
        <button
          type="button"
          onClick={confirm}
          disabled={isBusy}
          className="flex items-center gap-1 px-2.5 py-1 rounded-lg text-[11px] font-bold text-white accent-bg hover:opacity-90 disabled:opacity-40 cursor-pointer shrink-0"
        >
          <Check size={11} />
          <span>{isBusy ? 'Envoi…' : 'Étape terminée'}</span>
        </button>
      </div>
    </div>
  )
}
