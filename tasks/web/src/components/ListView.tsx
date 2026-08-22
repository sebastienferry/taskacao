import React, { useState } from 'react'
import {
  Flame,
  AlertCircle,
  Clock,
  CheckCircle2,
  Calendar,
  Trash2,
  FileCode,
  HelpCircle,
  ShieldCheck,
  ArrowUpDown,
  Sparkles,
  Loader2,
  GitBranch,
  ExternalLink,
  FolderGit2,
  Eye,
  EyeOff
} from 'lucide-react'
import { useApp } from '../context/AppContext'
import type { Task, Status, Priority } from '../types'

export const ListView: React.FC = () => {
  const {
    tasks,
    setSelectedTask,
    updateTask,
    deleteTask,
    runSkill,
    isSkillRunning,
    runningSkillId,
    hideDone,
    toggleHideDone,
    t,
  } = useApp()

  const [sortField, setSortField] = useState<'key' | 'title' | 'status' | 'priority' | 'dueDate' | 'createdAt'>('createdAt')
  const [sortAsc, setSortAsc] = useState(false)
  const [groupByStatus, setGroupByStatus] = useState(true)

  const handleSort = (field: typeof sortField) => {
    if (sortField === field) {
      setSortAsc(prev => !prev)
    } else {
      setSortField(field)
      setSortAsc(true)
    }
  }

  const sortedTasks = [...tasks].sort((a, b) => {
    let result = 0
    if (sortField === 'key') {
      result = a.key.localeCompare(b.key, undefined, { numeric: true })
    } else if (sortField === 'title') {
      result = a.title.localeCompare(b.title)
    } else if (sortField === 'status') {
      result = a.status.localeCompare(b.status)
    } else if (sortField === 'priority') {
      const priorityOrder: Record<Priority, number> = { urgent: 4, high: 3, medium: 2, low: 1 }
      result = (priorityOrder[a.priority] || 0) - (priorityOrder[b.priority] || 0)
    } else if (sortField === 'dueDate') {
      result = (a.dueDate || '').localeCompare(b.dueDate || '')
    } else {
      result = new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime()
    }
    return sortAsc ? result : -result
  })

  const doneTasksCount = tasks.filter(t => t.status === 'to_close' || t.status === 'done').length

  const visibleTasks = hideDone
    ? sortedTasks.filter(t => t.status !== 'to_close' && t.status !== 'done')
    : sortedTasks

  const statusList: { id: Status; label: string; stageLabel: string; stageColor: string; icon: React.ReactNode; color: string }[] = [
    { id: 'to_clarify', label: t.status.to_clarify, stageLabel: 'New', stageColor: 'bg-cyan-500/15 text-cyan-400 border-cyan-500/30', icon: <HelpCircle size={14} />, color: 'text-cyan-400' },
    { id: 'to_specify', label: t.status.to_specify, stageLabel: 'Clarified', stageColor: 'bg-amber-500/15 text-amber-400 border-amber-500/30', icon: <FileCode size={14} />, color: 'text-amber-400' },
    { id: 'to_implement', label: t.status.to_implement, stageLabel: 'Specified', stageColor: 'bg-blue-500/15 text-blue-400 border-blue-500/30', icon: <Flame size={14} />, color: 'text-blue-400' },
    { id: 'to_test', label: t.status.to_test, stageLabel: 'Implemented', stageColor: 'bg-indigo-500/15 text-indigo-400 border-indigo-500/30', icon: <ShieldCheck size={14} />, color: 'text-indigo-400' },
    { id: 'to_close', label: t.status.to_close, stageLabel: 'Reviewed', stageColor: 'bg-emerald-500/15 text-emerald-400 border-emerald-500/30', icon: <CheckCircle2 size={14} />, color: 'text-emerald-400' },
  ]

  const getPriorityBadge = (priority: Priority) => {
    switch (priority) {
      case 'urgent':
        return (
          <span className="inline-flex items-center gap-1 text-[11px] font-semibold text-rose-400 bg-rose-500/10 px-2 py-0.5 rounded">
            <Flame size={12} className="text-rose-500" />
            {t.priority.urgent}
          </span>
        )
      case 'high':
        return (
          <span className="inline-flex items-center gap-1 text-[11px] font-semibold text-amber-400 bg-amber-500/10 px-2 py-0.5 rounded">
            <AlertCircle size={12} className="text-amber-500" />
            {t.priority.high}
          </span>
        )
      case 'medium':
        return (
          <span className="inline-flex items-center gap-1 text-[11px] text-blue-400 bg-blue-500/10 px-2 py-0.5 rounded">
            <span className="w-1.5 h-1.5 rounded-full bg-blue-400"></span>
            {t.priority.medium}
          </span>
        )
      case 'low':
        return (
          <span className="inline-flex items-center gap-1 text-[11px] text-slate-400 bg-slate-500/10 px-2 py-0.5 rounded">
            <span className="w-1.5 h-1.5 rounded-full bg-slate-400"></span>
            {t.priority.low}
          </span>
        )
    }
  }

  const getNextSkillForTask = (taskStatus: Status) => {
    switch (taskStatus) {
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
        return { id: 'create_pr', label: 'PR' }
      default:
        return null
    }
  }

  const renderTaskRow = (task: Task) => {
    const nextSkill = getNextSkillForTask(task.status)

    return (
      <tr
        key={task.id}
        onClick={() => setSelectedTask(task)}
        className="border-b border-[var(--border-color)]/60 hover:bg-[var(--bg-tertiary)]/50 transition-colors cursor-pointer group"
      >
        {/* Source & Key */}
        <td className="py-2.5 px-3 whitespace-nowrap" onClick={e => e.stopPropagation()}>
          <div className="flex items-center gap-1.5 font-mono text-xs font-bold">
            {task.externalUrl ? (
              <a
                href={task.externalUrl}
                target="_blank"
                rel="noreferrer"
                className={`inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-[11px] font-mono transition-all shadow-2xs hover:scale-105 ${
                  task.source === 'linear'
                    ? 'bg-indigo-500/15 text-indigo-400 hover:bg-indigo-500/25 border border-indigo-500/30'
                    : task.source === 'github'
                    ? 'bg-purple-500/15 text-purple-400 hover:bg-purple-500/25 border border-purple-500/30'
                    : 'bg-[var(--accent-light)] text-[var(--accent-color)] hover:opacity-80'
                }`}
                title={task.source === 'linear' ? `Ouvrir ${task.key} sur Linear` : `Ouvrir ${task.key} sur GitHub`}
              >
                {task.source === 'linear' && <span className="text-indigo-400">◆</span>}
                {task.source === 'github' && <FolderGit2 size={11} className="text-purple-400" />}
                <span>{task.key}</span>
                <ExternalLink size={9} />
              </a>
            ) : (
              <span className="text-[var(--accent-color)] font-mono">
                {task.key}
              </span>
            )}

            {task.branchName && (
              <span title={`Branche: ${task.branchName}`} className="text-indigo-400">
                <GitBranch size={11} />
              </span>
            )}
          </div>
        </td>

        {/* Title */}
        <td className="py-2.5 px-3 max-w-[280px]">
          <div className="text-xs font-semibold text-[var(--text-primary)] truncate group-hover:text-[var(--accent-color)] transition-colors">
            {task.title}
          </div>
          {task.description && (
            <div className="text-[11px] text-[var(--text-muted)] line-clamp-1 mt-0.5">
              {task.description}
            </div>
          )}
        </td>

        {/* Status quick changer */}
        <td className="py-2.5 px-3 whitespace-nowrap" onClick={e => e.stopPropagation()}>
          <select
            value={task.status}
            onChange={e => updateTask(task.id, { status: e.target.value as Status })}
            className="text-[11px] font-medium bg-[var(--bg-secondary)] text-[var(--text-secondary)] border border-[var(--border-color)] rounded-md px-2 py-1 focus:outline-none focus:border-[var(--accent-color)]"
          >
            {statusList.map(s => (
              <option key={s.id} value={s.id}>
                {s.label} ({s.stageLabel})
              </option>
            ))}
          </select>
        </td>

        {/* Quick Skill Runner */}
        <td className="py-2.5 px-3 whitespace-nowrap" onClick={e => e.stopPropagation()}>
          {nextSkill ? (
            <button
              onClick={() => runSkill(task.id, nextSkill.id)}
              disabled={isSkillRunning}
              className="inline-flex items-center gap-1 px-2 py-1 rounded-md text-[10px] font-bold text-white accent-bg shadow-xs hover:opacity-90 active:scale-95 transition-all"
              title={`Exécuter ${nextSkill.label}`}
            >
              {isSkillRunning && runningSkillId === nextSkill.id ? (
                <Loader2 size={10} className="animate-spin" />
              ) : (
                <>
                  <Sparkles size={10} className="text-amber-300" />
                  <span>{nextSkill.label}</span>
                </>
              )}
            </button>
          ) : (
            <span className="text-[10px] text-emerald-400 font-medium">Prêt</span>
          )}
        </td>

        {/* Priority */}
        <td className="py-2.5 px-3 whitespace-nowrap">
          {getPriorityBadge(task.priority)}
        </td>

        {/* Labels */}
        <td className="py-2.5 px-3">
          <div className="flex flex-wrap gap-1 max-w-[180px]">
            {task.labels && task.labels.map(lbl => {
              const lower = lbl.toLowerCase()
              let badgeStyle = 'bg-[var(--bg-tertiary)] text-[var(--text-secondary)] border-[var(--border-color)]'
              if (lower === 'new') badgeStyle = 'bg-cyan-500/15 text-cyan-400 border-cyan-500/30 font-semibold'
              else if (lower === 'clarified') badgeStyle = 'bg-amber-500/15 text-amber-400 border-amber-500/30 font-semibold'
              else if (lower === 'specified') badgeStyle = 'bg-blue-500/15 text-blue-400 border-blue-500/30 font-semibold'
              else if (lower === 'implemented') badgeStyle = 'bg-indigo-500/15 text-indigo-400 border-indigo-500/30 font-semibold'
              else if (lower === 'reviewed') badgeStyle = 'bg-emerald-500/15 text-emerald-400 border-emerald-500/30 font-semibold'

              return (
                <span
                  key={lbl}
                  className={`text-[10px] px-1.5 py-0.2 rounded border whitespace-nowrap ${badgeStyle}`}
                >
                  #{lbl}
                </span>
              )
            })}
          </div>
        </td>

        {/* Assignee */}
        <td className="py-2.5 px-3 whitespace-nowrap text-xs text-[var(--text-secondary)]">
          {task.assignee ? (
            <div className="flex items-center gap-1.5">
              <div className="w-5 h-5 rounded-full accent-bg text-white flex items-center justify-center text-[9px] font-bold">
                {task.assignee.substring(0, 2).toUpperCase()}
              </div>
              <span className="truncate max-w-[90px]">{task.assignee}</span>
            </div>
          ) : (
            <span className="text-[var(--text-muted)] text-[11px]">-</span>
          )}
        </td>

        {/* Due Date & PR */}
        <td className="py-2.5 px-3 whitespace-nowrap text-xs text-[var(--text-muted)]">
          <div className="flex items-center gap-2">
            {task.dueDate ? (
              <span className="flex items-center gap-1 text-[11px] text-amber-400 font-medium">
                <Calendar size={12} />
                {task.dueDate}
              </span>
            ) : (
              <span className="text-[11px] opacity-60">
                {new Date(task.createdAt).toLocaleDateString()}
              </span>
            )}
            {task.prUrl && (
              <a
                href={task.prUrl}
                target="_blank"
                rel="noreferrer"
                onClick={e => e.stopPropagation()}
                className="text-purple-400 hover:text-purple-300"
                title="Voir PR GitHub"
              >
                <ExternalLink size={12} />
              </a>
            )}
          </div>
        </td>

        {/* Delete action */}
        <td className="py-2.5 px-3 text-right whitespace-nowrap" onClick={e => e.stopPropagation()}>
          <button
            onClick={() => {
              if (window.confirm(t.taskModal.deleteConfirm)) {
                deleteTask(task.id)
              }
            }}
            className="p-1 rounded text-[var(--text-muted)] hover:text-rose-400 hover:bg-rose-500/10 transition-colors opacity-0 group-hover:opacity-100"
            title={t.taskModal.delete}
          >
            <Trash2 size={14} />
          </button>
        </td>
      </tr>
    )
  }

  return (
    <div className="flex-1 overflow-y-auto p-4 bg-[var(--bg-primary)]">
      <div className="max-w-7xl mx-auto space-y-4">
        {/* List Toolbar */}
        <div className="flex items-center justify-between gap-4 pb-2 border-b border-[var(--border-color)]">
          <div className="flex items-center gap-4">
            <span className="text-xs font-semibold text-[var(--text-secondary)]">
              {visibleTasks.length} {visibleTasks.length > 1 ? 'tâches visibles' : 'tâche visible'}
            </span>
            <label className="flex items-center gap-1.5 text-xs text-[var(--text-muted)] cursor-pointer hover:text-[var(--text-primary)] transition-colors">
              <input
                type="checkbox"
                checked={groupByStatus}
                onChange={e => setGroupByStatus(e.target.checked)}
                className="rounded text-[var(--accent-color)] focus:ring-0"
              />
              Grouper par statut
            </label>
            <label className="flex items-center gap-1.5 text-xs text-[var(--text-muted)] cursor-pointer hover:text-[var(--text-primary)] transition-colors">
              <input
                type="checkbox"
                checked={hideDone}
                onChange={() => toggleHideDone()}
                className="rounded text-[var(--accent-color)] focus:ring-0"
              />
              Masquer terminées ({doneTasksCount})
            </label>
          </div>
        </div>

        {visibleTasks.length === 0 ? (
          <div className="py-16 text-center text-[var(--text-muted)] space-y-3">
            <Clock size={32} className="mx-auto opacity-40" />
            <p className="text-sm font-medium">{t.list.empty}</p>
            {hideDone && doneTasksCount > 0 && (
              <button
                onClick={toggleHideDone}
                className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-semibold text-emerald-400 bg-emerald-500/10 hover:bg-emerald-500/20 border border-emerald-500/30 transition-colors"
              >
                <Eye size={13} />
                <span>Afficher les {doneTasksCount} tâches terminées</span>
              </button>
            )}
          </div>
        ) : groupByStatus ? (
          <div className="space-y-6">
            {statusList.map(st => {
              if ((st.id === 'to_close' || st.id === 'done') && hideDone) return null
              const groupTasks = sortedTasks.filter(t => 
                t.status === st.id ||
                (st.id === 'to_clarify' && t.status === 'backlog') ||
                (st.id === 'to_specify' && t.status === 'specified') ||
                (st.id === 'to_implement' && t.status === 'in_progress') ||
                (st.id === 'to_test' && t.status === 'to_validate') ||
                (st.id === 'to_close' && t.status === 'done')
              )
              if (groupTasks.length === 0) return null

              return (
                <div key={st.id} className="rounded-xl border border-[var(--border-color)] bg-[var(--bg-secondary)] overflow-hidden shadow-xs">
                  {/* Status Group Header */}
                  <div className="flex items-center justify-between px-4 py-2.5 bg-[var(--bg-tertiary)]/60 border-b border-[var(--border-color)]">
                    <div className="flex items-center gap-2">
                      <span className={st.color}>{st.icon}</span>
                      <h3 className="text-xs font-bold text-[var(--text-primary)]">
                        {st.label}
                      </h3>
                      <span className={`text-[10px] font-bold px-1.5 py-0.2 rounded border ${st.stageColor}`}>
                        {st.stageLabel}
                      </span>
                      <span className="text-[11px] font-mono font-bold px-1.5 py-0.2 rounded-full bg-[var(--bg-secondary)] text-[var(--text-secondary)]">
                        {groupTasks.length}
                      </span>
                    </div>

                    {(st.id === 'to_close' || st.id === 'done') && (
                      <button
                        onClick={toggleHideDone}
                        className="text-[11px] text-[var(--text-muted)] hover:text-emerald-400 flex items-center gap-1"
                        title={t.header.hideDone}
                      >
                        <EyeOff size={13} />
                        <span>Masquer</span>
                      </button>
                    )}
                  </div>

                  {/* Table */}
                  <div className="overflow-x-auto">
                    <table className="w-full text-left border-collapse">
                      <thead>
                        <tr className="border-b border-[var(--border-color)]/60 text-[11px] uppercase tracking-wider text-[var(--text-muted)] font-semibold">
                          <th className="py-2 px-3">{t.list.columns.key}</th>
                          <th className="py-2 px-3">{t.list.columns.title}</th>
                          <th className="py-2 px-3">{t.list.columns.status}</th>
                          <th className="py-2 px-3">Skill</th>
                          <th className="py-2 px-3">{t.list.columns.priority}</th>
                          <th className="py-2 px-3">{t.list.columns.labels}</th>
                          <th className="py-2 px-3">{t.list.columns.assignee}</th>
                          <th className="py-2 px-3">{t.list.columns.dueDate}</th>
                          <th className="py-2 px-3 text-right"></th>
                        </tr>
                      </thead>
                      <tbody>
                        {groupTasks.map(task => renderTaskRow(task))}
                      </tbody>
                    </table>
                  </div>
                </div>
              )
            })}

            {/* Notice if done tasks are hidden */}
            {hideDone && doneTasksCount > 0 && (
              <div className="flex items-center justify-between p-3 rounded-xl bg-[var(--bg-secondary)] border border-[var(--border-color)] text-xs text-[var(--text-muted)]">
                <div className="flex items-center gap-2">
                  <EyeOff size={14} className="text-slate-400" />
                  <span>{doneTasksCount} {doneTasksCount > 1 ? 'tâches terminées sont masquées' : 'tâche terminée est masquée'}</span>
                </div>
                <button
                  onClick={toggleHideDone}
                  className="flex items-center gap-1 px-2.5 py-1 rounded-lg text-xs font-bold text-emerald-400 bg-emerald-500/10 hover:bg-emerald-500/20 border border-emerald-500/30 transition-colors"
                >
                  <Eye size={12} />
                  <span>Afficher</span>
                </button>
              </div>
            )}
          </div>
        ) : (
          /* Flat Table */
          <div className="space-y-4">
            <div className="rounded-xl border border-[var(--border-color)] bg-[var(--bg-secondary)] overflow-hidden shadow-xs">
              <div className="overflow-x-auto">
                <table className="w-full text-left border-collapse">
                  <thead>
                    <tr className="border-b border-[var(--border-color)] text-[11px] uppercase tracking-wider text-[var(--text-muted)] font-semibold bg-[var(--bg-tertiary)]/60">
                      <th className="py-2.5 px-3 cursor-pointer hover:text-[var(--text-primary)]" onClick={() => handleSort('key')}>
                        <div className="flex items-center gap-1">{t.list.columns.key} <ArrowUpDown size={12} /></div>
                      </th>
                      <th className="py-2.5 px-3 cursor-pointer hover:text-[var(--text-primary)]" onClick={() => handleSort('title')}>
                        <div className="flex items-center gap-1">{t.list.columns.title} <ArrowUpDown size={12} /></div>
                      </th>
                      <th className="py-2.5 px-3 cursor-pointer hover:text-[var(--text-primary)]" onClick={() => handleSort('status')}>
                        <div className="flex items-center gap-1">{t.list.columns.status} <ArrowUpDown size={12} /></div>
                      </th>
                      <th className="py-2.5 px-3">Skill</th>
                      <th className="py-2.5 px-3 cursor-pointer hover:text-[var(--text-primary)]" onClick={() => handleSort('priority')}>
                        <div className="flex items-center gap-1">{t.list.columns.priority} <ArrowUpDown size={12} /></div>
                      </th>
                      <th className="py-2.5 px-3">{t.list.columns.labels}</th>
                      <th className="py-2.5 px-3">{t.list.columns.assignee}</th>
                      <th className="py-2.5 px-3 cursor-pointer hover:text-[var(--text-primary)]" onClick={() => handleSort('dueDate')}>
                        <div className="flex items-center gap-1">{t.list.columns.dueDate} <ArrowUpDown size={12} /></div>
                      </th>
                      <th className="py-2.5 px-3 text-right"></th>
                    </tr>
                  </thead>
                  <tbody>
                    {visibleTasks.map(task => renderTaskRow(task))}
                  </tbody>
                </table>
              </div>
            </div>

            {hideDone && doneTasksCount > 0 && (
              <div className="flex items-center justify-between p-3 rounded-xl bg-[var(--bg-secondary)] border border-[var(--border-color)] text-xs text-[var(--text-muted)]">
                <div className="flex items-center gap-2">
                  <EyeOff size={14} className="text-slate-400" />
                  <span>{doneTasksCount} {doneTasksCount > 1 ? 'tâches terminées sont masquées' : 'tâche terminée est masquée'}</span>
                </div>
                <button
                  onClick={toggleHideDone}
                  className="flex items-center gap-1 px-2.5 py-1 rounded-lg text-xs font-bold text-emerald-400 bg-emerald-500/10 hover:bg-emerald-500/20 border border-emerald-500/30 transition-colors"
                >
                  <Eye size={12} />
                  <span>Afficher</span>
                </button>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  )
}
