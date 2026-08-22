import React, { useState } from 'react'
import {
  HelpCircle,
  FileCode,
  Flame,
  ShieldCheck,
  CheckCircle2,
  Plus,
  Eye,
  EyeOff
} from 'lucide-react'
import { useApp } from '../context/AppContext'
import { TaskCard } from './TaskCard'
import type { Status } from '../types'

interface ColumnConfig {
  id: Status
  title: string
  stageLabel: string
  stageColor: string
  icon: React.ReactNode
  color: string
  borderColor: string
  badgeBg: string
}

export const BoardView: React.FC = () => {
  const {
    tasks,
    moveTask,
    hideDone,
    toggleHideDone,
    setIsQuickAddOpen,
    setQuickAddInitialStatus,
    t,
  } = useApp()

  const [dragOverColumn, setDragOverColumn] = useState<Status | null>(null)
  const [draggingTaskId, setDraggingTaskId] = useState<string | null>(null)

  const isTaskInColumn = (taskStatus: Status, colId: Status) => {
    if (taskStatus === colId) return true
    if (colId === 'to_clarify' && taskStatus === 'backlog') return true
    if (colId === 'to_specify' && taskStatus === 'specified') return true
    if (colId === 'to_implement' && taskStatus === 'in_progress') return true
    if (colId === 'to_test' && taskStatus === 'to_validate') return true
    if (colId === 'to_close' && taskStatus === 'done') return true
    return false
  }

  const columns: ColumnConfig[] = [
    {
      id: 'to_clarify',
      title: t.status.to_clarify,
      stageLabel: 'New',
      stageColor: 'bg-cyan-500/15 text-cyan-400 border-cyan-500/30',
      icon: <HelpCircle size={16} className="text-cyan-400" />,
      color: 'text-cyan-400',
      borderColor: 'border-cyan-500/30',
      badgeBg: 'bg-cyan-500/20 text-cyan-300',
    },
    {
      id: 'to_specify',
      title: t.status.to_specify,
      stageLabel: 'Clarified',
      stageColor: 'bg-amber-500/15 text-amber-400 border-amber-500/30',
      icon: <FileCode size={16} className="text-amber-400" />,
      color: 'text-amber-400',
      borderColor: 'border-amber-500/30',
      badgeBg: 'bg-amber-500/20 text-amber-300',
    },
    {
      id: 'to_implement',
      title: t.status.to_implement,
      stageLabel: 'Specified',
      stageColor: 'bg-blue-500/15 text-blue-400 border-blue-500/30',
      icon: <Flame size={16} className="text-blue-400" />,
      color: 'text-blue-400',
      borderColor: 'border-blue-500/30',
      badgeBg: 'bg-blue-500/20 text-blue-300',
    },
    {
      id: 'to_test',
      title: t.status.to_test,
      stageLabel: 'Implemented',
      stageColor: 'bg-indigo-500/15 text-indigo-400 border-indigo-500/30',
      icon: <ShieldCheck size={16} className="text-indigo-400" />,
      color: 'text-indigo-400',
      borderColor: 'border-indigo-500/30',
      badgeBg: 'bg-indigo-500/20 text-indigo-300',
    },
    {
      id: 'to_close',
      title: t.status.to_close,
      stageLabel: 'Reviewed',
      stageColor: 'bg-emerald-500/15 text-emerald-400 border-emerald-500/30',
      icon: <CheckCircle2 size={16} className="text-emerald-400" />,
      color: 'text-emerald-400',
      borderColor: 'border-emerald-500/30',
      badgeBg: 'bg-emerald-500/20 text-emerald-300',
    },
  ]

  const handleDragOver = (e: React.DragEvent, status: Status) => {
    e.preventDefault()
    e.dataTransfer.dropEffect = 'move'
    if (dragOverColumn !== status) {
      setDragOverColumn(status)
    }
  }

  const handleDragLeave = (e: React.DragEvent, status: Status) => {
    e.preventDefault()
    const rect = (e.currentTarget as HTMLElement).getBoundingClientRect()
    if (
      e.clientX < rect.left ||
      e.clientX >= rect.right ||
      e.clientY < rect.top ||
      e.clientY >= rect.bottom
    ) {
      if (dragOverColumn === status) {
        setDragOverColumn(null)
      }
    }
  }

  const handleDrop = async (e: React.DragEvent, targetStatus: Status) => {
    e.preventDefault()
    setDragOverColumn(null)
    const taskId = e.dataTransfer.getData('text/plain') || draggingTaskId
    setDraggingTaskId(null)

    if (!taskId) return

    const task = tasks.find(t => t.id === taskId)
    if (!task || isTaskInColumn(task.status, targetStatus)) return

    const targetTasks = tasks.filter(t => isTaskInColumn(t.status, targetStatus))
    const newPos = targetTasks.length

    await moveTask(taskId, targetStatus, newPos)
  }

  const openQuickAddForColumn = (status: Status) => {
    setQuickAddInitialStatus(status)
    setIsQuickAddOpen(true)
  }

  return (
    <div className="flex-1 overflow-x-auto overflow-y-hidden p-4 bg-[var(--bg-primary)]">
      <div className="flex gap-4 h-full min-w-max pb-2">
        {columns.map(col => {
          const colTasks = tasks.filter(t => isTaskInColumn(t.status, col.id))
          const isOver = dragOverColumn === col.id

          // Collapsed Done Column
          if ((col.id === 'to_close' || col.id === 'done') && hideDone) {
            return (
              <div
                key={col.id}
                onDragOver={e => handleDragOver(e, col.id)}
                onDragLeave={e => handleDragLeave(e, col.id)}
                onDrop={e => handleDrop(e, col.id)}
                onClick={toggleHideDone}
                className={`w-14 shrink-0 flex flex-col items-center justify-between py-4 rounded-2xl bg-[var(--bg-secondary)]/60 border transition-all duration-200 cursor-pointer group hover:bg-[var(--bg-secondary)] select-none ${
                  isOver
                    ? 'border-emerald-500 ring-2 ring-emerald-500/30 bg-emerald-500/15'
                    : 'border-[var(--border-color)] hover:border-emerald-500/40'
                }`}
                title="Colonne À fermer masquée - Glissez une tâche ici pour la fermer, ou cliquez pour l'afficher"
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

                <div className="p-1.5 rounded-lg bg-[var(--bg-tertiary)] text-[var(--text-muted)] group-hover:text-emerald-400 transition-colors" title="Afficher la colonne">
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
              onDrop={e => handleDrop(e, col.id)}
              className={`kanban-column shrink-0 flex flex-col rounded-2xl bg-[var(--bg-secondary)]/70 border transition-all duration-200 ${
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
                  {col.id === 'done' && (
                    <button
                      onClick={toggleHideDone}
                      className="p-1 rounded-md text-[var(--text-muted)] hover:text-emerald-400 hover:bg-[var(--bg-tertiary)] transition-colors"
                      title={t.header.hideDone}
                    >
                      <EyeOff size={14} />
                    </button>
                  )}
                  <button
                    onClick={() => openQuickAddForColumn(col.id)}
                    className="p-1 rounded-md text-[var(--text-muted)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-tertiary)] transition-colors"
                    title={`${t.board.addTask} (${col.title})`}
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
  )
}
