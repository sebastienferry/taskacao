import React, { useState, useMemo, useRef } from 'react'
import {
  CalendarDays,
  Clock,
  Plus,
  RefreshCw,
  CheckCircle2,
  Trash2,
  Edit2,
  Layers,
  Search,
  Check,
  X,
  MessageSquare,
  LayoutGrid,
  Tag,
  Maximize2,
  Minimize2,
  CheckSquare,
  Square,
  GripVertical,
} from 'lucide-react'
import { useApp } from '../context/AppContext'
import {
  calculateSprintDates,
  generateDefaultSprints,
  formatDateFR,
  formatDateISO,
  getMonday,
  getSprintRelativeInfo,
} from '../lib/sprints'
import type { Task, TrackerSprint, WorkflowStage } from '../types'
import { resolveTaskStage } from '../lib/workflow'
import { Avatar } from './Avatar'

const DRAG_TASK_ID = 'application/x-taskflow-task-id'
const DRAG_TASK_IDS = 'application/x-taskflow-task-ids'

export const SprintTimelineView: React.FC = () => {
  const {
    tasks,
    currentProject,
    updateProject,
    setTaskSprint,
    setTasksSprint,
    setSelectedTask,
    setChatTask,
    addToast,
  } = useApp()

  // Project sprints (or defaults if none yet)
  const sprints: TrackerSprint[] = useMemo(() => {
    if (currentProject?.sprints && currentProject.sprints.length > 0) {
      return currentProject.sprints
    }
    return generateDefaultSprints(4)
  }, [currentProject?.sprints])

  // Configuration state
  const [durationDays, setDurationDays] = useState<number>(14)
  const [startDateStr, setStartDateStr] = useState<string>(() => {
    if (sprints.length > 0 && sprints[0].startDate) {
      return sprints[0].startDate
    }
    return formatDateISO(getMonday(new Date()))
  })

  const [editingSprintIndex, setEditingSprintIndex] = useState<number | null>(null)
  const [editSprintName, setEditSprintName] = useState<string>('')
  const [backlogSearch, setBacklogSearch] = useState<string>('')
  const [isBacklogOpen, setIsBacklogOpen] = useState<boolean>(true)
  const [dragOverSprint, setDragOverSprint] = useState<string | null>(null)

  // Toggle Display Mode: 'cards' vs 'chips'
  const [displayMode, setDisplayMode] = useState<'cards' | 'chips'>(() => {
    try {
      const stored = localStorage.getItem('taskflow_sprint_display_mode')
      return stored === 'chips' ? 'chips' : 'cards'
    } catch {
      return 'cards'
    }
  })

  const handleSetDisplayMode = (mode: 'cards' | 'chips') => {
    setDisplayMode(mode)
    try {
      localStorage.setItem('taskflow_sprint_display_mode', mode)
    } catch {}
  }

  // Backlog Width Resizing State
  const [backlogWidth, setBacklogWidth] = useState<number>(() => {
    try {
      const stored = localStorage.getItem('taskflow_sprint_backlog_width')
      const parsed = Number(stored)
      return parsed >= 280 && parsed <= 900 ? parsed : 360
    } catch {
      return 360
    }
  })
  const isResizingBacklog = useRef(false)

  const handleStartResizeBacklog = (e: React.PointerEvent) => {
    e.preventDefault()
    isResizingBacklog.current = true

    const startX = e.clientX
    const startWidth = backlogWidth

    const onPointerMove = (moveEvent: PointerEvent) => {
      if (!isResizingBacklog.current) return
      // Moving left increases width because drawer is on the right
      const deltaX = startX - moveEvent.clientX
      const newWidth = Math.min(Math.max(280, startWidth + deltaX), 850)
      setBacklogWidth(newWidth)
    }

    const onPointerUp = () => {
      isResizingBacklog.current = false
      window.removeEventListener('pointermove', onPointerMove)
      window.removeEventListener('pointerup', onPointerUp)
      try {
        localStorage.setItem('taskflow_sprint_backlog_width', String(backlogWidth))
      } catch {}
    }

    window.addEventListener('pointermove', onPointerMove)
    window.addEventListener('pointerup', onPointerUp)
  }

  // Multi-Selection State
  const [checkedTaskIds, setCheckedTaskIds] = useState<Record<string, boolean>>({})
  const [batchTargetSprint, setBatchTargetSprint] = useState<string>('')
  const [batchBusy, setBatchBusy] = useState(false)

  const getTaskStageBadge = (task: Task) => {
    const stage = resolveTaskStage(task, currentProject)
    const stageInfo: Record<WorkflowStage, { label: string; style: string }> = {
      new: { label: '#new', style: 'bg-cyan-500/15 text-cyan-400 border-cyan-500/30' },
      clarified: { label: '#clarified', style: 'bg-amber-500/15 text-amber-400 border-amber-500/30' },
      specified: { label: '#specified', style: 'bg-blue-500/15 text-blue-400 border-blue-500/30' },
      implemented: { label: '#implemented', style: 'bg-indigo-500/15 text-indigo-400 border-indigo-500/30' },
      reviewed: { label: '#reviewed', style: 'bg-purple-500/15 text-purple-400 border-purple-500/30' },
      finished: { label: '#finished', style: 'bg-emerald-500/15 text-emerald-400 border-emerald-500/30' },
    }
    const info = stageInfo[stage] || stageInfo.new
    return (
      <span className={`px-1.5 py-0.2 rounded font-mono font-bold text-[8.5px] border shrink-0 ${info.style}`} title={`Étape : ${info.label}`}>
        {info.label}
      </span>
    )
  }

  // Map tasks to sprints
  const tasksBySprint = useMemo(() => {
    const map = new Map<string, Task[]>()
    sprints.forEach(sp => map.set(sp.name.toLowerCase().trim(), []))

    tasks.forEach(task => {
      const spName = (task.sprint || '').toLowerCase().trim()
      if (spName && map.has(spName)) {
        map.get(spName)!.push(task)
      }
    })
    return map
  }, [tasks, sprints])

  const [showDoneInBacklog, setShowDoneInBacklog] = useState<boolean>(false)

  // Finished unscheduled count
  const finishedUnscheduledCount = useMemo(() => {
    return tasks.filter(t => {
      const sp = (t.sprint || '').trim()
      const hasNoSprint = !sp || !sprints.some(s => s.name.toLowerCase() === sp.toLowerCase())
      if (!hasNoSprint) return false
      return resolveTaskStage(t, currentProject) === 'finished' || t.status === 'finished' || t.status === 'done'
    }).length
  }, [tasks, sprints, currentProject])

  // Unscheduled tasks (Backlog)
  const unscheduledTasks = useMemo(() => {
    const q = backlogSearch.toLowerCase().trim()
    return tasks.filter(t => {
      const sp = (t.sprint || '').trim()
      const hasNoSprint = !sp || !sprints.some(s => s.name.toLowerCase() === sp.toLowerCase())
      if (!hasNoSprint) return false

      const isFinished = resolveTaskStage(t, currentProject) === 'finished' || t.status === 'finished' || t.status === 'done'
      if (isFinished && !showDoneInBacklog) return false

      if (!q) return true
      return (
        t.key.toLowerCase().includes(q) ||
        t.title.toLowerCase().includes(q) ||
        (t.parentTitle && t.parentTitle.toLowerCase().includes(q)) ||
        (t.assignee && t.assignee.toLowerCase().includes(q))
      )
    })
  }, [tasks, sprints, backlogSearch, currentProject, showDoneInBacklog])

  const selectedBacklogList = useMemo(
    () => unscheduledTasks.filter(t => checkedTaskIds[t.id]),
    [unscheduledTasks, checkedTaskIds]
  )

  const toggleTaskCheck = (taskId: string) => {
    setCheckedTaskIds(prev => ({
      ...prev,
      [taskId]: !prev[taskId],
    }))
  }

  const handleSelectAllBacklog = () => {
    const allSelected = unscheduledTasks.length > 0 && unscheduledTasks.every(t => checkedTaskIds[t.id])
    if (allSelected) {
      setCheckedTaskIds({})
    } else {
      const next: Record<string, boolean> = {}
      unscheduledTasks.forEach(t => {
        next[t.id] = true
      })
      setCheckedTaskIds(next)
    }
  }

  // Save updated sprints to project
  const saveSprints = async (newSprints: TrackerSprint[]) => {
    if (!currentProject?.id) return
    const updated = await updateProject(currentProject.id, { sprints: newSprints })
    if (updated) {
      addToast({
        type: 'success',
        title: 'Sprints mis à jour',
        description: `${newSprints.length} sprint(s) enregistrés pour ce projet.`,
      })
    }
  }

  // Recalculate all dates automatically
  const handleRecalculateAll = async () => {
    const updated = calculateSprintDates(sprints, startDateStr, durationDays)
    await saveSprints(updated)
  }

  // Add next consecutive sprint
  const handleAddSprint = async () => {
    let nextStart = new Date(startDateStr)
    if (sprints.length > 0) {
      const last = sprints[sprints.length - 1]
      if (last.endDate) {
        const lastEnd = new Date(last.endDate)
        lastEnd.setDate(lastEnd.getDate() + 1)
        nextStart = lastEnd
      }
    }

    const newIndex = sprints.length + 1
    const newSprintObj: TrackerSprint = {
      id: `sprint-${newIndex}`,
      name: `Sprint ${newIndex}`,
      state: 'future',
      startDate: formatDateISO(nextStart),
      endDate: formatDateISO(
        new Date(nextStart.getTime() + (durationDays - 1) * 24 * 60 * 60 * 1000)
      ),
    }

    const updated = [...sprints, newSprintObj]
    await saveSprints(updated)
  }

  // Delete a sprint
  const handleDeleteSprint = async (index: number) => {
    const sprintToDelete = sprints[index]
    if (!window.confirm(`Supprimer le ${sprintToDelete.name} ? Les tâches associées seront renvoyées au backlog.`)) {
      return
    }
    const updated = sprints.filter((_, i) => i !== index)
    await saveSprints(updated)
  }

  // Rename a sprint
  const handleStartRename = (index: number, currentName: string) => {
    setEditingSprintIndex(index)
    setEditSprintName(currentName)
  }

  const handleSaveRename = async (index: number) => {
    if (!editSprintName.trim()) {
      setEditingSprintIndex(null)
      return
    }
    const oldName = sprints[index].name
    const newName = editSprintName.trim()

    const updated = sprints.map((sp, i) => (i === index ? { ...sp, name: newName } : sp))
    setEditingSprintIndex(null)
    await saveSprints(updated)

    // Re-link tasks to new name
    const tasksToUpdate = tasks.filter(
      t => (t.sprint || '').toLowerCase() === oldName.toLowerCase()
    )
    for (const t of tasksToUpdate) {
      await setTaskSprint(t.id, newName, newName)
    }
  }

  // Drag & Drop Handlers with Multi-selection support
  const handleDragStart = (e: React.DragEvent, taskId: string) => {
    const isTargetSelected = Boolean(checkedTaskIds[taskId])
    const idsToDrag = isTargetSelected
      ? Object.keys(checkedTaskIds).filter(id => checkedTaskIds[id])
      : [taskId]

    e.dataTransfer.setData(DRAG_TASK_IDS, JSON.stringify(idsToDrag))
    e.dataTransfer.setData(DRAG_TASK_ID, taskId)
    e.dataTransfer.effectAllowed = 'move'
  }

  const handleDragOver = (e: React.DragEvent, sprintName: string) => {
    e.preventDefault()
    e.dataTransfer.dropEffect = 'move'
    setDragOverSprint(sprintName)
  }

  const handleDropOnSprint = async (e: React.DragEvent, sprintName: string) => {
    e.preventDefault()
    setDragOverSprint(null)

    const idsJson = e.dataTransfer.getData(DRAG_TASK_IDS)
    let taskIds: string[] = []
    if (idsJson) {
      try {
        taskIds = JSON.parse(idsJson)
      } catch {}
    }
    if (taskIds.length === 0) {
      const singleId = e.dataTransfer.getData(DRAG_TASK_ID)
      if (singleId) taskIds = [singleId]
    }
    if (taskIds.length === 0) return

    if (currentProject?.id && taskIds.length > 1) {
      await setTasksSprint(currentProject.id, taskIds, sprintName, sprintName)
      addToast({
        type: 'success',
        title: 'Tâches planifiées',
        description: `${taskIds.length} tâches déplacées dans le ${sprintName}.`,
      })
    } else {
      for (const id of taskIds) {
        await setTaskSprint(id, sprintName, sprintName)
      }
    }

    setCheckedTaskIds({})
  }

  const handleRemoveTaskFromSprint = async (taskId: string) => {
    await setTaskSprint(taskId, '', '')
  }

  // Batch assign from Backlog
  const handleApplyBatchSprint = async () => {
    const ids = Object.keys(checkedTaskIds).filter(id => checkedTaskIds[id])
    if (ids.length === 0 || !batchTargetSprint) return

    setBatchBusy(true)
    if (currentProject?.id) {
      await setTasksSprint(currentProject.id, ids, batchTargetSprint, batchTargetSprint)
      addToast({
        type: 'success',
        title: 'Tâches affectées',
        description: `${ids.length} tâche(s) affectée(s) au ${batchTargetSprint}.`,
      })
    } else {
      for (const id of ids) {
        await setTaskSprint(id, batchTargetSprint, batchTargetSprint)
      }
    }
    setCheckedTaskIds({})
    setBatchTargetSprint('')
    setBatchBusy(false)
  }

  return (
    <div className="flex-1 flex flex-col h-full bg-[var(--bg-primary)] overflow-hidden">
      {/* Top Controls & Auto-Calculation Bar */}
      <div className="border-b border-[var(--border-color)] bg-[var(--bg-secondary)] px-4 py-2.5 shrink-0">
        <div className="flex flex-col lg:flex-row items-start lg:items-center justify-between gap-3">
          {/* Title & Durations */}
          <div className="flex items-center gap-3 flex-wrap">
            <div className="flex items-center gap-2">
              <div className="p-1.5 rounded-lg bg-[var(--accent-light)] text-[var(--accent-color)] border border-[var(--accent-color)]/30">
                <CalendarDays size={16} />
              </div>
              <div>
                <h2 className="text-xs font-bold uppercase tracking-wider text-[var(--text-primary)]">
                  SPRINTS
                </h2>
                <span className="text-[10px] text-[var(--text-muted)]">
                  Découpage temporel & calcul automatique des cycles
                </span>
              </div>
            </div>

            <div className="h-4 w-px bg-[var(--border-color)] hidden sm:block mx-1" />

            {/* Sprint Duration Selector */}
            <div className="flex items-center gap-1 bg-[var(--bg-tertiary)] p-0.5 rounded-lg border border-[var(--border-color)] text-xs">
              <span className="text-[10px] text-[var(--text-muted)] font-medium px-2 flex items-center gap-1">
                <Clock size={11} /> Durée :
              </span>
              {[
                { label: '1 sem (7j)', days: 7 },
                { label: '2 sem (14j)', days: 14 },
                { label: '3 sem (21j)', days: 21 },
                { label: '4 sem (28j)', days: 28 },
              ].map(d => (
                <button
                  key={d.days}
                  type="button"
                  onClick={() => {
                    setDurationDays(d.days)
                    const updated = calculateSprintDates(sprints, startDateStr, d.days)
                    saveSprints(updated)
                  }}
                  className={`px-2.5 py-1 rounded-md text-[11px] font-medium transition-all cursor-pointer ${
                    durationDays === d.days
                      ? 'bg-[var(--accent-color)] text-white font-bold shadow-xs'
                      : 'text-[var(--text-secondary)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-secondary)]'
                  }`}
                >
                  {d.label}
                </button>
              ))}
            </div>

            {/* Start Date of Sprint 1 */}
            <div className="flex items-center gap-1.5 text-xs bg-[var(--bg-tertiary)] px-2.5 py-1 rounded-lg border border-[var(--border-color)]">
              <span className="text-[10px] text-[var(--text-muted)] font-medium">Début S1 :</span>
              <input
                type="date"
                value={startDateStr}
                onChange={e => {
                  setStartDateStr(e.target.value)
                  const updated = calculateSprintDates(sprints, e.target.value, durationDays)
                  saveSprints(updated)
                }}
                className="bg-transparent text-[11px] text-[var(--text-primary)] font-mono focus:outline-none cursor-pointer"
              />
            </div>
          </div>

          {/* Action Buttons & Display Mode Switcher */}
          <div className="flex items-center gap-2 self-end lg:self-auto flex-wrap">
            {/* Display Mode: Cards vs Chips */}
            <div className="flex items-center gap-0.5 p-0.5 rounded-lg bg-[var(--bg-tertiary)] border border-[var(--border-color)] text-xs">
              <button
                type="button"
                onClick={() => handleSetDisplayMode('cards')}
                className={`px-2 py-1 rounded flex items-center gap-1 text-[11px] font-semibold transition-all cursor-pointer ${
                  displayMode === 'cards'
                    ? 'bg-[var(--accent-color)] text-white shadow-xs font-bold'
                    : 'text-[var(--text-secondary)] hover:text-[var(--text-primary)]'
                }`}
                title="Affichage sous forme de cartes complètes"
              >
                <LayoutGrid size={12} />
                <span className="hidden sm:inline">Cartes</span>
              </button>
              <button
                type="button"
                onClick={() => handleSetDisplayMode('chips')}
                className={`px-2 py-1 rounded flex items-center gap-1 text-[11px] font-semibold transition-all cursor-pointer ${
                  displayMode === 'chips'
                    ? 'bg-[var(--accent-color)] text-white shadow-xs font-bold'
                    : 'text-[var(--text-secondary)] hover:text-[var(--text-primary)]'
                }`}
                title="Affichage sous forme de chips compactes"
              >
                <Tag size={12} />
                <span className="hidden sm:inline">Chips</span>
              </button>
            </div>

            <button
              type="button"
              onClick={handleRecalculateAll}
              className="flex items-center gap-1.5 px-2.5 py-1 rounded-lg text-xs font-semibold bg-[var(--bg-tertiary)] hover:bg-[var(--bg-primary)] border border-[var(--border-color)] text-[var(--text-primary)] transition-all cursor-pointer shadow-xs"
              title="Réaligner automatiquement les dates de tous les sprints"
            >
              <RefreshCw size={12} className="text-cyan-400" />
              <span>Recalculer</span>
            </button>

            <button
              type="button"
              onClick={handleAddSprint}
              className="flex items-center gap-1.5 px-3 py-1 rounded-lg text-xs font-bold bg-[var(--accent-color)] hover:opacity-90 text-white transition-all cursor-pointer shadow-xs"
              title="Ajouter un nouveau sprint consécutif"
            >
              <Plus size={13} />
              <span>+ Sprint</span>
            </button>

            <button
              type="button"
              onClick={() => setIsBacklogOpen(!isBacklogOpen)}
              className={`flex items-center gap-1.5 px-3 py-1 rounded-lg text-xs font-semibold border transition-all cursor-pointer ${
                isBacklogOpen
                  ? 'bg-[var(--accent-light)] accent-text border-[var(--accent-color)]/40 shadow-xs'
                  : 'bg-[var(--bg-tertiary)] text-[var(--text-muted)] border-[var(--border-color)] hover:text-[var(--text-primary)]'
              }`}
              title={
                isBacklogOpen
                  ? 'Masquer le Backlog (Vue synthétique compacte sans Drag & Drop)'
                  : 'Afficher le Backlog (Mode planification interactive avec Drag & Drop)'
              }
            >
              <Layers size={13} />
              <span>
                {isBacklogOpen ? 'Backlog (Glisser-Déposer actif)' : `Afficher Backlog (${unscheduledTasks.length})`}
              </span>
            </button>
          </div>
        </div>
      </div>

      {/* Main Timeline Workspace */}
      <div className="flex-1 flex min-h-0 overflow-hidden relative">
        {/* Vertical Chronological Timeline */}
        <div className={`flex-1 overflow-y-auto ${isBacklogOpen ? 'p-6 space-y-5' : 'p-4 space-y-3'}`}>
          <div className={`${isBacklogOpen ? 'max-w-4xl space-y-5' : 'max-w-5xl space-y-3'} mx-auto relative`}>
            {/* Continuous Vertical Timeline Line */}
            <div
              className={`absolute ${isBacklogOpen ? 'left-6 top-6 bottom-6' : 'left-4 top-4 bottom-4'} w-0.5 bg-gradient-to-b from-indigo-500/40 via-cyan-500/40 to-emerald-500/40 hidden md:block`}
            />

            {sprints.map((sprint, index) => {
              const sprintKey = sprint.name.toLowerCase().trim()
              const sprintTasks = tasksBySprint.get(sprintKey) || []
              const rel = getSprintRelativeInfo(sprint)
              const isOver = isBacklogOpen && dragOverSprint === sprint.name

              // Completed stats
              const doneTasks = sprintTasks.filter(
                t => t.status === 'finished' || t.status === 'done'
              )
              const progressPct =
                sprintTasks.length > 0 ? Math.round((doneTasks.length / sprintTasks.length) * 100) : 0

              return (
                <div
                  key={sprint.id || sprint.name}
                  className={`relative flex flex-col md:flex-row items-start ${isBacklogOpen ? 'gap-4' : 'gap-3'}`}
                >
                  {/* Timeline Node Icon (Desktop) */}
                  <div
                    className={`hidden md:flex items-center justify-center rounded-2xl bg-[var(--bg-secondary)] border-2 border-[var(--border-color)] z-10 shrink-0 shadow-md ${
                      isBacklogOpen ? 'w-12 h-12' : 'w-8 h-8 rounded-xl'
                    }`}
                  >
                    {rel.type === 'current' ? (
                      <span className="relative flex h-3.5 w-3.5">
                        <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
                        <span className="relative inline-flex rounded-full h-3.5 w-3.5 bg-emerald-500"></span>
                      </span>
                    ) : rel.type === 'past' ? (
                      <CheckCircle2 size={isBacklogOpen ? 20 : 15} className="text-slate-400" />
                    ) : (
                      <CalendarDays size={isBacklogOpen ? 18 : 14} className="text-cyan-400" />
                    )}
                  </div>

                  {/* Sprint Container Card */}
                  <div
                    onDragOver={isBacklogOpen ? e => handleDragOver(e, sprint.name) : undefined}
                    onDragLeave={isBacklogOpen ? () => setDragOverSprint(null) : undefined}
                    onDrop={isBacklogOpen ? e => handleDropOnSprint(e, sprint.name) : undefined}
                    className={`flex-1 w-full border bg-[var(--bg-secondary)]/90 shadow-sm transition-all duration-200 overflow-hidden ${
                      isBacklogOpen ? 'rounded-2xl' : 'rounded-xl'
                    } ${
                      isOver
                        ? 'border-[var(--accent-color)] ring-2 ring-[var(--accent-glow)] bg-[var(--accent-light)]/20'
                        : rel.type === 'current'
                        ? 'border-emerald-500/50 shadow-[0_0_15px_rgba(16,185,129,0.1)]'
                        : 'border-[var(--border-color)] hover:border-[var(--border-color)]/80'
                    }`}
                  >
                    {/* Sprint Header */}
                    <div
                      className={`border-b border-[var(--border-color)] flex flex-col sm:flex-row sm:items-center justify-between gap-2 bg-[var(--bg-tertiary)]/30 ${
                        isBacklogOpen ? 'p-3.5' : 'px-3 py-2'
                      }`}
                    >
                      <div className="flex items-center gap-2">
                        {editingSprintIndex === index ? (
                          <div className="flex items-center gap-1.5">
                            <input
                              type="text"
                              value={editSprintName}
                              onChange={e => setEditSprintName(e.target.value)}
                              className="px-2 py-0.5 text-xs font-bold rounded bg-[var(--bg-primary)] border border-[var(--accent-color)] text-[var(--text-primary)]"
                              autoFocus
                              onKeyDown={e => {
                                if (e.key === 'Enter') handleSaveRename(index)
                                if (e.key === 'Escape') setEditingSprintIndex(null)
                              }}
                            />
                            <button
                              onClick={() => handleSaveRename(index)}
                              className="p-1 text-emerald-400 hover:text-emerald-300"
                            >
                              <Check size={13} />
                            </button>
                            <button
                              onClick={() => setEditingSprintIndex(null)}
                              className="p-1 text-rose-400 hover:text-rose-300"
                            >
                              <X size={13} />
                            </button>
                          </div>
                        ) : (
                          <div className="flex items-center gap-1.5">
                            <h3
                              className={`${
                                isBacklogOpen ? 'text-sm' : 'text-xs'
                              } font-extrabold text-[var(--text-primary)]`}
                            >
                              {sprint.name}
                            </h3>
                            <button
                              onClick={() => handleStartRename(index, sprint.name)}
                              className="opacity-40 hover:opacity-100 text-[var(--text-muted)] p-0.5 transition-opacity cursor-pointer"
                              title="Renommer le sprint"
                            >
                              <Edit2 size={10} />
                            </button>
                          </div>
                        )}

                        <span
                          className={`text-[9px] font-bold px-2 py-0.5 rounded-full border ${
                            rel.type === 'current'
                              ? 'bg-emerald-500/15 text-emerald-400 border-emerald-500/30 animate-pulse'
                              : rel.type === 'past'
                              ? 'bg-slate-500/15 text-slate-400 border-slate-500/30'
                              : 'bg-cyan-500/15 text-cyan-400 border-cyan-500/30'
                          }`}
                        >
                          {rel.label}
                        </span>
                      </div>

                      {/* Dates & Quick Stats */}
                      <div className="flex items-center gap-2.5 text-xs">
                        <div className="flex items-center gap-1 text-[var(--text-muted)] font-mono text-[10.5px]">
                          <CalendarDays size={11} className="text-[var(--accent-color)]" />
                          <span>
                            {formatDateFR(sprint.startDate)} ➔ {formatDateFR(sprint.endDate)}
                          </span>
                        </div>

                        {sprintTasks.length > 0 && (
                          <div className="flex items-center gap-1.5 pl-2.5 border-l border-[var(--border-color)]">
                            <div className="w-14 h-1.5 rounded-full bg-[var(--bg-primary)] overflow-hidden border border-[var(--border-color)]">
                              <div
                                className="h-full bg-emerald-500 transition-all duration-300"
                                style={{ width: `${progressPct}%` }}
                              />
                            </div>
                            <span className="text-[9.5px] font-mono font-bold text-[var(--text-secondary)]">
                              {progressPct}% ({doneTasks.length}/{sprintTasks.length})
                            </span>
                          </div>
                        )}

                        <button
                          type="button"
                          onClick={() => handleDeleteSprint(index)}
                          className="p-1 rounded text-[var(--text-muted)] hover:text-rose-400 transition-colors ml-0.5 cursor-pointer"
                          title="Supprimer ce sprint"
                        >
                          <Trash2 size={12} />
                        </button>
                      </div>
                    </div>

                    {/* Mode 1: Compact View (Backlog Hidden, NO Drag & Drop) */}
                    {!isBacklogOpen ? (
                      <div className="p-2.5">
                        {sprintTasks.length === 0 ? (
                          <p className="text-[11px] text-[var(--text-muted)] italic px-1">
                            0 tâche planifiée dans ce sprint.
                          </p>
                        ) : displayMode === 'chips' ? (
                          <div className="flex flex-wrap gap-1.5">
                            {sprintTasks.map(task => (
                              <div
                                key={task.id}
                                onClick={() => setSelectedTask(task)}
                                className="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-md bg-[var(--bg-primary)] border border-[var(--border-color)] hover:border-[var(--accent-color)] text-xs cursor-pointer transition-colors group shadow-2xs"
                                title={`${task.key}: ${task.title}`}
                              >
                                <span className="font-mono font-bold text-[10px] text-[var(--accent-color)]">
                                  {task.key}
                                </span>
                                <span className="truncate max-w-[160px] text-[11px] font-medium text-[var(--text-primary)]">
                                  {task.title}
                                </span>
                                {getTaskStageBadge(task)}
                                {task.assignee && (
                                  <Avatar name={task.assignee} url={task.assigneeAvatar} size={14} />
                                )}
                              </div>
                            ))}
                          </div>
                        ) : (
                          <div className="flex flex-col divide-y divide-[var(--border-color)]/50">
                            {sprintTasks.map(task => (
                              <div
                                key={task.id}
                                onClick={() => setSelectedTask(task)}
                                className="flex items-center justify-between gap-2 py-1 px-1.5 rounded hover:bg-[var(--bg-tertiary)]/60 cursor-pointer transition-colors group"
                              >
                                <div className="flex items-center gap-2 min-w-0">
                                  <span className="text-[10px] font-mono font-bold text-[var(--accent-color)] shrink-0">
                                    {task.key}
                                  </span>
                                  {task.parentTitle && (
                                    <span
                                      className="text-[9px] px-1.5 py-0.2 rounded font-bold bg-[var(--bg-tertiary)] text-[var(--text-muted)] truncate max-w-[100px]"
                                      title={task.parentTitle}
                                    >
                                      {task.parentTitle}
                                    </span>
                                  )}
                                  <span className="text-xs text-[var(--text-primary)] truncate max-w-[360px] font-medium">
                                    {task.title}
                                  </span>
                                </div>

                                <div className="flex items-center gap-2 shrink-0">
                                  {getTaskStageBadge(task)}
                                  {task.assignee && (
                                    <Avatar name={task.assignee} url={task.assigneeAvatar} size={15} />
                                  )}
                                  <button
                                    type="button"
                                    onClick={e => {
                                      e.stopPropagation()
                                      setChatTask(task)
                                    }}
                                    className="text-[var(--text-muted)] hover:text-cyan-300 p-0.5 transition-colors opacity-0 group-hover:opacity-100"
                                    title="Discuter avec l'agent IA"
                                  >
                                    <MessageSquare size={11} />
                                  </button>
                                </div>
                              </div>
                            ))}
                          </div>
                        )}
                      </div>
                    ) : (
                      /* Mode 2: Large Planning View (Backlog Open, ACTIVE Drag & Drop) */
                      <div className="p-3.5 space-y-2">
                        {sprintTasks.length === 0 ? (
                          <div className="p-5 rounded-xl border-2 border-dashed border-[var(--border-color)]/70 text-center text-xs text-[var(--text-muted)] bg-[var(--bg-primary)]/40 hover:bg-[var(--bg-primary)]/60 transition-colors">
                            <p className="font-medium">Aucune tâche dans ce sprint</p>
                            <p className="text-[10px] opacity-70 mt-0.5">
                              Glissez des tâches depuis le Backlog ou un autre sprint ici.
                            </p>
                          </div>
                        ) : displayMode === 'chips' ? (
                          /* Chips display in planning mode */
                          <div className="flex flex-wrap gap-1.5">
                            {sprintTasks.map(task => (
                              <div
                                key={task.id}
                                draggable
                                onDragStart={e => handleDragStart(e, task.id)}
                                onClick={() => setSelectedTask(task)}
                                className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-lg border transition-all cursor-grab active:cursor-grabbing group text-xs select-none shadow-2xs ${
                                  checkedTaskIds[task.id]
                                    ? 'bg-[var(--accent-light)] border-[var(--accent-color)] text-[var(--accent-color)] font-semibold'
                                    : 'bg-[var(--bg-primary)] border-[var(--border-color)] hover:border-[var(--accent-color)]/60 text-[var(--text-primary)]'
                                }`}
                                title={`${task.key}: ${task.title}`}
                              >
                                <input
                                  type="checkbox"
                                  checked={Boolean(checkedTaskIds[task.id])}
                                  onChange={e => {
                                    e.stopPropagation()
                                    toggleTaskCheck(task.id)
                                  }}
                                  className="rounded text-[var(--accent-color)] w-3 h-3 cursor-pointer"
                                />
                                <span className="font-mono font-bold text-[10px] text-[var(--accent-color)] shrink-0">
                                  {task.key}
                                </span>
                                <span className="truncate max-w-[170px] text-[11px] font-medium">
                                  {task.title}
                                </span>
                                {getTaskStageBadge(task)}
                                {task.assignee && (
                                  <Avatar name={task.assignee} url={task.assigneeAvatar} size={14} />
                                )}
                                <button
                                  type="button"
                                  onClick={e => {
                                    e.stopPropagation()
                                    handleRemoveTaskFromSprint(task.id)
                                  }}
                                  className="p-0.5 rounded text-[var(--text-muted)] hover:text-rose-400 opacity-0 group-hover:opacity-100 transition-opacity cursor-pointer"
                                  title="Retirer du sprint"
                                >
                                  <X size={11} />
                                </button>
                              </div>
                            ))}
                          </div>
                        ) : (
                          /* Cards display in planning mode */
                          <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
                            {sprintTasks.map(task => (
                              <div
                                key={task.id}
                                draggable
                                onDragStart={e => handleDragStart(e, task.id)}
                                onClick={() => setSelectedTask(task)}
                                className={`p-2.5 rounded-xl border shadow-xs transition-all cursor-grab active:cursor-grabbing group space-y-1.5 ${
                                  checkedTaskIds[task.id]
                                    ? 'bg-[var(--accent-light)]/20 border-[var(--accent-color)]'
                                    : 'bg-[var(--bg-primary)] border-[var(--border-color)] hover:border-[var(--accent-color)]/50'
                                }`}
                              >
                                <div className="flex items-center justify-between gap-2">
                                  <div className="flex items-center gap-1.5 min-w-0">
                                    <input
                                      type="checkbox"
                                      checked={Boolean(checkedTaskIds[task.id])}
                                      onChange={e => {
                                        e.stopPropagation()
                                        toggleTaskCheck(task.id)
                                      }}
                                      className="rounded text-[var(--accent-color)] w-3 h-3 cursor-pointer"
                                    />
                                    <span className="text-[10px] font-mono font-bold text-[var(--accent-color)] shrink-0">
                                      {task.key}
                                    </span>
                                    {task.parentTitle && (
                                      <span
                                        className="text-[9px] px-1.5 py-0.2 rounded font-bold bg-[var(--bg-tertiary)] text-[var(--text-muted)] truncate max-w-[110px]"
                                        title={task.parentTitle}
                                      >
                                        {task.parentTitle}
                                      </span>
                                    )}
                                  </div>

                                  <div className="flex items-center gap-1 shrink-0">
                                    {task.assignee && (
                                      <Avatar name={task.assignee} url={task.assigneeAvatar} size={16} />
                                    )}
                                    <button
                                      type="button"
                                      onClick={e => {
                                        e.stopPropagation()
                                        handleRemoveTaskFromSprint(task.id)
                                      }}
                                      className="p-1 rounded text-[var(--text-muted)] hover:text-rose-400 hover:bg-rose-500/10 opacity-0 group-hover:opacity-100 transition-opacity cursor-pointer"
                                      title="Retirer du sprint (renvoyer au backlog)"
                                    >
                                      <X size={11} />
                                    </button>
                                  </div>
                                </div>

                                <h4 className="text-xs font-semibold text-[var(--text-primary)] line-clamp-1 leading-snug">
                                  {task.title}
                                </h4>

                                <div className="flex items-center justify-between text-[10px] pt-1 border-t border-[var(--border-color)]/40 text-[var(--text-muted)]">
                                  {getTaskStageBadge(task)}

                                  <button
                                    type="button"
                                    onClick={e => {
                                      e.stopPropagation()
                                      setChatTask(task)
                                    }}
                                    className="text-[var(--text-muted)] hover:text-cyan-300 p-0.5 transition-colors cursor-pointer"
                                    title="Discuter avec l'agent IA"
                                  >
                                    <MessageSquare size={11} />
                                  </button>
                                </div>
                              </div>
                            ))}
                          </div>
                        )}
                      </div>
                    )}
                  </div>
                </div>
              )
            })}
          </div>
        </div>

        {/* Backlog / Unscheduled Drawer (Right Side with Resizer) */}
        {isBacklogOpen && (
          <div
            style={{ width: `${backlogWidth}px` }}
            className="border-l border-[var(--border-color)] bg-[var(--bg-secondary)] flex flex-col shrink-0 min-h-0 relative select-none"
          >
            {/* Left Edge Resizer Handle */}
            <div
              onPointerDown={handleStartResizeBacklog}
              className="absolute left-0 top-0 bottom-0 w-2 -ml-1 cursor-col-resize hover:bg-[var(--accent-color)]/50 active:bg-[var(--accent-color)] transition-colors z-30 flex items-center justify-center group"
              title="Glisser pour redimensionner le panneau Backlog"
            >
              <div className="w-0.5 h-6 bg-[var(--border-color)] group-hover:bg-white rounded-full" />
            </div>

            {/* Backlog Header */}
            <div className="p-3 border-b border-[var(--border-color)] flex items-center justify-between gap-2 shrink-0">
              <div className="flex items-center gap-2">
                <Layers size={14} className="text-cyan-400" />
                <h3 className="text-xs font-bold text-[var(--text-primary)]">
                  Backlog non planifié
                </h3>
                <span className="text-[10px] font-mono font-bold px-1.5 py-0.2 rounded-full bg-[var(--bg-tertiary)] text-[var(--text-secondary)]">
                  {unscheduledTasks.length}
                </span>
              </div>

              <div className="flex items-center gap-1">
                <button
                  type="button"
                  onClick={() => setBacklogWidth(prev => (prev > 450 ? 340 : 540))}
                  className="p-1 rounded text-[var(--text-muted)] hover:text-[var(--text-primary)] cursor-pointer"
                  title={backlogWidth > 450 ? 'Réduire la largeur' : 'Agrandir la largeur'}
                >
                  {backlogWidth > 450 ? <Minimize2 size={13} /> : <Maximize2 size={13} />}
                </button>
                <button
                  type="button"
                  onClick={() => setIsBacklogOpen(false)}
                  className="p-1 rounded text-[var(--text-muted)] hover:text-[var(--text-primary)] cursor-pointer"
                  title="Fermer le Backlog"
                >
                  <X size={14} />
                </button>
              </div>
            </div>

            {/* Search filter in Backlog */}
            <div className="p-2 border-b border-[var(--border-color)] bg-[var(--bg-tertiary)]/40 flex items-center gap-2 shrink-0">
              <div className="relative flex-1">
                <input
                  type="text"
                  value={backlogSearch}
                  onChange={e => setBacklogSearch(e.target.value)}
                  placeholder="Filtrer le backlog..."
                  className="w-full pl-7 pr-2 py-1 text-xs rounded-lg bg-[var(--bg-primary)] border border-[var(--border-color)] text-[var(--text-primary)] focus:outline-none focus:border-[var(--accent-color)]"
                />
                <Search size={12} className="absolute left-2.5 top-2 text-[var(--text-muted)]" />
              </div>

              {finishedUnscheduledCount > 0 && (
                <button
                  type="button"
                  onClick={() => setShowDoneInBacklog(prev => !prev)}
                  className={`flex items-center gap-1 px-2 py-1 rounded-lg text-[10.5px] font-semibold border transition-all cursor-pointer shrink-0 ${
                    showDoneInBacklog
                      ? 'bg-emerald-500/15 text-emerald-400 border-emerald-500/30'
                      : 'bg-[var(--bg-primary)] text-[var(--text-muted)] border-[var(--border-color)] hover:text-[var(--text-primary)]'
                  }`}
                  title={showDoneInBacklog ? 'Masquer les tâches terminées (#finished)' : `Afficher les ${finishedUnscheduledCount} tâches terminées`}
                >
                  <CheckCircle2 size={11} />
                  <span>{showDoneInBacklog ? 'Terminées' : `Terminées (${finishedUnscheduledCount})`}</span>
                </button>
              )}
            </div>

            {/* Multi-Selection Action Strip */}
            <div className="px-3 py-1.5 border-b border-[var(--border-color)] bg-[var(--bg-secondary)] flex items-center justify-between gap-2 text-xs shrink-0">
              <button
                type="button"
                onClick={handleSelectAllBacklog}
                className="flex items-center gap-1.5 text-[11px] font-semibold text-[var(--text-muted)] hover:text-[var(--text-primary)] cursor-pointer"
              >
                {unscheduledTasks.length > 0 && unscheduledTasks.every(t => checkedTaskIds[t.id]) ? (
                  <>
                    <CheckSquare size={13} className="text-[var(--accent-color)]" />
                    <span>Tout désélectionner</span>
                  </>
                ) : (
                  <>
                    <Square size={13} />
                    <span>Tout cocher</span>
                  </>
                )}
              </button>

              {selectedBacklogList.length > 0 && (
                <span className="text-[10px] font-bold text-[var(--accent-color)] bg-[var(--accent-light)] px-1.5 py-0.5 rounded border border-[var(--accent-color)]/30">
                  {selectedBacklogList.length} sélectionné(s)
                </span>
              )}
            </div>

            {/* Batch Action Bar (if multiple items selected) */}
            {selectedBacklogList.length > 0 && (
              <div className="p-2 border-b border-[var(--border-color)] bg-[var(--accent-light)]/25 flex items-center gap-1.5 shrink-0">
                <select
                  value={batchTargetSprint}
                  onChange={e => setBatchTargetSprint(e.target.value)}
                  className="flex-1 text-xs px-2 py-1 rounded bg-[var(--bg-primary)] border border-[var(--border-color)] text-[var(--text-primary)] focus:outline-none cursor-pointer"
                >
                  <option value="">Déplacer vers un sprint...</option>
                  {sprints.map(sp => (
                    <option key={sp.name} value={sp.name}>
                      {sp.name}
                    </option>
                  ))}
                </select>
                <button
                  type="button"
                  disabled={!batchTargetSprint || batchBusy}
                  onClick={handleApplyBatchSprint}
                  className="px-2.5 py-1 rounded text-xs font-bold bg-[var(--accent-color)] text-white hover:opacity-90 disabled:opacity-40 cursor-pointer shadow-xs"
                >
                  {batchBusy ? '…' : 'Déplacer'}
                </button>
              </div>
            )}

            {/* Backlog List */}
            <div className="flex-1 overflow-y-auto p-2.5 space-y-1.5">
              {unscheduledTasks.length === 0 ? (
                <div className="py-12 text-center text-xs text-[var(--text-muted)]">
                  {backlogSearch ? 'Aucun résultat' : 'Toutes les tâches sont planifiées ! 🎉'}
                </div>
              ) : displayMode === 'chips' ? (
                /* Chips representation in Backlog */
                <div className="flex flex-wrap gap-1.5">
                  {unscheduledTasks.map(task => (
                    <div
                      key={task.id}
                      draggable
                      onDragStart={e => handleDragStart(e, task.id)}
                      onClick={() => setSelectedTask(task)}
                      className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-lg border transition-all cursor-grab active:cursor-grabbing group text-xs select-none shadow-2xs max-w-full ${
                        checkedTaskIds[task.id]
                          ? 'bg-[var(--accent-light)] border-[var(--accent-color)] text-[var(--accent-color)] font-semibold'
                          : 'bg-[var(--bg-primary)] border-[var(--border-color)] hover:border-[var(--accent-color)]/60 text-[var(--text-primary)]'
                      }`}
                      title={`${task.key}: ${task.title}`}
                    >
                      <input
                        type="checkbox"
                        checked={Boolean(checkedTaskIds[task.id])}
                        onChange={e => {
                          e.stopPropagation()
                          toggleTaskCheck(task.id)
                        }}
                        className="rounded text-[var(--accent-color)] w-3 h-3 cursor-pointer"
                      />
                      <span className="font-mono font-bold text-[10px] text-[var(--accent-color)] shrink-0">
                        {task.key}
                      </span>
                      <span className="truncate max-w-[160px] text-[11px] font-medium">
                        {task.title}
                      </span>
                      <select
                        value=""
                        onChange={e => {
                          if (e.target.value) {
                            setTaskSprint(task.id, e.target.value, e.target.value)
                          }
                        }}
                        onClick={e => e.stopPropagation()}
                        className="text-[9px] font-semibold px-1 py-0.2 rounded bg-[var(--bg-tertiary)] text-[var(--accent-color)] border border-[var(--border-color)] focus:outline-none cursor-pointer ml-auto"
                      >
                        <option value="">+ Sprint</option>
                        {sprints.map(sp => (
                          <option key={sp.name} value={sp.name}>
                            {sp.name}
                          </option>
                        ))}
                      </select>
                    </div>
                  ))}
                </div>
              ) : (
                /* Cards representation in Backlog */
                unscheduledTasks.map(task => (
                  <div
                    key={task.id}
                    draggable
                    onDragStart={e => handleDragStart(e, task.id)}
                    onClick={() => setSelectedTask(task)}
                    className={`p-2.5 rounded-xl border shadow-xs transition-all cursor-grab active:cursor-grabbing space-y-1.5 group ${
                      checkedTaskIds[task.id]
                        ? 'bg-[var(--accent-light)]/20 border-[var(--accent-color)]'
                        : 'bg-[var(--bg-primary)] border-[var(--border-color)] hover:border-[var(--accent-color)]/60'
                    }`}
                  >
                    <div className="flex items-center justify-between gap-1.5">
                      <div className="flex items-center gap-1.5 min-w-0">
                        <input
                          type="checkbox"
                          checked={Boolean(checkedTaskIds[task.id])}
                          onChange={e => {
                            e.stopPropagation()
                            toggleTaskCheck(task.id)
                          }}
                          className="rounded text-[var(--accent-color)] w-3 h-3 cursor-pointer"
                        />
                        <span className="text-[10px] font-mono font-bold text-[var(--accent-color)] shrink-0">
                          {task.key}
                        </span>
                        {task.parentTitle && (
                          <span className="text-[9px] px-1.5 py-0.2 rounded font-bold bg-[var(--bg-tertiary)] text-[var(--text-muted)] truncate max-w-[120px]">
                            {task.parentTitle}
                          </span>
                        )}
                      </div>

                      {task.assignee && (
                        <Avatar name={task.assignee} url={task.assigneeAvatar} size={15} />
                      )}
                    </div>

                    <h4 className="text-xs font-medium text-[var(--text-primary)] line-clamp-2 leading-snug">
                      {task.title}
                    </h4>

                    <div className="flex items-center justify-between pt-1 border-t border-[var(--border-color)]/40 text-[10px]">
                      <span className="text-[var(--text-muted)] flex items-center gap-1">
                        <GripVertical size={11} className="text-[var(--text-muted)]" /> Glisser
                      </span>

                      {/* Quick Assign Dropdown */}
                      <select
                        value=""
                        onChange={e => {
                          if (e.target.value) {
                            setTaskSprint(task.id, e.target.value, e.target.value)
                          }
                        }}
                        onClick={e => e.stopPropagation()}
                        className="text-[9.5px] font-semibold px-1.5 py-0.5 rounded bg-[var(--bg-tertiary)] text-[var(--accent-color)] border border-[var(--border-color)] focus:outline-none cursor-pointer"
                      >
                        <option value="">+ Sprint</option>
                        {sprints.map(sp => (
                          <option key={sp.name} value={sp.name}>
                            {sp.name}
                          </option>
                        ))}
                      </select>
                    </div>
                  </div>
                ))
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
