import React, { useState } from 'react'
import {
  X,
  Terminal,
  Maximize2,
  Minimize2,
  GitBranch,
  FolderGit2,
  Folder,
  Code2,
} from 'lucide-react'
import { useApp } from '../context/AppContext'
import { InteractiveTerminal } from './InteractiveTerminal'

/**
 * "Discuter avec l'agent" — a drawer hosting the task's real TTY session.
 *
 * There is no separate non-TTY chat view: the agent CLI runs in the PTY that
 * lives server-side under the `task-<id>` session, so the drawer only frames
 * that terminal and closing it never kills the shell.
 */
export const TaskChatDrawer: React.FC = () => {
  const { chatTask, setChatTask, projects, setDiffTask } = useApp()

  const [isFullscreen, setIsFullscreen] = useState(false)

  if (!chatTask) return null

  const handleClose = () => {
    setChatTask(null)
  }

  const project = projects.find(p => p.id === chatTask.projectId)
  // A ticket may pin its own repository; the header must name the one the PTY
  // actually runs in, not the project's.
  const effectiveRepo = (chatTask.repoPath || '').trim() || project?.repoPath || ''
  const cwdLabel = effectiveRepo ? (effectiveRepo.split('/').pop() || effectiveRepo) : (project?.name || 'Workspace')

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
                  <span>• CWD: {cwdLabel}</span>
                </div>
              </div>
            </div>

            {/* Header Right Actions */}
            <div className="flex items-center gap-2 shrink-0">
              {/* Session badge */}
              <span className="hidden sm:flex items-center gap-1.5 px-2.5 py-1 rounded-lg bg-slate-900 border border-slate-800 text-[11px] font-semibold text-emerald-400">
                <Terminal size={12} />
                <span className="font-mono">task-{chatTask.id.slice(0, 8)}</span>
              </span>

              {/* View Git Diff */}
              <button
                type="button"
                onClick={() => setDiffTask(chatTask)}
                className="p-1.5 rounded-lg text-indigo-400 hover:bg-indigo-500/10 transition-colors cursor-pointer border border-indigo-500/20"
                title="Inspecter le diff Git"
              >
                <Code2 size={15} />
              </button>

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

          {/* Interactive ZSH terminal (PTY), sole view of the drawer */}
          <div className="flex-1 min-h-0 p-3 sm:p-4 flex flex-col">
            <InteractiveTerminal task={chatTask} isExpanded={isFullscreen} />
          </div>
        </div>
      </div>
    </div>
  )
}
