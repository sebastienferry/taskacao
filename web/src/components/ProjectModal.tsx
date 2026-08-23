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
  Globe,
  Key,
  RotateCcw,
} from 'lucide-react'
import { useApp } from '../context/AppContext'
import type { AccentColor, IssueTracker, ProjectSkillsStatus, WorkflowStage, DetectedStatus } from '../types'

type ProjectTab = 'general' | 'git' | 'tracker' | 'skills'

const TABS: { id: ProjectTab; label: string; icon: React.FC<{ size?: number; className?: string }> }[] = [
  { id: 'general', label: 'Général', icon: Folder },
  { id: 'git', label: 'Git', icon: GitBranch },
  { id: 'tracker', label: 'Tracker', icon: Sliders },
  { id: 'skills', label: 'Compétences IA', icon: Sparkles },
]

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
  { name: 'neon-cyan', label: '⚡ Cyber Cyan', bgClass: 'bg-[#00f0ff]', ringClass: 'ring-[#00f0ff]' },
  { name: 'neon-purple', label: '🔮 Synthwave', bgClass: 'bg-[#d946ef]', ringClass: 'ring-[#d946ef]' },
  { name: 'neon-green', label: '🟢 Matrix Green', bgClass: 'bg-[#10f070]', ringClass: 'ring-[#10f070]' },
  { name: 'neon-amber', label: '✨ Laser Gold', bgClass: 'bg-[#ffd000]', ringClass: 'ring-[#ffd000]' },
]

const DEFAULT_STAGE_MAPPING: Record<WorkflowStage, string> = {
  new: 'to_clarify',
  clarified: 'to_specify',
  specified: 'to_implement',
  implemented: 'to_test',
  reviewed: 'to_close',
  finished: 'finished',
}

const STAGE_CONFIGS: { id: WorkflowStage; label: string; sub: string; color: string; Icon: React.ComponentType<{ size?: number; className?: string }> }[] = [
  { id: 'new', label: '#new', sub: 'Nouveau ticket brut', color: 'cyan', Icon: Sparkles },
  { id: 'clarified', label: '#clarified', sub: 'Questions & cadrage validés', color: 'amber', Icon: HelpCircle },
  { id: 'specified', label: '#specified', sub: 'Spécification technique prête', color: 'blue', Icon: FileCode },
  { id: 'implemented', label: '#implemented', sub: 'Développement terminé sur branche', color: 'indigo', Icon: Flame },
  { id: 'reviewed', label: '#reviewed', sub: 'Revue de code & PR prête', color: 'purple', Icon: ShieldCheck },
  { id: 'finished', label: '#finished', sub: 'Ticket validé & fusionné', color: 'emerald', Icon: CheckCircle2 },
]

const STATUS_OPTIONS: { id: string; label: string; stageCategory: string }[] = [
  { id: 'to_clarify', label: 'À clarifier / Todo (Backlog) [#new]', stageCategory: 'Todo' },
  { id: 'to_specify', label: 'À spécifier (Cadré) [#clarified]', stageCategory: 'In Progress' },
  { id: 'to_implement', label: 'À implémenter (En dev) [#specified]', stageCategory: 'In Progress' },
  { id: 'to_test', label: 'À tester (En revue / QA) [#implemented]', stageCategory: 'Review' },
  { id: 'to_close', label: 'En revue / PR prête [#reviewed]', stageCategory: 'Review' },
  { id: 'finished', label: 'Terminé / Mergé [#finished]', stageCategory: 'Done' },
]

const WORKFLOW_SKILLS: { id: string; defaultName: string; code: string; desc: string; icon: React.ComponentType<{ size?: number; className?: string }>; color: string }[] = [
  { id: 'clarify', defaultName: 'Clarify', code: 'clarify-issue', desc: 'Questions de cadrage & inputs produit', icon: HelpCircle, color: 'amber' },
  { id: 'specify', defaultName: 'Specify', code: 'specify-issue', desc: 'Spécification technique Speckit', icon: FileCode, color: 'blue' },
  { id: 'implement', defaultName: 'Implement', code: 'code-issue', desc: 'Développement & codage de la story', icon: Flame, color: 'indigo' },
  { id: 'create_pr', defaultName: 'Review & PR', code: 'create-pr', desc: 'Revue de code, tests & Pull Request', icon: ShieldCheck, color: 'purple' },
  { id: 'pick', defaultName: 'Auto-Pilot', code: 'pick-issue', desc: 'Prise en charge et analyse autonome', icon: Sparkles, color: 'emerald' },
]

const extractGithubRepoFromGitUrl = (url: string): string => {
  const clean = url.trim().replace(/\.git$/, '')
  if (clean.startsWith('git@github.com:')) return clean.replace('git@github.com:', '')
  if (clean.startsWith('https://github.com/')) return clean.replace('https://github.com/', '')
  if (clean.startsWith('http://github.com/')) return clean.replace('http://github.com/', '')
  if (clean.startsWith('ssh://git@github.com/')) return clean.replace('ssh://git@github.com/', '')
  return clean
}

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

  const [activeTab, setActiveTab] = useState<ProjectTab>('general')
  
  // Section 1: Général (Titre, description, icône, couleur, projet par défaut)
  const [name, setName] = useState('')
  const [slug, setSlug] = useState('')
  const [description, setDescription] = useState('')
  const [icon, setIcon] = useState('Folder')
  const [color, setColor] = useState<AccentColor>('indigo')
  const [isDefault, setIsDefault] = useState(false)

  // Section 2: Git (Local path, URL distante git@..., init git)
  const [repoPath, setRepoPath] = useState('')
  const [gitRemoteUrl, setGitRemoteUrl] = useState('')

  // Section 3: Tracker (Type, pas de défaut, URL, Clef, Mapping)
  const [issueTracker, setIssueTracker] = useState<IssueTracker>('linear')
  const [trackerUrl, setTrackerUrl] = useState('')
  const [linearTeam, setLinearTeam] = useState('')
  const [githubRepo, setGithubRepo] = useState('')
  const [stageMapping, setStageMapping] = useState<Record<WorkflowStage, string>>(DEFAULT_STAGE_MAPPING)
  const [customInputMode, setCustomInputMode] = useState<Record<WorkflowStage, boolean>>({
    new: false,
    clarified: false,
    specified: false,
    implemented: false,
    reviewed: false,
    finished: false,
  })
  const [detectedStatuses, setDetectedStatuses] = useState<DetectedStatus[]>([])
  const [isDetectingStatuses, setIsDetectingStatuses] = useState(false)

  // Section 4: Compétences IA (Installation & Surcharge noms)
  const [skillOverrides, setSkillOverrides] = useState<Record<string, string>>({})
  const [skillsStatus, setSkillsStatus] = useState<ProjectSkillsStatus | null>(null)
  const [isLoadingSkills, setIsLoadingSkills] = useState(false)
  const [isInstallingSkills, setIsInstallingSkills] = useState(false)
  const [isInitializingGit, setIsInitializingGit] = useState(false)

  const [isSubmitting, setIsSubmitting] = useState(false)
  const [isDeleting, setIsDeleting] = useState(false)

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

  const fetchSkillsStatus = async (targetProjectId: string) => {
    setIsLoadingSkills(true)
    try {
      const res = await fetch(`/api/projects/${encodeURIComponent(targetProjectId)}/skills-status`)
      if (res.ok) {
        const data: ProjectSkillsStatus = await res.json()
        if (data) {
          setSkillsStatus({
            ...data,
            skills: Array.isArray(data.skills) ? data.skills : [],
          })
        }
      }
    } catch {
      // ignore
    } finally {
      setIsLoadingSkills(false)
    }
  }

  const fetchDetectedStatuses = async (team?: string, tracker?: IssueTracker, ghRepo?: string) => {
    setIsDetectingStatuses(true)
    try {
      const targetTeam = team !== undefined ? team : linearTeam
      const targetTracker = tracker !== undefined ? tracker : issueTracker
      const targetRepo = ghRepo !== undefined ? ghRepo : (githubRepo || extractGithubRepoFromGitUrl(gitRemoteUrl))
      const params = new URLSearchParams()
      if (targetTeam) params.append('team', targetTeam)
      if (targetTracker) params.append('tracker', targetTracker)
      if (targetRepo) params.append('repo', targetRepo)
      if (editingProject) params.append('projectId', editingProject.id)

      const res = await fetch(`/api/projects/detected-statuses?${params.toString()}`)
      if (res.ok) {
        const data: { statuses: DetectedStatus[] } = await res.json()
        setDetectedStatuses(data.statuses || [])
      }
    } catch {
      // ignore
    } finally {
      setIsDetectingStatuses(false)
    }
  }

  useEffect(() => {
    if (editingProject) {
      setName(editingProject.name)
      setSlug(editingProject.slug)
      setDescription(editingProject.description || '')
      setIcon(editingProject.icon || 'Folder')
      setColor((editingProject.color as AccentColor) || 'indigo')
      setIsDefault(editingProject.isDefault || false)

      setRepoPath(editingProject.repoPath || '')
      setGitRemoteUrl(editingProject.gitRemoteUrl || '')

      setIssueTracker(editingProject.issueTracker || 'linear')
      setTrackerUrl(editingProject.trackerUrl || '')
      setLinearTeam(editingProject.linearTeam || '')
      setGithubRepo(editingProject.githubRepo || '')
      setStageMapping(
        editingProject.stageMapping && Object.keys(editingProject.stageMapping).length > 0
          ? { ...DEFAULT_STAGE_MAPPING, ...editingProject.stageMapping }
          : DEFAULT_STAGE_MAPPING
      )
      setSkillOverrides(editingProject.skillOverrides || {})

      fetchSkillsStatus(editingProject.id)
      fetchDetectedStatuses(editingProject.linearTeam, editingProject.issueTracker, editingProject.githubRepo)
    } else {
      setName('')
      setSlug('')
      setDescription('')
      setIcon('Folder')
      setColor('indigo')
      setIsDefault(false)

      setRepoPath('')
      setGitRemoteUrl('')

      setIssueTracker('linear')
      setTrackerUrl('')
      setLinearTeam('')
      setGithubRepo('')
      setStageMapping(DEFAULT_STAGE_MAPPING)
      setSkillOverrides({})
      setSkillsStatus(null)
      fetchDetectedStatuses('', 'linear', '')
    }
    setActiveTab('general')
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
    if (!targetPath || !targetPath.trim() || isInstallingSkills) return

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
        description: `Compétences IA prêtes pour l'espace dans ${targetPath}`,
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

  const handleGitRemoteChange = (val: string) => {
    setGitRemoteUrl(val)
    const extracted = extractGithubRepoFromGitUrl(val)
    if (extracted && (!githubRepo || githubRepo.includes('/'))) {
      setGithubRepo(extracted)
    }
  }

  const handleSkillOverrideChange = (skillId: string, customName: string) => {
    setSkillOverrides(prev => ({
      ...prev,
      [skillId]: customName,
    }))
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!name.trim() || isSubmitting) return

    setIsSubmitting(true)
    try {
      const computedGithubRepo = githubRepo.trim() || extractGithubRepoFromGitUrl(gitRemoteUrl)
      const payload = {
        name: name.trim(),
        slug: slug.trim() || name.toLowerCase().replace(/[^a-z0-9]+/g, '-'),
        description: description.trim(),
        icon,
        color,
        isDefault,
        repoPath: repoPath.trim(),
        gitRemoteUrl: gitRemoteUrl.trim(),
        issueTracker,
        trackerUrl: trackerUrl.trim(),
        linearTeam: linearTeam.trim().toUpperCase(),
        githubRepo: computedGithubRepo,
        stageMapping,
        skillOverrides,
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
    if (confirm(`Êtes-vous sûr de vouloir supprimer le projet "${editingProject.name}" ? Les tâches associées seront conservées.`)) {
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
      <div className="relative w-full max-w-3xl rounded-2xl bg-[var(--bg-secondary)] border border-[var(--border-color)] shadow-2xl overflow-hidden flex flex-col max-h-[92vh]">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-3.5 border-b border-[var(--border-color)] bg-[var(--bg-tertiary)]/40 shrink-0">
          <div className="flex items-center gap-2.5">
            <div className={`w-8 h-8 rounded-xl bg-${color}-500/20 text-${color}-400 flex items-center justify-center border border-${color}-500/30`}>
              <Layers size={16} />
            </div>
            <div>
              <h3 className="text-sm font-bold text-[var(--text-primary)]">
                {editingProject ? `Paramètres : ${editingProject.name}` : 'Nouveau Projet'}
              </h3>
              <p className="text-[11px] text-[var(--text-muted)]">
                {editingProject ? `Espace dédié avec son CWD Git, tracker et compétences IA` : 'Créez un espace dédié avec son propre dépôt Git et tracker'}
              </p>
            </div>
          </div>

          <button
            type="button"
            onClick={() => {
              setIsProjectModalOpen(false)
              setEditingProject(null)
            }}
            className="p-1.5 rounded-lg text-[var(--text-muted)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-tertiary)] transition-colors cursor-pointer"
          >
            <X size={18} />
          </button>
        </div>

        {/* Navigation Tabs Bar */}
        <div className="flex items-center gap-1 px-6 pt-1.5 border-b border-[var(--border-color)] bg-[var(--bg-secondary)] shrink-0 overflow-x-auto">
          {TABS.map(tab => {
            const Icon = tab.icon
            const isSel = activeTab === tab.id
            return (
              <button
                key={tab.id}
                type="button"
                onClick={() => setActiveTab(tab.id)}
                className={`flex items-center gap-2 px-3.5 py-2 text-xs font-bold border-b-2 transition-all cursor-pointer whitespace-nowrap ${
                  isSel
                    ? 'border-[var(--accent-color)] accent-text'
                    : 'border-transparent text-[var(--text-secondary)] hover:text-[var(--text-primary)] hover:border-[var(--border-color)]'
                }`}
              >
                <Icon size={14} className={isSel ? 'text-[var(--accent-color)]' : 'text-[var(--text-muted)]'} />
                <span>{tab.label}</span>
                {tab.id === 'git' && skillsStatus && (
                  <span className={`w-2 h-2 rounded-full ${skillsStatus.isGitRepo ? 'bg-emerald-400' : 'bg-amber-400'}`} />
                )}
                {tab.id === 'skills' && skillsStatus && (
                  <span className={`text-[9px] font-mono px-1.5 py-0.2 rounded-full font-bold ${
                    skillsStatus.installedAll ? 'bg-emerald-500/20 text-emerald-300' : 'bg-amber-500/20 text-amber-300'
                  }`}>
                    {(skillsStatus.skills || []).filter(s => s.installed).length}/5
                  </span>
                )}
                {tab.id === 'tracker' && detectedStatuses.length > 0 && (
                  <span className="text-[9px] font-mono px-1.5 py-0.2 rounded-full bg-[var(--accent-light)] accent-text font-bold">
                    {detectedStatuses.length}
                  </span>
                )}
              </button>
            )
          })}
        </div>

        {/* Form Body by Tab */}
        <form onSubmit={handleSubmit} className="p-5 overflow-y-auto space-y-4 flex-1 text-xs">
          {/* ========================================================= */}
          {/* SECTION 1: GÉNÉRAL (Titre, description, icône, couleur, défaut) */}
          {/* ========================================================= */}
          {activeTab === 'general' && (
            <div className="space-y-3.5 animate-in fade-in duration-150">
              {/* Titre */}
              <div>
                <label className="block text-[10px] font-bold uppercase tracking-wider text-[var(--text-muted)] mb-1">
                  Titre du Projet *
                </label>
                <input
                  type="text"
                  required
                  value={name}
                  onChange={e => handleNameChange(e.target.value)}
                  placeholder="Ex: Mon Projet, Mobile App, Backend API..."
                  className="w-full px-3 py-1.5 text-xs rounded-xl bg-[var(--bg-tertiary)] border border-[var(--border-color)] text-[var(--text-primary)] focus:outline-none focus:border-[var(--accent-color)] font-medium"
                />
              </div>

              {/* Description */}
              <div>
                <label className="block text-[10px] font-bold uppercase tracking-wider text-[var(--text-muted)] mb-1">
                  Description
                </label>
                <input
                  type="text"
                  value={description}
                  onChange={e => setDescription(e.target.value)}
                  placeholder="Ex: Application principale Web et backend Go..."
                  className="w-full px-3 py-1.5 text-xs rounded-xl bg-[var(--bg-tertiary)] border border-[var(--border-color)] text-[var(--text-primary)] focus:outline-none focus:border-[var(--accent-color)]"
                />
              </div>

              {/* Icon & Color Selector */}
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 pt-1">
                <div>
                  <label className="block text-[10px] font-bold uppercase tracking-wider text-[var(--text-muted)] mb-1.5">
                    Icône
                  </label>
                  <div className="flex flex-wrap gap-1.5 p-2 rounded-xl bg-[var(--bg-tertiary)] border border-[var(--border-color)]">
                    {AVAILABLE_ICONS.map(({ name: iconName, Icon }) => {
                      const isSel = icon === iconName
                      return (
                        <button
                          key={iconName}
                          type="button"
                          onClick={() => setIcon(iconName)}
                          className={`p-1.5 rounded-lg flex items-center justify-center transition-all cursor-pointer ${
                            isSel
                              ? 'bg-[var(--accent-color)] text-white shadow-sm scale-105'
                              : 'text-[var(--text-muted)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-primary)]'
                          }`}
                          title={iconName}
                        >
                          <Icon size={15} />
                        </button>
                      )
                    })}
                  </div>
                </div>

                <div>
                  <label className="block text-[10px] font-bold uppercase tracking-wider text-[var(--text-muted)] mb-1.5">
                    Couleur
                  </label>
                  <div className="grid grid-cols-6 gap-1.5 p-2 rounded-xl bg-[var(--bg-tertiary)] border border-[var(--border-color)]">
                    {AVAILABLE_COLORS.map(c => {
                      const isSel = color === c.name
                      return (
                        <button
                          key={c.name}
                          type="button"
                          onClick={() => setColor(c.name)}
                          className={`h-6 rounded-lg ${c.bgClass} flex items-center justify-center transition-all cursor-pointer ${
                            isSel ? 'ring-2 ring-white ring-offset-2 ring-offset-[var(--bg-secondary)] scale-105 shadow-sm' : 'opacity-80 hover:opacity-100'
                          }`}
                          title={c.label}
                        >
                          {isSel && <Check size={11} className="text-white drop-shadow" />}
                        </button>
                      )
                    })}
                  </div>
                </div>
              </div>

              {/* Projet par défaut */}
              <div className="p-2.5 rounded-xl bg-[var(--bg-tertiary)]/70 border border-[var(--border-color)] flex items-center justify-between">
                <div>
                  <span className="text-[11px] text-[var(--text-primary)] font-bold block">
                    Projet par défaut
                  </span>
                  <span className="text-[10px] text-[var(--text-muted)]">
                    Sélectionné automatiquement à l'ouverture de l'application et lors de la création de tâches.
                  </span>
                </div>
                <label className="relative inline-flex items-center cursor-pointer shrink-0">
                  <input
                    type="checkbox"
                    checked={isDefault}
                    onChange={e => setIsDefault(e.target.checked)}
                    className="sr-only peer"
                  />
                  <div className="w-9 h-5 bg-[var(--border-color)] peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-[var(--accent-color)]"></div>
                </label>
              </div>
            </div>
          )}

          {/* ========================================================= */}
          {/* SECTION 2: GIT (Local path, URL distante git@..., init git) */}
          {/* ========================================================= */}
          {activeTab === 'git' && (
            <div className="space-y-3.5 animate-in fade-in duration-150">
              {/* Local path */}
              <div>
                <label className="block text-[10px] font-bold uppercase tracking-wider text-[var(--text-muted)] mb-1">
                  Local path (CWD d'exécution IA) *
                </label>
                <div className="relative">
                  <input
                    type="text"
                    required
                    value={repoPath}
                    onChange={e => setRepoPath(e.target.value)}
                    placeholder="Ex: /Users/username/Sources/mon-projet"
                    className="w-full pl-8 pr-3 py-1.5 text-xs rounded-xl bg-[var(--bg-tertiary)] border border-[var(--border-color)] text-[var(--text-primary)] font-mono text-[11px] focus:outline-none focus:border-[var(--accent-color)]"
                  />
                  <Folder size={14} className="absolute left-2.5 top-2.5 text-[var(--accent-color)]" />
                </div>
                <p className="text-[10px] text-[var(--text-muted)] mt-1">
                  Répertoire local sur votre machine où les agents IA exécutent le code et gèrent les branches Git.
                </p>
              </div>

              {/* URL distante au format git */}
              <div>
                <label className="block text-[10px] font-bold uppercase tracking-wider text-[var(--text-muted)] mb-1">
                  URL distante au format git (git@github.com:xxx/xxxx.git)
                </label>
                <div className="relative">
                  <input
                    type="text"
                    value={gitRemoteUrl}
                    onChange={e => handleGitRemoteChange(e.target.value)}
                    placeholder="Ex: git@github.com:org/mon-projet.git"
                    className="w-full pl-8 pr-3 py-1.5 text-xs rounded-xl bg-[var(--bg-tertiary)] border border-[var(--border-color)] text-[var(--text-primary)] font-mono text-[11px] focus:outline-none focus:border-[var(--accent-color)]"
                  />
                  <FolderGit2 size={14} className="absolute left-2.5 top-2.5 text-[var(--text-muted)]" />
                </div>
                <p className="text-[10px] text-[var(--text-muted)] mt-1">
                  Format SSH ou HTTPS utilisé pour identifier le dépôt distant et synchroniser les Pull Requests.
                </p>
              </div>

              {/* Init git local si besoin */}
              {skillsStatus && (
                <div className="p-3 rounded-xl bg-[var(--bg-tertiary)]/70 border border-[var(--border-color)] space-y-2">
                  <div className="flex items-center justify-between">
                    <span className="text-[10px] font-bold uppercase tracking-wider text-[var(--text-muted)] flex items-center gap-1.5">
                      <GitBranch size={13} className="text-[var(--accent-color)]" />
                      État du dépôt local
                    </span>
                    <span className={`text-[10px] font-mono px-2 py-0.5 rounded-full font-bold flex items-center gap-1 ${
                      skillsStatus.isGitRepo
                        ? 'bg-emerald-500/15 text-emerald-400 border border-emerald-500/30'
                        : 'bg-amber-500/15 text-amber-400 border border-amber-500/30'
                    }`}>
                      {skillsStatus.isGitRepo ? (
                        <>
                          <CheckCircle2 size={11} />
                          <span>Dépôt Git valide ({skillsStatus.gitBranch || 'main'})</span>
                        </>
                      ) : (
                        <>
                          <AlertCircle size={11} />
                          <span>Pas de dépôt Git (.git manquant)</span>
                        </>
                      )}
                    </span>
                  </div>

                  {!skillsStatus.isGitRepo && (
                    <div className="flex items-center justify-between pt-1">
                      <span className="text-[11px] text-[var(--text-muted)]">
                        Initialisez Git pour activer la gestion des branches et du code.
                      </span>
                      <button
                        type="button"
                        onClick={handleInitGit}
                        disabled={isInitializingGit || !repoPath.trim()}
                        className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-lg text-xs font-bold text-white bg-indigo-600 hover:bg-indigo-500 active:scale-95 transition-all shadow-xs shrink-0 disabled:opacity-50 cursor-pointer"
                        title="Exécuter git init dans ce dossier"
                      >
                        {isInitializingGit ? (
                          <Loader2 size={12} className="animate-spin" />
                        ) : (
                          <Zap size={12} className="text-amber-300" />
                        )}
                        <span>Initialiser Git (git init)</span>
                      </button>
                    </div>
                  )}
                </div>
              )}
            </div>
          )}

          {/* ========================================================= */}
          {/* SECTION 3: TRACKER (Type, pas de défaut, URL, Clef, Mapping) */}
          {/* ========================================================= */}
          {activeTab === 'tracker' && (
            <div className="space-y-3.5 animate-in fade-in duration-150">
              {/* Type de tracker & Clef */}
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                <div>
                  <label className="block text-[10px] font-bold uppercase tracking-wider text-[var(--text-muted)] mb-1">
                    Type de tracker
                  </label>
                  <select
                    value={issueTracker}
                    onChange={e => setIssueTracker(e.target.value as IssueTracker)}
                    className="w-full px-3 py-1.5 text-xs rounded-xl bg-[var(--bg-tertiary)] border border-[var(--border-color)] text-[var(--text-primary)] focus:outline-none focus:border-[var(--accent-color)] font-medium"
                  >
                    <option value="linear">Linear</option>
                    <option value="github">GitHub Issues</option>
                    <option value="local">Local SQLite uniquement</option>
                  </select>
                </div>

                <div>
                  <label className="block text-[10px] font-bold uppercase tracking-wider text-[var(--text-muted)] mb-1 flex items-center justify-between">
                    <span>Clef (non obligatoire)</span>
                    <span className="text-[9px] text-[var(--text-muted)] font-normal font-sans">Team Key / Préfixe</span>
                  </label>
                  <div className="relative">
                    <input
                      type="text"
                      value={linearTeam}
                      onChange={e => setLinearTeam(e.target.value.toUpperCase())}
                      placeholder="Ex: ENG, PROJ, API..."
                      className="w-full pl-8 pr-3 py-1.5 text-xs rounded-xl bg-[var(--bg-tertiary)] border border-[var(--border-color)] text-[var(--text-primary)] font-mono focus:outline-none focus:border-[var(--accent-color)] font-bold"
                    />
                    <Key size={13} className="absolute left-2.5 top-2.5 text-[var(--text-muted)]" />
                  </div>
                </div>
              </div>

              {/* URL du projet */}
              <div>
                <label className="block text-[10px] font-bold uppercase tracking-wider text-[var(--text-muted)] mb-1">
                  URL du projet / espace tracker
                </label>
                <div className="relative">
                  <input
                    type="url"
                    value={trackerUrl}
                    onChange={e => setTrackerUrl(e.target.value)}
                    placeholder="Ex: https://linear.app/my-org/project/xxx ou https://github.com/owner/repo/issues"
                    className="w-full pl-8 pr-3 py-1.5 text-xs rounded-xl bg-[var(--bg-tertiary)] border border-[var(--border-color)] text-[var(--text-primary)] font-mono text-[11px] focus:outline-none focus:border-[var(--accent-color)]"
                  />
                  <Globe size={14} className="absolute left-2.5 top-2.5 text-[var(--accent-color)]" />
                </div>
              </div>

              {/* Stage Mapping Table Card */}
              <div className="p-3.5 rounded-xl bg-[var(--bg-tertiary)]/70 border border-[var(--border-color)] space-y-2.5">
                <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2">
                  <div className="flex items-center gap-2">
                    <Sliders size={14} className="text-[var(--accent-color)]" />
                    <span className="text-[10px] font-bold uppercase tracking-wider text-[var(--text-primary)]">
                      Mapping des statuts IA ➔ Tracker
                    </span>
                    {detectedStatuses.length > 0 && (
                      <span className="text-[10px] font-mono px-1.5 py-0.2 rounded-full bg-[var(--accent-light)] accent-text font-bold">
                        {detectedStatuses.length} détectés
                      </span>
                    )}
                  </div>

                  {/* Action Toolbar */}
                  <div className="flex items-center flex-wrap gap-1.5">
                    <button
                      type="button"
                      disabled={isDetectingStatuses}
                      onClick={() => fetchDetectedStatuses()}
                      className="flex items-center gap-1 px-2 py-0.5 rounded-lg text-[10px] font-bold bg-[var(--bg-primary)] border border-[var(--border-color)] text-[var(--text-primary)] hover:border-[var(--accent-color)] hover:bg-[var(--accent-light)] transition-all cursor-pointer disabled:opacity-50"
                      title="Scanner Linear / GitHub / Base pour détecter les statuts réels"
                    >
                      <RefreshCw size={10} className={isDetectingStatuses ? 'animate-spin text-[var(--accent-color)]' : 'text-cyan-400'} />
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
                          new: findStatus(['triage', 'backlog', 'unstarted', 'to_clarify', 'todo', 'open'], 'to_clarify'),
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
                      className="flex items-center gap-1 px-2 py-0.5 rounded-lg text-[10px] font-bold bg-[var(--accent-light)] accent-text border border-[var(--accent-color)]/30 hover:opacity-90 transition-all cursor-pointer"
                      title="Associer automatiquement les statuts détectés aux 6 étapes IA"
                    >
                      <Sparkles size={10} className="text-amber-400" />
                      <span>Auto-assigner</span>
                    </button>

                    <button
                      type="button"
                      onClick={() => setStageMapping(DEFAULT_STAGE_MAPPING)}
                      className="px-2 py-0.5 rounded-lg text-[10px] font-medium bg-[var(--bg-primary)] border border-[var(--border-color)] text-[var(--text-secondary)] hover:text-[var(--text-primary)] hover:border-[var(--accent-color)]/40 transition-colors cursor-pointer"
                      title="Réinitialiser avec le flux standard"
                    >
                      Défaut
                    </button>
                  </div>
                </div>

                <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
                  {STAGE_CONFIGS.map(stage => {
                    const currentStatus = stageMapping[stage.id] || DEFAULT_STAGE_MAPPING[stage.id]
                    const isCustom = customInputMode[stage.id]
                    const StageIcon = stage.Icon

                    return (
                      <div
                        key={stage.id}
                        className="p-2 rounded-lg bg-[var(--bg-primary)] border border-[var(--border-color)] flex items-center justify-between gap-2"
                      >
                        {/* Stage Label Left */}
                        <div className="flex items-center gap-1.5 min-w-0">
                          <span className={`text-[10px] font-bold px-1.5 py-0.5 rounded-md border font-mono shrink-0 flex items-center gap-1 ${
                            stage.id === 'new' ? 'bg-cyan-500/15 text-cyan-400 border-cyan-500/30' :
                            stage.id === 'clarified' ? 'bg-amber-500/15 text-amber-400 border-amber-500/30' :
                            stage.id === 'specified' ? 'bg-blue-500/15 text-blue-400 border-blue-500/30' :
                            stage.id === 'implemented' ? 'bg-indigo-500/15 text-indigo-400 border-indigo-500/30' :
                            stage.id === 'reviewed' ? 'bg-purple-500/15 text-purple-400 border-purple-500/30' :
                            'bg-emerald-500/15 text-emerald-400 border-emerald-500/30'
                          }`}>
                            <StageIcon size={11} />
                            <span>{stage.label}</span>
                          </span>
                        </div>

                        <div className="flex items-center gap-1.5 shrink-0">
                          <ArrowRight size={11} className="text-[var(--text-muted)] shrink-0" />

                          {isCustom ? (
                            <div className="flex items-center gap-1">
                              <input
                                type="text"
                                value={currentStatus}
                                onChange={e =>
                                  setStageMapping(prev => ({
                                    ...prev,
                                    [stage.id]: e.target.value,
                                  }))
                                }
                                placeholder="Nom du statut"
                                className="w-32 px-2 py-0.5 text-xs rounded-lg bg-[var(--bg-secondary)] border border-[var(--accent-color)] text-[var(--text-primary)] font-medium focus:outline-none"
                                autoFocus
                              />
                              <button
                                type="button"
                                onClick={() =>
                                  setCustomInputMode(prev => ({ ...prev, [stage.id]: false }))
                                }
                                className="px-1.5 py-0.5 rounded-lg text-[9px] font-medium bg-[var(--bg-tertiary)] text-[var(--text-secondary)] hover:text-[var(--text-primary)] border border-[var(--border-color)] cursor-pointer"
                                title="Revenir à la liste"
                              >
                                Liste
                              </button>
                            </div>
                          ) : (
                            <div className="flex items-center gap-1">
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
                                className="w-36 px-2 py-0.5 text-xs rounded-lg bg-[var(--bg-secondary)] border border-[var(--border-color)] text-[var(--text-primary)] focus:outline-none focus:border-[var(--accent-color)] font-medium"
                              >
                                {detectedStatuses.length > 0 && (
                                  <optgroup label="✨ Statuts détectés">
                                    {detectedStatuses.map(st => (
                                      <option key={`det-${st.id}`} value={st.name}>
                                        {st.name} {st.source ? `(${st.source})` : ''}
                                      </option>
                                    ))}
                                  </optgroup>
                                )}

                                <optgroup label="📋 Statuts Taskacao">
                                  {STATUS_OPTIONS.map(opt => (
                                    <option key={opt.id} value={opt.id}>
                                      {opt.label}
                                    </option>
                                  ))}
                                </optgroup>

                                <optgroup label="✏️ Personnalisé">
                                  <option value="__custom__">➕ Saisir libre...</option>
                                </optgroup>
                              </select>

                              <button
                                type="button"
                                onClick={() =>
                                  setCustomInputMode(prev => ({ ...prev, [stage.id]: true }))
                                }
                                className="p-0.5 rounded-lg text-[var(--text-muted)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-tertiary)] transition-colors cursor-pointer"
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
            </div>
          )}

          {/* ========================================================= */}
          {/* SECTION 4: COMPÉTENCES IA (Installation & Surcharge noms)   */}
          {/* ========================================================= */}
          {activeTab === 'skills' && (
            <div className="space-y-3.5 animate-in fade-in duration-150">
              {/* Header card with installation action */}
              <div className="p-3.5 rounded-xl bg-[var(--bg-tertiary)]/70 border border-[var(--border-color)] flex flex-col sm:flex-row sm:items-center justify-between gap-3">
                <div className="flex items-center gap-2.5 min-w-0">
                  <div className="w-8 h-8 rounded-xl bg-[var(--accent-light)] accent-text flex items-center justify-center shrink-0 border border-[var(--accent-color)]/30">
                    <Sparkles size={16} />
                  </div>
                  <div>
                    <div className="flex items-center gap-2">
                      <span className="text-xs font-bold text-[var(--text-primary)]">
                        Installation des Compétences IA
                      </span>
                      {isLoadingSkills ? (
                        <span className="inline-flex items-center gap-1 text-[9px] font-mono text-[var(--text-muted)]">
                          <Loader2 size={10} className="animate-spin text-[var(--accent-color)]" />
                          <span>Vérification...</span>
                        </span>
                      ) : skillsStatus ? (
                        <span className={`text-[9px] font-mono px-1.5 py-0.2 rounded-full font-bold ${
                          skillsStatus.installedAll ? 'bg-emerald-500/20 text-emerald-300 border border-emerald-500/40' : 'bg-amber-500/20 text-amber-300 border border-amber-500/40'
                        }`}>
                          {skillsStatus.installedAll ? '5/5 Prêtes' : `${(skillsStatus.skills || []).filter(s => s.installed).length}/5 installées`}
                        </span>
                      ) : null}
                    </div>
                    <span className="text-[10px] text-[var(--text-muted)] truncate block">
                      📁 CWD : {repoPath ? repoPath.split('/').slice(-2).join('/') : 'Non configuré'}
                    </span>
                  </div>
                </div>

                <button
                  type="button"
                  disabled={isInstallingSkills || !repoPath.trim()}
                  onClick={handleInstallSkills}
                  className="flex items-center gap-1.5 px-3 py-1.5 rounded-xl text-xs font-bold bg-[var(--accent-color)] hover:opacity-90 text-white shadow-xs transition-all hover:scale-102 active:scale-95 disabled:opacity-50 cursor-pointer shrink-0"
                >
                  {isInstallingSkills ? (
                    <Loader2 size={13} className="animate-spin text-white" />
                  ) : (
                    <Download size={13} />
                  )}
                  <span>
                    {isInstallingSkills
                      ? 'Installation...'
                      : skillsStatus?.installedAll
                      ? 'Réinstaller / Sync'
                      : '⚡ Installer les Skills'}
                  </span>
                </button>
              </div>

              {/* Skills Overrides Table List */}
              <div className="p-3.5 rounded-xl bg-[var(--bg-tertiary)]/70 border border-[var(--border-color)] space-y-2.5">
                <div className="flex items-center justify-between pb-1 border-b border-[var(--border-color)]">
                  <span className="text-[10px] font-bold uppercase tracking-wider text-[var(--text-primary)]">
                    Surcharge des Noms des Skills
                  </span>
                  <span className="text-[10px] text-[var(--text-muted)]">
                    Personnalisez le libellé de chaque compétence
                  </span>
                </div>

                <div className="space-y-2">
                  {WORKFLOW_SKILLS.map(s => {
                    const isInst = (skillsStatus?.skills || []).find(sk => sk.id === s.id)?.installed || false
                    const SkillIcon = s.icon
                    const customValue = skillOverrides[s.id] || ''

                    return (
                      <div
                        key={s.id}
                        className="p-2.5 rounded-xl bg-[var(--bg-primary)] border border-[var(--border-color)] flex flex-col sm:flex-row sm:items-center justify-between gap-2.5"
                      >
                        {/* Skill info Left */}
                        <div className="flex items-center gap-2.5 min-w-[200px]">
                          <div className={`w-6 h-6 rounded-lg flex items-center justify-center text-${s.color}-400 bg-${s.color}-500/15 shrink-0 border border-${s.color}-500/30`}>
                            <SkillIcon size={13} />
                          </div>
                          <div className="flex flex-col min-w-0">
                            <div className="flex items-center gap-1.5">
                              <span className="font-bold text-xs text-[var(--text-primary)]">
                                {s.defaultName}
                              </span>
                              <span className="text-[9px] font-mono text-[var(--text-muted)] bg-[var(--bg-tertiary)] px-1 py-0.2 rounded-md">
                                /{s.code}
                              </span>
                            </div>
                            <span className="text-[10px] text-[var(--text-muted)] truncate max-w-[220px]">
                              {s.desc}
                            </span>
                          </div>
                        </div>

                        {/* Custom Name Override Input Right */}
                        <div className="flex items-center gap-2 flex-1 justify-end">
                          <div className="relative flex-1 max-w-[260px]">
                            <input
                              type="text"
                              value={customValue}
                              onChange={e => handleSkillOverrideChange(s.id, e.target.value)}
                              placeholder={`Surcharge : ${s.defaultName}`}
                              className="w-full px-2.5 py-1 text-xs rounded-lg bg-[var(--bg-secondary)] border border-[var(--border-color)] text-[var(--text-primary)] focus:outline-none focus:border-[var(--accent-color)] font-medium"
                            />
                            {customValue && (
                              <button
                                type="button"
                                onClick={() => handleSkillOverrideChange(s.id, '')}
                                className="absolute right-1.5 top-1.5 text-[var(--text-muted)] hover:text-rose-400 p-0.5"
                                title="Réinitialiser au nom par défaut"
                              >
                                <RotateCcw size={11} />
                              </button>
                            )}
                          </div>

                          <div className="shrink-0 w-20 text-right">
                            {isInst ? (
                              <span className="inline-flex items-center gap-1 text-[10px] font-bold text-emerald-400">
                                <Check size={11} />
                                <span>Installé</span>
                              </span>
                            ) : (
                              <span className="text-[10px] font-medium text-[var(--text-muted)]">
                                Non installé
                              </span>
                            )}
                          </div>
                        </div>
                      </div>
                    )
                  })}
                </div>
              </div>
            </div>
          )}
        </form>

        {/* Modal Footer */}
        <div className="flex items-center justify-between px-6 py-3.5 border-t border-[var(--border-color)] bg-[var(--bg-tertiary)]/40 shrink-0">
          <div>
            {editingProject && !editingProject.isDefault && (
              <button
                type="button"
                onClick={handleDelete}
                disabled={isDeleting}
                className="flex items-center gap-1.5 px-3 py-1.5 rounded-xl text-xs font-semibold text-rose-400 bg-rose-500/10 hover:bg-rose-500/20 border border-rose-500/30 transition-colors cursor-pointer"
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
              className="px-4 py-2 rounded-xl text-xs font-medium text-[var(--text-secondary)] hover:bg-[var(--bg-tertiary)] transition-colors cursor-pointer"
            >
              {t.taskModal.cancel}
            </button>
            <button
              type="button"
              onClick={handleSubmit}
              disabled={isSubmitting || !name.trim()}
              className="px-5 py-2 rounded-xl text-xs font-bold text-white accent-bg shadow-md hover:opacity-90 active:scale-95 flex items-center gap-1.5 transition-all disabled:opacity-50 cursor-pointer"
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
