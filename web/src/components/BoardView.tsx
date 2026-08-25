import React, { useState } from 'react'
import {
  HelpCircle,
  FileCode,
  Flame,
  ShieldCheck,
  CheckCircle2,
  Plus,
  Eye,
  EyeOff,
  Sparkles,
  Kanban,
  ListFilter
} from 'lucide-react'
import { useApp } from '../context/AppContext'
import { TaskCard } from './TaskCard'
import { TaskFilters } from './TaskFilters'
import type { Task, Status, WorkflowStage, Priority } from '../types'

interface WorkflowColumnConfig {
  id: WorkflowStage
  title: string
  stageLabel: string
  stageColor: string
  icon: React.ReactNode
  color: string
  borderColor: string
  badgeBg: string
  description: string
}

interface StatusColumnConfig {
  id: Status
  title: string
  stageLabel: string
  stageColor: string
  icon: React.ReactNode
  color: string
  borderColor: string
  badgeBg: string
  description: string
}

export const getTaskWorkflowStage = (task: Task): WorkflowStage => {
  const labels = (task.labels || []).map(l => l.toLowerCase())
  if (labels.includes('finished') || labels.includes('closed') || labels.includes('done')) return 'finished'
  if (labels.includes('reviewed')) return 'reviewed'
  if (labels.includes('implemented')) return 'implemented'
  if (labels.includes('specified')) return 'specified'
  if (labels.includes('clarified')) return 'clarified'
  if (labels.includes('new') || labels.includes('untouched')) return 'new'

  // Fallback from status if no workflow label is present
  if (task.status === 'finished' || task.status === 'done') return 'finished'
  if (task.status === 'to_close') return 'reviewed'
  if (task.status === 'to_test' || task.status === 'to_validate') return 'implemented'
  if (task.status === 'to_implement' || task.status === 'in_progress') return 'specified'
  if (task.status === 'to_specify' || task.status === 'specified') return 'clarified'
  return 'new'
}

export const BoardView: React.FC = () => {
  const {
    tasks,
    moveTask,
    moveTaskWorkflowStage,
    boardGrouping,
    setBoardGrouping,
    hideDone,
    toggleHideDone,
    setIsQuickAddOpen,
    setQuickAddInitialStatus,
    t,
  } = useApp()

  const [dragOverColumn, setDragOverColumn] = useState<string | null>(null)
  const [draggingTaskId, setDraggingTaskId] = useState<string | null>(null)

  // Le plus urgent en haut de chaque colonne. Le glisser-déposer ne réordonne
  // pas à l'intérieur d'une colonne — il change d'étape — donc trier ici
  // n'écrase aucun ordre manuel.
  const PRIORITY_RANK: Record<Priority, number> = { urgent: 4, high: 3, medium: 2, low: 1 }
  const byPriorityDesc = (list: Task[]) =>
    [...list].sort((a, b) => (PRIORITY_RANK[b.priority] || 0) - (PRIORITY_RANK[a.priority] || 0))

  // -------------------------------------------------------------
  // MODE 1: Workflow Labels Columns (Pipeline IA)
  // new -> clarified -> specified -> implemented -> reviewed -> finished
  // -------------------------------------------------------------
  const workflowColumns: WorkflowColumnConfig[] = [
    {
      id: 'new',
      title: 'New',
      stageLabel: '#new',
      stageColor: 'bg-cyan-500/15 text-cyan-400 border-cyan-500/30',
      icon: <Sparkles size={16} className="text-cyan-400" />,
      color: 'text-cyan-400',
      borderColor: 'border-cyan-500/30',
      badgeBg: 'bg-cyan-500/20 text-cyan-300',
      description: 'Nouveaux tickets non encore traités par IA',
    },
    {
      id: 'clarified',
      title: 'Clarified',
      stageLabel: '#clarified',
      stageColor: 'bg-amber-500/15 text-amber-400 border-amber-500/30',
      icon: <HelpCircle size={16} className="text-amber-400" />,
      color: 'text-amber-400',
      borderColor: 'border-amber-500/30',
      badgeBg: 'bg-amber-500/20 text-amber-300',
      description: 'Cadrage & Q&A validés',
    },
    {
      id: 'specified',
      title: 'Specified',
      stageLabel: '#specified',
      stageColor: 'bg-blue-500/15 text-blue-400 border-blue-500/30',
      icon: <FileCode size={16} className="text-blue-400" />,
      color: 'text-blue-400',
      borderColor: 'border-blue-500/30',
      badgeBg: 'bg-blue-500/20 text-blue-300',
      description: 'Spécification technique prête',
    },
    {
      id: 'implemented',
      title: 'Implemented',
      stageLabel: '#implemented',
      stageColor: 'bg-indigo-500/15 text-indigo-400 border-indigo-500/30',
      icon: <Flame size={16} className="text-indigo-400" />,
      color: 'text-indigo-400',
      borderColor: 'border-indigo-500/30',
      badgeBg: 'bg-indigo-500/20 text-indigo-300',
      description: 'Code développé & commité',
    },
    {
      id: 'reviewed',
      title: 'Reviewed',
      stageLabel: '#reviewed',
      stageColor: 'bg-purple-500/15 text-purple-400 border-purple-500/30',
      icon: <ShieldCheck size={16} className="text-purple-400" />,
      color: 'text-purple-400',
      borderColor: 'border-purple-500/30',
      badgeBg: 'bg-purple-500/20 text-purple-300',
      description: 'Pull Request & revue prêtes',
    },
    {
      id: 'finished',
      title: 'Finished',
      stageLabel: '#finished',
      stageColor: 'bg-emerald-500/15 text-emerald-400 border-emerald-500/30',
      icon: <CheckCircle2 size={16} className="text-emerald-400" />,
      color: 'text-emerald-400',
      borderColor: 'border-emerald-500/30',
      badgeBg: 'bg-emerald-500/20 text-emerald-300',
      description: 'Tâche finalisée & mergée',
    },
  ]

  // -------------------------------------------------------------
  // MODE 2: Operational Status Columns (Kanban Agile)
  // Todo -> In Progress -> In Review & Testing -> Done
  // -------------------------------------------------------------
  const isTaskInStatusColumn = (taskStatus: Status, colId: Status) => {
    if (colId === 'to_clarify') {
      return taskStatus === 'to_clarify' || taskStatus === 'backlog'
    }
    if (colId === 'in_progress') {
      return taskStatus === 'in_progress' || taskStatus === 'to_specify' || taskStatus === 'to_implement' || taskStatus === 'specified'
    }
    if (colId === 'to_test') {
      return taskStatus === 'to_test' || taskStatus === 'to_validate' || taskStatus === 'to_close'
    }
    if (colId === 'to_close' || colId === 'finished') {
      return taskStatus === 'finished' || taskStatus === 'done'
    }
    return taskStatus === colId
  }

  const statusColumns: StatusColumnConfig[] = [
    {
      id: 'to_clarify',
      title: 'Todo',
      stageLabel: 'Backlog',
      stageColor: 'bg-cyan-500/15 text-cyan-400 border-cyan-500/30',
      icon: <ListFilter size={16} className="text-cyan-400" />,
      color: 'text-cyan-400',
      borderColor: 'border-cyan-500/30',
      badgeBg: 'bg-cyan-500/20 text-cyan-300',
      description: 'Tâches à faire',
    },
    {
      id: 'in_progress',
      title: 'In Progress',
      stageLabel: 'En cours',
      stageColor: 'bg-blue-500/15 text-blue-400 border-blue-500/30',
      icon: <Flame size={16} className="text-blue-400" />,
      color: 'text-blue-400',
      borderColor: 'border-blue-500/30',
      badgeBg: 'bg-blue-500/20 text-blue-300',
      description: 'En cours de dev / spec',
    },
    {
      id: 'to_test',
      title: 'In Review & Testing',
      stageLabel: 'Revue & Tests',
      stageColor: 'bg-purple-500/15 text-purple-400 border-purple-500/30',
      icon: <ShieldCheck size={16} className="text-purple-400" />,
      color: 'text-purple-400',
      borderColor: 'border-purple-500/30',
      badgeBg: 'bg-purple-500/20 text-purple-300',
      description: 'Prêt pour test & merge',
    },
    {
      id: 'finished',
      title: 'Done',
      stageLabel: 'Terminé',
      stageColor: 'bg-emerald-500/15 text-emerald-400 border-emerald-500/30',
      icon: <CheckCircle2 size={16} className="text-emerald-400" />,
      color: 'text-emerald-400',
      borderColor: 'border-emerald-500/30',
      badgeBg: 'bg-emerald-500/20 text-emerald-300',
      description: 'Tickets fermés / validés',
    },
  ]

  const handleDragOver = (e: React.DragEvent, colId: string) => {
    e.preventDefault()
    e.dataTransfer.dropEffect = 'move'
    if (dragOverColumn !== colId) {
      setDragOverColumn(colId)
    }
  }

  const handleDragLeave = (e: React.DragEvent, colId: string) => {
    e.preventDefault()
    const rect = (e.currentTarget as HTMLElement).getBoundingClientRect()
    if (
      e.clientX < rect.left ||
      e.clientX >= rect.right ||
      e.clientY < rect.top ||
      e.clientY >= rect.bottom
    ) {
      if (dragOverColumn === colId) {
        setDragOverColumn(null)
      }
    }
  }

  const handleDropWorkflow = async (e: React.DragEvent, targetStage: WorkflowStage) => {
    e.preventDefault()
    setDragOverColumn(null)
    const taskId = e.dataTransfer.getData('text/plain') || draggingTaskId
    setDraggingTaskId(null)

    if (!taskId) return
    const task = tasks.find(t => t.id === taskId || t.key === taskId)
    if (!task) return

    const currentStage = getTaskWorkflowStage(task)
    if (currentStage === targetStage) return

    await moveTaskWorkflowStage(task.id, targetStage)
  }

  const handleDropStatus = async (e: React.DragEvent, targetStatus: Status) => {
    e.preventDefault()
    setDragOverColumn(null)
    const taskId = e.dataTransfer.getData('text/plain') || draggingTaskId
    setDraggingTaskId(null)

    if (!taskId) return
    const task = tasks.find(t => t.id === taskId || t.key === taskId)
    if (!task || isTaskInStatusColumn(task.status, targetStatus)) return

    const targetTasks = tasks.filter(t => isTaskInStatusColumn(t.status, targetStatus))
    const newPos = targetTasks.length

    await moveTask(task.id, targetStatus, newPos)
  }

  const openQuickAddForWorkflow = (stage: WorkflowStage) => {
    let initialSt: Status = 'to_clarify'
    if (stage === 'specified') initialSt = 'to_implement'
    else if (stage === 'implemented' || stage === 'reviewed') initialSt = 'to_test'
    else if (stage === 'finished') initialSt = 'to_close'
    setQuickAddInitialStatus(initialSt)
    setIsQuickAddOpen(true)
  }

  const openQuickAddForStatus = (status: Status) => {
    setQuickAddInitialStatus(status)
    setIsQuickAddOpen(true)
  }

  return (
    <div className="flex-1 flex flex-col h-full overflow-hidden bg-[var(--bg-primary)] select-none">
      {/* Board Top Toolbar: Grouping Mode Switcher */}
      {/* View Subheader: View Mode Switcher (Icon-only) & Hide/Show Done Filter */}
      <div className="flex items-center justify-between px-4 py-2 border-b border-[var(--border-color)] bg-[var(--bg-secondary)]/50 shrink-0">
        <div className="flex items-center gap-3">
          <div className="flex items-center p-0.5 rounded-lg bg-[var(--bg-tertiary)] border border-[var(--border-color)] shadow-2xs">
            <button
              onClick={() => setBoardGrouping('workflow')}
              className={`p-1.5 rounded-md transition-all cursor-pointer ${
                boardGrouping === 'workflow'
                  ? 'bg-[var(--accent-color)] text-white shadow-xs'
                  : 'text-[var(--text-secondary)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-secondary)]'
              }`}
              title="Workflow Agentic (Labels : new ➔ clarified ➔ specified ➔ implemented ➔ reviewed ➔ finished)"
            >
              <Sparkles size={15} className={boardGrouping === 'workflow' ? 'text-amber-300' : 'text-amber-400'} />
            </button>

            <button
              onClick={() => setBoardGrouping('status')}
              className={`p-1.5 rounded-md transition-all cursor-pointer ${
                boardGrouping === 'status'
                  ? 'bg-[var(--accent-color)] text-white shadow-xs'
                  : 'text-[var(--text-secondary)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-secondary)]'
              }`}
              title="Board Agile (Statuts : Todo ➔ In Progress ➔ In Review & Testing ➔ Done)"
            >
              <Kanban size={15} className={boardGrouping === 'status' ? 'text-cyan-300' : 'text-cyan-400'} />
            </button>
          </div>
        </div>

        <div className="flex items-center gap-2">
          <TaskFilters />

          <button
            onClick={toggleHideDone}
            className={`flex items-center gap-1.5 px-2.5 py-1 rounded-lg text-xs font-medium border transition-colors cursor-pointer ${
              hideDone
                ? 'bg-emerald-500/15 text-emerald-300 border-emerald-500/40 shadow-2xs'
                : 'bg-[var(--bg-tertiary)] text-[var(--text-secondary)] border-[var(--border-color)] hover:text-[var(--text-primary)]'
            }`}
            title={hideDone ? 'Afficher les tâches terminées' : 'Masquer les tâches terminées'}
          >
            {hideDone ? <Eye size={13} className="text-emerald-400" /> : <EyeOff size={13} />}
            <span>{hideDone ? 'Afficher Terminé' : 'Masquer Terminé'}</span>
          </button>
        </div>
      </div>

      {/* Board Scrollable Columns Container */}
      <div className="flex-1 overflow-x-auto overflow-y-hidden p-4">
        <div className="flex gap-4 h-full min-w-max pb-2">
          {/* ========================================================= */}
          {/* RENDER MODE 1: WORKFLOW LABELS PIPELINE                   */}
          {/* ========================================================= */}
          {boardGrouping === 'workflow' &&
            workflowColumns.map(col => {
              const colTasks = byPriorityDesc(tasks.filter(t => getTaskWorkflowStage(t) === col.id))
              const isOver = dragOverColumn === col.id

              // Collapsed Finished Column when hideDone is enabled
              if (col.id === 'finished' && hideDone) {
                return (
                  <div
                    key={col.id}
                    onDragOver={e => handleDragOver(e, col.id)}
                    onDragLeave={e => handleDragLeave(e, col.id)}
                    onDrop={e => handleDropWorkflow(e, col.id)}
                    onClick={toggleHideDone}
                    className={`w-14 shrink-0 flex flex-col items-center justify-between py-4 rounded-2xl bg-[var(--bg-secondary)]/60 border transition-all duration-200 cursor-pointer group hover:bg-[var(--bg-secondary)] select-none ${
                      isOver
                        ? 'border-emerald-500 ring-2 ring-emerald-500/30 bg-emerald-500/15'
                        : 'border-[var(--border-color)] hover:border-emerald-500/40'
                    }`}
                    title="Colonne Finished masquée - Glissez une tâche ici pour la terminer, ou cliquez pour l'afficher"
                  >
                    <div className="flex flex-col items-center gap-1.5">
                      <CheckCircle2 size={18} className="text-emerald-400" />
                      <span className="text-[10px] font-mono font-bold px-1.5 py-0.2 rounded-full bg-emerald-500/20 text-emerald-300">
                        {colTasks.length}
                      </span>
                    </div>

                    <div className="flex-1 flex items-center justify-center my-4">
                      <span className="[writing-mode:vertical-lr] rotate-180 text-xs font-bold text-[var(--text-secondary)] group-hover:text-emerald-400 tracking-wider transition-colors">
                        Finished ({colTasks.length})
                      </span>
                    </div>

                    <div className="p-1.5 rounded-lg bg-[var(--bg-tertiary)] text-[var(--text-muted)] group-hover:text-emerald-400 transition-colors">
                      <Eye size={14} />
                    </div>
                  </div>
                )
              }

              return (
                <div
                  key={col.id}
                  onDragOver={e => handleDragOver(e, col.id)}
                  onDragLeave={e => handleDragLeave(e, col.id)}
                  onDrop={e => handleDropWorkflow(e, col.id)}
                  className={`kanban-column w-[320px] min-w-[290px] shrink-0 flex flex-col rounded-2xl bg-[var(--bg-secondary)]/70 border column-neon-${col.id} transition-all duration-200 ${
                    isOver
                      ? 'border-[var(--accent-color)] ring-2 ring-[var(--accent-glow)] bg-[var(--accent-light)]/20'
                      : 'border-[var(--border-color)]'
                  }`}
                >
                  {/* Column Header */}
                  <div className="flex items-center justify-between p-3 border-b border-[var(--border-color)]">
                    <div className="flex items-center gap-2">
                      <span>{col.icon}</span>
                      <h3 className="text-xs font-bold text-[var(--text-primary)] tracking-wide">
                        {col.title}
                      </h3>
                      <span className={`text-[10px] font-bold px-1.5 py-0.2 rounded border ${col.stageColor}`}>
                        {col.stageLabel}
                      </span>
                      <span className={`text-[11px] font-mono px-1.5 py-0.2 rounded-full font-bold ${col.badgeBg}`}>
                        {colTasks.length}
                      </span>
                    </div>

                    <div className="flex items-center gap-1">
                      {col.id === 'finished' && (
                        <button
                          onClick={toggleHideDone}
                          className="p-1 rounded-md text-[var(--text-muted)] hover:text-emerald-400 hover:bg-[var(--bg-tertiary)] transition-colors cursor-pointer"
                          title="Masquer la colonne Finished"
                        >
                          <EyeOff size={14} />
                        </button>
                      )}
                      <button
                        onClick={() => openQuickAddForWorkflow(col.id)}
                        className="p-1 rounded-md text-[var(--text-muted)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-tertiary)] transition-colors cursor-pointer"
                        title={`Ajouter un ticket (#${col.id})`}
                      >
                        <Plus size={15} />
                      </button>
                    </div>
                  </div>

                  {/* Task Cards Column Body */}
                  <div className="flex-1 overflow-y-auto p-2.5 space-y-2.5">
                    {colTasks.map(task => (
                      <TaskCard
                        key={task.id}
                        task={task}
                        isDragging={draggingTaskId === task.id}
                        onDragStart={() => setDraggingTaskId(task.id)}
                      />
                    ))}

                    {colTasks.length === 0 && (
                      <div className="h-32 flex flex-col items-center justify-center text-center p-4 border border-dashed border-[var(--border-color)]/60 rounded-xl">
                        <p className="text-xs text-[var(--text-muted)]">
                          {t.board.emptyColumn}
                        </p>
                      </div>
                    )}
                  </div>
                </div>
              )
            })}

          {/* ========================================================= */}
          {/* RENDER MODE 2: OPERATIONAL STATUS COLUMNS                 */}
          {/* ========================================================= */}
          {boardGrouping === 'status' &&
            statusColumns.map(col => {
              const colTasks = byPriorityDesc(tasks.filter(t => isTaskInStatusColumn(t.status, col.id)))
              const isOver = dragOverColumn === col.id

              // Collapsed Done Column when hideDone is enabled
              if (col.id === 'finished' && hideDone) {
                return (
                  <div
                    key={col.id}
                    onDragOver={e => handleDragOver(e, col.id)}
                    onDragLeave={e => handleDragLeave(e, col.id)}
                    onDrop={e => handleDropStatus(e, col.id)}
                    onClick={toggleHideDone}
                    className={`w-14 shrink-0 flex flex-col items-center justify-between py-4 rounded-2xl bg-[var(--bg-secondary)]/60 border transition-all duration-200 cursor-pointer group hover:bg-[var(--bg-secondary)] select-none ${
                      isOver
                        ? 'border-emerald-500 ring-2 ring-emerald-500/30 bg-emerald-500/15'
                        : 'border-[var(--border-color)] hover:border-emerald-500/40'
                    }`}
                    title="Colonne Done masquée - Glissez une tâche ici pour la fermer, ou cliquez pour l'afficher"
                  >
                    <div className="flex flex-col items-center gap-1.5">
                      <CheckCircle2 size={18} className="text-emerald-400" />
                      <span className="text-[10px] font-mono font-bold px-1.5 py-0.2 rounded-full bg-emerald-500/20 text-emerald-300">
                        {colTasks.length}
                      </span>
                    </div>

                    <div className="flex-1 flex items-center justify-center my-4">
                      <span className="[writing-mode:vertical-lr] rotate-180 text-xs font-bold text-[var(--text-secondary)] group-hover:text-emerald-400 tracking-wider transition-colors">
                        {col.title} ({colTasks.length})
                      </span>
                    </div>

                    <div className="p-1.5 rounded-lg bg-[var(--bg-tertiary)] text-[var(--text-muted)] group-hover:text-emerald-400 transition-colors">
                      <Eye size={14} />
                    </div>
                  </div>
                )
              }

              return (
                <div
                  key={col.id}
                  onDragOver={e => handleDragOver(e, col.id)}
                  onDragLeave={e => handleDragLeave(e, col.id)}
                  onDrop={e => handleDropStatus(e, col.id)}
                  className={`kanban-column w-[320px] min-w-[290px] shrink-0 flex flex-col rounded-2xl bg-[var(--bg-secondary)]/70 border transition-all duration-200 ${
                    isOver
                      ? 'border-[var(--accent-color)] ring-2 ring-[var(--accent-glow)] bg-[var(--accent-light)]/20'
                      : 'border-[var(--border-color)]'
                  }`}
                >
                  {/* Column Header */}
                  <div className="flex items-center justify-between p-3 border-b border-[var(--border-color)]">
                    <div className="flex items-center gap-2">
                      <span>{col.icon}</span>
                      <h3 className="text-xs font-bold text-[var(--text-primary)] tracking-wide">
                        {col.title}
                      </h3>
                      <span className={`text-[10px] font-bold px-1.5 py-0.2 rounded border ${col.stageColor}`}>
                        {col.stageLabel}
                      </span>
                      <span className={`text-[11px] font-mono px-1.5 py-0.2 rounded-full font-bold ${col.badgeBg}`}>
                        {colTasks.length}
                      </span>
                    </div>

                    <div className="flex items-center gap-1">
                      {col.id === 'to_close' && (
                        <button
                          onClick={toggleHideDone}
                          className="p-1 rounded-md text-[var(--text-muted)] hover:text-emerald-400 hover:bg-[var(--bg-tertiary)] transition-colors"
                          title="Masquer la colonne Done"
                        >
                          <EyeOff size={14} />
                        </button>
                      )}
                      <button
                        onClick={() => openQuickAddForStatus(col.id)}
                        className="p-1 rounded-md text-[var(--text-muted)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-tertiary)] transition-colors"
                        title={`Ajouter une tâche (${col.title})`}
                      >
                        <Plus size={15} />
                      </button>
                    </div>
                  </div>

                  {/* Task Cards Column Body */}
                  <div className="flex-1 overflow-y-auto p-2.5 space-y-2.5">
                    {colTasks.map(task => (
                      <TaskCard
                        key={task.id}
                        task={task}
                        isDragging={draggingTaskId === task.id}
                        onDragStart={() => setDraggingTaskId(task.id)}
                      />
                    ))}

                    {colTasks.length === 0 && (
                      <div className="h-32 flex flex-col items-center justify-center text-center p-4 border border-dashed border-[var(--border-color)]/60 rounded-xl">
                        <p className="text-xs text-[var(--text-muted)]">
                          {t.board.emptyColumn}
                        </p>
                      </div>
                    )}
                  </div>
                </div>
              )
            })}
        </div>
      </div>
    </div>
  )
}
