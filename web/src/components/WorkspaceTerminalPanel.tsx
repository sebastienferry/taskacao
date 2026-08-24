import React, { useCallback, useEffect, useRef, useState } from 'react'
import { InteractiveTerminal } from './InteractiveTerminal'
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
  const { isTerminalPanelOpen, setIsTerminalPanelOpen, currentProject, settings } = useApp()

  const [width, setWidth] = useState<number>(readStoredWidth)
  const [isResizing, setIsResizing] = useState(false)
  const widthRef = useRef(width)

  useEffect(() => {
    widthRef.current = width
  }, [width])

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

  const sessionId = currentProject ? `project-${currentProject.id}` : 'global-workspace'
  const cwd = currentProject?.repoPath || settings.repoPath || ''
  const label = currentProject ? `CLI · ${currentProject.name}` : 'CLI · Workspace'

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

      <div className="flex-1 min-w-0 p-2">
        <InteractiveTerminal
          sessionId={sessionId}
          cwd={cwd}
          projectId={currentProject?.id}
          label={label}
          onClose={() => setIsTerminalPanelOpen(false)}
        />
      </div>
    </aside>
  )
}
