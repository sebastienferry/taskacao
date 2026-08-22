import React, { useState, useEffect } from 'react'
import {
  X,
  User,
  Mail,
  Palette,
  Sun,
  Moon,
  Globe,
  Check,
  RotateCcw,
  Bot,
  Terminal,
  FolderGit2,
  CheckCircle2,
  XCircle,
  RefreshCw,
  Sliders,
  GitPullRequest,
  PanelRight,
  Square
} from 'lucide-react'
import { useApp } from '../context/AppContext'
import type { AccentColor, Theme, Language, Density, ViewMode, AIProvider, IssueTracker, DetailMode } from '../types'

export const ProfileModal: React.FC = () => {
  const {
    isProfileOpen,
    setIsProfileOpen,
    settings,
    updateSettings,
    reseedDemo,
    cliStatuses,
    fetchCliStatus,
    syncLinear,
    syncGithub,
    isSyncing,
    t,
  } = useApp()

  const [activeTab, setActiveTab] = useState<'appearance' | 'ai' | 'tracker'>('ai')

  // Appearance & User
  const [userName, setUserName] = useState(settings.userName)
  const [userEmail, setUserEmail] = useState(settings.userEmail)
  const [theme, setTheme] = useState<Theme>(settings.theme)
  const [accentColor, setAccentColor] = useState<AccentColor>(settings.accentColor)
  const [language, setLanguage] = useState<Language>(settings.language)
  const [density, setDensity] = useState<Density>(settings.density)
  const [defaultView, setDefaultView] = useState<ViewMode>(settings.defaultView)
  const [detailMode, setDetailMode] = useState<DetailMode>(settings.detailMode || 'panel')

  // AI & Prompts
  const [aiProvider, setAiProvider] = useState<AIProvider>(settings.aiProvider || 'agy')
  const [aiCommandTemplate, setAiCommandTemplate] = useState(settings.aiCommandTemplate || 'agy -p "{prompt}"')
  const [repoPath, setRepoPath] = useState(settings.repoPath || '/Users/sferry/Sources/fretzee-studio')
  const [promptClarify, setPromptClarify] = useState(settings.promptClarify || '')
  const [promptSpecify, setPromptSpecify] = useState(settings.promptSpecify || '')
  const [promptImplement, setPromptImplement] = useState(settings.promptImplement || '')
  const [promptCreatePr, setPromptCreatePr] = useState(settings.promptCreatePr || '')

  // Tracker
  const [issueTracker, setIssueTracker] = useState<IssueTracker>(settings.issueTracker || 'linear')
  const [linearTeam, setLinearTeam] = useState(settings.linearTeam || 'FRE')
  const [githubRepo, setGithubRepo] = useState(settings.githubRepo || 'fretzee/studio')

  useEffect(() => {
    if (isProfileOpen) {
      setUserName(settings.userName)
      setUserEmail(settings.userEmail)
      setTheme(settings.theme)
      setAccentColor(settings.accentColor)
      setLanguage(settings.language)
      setDensity(settings.density)
      setDefaultView(settings.defaultView)
      setDetailMode(settings.detailMode || 'panel')
      setAiProvider(settings.aiProvider || 'agy')
      setAiCommandTemplate(settings.aiCommandTemplate || 'agy -p "{prompt}"')
      setRepoPath(settings.repoPath || '/Users/sferry/Sources/fretzee-studio')
      setPromptClarify(settings.promptClarify || '')
      setPromptSpecify(settings.promptSpecify || '')
      setPromptImplement(settings.promptImplement || '')
      setPromptCreatePr(settings.promptCreatePr || '')
      setIssueTracker(settings.issueTracker || 'linear')
      setLinearTeam(settings.linearTeam || 'FRE')
      setGithubRepo(settings.githubRepo || 'sebastienferry/fretzee-studio')
      fetchCliStatus()
    }
  }, [isProfileOpen, settings, fetchCliStatus])

  if (!isProfileOpen) return null

  const accents: { id: AccentColor; name: string; hex: string }[] = [
    { id: 'indigo', name: t.profileModal.accents.indigo, hex: '#6366f1' },
    { id: 'violet', name: t.profileModal.accents.violet, hex: '#8b5cf6' },
    { id: 'emerald', name: t.profileModal.accents.emerald, hex: '#10b981' },
    { id: 'amber', name: t.profileModal.accents.amber, hex: '#f59e0b' },
    { id: 'rose', name: t.profileModal.accents.rose, hex: '#f43f5e' },
    { id: 'cyan', name: t.profileModal.accents.cyan, hex: '#06b6d4' },
    { id: 'blue', name: t.profileModal.accents.blue, hex: '#3b82f6' },
    { id: 'orange', name: t.profileModal.accents.orange, hex: '#f97316' },
  ]

  const densities: { id: Density; label: string; desc: string }[] = [
    { id: 'compact', label: 'Compact', desc: '13px font, padding réduit' },
    { id: 'standard', label: 'Standard', desc: '14px font, équilibre optimal' },
    { id: 'comfortable', label: 'Confortable', desc: '15px font, grands espacements' },
  ]

  const handleProviderChange = (p: AIProvider) => {
    setAiProvider(p)
    if (p === 'agy') setAiCommandTemplate('agy -p "{prompt}" --dangerously-skip-permissions')
    else if (p === 'vibe') setAiCommandTemplate('vibe -p "{prompt}" --auto-approve')
    else if (p === 'claude') setAiCommandTemplate('claude -p "{prompt}"')
  }

  const handleSave = async () => {
    await updateSettings({
      userName: userName.trim(),
      userEmail: userEmail.trim(),
      theme,
      accentColor,
      language,
      density,
      defaultView,
      detailMode,
      aiProvider,
      aiCommandTemplate,
      repoPath: repoPath.trim(),
      issueTracker,
      linearTeam: linearTeam.trim(),
      githubRepo: githubRepo.trim(),
      promptClarify,
      promptSpecify,
      promptImplement,
      promptCreatePr,
    })
    setIsProfileOpen(false)
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-xs animate-in fade-in duration-200">
      <div className="relative w-full max-w-2xl rounded-2xl bg-[var(--bg-secondary)] border border-[var(--border-color)] shadow-2xl overflow-hidden flex flex-col max-h-[92vh]">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-[var(--border-color)] bg-[var(--bg-tertiary)]/30 shrink-0">
          <div className="flex items-center gap-2.5">
            <div className="w-8 h-8 rounded-xl accent-bg text-white flex items-center justify-center shadow">
              <Sliders size={16} />
            </div>
            <div>
              <h3 className="text-sm font-bold text-[var(--text-primary)]">
                {t.profileModal.title}
              </h3>
              <p className="text-[11px] text-[var(--text-muted)]">
                {t.profileModal.subtitle}
              </p>
            </div>
          </div>
          <button
            onClick={() => setIsProfileOpen(false)}
            className="p-1.5 rounded-lg text-[var(--text-muted)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-tertiary)] transition-colors"
          >
            <X size={17} />
          </button>
        </div>

        {/* Tab Navigation */}
        <div className="flex items-center gap-4 px-6 pt-3 border-b border-[var(--border-color)] bg-[var(--bg-secondary)] shrink-0 text-xs font-semibold">
          <button
            onClick={() => setActiveTab('ai')}
            className={`pb-2.5 flex items-center gap-1.5 border-b-2 transition-all ${
              activeTab === 'ai'
                ? 'border-[var(--accent-color)] accent-text font-bold'
                : 'border-transparent text-[var(--text-muted)] hover:text-[var(--text-primary)]'
            }`}
          >
            <Bot size={14} className="text-amber-400" />
            <span>{t.profileModal.tabs.aiConfig}</span>
          </button>

          <button
            onClick={() => setActiveTab('tracker')}
            className={`pb-2.5 flex items-center gap-1.5 border-b-2 transition-all ${
              activeTab === 'tracker'
                ? 'border-[var(--accent-color)] accent-text font-bold'
                : 'border-transparent text-[var(--text-muted)] hover:text-[var(--text-primary)]'
            }`}
          >
            <FolderGit2 size={14} className="text-indigo-400" />
            <span>{t.profileModal.tabs.tracker}</span>
          </button>

          <button
            onClick={() => setActiveTab('appearance')}
            className={`pb-2.5 flex items-center gap-1.5 border-b-2 transition-all ${
              activeTab === 'appearance'
                ? 'border-[var(--accent-color)] accent-text font-bold'
                : 'border-transparent text-[var(--text-muted)] hover:text-[var(--text-primary)]'
            }`}
          >
            <Palette size={14} className="text-cyan-400" />
            <span>{t.profileModal.tabs.appearance}</span>
          </button>
        </div>

        {/* Modal Form Body */}
        <div className="p-6 overflow-y-auto space-y-6 flex-1 text-xs">
          {/* TAB 1: AI ENGINE & PROMPTS */}
          {activeTab === 'ai' && (
            <div className="space-y-6 animate-in fade-in duration-150">
              {/* CLI Status Badges */}
              <div className="p-3.5 rounded-xl bg-[var(--bg-tertiary)]/50 border border-[var(--border-color)] space-y-2">
                <div className="flex items-center justify-between">
                  <span className="text-[11px] font-bold uppercase tracking-wider text-[var(--text-muted)] flex items-center gap-1">
                    <Terminal size={13} />
                    {t.profileModal.ai.cliStatusTitle}
                  </span>
                  <button
                    onClick={() => fetchCliStatus()}
                    className="text-[10px] text-[var(--accent-color)] hover:underline flex items-center gap-1"
                  >
                    <RefreshCw size={10} /> Actualiser
                  </button>
                </div>
                <div className="grid grid-cols-2 sm:grid-cols-3 gap-2">
                  {cliStatuses.map(cli => (
                    <div
                      key={cli.tool}
                      className="p-2 rounded-lg bg-[var(--bg-secondary)] border border-[var(--border-color)] flex items-center justify-between"
                    >
                      <div className="truncate">
                        <span className="font-bold text-xs font-mono">{cli.tool}</span>
                        <div className="text-[10px] text-[var(--text-muted)] truncate">{cli.details}</div>
                      </div>
                      {cli.available ? (
                        <CheckCircle2 size={15} className="text-emerald-400 shrink-0" />
                      ) : (
                        <XCircle size={15} className="text-rose-400 shrink-0" />
                      )}
                    </div>
                  ))}
                </div>
              </div>

              {/* AI Engine Selection */}
              <div className="space-y-2">
                <label className="block text-[11px] font-bold uppercase tracking-wider text-[var(--text-muted)]">
                  {t.profileModal.ai.engine}
                </label>
                <div className="grid grid-cols-2 sm:grid-cols-4 gap-2">
                  {(['agy', 'vibe', 'claude', 'custom'] as AIProvider[]).map(p => {
                    const isSelected = aiProvider === p
                    return (
                      <button
                        key={p}
                        type="button"
                        onClick={() => handleProviderChange(p)}
                        className={`p-3 rounded-xl border text-center font-bold uppercase transition-all ${
                          isSelected
                            ? 'bg-[var(--accent-light)] border-[var(--accent-color)] accent-text ring-2 ring-[var(--accent-glow)]'
                            : 'bg-[var(--bg-tertiary)] border-[var(--border-color)] text-[var(--text-secondary)] hover:border-[var(--text-primary)]'
                        }`}
                      >
                        <div className="text-sm">{p}</div>
                        <div className="text-[10px] font-normal opacity-70 mt-0.5 lowercase">
                          {p === 'agy' ? 'Antigravity' : p === 'vibe' ? 'Mistral Vibe' : p === 'claude' ? 'Claude Code' : 'Custom'}
                        </div>
                      </button>
                    )
                  })}
                </div>
              </div>

              {/* Repo CWD & Command Template */}
              <div className="grid grid-cols-1 gap-3">
                <div>
                  <label className="block text-[11px] font-medium text-[var(--text-secondary)] mb-1">
                    {t.profileModal.ai.repoPath}
                  </label>
                  <input
                    type="text"
                    value={repoPath}
                    onChange={e => setRepoPath(e.target.value)}
                    placeholder="/Users/sferry/Sources/fretzee-studio"
                    className="w-full font-mono text-xs px-3 py-2 rounded-xl bg-[var(--bg-tertiary)] border border-[var(--border-color)] text-[var(--text-primary)] focus:outline-none focus:border-[var(--accent-color)]"
                  />
                  <span className="text-[10px] text-[var(--text-muted)] mt-1 block">
                    {t.profileModal.ai.repoPathDesc}
                  </span>
                </div>

                <div>
                  <label className="block text-[11px] font-medium text-[var(--text-secondary)] mb-1">
                    {t.profileModal.ai.cmdTemplate}
                  </label>
                  <input
                    type="text"
                    value={aiCommandTemplate}
                    onChange={e => setAiCommandTemplate(e.target.value)}
                    placeholder='agy -p "{prompt}"'
                    className="w-full font-mono text-xs px-3 py-2 rounded-xl bg-[var(--bg-tertiary)] border border-[var(--border-color)] text-[var(--text-primary)] focus:outline-none focus:border-[var(--accent-color)]"
                  />
                  <span className="text-[10px] text-[var(--text-muted)] mt-1 block">
                    Variables disponibles : <code className="text-amber-400">&#123;prompt&#125;</code>, <code className="text-amber-400">&#123;issueKey&#125;</code>, <code className="text-amber-400">&#123;issueTitle&#125;</code>, <code className="text-amber-400">&#123;branchName&#125;</code>
                  </span>
                </div>
              </div>

              {/* Prompt Templates */}
              <div className="space-y-3 pt-2 border-t border-[var(--border-color)]">
                <div className="text-[11px] font-bold uppercase tracking-wider text-[var(--text-muted)]">
                  {t.profileModal.ai.promptsTitle}
                </div>

                <div className="space-y-3">
                  <div>
                    <label className="block text-[11px] font-medium text-[var(--text-secondary)] mb-1">
                      {t.profileModal.ai.promptClarify}
                    </label>
                    <textarea
                      rows={2}
                      value={promptClarify}
                      onChange={e => setPromptClarify(e.target.value)}
                      placeholder="Tu es l'agent de clarification pour {issueKey} ({issueTitle}). Formule 3 questions d'alignement..."
                      className="w-full text-xs px-3 py-2 rounded-xl bg-[var(--bg-tertiary)] border border-[var(--border-color)] text-[var(--text-primary)] focus:outline-none focus:border-[var(--accent-color)]"
                    />
                  </div>

                  <div>
                    <label className="block text-[11px] font-medium text-[var(--text-secondary)] mb-1">
                      {t.profileModal.ai.promptSpecify}
                    </label>
                    <textarea
                      rows={2}
                      value={promptSpecify}
                      onChange={e => setPromptSpecify(e.target.value)}
                      placeholder="Tu es l'architecte Speckit pour {issueKey}. Rédige la spec technique, API et critères d'acceptation..."
                      className="w-full text-xs px-3 py-2 rounded-xl bg-[var(--bg-tertiary)] border border-[var(--border-color)] text-[var(--text-primary)] focus:outline-none focus:border-[var(--accent-color)]"
                    />
                  </div>

                  <div>
                    <label className="block text-[11px] font-medium text-[var(--text-secondary)] mb-1">
                      {t.profileModal.ai.promptImplement}
                    </label>
                    <textarea
                      rows={2}
                      value={promptImplement}
                      onChange={e => setPromptImplement(e.target.value)}
                      placeholder="Tu es le développeur. Prépare le plan de code et les tests pour {issueKey} sur la branche {branchName}..."
                      className="w-full text-xs px-3 py-2 rounded-xl bg-[var(--bg-tertiary)] border border-[var(--border-color)] text-[var(--text-primary)] focus:outline-none focus:border-[var(--accent-color)]"
                    />
                  </div>

                  <div>
                    <label className="block text-[11px] font-medium text-[var(--text-secondary)] mb-1">
                      {t.profileModal.ai.promptCreatePr}
                    </label>
                    <textarea
                      rows={2}
                      value={promptCreatePr}
                      onChange={e => setPromptCreatePr(e.target.value)}
                      placeholder="Génère le commit conventionnel, le titre et la description markdown de la PR pour {issueKey}..."
                      className="w-full text-xs px-3 py-2 rounded-xl bg-[var(--bg-tertiary)] border border-[var(--border-color)] text-[var(--text-primary)] focus:outline-none focus:border-[var(--accent-color)]"
                    />
                  </div>
                </div>
              </div>
            </div>
          )}

          {/* TAB 2: LINEAR & GITHUB CLI */}
          {activeTab === 'tracker' && (
            <div className="space-y-6 animate-in fade-in duration-150">
              <div className="space-y-2">
                <label className="block text-[11px] font-bold uppercase tracking-wider text-[var(--text-muted)]">
                  {t.profileModal.tracker.selectTracker}
                </label>
                <div className="grid grid-cols-3 gap-2">
                  {(['linear', 'github', 'local'] as IssueTracker[]).map(trk => (
                    <button
                      key={trk}
                      type="button"
                      onClick={() => setIssueTracker(trk)}
                      className={`p-3 rounded-xl border text-center font-bold uppercase transition-all ${
                        issueTracker === trk
                          ? 'bg-[var(--accent-light)] border-[var(--accent-color)] accent-text ring-2 ring-[var(--accent-glow)]'
                          : 'bg-[var(--bg-tertiary)] border-[var(--border-color)] text-[var(--text-secondary)] hover:border-[var(--text-primary)]'
                      }`}
                    >
                      <div className="text-sm">{trk}</div>
                      <div className="text-[10px] font-normal opacity-70 mt-0.5 lowercase">
                        {trk === 'linear' ? 'Linear CLI' : trk === 'github' ? 'GitHub CLI (gh)' : 'SQLite Local'}
                      </div>
                    </button>
                  ))}
                </div>
              </div>

              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                <div>
                  <label className="block text-[11px] font-medium text-[var(--text-secondary)] mb-1">
                    {t.profileModal.tracker.linearTeam}
                  </label>
                  <input
                    type="text"
                    value={linearTeam}
                    onChange={e => setLinearTeam(e.target.value)}
                    placeholder="FRE"
                    className="w-full font-mono text-xs px-3 py-2 rounded-xl bg-[var(--bg-tertiary)] border border-[var(--border-color)] text-[var(--text-primary)] focus:outline-none focus:border-[var(--accent-color)]"
                  />
                </div>

                <div>
                  <label className="block text-[11px] font-medium text-[var(--text-secondary)] mb-1">
                    {t.profileModal.tracker.githubRepo}
                  </label>
                  <input
                    type="text"
                    value={githubRepo}
                    onChange={e => setGithubRepo(e.target.value)}
                    placeholder="fretzee/studio"
                    className="w-full font-mono text-xs px-3 py-2 rounded-xl bg-[var(--bg-tertiary)] border border-[var(--border-color)] text-[var(--text-primary)] focus:outline-none focus:border-[var(--accent-color)]"
                  />
                </div>
              </div>

              {/* Sync Action Buttons */}
              <div className="space-y-3 pt-4 border-t border-[var(--border-color)]">
                <div className="text-[11px] font-bold uppercase tracking-wider text-[var(--text-muted)]">
                  Actions de Synchronisation Shell
                </div>
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                  <button
                    type="button"
                    onClick={syncLinear}
                    disabled={isSyncing}
                    className="flex items-center justify-center gap-2 p-3 rounded-xl font-semibold text-xs text-white bg-indigo-600 hover:bg-indigo-700 active:scale-98 transition-all disabled:opacity-50 shadow"
                  >
                    <RefreshCw size={14} className={isSyncing ? 'animate-spin' : ''} />
                    <span>{t.profileModal.tracker.syncLinearBtn}</span>
                  </button>

                  <button
                    type="button"
                    onClick={syncGithub}
                    disabled={isSyncing}
                    className="flex items-center justify-center gap-2 p-3 rounded-xl font-semibold text-xs text-white bg-slate-800 hover:bg-slate-700 active:scale-98 transition-all disabled:opacity-50 shadow border border-slate-600"
                  >
                    <GitPullRequest size={14} className={isSyncing ? 'animate-spin' : ''} />
                    <span>{t.profileModal.tracker.syncGithubBtn}</span>
                  </button>
                </div>
              </div>
            </div>
          )}

          {/* TAB 3: APPEARANCE & PROFILE */}
          {activeTab === 'appearance' && (
            <div className="space-y-6 animate-in fade-in duration-150">
              {/* User info */}
              <div className="space-y-3">
                <div className="text-[11px] font-bold uppercase tracking-wider text-[var(--text-muted)]">
                  {t.profileModal.userSection}
                </div>
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                  <div>
                    <label className="block text-[11px] font-medium text-[var(--text-secondary)] mb-1">
                      {t.profileModal.name}
                    </label>
                    <div className="relative">
                      <input
                        type="text"
                        value={userName}
                        onChange={e => setUserName(e.target.value)}
                        className="w-full pl-8 pr-3 py-2 rounded-xl bg-[var(--bg-tertiary)] border border-[var(--border-color)] text-[var(--text-primary)] focus:outline-none focus:border-[var(--accent-color)]"
                      />
                      <User size={14} className="absolute left-2.5 top-2.5 text-[var(--text-muted)]" />
                    </div>
                  </div>

                  <div>
                    <label className="block text-[11px] font-medium text-[var(--text-secondary)] mb-1">
                      {t.profileModal.email}
                    </label>
                    <div className="relative">
                      <input
                        type="email"
                        value={userEmail}
                        onChange={e => setUserEmail(e.target.value)}
                        className="w-full pl-8 pr-3 py-2 rounded-xl bg-[var(--bg-tertiary)] border border-[var(--border-color)] text-[var(--text-primary)] focus:outline-none focus:border-[var(--accent-color)]"
                      />
                      <Mail size={14} className="absolute left-2.5 top-2.5 text-[var(--text-muted)]" />
                    </div>
                  </div>
                </div>
              </div>

              {/* Accent Color */}
              <div className="space-y-3">
                <div className="flex items-center justify-between">
                  <div className="text-[11px] font-bold uppercase tracking-wider text-[var(--text-muted)] flex items-center gap-1.5">
                    <Palette size={13} />
                    {t.profileModal.accentColor}
                  </div>
                  <span className="text-[11px] font-semibold accent-text capitalize">
                    {accentColor}
                  </span>
                </div>

                <div className="grid grid-cols-4 sm:grid-cols-8 gap-2">
                  {accents.map(acc => {
                    const isSelected = accentColor === acc.id
                    return (
                      <button
                        key={acc.id}
                        type="button"
                        onClick={() => setAccentColor(acc.id)}
                        className={`h-10 rounded-xl flex flex-col items-center justify-center relative transition-all duration-150 ${
                          isSelected
                            ? 'ring-2 ring-white scale-105 shadow-md'
                            : 'opacity-85 hover:opacity-100 hover:scale-102'
                        }`}
                        style={{ backgroundColor: acc.hex }}
                        title={acc.name}
                      >
                        {isSelected && <Check size={16} className="text-white drop-shadow" />}
                      </button>
                    )
                  })}
                </div>
              </div>

              {/* Theme & Language */}
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                <div className="space-y-2">
                  <label className="block text-[11px] font-bold uppercase tracking-wider text-[var(--text-muted)]">
                    {t.profileModal.theme}
                  </label>
                  <div className="grid grid-cols-2 gap-2">
                    <button
                      type="button"
                      onClick={() => setTheme('dark')}
                      className={`flex items-center justify-center gap-2 py-2 rounded-xl border font-medium transition-all ${
                        theme === 'dark'
                          ? 'bg-[var(--accent-light)] border-[var(--accent-color)] accent-text'
                          : 'bg-[var(--bg-tertiary)] border-[var(--border-color)] text-[var(--text-secondary)]'
                      }`}
                    >
                      <Moon size={14} />
                      <span>{t.profileModal.themes.dark}</span>
                    </button>

                    <button
                      type="button"
                      onClick={() => setTheme('light')}
                      className={`flex items-center justify-center gap-2 py-2 rounded-xl border font-medium transition-all ${
                        theme === 'light'
                          ? 'bg-[var(--accent-light)] border-[var(--accent-color)] accent-text'
                          : 'bg-[var(--bg-tertiary)] border-[var(--border-color)] text-[var(--text-secondary)]'
                      }`}
                    >
                      <Sun size={14} />
                      <span>{t.profileModal.themes.light}</span>
                    </button>
                  </div>
                </div>

                <div className="space-y-2">
                  <label className="block text-[11px] font-bold uppercase tracking-wider text-[var(--text-muted)]">
                    {t.profileModal.language}
                  </label>
                  <div className="grid grid-cols-2 gap-2">
                    <button
                      type="button"
                      onClick={() => setLanguage('fr')}
                      className={`flex items-center justify-center gap-1.5 py-2 rounded-xl border font-medium transition-all ${
                        language === 'fr'
                          ? 'bg-[var(--accent-light)] border-[var(--accent-color)] accent-text'
                          : 'bg-[var(--bg-tertiary)] border-[var(--border-color)] text-[var(--text-secondary)]'
                      }`}
                    >
                      <Globe size={14} />
                      <span>FR</span>
                    </button>

                    <button
                      type="button"
                      onClick={() => setLanguage('en')}
                      className={`flex items-center justify-center gap-1.5 py-2 rounded-xl border font-medium transition-all ${
                        language === 'en'
                          ? 'bg-[var(--accent-light)] border-[var(--accent-color)] accent-text'
                          : 'bg-[var(--bg-tertiary)] border-[var(--border-color)] text-[var(--text-secondary)]'
                      }`}
                    >
                      <Globe size={14} />
                      <span>EN</span>
                    </button>
                  </div>
                </div>
              </div>

              {/* Story Detail Mode (Right Panel vs Modal) */}
              <div className="space-y-2">
                <label className="block text-[11px] font-bold uppercase tracking-wider text-[var(--text-muted)]">
                  {t.profileModal.detailMode}
                </label>
                <div className="grid grid-cols-2 gap-2">
                  <button
                    type="button"
                    onClick={() => setDetailMode('panel')}
                    className={`flex items-center gap-3 p-3 rounded-xl border text-left transition-all ${
                      detailMode === 'panel'
                        ? 'bg-[var(--accent-light)] border-[var(--accent-color)] accent-text ring-2 ring-[var(--accent-glow)]'
                        : 'bg-[var(--bg-tertiary)] border-[var(--border-color)] text-[var(--text-secondary)] hover:border-[var(--text-muted)]'
                    }`}
                  >
                    <div className="w-8 h-8 rounded-lg bg-[var(--bg-secondary)] flex items-center justify-center text-indigo-400 border border-[var(--border-color)] shrink-0">
                      <PanelRight size={18} />
                    </div>
                    <div className="truncate">
                      <div className="font-bold text-xs">{t.profileModal.detailModes.panel}</div>
                      <div className="text-[10px] text-[var(--text-muted)] opacity-80">Glissement latéral à droite</div>
                    </div>
                  </button>

                  <button
                    type="button"
                    onClick={() => setDetailMode('modal')}
                    className={`flex items-center gap-3 p-3 rounded-xl border text-left transition-all ${
                      detailMode === 'modal'
                        ? 'bg-[var(--accent-light)] border-[var(--accent-color)] accent-text ring-2 ring-[var(--accent-glow)]'
                        : 'bg-[var(--bg-tertiary)] border-[var(--border-color)] text-[var(--text-secondary)] hover:border-[var(--text-muted)]'
                    }`}
                  >
                    <div className="w-8 h-8 rounded-lg bg-[var(--bg-secondary)] flex items-center justify-center text-purple-400 border border-[var(--border-color)] shrink-0">
                      <Square size={18} />
                    </div>
                    <div className="truncate">
                      <div className="font-bold text-xs">{t.profileModal.detailModes.modal}</div>
                      <div className="text-[10px] text-[var(--text-muted)] opacity-80">Boîte de dialogue au centre</div>
                    </div>
                  </button>
                </div>
              </div>

              {/* Density */}
              <div className="space-y-2">
                <label className="block text-[11px] font-bold uppercase tracking-wider text-[var(--text-muted)]">
                  {t.profileModal.density}
                </label>
                <div className="grid grid-cols-1 sm:grid-cols-3 gap-2">
                  {densities.map(d => {
                    const isSelected = density === d.id
                    return (
                      <button
                        key={d.id}
                        type="button"
                        onClick={() => setDensity(d.id)}
                        className={`p-2.5 rounded-xl border text-left transition-all ${
                          isSelected
                            ? 'bg-[var(--accent-light)] border-[var(--accent-color)] accent-text'
                            : 'bg-[var(--bg-tertiary)] border-[var(--border-color)] text-[var(--text-secondary)] hover:border-[var(--text-muted)]'
                        }`}
                      >
                        <div className="font-bold text-xs">{d.label}</div>
                        <div className="text-[10px] opacity-75 mt-0.5">{d.desc}</div>
                      </button>
                    )
                  })}
                </div>
              </div>

              {/* Demo Reset */}
              <div className="pt-2">
                <button
                  type="button"
                  onClick={() => {
                    reseedDemo()
                    setIsProfileOpen(false)
                  }}
                  className="w-full flex items-center justify-center gap-2 py-2 px-3 rounded-xl text-xs font-semibold text-rose-400 bg-rose-500/10 hover:bg-rose-500/20 border border-rose-500/30 transition-colors"
                >
                  <RotateCcw size={14} />
                  {t.profileModal.reseedBtn}
                </button>
              </div>
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="flex items-center justify-end gap-2.5 px-6 py-4 border-t border-[var(--border-color)] bg-[var(--bg-tertiary)]/40 shrink-0">
          <button
            type="button"
            onClick={() => setIsProfileOpen(false)}
            className="px-4 py-2 rounded-xl text-xs font-medium text-[var(--text-secondary)] hover:bg-[var(--bg-tertiary)] transition-colors"
          >
            {t.taskModal.cancel}
          </button>
          <button
            type="button"
            onClick={handleSave}
            className="px-5 py-2 rounded-xl text-xs font-semibold text-white accent-bg shadow hover:opacity-90 active:scale-95 transition-all"
          >
            {t.profileModal.save}
          </button>
        </div>
      </div>
    </div>
  )
}
