import React, { useState, useRef, useEffect } from 'react'
import {
  Flame,
  AlertCircle,
  Calendar,
  ArrowDown,
  Sparkles,
  Loader2,
  GitBranch,
  GitPullRequest,
  ExternalLink,
  MessageSquare,
  Code2,
  MoreHorizontal,
  FileCode,
  CheckCircle2,
  Eye,
  Trash2,
  Copy,
} from 'lucide-react'
import type { Task, Priority } from '../types'
import { useApp } from '../context/AppContext'

interface TaskCardProps {
  task: Task
  isDragging?: boolean
  onDragStart?: (e: React.DragEvent) => void
}

export const TaskCard: React.FC<TaskCardProps> = ({ task, isDragging, onDragStart }) => {
  const {
    setSelectedTask,
    setChatTask,
    setDiffTask,
    runSkill,
    isSkillRunning,
    runningSkillId,
    activities,
    openInEditor,
    deleteTask,
    moveTaskWorkflowStage,
    projects,
    settings,
    t,
    addToast,
  } = useApp()

  const [isMenuOpen, setIsMenuOpen] = useState(false)
  const menuRef = useRef<HTMLDivElement>(null)

  const taskProject = projects.find(p => p.id === task.projectId)
  const targetGithubRepo = (taskProject?.githubRepo || settings.githubRepo || '').replace(/^https?:\/\/github\.com\//, '').replace(/\.git$/, '')
  const externalUrl = task.externalUrl || (
    task.source === 'github' && targetGithubRepo && task.key?.startsWith('#')
      ? `https://github.com/${targetGithubRepo}/issues/${task.key.replace('#', '')}`
      : undefined
  )

  useEffect(() => {
    if (!isMenuOpen) return
    const handleClickOutside = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setIsMenuOpen(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [isMenuOpen])

  const latestActivity = React.useMemo(() => {
    if (!activities || activities.length === 0) return null
    const taskActs = activities.filter(a => a.taskId === task.id)
    if (taskActs.length === 0) return null
    const running = taskActs.find(a => a.status === 'running' || a.status === 'queued' || a.status === 'pending')
    if (running) return running
    return taskActs[0]
  }, [activities, task.id])

  const getPriorityBadge = (priority: Priority) => {
    switch (priority) {
      case 'urgent':
        return (
          <span className="flex items-center gap-1 text-[10px] font-bold text-rose-400 bg-rose-500/10 border border-rose-500/20 px-1.5 py-0.5 rounded shrink-0" title={t.priority.urgent}>
            <Flame size={11} className="text-rose-500" />
            <span className="hidden sm:inline">{t.priority.urgent}</span>
          </span>
        )
      case 'high':
        return (
          <span className="flex items-center gap-1 text-[10px] font-bold text-amber-400 bg-amber-500/10 border border-amber-500/20 px-1.5 py-0.5 rounded shrink-0" title={t.priority.high}>
            <AlertCircle size={11} className="text-amber-500" />
            <span className="hidden sm:inline">{t.priority.high}</span>
          </span>
        )
      case 'medium':
        return (
          <span className="flex items-center gap-1 text-[10px] text-blue-400 bg-blue-500/10 border border-blue-500/20 px-1.5 py-0.5 rounded shrink-0" title={t.priority.medium}>
            <span className="w-1.5 h-1.5 rounded-full bg-blue-400"></span>
            <span className="hidden sm:inline">{t.priority.medium}</span>
          </span>
        )
      case 'low':
        return (
          <span className="flex items-center gap-1 text-[10px] text-slate-400 bg-slate-500/10 border border-slate-500/20 px-1.5 py-0.5 rounded shrink-0" title={t.priority.low}>
            <ArrowDown size={11} />
            <span className="hidden sm:inline">{t.priority.low}</span>
          </span>
        )
    }
  }

  const handleDragStartInternal = (e: React.DragEvent) => {
    e.dataTransfer.setData('text/plain', task.id)
    e.dataTransfer.effectAllowed = 'move'
    if (onDragStart) onDragStart(e)
  }

  // Determine current workflow stage action (Clarifier ➔ Spécifier ➔ Coder ➔ Créer PR ➔ Merge ➔ #finished)
  const getWorkflowAction = () => {
    const isFinished = task.status === 'finished' || task.status === 'done' || task.labels?.some(l => l.toLowerCase() === 'finished')
    if (isFinished) return null

    // If PR is already created or task is in reviewed stage -> Action is "Merge"
    if (
      task.status === 'to_close' ||
      task.labels?.some(l => l.toLowerCase() === 'reviewed') ||
      Boolean(task.prUrl && (task.status === 'to_test' || task.status === 'to_validate'))
    ) {
      return {
        id: 'merge',
        label: 'Merge',
        icon: <CheckCircle2 size={11} className="text-emerald-400" />,
        className: 'bg-emerald-500/10 text-emerald-300 hover:bg-emerald-500/20 border border-emerald-500/25',
        title: 'Fusionner la Pull Request / branche et finaliser la tâche (#finished)',
        action: async (e: React.MouseEvent) => {
          e.stopPropagation()
          await moveTaskWorkflowStage(task.id, 'finished')
        },
      }
    }

    // If code is implemented / to_test (and no PR created yet) -> Action is "Créer PR"
    if (
      task.status === 'to_test' ||
      task.status === 'to_validate' ||
      task.labels?.some(l => l.toLowerCase() === 'implemented')
    ) {
      return {
        id: 'create_pr',
        label: 'Créer PR',
        icon: <GitPullRequest size={11} className="text-purple-400" />,
        className: 'bg-purple-500/10 text-purple-300 hover:bg-purple-500/20 border border-purple-500/25',
        title: 'Lancer la revue de code et générer la Pull Request',
        action: async (e: React.MouseEvent) => {
          e.stopPropagation()
          if (isSkillRunning) return
          await runSkill(task.id, 'create_pr')
        },
      }
    }

    // If spec is ready / to_implement -> Action is "Coder"
    if (
      task.status === 'to_implement' ||
      task.status === 'in_progress' ||
      task.labels?.some(l => l.toLowerCase() === 'specified')
    ) {
      return {
        id: 'implement',
        label: 'Coder',
        icon: <Flame size={11} className="text-indigo-400" />,
        className: 'bg-indigo-500/10 text-indigo-300 hover:bg-indigo-500/20 border border-indigo-500/25',
        title: "Lancer l'implémentation du code par l'agent IA",
        action: async (e: React.MouseEvent) => {
          e.stopPropagation()
          if (isSkillRunning) return
          await runSkill(task.id, 'implement')
        },
      }
    }

    // If clarified / to_specify -> Action is "Spécifier"
    if (
      task.status === 'to_specify' ||
      task.labels?.some(l => l.toLowerCase() === 'clarified')
    ) {
      return {
        id: 'specify',
        label: 'Spécifier',
        icon: <FileCode size={11} className="text-blue-400" />,
        className: 'bg-blue-500/10 text-blue-300 hover:bg-blue-500/20 border border-blue-500/25',
        title: 'Rédiger la spécification technique Speckit',
        action: async (e: React.MouseEvent) => {
          e.stopPropagation()
          if (isSkillRunning) return
          await runSkill(task.id, 'specify')
        },
      }
    }

    // Default: Backlog / to_clarify -> Action is "Clarifier"
    return {
      id: 'clarify',
      label: 'Clarifier',
      icon: <Sparkles size={11} className="text-amber-400" />,
      className: 'bg-amber-500/10 text-amber-300 hover:bg-amber-500/20 border border-amber-500/25',
      title: 'Clarifier les exigences et cadrer la tâche',
      action: async (e: React.MouseEvent) => {
        e.stopPropagation()
        if (isSkillRunning) return
        await runSkill(task.id, 'clarify')
      },
    }
  }

  const workflowAction = getWorkflowAction()

  return (
    <div
      draggable
      onDragStart={handleDragStartInternal}
      onClick={() => setSelectedTask(task)}
      className={`group relative rounded-2xl border bg-[var(--bg-secondary)] border-[var(--border-color)] p-3 hover:border-[var(--accent-color)]/60 hover:shadow-md transition-all duration-150 cursor-grab active:cursor-grabbing select-none ${
        isDragging ? 'opacity-40 scale-95 ring-2 ring-[var(--accent-color)] ring-dashed' : ''
      }`}
    >
      {/* Ligne 1 : Clé + Titre + Priorité */}
      <div className="flex items-start justify-between gap-2 mb-1.5">
        <div className="flex items-baseline gap-1.5 min-w-0 flex-1">
          {/* Key / Tracker Link */}
          {externalUrl ? (
            <a
              href={externalUrl}
              target="_blank"
              rel="noreferrer"
              onClick={e => e.stopPropagation()}
              className="inline-flex items-center gap-0.5 text-[11px] font-mono font-bold text-[var(--accent-color)] hover:underline shrink-0"
              title={`Ouvrir ${task.key} sur le tracker externe`}
            >
              <span>{task.key}</span>
              <ExternalLink size={9} className="opacity-70" />
            </a>
          ) : (
            <span className="text-[11px] font-mono font-bold text-[var(--accent-color)] shrink-0">
              {task.key}
            </span>
          )}

          {/* Title */}
          <h4 className="text-xs font-semibold text-[var(--text-primary)] leading-snug line-clamp-2">
            {task.title}
          </h4>
        </div>

        {/* Priority Badge */}
        {getPriorityBadge(task.priority)}
      </div>

      {/* Ligne 2 : Description tronquée */}
      {task.description && (
        <p className="text-[11px] text-[var(--text-muted)] line-clamp-2 mb-2 leading-relaxed">
          {task.description}
        </p>
      )}

      {/* Ligne 3 : Métadonnées / Liens : Branche Git + Icône PR + Labels */}
      <div className="flex items-center justify-between gap-1.5 mb-2.5 flex-wrap">
        <div className="flex items-center gap-1.5 flex-wrap min-w-0" onClick={e => e.stopPropagation()}>
          {/* Lien Branche Git */}
          {task.branchName && (
            <button
              type="button"
              onClick={() => setDiffTask(task)}
              className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-[10px] font-mono font-medium text-cyan-300 bg-cyan-500/10 hover:bg-cyan-500/20 border border-cyan-500/25 transition-colors cursor-pointer"
              title={`Branche: ${task.branchName} (Cliquer pour voir le Diff)`}
            >
              <GitBranch size={11} className="text-cyan-400 shrink-0" />
              <span className="truncate max-w-[130px]">{task.branchName}</span>
            </button>
          )}

          {/* Icône PR uniquement */}
          {task.prUrl && (
            <a
              href={task.prUrl}
              target="_blank"
              rel="noreferrer"
              className="p-1 rounded text-purple-300 bg-purple-500/10 hover:bg-purple-500/25 border border-purple-500/30 transition-all hover:scale-105"
              title={task.prUrl.includes('gitlab') ? `GitLab MR: ${task.prUrl}` : `GitHub PR: ${task.prUrl}`}
            >
              <GitPullRequest size={12} className="text-purple-400" />
            </a>
          )}

          {/* Labels compacts (max 2 visibles pour ne pas surcharger) */}
          {task.labels && task.labels.slice(0, 2).map(lbl => (
            <span
              key={lbl}
              className="text-[9.5px] px-1.5 py-0.2 rounded bg-[var(--bg-tertiary)] text-[var(--text-muted)] border border-[var(--border-color)]/70 font-mono"
            >
              #{lbl}
            </span>
          ))}
          {task.labels && task.labels.length > 2 && (
            <span className="text-[9px] text-[var(--text-muted)] opacity-70">
              +{task.labels.length - 2}
            </span>
          )}
        </div>

        {/* Assignee / Date */}
        <div className="flex items-center gap-1.5 shrink-0 ml-auto text-[10px] text-[var(--text-muted)]">
          {task.assignee && (
            <div
              className="w-4.5 h-4.5 rounded-full accent-bg text-white flex items-center justify-center text-[8.5px] font-bold shadow-2xs"
              title={`Assigné à : ${task.assignee}`}
            >
              {task.assignee.substring(0, 2).toUpperCase()}
            </div>
          )}
          {task.dueDate && (
            <span className="flex items-center gap-0.5 text-amber-400 font-medium" title={`Échéance : ${task.dueDate}`}>
              <Calendar size={10} />
              <span>{task.dueDate.split('T')[0]?.slice(5)}</span>
            </span>
          )}
        </div>
      </div>

      {/* Ligne 4 : Barre d'action (Skill IA / Discuter) + Bouton Code + Menu (...) */}
      <div className="pt-2 border-t border-[var(--border-color)]/50 flex items-center justify-between gap-1.5" onClick={e => e.stopPropagation()}>
        {/* Left: Action Button (AI Skill or Discuter) */}
        <div className="flex items-center gap-1">
          {workflowAction ? (
            <button
              type="button"
              onClick={workflowAction.action}
              disabled={isSkillRunning}
              className={`flex items-center gap-1 px-2 py-0.5 rounded-md text-[10.5px] font-medium active:scale-95 transition-all cursor-pointer disabled:opacity-50 ${workflowAction.className}`}
              title={workflowAction.title}
            >
              {isSkillRunning && runningSkillId === workflowAction.id ? (
                <Loader2 size={10} className="animate-spin" />
              ) : (
                workflowAction.icon
              )}
              <span>{workflowAction.label}</span>
            </button>
          ) : (
            <button
              type="button"
              onClick={() => setChatTask(task)}
              className="flex items-center gap-1 px-2 py-0.5 rounded-md text-[10.5px] font-medium text-cyan-300 bg-cyan-500/10 hover:bg-cyan-500/20 border border-cyan-500/25 transition-colors cursor-pointer"
              title="Discuter avec l'agent Copilot"
            >
              <MessageSquare size={10} className="text-cyan-400" />
              <span>Discuter</span>
            </button>
          )}

          {/* Live Activity running pill if any */}
          {latestActivity && latestActivity.status === 'running' && (
            <span
              onClick={() => setChatTask(task)}
              className="flex items-center gap-1 px-1.5 py-0.5 rounded-md text-[9.5px] font-mono font-bold bg-indigo-500/15 text-indigo-300 border border-indigo-500/30 animate-pulse cursor-pointer"
              title="Activité IA en cours d'exécution"
            >
              <Loader2 size={9} className="animate-spin text-indigo-400" />
              <span>Live</span>
            </span>
          )}
        </div>

        {/* Right: Code button + Contextual Menu (...) */}
        <div className="flex items-center gap-1 relative" ref={menuRef}>
          {/* Quick Code Button */}
          <button
            type="button"
            onClick={() => openInEditor({ taskId: task.id })}
            className="flex items-center gap-1 px-2 py-0.5 rounded-md text-[10.5px] font-medium text-[var(--text-secondary)] hover:text-cyan-300 bg-[var(--bg-tertiary)]/70 hover:bg-cyan-500/10 border border-[var(--border-color)]/70 hover:border-cyan-500/30 transition-colors cursor-pointer"
            title={`Ouvrir dans ${settings.editorCommand || 'VS Code'}`}
          >
            <Code2 size={11} className="text-cyan-400" />
            <span>Code</span>
          </button>

          {/* Menu (...) Button */}
          <button
            type="button"
            onClick={() => setIsMenuOpen(prev => !prev)}
            className="p-1 rounded-md text-[var(--text-muted)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-tertiary)] border border-transparent hover:border-[var(--border-color)]/60 transition-colors cursor-pointer"
            title="Options supplémentaires"
          >
            <MoreHorizontal size={14} />
          </button>

          {/* Contextual Dropdown Menu */}
          {isMenuOpen && (
            <div className="absolute right-0 bottom-full mb-1 w-48 rounded-xl bg-[var(--bg-secondary)] border border-[var(--border-color)] shadow-xl p-1 z-50 animate-in fade-in-0 zoom-in-95 duration-100 text-xs">
              <button
                type="button"
                onClick={() => {
                  setIsMenuOpen(false)
                  setSelectedTask(task)
                }}
                className="w-full flex items-center gap-2 px-2.5 py-1.5 rounded-lg text-[11px] text-[var(--text-primary)] hover:bg-[var(--bg-tertiary)] transition-colors cursor-pointer"
              >
                <Eye size={12} className="text-blue-400" />
                <span>Voir les détails</span>
              </button>

              <button
                type="button"
                onClick={() => {
                  setIsMenuOpen(false)
                  setChatTask(task)
                }}
                className="w-full flex items-center gap-2 px-2.5 py-1.5 rounded-lg text-[11px] text-[var(--text-primary)] hover:bg-[var(--bg-tertiary)] transition-colors cursor-pointer"
              >
                <MessageSquare size={12} className="text-cyan-400" />
                <span>Discuter avec l'agent</span>
              </button>

              {/* Créer PR option in menu */}
              {!task.prUrl && task.status !== 'finished' && (
                <button
                  type="button"
                  onClick={async () => {
                    setIsMenuOpen(false)
                    await runSkill(task.id, 'create_pr')
                  }}
                  className="w-full flex items-center gap-2 px-2.5 py-1.5 rounded-lg text-[11px] text-purple-400 hover:bg-purple-500/10 transition-colors cursor-pointer"
                >
                  <GitPullRequest size={12} />
                  <span>Créer la Pull Request</span>
                </button>
              )}

              {/* Merge / Finaliser option in menu */}
              {task.status !== 'finished' && (
                <button
                  type="button"
                  onClick={async () => {
                    setIsMenuOpen(false)
                    await moveTaskWorkflowStage(task.id, 'finished')
                  }}
                  className="w-full flex items-center gap-2 px-2.5 py-1.5 rounded-lg text-[11px] text-emerald-400 hover:bg-emerald-500/10 transition-colors cursor-pointer"
                >
                  <CheckCircle2 size={12} />
                  <span>Fusionner & Finir (#finished)</span>
                </button>
              )}

              {task.branchName && (
                <button
                  type="button"
                  onClick={() => {
                    setIsMenuOpen(false)
                    setDiffTask(task)
                  }}
                  className="w-full flex items-center gap-2 px-2.5 py-1.5 rounded-lg text-[11px] text-[var(--text-primary)] hover:bg-[var(--bg-tertiary)] transition-colors cursor-pointer"
                >
                  <GitBranch size={12} className="text-indigo-400" />
                  <span>Inspecter le Diff Git</span>
                </button>
              )}

              {externalUrl && (
                <a
                  href={externalUrl}
                  target="_blank"
                  rel="noreferrer"
                  onClick={() => setIsMenuOpen(false)}
                  className="w-full flex items-center gap-2 px-2.5 py-1.5 rounded-lg text-[11px] text-[var(--text-primary)] hover:bg-[var(--bg-tertiary)] transition-colors cursor-pointer"
                >
                  <ExternalLink size={12} className="text-amber-400" />
                  <span>Ouvrir sur le tracker</span>
                </a>
              )}

              <button
                type="button"
                onClick={() => {
                  setIsMenuOpen(false)
                  navigator.clipboard.writeText(`${task.key}: ${task.title}`)
                  addToast({ type: 'info', title: 'Copié', description: `${task.key} copié dans le presse-papier` })
                }}
                className="w-full flex items-center gap-2 px-2.5 py-1.5 rounded-lg text-[11px] text-[var(--text-primary)] hover:bg-[var(--bg-tertiary)] transition-colors cursor-pointer"
              >
                <Copy size={12} className="text-slate-400" />
                <span>Copier la référence</span>
              </button>

              <div className="h-px bg-[var(--border-color)] my-1" />

              <button
                type="button"
                onClick={() => {
                  setIsMenuOpen(false)
                  if (window.confirm(`Supprimer la tâche ${task.key} ?`)) {
                    deleteTask(task.id)
                  }
                }}
                className="w-full flex items-center gap-2 px-2.5 py-1.5 rounded-lg text-[11px] text-rose-400 hover:bg-rose-500/10 transition-colors cursor-pointer"
              >
                <Trash2 size={12} />
                <span>Supprimer la tâche</span>
              </button>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
