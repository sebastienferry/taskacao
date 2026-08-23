import React from 'react'
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
  FolderGit2,
  Folder,
  MessageSquare,
  CheckCircle2,
  Clock,
} from 'lucide-react'
import type { Task, Priority } from '../types'
import { useApp } from '../context/AppContext'

interface TaskCardProps {
  task: Task
  isDragging?: boolean
  onDragStart?: (e: React.DragEvent) => void
}

export const TaskCard: React.FC<TaskCardProps> = ({ task, isDragging, onDragStart }) => {
  const { setSelectedTask, setChatTask, setDiffTask, runSkill, isSkillRunning, runningSkillId, activities, t } = useApp()

  const formatRelativeTime = (dateStr?: string) => {
    if (!dateStr) return ''
    try {
      const diff = Math.max(0, (Date.now() - new Date(dateStr).getTime()) / 1000)
      if (diff < 60) return "à l'instant"
      if (diff < 3600) return `${Math.floor(diff / 60)}m`
      if (diff < 86400) return `${Math.floor(diff / 3600)}h`
      return `${Math.floor(diff / 86400)}j`
    } catch {
      return ''
    }
  }

  const latestActivity = React.useMemo(() => {
    if (!activities || activities.length === 0) return null
    const taskActs = activities.filter(a => a.taskId === task.id)
    if (taskActs.length === 0) return null
    // Prioritize active (running/queued/pending)
    const running = taskActs.find(a => a.status === 'running' || a.status === 'queued' || a.status === 'pending')
    if (running) return running
    // Otherwise the most recently finished or started
    return taskActs[0]
  }, [activities, task.id])

  const getPriorityBadge = (priority: Priority) => {
    switch (priority) {
      case 'urgent':
        return (
          <span className="flex items-center gap-1 text-[10px] font-semibold text-rose-400 bg-rose-500/10 px-1.5 py-0.5 rounded" title={t.priority.urgent}>
            <Flame size={11} className="text-rose-500" />
            <span>{t.priority.urgent}</span>
          </span>
        )
      case 'high':
        return (
          <span className="flex items-center gap-1 text-[10px] font-semibold text-amber-400 bg-amber-500/10 px-1.5 py-0.5 rounded" title={t.priority.high}>
            <AlertCircle size={11} className="text-amber-500" />
            <span>{t.priority.high}</span>
          </span>
        )
      case 'medium':
        return (
          <span className="flex items-center gap-1 text-[10px] text-blue-400 bg-blue-500/10 px-1.5 py-0.5 rounded" title={t.priority.medium}>
            <span className="w-1.5 h-1.5 rounded-full bg-blue-400"></span>
            <span>{t.priority.medium}</span>
          </span>
        )
      case 'low':
        return (
          <span className="flex items-center gap-1 text-[10px] text-slate-400 bg-slate-500/10 px-1.5 py-0.5 rounded" title={t.priority.low}>
            <ArrowDown size={11} />
            <span>{t.priority.low}</span>
          </span>
        )
    }
  }

  const handleDragStartInternal = (e: React.DragEvent) => {
    e.dataTransfer.setData('text/plain', task.id)
    e.dataTransfer.effectAllowed = 'move'
    if (onDragStart) onDragStart(e)
  }

  const formatDueDate = (due: string) => {
    try {
      const d = new Date(due)
      return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' })
    } catch {
      return due
    }
  }

  const getNextSkillForTask = () => {
    switch (task.status) {
      case 'to_clarify':
      case 'backlog':
        return { id: 'clarify', label: 'Clarify' }
      case 'to_specify':
      case 'specified':
        return { id: 'specify', label: 'Specify' }
      case 'to_implement':
      case 'in_progress':
        return { id: 'implement', label: 'Code' }
      case 'to_test':
      case 'to_validate':
        return { id: 'create_pr', label: 'PR / Merge' }
      default:
        return null
    }
  }

  const nextSkill = getNextSkillForTask()

  const handleQuickSkillClick = async (e: React.MouseEvent) => {
    e.stopPropagation()
    if (!nextSkill || isSkillRunning) return
    await runSkill(task.id, nextSkill.id)
  }

  const isNewOrBacklog =
    task.status === 'to_clarify' ||
    task.status === 'backlog' ||
    (task.labels && (task.labels.includes('new') || task.labels.includes('untouched')))

  return (
    <div
      draggable
      onDragStart={handleDragStartInternal}
      onClick={() => setSelectedTask(task)}
      className={`group relative rounded-2xl border bg-[var(--bg-secondary)] border-[var(--border-color)] p-3.5 hover:border-[var(--accent-color)]/60 hover:shadow-lg transition-all duration-150 cursor-grab active:cursor-grabbing select-none ${
        isDragging ? 'opacity-40 scale-95 ring-2 ring-[var(--accent-color)] ring-dashed' : ''
      }`}
    >
      {/* Top row: Key & Clickable Tracker Link (Left), Priority Badge (Right) */}
      <div className="flex items-center justify-between gap-2 mb-2">
        <div className="flex items-center gap-1.5 min-w-0">
          {/* Direct Clickable Tracker Badge / Link */}
          {task.externalUrl ? (
            <a
              href={task.externalUrl}
              target="_blank"
              rel="noreferrer"
              onClick={e => e.stopPropagation()}
              className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-md text-[11px] font-mono font-bold transition-all shadow-2xs hover:scale-105 shrink-0 ${
                task.source === 'linear'
                  ? 'bg-indigo-500/15 text-indigo-400 hover:bg-indigo-500/25 border border-indigo-500/30'
                  : task.source === 'github'
                  ? 'bg-purple-500/15 text-purple-400 hover:bg-purple-500/25 border border-purple-500/30'
                  : 'bg-[var(--accent-light)] text-[var(--accent-color)] hover:opacity-80'
              }`}
              title={task.source === 'linear' ? `Ouvrir ${task.key} sur Linear` : `Ouvrir ${task.key} sur GitHub`}
            >
              {task.source === 'linear' && <span className="text-indigo-400">◆</span>}
              {task.source === 'github' && <FolderGit2 size={12} className="text-purple-400" />}
              <span className="truncate max-w-[120px]">{task.key}</span>
              <ExternalLink size={9} className="opacity-70 group-hover:opacity-100" />
            </a>
          ) : (
            <span className="font-mono text-[11px] font-bold text-[var(--text-muted)] group-hover:text-[var(--accent-color)] transition-colors inline-flex items-center gap-1 px-1.5 py-0.5 rounded bg-[var(--bg-primary)] border border-[var(--border-color)] shrink-0">
              <Folder size={11} className="text-emerald-400" />
              <span>{task.key}</span>
            </span>
          )}

          {task.branchName && (
            <span title={`Branche: ${task.branchName}`} className="text-indigo-400 hidden sm:inline shrink-0">
              <GitBranch size={11} />
            </span>
          )}
        </div>

        {/* Priority Badge */}
        <div className="flex items-center gap-1 shrink-0">
          {getPriorityBadge(task.priority)}
        </div>
      </div>

      {/* Task Title */}
      <h4 className="text-xs font-semibold text-[var(--text-primary)] leading-snug line-clamp-2 mb-1.5">
        {task.title}
      </h4>

      {/* Description Snippet */}
      {task.description && (
        <p className="text-[11px] text-[var(--text-muted)] line-clamp-2 mb-2.5 leading-relaxed">
          {task.description}
        </p>
      )}

      {/* Labels */}
      {task.labels && task.labels.length > 0 && (
        <div className="flex flex-wrap gap-1 mb-2.5">
          {task.labels.map(lbl => {
            const lower = lbl.toLowerCase()
            let badgeStyle = 'bg-[var(--bg-tertiary)] text-[var(--text-secondary)] border-[var(--border-color)]'
            if (lower === 'new' || lower === 'untouched') badgeStyle = 'bg-cyan-500/15 text-cyan-400 border-cyan-500/30 font-semibold'
            else if (lower === 'clarified') badgeStyle = 'bg-amber-500/15 text-amber-400 border-amber-500/30 font-semibold'
            else if (lower === 'specified') badgeStyle = 'bg-blue-500/15 text-blue-400 border-blue-500/30 font-semibold'
            else if (lower === 'implemented') badgeStyle = 'bg-indigo-500/15 text-indigo-400 border-indigo-500/30 font-semibold'
            else if (lower === 'reviewed') badgeStyle = 'bg-purple-500/15 text-purple-400 border-purple-500/30 font-semibold'
            else if (lower === 'finished' || lower === 'closed') badgeStyle = 'bg-emerald-500/15 text-emerald-400 border-emerald-500/30 font-semibold'

            return (
              <span
                key={lbl}
                className={`inline-flex items-center gap-0.5 text-[10px] px-1.5 py-0.2 rounded-md border ${badgeStyle}`}
              >
                #{lbl}
              </span>
            )
          })}
        </div>
      )}

      {/* Git Branch & PR/MR Badges */}
      {(task.branchName || task.prUrl) && (
        <div className="flex flex-wrap items-center gap-1.5 mb-2.5" onClick={e => e.stopPropagation()}>
          {task.branchName && (
            <div
              onClick={() => setDiffTask(task)}
              className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-[10px] font-mono text-indigo-300 bg-indigo-500/10 hover:bg-indigo-500/20 border border-indigo-500/25 transition-colors cursor-pointer group/branch"
              title={`Branche Git: ${task.branchName} (Cliquer pour voir le Diff Git)`}
            >
              <GitBranch size={10} className="text-indigo-400 shrink-0 group-hover/branch:scale-110 transition-transform" />
              <span className="truncate max-w-[130px]">{task.branchName}</span>
            </div>
          )}

          {task.prUrl && (
            <a
              href={task.prUrl}
              target="_blank"
              rel="noreferrer"
              className={`inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-[10px] font-mono font-bold transition-all shadow-2xs hover:scale-105 ${
                task.prUrl.includes('gitlab')
                  ? 'bg-orange-500/15 text-orange-400 hover:bg-orange-500/25 border border-orange-500/30'
                  : 'bg-purple-500/15 text-purple-400 hover:bg-purple-500/25 border border-purple-500/30'
              }`}
              title={task.prUrl.includes('gitlab') ? `Merge Request GitLab: ${task.prUrl}` : `Pull Request GitHub: ${task.prUrl}`}
            >
              <GitPullRequest size={10} className={task.prUrl.includes('gitlab') ? 'text-orange-400' : 'text-purple-400'} />
              <span>{task.prUrl.includes('gitlab') ? 'GitLab MR' : 'GitHub PR'}</span>
              <ExternalLink size={8} className="opacity-70" />
            </a>
          )}
        </div>
      )}

      {/* Latest Activity on this task (Started / Running / Completed / Failed) */}
      {latestActivity && (
        <div
          onClick={(e) => {
            e.stopPropagation()
            setChatTask(task)
          }}
          className={`flex items-center justify-between gap-1.5 px-2.5 py-1.5 rounded-xl text-[10.5px] mb-2 transition-all cursor-pointer border shadow-2xs group/activity ${
            latestActivity.status === 'running'
              ? 'bg-indigo-500/10 border-indigo-500/30 text-indigo-300 ring-1 ring-indigo-500/20 animate-pulse'
              : latestActivity.status === 'failed'
              ? 'bg-rose-500/10 border-rose-500/25 text-rose-300 hover:border-rose-500/40'
              : latestActivity.status === 'queued' || latestActivity.status === 'pending'
              ? 'bg-amber-500/10 border-amber-500/25 text-amber-300 hover:border-amber-500/40'
              : 'bg-emerald-500/10 border-emerald-500/20 text-emerald-300 hover:border-emerald-500/40'
          }`}
          title={`Dernière activité: ${latestActivity.skillName || latestActivity.action} (${latestActivity.status}) - Cliquer pour ouvrir la discussion`}
        >
          <div className="flex items-center gap-1.5 min-w-0">
            {latestActivity.status === 'running' ? (
              <Loader2 size={11} className="animate-spin text-indigo-400 shrink-0" />
            ) : latestActivity.status === 'failed' ? (
              <AlertCircle size={11} className="text-rose-400 shrink-0" />
            ) : latestActivity.status === 'queued' || latestActivity.status === 'pending' ? (
              <Clock size={11} className="text-amber-400 shrink-0" />
            ) : (
              <CheckCircle2 size={11} className="text-emerald-400 shrink-0" />
            )}
            <span className="font-bold truncate max-w-[130px]">
              {latestActivity.skillName || latestActivity.action || 'Activité Agent'}
            </span>
            {latestActivity.status === 'running' && (
              <span className="text-[9.5px] text-indigo-400/80 truncate hidden sm:inline">• en cours</span>
            )}
            {latestActivity.status === 'completed' && latestActivity.summary && (
              <span className="text-[9.5px] text-emerald-400/80 truncate hidden sm:inline max-w-[130px]" title={latestActivity.summary}>
                • {latestActivity.summary}
              </span>
            )}
          </div>
          <span className="text-[9.5px] font-mono opacity-80 shrink-0">
            {latestActivity.status === 'running'
              ? 'Live'
              : formatRelativeTime(latestActivity.completedAt || latestActivity.startedAt || latestActivity.createdAt)}
          </span>
        </div>
      )}

      {/* Quick AI Skill Actions Bar (Visible only on card hover or when running) */}
      <div
        className={`pt-2 pb-0.5 flex items-center justify-between gap-1.5 transition-all duration-150 min-h-[30px] ${
          isSkillRunning
            ? 'opacity-100'
            : 'opacity-0 pointer-events-none group-hover:opacity-100 group-hover:pointer-events-auto'
        }`}
        onClick={e => e.stopPropagation()}
      >
        <div className="flex items-center gap-1.5 flex-wrap">
          {/* Discuss with Agent button */}
          <button
            type="button"
            onClick={(e) => {
              e.stopPropagation()
              setChatTask(task)
            }}
            className="flex items-center gap-1 px-2 py-1 rounded-lg text-[10px] font-bold text-white bg-[var(--accent-color)] hover:opacity-90 transition-all cursor-pointer shadow-2xs active:scale-95"
            title="💬 Discuter de cette tâche avec l'agent Copilot"
          >
            <MessageSquare size={10} />
            <span>Discuter</span>
          </button>

          {isNewOrBacklog ? (
            <>
              <button
                type="button"
                onClick={handleQuickSkillClick}
                disabled={isSkillRunning}
                className="flex items-center gap-1 px-2 py-1 rounded-lg text-[10px] font-bold bg-[var(--accent-light)] accent-text border border-[var(--accent-color)]/30 hover:opacity-90 transition-all cursor-pointer shadow-2xs active:scale-95 disabled:opacity-50"
                title="Lancer le cadrage initial (Clarify)"
              >
                {isSkillRunning && runningSkillId === 'clarify' ? (
                  <Loader2 size={10} className="animate-spin" />
                ) : (
                  <Sparkles size={10} className="text-amber-400" />
                )}
                <span>Clarifier</span>
              </button>

              <button
                type="button"
                onClick={async (e) => {
                  e.stopPropagation()
                  if (isSkillRunning) return
                  await runSkill(task.id, 'implement')
                }}
                disabled={isSkillRunning}
                className="flex items-center gap-1 px-2 py-1 rounded-lg text-[10px] font-bold text-white bg-linear-to-r from-blue-600 to-indigo-600 hover:from-blue-500 hover:to-indigo-500 transition-all cursor-pointer shadow-2xs active:scale-95 disabled:opacity-50"
                title="Passer directement au code sans étape préalable"
              >
                <Flame size={10} className="text-amber-300" />
                <span>Coder</span>
              </button>
            </>
          ) : nextSkill ? (
            <button
              type="button"
              onClick={handleQuickSkillClick}
              disabled={isSkillRunning}
              className="flex items-center gap-1 px-2.5 py-1 rounded-lg text-[10px] font-bold text-white accent-bg shadow-xs hover:opacity-90 transition-all cursor-pointer active:scale-95 disabled:opacity-50"
              title={`Lancer skill: ${nextSkill.label}`}
            >
              {isSkillRunning && runningSkillId === nextSkill.id ? (
                <Loader2 size={10} className="animate-spin" />
              ) : (
                <Sparkles size={10} className="text-amber-300" />
              )}
              <span>{nextSkill.label}</span>
            </button>
          ) : null}
        </div>
      </div>

      {/* Footer: Due date & Assignee */}
      <div className="flex items-center justify-between text-[11px] text-[var(--text-muted)] pt-2 border-t border-[var(--border-color)]/60">
        <div className="flex items-center gap-1.5">
          {task.dueDate ? (
            <span className="flex items-center gap-1 text-[10px] text-amber-400/90 font-medium">
              <Calendar size={11} />
              {formatDueDate(task.dueDate)}
            </span>
          ) : (
            <span className="text-[10px] text-[var(--text-muted)] opacity-60 font-mono">
              {new Date(task.createdAt).toLocaleDateString(undefined, { month: 'short', day: 'numeric' })}
            </span>
          )}
        </div>

        {task.assignee && (
          <div className="flex items-center gap-1 text-[11px] font-medium text-[var(--text-secondary)]">
            <div className="w-5 h-5 rounded-full accent-bg text-white flex items-center justify-center text-[9px] font-bold shadow-xs">
              {task.assignee.substring(0, 2).toUpperCase()}
            </div>
            <span className="max-w-[80px] truncate text-[10px] hidden sm:inline">
              {task.assignee.split(' ')[0]}
            </span>
          </div>
        )}
      </div>
    </div>
  )
}
