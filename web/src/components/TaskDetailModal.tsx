import React, { useState, useEffect } from 'react'
import {
  X,
  Trash2,
  Calendar,
  User,
  Sparkles,
  HelpCircle,
  FileCode,
  Flame,
  ShieldCheck,
  GitBranch,
  ExternalLink,
  Loader2,
  CheckCircle2,
  History,
  Terminal,
  PanelRight,
  Square,
  Bot,
  Save,
  Check,
  Copy,
  MessageSquare,
  Send,
  ArrowRight,
  Folder,
  FolderGit2,
  GitPullRequest,
  Code2,
  Maximize2,
  Minimize2,
} from 'lucide-react'
import { useApp } from '../context/AppContext'
import type { Status, Priority, DetailMode } from '../types'

export const TaskDetailModal: React.FC = () => {
  const {
    selectedTask,
    setSelectedTask,
    setDiffTask,
    updateTask,
    deleteTask,
    runSkill,
    isSkillRunning,
    runningSkillId,
    skills,
    projects,
    activities: globalActivities,
    settings,
    updateSettings,
    addToast,
    t,
  } = useApp()

  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [status, setStatus] = useState<Status>('backlog')
  const [priority, setPriority] = useState<Priority>('medium')
  const [taskProjectId, setTaskProjectId] = useState<string>('fretzee-studio')
  const [branchName, setBranchName] = useState('')
  const [prUrl, setPrUrl] = useState('')
  const [labels, setLabels] = useState<string[]>([])
  const [newLabelInput, setNewLabelInput] = useState('')
  const [assignee, setAssignee] = useState('')
  const [dueDate, setDueDate] = useState('')
  const [isSaving, setIsSaving] = useState(false)
  const [activeTab, setActiveTab] = useState<'details' | 'skills' | 'history'>('details')
  const [customPrompt, setCustomPrompt] = useState('')
  const [copiedBranch, setCopiedBranch] = useState(false)
  const [clarificationInput, setClarificationInput] = useState('')
  const [isPostingComment, setIsPostingComment] = useState(false)
  const [isEnrichingDesc, setIsEnrichingDesc] = useState(false)
  const [isExpandedQA, setIsExpandedQA] = useState(false)
  const [isExpandedSpec, setIsExpandedSpec] = useState(false)
  const [copiedSpec, setCopiedSpec] = useState(false)

  const detailMode: DetailMode = settings.detailMode || 'panel'

  useEffect(() => {
    if (selectedTask) {
      setTitle(selectedTask.title)
      setDescription(selectedTask.description || '')
      setStatus(selectedTask.status)
      setPriority(selectedTask.priority)
      setTaskProjectId(selectedTask.projectId || 'fretzee-studio')
      setBranchName(selectedTask.branchName || '')
      setPrUrl(selectedTask.prUrl || '')
      setLabels(selectedTask.labels || [])
      setAssignee(selectedTask.assignee || '')
      setDueDate(selectedTask.dueDate || '')
    }
  }, [selectedTask])

  useEffect(() => {
    if (!selectedTask) return
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        if (isExpandedQA) {
          setIsExpandedQA(false)
        } else if (isExpandedSpec) {
          setIsExpandedSpec(false)
        } else {
          setSelectedTask(null)
        }
      }
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [selectedTask, setSelectedTask, isExpandedQA, isExpandedSpec])

  if (!selectedTask) return null

  const handleSave = async () => {
    if (!title.trim() || isSaving) return

    setIsSaving(true)
    await updateTask(selectedTask.id, {
      title: title.trim(),
      description: description.trim(),
      status,
      priority,
      projectId: taskProjectId,
      labels,
      assignee: assignee.trim(),
      dueDate: dueDate || null,
      branchName: branchName.trim() || undefined,
      prUrl: prUrl.trim() || undefined,
    })
    setIsSaving(false)
    setSelectedTask(null)
  }

  const handleStatusChange = async (newStatus: Status) => {
    setStatus(newStatus)
    if (selectedTask) {
      await updateTask(selectedTask.id, { status: newStatus })
    }
  }

  const handlePriorityChange = async (newPriority: Priority) => {
    setPriority(newPriority)
    if (selectedTask) {
      await updateTask(selectedTask.id, { priority: newPriority })
    }
  }

  const handleSaveAnswersToDescription = async () => {
    if (!clarificationInput.trim() || !selectedTask) return
    setIsEnrichingDesc(true)
    const updatedDesc = description
      ? `${description}\n\n### 💬 Réponses aux questions de cadrage (Inputs) :\n${clarificationInput.trim()}`
      : `### 💬 Réponses aux questions de cadrage (Inputs) :\n${clarificationInput.trim()}`
    setDescription(updatedDesc)
    await updateTask(selectedTask.id, { description: updatedDesc })
    setClarificationInput('')
    setIsEnrichingDesc(false)
    addToast({
      type: 'success',
      title: 'Description enrichie',
      description: 'Les réponses ont été ajoutées à la story.',
    })
  }

  const handlePostAnswersComment = async () => {
    if (!clarificationInput.trim() || !selectedTask) return
    setIsPostingComment(true)
    try {
      const commentBody = `### 💬 [Fretzee Tasks] Réponses aux questions de cadrage (Inputs)\n\n${clarificationInput.trim()}`
      const res = await fetch(`/api/tasks/${encodeURIComponent(selectedTask.id)}/comment`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ body: commentBody }),
      })
      if (!res.ok) throw new Error('Comment failed')
      addToast({
        type: 'success',
        title: 'Commentaire posté',
        description: `Le commentaire a été publié sur ${selectedTask.source === 'linear' ? 'Linear' : 'GitHub'}.`,
      })
      setClarificationInput('')
    } catch (err: any) {
      addToast({
        type: 'error',
        title: 'Erreur',
        description: err.message,
      })
    } finally {
      setIsPostingComment(false)
    }
  }

  const handleSaveAndSpecify = async () => {
    if (!selectedTask) return
    const userAnswers = clarificationInput.trim()
    if (userAnswers) {
      const updatedDesc = description
        ? `${description}\n\n### 💬 Réponses aux questions de cadrage (Inputs) :\n${userAnswers}`
        : `### 💬 Réponses aux questions de cadrage (Inputs) :\n${userAnswers}`
      setDescription(updatedDesc)
      await updateTask(selectedTask.id, { description: updatedDesc, status: 'to_specify' })
      
      // Also post comment on Linear/GitHub if remote task
      if (selectedTask.source === 'linear' || selectedTask.source === 'github') {
        try {
          await fetch(`/api/tasks/${encodeURIComponent(selectedTask.id)}/comment`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ body: `### 💬 [Fretzee Tasks] Réponses aux questions de cadrage\n\n${userAnswers}` }),
          })
        } catch (e) {
          console.error('Failed to post comment', e)
        }
      }

      setClarificationInput('')
      setStatus('to_specify')
      await runSkill(selectedTask.id, 'specify', `Réponses de cadrage fournies par l'utilisateur :\n${userAnswers}`)
    } else {
      await updateTask(selectedTask.id, { status: 'to_specify' })
      setStatus('to_specify')
      await runSkill(selectedTask.id, 'specify', customPrompt)
    }
  }

  const handleDelete = async () => {
    if (window.confirm(t.taskModal.deleteConfirm)) {
      await deleteTask(selectedTask.id)
      setSelectedTask(null)
    }
  }

  const handleAddLabel = () => {
    const clean = newLabelInput.replace(/^#/, '').trim()
    if (clean && !labels.includes(clean)) {
      setLabels([...labels, clean])
      setNewLabelInput('')
    }
  }

  const handleRemoveLabel = (tagToRemove: string) => {
    setLabels(labels.filter(l => l !== tagToRemove))
  }

  const handleToggleDetailMode = async () => {
    const nextMode: DetailMode = detailMode === 'panel' ? 'modal' : 'panel'
    await updateSettings({ detailMode: nextMode })
  }

  const getNextRecommendedSkill = () => {
    switch (status) {
      case 'to_clarify':
      case 'backlog':
        return skills.find(s => s.id === 'clarify') || skills[0]
      case 'to_specify':
      case 'specified':
        return skills.find(s => s.id === 'specify') || skills[1]
      case 'to_implement':
      case 'in_progress':
        return skills.find(s => s.id === 'implement') || skills[2]
      case 'to_test':
      case 'to_validate':
        return skills.find(s => s.id === 'create_pr' || s.id === 'review') || skills[3]
      default:
        return skills.find(s => s.id === 'pick') || skills[4]
    }
  }

  const handleTriggerSkill = async (skillId: string) => {
    if (!selectedTask || isSkillRunning) return
    const activity = await runSkill(selectedTask.id, skillId, customPrompt)
    if (activity) {
      setCustomPrompt('')
    }
  }

  const nextSkill = getNextRecommendedSkill()

  const getSkillIcon = (iconName: string) => {
    switch (iconName) {
      case 'HelpCircle':
        return <HelpCircle size={15} className="text-amber-400" />
      case 'FileCode':
        return <FileCode size={15} className="text-blue-400" />
      case 'Flame':
        return <Flame size={15} className="text-indigo-400" />
      case 'ShieldCheck':
        return <ShieldCheck size={15} className="text-purple-400" />
      default:
        return <Sparkles size={15} className="text-emerald-400" />
    }
  }

  const copyBranchCommand = (branch: string) => {
    navigator.clipboard.writeText(`git checkout -b ${branch}`)
    setCopiedBranch(true)
    setTimeout(() => setCopiedBranch(false), 2000)
  }

  const activities = selectedTask
    ? globalActivities.filter(a => a.taskId === selectedTask.id || a.taskKey === selectedTask.key)
    : []
  const latestActivity = activities.length > 0 ? activities[0] : null
  const clarifyActivity = activities.find(a => a.skillId === 'clarify')
  const specifyActivity = activities.find(a => a.skillId === 'specify')

  const handleCopySpec = () => {
    if (!specifyActivity?.output) return
    navigator.clipboard?.writeText(specifyActivity.output)
    setCopiedSpec(true)
    setTimeout(() => setCopiedSpec(false), 2000)
    addToast({
      type: 'success',
      title: 'Spécification copiée',
      description: 'La spécification technique a été copiée dans votre presse-papiers.',
    })
  }

  // Interactive Clarification Q&A / Inputs Box
  const renderClarificationQASection = () => {
    const shouldShow = Boolean(
      clarifyActivity ||
      status === 'to_clarify' ||
      status === 'to_specify' ||
      status === 'backlog' ||
      labels.some(l => {
        const low = l.toLowerCase()
        return low === 'clarified' || low === 'to-clarify' || low === 'new'
      })
    )

    if (!shouldShow) return null

    return (
      <div className="p-4 rounded-2xl bg-linear-to-b from-[var(--bg-tertiary)] to-[var(--bg-secondary)] border border-amber-500/40 shadow-md space-y-3">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <MessageSquare size={16} className="text-amber-400" />
            <h4 className="text-xs font-bold text-[var(--text-primary)]">
              Réponses aux questions de cadrage (Inputs)
            </h4>
          </div>
          <div className="flex items-center gap-2">
            <span className="text-[10px] px-2 py-0.5 rounded-full bg-amber-500/20 text-amber-300 font-bold uppercase border border-amber-500/30">
              Alignement Produit & Tech
            </span>
            <button
              type="button"
              onClick={() => setIsExpandedQA(true)}
              className="inline-flex items-center gap-1 px-2.5 py-1 rounded-lg text-[10px] font-bold text-amber-300 bg-amber-500/15 hover:bg-amber-500/25 border border-amber-500/30 transition-all shadow-2xs hover:scale-105"
              title="Agrandir en mode grand format / plein écran"
            >
              <Maximize2 size={11} />
              <span>Agrandir</span>
            </button>
          </div>
        </div>

        {/* Display questions from clarifyActivity if available */}
        {clarifyActivity?.output && (
          <div className="p-3.5 rounded-xl bg-slate-950 border border-slate-800 text-slate-300 font-mono text-[11px] max-h-72 overflow-y-auto space-y-1.5 shadow-inner leading-relaxed">
            <div className="text-[10px] font-bold text-amber-400 uppercase tracking-wider flex items-center justify-between pb-1 border-b border-slate-800">
              <span className="flex items-center gap-1">
                <HelpCircle size={12} />
                <span>Questions de cadrage posées par l'agent Clarify :</span>
              </span>
              <button
                type="button"
                onClick={() => {
                  navigator.clipboard?.writeText(clarifyActivity.output)
                  addToast({ type: 'success', title: 'Questions copiées', description: 'Les questions ont été copiées.' })
                }}
                className="text-[9px] text-slate-400 hover:text-white flex items-center gap-1 font-mono"
              >
                <Copy size={10} />
                <span>Copier</span>
              </button>
            </div>
            <div className="whitespace-pre-wrap pt-1">
              {clarifyActivity.output}
            </div>
          </div>
        )}

        <p className="text-[11px] text-[var(--text-muted)] leading-relaxed">
          Renseignez vos arbitrages produit & technique aux questions posées par l'agent Clarify pour alimenter l'étape de spécification.
        </p>

        <textarea
          value={clarificationInput}
          onChange={e => setClarificationInput(e.target.value)}
          rows={6}
          placeholder={`Saisissez vos réponses aux questions (ex :\n1. Mode Serverless avec scale-to-zero uniquement en Staging/Dev\n2. minConnections=0 et idleTimeout=30s pour libérer les connexions\n3. Neutraliser les sondes healthcheck périodiques vers la DB...)`}
          className="w-full p-3.5 text-xs rounded-xl bg-[var(--bg-primary)] border border-[var(--border-color)] text-[var(--text-primary)] placeholder-[var(--text-muted)]/70 focus:outline-none focus:border-[var(--accent-color)] focus:ring-1 focus:ring-[var(--accent-color)] font-mono leading-relaxed resize-y min-h-[130px]"
        />

        <div className="flex flex-wrap items-center justify-between gap-2 pt-1">
          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={handleSaveAnswersToDescription}
              disabled={!clarificationInput.trim() || isEnrichingDesc}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-xl text-xs font-semibold bg-[var(--bg-tertiary)] hover:bg-[var(--border-color)] text-[var(--text-secondary)] hover:text-[var(--text-primary)] border border-[var(--border-color)] transition-all disabled:opacity-40"
              title="Ajouter ces réponses à la description de la story"
            >
              {isEnrichingDesc ? <Loader2 size={12} className="animate-spin" /> : <Save size={12} />}
              <span>Enrichir la description</span>
            </button>

            <button
              type="button"
              onClick={handlePostAnswersComment}
              disabled={!clarificationInput.trim() || isPostingComment}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-xl text-xs font-semibold bg-[var(--bg-tertiary)] hover:bg-[var(--border-color)] text-[var(--text-secondary)] hover:text-[var(--text-primary)] border border-[var(--border-color)] transition-all disabled:opacity-40"
              title={`Poster ce commentaire directement sur ${selectedTask.source === 'linear' ? 'Linear' : 'GitHub'}`}
            >
              {isPostingComment ? <Loader2 size={12} className="animate-spin" /> : <Send size={12} />}
              <span className="hidden sm:inline">Commenter ({selectedTask.source === 'linear' ? 'Linear' : 'GitHub'})</span>
            </button>
          </div>

          <button
            type="button"
            onClick={handleSaveAndSpecify}
            disabled={isSkillRunning}
            className="flex items-center gap-1.5 px-3.5 py-1.5 rounded-xl text-xs font-bold text-white shadow-md accent-bg hover:opacity-90 active:scale-95 transition-all disabled:opacity-50"
            title="Sauvegarder les réponses, passer en statut Spécifié et lancer le skill Specify"
          >
            {isSkillRunning && runningSkillId === 'specify' ? (
              <Loader2 size={13} className="animate-spin" />
            ) : (
              <>
                <Sparkles size={13} className="text-amber-300" />
                <span>Enregistrer & Lancer Specify</span>
                <ArrowRight size={13} />
              </>
            )}
          </button>
        </div>
      </div>
    )
  }

  // Technical Specification (Speckit) Section Box
  const renderSpecificationSection = () => {
    if (!specifyActivity?.output) return null

    return (
      <div className="p-4 rounded-2xl bg-linear-to-b from-[var(--bg-tertiary)] to-[var(--bg-secondary)] border border-blue-500/40 shadow-md space-y-3">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <FileCode size={16} className="text-blue-400" />
            <h4 className="text-xs font-bold text-[var(--text-primary)]">
              Spécification Technique Speckit
            </h4>
          </div>
          <div className="flex items-center gap-2">
            <span className="text-[10px] px-2 py-0.5 rounded-full bg-blue-500/20 text-blue-300 font-bold uppercase border border-blue-500/30">
              Spécifié
            </span>
            <button
              type="button"
              onClick={handleCopySpec}
              className="inline-flex items-center gap-1 px-2 py-1 rounded-lg text-[10px] font-mono font-bold bg-[var(--bg-primary)] hover:bg-[var(--bg-tertiary)] text-[var(--text-secondary)] border border-[var(--border-color)] transition-colors"
              title="Copier la spec complète"
            >
              {copiedSpec ? <Check size={11} className="text-emerald-400" /> : <Copy size={11} />}
              <span>{copiedSpec ? 'Copié !' : 'Copier'}</span>
            </button>
            <button
              type="button"
              onClick={() => setIsExpandedSpec(true)}
              className="inline-flex items-center gap-1 px-2.5 py-1 rounded-lg text-[10px] font-bold text-blue-300 bg-blue-500/15 hover:bg-blue-500/25 border border-blue-500/30 transition-all shadow-2xs hover:scale-105"
              title="Agrandir la spécification en mode grand format / plein écran"
            >
              <Maximize2 size={11} />
              <span>Agrandir</span>
            </button>
          </div>
        </div>

        <div className="p-3.5 rounded-xl bg-slate-950 border border-slate-800 text-slate-200 font-mono text-[11px] max-h-80 overflow-y-auto leading-relaxed shadow-inner">
          <div className="whitespace-pre-wrap">
            {specifyActivity.output}
          </div>
        </div>

        <div className="flex items-center justify-between pt-1">
          <div className="text-[11px] text-[var(--text-muted)] font-mono">
            {specifyActivity.completedAt ? `Généré le ${new Date(specifyActivity.completedAt).toLocaleString()}` : 'Spécification prête'}
          </div>
          <button
            type="button"
            onClick={() => handleTriggerSkill('implement')}
            disabled={isSkillRunning}
            className="flex items-center gap-1.5 px-3.5 py-1.5 rounded-xl text-xs font-bold text-white shadow-md bg-linear-to-r from-blue-600 to-indigo-600 hover:opacity-90 active:scale-95 transition-all disabled:opacity-50"
            title="Lancer l'implémentation du code conformément à cette spécification"
          >
            {isSkillRunning && runningSkillId === 'implement' ? (
              <Loader2 size={13} className="animate-spin" />
            ) : (
              <>
                <Flame size={13} className="text-amber-300" />
                <span>Lancer Implement Code</span>
                <ArrowRight size={13} />
              </>
            )}
          </button>
        </div>
      </div>
    )
  }

  // SHARED: Story Info Section Content
  const renderStoryInfoSection = () => (
    <div className="space-y-5">
      {/* Title Input */}
      <div>
        <label className="block text-[10px] font-bold uppercase tracking-wider text-[var(--text-muted)] mb-1">
          Titre de la Story
        </label>
        <input
          type="text"
          value={title}
          onChange={e => setTitle(e.target.value)}
          placeholder={t.taskModal.titlePlaceholder}
          className="w-full px-3.5 py-2 text-sm font-semibold rounded-xl bg-[var(--bg-tertiary)] border border-[var(--border-color)] text-[var(--text-primary)] focus:outline-none focus:border-[var(--accent-color)]"
        />
      </div>

      {/* Metadata Grid: Status, Priority, Project, Assignee, Due Date */}
      <div className="grid grid-cols-2 sm:grid-cols-5 gap-3">
        {/* Status */}
        <div>
          <label className="block text-[10px] font-bold uppercase tracking-wider text-[var(--text-muted)] mb-1">
            {t.taskModal.status}
          </label>
          <select
            value={status}
            onChange={e => handleStatusChange(e.target.value as Status)}
            className="w-full px-2.5 py-1.5 text-xs rounded-xl bg-[var(--bg-tertiary)] border border-[var(--border-color)] text-[var(--text-primary)] focus:outline-none focus:border-[var(--accent-color)]"
          >
            <option value="to_clarify">{t.status.to_clarify} (New)</option>
            <option value="to_specify">{t.status.to_specify} (Clarified)</option>
            <option value="to_implement">{t.status.to_implement} (Specified)</option>
            <option value="to_test">{t.status.to_test} (Implemented)</option>
            <option value="to_close">{t.status.to_close} (Reviewed)</option>
          </select>
        </div>

        {/* Priority */}
        <div>
          <label className="block text-[10px] font-bold uppercase tracking-wider text-[var(--text-muted)] mb-1">
            {t.taskModal.priority}
          </label>
          <select
            value={priority}
            onChange={e => handlePriorityChange(e.target.value as Priority)}
            className="w-full px-2.5 py-1.5 text-xs rounded-xl bg-[var(--bg-tertiary)] border border-[var(--border-color)] text-[var(--text-primary)] focus:outline-none focus:border-[var(--accent-color)]"
          >
            <option value="urgent">{t.priority.urgent}</option>
            <option value="high">{t.priority.high}</option>
            <option value="medium">{t.priority.medium}</option>
            <option value="low">{t.priority.low}</option>
          </select>
        </div>

        {/* Project */}
        <div>
          <label className="block text-[10px] font-bold uppercase tracking-wider text-[var(--text-muted)] mb-1">
            Projet
          </label>
          <select
            value={taskProjectId}
            onChange={async (e) => {
              const val = e.target.value
              setTaskProjectId(val)
              if (selectedTask) {
                await updateTask(selectedTask.id, { projectId: val })
              }
            }}
            className="w-full px-2.5 py-1.5 text-xs rounded-xl bg-[var(--bg-tertiary)] border border-[var(--border-color)] text-[var(--text-primary)] focus:outline-none focus:border-[var(--accent-color)] font-medium"
          >
            {projects.map(p => (
              <option key={p.id} value={p.id}>
                {p.name}
              </option>
            ))}
          </select>
        </div>

        {/* Assignee */}
        <div>
          <label className="block text-[10px] font-bold uppercase tracking-wider text-[var(--text-muted)] mb-1">
            {t.taskModal.assignee}
          </label>
          <div className="relative">
            <input
              type="text"
              value={assignee}
              onChange={e => setAssignee(e.target.value)}
              placeholder="Assigné à..."
              className="w-full pl-7 pr-2.5 py-1.5 text-xs rounded-xl bg-[var(--bg-tertiary)] border border-[var(--border-color)] text-[var(--text-primary)] focus:outline-none focus:border-[var(--accent-color)]"
            />
            <User size={12} className="absolute left-2.5 top-2.5 text-[var(--text-muted)]" />
          </div>
        </div>

        {/* Due Date */}
        <div>
          <label className="block text-[10px] font-bold uppercase tracking-wider text-[var(--text-muted)] mb-1">
            {t.taskModal.dueDate}
          </label>
          <div className="relative">
            <input
              type="date"
              value={dueDate}
              onChange={e => setDueDate(e.target.value)}
              className="w-full pl-7 pr-2 py-1.5 text-xs rounded-xl bg-[var(--bg-tertiary)] border border-[var(--border-color)] text-[var(--text-primary)] focus:outline-none focus:border-[var(--accent-color)]"
            />
            <Calendar size={12} className="absolute left-2.5 top-2.5 text-[var(--text-muted)] pointer-events-none" />
          </div>
        </div>
      </div>

      {/* Git Branch & Pull Request / Merge Request Section */}
      <div className="p-3.5 rounded-xl bg-[var(--bg-tertiary)]/70 border border-[var(--border-color)] space-y-3">
        <div className="flex items-center justify-between">
          <span className="text-[10px] font-bold uppercase tracking-wider text-[var(--text-muted)] flex items-center gap-1.5">
            <GitBranch size={13} className="text-indigo-400" />
            Contrôle de Version Git & Revue (PR / MR)
          </span>
          <button
            type="button"
            onClick={() => setDiffTask(selectedTask)}
            className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-lg text-[10px] font-bold text-indigo-300 bg-indigo-500/15 hover:bg-indigo-500/25 border border-indigo-500/30 transition-all shadow-2xs hover:scale-105 active:scale-95"
            title="Inspecter le diff Git et les fichiers modifiés pour cette tâche"
          >
            <Code2 size={11} className="text-indigo-400" />
            <span>👁️ Voir le Diff Git</span>
          </button>
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
          {/* Git Branch Field */}
          <div>
            <div className="flex items-center justify-between mb-1">
              <label className="text-[10px] font-medium text-[var(--text-secondary)]">
                Branche Git
              </label>
              {branchName && (
                <button
                  type="button"
                  onClick={() => copyBranchCommand(branchName)}
                  className="text-[9px] font-mono font-bold text-indigo-400 hover:underline flex items-center gap-1"
                  title="Copier la commande git checkout"
                >
                  {copiedBranch ? <Check size={10} className="text-emerald-400" /> : <Copy size={10} />}
                  <span>{copiedBranch ? 'Copié !' : 'git checkout'}</span>
                </button>
              )}
            </div>
            <div className="relative">
              <input
                type="text"
                value={branchName}
                onChange={e => setBranchName(e.target.value)}
                placeholder={selectedTask.key ? `${selectedTask.key}-feature-name` : 'main'}
                className="w-full pl-7 pr-2.5 py-1.5 text-xs rounded-xl bg-[var(--bg-primary)] border border-[var(--border-color)] text-[var(--text-primary)] font-mono text-[11px] focus:outline-none focus:border-[var(--accent-color)]"
              />
              <GitBranch size={12} className="absolute left-2.5 top-2.5 text-indigo-400" />
            </div>
          </div>

          {/* Merge Request / Pull Request Field */}
          <div>
            <div className="flex items-center justify-between mb-1">
              <label className="text-[10px] font-medium text-[var(--text-secondary)]">
                Merge Request / Pull Request
              </label>
              {prUrl && (
                <a
                  href={prUrl}
                  target="_blank"
                  rel="noreferrer"
                  className={`text-[9px] font-bold flex items-center gap-1 hover:underline ${
                    prUrl.includes('gitlab') ? 'text-orange-400' : 'text-purple-400'
                  }`}
                  title="Ouvrir la Pull/Merge Request dans le navigateur"
                >
                  <span>{prUrl.includes('gitlab') ? 'Ouvrir GitLab MR' : 'Ouvrir GitHub PR'}</span>
                  <ExternalLink size={9} />
                </a>
              )}
            </div>
            <div className="relative">
              <input
                type="url"
                value={prUrl}
                onChange={e => setPrUrl(e.target.value)}
                placeholder="https://github.com/.../pull/123 ou GitLab MR"
                className="w-full pl-7 pr-2.5 py-1.5 text-xs rounded-xl bg-[var(--bg-primary)] border border-[var(--border-color)] text-[var(--text-primary)] font-mono text-[11px] focus:outline-none focus:border-[var(--accent-color)]"
              />
              <GitPullRequest size={12} className={`absolute left-2.5 top-2.5 ${prUrl.includes('gitlab') ? 'text-orange-400' : 'text-purple-400'}`} />
            </div>
          </div>
        </div>
      </div>

      {/* Description / Acceptance criteria */}
      <div>
        <label className="block text-[10px] font-bold uppercase tracking-wider text-[var(--text-muted)] mb-1">
          Description & Contexte Technique
        </label>
        <textarea
          rows={4}
          value={description}
          onChange={e => setDescription(e.target.value)}
          placeholder={t.taskModal.descPlaceholder}
          className="w-full px-3.5 py-2.5 text-xs rounded-xl bg-[var(--bg-tertiary)] border border-[var(--border-color)] text-[var(--text-primary)] focus:outline-none focus:border-[var(--accent-color)] leading-relaxed resize-y"
        />
      </div>

      {/* Labels */}
      <div>
        <label className="block text-[10px] font-bold uppercase tracking-wider text-[var(--text-muted)] mb-1">
          {t.taskModal.labels}
        </label>
        <div className="flex flex-wrap items-center gap-1.5 p-2 rounded-xl bg-[var(--bg-tertiary)] border border-[var(--border-color)]">
          {labels.map(lbl => {
            const lower = lbl.toLowerCase()
            let badgeStyle = 'bg-[var(--accent-light)] accent-text'
            if (lower === 'new') badgeStyle = 'bg-cyan-500/20 text-cyan-300 border border-cyan-500/40 font-bold'
            else if (lower === 'clarified') badgeStyle = 'bg-amber-500/20 text-amber-300 border border-amber-500/40 font-bold'
            else if (lower === 'specified') badgeStyle = 'bg-blue-500/20 text-blue-300 border border-blue-500/40 font-bold'
            else if (lower === 'implemented') badgeStyle = 'bg-indigo-500/20 text-indigo-300 border border-indigo-500/40 font-bold'
            else if (lower === 'reviewed') badgeStyle = 'bg-emerald-500/20 text-emerald-300 border border-emerald-500/40 font-bold'

            return (
              <span
                key={lbl}
                className={`inline-flex items-center gap-1 text-[11px] px-2 py-0.5 rounded-md ${badgeStyle}`}
              >
                #{lbl}
                <button
                  type="button"
                  onClick={() => handleRemoveLabel(lbl)}
                  className="hover:opacity-75 ml-0.5"
                >
                  <X size={11} />
                </button>
              </span>
            )
          })}
          <div className="flex items-center gap-1 min-w-[120px] flex-1">
            <input
              type="text"
              value={newLabelInput}
              onChange={e => setNewLabelInput(e.target.value)}
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

      {/* Interactive Clarification Section directly in Info Tab */}
      {renderClarificationQASection()}

      {/* Technical Specification (Speckit) Section in Info Tab */}
      {renderSpecificationSection()}
    </div>
  )

  // SHARED: Skills & Agent Copilot Section Content
  const renderSkillsCopilotSection = () => (
    <div className="space-y-4 pt-4 border-t border-[var(--border-color)]">
      {/* Copilot Header / Engine Banner */}
      <div className="flex items-center justify-between bg-[var(--bg-tertiary)]/50 p-3 rounded-xl border border-[var(--border-color)]">
        <div className="flex items-center gap-2">
          <div className="w-7 h-7 rounded-lg accent-bg text-white flex items-center justify-center font-bold">
            <Bot size={15} />
          </div>
          <div>
            <div className="text-xs font-bold text-[var(--text-primary)] flex items-center gap-1.5">
              <span>Agent Copilot ({settings.aiProvider.toUpperCase()})</span>
              <span className="w-2 h-2 rounded-full bg-emerald-400 animate-pulse"></span>
            </div>
            <div className="text-[10px] text-[var(--text-muted)] font-mono truncate max-w-[280px]">
              {settings.repoPath || 'Workspace standard'}
            </div>
          </div>
        </div>

        {/* Auto-Pilot Trigger */}
        <button
          onClick={() => handleTriggerSkill('pick')}
          disabled={isSkillRunning}
          className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-bold text-white bg-linear-to-r from-amber-500 to-orange-500 shadow hover:opacity-90 active:scale-95 transition-all disabled:opacity-50"
          title="Détecter automatiquement l'état et exécuter la compétence idéale"
        >
          <Sparkles size={13} className="text-amber-200" />
          <span>{t.skills.autoPilot}</span>
        </button>
      </div>

      {/* Pipeline Stepper (Clarify -> Specify -> Implement -> PR) */}
      <div className="space-y-2">
        <label className="block text-[10px] font-bold uppercase tracking-wider text-[var(--text-muted)]">
          Pipeline d'Avancement des Skills
        </label>
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-2">
          {skills
            .filter(s => s.id !== 'pick')
            .map((s, index) => {
              const isRecommended = nextSkill?.id === s.id
              const isCurrentRunning = isSkillRunning && runningSkillId === s.id

              return (
                <button
                  key={s.id}
                  onClick={() => handleTriggerSkill(s.id)}
                  disabled={isSkillRunning}
                  className={`p-2.5 rounded-xl border text-left flex flex-col justify-between transition-all group relative overflow-hidden ${
                    isRecommended
                      ? 'bg-[var(--accent-light)] border-[var(--accent-color)] accent-text ring-2 ring-[var(--accent-glow)]'
                      : 'bg-[var(--bg-tertiary)] border-[var(--border-color)] text-[var(--text-secondary)] hover:border-[var(--accent-color)]/60'
                  }`}
                >
                  <div className="flex items-center justify-between mb-1.5">
                    <span className="text-[10px] font-mono font-bold opacity-60">
                      0{index + 1}
                    </span>
                    {isCurrentRunning ? (
                      <Loader2 size={13} className="animate-spin text-[var(--accent-color)]" />
                    ) : (
                      getSkillIcon(s.icon)
                    )}
                  </div>
                  <div>
                    <div className="font-bold text-xs leading-tight text-[var(--text-primary)] group-hover:text-[var(--accent-color)]">
                      {s.name}
                    </div>
                    <div className="text-[9px] text-[var(--text-muted)] font-mono mt-0.5">
                      {s.command}
                    </div>
                  </div>
                </button>
              )
            })}
        </div>
      </div>

      {/* Main Recommended Action Callout */}
      {nextSkill && (
        <div className="flex items-center justify-between p-3 rounded-xl bg-linear-to-r from-[var(--accent-light)] to-[var(--bg-tertiary)] border border-[var(--accent-color)]/40 shadow-xs">
          <div className="flex items-center gap-2">
            <span className="text-amber-400 animate-bounce">⚡</span>
            <div>
              <div className="text-xs font-bold text-[var(--text-primary)]">
                Étape recommandée : {nextSkill.name}
              </div>
              <div className="text-[10px] text-[var(--text-muted)]">
                {nextSkill.description}
              </div>
            </div>
          </div>
          <button
            onClick={() => handleTriggerSkill(nextSkill.id)}
            disabled={isSkillRunning}
            className="flex items-center gap-1.5 px-4 py-2 rounded-xl text-xs font-bold text-white accent-bg shadow-md hover:opacity-90 active:scale-95 transition-all disabled:opacity-50"
          >
            {isSkillRunning && runningSkillId === nextSkill.id ? (
              <>
                <Loader2 size={13} className="animate-spin" />
                <span>Exécution {settings.aiProvider}...</span>
              </>
            ) : (
              <>
                <Sparkles size={13} className="text-amber-300" />
                <span>Lancer {nextSkill.name}</span>
              </>
            )}
          </button>
        </div>
      )}

      {/* Optional Prompt Refinement */}
      <div>
        <div className="flex items-center justify-between mb-1">
          <label className="text-[10px] font-bold uppercase tracking-wider text-[var(--text-muted)]">
            Instruction / Contexte additionnel pour l'IA (Optionnel)
          </label>
        </div>
        <input
          type="text"
          value={customPrompt}
          onChange={e => setCustomPrompt(e.target.value)}
          placeholder="Ex: Utilise Tailwind v4, ajoute des tests Go avec testify..."
          className="w-full px-3 py-1.5 text-xs rounded-xl bg-[var(--bg-tertiary)] border border-[var(--border-color)] text-[var(--text-primary)] focus:outline-none focus:border-[var(--accent-color)] font-mono text-[11px]"
        />
      </div>

      {/* Latest Output / Console Display */}
      {latestActivity && (
        <div className="space-y-1.5">
          <div className="flex items-center justify-between text-[11px]">
            <span className="font-bold uppercase tracking-wider text-[var(--text-muted)] flex items-center gap-1 text-[10px]">
              <Terminal size={12} className="text-[var(--accent-color)]" />
              Dernière sortie ({latestActivity.skillName})
            </span>
            <span className="text-[10px] text-[var(--text-muted)]">
              {new Date(latestActivity.createdAt).toLocaleTimeString()}
            </span>
          </div>
          <div className="p-3.5 rounded-xl bg-slate-950 text-slate-200 border border-slate-800 font-mono text-xs space-y-2 max-h-56 overflow-y-auto leading-relaxed shadow-inner">
            <div className="flex items-center gap-2 text-emerald-400 font-bold text-[11px] pb-1 border-b border-slate-800">
              <CheckCircle2 size={13} />
              <span>{latestActivity.summary}</span>
            </div>
            {latestActivity.output && (
              <pre className="whitespace-pre-wrap text-[11px] text-slate-300 font-mono">
                {latestActivity.output}
              </pre>
            )}
          </div>
        </div>
      )}

      {/* Interactive Clarification Q&A / Inputs Box */}
      {renderClarificationQASection()}

      {/* Past Activity Accordion if multiple runs */}
      {activities.length > 1 && (
        <div className="space-y-1.5 pt-2">
          <span className="text-[10px] font-bold uppercase tracking-wider text-[var(--text-muted)] flex items-center gap-1">
            <History size={12} />
            Historique des exécutions ({activities.length})
          </span>
          <div className="space-y-1.5">
            {activities.slice(1, 4).map(act => (
              <div
                key={act.id}
                className="p-2 rounded-lg bg-[var(--bg-tertiary)]/60 border border-[var(--border-color)] text-xs flex items-center justify-between"
              >
                <div className="flex items-center gap-2 truncate">
                  <span className="w-1.5 h-1.5 rounded-full bg-emerald-400"></span>
                  <span className="font-bold text-[11px] text-[var(--text-primary)]">{act.skillName}</span>
                  <span className="text-[10px] text-[var(--text-muted)] truncate max-w-[200px]">{act.summary}</span>
                </div>
                <span className="text-[10px] text-[var(--text-muted)] font-mono">
                  {new Date(act.createdAt).toLocaleDateString(undefined, { hour: '2-digit', minute: '2-digit' })}
                </span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )

  // -------------------------------------------------------------
  // MODE 1: RIGHT SLIDING PANEL (DRAWER)
  // Layout: Story infos on top, Skills & Copilot underneath!
  // -------------------------------------------------------------
  if (detailMode === 'panel') {
    return (
      <div className="fixed inset-0 z-50 overflow-hidden select-none">
        {/* Backdrop overlay */}
        <div
          onClick={() => setSelectedTask(null)}
          className="absolute inset-0 bg-black/40 backdrop-blur-2xs animate-in fade-in duration-200"
        />

        {/* Sliding Panel */}
        <div className="absolute inset-y-0 right-0 max-w-full flex pl-10">
          <div className="w-screen max-w-2xl bg-[var(--bg-secondary)] border-l border-[var(--border-color)] shadow-2xl flex flex-col h-full animate-in slide-in-from-right duration-200">
            {/* Panel Header */}
            <div className="flex items-center justify-between px-6 py-3.5 border-b border-[var(--border-color)] bg-[var(--bg-tertiary)]/40 shrink-0">
              {/* Left: Key & External Link */}
              <div className="flex items-center gap-2.5 min-w-0">
                <span className="font-mono text-sm font-bold text-[var(--accent-color)] bg-[var(--accent-light)] px-2.5 py-1 rounded-lg flex items-center gap-1.5 shrink-0">
                  {selectedTask.source === 'linear' && <span className="text-indigo-400 font-bold font-mono">◆</span>}
                  {selectedTask.source === 'github' && <FolderGit2 size={13} className="text-purple-400" />}
                  {(!selectedTask.source || selectedTask.source === 'local') && <Folder size={13} className="text-emerald-400" />}
                  {selectedTask.key}
                </span>

                {selectedTask.externalUrl && (
                  <a
                    href={selectedTask.externalUrl}
                    target="_blank"
                    rel="noreferrer"
                    className={`flex items-center gap-1.5 px-2.5 py-1 rounded-lg text-xs font-bold transition-all shadow-2xs hover:scale-105 ${
                      selectedTask.source === 'linear'
                        ? 'text-indigo-400 bg-indigo-500/15 hover:bg-indigo-500/25 border border-indigo-500/30'
                        : selectedTask.source === 'github'
                        ? 'text-purple-400 bg-purple-500/15 hover:bg-purple-500/25 border border-purple-500/30'
                        : 'text-[var(--accent-color)] bg-[var(--accent-light)]'
                    }`}
                    title="Ouvrir dans Linear/GitHub"
                  >
                    <span>{selectedTask.source === 'linear' ? 'Linear' : selectedTask.source === 'github' ? 'GitHub' : 'Ouvrir'}</span>
                    <ExternalLink size={11} />
                  </a>
                )}
              </div>

              {/* Right: Quick switcher to Modal, PR Link, Delete, Close */}
              <div className="flex items-center gap-1.5 shrink-0">
                {/* Switch to Modal Button */}
                <button
                  type="button"
                  onClick={handleToggleDetailMode}
                  className="flex items-center gap-1 px-2 py-1 rounded-lg text-[11px] font-medium text-[var(--text-secondary)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-tertiary)] border border-[var(--border-color)] transition-colors"
                  title="Afficher en modale centrée"
                >
                  <Square size={13} className="text-purple-400" />
                  <span className="hidden sm:inline">Modale</span>
                </button>

                {selectedTask.prUrl && (
                  <a
                    href={selectedTask.prUrl}
                    target="_blank"
                    rel="noreferrer"
                    className="flex items-center gap-1 px-2.5 py-1 rounded-lg text-xs font-semibold text-purple-400 bg-purple-500/10 hover:bg-purple-500/20 border border-purple-500/30 transition-colors"
                    title={t.skills.viewPr}
                  >
                    <ExternalLink size={12} />
                    <span>PR</span>
                  </a>
                )}

                <button
                  onClick={handleDelete}
                  className="p-1.5 rounded-lg text-rose-400 hover:bg-rose-500/10 transition-colors"
                  title={t.taskModal.delete}
                >
                  <Trash2 size={15} />
                </button>

                <button
                  onClick={() => setSelectedTask(null)}
                  className="p-1.5 rounded-lg text-[var(--text-muted)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-tertiary)] transition-colors"
                >
                  <X size={17} />
                </button>
              </div>
            </div>

            {/* Panel Scrollable Content Body */}
            <div className="p-6 overflow-y-auto space-y-6 flex-1 text-xs">
              {/* 1. TOP: Story Infos */}
              {renderStoryInfoSection()}

              {/* 2. BOTTOM: Skills & Copilot Pipeline */}
              {renderSkillsCopilotSection()}
            </div>

            {/* Panel Sticky Footer */}
            <div className="flex items-center justify-between px-6 py-3.5 border-t border-[var(--border-color)] bg-[var(--bg-tertiary)]/40 shrink-0">
              <span className="text-[11px] text-[var(--text-muted)]">
                {t.taskModal.created} {new Date(selectedTask.createdAt).toLocaleDateString()}
              </span>
              <div className="flex items-center gap-2">
                <button
                  type="button"
                  onClick={() => setSelectedTask(null)}
                  className="px-3.5 py-1.5 rounded-xl text-xs font-medium text-[var(--text-secondary)] hover:bg-[var(--bg-tertiary)] transition-colors"
                >
                  {t.taskModal.cancel}
                </button>
                <button
                  type="button"
                  onClick={handleSave}
                  disabled={isSaving}
                  className="px-4 py-1.5 rounded-xl text-xs font-bold text-white accent-bg shadow hover:opacity-90 active:scale-95 flex items-center gap-1.5 transition-all"
                >
                  <Save size={13} />
                  <span>{isSaving ? 'Enregistrement...' : t.taskModal.save}</span>
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    )
  }

  // -------------------------------------------------------------
  // MODE 2: CENTERED MODAL DIALOG
  // With tab navigation & switcher back to right panel
  // -------------------------------------------------------------
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-xs animate-in fade-in duration-200 select-none">
      <div className="relative w-full max-w-3xl rounded-2xl bg-[var(--bg-secondary)] border border-[var(--border-color)] shadow-2xl overflow-hidden flex flex-col max-h-[92vh]">
        {/* Modal Header */}
        <div className="flex items-center justify-between px-6 py-3.5 border-b border-[var(--border-color)] bg-[var(--bg-tertiary)]/30 shrink-0">
          <div className="flex items-center gap-3">
            <span className="font-mono text-sm font-bold text-[var(--accent-color)] bg-[var(--accent-light)] px-2.5 py-1 rounded-lg flex items-center gap-1.5">
              {selectedTask.source === 'linear' && <span className="text-indigo-400 font-bold font-mono">◆</span>}
              {selectedTask.source === 'github' && <FolderGit2 size={13} className="text-purple-400" />}
              {(!selectedTask.source || selectedTask.source === 'local') && <Folder size={13} className="text-emerald-400" />}
              {selectedTask.key}
            </span>
            <div className="flex items-center gap-2 text-xs text-[var(--text-muted)]">
              {selectedTask.externalUrl && (
                <a
                  href={selectedTask.externalUrl}
                  target="_blank"
                  rel="noreferrer"
                  className="flex items-center gap-1 text-[11px] font-semibold text-[var(--accent-color)] hover:underline"
                  title="Ouvrir dans Linear/GitHub"
                >
                  <ExternalLink size={12} />
                  <span>{selectedTask.source === 'linear' ? 'Linear' : selectedTask.source === 'github' ? 'GitHub' : 'Ouvrir'}</span>
                </a>
              )}
            </div>
          </div>

          <div className="flex items-center gap-2">
            {/* Switch to Right Panel Button */}
            <button
              type="button"
              onClick={handleToggleDetailMode}
              className="flex items-center gap-1 px-2.5 py-1 rounded-lg text-xs font-medium text-[var(--text-secondary)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-tertiary)] border border-[var(--border-color)] transition-colors"
              title="Afficher en panneau latéral droit"
            >
              <PanelRight size={14} className="text-indigo-400" />
              <span className="hidden sm:inline">Panneau droit</span>
            </button>

            {selectedTask.prUrl && (
              <a
                href={selectedTask.prUrl}
                target="_blank"
                rel="noreferrer"
                className="flex items-center gap-1 px-2.5 py-1 rounded-lg text-xs font-semibold text-purple-400 bg-purple-500/10 hover:bg-purple-500/20 border border-purple-500/30 transition-colors"
                title={t.skills.viewPr}
              >
                <ExternalLink size={13} />
                <span>PR</span>
              </a>
            )}

            <button
              onClick={handleDelete}
              className="p-1.5 rounded-lg text-rose-400 hover:bg-rose-500/10 transition-colors"
              title={t.taskModal.delete}
            >
              <Trash2 size={16} />
            </button>

            <button
              onClick={() => setSelectedTask(null)}
              className="p-1.5 rounded-lg text-[var(--text-muted)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-tertiary)] transition-colors"
            >
              <X size={18} />
            </button>
          </div>
        </div>

        {/* Tab Navigation Header */}
        <div className="flex items-center justify-between px-6 pt-3 border-b border-[var(--border-color)] bg-[var(--bg-secondary)] shrink-0">
          <div className="flex items-center gap-4 text-xs font-semibold">
            <button
              onClick={() => setActiveTab('details')}
              className={`pb-2.5 flex items-center gap-1.5 border-b-2 transition-all ${
                activeTab === 'details'
                  ? 'border-[var(--accent-color)] accent-text font-bold'
                  : 'border-transparent text-[var(--text-muted)] hover:text-[var(--text-primary)]'
              }`}
            >
              <FileCode size={14} />
              <span>Infos de la Story</span>
            </button>

            <button
              onClick={() => setActiveTab('skills')}
              className={`pb-2.5 flex items-center gap-1.5 border-b-2 transition-all ${
                activeTab === 'skills'
                  ? 'border-[var(--accent-color)] accent-text font-bold'
                  : 'border-transparent text-[var(--text-muted)] hover:text-[var(--text-primary)]'
              }`}
            >
              <Sparkles size={14} className="text-amber-400" />
              <span>Skills & Agent Copilot</span>
            </button>
          </div>
        </div>

        {/* Modal Scrollable Body */}
        <div className="p-6 overflow-y-auto space-y-6 flex-1 text-xs">
          {activeTab === 'details' && renderStoryInfoSection()}
          {activeTab === 'skills' && renderSkillsCopilotSection()}
        </div>

        {/* Modal Footer */}
        <div className="flex items-center justify-between px-6 py-4 border-t border-[var(--border-color)] bg-[var(--bg-tertiary)]/30 shrink-0">
          <span className="text-[11px] text-[var(--text-muted)]">
            {t.taskModal.created} {new Date(selectedTask.createdAt).toLocaleDateString()}
          </span>
          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={() => setSelectedTask(null)}
              className="px-4 py-2 rounded-xl text-xs font-medium text-[var(--text-secondary)] hover:bg-[var(--bg-tertiary)] transition-colors"
            >
              {t.taskModal.cancel}
            </button>
            <button
              type="button"
              onClick={handleSave}
              disabled={isSaving}
              className="px-5 py-2 rounded-xl text-xs font-bold text-white accent-bg shadow-md hover:opacity-90 active:scale-95 flex items-center gap-1.5 transition-all"
            >
              <Save size={14} />
              <span>{isSaving ? 'Enregistrement...' : t.taskModal.save}</span>
            </button>
          </div>
        </div>
      </div>

      {/* Fullscreen Expanded Q&A Alignment Modal */}
      {isExpandedQA && (
        <div className="fixed inset-0 z-60 flex items-center justify-center p-3 sm:p-6 bg-black/80 backdrop-blur-xs animate-in fade-in duration-150">
          <div className="relative w-full max-w-5xl h-[88vh] rounded-2xl bg-[var(--bg-secondary)] border border-amber-500/40 shadow-2xl flex flex-col overflow-hidden">
            {/* Header */}
            <div className="flex items-center justify-between px-6 py-4 border-b border-[var(--border-color)] bg-[var(--bg-tertiary)]/70 shrink-0">
              <div className="flex items-center gap-2.5">
                <div className="w-8 h-8 rounded-xl bg-amber-500/20 text-amber-400 border border-amber-500/30 flex items-center justify-center font-bold">
                  <MessageSquare size={16} />
                </div>
                <div>
                  <div className="flex items-center gap-2">
                    <span className="font-mono text-xs font-bold text-amber-400 bg-amber-500/10 px-2 py-0.5 rounded border border-amber-500/20">
                      {selectedTask.key}
                    </span>
                    <h3 className="text-sm font-bold text-[var(--text-primary)]">
                      Alignement & Réponses aux questions de cadrage (Mode Grand Format)
                    </h3>
                  </div>
                  <p className="text-[11px] text-[var(--text-muted)] mt-0.5">
                    {selectedTask.title}
                  </p>
                </div>
              </div>

              <div className="flex items-center gap-2">
                <button
                  type="button"
                  onClick={() => setIsExpandedQA(false)}
                  className="p-2 rounded-xl hover:bg-[var(--bg-tertiary)] text-[var(--text-muted)] hover:text-[var(--text-primary)] transition-colors"
                  title="Réduire (ESC)"
                >
                  <Minimize2 size={16} />
                </button>
                <button
                  type="button"
                  onClick={() => setIsExpandedQA(false)}
                  className="p-2 rounded-xl hover:bg-[var(--bg-tertiary)] text-[var(--text-muted)] hover:text-[var(--text-primary)] transition-colors"
                  title="Fermer"
                >
                  <X size={18} />
                </button>
              </div>
            </div>

            {/* Split Content: Questions on Left / Answers on Right */}
            <div className="flex-1 flex flex-col sm:flex-row min-h-0 divide-y sm:divide-y-0 sm:divide-x divide-[var(--border-color)] overflow-hidden">
              {/* Left Column: Questions */}
              <div className="sm:w-1/2 flex flex-col p-4 bg-slate-950 text-slate-300 overflow-hidden">
                <div className="flex items-center justify-between pb-2 border-b border-slate-800 shrink-0">
                  <span className="text-[10px] font-bold text-amber-400 uppercase tracking-wider flex items-center gap-1.5">
                    <HelpCircle size={13} />
                    Questions posées par l'agent Clarify
                  </span>
                  {clarifyActivity?.output && (
                    <button
                      type="button"
                      onClick={() => {
                        navigator.clipboard?.writeText(clarifyActivity.output)
                        addToast({ type: 'success', title: 'Questions copiées', description: 'Copié dans le presse-papiers.' })
                      }}
                      className="text-[10px] text-slate-400 hover:text-white flex items-center gap-1 font-mono"
                    >
                      <Copy size={11} />
                      <span>Copier</span>
                    </button>
                  )}
                </div>
                <div className="flex-1 overflow-y-auto pt-3 font-mono text-xs leading-relaxed whitespace-pre-wrap">
                  {clarifyActivity?.output || "Aucune question générée par l'agent Clarify pour l'instant. Vous pouvez renseigner directement vos directives de cadrage ci-contre."}
                </div>
              </div>

              {/* Right Column: User Answer Editor */}
              <div className="sm:w-1/2 flex flex-col p-4 bg-[var(--bg-primary)] overflow-hidden">
                <div className="flex items-center justify-between pb-2 border-b border-[var(--border-color)] shrink-0">
                  <span className="text-[10px] font-bold uppercase tracking-wider text-[var(--text-primary)] flex items-center gap-1.5">
                    <MessageSquare size={13} className="text-amber-400" />
                    Vos Arbitrages & Réponses Techniques
                  </span>
                  <span className="text-[10px] font-mono text-[var(--text-muted)]">
                    {clarificationInput.length} caractères
                  </span>
                </div>

                <div className="flex-1 py-3 flex flex-col min-h-0">
                  <textarea
                    value={clarificationInput}
                    onChange={e => setClarificationInput(e.target.value)}
                    placeholder={`Saisissez vos réponses détaillées ici...\n\nExemple :\n1. Architecture : Stateless avec pool de connexions optimisé\n2. Timeout : 30 secondes pour libérer les connexions inactives\n3. Sécurité : Validation des tokens et rate-limiting...`}
                    className="w-full flex-1 p-3.5 text-xs font-mono rounded-xl bg-[var(--bg-secondary)] border border-[var(--border-color)] text-[var(--text-primary)] placeholder-[var(--text-muted)]/60 focus:outline-none focus:border-[var(--accent-color)] leading-relaxed resize-none"
                  />
                </div>

                {/* Actions Footer */}
                <div className="flex flex-wrap items-center justify-between gap-2 pt-2 border-t border-[var(--border-color)] shrink-0">
                  <div className="flex items-center gap-2">
                    <button
                      type="button"
                      onClick={handleSaveAnswersToDescription}
                      disabled={!clarificationInput.trim() || isEnrichingDesc}
                      className="flex items-center gap-1.5 px-3 py-1.5 rounded-xl text-xs font-semibold bg-[var(--bg-tertiary)] hover:bg-[var(--border-color)] text-[var(--text-secondary)] hover:text-[var(--text-primary)] border border-[var(--border-color)] transition-all disabled:opacity-40"
                    >
                      {isEnrichingDesc ? <Loader2 size={12} className="animate-spin" /> : <Save size={12} />}
                      <span>Enrichir description</span>
                    </button>

                    <button
                      type="button"
                      onClick={handlePostAnswersComment}
                      disabled={!clarificationInput.trim() || isPostingComment}
                      className="flex items-center gap-1.5 px-3 py-1.5 rounded-xl text-xs font-semibold bg-[var(--bg-tertiary)] hover:bg-[var(--border-color)] text-[var(--text-secondary)] hover:text-[var(--text-primary)] border border-[var(--border-color)] transition-all disabled:opacity-40"
                    >
                      {isPostingComment ? <Loader2 size={12} className="animate-spin" /> : <Send size={12} />}
                      <span>Commenter ({selectedTask.source === 'linear' ? 'Linear' : 'GitHub'})</span>
                    </button>
                  </div>

                  <button
                    type="button"
                    onClick={async () => {
                      setIsExpandedQA(false)
                      await handleSaveAndSpecify()
                    }}
                    disabled={isSkillRunning}
                    className="flex items-center gap-2 px-4 py-2 rounded-xl text-xs font-bold text-white shadow-md accent-bg hover:opacity-90 active:scale-95 transition-all disabled:opacity-50"
                  >
                    {isSkillRunning && runningSkillId === 'specify' ? (
                      <Loader2 size={13} className="animate-spin" />
                    ) : (
                      <>
                        <Sparkles size={13} className="text-amber-300" />
                        <span>Enregistrer & Lancer Specify</span>
                        <ArrowRight size={13} />
                      </>
                    )}
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Fullscreen Expanded Specification Reader Modal */}
      {isExpandedSpec && specifyActivity?.output && (
        <div className="fixed inset-0 z-60 flex items-center justify-center p-3 sm:p-6 bg-black/80 backdrop-blur-xs animate-in fade-in duration-150">
          <div className="relative w-full max-w-5xl h-[88vh] rounded-2xl bg-[var(--bg-secondary)] border border-blue-500/40 shadow-2xl flex flex-col overflow-hidden">
            {/* Header */}
            <div className="flex items-center justify-between px-6 py-4 border-b border-[var(--border-color)] bg-[var(--bg-tertiary)]/70 shrink-0">
              <div className="flex items-center gap-2.5">
                <div className="w-8 h-8 rounded-xl bg-blue-500/20 text-blue-400 border border-blue-500/30 flex items-center justify-center font-bold">
                  <FileCode size={16} />
                </div>
                <div>
                  <div className="flex items-center gap-2">
                    <span className="font-mono text-xs font-bold text-blue-400 bg-blue-500/10 px-2 py-0.5 rounded border border-blue-500/20">
                      {selectedTask.key}
                    </span>
                    <h3 className="text-sm font-bold text-[var(--text-primary)]">
                      Spécification Technique Speckit (Vue Détaillée)
                    </h3>
                  </div>
                  <p className="text-[11px] text-[var(--text-muted)] mt-0.5">
                    {selectedTask.title}
                  </p>
                </div>
              </div>

              <div className="flex items-center gap-2">
                <button
                  type="button"
                  onClick={handleCopySpec}
                  className="flex items-center gap-1.5 px-3 py-1.5 rounded-xl text-xs font-semibold bg-[var(--bg-primary)] hover:bg-[var(--bg-tertiary)] text-[var(--text-secondary)] border border-[var(--border-color)] transition-colors"
                >
                  {copiedSpec ? <Check size={13} className="text-emerald-400" /> : <Copy size={13} />}
                  <span>{copiedSpec ? 'Copié !' : 'Copier spec'}</span>
                </button>
                <button
                  type="button"
                  onClick={() => setIsExpandedSpec(false)}
                  className="p-2 rounded-xl hover:bg-[var(--bg-tertiary)] text-[var(--text-muted)] hover:text-[var(--text-primary)] transition-colors"
                  title="Réduire (ESC)"
                >
                  <Minimize2 size={16} />
                </button>
                <button
                  type="button"
                  onClick={() => setIsExpandedSpec(false)}
                  className="p-2 rounded-xl hover:bg-[var(--bg-tertiary)] text-[var(--text-muted)] hover:text-[var(--text-primary)] transition-colors"
                  title="Fermer"
                >
                  <X size={18} />
                </button>
              </div>
            </div>

            {/* Spec Body */}
            <div className="flex-1 p-6 overflow-y-auto bg-slate-950 text-slate-200 font-mono text-xs leading-relaxed whitespace-pre-wrap select-text">
              {specifyActivity.output}
            </div>

            {/* Footer */}
            <div className="flex items-center justify-between px-6 py-3.5 border-t border-[var(--border-color)] bg-[var(--bg-tertiary)]/50 shrink-0">
              <span className="text-xs text-[var(--text-muted)] font-mono">
                {specifyActivity.completedAt ? `Généré le ${new Date(specifyActivity.completedAt).toLocaleString()}` : ''}
              </span>

              <button
                type="button"
                onClick={() => {
                  setIsExpandedSpec(false)
                  handleTriggerSkill('implement')
                }}
                disabled={isSkillRunning}
                className="flex items-center gap-2 px-4 py-2 rounded-xl text-xs font-bold text-white shadow-md bg-linear-to-r from-blue-600 to-indigo-600 hover:opacity-90 active:scale-95 transition-all disabled:opacity-50"
              >
                {isSkillRunning && runningSkillId === 'implement' ? (
                  <Loader2 size={13} className="animate-spin" />
                ) : (
                  <>
                    <Flame size={14} className="text-amber-300" />
                    <span>Lancer Implement Code</span>
                    <ArrowRight size={13} />
                  </>
                )}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
