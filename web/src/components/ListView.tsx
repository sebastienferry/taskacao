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
  ChevronsDown,
  ChevronDown,
  Sparkles,
  Loader2,
  GitBranch,
  GitPullRequest,
  ExternalLink,
  FolderGit2,
  Eye,
  EyeOff,
  MessageSquare,
  Code2,
  Layers,
  Pin,
  ListChecks,
} from 'lucide-react'
import { useApp } from '../context/AppContext'
import { TaskFilters } from './TaskFilters'
import { CurationTable } from './CurationTable'
import type { Task, Status, Priority } from '../types'

export const ListView: React.FC = () => {
  const {
    tasks,
    advanceTask,
    launchInteractiveStep,
    isPinned,
    togglePin,
    setSelectedTask,
    setChatTask,
    setDiffTask,
    updateTask,
    deleteTask,
    activities,
    hideDone,
    toggleHideDone,
    openInEditor,
    settings,
    t,
  } = useApp()

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

  // Par défaut, le plus urgent en premier.
  const [sortField, setSortField] = useState<'key' | 'title' | 'status' | 'priority' | 'dueDate' | 'createdAt'>('priority')
  // Avance en cours, par tâche : deux lignes ne doivent pas se bloquer l'une
  // l'autre.
  const [advancing, setAdvancing] = useState<Record<string, 'step' | 'auto'>>({})
  const [sortAsc, setSortAsc] = useState(false)
  const [groupByStatus, setGroupByStatus] = useState(true)
  // Mode triage : la même vue, mais orientée « qu'est-ce qui n'est rattaché à
  // rien », avec les champs éditables dans la ligne. Le choix est mémorisé, sinon
  // il faut le refaire à chaque retour dans la vue.
  const [curationMode, setCurationMode] = useState<boolean>(() => {
    try {
      return localStorage.getItem('taskacao_list_curation') === '1'
    } catch {
      return false
    }
  })

  const toggleCuration = () => {
    setCurationMode(prev => {
      const next = !prev
      try {
        localStorage.setItem('taskacao_list_curation', next ? '1' : '0')
      } catch {
        // stockage indisponible : le mode vaut pour cette session
      }
      return next
    })
  }

  const handleSort = (field: typeof sortField) => {
    if (sortField === field) {
      setSortAsc(prev => !prev)
    } else {
      setSortField(field)
      // Priority reads best highest-first; the other columns read best ascending.
      setSortAsc(field !== 'priority')
    }
  }

  const handleAdvance = async (task: Task, auto: boolean) => {
    if (advancing[task.id]) return
    setAdvancing(prev => ({ ...prev, [task.id]: auto ? 'auto' : 'step' }))
    const result = await advanceTask(task.id, auto)
    if (result?.mode === 'interactive' && result.skillId) {
      await launchInteractiveStep(task, result.skillId, result.label || 'Étape interactive')
    }
    setAdvancing(prev => {
      const next = { ...prev }
      delete next[task.id]
      return next
    })
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

  const doneTasksCount = tasks.filter(t => t.status === 'finished' || t.status === 'done').length

  const visibleTasks = hideDone
    ? sortedTasks.filter(t => t.status !== 'finished' && t.status !== 'done')
    : sortedTasks

  const statusList: { id: Status; label: string; stageLabel: string; stageColor: string; icon: React.ReactNode; color: string }[] = [
    { id: 'to_clarify', label: t.status.to_clarify, stageLabel: '#new', stageColor: 'bg-cyan-500/15 text-cyan-400 border-cyan-500/30', icon: <Sparkles size={14} />, color: 'text-cyan-400' },
    { id: 'to_specify', label: t.status.to_specify, stageLabel: '#clarified', stageColor: 'bg-amber-500/15 text-amber-400 border-amber-500/30', icon: <HelpCircle size={14} />, color: 'text-amber-400' },
    { id: 'to_implement', label: t.status.to_implement, stageLabel: '#specified', stageColor: 'bg-blue-500/15 text-blue-400 border-blue-500/30', icon: <FileCode size={14} />, color: 'text-blue-400' },
    { id: 'to_test', label: t.status.to_test, stageLabel: '#implemented', stageColor: 'bg-indigo-500/15 text-indigo-400 border-indigo-500/30', icon: <Flame size={14} />, color: 'text-indigo-400' },
    { id: 'to_close', label: t.status.to_close, stageLabel: '#reviewed', stageColor: 'bg-purple-500/15 text-purple-400 border-purple-500/30', icon: <ShieldCheck size={14} />, color: 'text-purple-400' },
    { id: 'finished', label: t.status.finished, stageLabel: '#finished', stageColor: 'bg-emerald-500/15 text-emerald-400 border-emerald-500/30', icon: <CheckCircle2 size={14} />, color: 'text-emerald-400' },
  ]

  // Même pastille que sur les cartes du board : la couleur porte l'information,
  // le libellé reste en infobulle. La colonne s'appelle déjà « Priorité ».
  const PRIORITY_DOTS: Record<Priority, { color: string; label: string }> = {
    urgent: { color: 'var(--status-danger)', label: t.priority.urgent },
    high: { color: 'var(--status-warn)', label: t.priority.high },
    medium: { color: 'var(--status-info)', label: t.priority.medium },
    low: { color: 'var(--text-muted)', label: t.priority.low },
  }

  const getPriorityBadge = (priority: Priority) => {
    const dot = PRIORITY_DOTS[priority]
    if (!dot) return null
    return (
      <span className="inline-flex items-center" title={`Priorité : ${dot.label}`}>
        <span
          className="w-2.5 h-2.5 rounded-full shrink-0 ring-1 ring-black/10"
          style={{ backgroundColor: dot.color }}
        />
      </span>
    )
  }

  const renderTaskRow = (task: Task) => {
    // Plus de pas possible sur une tâche terminée : les chevrons sont éteints
    // plutôt que de renvoyer une erreur au clic.
    const isFinished = task.status === 'finished' || task.status === 'done'
    const taskActs = activities?.filter(a => a.taskId === task.id) || []
    const latestActivity = taskActs.find(a => a.status === 'running' || a.status === 'queued' || a.status === 'pending') || taskActs[0]

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
                    : task.source === 'jira'
                    ? 'bg-blue-500/15 text-blue-400 hover:bg-blue-500/25 border border-blue-500/30'
                    : 'bg-[var(--accent-light)] text-[var(--accent-color)] hover:opacity-80'
                }`}
                title={
                  task.source === 'linear'
                    ? `Ouvrir ${task.key} sur Linear`
                    : task.source === 'github'
                    ? `Ouvrir ${task.key} sur GitHub`
                    : task.source === 'jira'
                    ? `Ouvrir ${task.key} sur Jira`
                    : `Ouvrir ${task.key}`
                }
              >
                {task.source === 'linear' && <span className="text-indigo-400">◆</span>}
                {task.source === 'github' && <FolderGit2 size={11} className="text-purple-400" />}
                {task.source === 'jira' && <span className="text-blue-400 font-sans font-black text-[9px]">J</span>}
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

        {/* Title & Activity */}
        <td className="py-2.5 px-3 max-w-[560px]">
          <div className="flex items-center gap-1.5">
            <span className="text-xs font-semibold text-[var(--text-primary)] truncate group-hover:text-[var(--accent-color)] transition-colors">
              {task.title}
            </span>
          </div>

          {/* Parent (Epic ou Story) — propriété du tracker, jamais une tâche */}
          {task.parentKey && (
            <div
              className="inline-flex items-center gap-1 px-1.5 py-0.2 rounded text-[10px] mt-1 mr-1 text-violet-300 bg-violet-500/10 border border-violet-500/25 max-w-[220px]"
              title={`${task.parentType || 'Parent'} ${task.parentKey}${task.parentTitle ? ` — ${task.parentTitle}` : ''}`}
            >
              <Layers size={9} className="shrink-0 opacity-80" />
              <span className="font-mono font-bold shrink-0">{task.parentKey}</span>
              {task.parentTitle && <span className="truncate opacity-80">{task.parentTitle}</span>}
            </div>
          )}

          {/* Activity badge if exists */}
          {latestActivity && (
            <div
              onClick={(e) => {
                e.stopPropagation()
                setChatTask(task)
              }}
              className={`inline-flex items-center gap-1 px-1.5 py-0.2 rounded text-[10px] mt-1 border transition-colors ${
                latestActivity.status === 'running'
                  ? 'bg-indigo-500/10 border-indigo-500/30 text-indigo-300 animate-pulse'
                  : latestActivity.status === 'failed'
                  ? 'bg-rose-500/10 border-rose-500/25 text-rose-300'
                  : latestActivity.status === 'queued' || latestActivity.status === 'pending'
                  ? 'bg-amber-500/10 border-amber-500/25 text-amber-300'
                  : 'bg-emerald-500/10 border-emerald-500/20 text-emerald-300'
              }`}
              title={`Activité: ${latestActivity.skillName || latestActivity.action} (${latestActivity.status}) - Cliquer pour ouvrir`}
            >
              {latestActivity.status === 'running' ? (
                <Loader2 size={9} className="animate-spin text-indigo-400" />
              ) : latestActivity.status === 'failed' ? (
                <AlertCircle size={9} className="text-rose-400" />
              ) : latestActivity.status === 'queued' || latestActivity.status === 'pending' ? (
                <Clock size={9} className="text-amber-400" />
              ) : (
                <CheckCircle2 size={9} className="text-emerald-400" />
              )}
              <span className="font-bold">{latestActivity.skillName || latestActivity.action}</span>
              <span className="opacity-70 font-mono text-[9px]">
                • {latestActivity.status === 'running' ? 'en cours' : formatRelativeTime(latestActivity.completedAt || latestActivity.startedAt || latestActivity.createdAt)}
              </span>
            </div>
          )}

          {task.description && !latestActivity && (
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

        {/* Quick Skill Runner & Agent Chat */}
        <td className="py-2.5 px-3 whitespace-nowrap" onClick={e => e.stopPropagation()}>
          <div className="flex items-center gap-1.5">
            <button
              onClick={() => togglePin(task.id)}
              className={`inline-flex items-center justify-center p-1 rounded-md border transition-colors cursor-pointer ${
                isPinned(task.id)
                  ? 'accent-text bg-[var(--accent-light)] border-[var(--accent-color)]/40'
                  : 'text-[var(--text-muted)] hover:text-[var(--accent-color)] border-transparent hover:border-[var(--accent-color)]/30'
              }`}
              title={isPinned(task.id) ? 'Retirer de la barre des épinglés' : 'Épingler pour basculer vite dessus'}
            >
              <Pin size={12} />
            </button>

            <button
              onClick={() => setChatTask(task)}
              className="inline-flex items-center gap-1 px-2 py-1 rounded-md text-[10px] font-bold text-white bg-[var(--accent-color)] shadow-xs hover:opacity-90 active:scale-95 transition-all cursor-pointer"
              title="💬 Discuter de cette tâche avec l'agent Copilot"
            >
              <MessageSquare size={10} />
              <span>Discuter</span>
            </button>

            {/* Un pas de workflow, puis la chaîne autonome. Chevrons vers le
                bas : dans un tableau, l'avancement se lit verticalement. */}
            <button
              onClick={() => handleAdvance(task, false)}
              disabled={Boolean(advancing[task.id]) || isFinished}
              className="inline-flex items-center justify-center p-1 rounded-md text-[var(--text-muted)] hover:text-[var(--accent-color)] hover:bg-[var(--accent-light)] border border-transparent hover:border-[var(--accent-color)]/30 transition-colors cursor-pointer disabled:opacity-40"
              title="Avancer d'un pas dans le workflow agentique"
            >
              {advancing[task.id] === 'step' ? <Loader2 size={12} className="animate-spin" /> : <ChevronDown size={12} />}
            </button>

            <button
              onClick={() => handleAdvance(task, true)}
              disabled={Boolean(advancing[task.id]) || isFinished}
              className="inline-flex items-center justify-center p-1 rounded-md text-[var(--text-muted)] hover:text-[var(--accent-color)] hover:bg-[var(--accent-light)] border border-transparent hover:border-[var(--accent-color)]/30 transition-colors cursor-pointer disabled:opacity-40"
              title="Avancer en autonomie jusqu'à l'étape de revue"
            >
              {advancing[task.id] === 'auto' ? <Loader2 size={12} className="animate-spin" /> : <ChevronsDown size={12} />}
            </button>

            {isFinished && <span className="text-[10px] text-emerald-400 font-medium">Terminé</span>}
          </div>
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
              if (lower === 'new' || lower === 'untouched') badgeStyle = 'bg-cyan-500/15 text-cyan-400 border-cyan-500/30 font-semibold'
              else if (lower === 'clarified') badgeStyle = 'bg-amber-500/15 text-amber-400 border-amber-500/30 font-semibold'
              else if (lower === 'specified') badgeStyle = 'bg-blue-500/15 text-blue-400 border-blue-500/30 font-semibold'
              else if (lower === 'implemented') badgeStyle = 'bg-indigo-500/15 text-indigo-400 border-indigo-500/30 font-semibold'
              else if (lower === 'reviewed') badgeStyle = 'bg-purple-500/15 text-purple-400 border-purple-500/30 font-semibold'
              else if (lower === 'finished' || lower === 'closed') badgeStyle = 'bg-emerald-500/15 text-emerald-400 border-emerald-500/30 font-semibold'

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

        {/* Due Date & Git / PR / MR */}
        <td className="py-2.5 px-3 whitespace-nowrap text-xs text-[var(--text-muted)]">
          <div className="flex flex-col gap-1">
            <div className="flex items-center gap-2">
              {task.dueDate ? (
                <span className="flex items-center gap-1 text-[11px] text-amber-400 font-medium">
                  <Calendar size={12} />
                  {task.dueDate}
                </span>
              ) : (
                <span className="text-[10px] opacity-60 font-mono">
                  {new Date(task.createdAt).toLocaleDateString()}
                </span>
              )}

              {task.branchName && (
                <div
                  onClick={(e) => {
                    e.stopPropagation()
                    setDiffTask(task)
                  }}
                  className="inline-flex items-center gap-1 px-1.5 py-0.2 rounded text-[10px] font-mono text-indigo-300 bg-indigo-500/10 hover:bg-indigo-500/20 border border-indigo-500/25 transition-colors cursor-pointer group/branch"
                  title={`Branche: ${task.branchName} (Cliquer pour voir le Diff Git)`}
                >
                  <GitBranch size={10} className="text-indigo-400 shrink-0 group-hover/branch:scale-110 transition-transform" />
                  <span className="truncate max-w-[100px]">{task.branchName}</span>
                </div>
              )}
            </div>

            {task.prUrl && (
              <a
                href={task.prUrl}
                target="_blank"
                rel="noreferrer"
                onClick={e => e.stopPropagation()}
                className={`inline-flex items-center gap-1 px-1.5 py-0.2 rounded text-[10px] font-mono font-bold transition-all w-fit ${
                  task.prUrl.includes('gitlab')
                    ? 'bg-orange-500/15 text-orange-400 hover:bg-orange-500/25 border border-orange-500/30'
                    : 'bg-purple-500/15 text-purple-400 hover:bg-purple-500/25 border border-purple-500/30'
                }`}
                title={task.prUrl.includes('gitlab') ? `Voir MR GitLab : ${task.prUrl}` : `Voir PR GitHub : ${task.prUrl}`}
              >
                <GitPullRequest size={10} className={task.prUrl.includes('gitlab') ? 'text-orange-400' : 'text-purple-400'} />
                <span>{task.prUrl.includes('gitlab') ? 'GitLab MR' : 'GitHub PR'}</span>
                <ExternalLink size={8} />
              </a>
            )}
          </div>
        </td>

        {/* Actions */}
        <td className="py-2.5 px-3 text-right whitespace-nowrap" onClick={e => e.stopPropagation()}>
          <div className="flex items-center justify-end gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
            <button
              type="button"
              onClick={() => openInEditor({ taskId: task.id })}
              className="p-1 rounded text-[var(--text-muted)] hover:text-cyan-400 hover:bg-cyan-500/10 transition-colors"
              title={`Ouvrir le code / worktree dans ${settings.editorCommand || 'VS Code'}`}
            >
              <Code2 size={14} />
            </button>
            <button
              type="button"
              onClick={() => {
                if (window.confirm(t.taskModal.deleteConfirm)) {
                  deleteTask(task.id)
                }
              }}
              className="p-1 rounded text-[var(--text-muted)] hover:text-rose-400 hover:bg-rose-500/10 transition-colors"
              title={t.taskModal.delete}
            >
              <Trash2 size={14} />
            </button>
          </div>
        </td>
      </tr>
    )
  }

  return (
    <div className="flex-1 overflow-y-auto p-4 bg-[var(--bg-primary)]">
      <div className="space-y-4">
        {/* List Toolbar */}
        <div className="flex items-center justify-between gap-4 pb-2 border-b border-[var(--border-color)]">
          <div className="flex items-center gap-4">
            <span className="text-xs font-semibold text-[var(--text-secondary)]">
              {visibleTasks.length} {visibleTasks.length > 1 ? 'tâches visibles' : 'tâche visible'}
            </span>
            <label className={`flex items-center gap-1.5 text-xs text-[var(--text-muted)] cursor-pointer hover:text-[var(--text-primary)] transition-colors ${curationMode ? 'hidden' : ''}`}>
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

          <div className="flex items-center gap-2 shrink-0">
            <button
              type="button"
              onClick={toggleCuration}
              title="Triage : ce qui n'a ni sprint, ni équipe, ni épic, ni assigné, modifiable dans la ligne"
              className={`flex items-center gap-1.5 px-2 py-1 rounded-md text-xs font-semibold border transition-colors cursor-pointer ${
                curationMode
                  ? 'bg-[var(--accent-light)] accent-text border-[var(--accent-color)]/40'
                  : 'bg-[var(--bg-secondary)] text-[var(--text-secondary)] border-[var(--border-color)] hover:text-[var(--text-primary)]'
              }`}
            >
              <ListChecks size={12} />
              <span>Triage</span>
            </button>
            <TaskFilters />
            <button
              type="button"
              onClick={() => handleSort('priority')}
              title="Trier par priorité"
              className={`flex items-center gap-1 px-2 py-1 rounded-md text-xs font-semibold border transition-colors cursor-pointer ${
                sortField === 'priority'
                  ? 'bg-[var(--accent-light)] accent-text border-[var(--accent-color)]/40'
                  : 'bg-[var(--bg-secondary)] text-[var(--text-secondary)] border-[var(--border-color)] hover:text-[var(--text-primary)]'
              }`}
            >
              <ArrowUpDown size={11} />
              <span>Priorité</span>
              {sortField === 'priority' && <span className="font-mono">{sortAsc ? '↑' : '↓'}</span>}
            </button>
          </div>
        </div>

        {curationMode ? (
          <CurationTable />
        ) : visibleTasks.length === 0 ? (
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
              if ((st.id === 'finished' || st.id === 'done') && hideDone) return null
              const groupTasks = sortedTasks.filter(t => 
                t.status === st.id ||
                (st.id === 'to_clarify' && t.status === 'backlog') ||
                (st.id === 'to_specify' && t.status === 'specified') ||
                (st.id === 'to_implement' && t.status === 'in_progress') ||
                (st.id === 'to_test' && t.status === 'to_validate') ||
                (st.id === 'finished' && t.status === 'done')
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

                    {(st.id === 'finished' || st.id === 'done') && (
                      <button
                        onClick={toggleHideDone}
                        className="text-[11px] text-[var(--text-muted)] hover:text-emerald-400 flex items-center gap-1 cursor-pointer"
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
