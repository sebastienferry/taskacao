import React, { useState, useEffect } from 'react'
import {
  X,
  Folder,
  Terminal,
  Zap,
  Flame,
  Layers,
  Box,
  Code2,
  Cpu,
  Sparkles,
  Workflow,
  Save,
  Trash2,
  FolderGit2,
  Check,
  Download,
  CheckCircle2,
  AlertCircle,
  Loader2,
  FileCode,
  ShieldCheck,
  HelpCircle,
  GitBranch,
  ArrowRight,
  Sliders,
  RefreshCw,
} from 'lucide-react'
import { useApp } from '../context/AppContext'
import type { AccentColor, IssueTracker, ProjectSkillsStatus, WorkflowStage, DetectedStatus } from '../types'

const AVAILABLE_ICONS = [
  { name: 'Folder', Icon: Folder, label: 'Dossier' },
  { name: 'Terminal', Icon: Terminal, label: 'Terminal' },
  { name: 'Zap', Icon: Zap, label: 'Éclair' },
  { name: 'Flame', Icon: Flame, label: 'Flamme' },
  { name: 'Layers', Icon: Layers, label: 'Calques' },
  { name: 'Box', Icon: Box, label: 'Module' },
  { name: 'Code2', Icon: Code2, label: 'Code' },
  { name: 'Cpu', Icon: Cpu, label: 'Core / CPU' },
  { name: 'Sparkles', Icon: Sparkles, label: 'IA / Magic' },
  { name: 'Workflow', Icon: Workflow, label: 'Workflow' },
]

const AVAILABLE_COLORS: { name: AccentColor; label: string; bgClass: string; ringClass: string }[] = [
  { name: 'indigo', label: 'Indigo', bgClass: 'bg-indigo-500', ringClass: 'ring-indigo-400' },
  { name: 'violet', label: 'Violet', bgClass: 'bg-violet-500', ringClass: 'ring-violet-400' },
  { name: 'emerald', label: 'Émeraude', bgClass: 'bg-emerald-500', ringClass: 'ring-emerald-400' },
  { name: 'amber', label: 'Ambre', bgClass: 'bg-amber-500', ringClass: 'ring-amber-400' },
  { name: 'rose', label: 'Rose', bgClass: 'bg-rose-500', ringClass: 'ring-rose-400' },
  { name: 'cyan', label: 'Cyan', bgClass: 'bg-cyan-500', ringClass: 'ring-cyan-400' },
  { name: 'blue', label: 'Bleu', bgClass: 'bg-blue-500', ringClass: 'ring-blue-400' },
  { name: 'orange', label: 'Orange', bgClass: 'bg-orange-500', ringClass: 'ring-orange-400' },
]

const DEFAULT_STAGE_MAPPING: Record<WorkflowStage, string> = {
  untouched: 'to_clarify',
  clarified: 'to_specify',
  specified: 'to_implement',
  implemented: 'to_test',
  reviewed: 'to_test',
  finished: 'to_close',
}

const STAGE_CONFIGS: { id: WorkflowStage; label: string; sub: string; color: string }[] = [
  { id: 'untouched', label: '#untouched', sub: 'Cadrage initial brut', color: 'cyan' },
  { id: 'clarified', label: '#clarified', sub: 'Questions & cadrage validés', color: 'amber' },
  { id: 'specified', label: '#specified', sub: 'Spécification technique prête', color: 'blue' },
  { id: 'implemented', label: '#implemented', sub: 'Développement terminé sur branche', color: 'indigo' },
  { id: 'reviewed', label: '#reviewed', sub: 'Revue de code & PR prête', color: 'purple' },
  { id: 'finished', label: '#finished', sub: 'Ticket validé & fusionné', color: 'emerald' },
]

const STATUS_OPTIONS: { id: string; label: string; stageCategory: string }[] = [
  { id: 'to_clarify', label: 'À clarifier / Todo (Backlog)', stageCategory: 'Todo' },
  { id: 'to_specify', label: 'À spécifier (Cadré)', stageCategory: 'In Progress' },
  { id: 'to_implement', label: 'À implémenter (En dev)', stageCategory: 'In Progress' },
  { id: 'to_test', label: 'À tester (En revue / QA)', stageCategory: 'Review' },
  { id: 'to_close', label: 'À fermer (Terminé / Mergé)', stageCategory: 'Done' },
]

export const ProjectModal: React.FC = () => {
  const {
    isProjectModalOpen,
    setIsProjectModalOpen,
    editingProject,
    setEditingProject,
    createProject,
    updateProject,
    deleteProject,
    addToast,
    t,
  } = useApp()

  const [name, setName] = useState('')
  const [slug, setSlug] = useState('')
  const [description, setDescription] = useState('')
  const [icon, setIcon] = useState('Folder')
  const [color, setColor] = useState<AccentColor>('indigo')
  const [repoPath, setRepoPath] = useState('')
  const [linearTeam, setLinearTeam] = useState('FRE')
  const [githubRepo, setGithubRepo] = useState('')
  const [issueTracker, setIssueTracker] = useState<IssueTracker>('linear')
  const [isDefault, setIsDefault] = useState(false)
  const [stageMapping, setStageMapping] = useState<Record<WorkflowStage, string>>(DEFAULT_STAGE_MAPPING)
  const [customInputMode, setCustomInputMode] = useState<Record<WorkflowStage, boolean>>({
    untouched: false,
    clarified: false,
    specified: false,
    implemented: false,
    reviewed: false,
    finished: false,
  })
  const [detectedStatuses, setDetectedStatuses] = useState<DetectedStatus[]>([])
  const [isDetectingStatuses, setIsDetectingStatuses] = useState(false)
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [isDeleting, setIsDeleting] = useState(false)

  const fetchDetectedStatuses = async (customTeam?: string, customTracker?: string, customRepo?: string) => {
    setIsDetectingStatuses(true)
    try {
      const res = await fetch('/api/projects/detect-statuses', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          projectId: editingProject?.id || '',
          issueTracker: customTracker || issueTracker,
          linearTeam: customTeam || linearTeam,
          githubRepo: customRepo || githubRepo,
        }),
      })
      if (res.ok) {
        const data: DetectedStatus[] = await res.json()
        setDetectedStatuses(data)
      }
    } catch (err) {
      console.error('Failed to detect statuses', err)
    } finally {
      setIsDetectingStatuses(false)
    }
  }

  // Skills & Git status in project CWD
  const [skillsStatus, setSkillsStatus] = useState<ProjectSkillsStatus | null>(null)
  const [isLoadingSkills, setIsLoadingSkills] = useState(false)
  const [isInstallingSkills, setIsInstallingSkills] = useState(false)
  const [isInitializingGit, setIsInitializingGit] = useState(false)

  const handleInitGit = async () => {
    const targetPath = repoPath.trim()
    if (!targetPath || isInitializingGit) return

    setIsInitializingGit(true)
    try {
      const target = editingProject ? editingProject.id : targetPath
      const res = await fetch(`/api/projects/${encodeURIComponent(target)}/init-git`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          repoPath: targetPath,
          projectId: editingProject?.id || slug || 'project',
        }),
      })

      if (!res.ok) {
        const errData = await res.json()
        throw new Error(errData.error || 'Erreur lors de l\'initialisation Git')
      }

      const result = await res.json()
      await fetchSkillsStatus(target)

      addToast({
        type: 'success',
        title: 'Dépôt Git initialisé !',
        description: result.message || `Dépôt Git configuré dans ${targetPath}`,
      })
    } catch (err: any) {
      addToast({
        type: 'error',
        title: 'Échec de git init',
        description: err.message,
      })
    } finally {
      setIsInitializingGit(false)
    }
  }

  const fetchSkillsStatus = async (targetIdOrPath: string) => {
    if (!targetIdOrPath || !targetIdOrPath.trim()) return
    setIsLoadingSkills(true)
    try {
      const res = await fetch(`/api/projects/${encodeURIComponent(targetIdOrPath.trim())}/skills-status`)
      if (res.ok) {
        const data: ProjectSkillsStatus = await res.json()
        setSkillsStatus(data)
      }
    } catch (err) {
      console.error('Failed to fetch skills status', err)
    } finally {
      setIsLoadingSkills(false)
    }
  }

  useEffect(() => {
    if (editingProject) {
      setName(editingProject.name)
      setSlug(editingProject.slug)
      setDescription(editingProject.description || '')
      setIcon(editingProject.icon || 'Folder')
      setColor((editingProject.color as AccentColor) || 'indigo')
      setRepoPath(editingProject.repoPath || '')
      setLinearTeam(editingProject.linearTeam || 'FRE')
      setGithubRepo(editingProject.githubRepo || '')
      setIssueTracker(editingProject.issueTracker || 'linear')
      setIsDefault(editingProject.isDefault || false)
      setStageMapping(
        editingProject.stageMapping && Object.keys(editingProject.stageMapping).length > 0
          ? { ...DEFAULT_STAGE_MAPPING, ...editingProject.stageMapping }
          : DEFAULT_STAGE_MAPPING
      )
      fetchSkillsStatus(editingProject.id)
      fetchDetectedStatuses(editingProject.linearTeam, editingProject.issueTracker, editingProject.githubRepo)
    } else {
      setName('')
      setSlug('')
      setDescription('')
      setIcon('Folder')
      setColor('indigo')
      setRepoPath('')
      setLinearTeam('FRE')
      setGithubRepo('')
      setIssueTracker('linear')
      setIsDefault(false)
      setStageMapping(DEFAULT_STAGE_MAPPING)
      setSkillsStatus(null)
      fetchDetectedStatuses('FRE', 'linear', '')
    }
  }, [editingProject, isProjectModalOpen])

  // Trigger skills check when repoPath changes
  useEffect(() => {
    if (!editingProject && repoPath && repoPath.length > 5 && repoPath.includes('/')) {
      const timeout = setTimeout(() => {
        fetchSkillsStatus(repoPath)
      }, 500)
      return () => clearTimeout(timeout)
    }
  }, [repoPath, editingProject])

  const handleInstallSkills = async () => {
    const targetPath = repoPath.trim()
    if (!targetPath || isInstallingSkills) return

    setIsInstallingSkills(true)
    try {
      const target = editingProject ? editingProject.id : targetPath
      const res = await fetch(`/api/projects/${encodeURIComponent(target)}/install-skills`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          repoPath: targetPath,
          projectId: editingProject?.id || slug || 'project',
        }),
      })

      if (!res.ok) {
        const errData = await res.json()
        throw new Error(errData.error || 'Erreur lors de l\'installation des skills')
      }

      const result = await res.json()
      if (result.status) {
        setSkillsStatus(result.status)
      } else {
        await fetchSkillsStatus(target)
      }

      addToast({
        type: 'success',
        title: 'Skills IA installées avec succès !',
        description: `5 compétences IA et .fretzee/config.json configurés dans ${targetPath}`,
      })
    } catch (err: any) {
      addToast({
        type: 'error',
        title: 'Échec de l\'installation',
        description: err.message,
      })
    } finally {
      setIsInstallingSkills(false)
    }
  }

  useEffect(() => {
    if (!isProjectModalOpen) return
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        setIsProjectModalOpen(false)
        setEditingProject(null)
      }
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [isProjectModalOpen, setIsProjectModalOpen, setEditingProject])

  if (!isProjectModalOpen) return null

  const handleNameChange = (val: string) => {
    setName(val)
    if (!editingProject) {
      const generatedSlug = val
        .toLowerCase()
        .replace(/[^a-z0-9]+/g, '-')
        .replace(/(^-|-$)/g, '')
      setSlug(generatedSlug)
    }
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!name.trim() || isSubmitting) return

    setIsSubmitting(true)
    try {
      const payload = {
        name: name.trim(),
        slug: slug.trim() || name.toLowerCase().replace(/[^a-z0-9]+/g, '-'),
        description: description.trim(),
        icon,
        color,
        repoPath: repoPath.trim(),
        linearTeam: linearTeam.trim().toUpperCase(),
        githubRepo: githubRepo.trim(),
        issueTracker,
        isDefault,
        stageMapping,
      }

      if (editingProject) {
        await updateProject(editingProject.id, payload)
      } else {
        await createProject(payload)
      }
      setIsProjectModalOpen(false)
      setEditingProject(null)
    } finally {
      setIsSubmitting(false)
    }
  }

  const handleDelete = async () => {
    if (!editingProject || isDeleting) return
    if (confirm(`Êtes-vous sûr de vouloir supprimer le projet "${editingProject.name}" ? Les tâches associées seront réassignées au projet principal.`)) {
      setIsDeleting(true)
      try {
        await deleteProject(editingProject.id)
        setIsProjectModalOpen(false)
        setEditingProject(null)
      } finally {
        setIsDeleting(false)
      }
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-xs animate-in fade-in duration-150 select-none">
      <div className="relative w-full max-w-xl rounded-2xl bg-[var(--bg-secondary)] border border-[var(--border-color)] shadow-2xl overflow-hidden flex flex-col max-h-[92vh]">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-[var(--border-color)] bg-[var(--bg-tertiary)]/40 shrink-0">
          <div className="flex items-center gap-2.5">
            <div className={`w-8 h-8 rounded-xl bg-${color}-500/20 text-${color}-400 flex items-center justify-center border border-${color}-500/30`}>
              <Layers size={16} />
            </div>
            <div>
              <h3 className="text-sm font-bold text-[var(--text-primary)]">
                {editingProject ? 'Paramètres du Projet' : 'Nouveau Projet'}
              </h3>
              <p className="text-[11px] text-[var(--text-muted)]">
                {editingProject ? `Configurer ${editingProject.name}` : 'Créez un espace dédié avec son propre dépôt Git et tracker'}
              </p>
            </div>
          </div>

          <button
            type="button"
            onClick={() => {
              setIsProjectModalOpen(false)
              setEditingProject(null)
            }}
            className="p-1.5 rounded-lg text-[var(--text-muted)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-tertiary)] transition-colors"
          >
            <X size={18} />
          </button>
        </div>

        {/* Scrollable Form Body */}
        <form onSubmit={handleSubmit} className="p-6 overflow-y-auto space-y-5 flex-1 text-xs">
          {/* Name and Slug Grid */}
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <label className="block text-[10px] font-bold uppercase tracking-wider text-[var(--text-muted)] mb-1">
                Nom du Projet *
              </label>
              <input
                type="text"
                required
                value={name}
                onChange={e => handleNameChange(e.target.value)}
                placeholder="Ex: Fretzee Studio, Mobile App..."
                className="w-full px-3 py-2 text-xs rounded-xl bg-[var(--bg-tertiary)] border border-[var(--border-color)] text-[var(--text-primary)] focus:outline-none focus:border-[var(--accent-color)] font-medium"
              />
            </div>

            <div>
              <label className="block text-[10px] font-bold uppercase tracking-wider text-[var(--text-muted)] mb-1">
                Identifiant / Slug
              </label>
              <input
                type="text"
                value={slug}
                onChange={e => setSlug(e.target.value)}
                placeholder="Ex: fretzee-studio"
                className="w-full px-3 py-2 text-xs rounded-xl bg-[var(--bg-tertiary)] border border-[var(--border-color)] text-[var(--text-primary)] focus:outline-none focus:border-[var(--accent-color)] font-mono text-[11px]"
              />
            </div>
          </div>

          {/* Description */}
          <div>
            <label className="block text-[10px] font-bold uppercase tracking-wider text-[var(--text-muted)] mb-1">
              Description (Optionnel)
            </label>
            <input
              type="text"
              value={description}
              onChange={e => setDescription(e.target.value)}
              placeholder="Ex: Application principale Web et backend Go..."
              className="w-full px-3 py-2 text-xs rounded-xl bg-[var(--bg-tertiary)] border border-[var(--border-color)] text-[var(--text-primary)] focus:outline-none focus:border-[var(--accent-color)]"
            />
          </div>

          {/* Icon & Color Selector */}
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 pt-1">
            <div>
              <label className="block text-[10px] font-bold uppercase tracking-wider text-[var(--text-muted)] mb-2">
                Icône
              </label>
              <div className="grid grid-cols-5 gap-1.5 p-2 rounded-xl bg-[var(--bg-tertiary)] border border-[var(--border-color)]">
                {AVAILABLE_ICONS.map(({ name: iconName, Icon }) => {
                  const isSel = icon === iconName
                  return (
                    <button
                      key={iconName}
                      type="button"
                      onClick={() => setIcon(iconName)}
                      className={`p-2 rounded-lg flex items-center justify-center transition-all ${
                        isSel
                          ? 'bg-[var(--accent-color)] text-white shadow-sm scale-105'
                          : 'text-[var(--text-muted)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-primary)]'
                      }`}
                      title={iconName}
                    >
                      <Icon size={16} />
                    </button>
                  )
                })}
              </div>
            </div>

            <div>
              <label className="block text-[10px] font-bold uppercase tracking-wider text-[var(--text-muted)] mb-2">
                Couleur d'accentuation
              </label>
              <div className="grid grid-cols-4 gap-2 p-2.5 rounded-xl bg-[var(--bg-tertiary)] border border-[var(--border-color)]">
                {AVAILABLE_COLORS.map(c => {
                  const isSel = color === c.name
                  return (
                    <button
                      key={c.name}
                      type="button"
                      onClick={() => setColor(c.name)}
                      className={`h-7 rounded-lg ${c.bgClass} flex items-center justify-center transition-all ${
                        isSel ? 'ring-2 ring-white ring-offset-2 ring-offset-[var(--bg-secondary)] scale-105 shadow-sm' : 'opacity-80 hover:opacity-100'
                      }`}
                      title={c.label}
                    >
                      {isSel && <Check size={13} className="text-white drop-shadow" />}
                    </button>
                  )
                })}
              </div>
            </div>
          </div>

          {/* Local Repository Directory (CWD for AI skills) */}
          <div>
            <div className="flex items-center justify-between mb-1">
              <label className="block text-[10px] font-bold uppercase tracking-wider text-[var(--text-muted)]">
                📁 Dossier local du projet (CWD d'exécution IA) *
              </label>
            </div>
            <div className="relative">
              <input
                type="text"
                required
                value={repoPath}
                onChange={e => setRepoPath(e.target.value)}
                placeholder="Ex: /Users/sferry/Sources/mon-projet"
                className="w-full pl-8 pr-3 py-2 text-xs rounded-xl bg-[var(--bg-tertiary)] border border-[var(--border-color)] text-[var(--text-primary)] font-mono text-[11px] focus:outline-none focus:border-[var(--accent-color)]"
              />
              <Folder size={14} className="absolute left-2.5 top-2.5 text-[var(--accent-color)]" />
            </div>
            <p className="text-[10px] text-[var(--text-muted)] mt-1">
              Les compétences d'agents IA (Clarify, Specify, Implement, PR) s'exécuteront directement dans ce répertoire pour ce projet.
            </p>

            {/* Git Status / Init Git Badge & Action */}
            {repoPath && (
              <div className="mt-2.5 flex items-center justify-between p-2.5 rounded-xl bg-[var(--bg-tertiary)] border border-[var(--border-color)]">
                <div className="flex items-center gap-2.5 min-w-0">
                  <div className={`w-6 h-6 rounded-lg flex items-center justify-center shrink-0 ${
                    skillsStatus?.isGitRepo ? 'bg-emerald-500/20 text-emerald-400' : 'bg-amber-500/20 text-amber-400'
                  }`}>
                    <GitBranch size={13} />
                  </div>
                  <div className="flex flex-col min-w-0">
                    <div className="flex items-center gap-1.5 flex-wrap">
                      <span className={`text-xs font-bold ${skillsStatus?.isGitRepo ? 'text-emerald-400' : 'text-amber-400'}`}>
                        {skillsStatus?.isGitRepo ? 'Dépôt Git initialisé' : 'Dépôt Git non détecté'}
                      </span>
                      {skillsStatus?.isGitRepo && (
                        <span className="text-[10px] font-mono px-1.5 py-0.2 rounded bg-emerald-500/10 text-emerald-300 border border-emerald-500/20">
                          {skillsStatus.gitBranch || 'main'}
                        </span>
                      )}
                    </div>
                    <span className="text-[10px] text-[var(--text-muted)] truncate">
                      {skillsStatus?.isGitRepo
                        ? 'Gestion des branches et visualisation du code diff prêtes.'
                        : 'Initialisez Git pour activer la création de branches de tickets et l\'analyse de diff.'}
                    </span>
                  </div>
                </div>

                {!skillsStatus?.isGitRepo && (
                  <button
                    type="button"
                    onClick={handleInitGit}
                    disabled={isInitializingGit || !repoPath.trim()}
                    className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-xl text-xs font-bold text-white bg-indigo-600 hover:bg-indigo-500 active:scale-95 transition-all shadow-xs shrink-0 disabled:opacity-50"
                    title="Exécuter git init dans ce dossier"
                  >
                    {isInitializingGit ? (
                      <Loader2 size={13} className="animate-spin" />
                    ) : (
                      <Zap size={13} className="text-amber-300" />
                    )}
                    <span>Initialiser Git (git init)</span>
                  </button>
                )}
              </div>
            )}
          </div>

          {/* Issue Tracker & Remote Repositories */}
          <div className="p-3.5 rounded-xl bg-[var(--bg-tertiary)]/50 border border-[var(--border-color)] space-y-3">
            <div className="flex items-center justify-between">
              <span className="text-[10px] font-bold uppercase tracking-wider text-[var(--text-muted)] flex items-center gap-1.5">
                <FolderGit2 size={13} className="text-[var(--accent-color)]" />
                Intégrations & Suivi des Issues
              </span>
            </div>

            {/* Ligne 1 : Tracker et Équipe Linear */}
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <div>
                <label className="block text-[10px] font-medium text-[var(--text-secondary)] mb-1">
                  Tracker par défaut
                </label>
                <select
                  value={issueTracker}
                  onChange={e => setIssueTracker(e.target.value as IssueTracker)}
                  className="w-full px-2.5 py-1.5 text-xs rounded-xl bg-[var(--bg-primary)] border border-[var(--border-color)] text-[var(--text-primary)] focus:outline-none focus:border-[var(--accent-color)]"
                >
                  <option value="linear">Linear (Recommandé)</option>
                  <option value="github">GitHub Issues</option>
                  <option value="local">Local uniquement</option>
                </select>
              </div>

              <div>
                <label className="block text-[10px] font-medium text-[var(--text-secondary)] mb-1">
                  Équipe Linear (Team Key)
                </label>
                <input
                  type="text"
                  value={linearTeam}
                  onChange={e => setLinearTeam(e.target.value.toUpperCase())}
                  placeholder="Ex: FRE"
                  className="w-full px-2.5 py-1.5 text-xs rounded-xl bg-[var(--bg-primary)] border border-[var(--border-color)] text-[var(--text-primary)] font-mono focus:outline-none focus:border-[var(--accent-color)]"
                />
              </div>
            </div>

            {/* Ligne 2 : Dépôt GitHub distant avec passage à la ligne */}
            <div>
              <label className="block text-[10px] font-medium text-[var(--text-secondary)] mb-1">
                Dépôt GitHub distant (owner/repo)
              </label>
              <div className="relative">
                <input
                  type="text"
                  value={githubRepo}
                  onChange={e => setGithubRepo(e.target.value)}
                  placeholder="Ex: sebastienferry/mon-projet"
                  className="w-full pl-7 pr-3 py-1.5 text-xs rounded-xl bg-[var(--bg-primary)] border border-[var(--border-color)] text-[var(--text-primary)] font-mono text-[11px] focus:outline-none focus:border-[var(--accent-color)]"
                />
                <FolderGit2 size={13} className="absolute left-2.5 top-2.5 text-[var(--text-muted)]" />
              </div>
            </div>
          </div>

          {/* Mapping Pipeline IA ➔ Statuts Tracker */}
          <div className="p-4 rounded-xl bg-[var(--bg-tertiary)]/60 border border-[var(--border-color)] space-y-3.5">
            <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2">
              <div className="flex items-center gap-2">
                <Sliders size={14} className="text-[var(--accent-color)]" />
                <span className="text-[10px] font-bold uppercase tracking-wider text-[var(--text-primary)]">
                  Mapping Pipeline IA ➔ Statuts Tracker
                </span>
                {detectedStatuses.length > 0 && (
                  <span className="text-[10px] font-mono px-1.5 py-0.2 rounded-full bg-[var(--accent-light)] accent-text font-bold">
                    {detectedStatuses.length} statuts détectés
                  </span>
                )}
              </div>

              {/* Action Toolbar */}
              <div className="flex items-center flex-wrap gap-1.5">
                <button
                  type="button"
                  disabled={isDetectingStatuses}
                  onClick={() => fetchDetectedStatuses()}
                  className="flex items-center gap-1 px-2.5 py-1 rounded-lg text-[10px] font-bold bg-[var(--bg-primary)] border border-[var(--border-color)] text-[var(--text-primary)] hover:border-[var(--accent-color)] hover:bg-[var(--accent-light)] transition-all cursor-pointer disabled:opacity-50"
                  title="Scanner Linear / GitHub / Base de données pour détecter les statuts réels"
                >
                  <RefreshCw size={11} className={isDetectingStatuses ? 'animate-spin text-[var(--accent-color)]' : 'text-cyan-400'} />
                  <span>{isDetectingStatuses ? 'Scan...' : 'Auto-détecter'}</span>
                </button>

                <button
                  type="button"
                  onClick={() => {
                    if (detectedStatuses.length === 0) {
                      addToast({
                        type: 'info',
                        title: 'Aucun statut détecté',
                        description: 'Cliquez d\'abord sur "Auto-détecter" pour scanner votre tracker.',
                      })
                      return
                    }
                    const findStatus = (keywords: string[], fallback: string): string => {
                      for (const kw of keywords) {
                        const found = detectedStatuses.find(
                          s => s.name.toLowerCase().includes(kw) || (s.type && s.type.toLowerCase().includes(kw))
                        )
                        if (found) return found.name
                      }
                      return fallback
                    }
                    setStageMapping({
                      untouched: findStatus(['triage', 'backlog', 'unstarted', 'to_clarify', 'todo', 'open'], 'to_clarify'),
                      clarified: findStatus(['cadré', 'clarified', 'specify', 'to_specify', 'triage', 'todo', 'unstarted'], 'to_specify'),
                      specified: findStatus(['ready', 'specified', 'spec', 'plan', 'to_implement', 'todo'], 'to_implement'),
                      implemented: findStatus(['in progress', 'progress', 'dev', 'started', 'implemented', 'doing', 'to_test'], 'to_test'),
                      reviewed: findStatus(['review', 'pr', 'qa', 'test', 'reviewed', 'to_test'], 'to_test'),
                      finished: findStatus(['done', 'closed', 'completed', 'finished', 'to_close'], 'to_close'),
                    })
                    addToast({
                      type: 'success',
                      title: 'Mapping auto-assigné !',
                      description: 'Les statuts ont été mappés intelligemment sur vos 6 étapes IA.',
                    })
                  }}
                  className="flex items-center gap-1 px-2.5 py-1 rounded-lg text-[10px] font-bold bg-[var(--accent-light)] accent-text border border-[var(--accent-color)]/30 hover:opacity-90 transition-all cursor-pointer"
                  title="Associer automatiquement les statuts détectés aux 6 étapes IA"
                >
                  <Sparkles size={11} className="text-amber-400" />
                  <span>Auto-assigner</span>
                </button>

                <button
                  type="button"
                  onClick={() => setStageMapping(DEFAULT_STAGE_MAPPING)}
                  className="px-2 py-1 rounded-lg text-[10px] font-medium bg-[var(--bg-primary)] border border-[var(--border-color)] text-[var(--text-secondary)] hover:text-[var(--text-primary)] hover:border-[var(--accent-color)]/40 transition-colors cursor-pointer"
                  title="Réinitialiser avec le flux standard"
                >
                  Défaut
                </button>
              </div>
            </div>

            <p className="text-[11px] text-[var(--text-muted)] leading-relaxed">
              Associez chaque étape IA à un statut de votre tracker (choix dans la liste détectée ou saisie de n'importe quel statut personnalisé).
            </p>

            <div className="space-y-2">
              {STAGE_CONFIGS.map(stage => {
                const currentStatus = stageMapping[stage.id] || DEFAULT_STAGE_MAPPING[stage.id]
                const isCustom = customInputMode[stage.id]

                return (
                  <div
                    key={stage.id}
                    className="p-2.5 rounded-xl bg-[var(--bg-primary)] border border-[var(--border-color)] flex flex-col sm:flex-row sm:items-center justify-between gap-2.5"
                  >
                    {/* Stage Label Left */}
                    <div className="flex items-center gap-2 min-w-0">
                      <span className={`text-[11px] font-bold px-2 py-0.5 rounded-md border font-mono shrink-0 ${
                        stage.id === 'untouched' ? 'bg-cyan-500/15 text-cyan-400 border-cyan-500/30' :
                        stage.id === 'clarified' ? 'bg-amber-500/15 text-amber-400 border-amber-500/30' :
                        stage.id === 'specified' ? 'bg-blue-500/15 text-blue-400 border-blue-500/30' :
                        stage.id === 'implemented' ? 'bg-indigo-500/15 text-indigo-400 border-indigo-500/30' :
                        stage.id === 'reviewed' ? 'bg-purple-500/15 text-purple-400 border-purple-500/30' :
                        'bg-emerald-500/15 text-emerald-400 border-emerald-500/30'
                      }`}>
                        {stage.label}
                      </span>
                      <span className="text-[11px] text-[var(--text-muted)] truncate">
                        {stage.sub}
                      </span>
                    </div>

                    <div className="flex items-center gap-2 self-end sm:self-center shrink-0">
                      <ArrowRight size={13} className="text-[var(--text-muted)] hidden sm:inline shrink-0" />

                      {isCustom ? (
                        <div className="flex items-center gap-1.5">
                          <input
                            type="text"
                            value={currentStatus}
                            onChange={e =>
                              setStageMapping(prev => ({
                                ...prev,
                                [stage.id]: e.target.value,
                              }))
                            }
                            placeholder="Nom du statut (ex: QA, Ready...)"
                            className="w-44 px-2.5 py-1 text-xs rounded-lg bg-[var(--bg-secondary)] border border-[var(--accent-color)] text-[var(--text-primary)] font-medium focus:outline-none"
                            autoFocus
                          />
                          <button
                            type="button"
                            onClick={() =>
                              setCustomInputMode(prev => ({ ...prev, [stage.id]: false }))
                            }
                            className="px-2 py-1 rounded-lg text-[10px] font-medium bg-[var(--bg-tertiary)] text-[var(--text-secondary)] hover:text-[var(--text-primary)] border border-[var(--border-color)] transition-colors cursor-pointer"
                            title="Revenir à la liste"
                          >
                            Liste
                          </button>
                        </div>
                      ) : (
                        <div className="flex items-center gap-1.5">
                          <select
                            value={currentStatus}
                            onChange={e => {
                              if (e.target.value === '__custom__') {
                                setCustomInputMode(prev => ({ ...prev, [stage.id]: true }))
                              } else {
                                setStageMapping(prev => ({
                                  ...prev,
                                  [stage.id]: e.target.value,
                                }))
                              }
                            }}
                            className="w-48 px-2 py-1 text-xs rounded-lg bg-[var(--bg-secondary)] border border-[var(--border-color)] text-[var(--text-primary)] focus:outline-none focus:border-[var(--accent-color)] font-medium"
                          >
                            {detectedStatuses.length > 0 && (
                              <optgroup label="✨ Statuts détectés (Tracker / DB)">
                                {detectedStatuses.map(st => (
                                  <option key={`det-${st.id}`} value={st.name}>
                                    {st.name} {st.source ? `(${st.source})` : ''}
                                  </option>
                                ))}
                              </optgroup>
                            )}

                            <optgroup label="📋 Statuts standards Taskacao">
                              {STATUS_OPTIONS.map(opt => (
                                <option key={opt.id} value={opt.id}>
                                  {opt.label}
                                </option>
                              ))}
                            </optgroup>

                            <optgroup label="✏️ Personnalisé">
                              <option value="__custom__">➕ Saisir un statut libre...</option>
                            </optgroup>
                          </select>

                          <button
                            type="button"
                            onClick={() =>
                              setCustomInputMode(prev => ({ ...prev, [stage.id]: true }))
                            }
                            className="p-1 rounded-lg text-[var(--text-muted)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-tertiary)] transition-colors cursor-pointer"
                            title="Saisir un statut libre en texte"
                          >
                            ✏️
                          </button>
                        </div>
                      )}
                    </div>
                  </div>
                )
              })}
            </div>
          </div>

          {/* AI Skills Provisioning Section */}
          <div className="p-4 rounded-xl bg-[var(--bg-tertiary)]/60 border border-[var(--border-color)] space-y-3.5">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <Sparkles size={14} className="text-[var(--accent-color)]" />
                <span className="text-[10px] font-bold uppercase tracking-wider text-[var(--text-primary)]">
                  Compétences IA du Projet (.gemini/skills & .fretzee)
                </span>
              </div>

              {isLoadingSkills ? (
                <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-md text-[10px] font-mono text-[var(--text-muted)] bg-[var(--bg-primary)]">
                  <Loader2 size={11} className="animate-spin text-[var(--accent-color)]" />
                  <span>Vérification...</span>
                </span>
              ) : skillsStatus ? (
                <span className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-md text-[10px] font-mono font-bold ${
                  skillsStatus.installedAll
                    ? 'bg-emerald-500/20 text-emerald-300 border border-emerald-500/40'
                    : 'bg-amber-500/20 text-amber-300 border border-amber-500/40'
                }`}>
                  {skillsStatus.installedAll ? (
                    <>
                      <CheckCircle2 size={11} />
                      <span>5/5 Skills Prêtes</span>
                    </>
                  ) : (
                    <>
                      <AlertCircle size={11} />
                      <span>{skillsStatus.skills.filter(s => s.installed).length}/5 installées</span>
                    </>
                  )}
                </span>
              ) : null}
            </div>

            <p className="text-[11px] text-[var(--text-muted)] leading-relaxed">
              Installe et configure les 5 compétences de workflow agentique (Clarify, Specify, Implement, PR, Pick) directement dans le dossier local du projet pour l'IA.
            </p>

            {/* List of Skills with status badges */}
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
              {[
                { id: 'clarify', name: 'Clarify Issue', code: 'clarify-issue', icon: HelpCircle, color: 'amber' },
                { id: 'specify', name: 'Specify (Speckit)', code: 'specify-issue', icon: FileCode, color: 'blue' },
                { id: 'implement', name: 'Implement Code', code: 'code-issue', icon: Flame, color: 'indigo' },
                { id: 'create_pr', name: 'Review & PR', code: 'create-pr', icon: ShieldCheck, color: 'purple' },
                { id: 'pick', name: 'Auto-Pilot', code: 'pick-issue', icon: Sparkles, color: 'emerald' },
              ].map(s => {
                const isInst = skillsStatus?.skills.find(sk => sk.id === s.id)?.installed || false
                const SkillIcon = s.icon
                return (
                  <div
                    key={s.id}
                    className="p-2 rounded-lg bg-[var(--bg-primary)] border border-[var(--border-color)] flex items-center justify-between gap-2"
                  >
                    <div className="flex items-center gap-2 min-w-0">
                      <div className={`w-5 h-5 rounded-md flex items-center justify-center text-${s.color}-400 bg-${s.color}-500/10`}>
                        <SkillIcon size={12} />
                      </div>
                      <div className="flex flex-col min-w-0">
                        <span className="font-semibold text-[11px] text-[var(--text-primary)] truncate">
                          {s.name}
                        </span>
                        <span className="text-[9px] text-[var(--text-muted)] font-mono truncate">
                          /{s.code}
                        </span>
                      </div>
                    </div>

                    <div className="shrink-0">
                      {isInst ? (
                        <span className="inline-flex items-center gap-1 text-[9px] font-bold text-emerald-400">
                          <Check size={11} />
                          <span>Installé</span>
                        </span>
                      ) : (
                        <span className="text-[9px] font-medium text-[var(--text-muted)]">
                          Non installé
                        </span>
                      )}
                    </div>
                  </div>
                )
              })}
            </div>

            {/* Install Button */}
            <div className="pt-1 flex items-center justify-between">
              <span className="text-[10px] text-[var(--text-muted)] font-mono truncate max-w-[200px]" title={repoPath}>
                📁 {repoPath ? repoPath.split('/').slice(-2).join('/') : 'Non défini'}
              </span>

              <button
                type="button"
                disabled={isInstallingSkills || !repoPath.trim()}
                onClick={handleInstallSkills}
                className="flex items-center gap-1.5 px-3 py-1.5 rounded-xl text-xs font-bold bg-[var(--accent-color)] hover:opacity-90 text-white shadow-sm transition-all hover:scale-102 active:scale-95 disabled:opacity-50 cursor-pointer"
              >
                {isInstallingSkills ? (
                  <Loader2 size={13} className="animate-spin text-white" />
                ) : (
                  <Download size={13} />
                )}
                <span>
                  {isInstallingSkills
                    ? 'Installation en cours...'
                    : skillsStatus?.installedAll
                    ? 'Réinstaller / Synchroniser'
                    : '⚡ Installer les Skills dans le CWD'}
                </span>
              </button>
            </div>
          </div>

          {/* Default Project Checkbox */}
          <label className="flex items-center gap-2.5 cursor-pointer pt-1">
            <input
              type="checkbox"
              checked={isDefault}
              onChange={e => setIsDefault(e.target.checked)}
              className="rounded text-[var(--accent-color)] focus:ring-0 w-4 h-4 cursor-pointer"
            />
            <span className="text-xs text-[var(--text-secondary)] font-medium">
              Définir comme projet principal par défaut
            </span>
          </label>
        </form>

        {/* Modal Footer */}
        <div className="flex items-center justify-between px-6 py-4 border-t border-[var(--border-color)] bg-[var(--bg-tertiary)]/40 shrink-0">
          <div>
            {editingProject && !editingProject.isDefault && editingProject.id !== 'fretzee-studio' && (
              <button
                type="button"
                onClick={handleDelete}
                disabled={isDeleting}
                className="flex items-center gap-1.5 px-3 py-1.5 rounded-xl text-xs font-semibold text-rose-400 bg-rose-500/10 hover:bg-rose-500/20 border border-rose-500/30 transition-colors"
                title="Supprimer ce projet"
              >
                <Trash2 size={13} />
                <span>{isDeleting ? 'Suppression...' : 'Supprimer'}</span>
              </button>
            )}
          </div>

          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={() => {
                setIsProjectModalOpen(false)
                setEditingProject(null)
              }}
              className="px-4 py-2 rounded-xl text-xs font-medium text-[var(--text-secondary)] hover:bg-[var(--bg-tertiary)] transition-colors"
            >
              {t.taskModal.cancel}
            </button>
            <button
              type="button"
              onClick={handleSubmit}
              disabled={isSubmitting || !name.trim()}
              className="px-5 py-2 rounded-xl text-xs font-bold text-white accent-bg shadow-md hover:opacity-90 active:scale-95 flex items-center gap-1.5 transition-all disabled:opacity-50"
            >
              <Save size={14} />
              <span>{isSubmitting ? 'Enregistrement...' : editingProject ? 'Mettre à jour' : 'Créer le projet'}</span>
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
