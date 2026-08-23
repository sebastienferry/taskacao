import React, { useState } from 'react'
import {
  RefreshCw,
  FolderGit2,
  CheckCircle2,
  Clock,
  ExternalLink,
  Save,
  Check,
  Zap,
  Globe2,
  Terminal,
  Activity as ActivityIcon,
  ChevronRight,
  Sliders,
  AlertCircle
} from 'lucide-react'
import { useApp } from '../context/AppContext'
import type { IssueTracker, TaskActivity } from '../types'

export const SyncView: React.FC = () => {
  const {
    currentProject,
    updateProject,
    setIsProjectModalOpen,
    setEditingProject,
    tasks,
    activities,
    setSelectedActivity,
    setActiveView,
    settings,
    updateSettings,
    syncLinear,
    syncGithub,
    syncAll,
    isSyncing,
    activeJobCount,
    t,
  } = useApp()

  // Local form state initialized from active project or fallback to global settings
  const [linearTeam, setLinearTeam] = useState(currentProject?.linearTeam || settings.linearTeam || '')
  const [githubRepo, setGithubRepo] = useState(currentProject?.githubRepo || settings.githubRepo || '')
  const [repoPath, setRepoPath] = useState(currentProject?.repoPath || settings.repoPath || '')
  const [issueTracker, setIssueTracker] = useState<IssueTracker>(currentProject?.issueTracker || settings.issueTracker || 'linear')
  const [isSaved, setIsSaved] = useState(false)

  // Custom parameters for manual triggers
  const [customLinearTeam, setCustomLinearTeam] = useState(currentProject?.linearTeam || settings.linearTeam || '')
  const [customGithubRepo, setCustomGithubRepo] = useState(currentProject?.githubRepo || settings.githubRepo || '')

  // Keep form updated when currentProject changes
  React.useEffect(() => {
    if (currentProject) {
      setLinearTeam(currentProject.linearTeam || '')
      setGithubRepo(currentProject.githubRepo || '')
      setRepoPath(currentProject.repoPath || '')
      setIssueTracker(currentProject.issueTracker || 'linear')
      setCustomLinearTeam(currentProject.linearTeam || '')
      setCustomGithubRepo(currentProject.githubRepo || '')
    }
  }, [currentProject])

  const handleSaveOptions = async (e: React.FormEvent) => {
    e.preventDefault()
    if (currentProject) {
      await updateProject(currentProject.id, {
        linearTeam: linearTeam.trim().toUpperCase(),
        githubRepo: githubRepo.trim(),
        repoPath: repoPath.trim(),
        issueTracker,
      })
    }
    await updateSettings({
      linearTeam: linearTeam.trim().toUpperCase(),
      githubRepo: githubRepo.trim(),
      repoPath: repoPath.trim(),
      issueTracker,
    })
    setIsSaved(true)
    setTimeout(() => setIsSaved(false), 3000)
  }

  // Filter activities that are sync related
  const syncActivities = activities.filter(
    a => a.skillId === 'sync_linear' || a.skillId === 'sync_github' || a.skillId === 'sync_all' || a.skillId.startsWith('sync')
  )

  const linearCount = tasks.filter(t => t.source === 'linear').length
  const githubCount = tasks.filter(t => t.source === 'github').length
  const localCount = tasks.filter(t => !t.source || t.source === 'local').length

  const getStatusBadge = (status: string) => {
    switch (status) {
      case 'running':
        return (
          <span className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-semibold bg-amber-500/15 text-amber-400 border border-amber-500/30">
            <RefreshCw size={11} className="animate-spin" /> En cours
          </span>
        )
      case 'queued':
      case 'pending':
        return (
          <span className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-semibold bg-blue-500/15 text-blue-400 border border-blue-500/30">
            <Clock size={11} /> En file d'attente
          </span>
        )
      case 'completed':
        return (
          <span className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-semibold bg-emerald-500/15 text-emerald-400 border border-emerald-500/30">
            <CheckCircle2 size={11} /> Terminé
          </span>
        )
      case 'failed':
        return (
          <span className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-semibold bg-rose-500/15 text-rose-400 border border-rose-500/30">
            <AlertCircle size={11} /> Échoué
          </span>
        )
      default:
        return (
          <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-[var(--bg-tertiary)] text-[var(--text-muted)]">
            {status}
          </span>
        )
    }
  }

  const handleInspectActivity = (act: TaskActivity) => {
    setSelectedActivity(act)
    setActiveView('activities')
  }

  return (
    <div className="flex-1 flex flex-col h-full bg-[var(--bg-primary)] overflow-y-auto">
      {/* Header */}
      <div className="border-b border-[var(--border-color)] bg-[var(--bg-secondary)] px-6 py-5 shrink-0">
        <div className="max-w-6xl mx-auto flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
          <div>
            <div className="flex items-center gap-2.5">
              <div className="p-2 rounded-lg bg-indigo-500/15 text-indigo-400 border border-indigo-500/30">
                <RefreshCw size={20} className={isSyncing ? 'animate-spin' : ''} />
              </div>
              <div>
                <h1 className="text-xl font-bold text-[var(--text-primary)] flex items-center gap-2">
                  {t.syncView.title}
                  <span className="text-xs px-2 py-0.5 rounded-full font-mono font-medium bg-[var(--accent-light)] accent-text border border-[var(--accent-color)]/30">
                    File asynchrone
                  </span>
                </h1>
                <p className="text-xs text-[var(--text-muted)] mt-0.5">
                  {t.syncView.subtitle}
                </p>
              </div>
            </div>
          </div>

          <div className="flex items-center gap-3">
            {activeJobCount > 0 && (
              <button
                onClick={() => setActiveView('activities')}
                className="flex items-center gap-2 px-3 py-1.5 rounded-lg text-xs font-semibold bg-blue-500/15 text-blue-400 border border-blue-500/30 hover:bg-blue-500/25 transition-all animate-pulse"
              >
                <ActivityIcon size={14} className="animate-spin" />
                <span>{activeJobCount} job(s) actif(s) dans Activités</span>
                <ChevronRight size={13} />
              </button>
            )}

            <button
              onClick={syncAll}
              disabled={isSyncing}
              className="flex items-center gap-2 px-4 py-2 rounded-lg text-xs font-bold text-white shadow-md accent-bg hover:opacity-90 transition-all disabled:opacity-50"
              title="Tout synchroniser (Linear + GitHub)"
            >
              <Zap size={14} className={isSyncing ? 'animate-spin' : ''} />
              <span>{t.syncView.globalCard.btnSync}</span>
            </button>
          </div>
        </div>
      </div>

      {/* Main Content */}
      <div className="flex-1 max-w-6xl w-full mx-auto p-6 space-y-6">
        {/* Active Project Segregation Scope Banner */}
        {currentProject && (
          <div className="rounded-xl border border-[var(--sidebar-border)] bg-[var(--accent-light)]/20 p-4 flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3 shadow-xs">
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 rounded-xl bg-[var(--accent-light)] border border-[var(--accent-color)]/30 flex items-center justify-center accent-text font-bold text-base shadow-[0_0_12px_var(--accent-glow)] shrink-0">
                ⚡
              </div>
              <div>
                <div className="flex items-center gap-2">
                  <span className="text-xs font-bold uppercase tracking-wider text-[var(--accent-color)]">Espace Projet Cible</span>
                  <span className="text-[10px] px-2 py-0.2 rounded-full font-mono bg-[var(--bg-tertiary)] text-[var(--text-secondary)]">
                    {tasks.length} tâches
                  </span>
                </div>
                <h3 className="text-sm font-bold text-[var(--text-primary)]">
                  {currentProject.name}
                </h3>
                <p className="text-xs text-[var(--text-muted)] font-mono truncate max-w-lg">
                  {currentProject.repoPath || 'Aucun chemin local configuré'}
                  {currentProject.linearTeam ? ` · Team Linear: ${currentProject.linearTeam}` : ''}
                  {currentProject.githubRepo ? ` · GitHub: ${currentProject.githubRepo}` : ''}
                </p>
              </div>
            </div>

            <button
              type="button"
              onClick={() => {
                setEditingProject(currentProject)
                setIsProjectModalOpen(true)
              }}
              className="px-3 py-1.5 rounded-lg text-xs font-semibold bg-[var(--bg-secondary)] hover:bg-[var(--bg-tertiary)] text-[var(--text-primary)] border border-[var(--border-color)] transition-all cursor-pointer shrink-0"
            >
              ⚙️ Configurer ce projet
            </button>
          </div>
        )}

        {/* Tracker Action Cards Grid */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-5">
          {/* Card 1: Linear */}
          <div className="rounded-xl border border-[var(--border-color)] bg-[var(--bg-secondary)] p-5 flex flex-col justify-between shadow-xs hover:border-indigo-500/40 transition-all">
            <div className="space-y-3">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <div className="w-8 h-8 rounded-lg bg-indigo-500/15 border border-indigo-500/30 flex items-center justify-center text-indigo-400 font-mono font-bold text-sm">
                    ◆
                  </div>
                  <div>
                    <h2 className="text-sm font-bold text-[var(--text-primary)]">{t.syncView.linearCard.title}</h2>
                    <span className="text-[11px] text-emerald-400 flex items-center gap-1 font-medium">
                      <span className="w-1.5 h-1.5 rounded-full bg-emerald-400"></span>
                      {t.syncView.linearCard.statusConnected}
                    </span>
                  </div>
                </div>
                <span className="text-xs px-2 py-0.5 rounded-md font-mono bg-[var(--bg-tertiary)] text-[var(--text-secondary)]">
                  {linearCount} issues
                </span>
              </div>

              <p className="text-xs text-[var(--text-secondary)] leading-relaxed">
                {t.syncView.linearCard.desc}
              </p>

              <div className="pt-2">
                <label className="block text-[11px] font-semibold text-[var(--text-muted)] uppercase tracking-wider mb-1">
                  {t.syncView.linearCard.teamLabel}
                </label>
                <div className="flex items-center gap-2">
                  <input
                    type="text"
                    value={customLinearTeam}
                    onChange={e => setCustomLinearTeam(e.target.value.toUpperCase())}
                    placeholder="FRE"
                    className="w-full px-3 py-1.5 text-xs font-mono rounded-lg bg-[var(--bg-tertiary)] border border-[var(--border-color)] text-[var(--text-primary)] focus:outline-none focus:border-indigo-500 transition-all uppercase"
                  />
                </div>
              </div>
            </div>

            <div className="pt-4 mt-2 border-t border-[var(--border-color)]">
              <button
                onClick={() => syncLinear(customLinearTeam)}
                disabled={isSyncing}
                className="w-full flex items-center justify-center gap-2 py-2 px-3 rounded-lg text-xs font-bold bg-indigo-600 hover:bg-indigo-700 text-white transition-all disabled:opacity-50 shadow-xs cursor-pointer"
              >
                <RefreshCw size={13} className={isSyncing ? 'animate-spin' : ''} />
                <span>{t.syncView.linearCard.btnSync}</span>
              </button>
            </div>
          </div>

          {/* Card 2: GitHub */}
          <div className="rounded-xl border border-[var(--border-color)] bg-[var(--bg-secondary)] p-5 flex flex-col justify-between shadow-xs hover:border-purple-500/40 transition-all">
            <div className="space-y-3">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <div className="w-8 h-8 rounded-lg bg-purple-500/15 border border-purple-500/30 flex items-center justify-center text-purple-400">
                    <FolderGit2 size={16} />
                  </div>
                  <div>
                    <h2 className="text-sm font-bold text-[var(--text-primary)]">{t.syncView.githubCard.title}</h2>
                    <span className="text-[11px] text-emerald-400 flex items-center gap-1 font-medium">
                      <span className="w-1.5 h-1.5 rounded-full bg-emerald-400"></span>
                      {t.syncView.githubCard.statusConnected}
                    </span>
                  </div>
                </div>
                <span className="text-xs px-2 py-0.5 rounded-md font-mono bg-[var(--bg-tertiary)] text-[var(--text-secondary)]">
                  {githubCount} issues
                </span>
              </div>

              <p className="text-xs text-[var(--text-secondary)] leading-relaxed">
                {t.syncView.githubCard.desc}
              </p>

              <div className="pt-2">
                <label className="block text-[11px] font-semibold text-[var(--text-muted)] uppercase tracking-wider mb-1">
                  {t.syncView.githubCard.repoLabel}
                </label>
                <input
                  type="text"
                  value={customGithubRepo}
                  onChange={e => setCustomGithubRepo(e.target.value)}
                  placeholder="owner/repo"
                  className="w-full px-3 py-1.5 text-xs font-mono rounded-lg bg-[var(--bg-tertiary)] border border-[var(--border-color)] text-[var(--text-primary)] focus:outline-none focus:border-purple-500 transition-all"
                />
              </div>
            </div>

            <div className="pt-4 mt-2 border-t border-[var(--border-color)]">
              <button
                onClick={() => syncGithub(customGithubRepo)}
                disabled={isSyncing}
                className="w-full flex items-center justify-center gap-2 py-2 px-3 rounded-lg text-xs font-bold bg-purple-600 hover:bg-purple-700 text-white transition-all disabled:opacity-50 shadow-xs cursor-pointer"
              >
                <RefreshCw size={13} className={isSyncing ? 'animate-spin' : ''} />
                <span>{t.syncView.githubCard.btnSync}</span>
              </button>
            </div>
          </div>

          {/* Card 3: Global / Statistics */}
          <div className="rounded-xl border border-[var(--border-color)] bg-[var(--bg-secondary)] p-5 flex flex-col justify-between shadow-xs hover:border-[var(--accent-color)]/40 transition-all">
            <div className="space-y-3">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <div className="w-8 h-8 rounded-lg bg-[var(--accent-light)] border border-[var(--accent-color)]/30 flex items-center justify-center accent-text">
                    <Globe2 size={16} />
                  </div>
                  <div>
                    <h2 className="text-sm font-bold text-[var(--text-primary)]">{t.syncView.globalCard.title}</h2>
                    <span className="text-[11px] text-[var(--text-muted)]">Toutes sources</span>
                  </div>
                </div>
                <span className="text-xs px-2 py-0.5 rounded-md font-mono bg-[var(--accent-light)] accent-text font-bold">
                  {tasks.length} total
                </span>
              </div>

              <p className="text-xs text-[var(--text-secondary)] leading-relaxed">
                {t.syncView.globalCard.desc}
              </p>

              {/* Source breakdown bars */}
              <div className="space-y-1.5 pt-2">
                <div className="flex justify-between text-[11px]">
                  <span className="text-indigo-400 font-medium">Linear: {linearCount}</span>
                  <span className="text-purple-400 font-medium">GitHub: {githubCount}</span>
                  <span className="text-emerald-400 font-medium">Local SQLite: {localCount}</span>
                </div>
                <div className="h-2 w-full rounded-full bg-[var(--bg-tertiary)] flex overflow-hidden">
                  <div style={{ width: `${tasks.length ? (linearCount / tasks.length) * 100 : 0}%` }} className="bg-indigo-500" />
                  <div style={{ width: `${tasks.length ? (githubCount / tasks.length) * 100 : 0}%` }} className="bg-purple-500" />
                  <div style={{ width: `${tasks.length ? (localCount / tasks.length) * 100 : 100}%` }} className="bg-emerald-500" />
                </div>
              </div>
            </div>

            <div className="pt-4 mt-2 border-t border-[var(--border-color)]">
              <button
                onClick={syncAll}
                disabled={isSyncing}
                className="w-full flex items-center justify-center gap-2 py-2 px-3 rounded-lg text-xs font-bold text-white shadow-md accent-bg hover:opacity-90 transition-all disabled:opacity-50 cursor-pointer"
              >
                <Zap size={13} className={isSyncing ? 'animate-spin' : ''} />
                <span>{t.syncView.globalCard.btnSync}</span>
              </button>
            </div>
          </div>
        </div>

        {/* Sync Options Form Section */}
        <div className="rounded-xl border border-[var(--border-color)] bg-[var(--bg-secondary)] p-6 shadow-xs">
          <div className="flex items-center justify-between pb-4 border-b border-[var(--border-color)] mb-5">
            <div className="flex items-center gap-2.5">
              <div className="p-2 rounded-lg bg-[var(--bg-tertiary)] text-[var(--text-secondary)]">
                <Sliders size={18} />
              </div>
              <div>
                <h3 className="text-sm font-bold text-[var(--text-primary)]">
                  {t.syncView.options.title}
                </h3>
                <p className="text-xs text-[var(--text-muted)]">
                  {t.syncView.options.desc}
                </p>
              </div>
            </div>

            {isSaved && (
              <span className="inline-flex items-center gap-1 px-3 py-1 rounded-full text-xs font-semibold bg-emerald-500/15 text-emerald-400 border border-emerald-500/30 animate-fade-in">
                <Check size={13} /> {t.syncView.options.saved}
              </span>
            )}
          </div>

          <form onSubmit={handleSaveOptions} className="space-y-4">
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              {/* Default Issue Tracker */}
              <div>
                <label className="block text-xs font-semibold text-[var(--text-secondary)] mb-1.5">
                  {t.syncView.options.defaultTracker}
                </label>
                <select
                  value={issueTracker}
                  onChange={e => setIssueTracker(e.target.value as IssueTracker)}
                  className="w-full px-3 py-2 text-xs rounded-lg bg-[var(--bg-tertiary)] border border-[var(--border-color)] text-[var(--text-primary)] focus:outline-none focus:border-[var(--accent-color)]"
                >
                  <option value="linear">Linear (Équipe FRE)</option>
                  <option value="github">GitHub Issues (CLI)</option>
                  <option value="local">Local uniquement (SQLite)</option>
                </select>
              </div>

              {/* Linear Team */}
              <div>
                <label className="block text-xs font-semibold text-[var(--text-secondary)] mb-1.5">
                  {t.syncView.options.linearTeam}
                </label>
                <input
                  type="text"
                  value={linearTeam}
                  onChange={e => {
                    setLinearTeam(e.target.value)
                    setCustomLinearTeam(e.target.value)
                  }}
                  placeholder="Ex: ENG"
                  className="w-full px-3 py-2 text-xs font-mono rounded-lg bg-[var(--bg-tertiary)] border border-[var(--border-color)] text-[var(--text-primary)] focus:outline-none focus:border-[var(--accent-color)] uppercase"
                />
              </div>

              {/* GitHub Repo */}
              <div>
                <label className="block text-xs font-semibold text-[var(--text-secondary)] mb-1.5">
                  {t.syncView.options.githubRepo}
                </label>
                <input
                  type="text"
                  value={githubRepo}
                  onChange={e => {
                    setGithubRepo(e.target.value)
                    setCustomGithubRepo(e.target.value)
                  }}
                  placeholder="Ex: owner/repo"
                  className="w-full px-3 py-2 text-xs font-mono rounded-lg bg-[var(--bg-tertiary)] border border-[var(--border-color)] text-[var(--text-primary)] focus:outline-none focus:border-[var(--accent-color)]"
                />
              </div>

              {/* Repo Path */}
              <div>
                <label className="block text-xs font-semibold text-[var(--text-secondary)] mb-1.5">
                  {t.syncView.options.repoPath}
                </label>
                <input
                  type="text"
                  value={repoPath}
                  onChange={e => setRepoPath(e.target.value)}
                  placeholder="Ex: /Users/username/Sources/my-project"
                  className="w-full px-3 py-2 text-xs font-mono rounded-lg bg-[var(--bg-tertiary)] border border-[var(--border-color)] text-[var(--text-primary)] focus:outline-none focus:border-[var(--accent-color)]"
                />
              </div>
            </div>

            <div className="pt-3 flex justify-end">
              <button
                type="submit"
                className="flex items-center gap-2 px-4 py-2 rounded-lg text-xs font-bold text-white shadow-xs accent-bg hover:opacity-90 transition-all cursor-pointer"
              >
                <Save size={14} />
                <span>{t.syncView.options.save}</span>
              </button>
            </div>
          </form>
        </div>

        {/* Recent Sync Activities History */}
        <div className="rounded-xl border border-[var(--border-color)] bg-[var(--bg-secondary)] p-6 shadow-xs">
          <div className="flex items-center justify-between pb-4 border-b border-[var(--border-color)] mb-4">
            <div className="flex items-center gap-2.5">
              <div className="p-2 rounded-lg bg-indigo-500/15 text-indigo-400">
                <Terminal size={18} />
              </div>
              <div>
                <h3 className="text-sm font-bold text-[var(--text-primary)]">
                  {t.syncView.history.title}
                </h3>
                <p className="text-xs text-[var(--text-muted)]">
                  {t.syncView.history.desc}
                </p>
              </div>
            </div>

            <button
              onClick={() => setActiveView('activities')}
              className="text-xs text-[var(--accent-color)] hover:underline flex items-center gap-1 font-medium cursor-pointer"
            >
              <span>{t.syncView.history.viewInActivities}</span>
              <ChevronRight size={13} />
            </button>
          </div>

          {syncActivities.length === 0 ? (
            <div className="text-center py-8 text-[var(--text-muted)]">
              <RefreshCw size={24} className="mx-auto mb-2 opacity-40" />
              <p className="text-xs">{t.syncView.history.noHistory}</p>
            </div>
          ) : (
            <div className="divide-y divide-[var(--border-color)]">
              {syncActivities.slice(0, 10).map(act => (
                <div
                  key={act.id}
                  onClick={() => handleInspectActivity(act)}
                  className="py-3.5 px-2 flex flex-col sm:flex-row sm:items-center justify-between gap-3 hover:bg-[var(--bg-tertiary)]/50 rounded-lg cursor-pointer transition-colors"
                >
                  <div className="flex items-center gap-3 min-w-0">
                    <div className="w-7 h-7 rounded-md bg-[var(--bg-tertiary)] border border-[var(--border-color)] flex items-center justify-center shrink-0">
                      {act.skillId === 'sync_linear' ? (
                        <span className="text-indigo-400 font-bold font-mono text-xs">◆</span>
                      ) : act.skillId === 'sync_github' ? (
                        <FolderGit2 size={14} className="text-purple-400" />
                      ) : (
                        <Globe2 size={14} className="text-cyan-400" />
                      )}
                    </div>
                    <div className="min-w-0">
                      <div className="flex items-center gap-2">
                        <span className="text-xs font-bold text-[var(--text-primary)] truncate">
                          {act.skillName}
                        </span>
                        {getStatusBadge(act.status)}
                      </div>
                      <p className="text-xs text-[var(--text-secondary)] truncate mt-0.5">
                        {act.summary || act.action}
                      </p>
                    </div>
                  </div>

                  <div className="flex items-center gap-3 shrink-0 text-xs text-[var(--text-muted)] sm:self-center">
                    {act.duration && (
                      <span className="font-mono text-[11px] bg-[var(--bg-tertiary)] px-2 py-0.5 rounded">
                        {act.duration}
                      </span>
                    )}
                    <span className="text-[11px]">
                      {new Date(act.createdAt).toLocaleString(settings.language === 'fr' ? 'fr-FR' : 'en-US', {
                        hour: '2-digit',
                        minute: '2-digit',
                        day: '2-digit',
                        month: 'short',
                      })}
                    </span>
                    <button
                      onClick={e => {
                        e.stopPropagation()
                        handleInspectActivity(act)
                      }}
                      className="p-1 rounded-md text-[var(--text-muted)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-tertiary)]"
                      title="Inspecter dans Activités"
                    >
                      <ExternalLink size={13} />
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
