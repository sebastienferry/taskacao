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
  Bot,
  Info,
} from 'lucide-react'
import { useApp } from '../context/AppContext'
import type { AccentColor, IssueTracker, ProjectSkillsStatus, WorkflowStage, DetectedStatus, AIProvider, SpecFramework } from '../types'

type ProjectTab = 'general' | 'git' | 'agent' | 'tracker' | 'skills'

const TABS: { id: ProjectTab; label: string; icon: React.FC<{ size?: number; className?: string }> }[] = [
  { id: 'general', label: 'Général', icon: Folder },
  { id: 'git', label: 'Git & Worktrees', icon: GitBranch },
  { id: 'agent', label: 'Agent IA & CLI', icon: Bot },
  { id: 'tracker', label: 'Tracker', icon: Sliders },
  { id: 'skills', label: 'Compétences IA & SDD', icon: Sparkles },
]

const AI_PROVIDERS: { id: AIProvider; label: string; sub: string; defaultCmd: string; icon: string }[] = [
  { id: 'agy', label: 'AGY CLI (Google Antigravity)', sub: 'Agent autonome DeepMind & outils natifs', defaultCmd: 'agy --dangerously-skip-permissions -p "{prompt}"', icon: '🤖' },
  { id: 'claude', label: 'Claude Code CLI (Anthropic)', sub: 'Agent Terminal Claude 3.7 Sonnet', defaultCmd: 'claude --dangerously-skip-permissions -p "{prompt}"', icon: '🧠' },
  { id: 'vibe', label: 'Mistral Vibe', sub: 'CLI Agentic Mistral Open Source', defaultCmd: 'vibe -p "{prompt}" --auto-approve', icon: '⚡' },
  { id: 'gemini', label: 'Gemini Code Assist CLI', sub: 'Google Cloud Gemini CLI', defaultCmd: 'gemini -p "{prompt}"', icon: '✨' },
  { id: 'cursor', label: 'Cursor Agent CLI', sub: 'Cursor Editor Agent Headless', defaultCmd: 'cursor agent -p "{prompt}"', icon: '📐' },
  { id: 'codex', label: 'Codex / Custom Shell', sub: 'CLI personnalisé ou script zsh / bash', defaultCmd: 'codex run "{prompt}"', icon: '💻' },
  { id: 'custom', label: 'Commande Personnalisée', sub: 'Modèle de commande arbitraire', defaultCmd: '{prompt}', icon: '⚙️' },
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
    settings,
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

  // Section 3: Agent IA & CLI
  const [aiProvider, setAiProvider] = useState<AIProvider | ''>('')
  const [aiCommandTemplate, setAiCommandTemplate] = useState('')
  const [useCustomAgent, setUseCustomAgent] = useState(false)

  // Section 4: Compétences IA & Framework SDD
  const [specFramework, setSpecFramework] = useState<SpecFramework>('speckit')
  const [skillOverrides, setSkillOverrides] = useState<Record<string, string>>({})
  const [skillsStatus, setSkillsStatus] = useState<ProjectSkillsStatus | null>(null)
  const [isLoadingSkills, setIsLoadingSkills] = useState(false)
  const [isInstallingSkills, setIsInstallingSkills] = useState(false)
  const [isInitializingGit, setIsInitializingGit] = useState(false)

  // Section 5: Tracker (Type, pas de défaut, URL, Clef, Mapping)
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
          if (data.specFramework) {
            setSpecFramework(data.specFramework)
          }
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

      const hasCustomAgent = Boolean(editingProject.aiProvider || editingProject.aiCommandTemplate)
      setUseCustomAgent(hasCustomAgent)
      setAiProvider(editingProject.aiProvider || '')
      setAiCommandTemplate(editingProject.aiCommandTemplate || '')
      setSpecFramework(editingProject.specFramework || settings.specFramework || 'speckit')

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

      setUseCustomAgent(false)
      setAiProvider('')
      setAiCommandTemplate('')
      setSpecFramework(settings.specFramework || 'speckit')

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
  }, [editingProject, isProjectModalOpen, settings.specFramework])

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
          specFramework,
          aiProvider: useCustomAgent ? (aiProvider || settings.aiProvider) : settings.aiProvider,
          aiCommandTemplate: useCustomAgent ? aiCommandTemplate : settings.aiCommandTemplate,
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
        title: 'Compétences IA & SDD scaffoldées !',
        description: `Compétences (${specFramework === 'openfeature' ? 'Open Feature' : 'SpecKit'}) installées dans le projet racine et tous les worktrees.`,
      })
    } catch (err: any) {
      addToast({
        type: 'error',
        title: 'Échec du scaffolding',
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
        aiProvider: useCustomAgent && aiProvider ? (aiProvider as AIProvider) : undefined,
        aiCommandTemplate: useCustomAgent && aiCommandTemplate.trim() ? aiCommandTemplate.trim() : undefined,
        specFramework,
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
                {editingProject ? `Espace dédié avec son CWD Git, agent IA, tracker et compétences SDD` : 'Créez un espace dédié avec son propre dépôt Git, agent IA et tracker'}
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
                {tab.id === 'agent' && useCustomAgent && (
                  <span className="w-2 h-2 rounded-full bg-[var(--accent-color)]" />
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

              {/* Slug & Projet par défaut */}
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 pt-2">
                <div>
                  <label className="block text-[10px] font-bold uppercase tracking-wider text-[var(--text-muted)] mb-1">
                    Identifiant / Slug
                  </label>
                  <input
                    type="text"
                    value={slug}
                    onChange={e => setSlug(e.target.value.toLowerCase().replace(/[^a-z0-9]+/g, '-'))}
                    placeholder="mon-projet"
                    className="w-full px-3 py-1.5 text-xs rounded-xl bg-[var(--bg-tertiary)] border border-[var(--border-color)] text-[var(--text-primary)] font-mono focus:outline-none focus:border-[var(--accent-color)]"
                  />
                </div>

                <div className="flex items-center gap-2 pt-4">
                  <input
                    type="checkbox"
                    id="isDefault"
                    checked={isDefault}
                    onChange={e => setIsDefault(e.target.checked)}
                    className="rounded border-[var(--border-color)] text-[var(--accent-color)] focus:ring-[var(--accent-color)] cursor-pointer"
                  />
                  <label htmlFor="isDefault" className="text-xs text-[var(--text-primary)] cursor-pointer font-medium">
                    Définir comme projet par défaut
                  </label>
                </div>
              </div>
            </div>
          )}

          {/* ========================================================= */}
          {/* SECTION 2: GIT (Chemin Local CWD, Remote URL, Init Git)   */}
          {/* ========================================================= */}
          {activeTab === 'git' && (
            <div className="space-y-3.5 animate-in fade-in duration-150">
              {/* Repo Local Path */}
              <div>
                <div className="flex items-center justify-between mb-1">
                  <label className="block text-[10px] font-bold uppercase tracking-wider text-[var(--text-muted)]">
                    Dossier du Dépôt Local (CWD pour les Skills IA)
                  </label>
                  {skillsStatus && (
                    <span className={`text-[10px] font-mono font-bold flex items-center gap-1 ${
                      skillsStatus.pathExists ? 'text-emerald-400' : 'text-rose-400'
                    }`}>
                      {skillsStatus.pathExists ? '✓ Dossier existant' : '✗ Dossier introuvable'}
                    </span>
                  )}
                </div>
                <div className="relative">
                  <input
                    type="text"
                    value={repoPath}
                    onChange={e => setRepoPath(e.target.value)}
                    placeholder="/Users/username/Sources/my-app ou ."
                    className="w-full pl-8 pr-3 py-1.5 text-xs rounded-xl bg-[var(--bg-tertiary)] border border-[var(--border-color)] text-[var(--text-primary)] font-mono focus:outline-none focus:border-[var(--accent-color)]"
                  />
                  <FolderGit2 size={14} className="absolute left-2.5 top-2.5 text-[var(--text-muted)]" />
                </div>
                <span className="text-[10px] text-[var(--text-muted)] mt-1 block">
                  Répertoire racine dans lequel s'exécutent les commandes git, worktrees et agents autonomes.
                </span>
              </div>

              {/* Remote URL */}
              <div>
                <label className="block text-[10px] font-bold uppercase tracking-wider text-[var(--text-muted)] mb-1">
                  URL du Dépôt Distant (Git Remote)
                </label>
                <div className="relative">
                  <input
                    type="text"
                    value={gitRemoteUrl}
                    onChange={e => handleGitRemoteChange(e.target.value)}
                    placeholder="git@github.com:owner/repo.git ou https://github.com/owner/repo"
                    className="w-full pl-8 pr-3 py-1.5 text-xs rounded-xl bg-[var(--bg-tertiary)] border border-[var(--border-color)] text-[var(--text-primary)] font-mono focus:outline-none focus:border-[var(--accent-color)]"
                  />
                  <Globe size={14} className="absolute left-2.5 top-2.5 text-[var(--text-muted)]" />
                </div>
              </div>

              {/* Status & Git Init card */}
              <div className="p-3.5 rounded-xl bg-[var(--bg-tertiary)]/70 border border-[var(--border-color)] flex items-center justify-between gap-3">
                <div className="flex items-center gap-2.5 min-w-0">
                  <div className={`w-7 h-7 rounded-lg flex items-center justify-center shrink-0 ${
                    skillsStatus?.isGitRepo ? 'bg-emerald-500/20 text-emerald-400' : 'bg-amber-500/20 text-amber-400'
                  }`}>
                    <GitBranch size={15} />
                  </div>
                  <div>
                    <span className="text-xs font-bold text-[var(--text-primary)] block">
                      {skillsStatus?.isGitRepo ? `Dépôt Git actif (branche: ${skillsStatus.gitBranch || 'main'})` : 'Aucun dépôt Git initialisé'}
                    </span>
                    <span className="text-[10px] text-[var(--text-muted)] truncate block">
                      {skillsStatus?.isGitRepo
                        ? `Worktrees & commits opérationnels dans ce répertoire`
                        : `Initialisez git dans le dossier pour activer les worktrees et les branches de tâches`}
                    </span>
                  </div>
                </div>

                {!skillsStatus?.isGitRepo && repoPath && (
                  <button
                    type="button"
                    disabled={isInitializingGit}
                    onClick={handleInitGit}
                    className="flex items-center gap-1.5 px-3 py-1.5 rounded-xl text-xs font-bold bg-indigo-600 hover:bg-indigo-500 text-white shadow-xs transition-all cursor-pointer shrink-0 disabled:opacity-50"
                  >
                    {isInitializingGit ? <Loader2 size={13} className="animate-spin text-white" /> : <Sparkles size={13} />}
                    <span>{isInitializingGit ? 'Initialisation...' : 'Initialiser Git'}</span>
                  </button>
                )}
              </div>
            </div>
          )}

          {/* ========================================================= */}
          {/* SECTION 3: AGENT IA & CLI (Configuration du moteur/CLI)   */}
          {/* ========================================================= */}
          {activeTab === 'agent' && (
            <div className="space-y-4 animate-in fade-in duration-150">
              {/* Header card */}
              <div className="p-3.5 rounded-xl bg-[var(--bg-tertiary)]/70 border border-[var(--border-color)]">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2.5">
                    <div className="w-8 h-8 rounded-xl bg-[var(--accent-light)] accent-text flex items-center justify-center shrink-0 border border-[var(--accent-color)]/30">
                      <Bot size={16} />
                    </div>
                    <div>
                      <span className="text-xs font-bold text-[var(--text-primary)] block">
                        Moteur IA & Ligne de Commande pour ce Projet
                      </span>
                      <span className="text-[10px] text-[var(--text-muted)] block">
                        Personnalisez le CLI utilisé (AGY, Claude Code, Vibe, Cursor...) et les options de prompt pour ce dépôt.
                      </span>
                    </div>
                  </div>

                  <div className="flex items-center gap-1 bg-[var(--bg-secondary)] p-1 rounded-xl border border-[var(--border-color)]">
                    <button
                      type="button"
                      onClick={() => setUseCustomAgent(false)}
                      className={`px-2.5 py-1 text-[11px] font-bold rounded-lg transition-all cursor-pointer ${
                        !useCustomAgent
                          ? 'bg-[var(--accent-color)] text-white shadow-xs'
                          : 'text-[var(--text-secondary)] hover:text-[var(--text-primary)]'
                      }`}
                    >
                      Hériter du Global
                    </button>
                    <button
                      type="button"
                      onClick={() => {
                        setUseCustomAgent(true)
                        if (!aiProvider) setAiProvider(settings.aiProvider || 'agy')
                        if (!aiCommandTemplate) setAiCommandTemplate(settings.aiCommandTemplate || 'agy -p "{prompt}"')
                      }}
                      className={`px-2.5 py-1 text-[11px] font-bold rounded-lg transition-all cursor-pointer ${
                        useCustomAgent
                          ? 'bg-[var(--accent-color)] text-white shadow-xs'
                          : 'text-[var(--text-secondary)] hover:text-[var(--text-primary)]'
                      }`}
                    >
                      Spécifique au Projet
                    </button>
                  </div>
                </div>
              </div>

              {!useCustomAgent ? (
                <div className="p-4 rounded-xl bg-[var(--bg-primary)] border border-dashed border-[var(--border-color)] flex items-center gap-3 text-[var(--text-secondary)]">
                  <Info size={16} className="text-[var(--accent-color)] shrink-0" />
                  <div className="text-xs">
                    <span className="font-semibold text-[var(--text-primary)]">Configuration Globale Active : </span>
                    <span className="font-mono text-[var(--accent-color)] font-bold">{settings.aiProvider.toUpperCase()}</span>
                    <span className="text-[var(--text-muted)] block mt-0.5 font-mono text-[11px]">
                      Modèle de commande : {settings.aiCommandTemplate || 'agy -p "{prompt}"'}
                    </span>
                  </div>
                </div>
              ) : (
                <div className="space-y-3.5 animate-in fade-in duration-150">
                  {/* Select AI Provider */}
                  <div>
                    <label className="block text-[10px] font-bold uppercase tracking-wider text-[var(--text-muted)] mb-1.5">
                      Fournisseur de l'Agent IA
                    </label>
                    <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
                      {AI_PROVIDERS.map(p => {
                        const isSel = (aiProvider || settings.aiProvider) === p.id
                        return (
                          <button
                            key={p.id}
                            type="button"
                            onClick={() => {
                              setAiProvider(p.id)
                              if (!aiCommandTemplate || aiCommandTemplate.startsWith('agy') || aiCommandTemplate.startsWith('claude') || aiCommandTemplate.startsWith('vibe') || aiCommandTemplate.startsWith('gemini') || aiCommandTemplate.startsWith('cursor')) {
                                setAiCommandTemplate(p.defaultCmd)
                              }
                            }}
                            className={`p-2.5 rounded-xl border text-left transition-all cursor-pointer flex items-start gap-2.5 ${
                              isSel
                                ? 'bg-[var(--accent-light)] border-[var(--accent-color)] shadow-xs'
                                : 'bg-[var(--bg-tertiary)] border-[var(--border-color)] hover:border-[var(--accent-color)]/50'
                            }`}
                          >
                            <span className="text-lg">{p.icon}</span>
                            <div className="min-w-0 flex-1">
                              <span className={`text-xs font-bold block truncate ${isSel ? 'accent-text' : 'text-[var(--text-primary)]'}`}>
                                {p.label}
                              </span>
                              <span className="text-[10px] text-[var(--text-muted)] block truncate">
                                {p.sub}
                              </span>
                            </div>
                            {isSel && <Check size={14} className="text-[var(--accent-color)] shrink-0 mt-0.5" />}
                          </button>
                        )
                      })}
                    </div>
                  </div>

                  {/* AI Command Line Template */}
                  <div>
                    <div className="flex items-center justify-between mb-1">
                      <label className="block text-[10px] font-bold uppercase tracking-wider text-[var(--text-muted)]">
                        Modèle de Commande Ligne de Commande (CLI Template)
                      </label>
                      <span className="text-[10px] text-[var(--text-muted)] font-mono">
                        Token requis : <code className="text-amber-400 font-bold">{'{prompt}'}</code>
                      </span>
                    </div>
                    <div className="relative">
                      <input
                        type="text"
                        value={aiCommandTemplate}
                        onChange={e => setAiCommandTemplate(e.target.value)}
                        placeholder='agy -p "{prompt}" --options...'
                        className="w-full pl-8 pr-3 py-2 text-xs rounded-xl bg-[var(--bg-tertiary)] border border-[var(--border-color)] text-[var(--text-primary)] font-mono focus:outline-none focus:border-[var(--accent-color)]"
                      />
                      <Terminal size={14} className="absolute left-2.5 top-2.5 text-[var(--accent-color)]" />
                    </div>

                    {/* Presets */}
                    <div className="flex items-center flex-wrap gap-1.5 mt-2">
                      <span className="text-[10px] text-[var(--text-muted)] mr-1 font-semibold">Presets :</span>
                      {[
                        { label: 'agy --dangerously-skip-permissions -p "{prompt}"', cmd: 'agy --dangerously-skip-permissions -p "{prompt}"' },
                        { label: 'agy -i "{prompt}"', cmd: 'agy -i "{prompt}"' },
                        { label: 'claude --dangerously-skip-permissions -p "{prompt}"', cmd: 'claude --dangerously-skip-permissions -p "{prompt}"' },
                        { label: 'vibe -p "{prompt}" --auto-approve', cmd: 'vibe -p "{prompt}" --auto-approve' },
                        { label: 'gemini -p "{prompt}"', cmd: 'gemini -p "{prompt}"' },
                        { label: 'cursor agent -p "{prompt}"', cmd: 'cursor agent -p "{prompt}"' },
                      ].map(pr => (
                        <button
                          key={pr.cmd}
                          type="button"
                          onClick={() => setAiCommandTemplate(pr.cmd)}
                          className="px-2 py-0.5 rounded-lg text-[10px] font-mono bg-[var(--bg-primary)] border border-[var(--border-color)] text-[var(--text-secondary)] hover:text-[var(--text-primary)] hover:border-[var(--accent-color)] transition-colors cursor-pointer"
                        >
                          {pr.label}
                        </button>
                      ))}
                    </div>

                    {/* Variable tokens guide */}
                    <div className="p-2.5 rounded-xl bg-[var(--bg-primary)] border border-[var(--border-color)] mt-2 flex items-center flex-wrap gap-2 text-[10px] text-[var(--text-muted)]">
                      <span className="font-bold text-[var(--text-secondary)]">Variables disponibles :</span>
                      {['{prompt}', '{issueKey}', '{issueTitle}', '{repoPath}', '{branchName}'].map(tag => (
                        <span key={tag} className="font-mono bg-[var(--bg-tertiary)] px-1.5 py-0.5 rounded text-[var(--text-primary)] border border-[var(--border-color)]">
                          {tag}
                        </span>
                      ))}
                    </div>
                  </div>
                </div>
              )}
            </div>
          )}

          {/* ========================================================= */}
          {/* SECTION 4: TRACKER (Linear, GitHub, Jira, Stage Mapping) */}
          {/* ========================================================= */}
          {activeTab === 'tracker' && (
            <div className="space-y-3.5 animate-in fade-in duration-150">
              {/* Tracker Type Radio Pills */}
              <div>
                <label className="block text-[10px] font-bold uppercase tracking-wider text-[var(--text-muted)] mb-1.5">
                  Type de Tracker d'Issues
                </label>
                <div className="grid grid-cols-2 sm:grid-cols-4 gap-2">
                  {[
                    { id: 'linear', label: 'Linear', icon: '⚡' },
                    { id: 'github', label: 'GitHub Issues', icon: '🐙' },
                    { id: 'jira', label: 'Jira Software', icon: '🔷' },
                    { id: 'local', label: 'Local Uniquement', icon: '💾' },
                  ].map(tItem => {
                    const isSel = issueTracker === tItem.id
                    return (
                      <button
                        key={tItem.id}
                        type="button"
                        onClick={() => {
                          const newTrk = tItem.id as IssueTracker
                          setIssueTracker(newTrk)
                          fetchDetectedStatuses(undefined, newTrk)
                        }}
                        className={`p-2 rounded-xl border text-center font-bold text-xs flex items-center justify-center gap-1.5 transition-all cursor-pointer ${
                          isSel
                            ? 'bg-[var(--accent-light)] border-[var(--accent-color)] accent-text shadow-xs'
                            : 'bg-[var(--bg-tertiary)] border-[var(--border-color)] text-[var(--text-secondary)] hover:text-[var(--text-primary)]'
                        }`}
                      >
                        <span>{tItem.icon}</span>
                        <span>{tItem.label}</span>
                      </button>
                    )
                  })}
                </div>
              </div>

              {/* Team Key & Github Repo inputs */}
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                {issueTracker === 'linear' && (
                  <div>
                    <label className="block text-[10px] font-bold uppercase tracking-wider text-[var(--text-muted)] mb-1">
                      Préfixe / Équipe Linear
                    </label>
                    <div className="relative">
                      <input
                        type="text"
                        value={linearTeam}
                        onChange={e => {
                          const val = e.target.value.toUpperCase()
                          setLinearTeam(val)
                          fetchDetectedStatuses(val)
                        }}
                        placeholder="Ex: ENG, PROD, API..."
                        className="w-full pl-8 pr-3 py-1.5 text-xs rounded-xl bg-[var(--bg-tertiary)] border border-[var(--border-color)] text-[var(--text-primary)] font-mono uppercase focus:outline-none focus:border-[var(--accent-color)]"
                      />
                      <Key size={14} className="absolute left-2.5 top-2.5 text-[var(--accent-color)]" />
                    </div>
                  </div>
                )}

                {issueTracker === 'github' && (
                  <div>
                    <label className="block text-[10px] font-bold uppercase tracking-wider text-[var(--text-muted)] mb-1">
                      Dépôt GitHub (owner/repo)
                    </label>
                    <div className="relative">
                      <input
                        type="text"
                        value={githubRepo}
                        onChange={e => {
                          setGithubRepo(e.target.value)
                          fetchDetectedStatuses(undefined, 'github', e.target.value)
                        }}
                        placeholder="owner/nom-du-repo"
                        className="w-full pl-8 pr-3 py-1.5 text-xs rounded-xl bg-[var(--bg-tertiary)] border border-[var(--border-color)] text-[var(--text-primary)] font-mono focus:outline-none focus:border-[var(--accent-color)]"
                      />
                      <Globe size={14} className="absolute left-2.5 top-2.5 text-[var(--accent-color)]" />
                    </div>
                  </div>
                )}

                <div className={issueTracker === 'local' ? 'col-span-2' : ''}>
                  <label className="block text-[10px] font-bold uppercase tracking-wider text-[var(--text-muted)] mb-1">
                    URL du Tracker
                  </label>
                  <div className="relative">
                    <input
                      type="text"
                      value={trackerUrl}
                      onChange={e => setTrackerUrl(e.target.value)}
                      placeholder="https://linear.app/team/project/..."
                      className="w-full pl-8 pr-3 py-1.5 text-xs rounded-xl bg-[var(--bg-tertiary)] border border-[var(--border-color)] text-[var(--text-primary)] focus:outline-none focus:border-[var(--accent-color)]"
                    />
                    <Globe size={14} className="absolute left-2.5 top-2.5 text-[var(--accent-color)]" />
                  </div>
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
          {/* SECTION 5: COMPÉTENCES IA & SDD (SpecKit vs OpenFeature)   */}
          {/* ========================================================= */}
          {activeTab === 'skills' && (
            <div className="space-y-4 animate-in fade-in duration-150">
              {/* SDD Framework Selection Cards */}
              <div>
                <label className="block text-[10px] font-bold uppercase tracking-wider text-[var(--text-muted)] mb-1.5">
                  Framework Spec-Driven Design (SDD) pour ce Projet
                </label>
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-2.5">
                  <button
                    type="button"
                    onClick={() => setSpecFramework('speckit')}
                    className={`p-3 rounded-xl border text-left transition-all cursor-pointer flex items-start gap-3 ${
                      specFramework === 'speckit'
                        ? 'bg-[var(--accent-light)] border-[var(--accent-color)] shadow-xs'
                        : 'bg-[var(--bg-tertiary)] border-[var(--border-color)] hover:border-[var(--accent-color)]/50'
                    }`}
                  >
                    <div className={`p-2 rounded-xl shrink-0 ${specFramework === 'speckit' ? 'bg-[var(--accent-color)] text-white' : 'bg-[var(--bg-primary)] text-[var(--text-muted)]'}`}>
                      <FileCode size={18} />
                    </div>
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-1.5 mb-0.5">
                        <span className={`font-bold text-xs ${specFramework === 'speckit' ? 'accent-text' : 'text-[var(--text-primary)]'}`}>
                          SpecKit (Speckit SDD)
                        </span>
                        <span className="text-[9px] font-mono px-1.5 py-0.2 rounded-full font-bold bg-blue-500/20 text-blue-300 border border-blue-500/30">
                          Recommandé
                        </span>
                      </div>
                      <span className="text-[10px] text-[var(--text-muted)] leading-relaxed block">
                        Spécifications techniques standardisées, User Stories, BDD Given/When/Then et architecture Mermaid.
                      </span>
                    </div>
                  </button>

                  <button
                    type="button"
                    onClick={() => setSpecFramework('openfeature')}
                    className={`p-3 rounded-xl border text-left transition-all cursor-pointer flex items-start gap-3 ${
                      specFramework === 'openfeature'
                        ? 'bg-[var(--accent-light)] border-[var(--accent-color)] shadow-xs'
                        : 'bg-[var(--bg-tertiary)] border-[var(--border-color)] hover:border-[var(--accent-color)]/50'
                    }`}
                  >
                    <div className={`p-2 rounded-xl shrink-0 ${specFramework === 'openfeature' ? 'bg-[var(--accent-color)] text-white' : 'bg-[var(--bg-primary)] text-[var(--text-muted)]'}`}>
                      <Sparkles size={18} />
                    </div>
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-1.5 mb-0.5">
                        <span className={`font-bold text-xs ${specFramework === 'openfeature' ? 'accent-text' : 'text-[var(--text-primary)]'}`}>
                          Open Feature (OpenFeature SDD)
                        </span>
                        <span className="text-[9px] font-mono px-1.5 py-0.2 rounded-full font-bold bg-purple-500/20 text-purple-300 border border-purple-500/30">
                          Feature Flags
                        </span>
                      </div>
                      <span className="text-[10px] text-[var(--text-muted)] leading-relaxed block">
                        Feature Flags déclaratifs, Evaluation Contexts, Hooks SDK OpenFeature et gestion du cycle de vie du flag.
                      </span>
                    </div>
                  </button>
                </div>
              </div>

              {/* Scaffolding Action Header */}
              <div className="p-3.5 rounded-xl bg-[var(--bg-tertiary)]/70 border border-[var(--border-color)] flex flex-col sm:flex-row sm:items-center justify-between gap-3">
                <div className="flex items-center gap-2.5 min-w-0">
                  <div className="w-8 h-8 rounded-xl bg-[var(--accent-light)] accent-text flex items-center justify-center shrink-0 border border-[var(--accent-color)]/30">
                    <Sparkles size={16} />
                  </div>
                  <div>
                    <div className="flex items-center gap-2">
                      <span className="text-xs font-bold text-[var(--text-primary)]">
                        Scaffolding des Compétences & Worktrees
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
                      {skillsStatus?.worktreesCount ? ` • ${skillsStatus.worktreesCount} worktree(s) couvert(s)` : ''}
                    </span>
                  </div>
                </div>

                <button
                  type="button"
                  disabled={isInstallingSkills || !repoPath.trim()}
                  onClick={handleInstallSkills}
                  className="flex items-center gap-1.5 px-3.5 py-2 rounded-xl text-xs font-bold bg-[var(--accent-color)] hover:opacity-90 text-white shadow-xs transition-all hover:scale-102 active:scale-95 disabled:opacity-50 cursor-pointer shrink-0"
                >
                  {isInstallingSkills ? (
                    <Loader2 size={13} className="animate-spin text-white" />
                  ) : (
                    <Download size={13} />
                  )}
                  <span>
                    {isInstallingSkills
                      ? 'Scaffolding en cours...'
                      : skillsStatus?.installedAll
                      ? 'Réinstaller / Sync Worktrees'
                      : '⚡ Scaffolder dans le projet & worktrees'}
                  </span>
                </button>
              </div>

              {/* Worktrees Coverage Notice */}
              {skillsStatus && skillsStatus.worktreePaths && skillsStatus.worktreePaths.length > 0 && (
                <div className="p-3 rounded-xl bg-[var(--bg-primary)] border border-[var(--border-color)] text-[11px] text-[var(--text-secondary)]">
                  <div className="flex items-center gap-1.5 font-semibold text-[var(--text-primary)] mb-1">
                    <FolderGit2 size={13} className="text-[var(--accent-color)]" />
                    <span>Dépôts et Worktrees synchronisés ({skillsStatus.worktreePaths.length}) :</span>
                  </div>
                  <div className="flex flex-wrap gap-1.5">
                    {skillsStatus.worktreePaths.map(wp => (
                      <span key={wp} className="font-mono text-[10px] px-2 py-0.5 rounded-md bg-[var(--bg-tertiary)] text-[var(--text-muted)] border border-[var(--border-color)]">
                        {wp.split('/').slice(-2).join('/')}
                      </span>
                    ))}
                  </div>
                  <span className="text-[9px] text-[var(--text-muted)] mt-1.5 block">
                    Les fichiers de skills sont écrits dans <code className="text-cyan-400">.agents/skills/</code>, <code className="text-cyan-400">.gemini/skills/</code>, <code className="text-cyan-400">.agy/skills/</code> et <code className="text-cyan-400">.skills/</code> de chaque worktree.
                  </span>
                </div>
              )}

              {/* Skills Overrides Table List */}
              <div className="p-3.5 rounded-xl bg-[var(--bg-tertiary)]/70 border border-[var(--border-color)] space-y-2.5">
                <div className="flex items-center justify-between pb-1 border-b border-[var(--border-color)]">
                  <span className="text-[10px] font-bold uppercase tracking-wider text-[var(--text-primary)]">
                    Compétences du Workflow & Surcharge des Noms
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
                    const displayDefaultName = s.id === 'specify'
                      ? (specFramework === 'openfeature' ? 'Specify (Open Feature)' : 'Specify (SpecKit)')
                      : s.defaultName

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
                                {displayDefaultName}
                              </span>
                              <span className="text-[9px] font-mono text-[var(--text-muted)] bg-[var(--bg-tertiary)] px-1 py-0.2 rounded-md">
                                /{s.code}
                              </span>
                            </div>
                            <span className="text-[10px] text-[var(--text-muted)] truncate max-w-[220px]">
                              {s.id === 'specify'
                                ? (specFramework === 'openfeature' ? 'Spécification technique Open Feature (Feature Flags)' : 'Spécification technique standard SpecKit (BDD Given/When/Then)')
                                : s.desc}
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
                              placeholder={`Surcharge : ${displayDefaultName}`}
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
