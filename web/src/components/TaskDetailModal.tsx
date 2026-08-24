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
import { InteractiveTerminal } from './InteractiveTerminal'

export const TaskDetailModal: React.FC = () => {
  const {
    selectedTask,
    setSelectedTask,
    setChatTask,
    setDiffTask,
    updateTask,
    deleteTask,
    runSkill,
    isSkillRunning,
    runningSkillId,
    skills,
    projects,
    activities: globalActivities,
    gitStatus,
    checkoutTaskBranch,
    switchGitBranch,
    settings,
    updateSettings,
    openInEditor,
    addToast,
    t,
  } = useApp()

  const [isSwitchingBranch, setIsSwitchingBranch] = useState(false)

  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [status, setStatus] = useState<Status>('backlog')
  const [priority, setPriority] = useState<Priority>('medium')
  const [taskProjectId, setTaskProjectId] = useState<string>(projects[0]?.id || '')
  const [branchName, setBranchName] = useState('')
  const [prUrl, setPrUrl] = useState('')
  const [labels, setLabels] = useState<string[]>([])
  const [newLabelInput, setNewLabelInput] = useState('')
  const [assignee, setAssignee] = useState('')
  const [dueDate, setDueDate] = useState('')
  const [isSaving, setIsSaving] = useState(false)
  const [activeTab, setActiveTab] = useState<'details' | 'cadrage' | 'skills' | 'history'>('details')
  const [customPrompt, setCustomPrompt] = useState('')
  const [copiedBranch, setCopiedBranch] = useState(false)
  const [specFramework, setSpecFramework] = useState<'speckit' | 'openfeature'>(settings.specFramework || 'speckit')
  const [isExpandedSpec, setIsExpandedSpec] = useState(false)
  const [copiedSpec, setCopiedSpec] = useState(false)
  const [isMaximized, setIsMaximized] = useState(false)
  const [isTtyOpen, setIsTtyOpen] = useState(false)
  const [isTtyExpanded, setIsTtyExpanded] = useState(false)
  const [ttyCommand, setTtyCommand] = useState('')

  const detailMode: DetailMode = settings.detailMode || 'panel'

  useEffect(() => {
    if (selectedTask) {
      setTitle(selectedTask.title)
      setDescription(selectedTask.description || '')
      setStatus(selectedTask.status)
      setPriority(selectedTask.priority)
      setTaskProjectId(selectedTask.projectId || projects[0]?.id || '')
      setBranchName(selectedTask.branchName || '')
      setPrUrl(selectedTask.prUrl || '')
      setLabels(selectedTask.labels || [])
      setAssignee(selectedTask.assignee || '')
      setDueDate(selectedTask.dueDate || '')
      setSpecFramework(settings.specFramework || 'speckit')
    }
  }, [selectedTask, projects, settings.specFramework])

  const currentTaskProject = projects.find(p => p.id === (selectedTask?.projectId || taskProjectId))
  const targetGithubRepo = (currentTaskProject?.githubRepo || settings.githubRepo || '').replace(/^https?:\/\/github\.com\//, '').replace(/\.git$/, '')
  const externalUrl = selectedTask?.externalUrl || (
    selectedTask?.source === 'github' && targetGithubRepo && selectedTask?.key?.startsWith('#')
      ? `https://github.com/${targetGithubRepo}/issues/${selectedTask.key.replace('#', '')}`
      : undefined
  )

  const handleClose = async () => {
    if (selectedTask && title.trim()) {
      const isModified =
        title.trim() !== selectedTask.title ||
        description.trim() !== (selectedTask.description || '').trim() ||
        status !== selectedTask.status ||
        priority !== selectedTask.priority ||
        taskProjectId !== (selectedTask.projectId || '') ||
        branchName.trim() !== (selectedTask.branchName || '').trim() ||
        prUrl.trim() !== (selectedTask.prUrl || '').trim() ||
        assignee.trim() !== (selectedTask.assignee || '').trim() ||
        (dueDate || '') !== (selectedTask.dueDate || '') ||
        JSON.stringify(labels) !== JSON.stringify(selectedTask.labels || [])

      if (isModified) {
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
      }
    }
    setSelectedTask(null)
  }

  useEffect(() => {
    if (!selectedTask) return
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        if (isTtyExpanded) {
          setIsTtyExpanded(false)
        } else if (isExpandedSpec) {
          setIsExpandedSpec(false)
        } else {
          handleClose()
        }
      }
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [selectedTask, isTtyExpanded, isExpandedSpec, title, description, status, priority, taskProjectId, branchName, prUrl, assignee, dueDate, labels])

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

  const handleDelete = async () => {
    if (window.confirm(t.taskModal.deleteConfirm)) {
      await deleteTask(selectedTask.id)
      setSelectedTask(null)
    }
  }

  const handleAddLabel = async () => {
    const clean = newLabelInput.replace(/^#/, '').trim()
    if (clean && !labels.includes(clean)) {
      const nextLabels = [...labels, clean]
      setLabels(nextLabels)
      setNewLabelInput('')
      if (selectedTask) {
        await updateTask(selectedTask.id, { labels: nextLabels })
      }
    }
  }

  const handleRemoveLabel = async (tagToRemove: string) => {
    const nextLabels = labels.filter(l => l !== tagToRemove)
    setLabels(nextLabels)
    if (selectedTask) {
      await updateTask(selectedTask.id, { labels: nextLabels })
    }
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

  const handleTriggerSkill = async (skillId: string, overridePrompt?: string) => {
    if (!selectedTask || isSkillRunning) return
    const promptToUse = overridePrompt || customPrompt
    const activity = await runSkill(selectedTask.id, skillId, promptToUse)
    if (activity && !overridePrompt) {
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

  // Technical Specification (SpecKit / Open Feature) Section Box
  const renderSpecificationSection = () => {
    if (!specifyActivity?.output) return null

    const isOF = specFramework === 'openfeature'
    const frameworkLabel = isOF ? 'Open Feature' : 'SpecKit'

    return (
      <div className="p-4 rounded-2xl bg-linear-to-b from-[var(--bg-tertiary)] to-[var(--bg-secondary)] border border-blue-500/40 shadow-md space-y-3">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <FileCode size={16} className={isOF ? 'text-emerald-400' : 'text-blue-400'} />
            <h4 className="text-xs font-bold text-[var(--text-primary)]">
              Spécification Technique ({frameworkLabel})
            </h4>
          </div>
          <div className="flex items-center gap-2">
            <span className={`text-[10px] px-2 py-0.5 rounded-full font-bold uppercase border ${
              isOF
                ? 'bg-emerald-500/20 text-emerald-300 border-emerald-500/30'
                : 'bg-blue-500/20 text-blue-300 border-blue-500/30'
            }`}>
              {frameworkLabel} SDD
            </span>
            <button
              type="button"
              onClick={handleCopySpec}
              className="inline-flex items-center gap-1 px-2 py-1 rounded-lg text-[10px] font-mono font-bold bg-[var(--bg-primary)] hover:bg-[var(--bg-tertiary)] text-[var(--text-secondary)] border border-[var(--border-color)] transition-colors cursor-pointer"
              title="Copier la spec complète"
            >
              {copiedSpec ? <Check size={11} className="text-emerald-400" /> : <Copy size={11} />}
              <span>{copiedSpec ? 'Copié !' : 'Copier'}</span>
            </button>
            <button
              type="button"
              onClick={() => setIsExpandedSpec(true)}
              className={`inline-flex items-center gap-1 px-2.5 py-1 rounded-lg text-[10px] font-bold transition-all shadow-2xs hover:scale-105 cursor-pointer ${
                isOF
                  ? 'text-emerald-300 bg-emerald-500/15 hover:bg-emerald-500/25 border border-emerald-500/30'
                  : 'text-blue-300 bg-blue-500/15 hover:bg-blue-500/25 border border-blue-500/30'
              }`}
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
            className="flex items-center gap-1.5 px-3.5 py-1.5 rounded-xl text-xs font-bold text-white shadow-md bg-linear-to-r from-blue-600 to-indigo-600 hover:opacity-90 active:scale-95 transition-all disabled:opacity-50 cursor-pointer"
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

  // -------------------------------------------------------------
  // DEDICATED TAB: Cadrage & Spécifications
  // -------------------------------------------------------------
  const renderCadrageSection = () => {
    if (!selectedTask) return null

    const currentProject = projects.find(p => p.id === (selectedTask.projectId || taskProjectId))
    const issueTracker = currentProject?.issueTracker || selectedTask.source || 'github'
    const repoName = currentProject?.githubRepo || currentProject?.name || settings.repoPath?.split('/').pop() || 'repo'
    const provider = currentProject?.aiProvider || settings.aiProvider || 'agy'
    const cmdTemplate = currentProject?.aiCommandTemplate || settings.aiCommandTemplate

    const buildCliCommand = (prompt: string): string => {
      if (cmdTemplate && cmdTemplate.includes('{prompt}')) {
        return cmdTemplate
          .replace('{prompt}', prompt)
          .replace('{issueKey}', selectedTask.key)
          .replace('{issueTitle}', selectedTask.title)
          .replace('{branchName}', selectedTask.branchName || '')
          .replace('{repoPath}', currentProject?.repoPath || settings.repoPath || '')
          .replace('{tracker}', issueTracker)
          .replace('{repo}', repoName)
      }
      if (provider === 'agy') {
        return `agy --dangerously-skip-permissions -p "${prompt}"`
      }
      if (provider === 'claude') {
        return `claude --dangerously-skip-permissions -p "${prompt}"`
      }
      if (provider === 'vibe') {
        return `vibe -p "${prompt}" --auto-approve`
      }
      return `${provider} -p "${prompt}"`
    }

    const clarifyPrompt = `/clarify-issue ${selectedTask.key} tracked on ${issueTracker} in ${repoName}`
    const clarifyCliCommand = buildCliCommand(clarifyPrompt)

    const specifyPrompt = `/specify-issue ${selectedTask.key} --framework ${specFramework}`
    const specifyCliCommand = buildCliCommand(specifyPrompt)

    return (
      <div className="space-y-6">
        {/* Cadrage Header & Stage Banner */}
        <div className="p-4 rounded-2xl bg-linear-to-r from-amber-500/10 via-blue-500/10 to-[var(--bg-tertiary)] border border-[var(--border-color)] flex items-center justify-between gap-4">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl bg-amber-500/20 text-amber-400 border border-amber-500/30 flex items-center justify-center font-bold shrink-0 shadow-2xs">
              <HelpCircle size={20} />
            </div>
            <div>
              <h3 className="text-sm font-bold text-[var(--text-primary)] flex items-center gap-2">
                <span>Cadrage Produit & Spécifications ({specFramework === 'openfeature' ? 'Open Feature' : 'SpecKit'})</span>
                <span className="text-[10px] font-mono font-bold px-2 py-0.5 rounded-full bg-amber-500/20 text-amber-300 border border-amber-500/30">
                  {status === 'to_clarify' || status === 'backlog' ? '#new' : status === 'to_specify' ? '#clarified' : status === 'to_implement' ? '#specified' : `#${status}`}
                </span>
              </h3>
              <p className="text-xs text-[var(--text-muted)] mt-0.5">
                Clarification interactive en console TTY et génération de spécifications formelles (SpecKit ou Open Feature).
              </p>
            </div>
          </div>

          <div className="flex items-center gap-2 shrink-0">
            <span className="text-[10px] font-mono px-2 py-1 rounded-lg bg-[var(--bg-secondary)] border border-[var(--border-color)] text-[var(--text-muted)]">
              {repoName}
            </span>
          </div>
        </div>

        {/* 1. Interactive TTY Cadrage Runner Card */}
        <div className="p-4 rounded-2xl bg-[var(--bg-tertiary)]/70 border border-amber-500/30 shadow-md space-y-3">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <div className="flex items-center gap-2">
              <Terminal size={16} className="text-amber-400" />
              <h4 className="text-xs font-bold text-[var(--text-primary)]">
                1. Cadrage Interactif TTY (/clarify-issue)
              </h4>
            </div>
            <div className="flex items-center gap-2">
              <span className="text-[10px] font-mono px-2 py-0.5 rounded-md bg-amber-500/15 text-amber-300 font-bold border border-amber-500/30">
                {settings.aiProvider?.toUpperCase() || 'AGY'} CLI
              </span>
            </div>
          </div>

          {/* Code block with exact prompt */}
          <div className="p-3 rounded-xl bg-slate-950 border border-slate-800 flex items-center justify-between gap-3 text-xs font-mono">
            <div className="flex items-center gap-2 truncate min-w-0">
              <span className="text-amber-400 font-bold">$</span>
              <span className="text-slate-200 select-all truncate">
                {clarifyPrompt}
              </span>
            </div>
            <div className="flex items-center gap-1.5 shrink-0">
              <button
                type="button"
                onClick={() => {
                  navigator.clipboard?.writeText(clarifyPrompt)
                  addToast({
                    type: 'success',
                    title: 'Prompt copié',
                    description: `Prompt copié : ${clarifyPrompt}`,
                  })
                }}
                className="px-2 py-1 rounded-lg text-[10.5px] font-medium text-slate-300 hover:text-white bg-slate-900 hover:bg-slate-800 border border-slate-700 transition-colors flex items-center gap-1 cursor-pointer"
                title="Copier le prompt"
              >
                <Copy size={11} />
                <span>Copier prompt</span>
              </button>
              <button
                type="button"
                onClick={() => {
                  navigator.clipboard?.writeText(clarifyCliCommand)
                  addToast({
                    type: 'success',
                    title: 'Commande CLI copiée',
                    description: `Commande copiée : ${clarifyCliCommand}`,
                  })
                }}
                className="px-2 py-1 rounded-lg text-[10.5px] font-medium text-slate-300 hover:text-white bg-slate-900 hover:bg-slate-800 border border-slate-700 transition-colors flex items-center gap-1 cursor-pointer"
                title="Copier la commande CLI complète"
              >
                <Code2 size={11} />
                <span>Copier CLI</span>
              </button>
            </div>
          </div>

          {/* Action trigger buttons */}
          <div className="flex flex-wrap items-center justify-between gap-2 pt-1">
            <div className="text-[11px] text-[var(--text-muted)]">
              Pose les questions d'arbitrage et clarifie le périmètre en direct dans la console TTY.
            </div>

            <div className="flex items-center gap-2">
              <button
                type="button"
                onClick={() => {
                  const willOpen = !isTtyOpen || ttyCommand !== clarifyCliCommand
                  setIsTtyOpen(willOpen)
                  setTtyCommand(clarifyCliCommand)
                }}
                className={`flex items-center gap-1.5 px-3.5 py-1.5 rounded-xl text-xs font-bold transition-all cursor-pointer shadow-xs active:scale-95 ${
                  isTtyOpen && ttyCommand === clarifyCliCommand
                    ? 'bg-amber-600 text-white hover:bg-amber-500'
                    : 'bg-amber-500/15 text-amber-300 hover:bg-amber-500/25 border border-amber-500/30'
                }`}
                title="Ouvrir la console TTY interactive intégrée et exécuter /clarify-issue"
              >
                <Terminal size={13} className={isTtyOpen && ttyCommand === clarifyCliCommand ? 'text-white' : 'text-amber-400'} />
                <span>{isTtyOpen && ttyCommand === clarifyCliCommand ? 'Masquer TTY' : 'Lancer Console TTY Cadrage'}</span>
              </button>

              <button
                type="button"
                onClick={() => handleTriggerSkill('clarify', clarifyPrompt)}
                disabled={isSkillRunning}
                className="flex items-center gap-1.5 px-3.5 py-1.5 rounded-xl text-xs font-bold text-white bg-amber-600 hover:bg-amber-500 transition-all shadow-xs cursor-pointer active:scale-95 disabled:opacity-50"
                title="Lancer le prompt /clarify-issue via l'agent en arrière-plan"
              >
                {isSkillRunning && runningSkillId === 'clarify' ? (
                  <Loader2 size={13} className="animate-spin" />
                ) : (
                  <Sparkles size={13} className="text-amber-200" />
                )}
                <span>Lancer en tâche de fond</span>
              </button>
            </div>
          </div>
        </div>

        {/* 2. Spec-Driven Design Framework & Specification Runner Card */}
        <div className="p-4 rounded-2xl bg-[var(--bg-tertiary)]/70 border border-blue-500/30 shadow-md space-y-3.5">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <div className="flex items-center gap-2">
              <FileCode size={16} className="text-blue-400" />
              <h4 className="text-xs font-bold text-[var(--text-primary)]">
                2. Framework Spec-Driven Design & Spécification (/specify-issue)
              </h4>
            </div>

            {/* Framework Toggle Buttons: SpecKit vs Open Feature */}
            <div className="flex items-center gap-1.5 p-1 rounded-xl bg-[var(--bg-secondary)] border border-[var(--border-color)]">
              <button
                type="button"
                onClick={() => setSpecFramework('speckit')}
                className={`px-2.5 py-1 rounded-lg text-xs font-bold transition-all cursor-pointer flex items-center gap-1.5 ${
                  specFramework === 'speckit'
                    ? 'bg-blue-600 text-white shadow-xs'
                    : 'text-[var(--text-secondary)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-tertiary)]'
                }`}
              >
                <span>📑</span>
                <span>SpecKit</span>
              </button>
              <button
                type="button"
                onClick={() => setSpecFramework('openfeature')}
                className={`px-2.5 py-1 rounded-lg text-xs font-bold transition-all cursor-pointer flex items-center gap-1.5 ${
                  specFramework === 'openfeature'
                    ? 'bg-emerald-600 text-white shadow-xs'
                    : 'text-[var(--text-secondary)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-tertiary)]'
                }`}
              >
                <span>🚩</span>
                <span>Open Feature</span>
              </button>
            </div>
          </div>

          <div className="text-[11px] text-[var(--text-muted)] flex items-center gap-2">
            {specFramework === 'openfeature' ? (
              <span>
                <strong className="text-emerald-400">Framework Open Feature :</strong> Spécification axée sur les Feature Flags, contextes d'évaluation, hooks SDK et cycle de vie.
              </span>
            ) : (
              <span>
                <strong className="text-blue-400">Framework SpecKit :</strong> Spécification technique standard avec user stories, architecture et critères BDD (Given/When/Then).
              </span>
            )}
          </div>

          {/* Code block with exact specify prompt */}
          <div className="p-3 rounded-xl bg-slate-950 border border-slate-800 flex items-center justify-between gap-3 text-xs font-mono">
            <div className="flex items-center gap-2 truncate min-w-0">
              <span className={specFramework === 'openfeature' ? 'text-emerald-400 font-bold' : 'text-blue-400 font-bold'}>$</span>
              <span className="text-slate-200 select-all truncate">
                {specifyPrompt}
              </span>
            </div>
            <div className="flex items-center gap-1.5 shrink-0">
              <button
                type="button"
                onClick={() => {
                  navigator.clipboard?.writeText(specifyPrompt)
                  addToast({
                    type: 'success',
                    title: 'Prompt copié',
                    description: `Prompt copié : ${specifyPrompt}`,
                  })
                }}
                className="px-2 py-1 rounded-lg text-[10.5px] font-medium text-slate-300 hover:text-white bg-slate-900 hover:bg-slate-800 border border-slate-700 transition-colors flex items-center gap-1 cursor-pointer"
                title="Copier le prompt specify"
              >
                <Copy size={11} />
                <span>Copier prompt</span>
              </button>
              <button
                type="button"
                onClick={() => {
                  navigator.clipboard?.writeText(specifyCliCommand)
                  addToast({
                    type: 'success',
                    title: 'Commande CLI copiée',
                    description: `Commande copiée : ${specifyCliCommand}`,
                  })
                }}
                className="px-2 py-1 rounded-lg text-[10.5px] font-medium text-slate-300 hover:text-white bg-slate-900 hover:bg-slate-800 border border-slate-700 transition-colors flex items-center gap-1 cursor-pointer"
                title="Copier la commande CLI complète"
              >
                <Code2 size={11} />
                <span>Copier CLI</span>
              </button>
            </div>
          </div>

          {/* Action trigger buttons */}
          <div className="flex flex-wrap items-center justify-between gap-2 pt-1">
            <div className="text-[11px] text-[var(--text-muted)]">
              Génère la spécification formelle selon la norme <span className="font-bold text-[var(--text-primary)]">{specFramework === 'openfeature' ? 'Open Feature' : 'SpecKit'}</span>.
            </div>

            <div className="flex items-center gap-2">
              <button
                type="button"
                onClick={() => {
                  const willOpen = !isTtyOpen || ttyCommand !== specifyCliCommand
                  setIsTtyOpen(willOpen)
                  setTtyCommand(specifyCliCommand)
                }}
                className={`flex items-center gap-1.5 px-3.5 py-1.5 rounded-xl text-xs font-bold transition-all cursor-pointer shadow-xs active:scale-95 ${
                  isTtyOpen && ttyCommand === specifyCliCommand
                    ? specFramework === 'openfeature' ? 'bg-emerald-600 text-white hover:bg-emerald-500' : 'bg-blue-600 text-white hover:bg-blue-500'
                    : specFramework === 'openfeature'
                      ? 'bg-emerald-500/15 text-emerald-300 hover:bg-emerald-500/25 border border-emerald-500/30'
                      : 'bg-blue-500/15 text-blue-300 hover:bg-blue-500/25 border border-blue-500/30'
                }`}
                title="Ouvrir la console TTY interactive intégrée et exécuter /specify-issue"
              >
                <Terminal size={13} />
                <span>{isTtyOpen && ttyCommand === specifyCliCommand ? 'Masquer TTY' : 'Lancer Console TTY Spécifier'}</span>
              </button>

              <button
                type="button"
                onClick={async () => {
                  if (status === 'to_clarify' || status === 'backlog') {
                    await updateTask(selectedTask.id, { status: 'to_specify' })
                    setStatus('to_specify')
                  }
                  await handleTriggerSkill('specify', specifyPrompt)
                }}
                disabled={isSkillRunning}
                className={`flex items-center gap-1.5 px-3.5 py-1.5 rounded-xl text-xs font-bold text-white transition-all shadow-xs cursor-pointer active:scale-95 disabled:opacity-50 ${
                  specFramework === 'openfeature'
                    ? 'bg-emerald-600 hover:bg-emerald-500'
                    : 'bg-blue-600 hover:bg-blue-500'
                }`}
                title="Lancer la génération de spécification en arrière-plan"
              >
                {isSkillRunning && runningSkillId === 'specify' ? (
                  <Loader2 size={13} className="animate-spin" />
                ) : (
                  <Sparkles size={13} className="text-blue-200" />
                )}
                <span>Lancer en tâche de fond</span>
              </button>
            </div>
          </div>
        </div>

        {/* Embedded Interactive Terminal if toggled */}
        {isTtyOpen && !isTtyExpanded && (
          <div className="pt-1">
            <div className="h-96 w-full rounded-xl overflow-hidden border border-slate-800 shadow-xl">
              <InteractiveTerminal
                task={selectedTask}
                isExpanded={false}
                initialCommand={ttyCommand}
                onToggleExpand={() => setIsTtyExpanded(true)}
                onClose={() => setIsTtyOpen(false)}
              />
            </div>
          </div>
        )}

        {/* Fullscreen Expanded Interactive Terminal Modal Overlay */}
        {isTtyOpen && isTtyExpanded && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-3 sm:p-6 bg-black/80 backdrop-blur-md animate-in fade-in duration-150">
            <div className="relative w-full h-[92vh] rounded-2xl overflow-hidden shadow-2xl border border-indigo-500/40 bg-[#070b14] flex flex-col">
              <InteractiveTerminal
                task={selectedTask}
                isExpanded={true}
                initialCommand={ttyCommand}
                onToggleExpand={() => setIsTtyExpanded(false)}
                onClose={() => {
                  setIsTtyExpanded(false)
                  setIsTtyOpen(false)
                }}
              />
            </div>
          </div>
        )}

        {/* 3. Technical Specification (Speckit / Open Feature) Section Box */}
        {renderSpecificationSection()}

        {/* 4. Empty state helper if no clarification or spec yet */}
        {!clarifyActivity && !specifyActivity && (
          <div className="p-8 rounded-2xl border border-dashed border-[var(--border-color)] bg-[var(--bg-tertiary)]/20 text-center space-y-3">
            <div className="w-12 h-12 rounded-full bg-amber-500/10 text-amber-400 border border-amber-500/20 flex items-center justify-center mx-auto">
              <Sparkles size={24} />
            </div>
            <div className="max-w-md mx-auto space-y-1">
              <h4 className="text-xs font-bold text-[var(--text-primary)]">
                Prêt pour le cadrage assisté par IA
              </h4>
              <p className="text-[11px] text-[var(--text-muted)] leading-relaxed">
                Lancez l'agent Clarify en mode interactif TTY pour cadrer les besoins, ou démarrez directement la rédaction de la spécification technique ({specFramework === 'openfeature' ? 'Open Feature' : 'SpecKit'}).
              </p>
            </div>
            <div className="flex items-center justify-center gap-2">
              <button
                type="button"
                onClick={() => {
                  setIsTtyOpen(true)
                  setTtyCommand(clarifyCliCommand)
                }}
                className="inline-flex items-center gap-1.5 px-4 py-2 rounded-xl text-xs font-bold text-white bg-amber-600 hover:bg-amber-500 transition-all shadow-xs cursor-pointer active:scale-95"
              >
                <Terminal size={14} />
                <span>Ouvrir Console TTY Cadrage</span>
              </button>
            </div>
          </div>
        )}
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
            <option value="to_clarify">{t.status.to_clarify} (#new)</option>
            <option value="to_specify">{t.status.to_specify} (#clarified)</option>
            <option value="to_implement">{t.status.to_implement} (#specified)</option>
            <option value="to_test">{t.status.to_test} (#implemented)</option>
            <option value="to_close">{t.status.to_close} (#reviewed)</option>
            <option value="finished">{t.status.finished} (#finished)</option>
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
          <div className="flex items-center justify-between mb-1">
            <label className="block text-[10px] font-bold uppercase tracking-wider text-[var(--text-muted)]">
              Projet
            </label>
            {projects.find(p => p.id === taskProjectId) && (
              <span className="text-[10px] text-[var(--text-muted)] font-mono">
                Tracker : <span className="font-semibold text-[var(--accent-color)]">
                  {projects.find(p => p.id === taskProjectId)?.issueTracker === 'linear'
                    ? `Linear (${projects.find(p => p.id === taskProjectId)?.linearTeam || 'FRE'})`
                    : projects.find(p => p.id === taskProjectId)?.issueTracker === 'github'
                    ? 'GitHub'
                    : 'Local'}
                </span>
              </span>
            )}
          </div>
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
              <label className="text-[10px] font-medium text-[var(--text-secondary)] flex items-center gap-1.5">
                <span>Branche Git</span>
                {branchName && gitStatus?.branch === branchName && (
                  <span className="text-[9px] font-bold text-emerald-400 bg-emerald-500/15 border border-emerald-500/30 px-1.5 py-0.2 rounded">
                    Active
                  </span>
                )}
              </label>
              <div className="flex items-center gap-2">
                {branchName && (
                  <button
                    type="button"
                    disabled={isSwitchingBranch}
                    onClick={async () => {
                      setIsSwitchingBranch(true)
                      try {
                        if (selectedTask?.id) {
                          await checkoutTaskBranch(selectedTask.id)
                        } else if (branchName) {
                          await switchGitBranch(branchName)
                        }
                      } finally {
                        setIsSwitchingBranch(false)
                      }
                    }}
                    className={`text-[9px] font-mono font-bold flex items-center gap-1 px-1.5 py-0.5 rounded cursor-pointer transition-colors ${
                      gitStatus?.branch === branchName
                        ? 'bg-emerald-500/20 text-emerald-300 border border-emerald-500/40'
                        : 'bg-cyan-500/20 text-cyan-300 border border-cyan-500/40 hover:bg-cyan-500/30'
                    }`}
                    title="Basculer le projet sur cette branche"
                  >
                    <GitBranch size={10} className={isSwitchingBranch ? 'animate-spin text-cyan-400' : ''} />
                    <span>{gitStatus?.branch === branchName ? '✓ Active' : isSwitchingBranch ? 'Bascule...' : 'Basculer'}</span>
                  </button>
                )}
                {branchName && (
                  <button
                    type="button"
                    onClick={() => copyBranchCommand(branchName)}
                    className="text-[9px] font-mono font-bold text-indigo-400 hover:underline flex items-center gap-1 cursor-pointer"
                    title="Copier la commande git checkout"
                  >
                    {copiedBranch ? <Check size={10} className="text-emerald-400" /> : <Copy size={10} />}
                    <span>{copiedBranch ? 'Copié !' : 'Copier'}</span>
                  </button>
                )}
                <button
                  type="button"
                  onClick={() => selectedTask && openInEditor({ taskId: selectedTask.id })}
                  className="text-[9px] font-mono font-bold text-cyan-400 hover:underline flex items-center gap-1 cursor-pointer"
                  title={`Ouvrir le dossier / worktree dans ${settings.editorCommand || 'VS Code'}`}
                >
                  <Code2 size={10} className="text-cyan-400" />
                  <span>Code</span>
                </button>
              </div>
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
            if (lower === 'new' || lower === 'untouched') badgeStyle = 'bg-cyan-500/20 text-cyan-300 border border-cyan-500/40 font-bold'
            else if (lower === 'clarified') badgeStyle = 'bg-amber-500/20 text-amber-300 border border-amber-500/40 font-bold'
            else if (lower === 'specified') badgeStyle = 'bg-blue-500/20 text-blue-300 border border-blue-500/40 font-bold'
            else if (lower === 'implemented') badgeStyle = 'bg-indigo-500/20 text-indigo-300 border border-indigo-500/40 font-bold'
            else if (lower === 'reviewed') badgeStyle = 'bg-purple-500/20 text-purple-300 border border-purple-500/40 font-bold'
            else if (lower === 'finished' || lower === 'closed') badgeStyle = 'bg-emerald-500/20 text-emerald-300 border border-emerald-500/40 font-bold'

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
        <div className="flex flex-wrap items-center justify-between gap-3 p-3 rounded-xl bg-linear-to-r from-[var(--accent-light)] to-[var(--bg-tertiary)] border border-[var(--accent-color)]/40 shadow-xs">
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
          <div className="flex items-center gap-2 flex-wrap">
            {(status === 'to_clarify' || status === 'backlog' || status === 'to_specify' || status === 'specified') && (
              <button
                type="button"
                onClick={() => handleTriggerSkill('implement')}
                disabled={isSkillRunning}
                className="flex items-center gap-1.5 px-3.5 py-2 rounded-xl text-xs font-bold text-white bg-linear-to-r from-blue-600 to-indigo-600 shadow hover:opacity-90 active:scale-95 transition-all disabled:opacity-50"
                title="Sauter directement le cadrage et lancer l'implémentation du code"
              >
                <Flame size={13} className="text-amber-300" />
                <span>🚀 Passer direct au Code</span>
              </button>
            )}

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
          onClick={handleClose}
          className="absolute inset-0 bg-black/40 backdrop-blur-2xs animate-in fade-in duration-200"
        />

        {/* Sliding Panel */}
        <div className="absolute inset-y-0 right-0 max-w-full flex pl-6 sm:pl-10">
          <div className="w-screen max-w-3xl 2xl:max-w-4xl bg-[var(--bg-secondary)] border-l border-[var(--border-color)] shadow-2xl flex flex-col h-full animate-in slide-in-from-right duration-200">
            {/* Panel Header */}
            <div className="flex items-center justify-between px-6 py-3.5 border-b border-[var(--border-color)] bg-[var(--bg-tertiary)]/40 shrink-0">
              {/* Left: Key & External Link */}
              <div className="flex items-center gap-2.5 min-w-0">
                <span className="font-mono text-sm font-bold text-[var(--accent-color)] bg-[var(--accent-light)] px-2.5 py-1 rounded-lg flex items-center gap-1.5 shrink-0">
                  {selectedTask.source === 'linear' && <span className="text-indigo-400 font-bold font-mono">◆</span>}
                  {selectedTask.source === 'github' && <FolderGit2 size={13} className="text-purple-400" />}
                  {selectedTask.source === 'jira' && <span className="text-blue-400 font-sans font-black text-xs">J</span>}
                  {(!selectedTask.source || selectedTask.source === 'local') && <Folder size={13} className="text-emerald-400" />}
                  {selectedTask.key}
                </span>

                {externalUrl && (
                  <a
                    href={externalUrl}
                    target="_blank"
                    rel="noreferrer"
                    className={`flex items-center gap-1.5 px-2.5 py-1 rounded-lg text-xs font-bold transition-all shadow-2xs hover:scale-105 ${
                      selectedTask.source === 'linear'
                        ? 'text-indigo-400 bg-indigo-500/15 hover:bg-indigo-500/25 border border-indigo-500/30'
                        : selectedTask.source === 'github'
                        ? 'text-purple-400 bg-purple-500/15 hover:bg-purple-500/25 border border-purple-500/30'
                        : selectedTask.source === 'jira'
                        ? 'text-blue-400 bg-blue-500/15 hover:bg-blue-500/25 border border-blue-500/30'
                        : 'text-[var(--accent-color)] bg-[var(--accent-light)]'
                    }`}
                    title={selectedTask.source === 'linear' ? 'Ouvrir dans Linear' : selectedTask.source === 'github' ? 'Ouvrir dans GitHub' : selectedTask.source === 'jira' ? 'Ouvrir dans Jira' : 'Ouvrir'}
                  >
                    <span>{selectedTask.source === 'linear' ? 'Linear' : selectedTask.source === 'github' ? 'GitHub' : selectedTask.source === 'jira' ? 'Jira' : 'Ouvrir'}</span>
                    <ExternalLink size={11} />
                  </a>
                )}
              </div>

              {/* Right: Quick switcher to Modal, PR Link, Delete, Close */}
              <div className="flex items-center gap-1.5 shrink-0">
                {/* Discuss with Agent Button */}
                <button
                  type="button"
                  onClick={() => {
                    if (selectedTask) setChatTask(selectedTask)
                  }}
                  className="flex items-center gap-1.5 px-2.5 py-1 rounded-lg text-xs font-bold text-white bg-[var(--accent-color)] hover:opacity-90 transition-all shadow-xs cursor-pointer active:scale-95"
                  title="💬 Discuter en direct avec l'agent Copilot"
                >
                  <MessageSquare size={12} />
                  <span>Discuter</span>
                </button>

                {/* Open in Editor Button */}
                <button
                  type="button"
                  onClick={() => {
                    if (selectedTask) openInEditor({ taskId: selectedTask.id })
                  }}
                  className="flex items-center gap-1.5 px-2.5 py-1 rounded-lg text-xs font-semibold text-cyan-300 bg-cyan-500/10 hover:bg-cyan-500/20 border border-cyan-500/30 transition-all cursor-pointer shadow-xs active:scale-95"
                  title={`Ouvrir le code / worktree dans ${settings.editorCommand || 'VS Code'}`}
                >
                  <Code2 size={12} className="text-cyan-400" />
                  <span className="hidden sm:inline">Code</span>
                </button>

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
                  onClick={handleClose}
                  className="p-1.5 rounded-lg text-[var(--text-muted)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-tertiary)] transition-colors"
                >
                  <X size={17} />
                </button>
              </div>
            </div>

            {/* Panel Tab Navigation Header */}
            <div className="flex items-center gap-2 px-6 pt-2.5 border-b border-[var(--border-color)] bg-[var(--bg-secondary)] shrink-0 text-xs font-semibold">
              <button
                type="button"
                onClick={() => setActiveTab('details')}
                className={`pb-2 flex items-center gap-1.5 border-b-2 transition-all cursor-pointer ${
                  activeTab === 'details'
                    ? 'border-[var(--accent-color)] accent-text font-bold'
                    : 'border-transparent text-[var(--text-muted)] hover:text-[var(--text-primary)]'
                }`}
              >
                <FileCode size={13} />
                <span>Story</span>
              </button>

              <button
                type="button"
                onClick={() => setActiveTab('cadrage')}
                className={`pb-2 flex items-center gap-1.5 border-b-2 transition-all cursor-pointer ${
                  activeTab === 'cadrage'
                    ? 'border-amber-400 text-amber-400 font-bold'
                    : 'border-transparent text-[var(--text-muted)] hover:text-[var(--text-primary)]'
                }`}
              >
                <HelpCircle size={13} className="text-amber-400" />
                <span>Cadrage & Specs</span>
                {Boolean(clarifyActivity || specifyActivity) && (
                  <span className="w-1.5 h-1.5 rounded-full bg-amber-400 animate-pulse" />
                )}
              </button>

              <button
                type="button"
                onClick={() => setActiveTab('skills')}
                className={`pb-2 flex items-center gap-1.5 border-b-2 transition-all cursor-pointer ${
                  activeTab === 'skills'
                    ? 'border-[var(--accent-color)] accent-text font-bold'
                    : 'border-transparent text-[var(--text-muted)] hover:text-[var(--text-primary)]'
                }`}
              >
                <Sparkles size={13} className="text-purple-400" />
                <span>Skills & Copilot</span>
              </button>
            </div>

            {/* Panel Scrollable Content Body */}
            <div className="p-6 overflow-y-auto space-y-6 flex-1 text-xs">
              {activeTab === 'details' && renderStoryInfoSection()}
              {activeTab === 'cadrage' && renderCadrageSection()}
              {activeTab === 'skills' && renderSkillsCopilotSection()}
            </div>

            {/* Panel Sticky Footer */}
            <div className="flex items-center justify-between px-6 py-3.5 border-t border-[var(--border-color)] bg-[var(--bg-tertiary)]/40 shrink-0">
              <span className="text-[11px] text-[var(--text-muted)]">
                {t.taskModal.created} {new Date(selectedTask.createdAt).toLocaleDateString()}
              </span>
              <div className="flex items-center gap-2">
                <button
                  type="button"
                  onClick={handleClose}
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
    <div className="fixed inset-0 z-50 flex items-center justify-center p-3 sm:p-6 bg-black/60 backdrop-blur-xs animate-in fade-in duration-200 select-none">
      <div
        className={`relative w-full transition-all duration-200 bg-[var(--bg-secondary)] border border-[var(--border-color)] shadow-2xl overflow-hidden flex flex-col ${
          isMaximized
            ? 'w-full h-full max-w-none max-h-none rounded-2xl'
            : 'max-w-5xl 2xl:max-w-6xl h-[90vh] max-h-[92vh] rounded-2xl'
        }`}
      >
        {/* Modal Header */}
        <div className="flex items-center justify-between px-6 py-3.5 border-b border-[var(--border-color)] bg-[var(--bg-tertiary)]/30 shrink-0">
          <div className="flex items-center gap-3">
            <span className="font-mono text-sm font-bold text-[var(--accent-color)] bg-[var(--accent-light)] px-2.5 py-1 rounded-lg flex items-center gap-1.5">
              {selectedTask.source === 'linear' && <span className="text-indigo-400 font-bold font-mono">◆</span>}
              {selectedTask.source === 'github' && <FolderGit2 size={13} className="text-purple-400" />}
              {selectedTask.source === 'jira' && <span className="text-blue-400 font-sans font-black text-xs">J</span>}
              {(!selectedTask.source || selectedTask.source === 'local') && <Folder size={13} className="text-emerald-400" />}
              {selectedTask.key}
            </span>
            <div className="flex items-center gap-2 text-xs text-[var(--text-muted)]">
              {externalUrl && (
                <a
                  href={externalUrl}
                  target="_blank"
                  rel="noreferrer"
                  className="flex items-center gap-1 text-[11px] font-semibold text-[var(--accent-color)] hover:underline"
                  title={selectedTask.source === 'linear' ? 'Ouvrir dans Linear' : selectedTask.source === 'github' ? 'Ouvrir dans GitHub' : selectedTask.source === 'jira' ? 'Ouvrir dans Jira' : 'Ouvrir'}
                >
                  <ExternalLink size={12} />
                  <span>{selectedTask.source === 'linear' ? 'Linear' : selectedTask.source === 'github' ? 'GitHub' : selectedTask.source === 'jira' ? 'Jira' : 'Ouvrir'}</span>
                </a>
              )}
            </div>
          </div>

          <div className="flex items-center gap-2">
            {/* Discuss with Agent Button */}
            <button
              type="button"
              onClick={() => {
                if (selectedTask) setChatTask(selectedTask)
              }}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-xl text-xs font-bold text-white bg-[var(--accent-color)] hover:opacity-90 transition-all shadow-xs cursor-pointer active:scale-95"
              title="💬 Discuter en direct avec l'agent Copilot"
            >
              <MessageSquare size={13} />
              <span>Discuter avec l'agent</span>
            </button>

            {/* Open in Editor Button */}
            <button
              type="button"
              onClick={() => {
                if (selectedTask) openInEditor({ taskId: selectedTask.id })
              }}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-xl text-xs font-semibold text-cyan-300 bg-cyan-500/10 hover:bg-cyan-500/20 border border-cyan-500/30 transition-all cursor-pointer shadow-xs active:scale-95"
              title={`Ouvrir le code / worktree dans ${settings.editorCommand || 'VS Code'}`}
            >
              <Code2 size={13} className="text-cyan-400" />
              <span>Code</span>
            </button>

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
              className="p-1.5 rounded-lg text-rose-400 hover:bg-rose-500/10 transition-colors cursor-pointer"
              title={t.taskModal.delete}
            >
              <Trash2 size={16} />
            </button>

            {/* Maximize / Minimize toggle */}
            <button
              type="button"
              onClick={() => setIsMaximized(prev => !prev)}
              className="p-1.5 rounded-lg text-[var(--text-muted)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-tertiary)] transition-colors cursor-pointer"
              title={isMaximized ? "Réduire" : "Plein écran"}
            >
              {isMaximized ? <Minimize2 size={16} /> : <Maximize2 size={16} />}
            </button>

            <button
              onClick={handleClose}
              className="p-1.5 rounded-lg text-[var(--text-muted)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-tertiary)] transition-colors cursor-pointer"
            >
              <X size={18} />
            </button>
          </div>
        </div>

        {/* Tab Navigation Header */}
        <div className="flex items-center justify-between px-6 pt-3 border-b border-[var(--border-color)] bg-[var(--bg-secondary)] shrink-0">
          <div className="flex items-center gap-5 text-xs font-semibold">
            <button
              type="button"
              onClick={() => setActiveTab('details')}
              className={`pb-2.5 flex items-center gap-1.5 border-b-2 transition-all cursor-pointer ${
                activeTab === 'details'
                  ? 'border-[var(--accent-color)] accent-text font-bold'
                  : 'border-transparent text-[var(--text-muted)] hover:text-[var(--text-primary)]'
              }`}
            >
              <FileCode size={14} />
              <span>Infos de la Story</span>
            </button>

            <button
              type="button"
              onClick={() => setActiveTab('cadrage')}
              className={`pb-2.5 flex items-center gap-1.5 border-b-2 transition-all cursor-pointer ${
                activeTab === 'cadrage'
                  ? 'border-amber-400 text-amber-400 font-bold'
                  : 'border-transparent text-[var(--text-muted)] hover:text-[var(--text-primary)]'
              }`}
            >
              <HelpCircle size={14} className="text-amber-400" />
              <span>Cadrage & Spécifications</span>
              {Boolean(clarifyActivity || specifyActivity) && (
                <span className="w-2 h-2 rounded-full bg-amber-400 animate-pulse ml-0.5" />
              )}
            </button>

            <button
              type="button"
              onClick={() => setActiveTab('skills')}
              className={`pb-2.5 flex items-center gap-1.5 border-b-2 transition-all cursor-pointer ${
                activeTab === 'skills'
                  ? 'border-[var(--accent-color)] accent-text font-bold'
                  : 'border-transparent text-[var(--text-muted)] hover:text-[var(--text-primary)]'
              }`}
            >
              <Sparkles size={14} className="text-purple-400" />
              <span>Skills & Agent Copilot</span>
            </button>
          </div>
        </div>

        {/* Modal Scrollable Body */}
        <div className="p-6 overflow-y-auto space-y-6 flex-1 text-xs">
          {activeTab === 'details' && renderStoryInfoSection()}
          {activeTab === 'cadrage' && renderCadrageSection()}
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
              onClick={handleClose}
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
