import React, { useState, useEffect, useRef } from 'react'
import {
  X,
  Plus,
  Loader2
} from 'lucide-react'
import { useApp } from '../context/AppContext'
import type { Status, Priority, TaskSource } from '../types'

export const QuickAddModal: React.FC = () => {
  const {
    isQuickAddOpen,
    setIsQuickAddOpen,
    quickAddInitialStatus,
    createTask,
    projects,
    selectedProjectId,
    t,
  } = useApp()

  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [status, setStatus] = useState<Status>(quickAddInitialStatus)
  const [priority, setPriority] = useState<Priority>('medium')
  const fallbackProjectId = selectedProjectId !== 'all' ? selectedProjectId : (projects[0]?.id || 'default')
  const [taskProjectId, setTaskProjectId] = useState<string>(fallbackProjectId)
  const defaultProj = projects.find(p => p.id === fallbackProjectId) || projects[0]
  const [source, setSource] = useState<TaskSource>(
    (defaultProj?.issueTracker as TaskSource) || 'local'
  )
  const [labels, setLabels] = useState<string[]>([])
  const [labelInput, setLabelInput] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)

  const inputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    if (isQuickAddOpen) {
      setTitle('')
      setDescription('')
      setStatus(quickAddInitialStatus || 'to_clarify')
      setPriority('medium')
      const initialProjId = selectedProjectId !== 'all' ? selectedProjectId : (projects[0]?.id || 'default')
      setTaskProjectId(initialProjId)
      const proj = projects.find(p => p.id === initialProjId) || projects[0]
      setSource((proj?.issueTracker as TaskSource) || 'local')
      setLabels(['New'])
      setLabelInput('')
      setTimeout(() => {
        inputRef.current?.focus()
      }, 50)
    }
  }, [isQuickAddOpen, quickAddInitialStatus, selectedProjectId, projects])

  useEffect(() => {
    if (!isQuickAddOpen) return
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        setIsQuickAddOpen(false)
      }
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [isQuickAddOpen, setIsQuickAddOpen])

  const activeProject = projects.find(p => p.id === taskProjectId) || projects[0]

  if (!isQuickAddOpen) return null

  const handleProjectChange = (projId: string) => {
    setTaskProjectId(projId)
    const proj = projects.find(p => p.id === projId)
    if (proj?.issueTracker) {
      setSource(proj.issueTracker as TaskSource)
    }
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!title.trim() || isSubmitting) return

    setIsSubmitting(true)
    const created = await createTask({
      title: title.trim(),
      description: description.trim(),
      status,
      priority,
      labels,
      source,
      projectId: taskProjectId,
    })
    setIsSubmitting(false)

    if (created) {
      setIsQuickAddOpen(false)
    }
  }

  const handleAddLabel = () => {
    const clean = labelInput.replace(/^#/, '').trim()
    if (clean && !labels.includes(clean)) {
      setLabels([...labels, clean])
      setLabelInput('')
    }
  }

  const removeLabel = (tag: string) => {
    setLabels(labels.filter(l => l !== tag))
  }

  return (
    <div className="fixed top-0 left-0 h-[var(--app-h)] w-[var(--app-w)] z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-xs animate-in fade-in duration-150">
      <div className="relative w-full max-w-lg rounded-2xl bg-[var(--bg-secondary)] border border-[var(--border-color)] shadow-2xl overflow-hidden">
        {/* Header */}
        <div className="flex items-center justify-between px-5 py-3.5 border-b border-[var(--border-color)] bg-[var(--bg-tertiary)]/40">
          <div className="flex items-center gap-2 font-semibold text-xs text-[var(--text-primary)]">
            <div className="w-5 h-5 rounded-md accent-bg text-white flex items-center justify-center">
              <Plus size={13} />
            </div>
            <span>{t.quickAdd.title}</span>
          </div>
          <button
            onClick={() => setIsQuickAddOpen(false)}
            className="p-1 rounded-md text-[var(--text-muted)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-tertiary)] transition-colors"
          >
            <X size={16} />
          </button>
        </div>

        {/* Form Body */}
        <form onSubmit={handleSubmit} className="p-5 space-y-4">
          {/* Title input */}
          <div>
            <input
              ref={inputRef}
              type="text"
              value={title}
              onChange={e => setTitle(e.target.value)}
              placeholder={t.quickAdd.placeholder}
              className="w-full px-3.5 py-2 text-xs font-semibold rounded-xl bg-[var(--bg-tertiary)] border border-[var(--border-color)] text-[var(--text-primary)] placeholder-[var(--text-muted)] focus:outline-none focus:border-[var(--accent-color)] focus:ring-1 focus:ring-[var(--accent-color)]"
            />
          </div>

          {/* Description input */}
          <div>
            <textarea
              rows={3}
              value={description}
              onChange={e => setDescription(e.target.value)}
              placeholder={t.taskModal.descPlaceholder}
              className="w-full px-3.5 py-2 text-xs rounded-xl bg-[var(--bg-tertiary)] border border-[var(--border-color)] text-[var(--text-primary)] placeholder-[var(--text-muted)] focus:outline-none focus:border-[var(--accent-color)] leading-relaxed resize-none"
            />
          </div>

          {/* Project Target */}
          {projects.length > 0 && (
            <div>
              <div className="flex items-center justify-between mb-1">
                <label className="block text-[10px] font-bold uppercase tracking-wider text-[var(--text-muted)]">
                  Projet *
                </label>
                {activeProject && (
                  <span className="text-[10px] text-[var(--text-muted)] flex items-center gap-1 font-mono">
                    <span>Tracker :</span>
                    <span className="font-semibold text-[var(--accent-color)]">
                      {activeProject.issueTracker === 'linear'
                        ? `Linear (${activeProject.linearTeam || 'FRE'})`
                        : activeProject.issueTracker === 'github'
                        ? `GitHub (${activeProject.githubRepo ? activeProject.githubRepo.split('/')[1] || activeProject.githubRepo : 'repo'})`
                        : 'Local'}
                    </span>
                  </span>
                )}
              </div>
              <select
                value={taskProjectId}
                onChange={e => handleProjectChange(e.target.value)}
                className="w-full px-3 py-2 text-xs rounded-xl bg-[var(--bg-tertiary)] border border-[var(--border-color)] text-[var(--text-primary)] focus:outline-none focus:border-[var(--accent-color)] font-medium"
              >
                {projects.map(p => (
                  <option key={p.id} value={p.id}>
                    {p.name}
                  </option>
                ))}
              </select>
            </div>
          )}

          {/* Creation destination: which tracker actually receives the issue */}
          <div>
            <label className="block text-[10px] font-bold uppercase tracking-wider text-[var(--text-muted)] mb-1">
              Destination
            </label>
            <div className="grid grid-cols-3 gap-1.5">
              {([
                { id: 'linear', label: 'Linear', icon: '🟣', hint: activeProject?.linearTeam || 'équipe non configurée' },
                { id: 'github', label: 'GitHub', icon: '🐙', hint: activeProject?.githubRepo || 'dépôt non configuré' },
                { id: 'local', label: 'Local', icon: '📁', hint: 'SQLite' },
              ] as { id: TaskSource; label: string; icon: string; hint: string }[]).map(opt => (
                <button
                  key={opt.id}
                  type="button"
                  title={`${opt.label} — ${opt.hint}`}
                  onClick={() => setSource(opt.id)}
                  className={`flex flex-col items-center gap-0.5 px-2 py-1.5 rounded-xl border text-[11px] font-semibold transition-all ${
                    source === opt.id
                      ? 'border-[var(--accent-color)] bg-[var(--accent-light)] accent-text'
                      : 'border-[var(--border-color)] bg-[var(--bg-tertiary)] text-[var(--text-secondary)] hover:border-[var(--accent-color)]/50'
                  }`}
                >
                  <span className="text-sm leading-none">{opt.icon}</span>
                  <span>{opt.label}</span>
                </button>
              ))}
            </div>
          </div>

          {/* Status & Priority Selectors */}
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-[10px] font-bold uppercase tracking-wider text-[var(--text-muted)] mb-1">
                {t.quickAdd.status}
              </label>
              <select
                value={status}
                onChange={e => setStatus(e.target.value as Status)}
                className="w-full px-3 py-1.5 text-xs rounded-xl bg-[var(--bg-tertiary)] border border-[var(--border-color)] text-[var(--text-primary)] focus:outline-none focus:border-[var(--accent-color)]"
              >
                <option value="to_clarify">{t.status.to_clarify} (#new)</option>
                <option value="to_specify">{t.status.to_specify} (#clarified)</option>
                <option value="to_implement">{t.status.to_implement} (#specified)</option>
                <option value="to_test">{t.status.to_test} (#implemented)</option>
                <option value="to_close">{t.status.to_close} (#reviewed)</option>
                <option value="finished">{t.status.finished} (#finished)</option>
              </select>
            </div>

            <div>
              <label className="block text-[10px] font-bold uppercase tracking-wider text-[var(--text-muted)] mb-1">
                {t.quickAdd.priority}
              </label>
              <select
                value={priority}
                onChange={e => setPriority(e.target.value as Priority)}
                className="w-full px-3 py-1.5 text-xs rounded-xl bg-[var(--bg-tertiary)] border border-[var(--border-color)] text-[var(--text-primary)] focus:outline-none focus:border-[var(--accent-color)]"
              >
                <option value="urgent">{t.priority.urgent}</option>
                <option value="high">{t.priority.high}</option>
                <option value="medium">{t.priority.medium}</option>
                <option value="low">{t.priority.low}</option>
              </select>
            </div>
          </div>

          {/* Labels */}
          <div>
            <label className="block text-[10px] font-bold uppercase tracking-wider text-[var(--text-muted)] mb-1">
              {t.taskModal.labels}
            </label>
            <div className="flex flex-wrap items-center gap-1.5 p-2 rounded-xl bg-[var(--bg-tertiary)] border border-[var(--border-color)]">
              {labels.map(l => {
                const lower = l.toLowerCase()
                let badgeStyle = 'bg-[var(--accent-light)] accent-text'
                if (lower === 'new') badgeStyle = 'bg-cyan-500/20 text-cyan-300 border border-cyan-500/40 font-bold'
                else if (lower === 'clarified') badgeStyle = 'bg-amber-500/20 text-amber-300 border border-amber-500/40 font-bold'
                else if (lower === 'specified') badgeStyle = 'bg-blue-500/20 text-blue-300 border border-blue-500/40 font-bold'
                else if (lower === 'implemented') badgeStyle = 'bg-indigo-500/20 text-indigo-300 border border-indigo-500/40 font-bold'
                else if (lower === 'reviewed') badgeStyle = 'bg-emerald-500/20 text-emerald-300 border border-emerald-500/40 font-bold'

                return (
                  <span
                    key={l}
                    className={`inline-flex items-center gap-1 text-[11px] px-2 py-0.5 rounded-md ${badgeStyle}`}
                  >
                    #{l}
                    <button
                      type="button"
                      onClick={() => removeLabel(l)}
                      className="hover:opacity-75 ml-0.5"
                    >
                      <X size={11} />
                    </button>
                  </span>
                )
              })}
              <div className="flex items-center gap-1 min-w-[100px] flex-1">
                <input
                  type="text"
                  value={labelInput}
                  onChange={e => setLabelInput(e.target.value)}
                  onKeyDown={e => {
                    if (e.key === 'Enter' || e.key === ',') {
                      e.preventDefault()
                      handleAddLabel()
                    }
                  }}
                  placeholder={t.taskModal.addLabel}
                  className="w-full text-xs bg-transparent border-none text-[var(--text-primary)] placeholder-[var(--text-muted)] focus:outline-none px-1"
                />
              </div>
            </div>
          </div>

          {/* Footer */}
          <div className="flex items-center justify-between pt-2 border-t border-[var(--border-color)]">
            <span className="text-[11px] text-[var(--text-muted)]">
              {t.quickAdd.hint}
            </span>
            <div className="flex items-center gap-2">
              <button
                type="button"
                onClick={() => setIsQuickAddOpen(false)}
                className="px-3 py-1.5 rounded-xl text-xs font-medium text-[var(--text-secondary)] hover:bg-[var(--bg-tertiary)] transition-colors"
              >
                {t.taskModal.cancel}
              </button>
              <button
                type="submit"
                disabled={!title.trim() || isSubmitting}
                className="px-4 py-1.5 rounded-xl text-xs font-bold text-white accent-bg shadow hover:opacity-90 active:scale-95 disabled:opacity-50 flex items-center gap-1.5 transition-all"
              >
                {isSubmitting ? (
                  <>
                    <Loader2 size={13} className="animate-spin" />
                    <span>Création CLI...</span>
                  </>
                ) : (
                  <>
                    <Plus size={13} />
                    <span>{t.taskModal.create}</span>
                  </>
                )}
              </button>
            </div>
          </div>
        </form>
      </div>
    </div>
  )
}
