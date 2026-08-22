import React from 'react'
import {
  Inbox,
  HelpCircle,
  FileCode,
  Flame,
  CheckCircle2,
  Columns,
  ListFilter,
  Tag,
  Settings,
  ChevronLeft,
  ChevronRight,
  ShieldCheck,
  User,
  Layers,
  RotateCcw,
  RefreshCw,
  Bot,
  Globe2,
  FolderGit2,
  Folder
} from 'lucide-react'
import { useApp } from '../context/AppContext'
import type { Status, TaskSource } from '../types'

export const Sidebar: React.FC = () => {
  const {
    tasks,
    activeView,
    setActiveView,
    statusFilter,
    setStatusFilter,
    priorityFilter,
    setPriorityFilter,
    labelFilter,
    setLabelFilter,
    assigneeFilter,
    setAssigneeFilter,
    sourceFilter,
    setSourceFilter,
    sidebarCollapsed,
    setSidebarCollapsed,
    setIsProfileOpen,
    syncAll,
    syncLinear,
    syncGithub,
    isSyncing,
    reseedDemo,
    settings,
    availableLabels,
    t,
  } = useApp()

  const counts: Record<string, number> = {
    all: tasks.length,
    to_clarify: tasks.filter(t => t.status === 'to_clarify' || t.status === 'backlog').length,
    to_specify: tasks.filter(t => t.status === 'to_specify' || t.status === 'specified').length,
    to_implement: tasks.filter(t => t.status === 'to_implement' || t.status === 'in_progress').length,
    to_test: tasks.filter(t => t.status === 'to_test' || t.status === 'to_validate').length,
    to_close: tasks.filter(t => t.status === 'to_close' || t.status === 'done').length,
    srcLinear: tasks.filter(t => t.source === 'linear').length,
    srcGithub: tasks.filter(t => t.source === 'github').length,
    srcLocal: tasks.filter(t => !t.source || t.source === 'local').length,
  }

  const workflowItems: { status: Status | null; label: string; stageLabel: string; stageColor: string; icon: React.ReactNode; count: number; color: string }[] = [
    { status: null, label: t.nav.allTasks, stageLabel: '', stageColor: '', icon: <Inbox size={16} />, count: counts.all, color: 'text-slate-400' },
    { status: 'to_clarify', label: t.status.to_clarify, stageLabel: 'New', stageColor: 'bg-cyan-500/15 text-cyan-400 border-cyan-500/30', icon: <HelpCircle size={16} />, count: counts.to_clarify, color: 'text-cyan-400' },
    { status: 'to_specify', label: t.status.to_specify, stageLabel: 'Clarified', stageColor: 'bg-amber-500/15 text-amber-400 border-amber-500/30', icon: <FileCode size={16} />, count: counts.to_specify, color: 'text-amber-400' },
    { status: 'to_implement', label: t.status.to_implement, stageLabel: 'Specified', stageColor: 'bg-blue-500/15 text-blue-400 border-blue-500/30', icon: <Flame size={16} />, count: counts.to_implement, color: 'text-blue-400' },
    { status: 'to_test', label: t.status.to_test, stageLabel: 'Implemented', stageColor: 'bg-indigo-500/15 text-indigo-400 border-indigo-500/30', icon: <ShieldCheck size={16} />, count: counts.to_test, color: 'text-indigo-400' },
    { status: 'to_close', label: t.status.to_close, stageLabel: 'Reviewed', stageColor: 'bg-emerald-500/15 text-emerald-400 border-emerald-500/30', icon: <CheckCircle2 size={16} />, count: counts.to_close, color: 'text-emerald-400' },
  ]

  const sourceItems: { id: 'all' | TaskSource; label: string; icon: React.ReactNode; count: number; color: string }[] = [
    { id: 'all', label: 'Toutes les sources', icon: <Globe2 size={15} />, count: counts.all, color: 'text-slate-400' },
    { id: 'linear', label: `Linear (${settings.linearTeam || 'FRE'})`, icon: <span className="font-bold text-indigo-400 font-mono text-xs">◆</span>, count: counts.srcLinear, color: 'text-indigo-400' },
    { id: 'github', label: `GitHub (${settings.githubRepo ? settings.githubRepo.split('/')[1] || 'GH' : 'GH'})`, icon: <FolderGit2 size={15} />, count: counts.srcGithub, color: 'text-purple-400' },
    { id: 'local', label: 'Local SQLite', icon: <Folder size={15} />, count: counts.srcLocal, color: 'text-emerald-400' },
  ]

  const isMyTasksActive = assigneeFilter === settings.userName
  const isUrgentActive = priorityFilter === 'urgent' || priorityFilter === 'high'

  return (
    <aside
      className={`relative flex flex-col border-r transition-all duration-300 ease-in-out select-none bg-[var(--bg-secondary)] border-[var(--border-color)] ${
        sidebarCollapsed ? 'w-16' : 'w-64'
      } h-screen z-20 shrink-0`}
    >
      {/* Brand Header */}
      <div className="flex items-center justify-between h-14 px-3.5 border-b border-[var(--border-color)]">
        {!sidebarCollapsed ? (
          <div className="flex items-center gap-2.5 overflow-hidden">
            <div className="w-8 h-8 rounded-lg flex items-center justify-center font-bold text-white shadow-md accent-bg shrink-0">
              <Layers size={18} />
            </div>
            <div className="flex flex-col min-w-0">
              <span className="font-semibold tracking-tight text-sm truncate text-[var(--text-primary)]">
                {t.app.title}
              </span>
              <span className="text-[10px] font-mono text-[var(--text-muted)] truncate flex items-center gap-1">
                <span className="w-1.5 h-1.5 rounded-full bg-emerald-400"></span>
                AI: {settings.aiProvider.toUpperCase()} | Multi-Tracker
              </span>
            </div>
          </div>
        ) : (
          <div className="mx-auto w-8 h-8 rounded-lg flex items-center justify-center font-bold text-white shadow-md accent-bg">
            <Layers size={18} />
          </div>
        )}

        <button
          onClick={() => setSidebarCollapsed(prev => !prev)}
          className="p-1.5 rounded-md text-[var(--text-muted)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-tertiary)] transition-colors"
          title={sidebarCollapsed ? 'Déplier' : 'Replier'}
        >
          {sidebarCollapsed ? <ChevronRight size={16} /> : <ChevronLeft size={16} />}
        </button>
      </div>

      {/* Navigation & Filters Container */}
      <div className="flex-1 overflow-y-auto px-2.5 py-3 space-y-4">
        {/* Quick Views */}
        <div>
          {!sidebarCollapsed && (
            <div className="px-2 pb-1 text-[10px] font-bold uppercase tracking-wider text-[var(--text-muted)]">
              Vues
            </div>
          )}
          <div className="space-y-0.5">
            <button
              onClick={() => setActiveView('board')}
              className={`w-full flex items-center gap-2.5 px-2.5 py-1.5 rounded-md text-xs font-medium transition-all ${
                activeView === 'board'
                  ? 'bg-[var(--accent-light)] accent-text font-semibold'
                  : 'text-[var(--text-secondary)] hover:bg-[var(--bg-tertiary)] hover:text-[var(--text-primary)]'
              }`}
              title={t.nav.board}
            >
              <Columns size={15} className="shrink-0" />
              {!sidebarCollapsed && <span className="truncate">{t.nav.board}</span>}
            </button>
            <button
              onClick={() => setActiveView('list')}
              className={`w-full flex items-center gap-2.5 px-2.5 py-1.5 rounded-md text-xs font-medium transition-all ${
                activeView === 'list'
                  ? 'bg-[var(--accent-light)] accent-text font-semibold'
                  : 'text-[var(--text-secondary)] hover:bg-[var(--bg-tertiary)] hover:text-[var(--text-primary)]'
              }`}
              title={t.nav.list}
            >
              <ListFilter size={15} className="shrink-0" />
              {!sidebarCollapsed && <span className="truncate">{t.nav.list}</span>}
            </button>
          </div>
        </div>

        {/* Source Filter Section (Linear / GitHub / Local) */}
        <div>
          {!sidebarCollapsed && (
            <div className="px-2 pb-1 text-[10px] font-bold uppercase tracking-wider text-[var(--text-muted)] flex items-center justify-between">
              <span>Trackers & Sources</span>
              <button
                onClick={syncAll}
                disabled={isSyncing}
                className="text-[10px] text-[var(--accent-color)] hover:underline flex items-center gap-1"
                title="Tout synchroniser (Linear + GitHub)"
              >
                <RefreshCw size={10} className={isSyncing ? 'animate-spin' : ''} />
                <span>Sync</span>
              </button>
            </div>
          )}
          <div className="space-y-0.5">
            {sourceItems.map(src => {
              const isSelected = sourceFilter === src.id
              return (
                <button
                  key={src.id}
                  onClick={() => setSourceFilter(src.id)}
                  className={`w-full flex items-center justify-between px-2.5 py-1.5 rounded-md text-xs font-medium transition-all ${
                    isSelected
                      ? 'bg-[var(--accent-light)] accent-text font-semibold'
                      : 'text-[var(--text-secondary)] hover:bg-[var(--bg-tertiary)] hover:text-[var(--text-primary)]'
                  }`}
                  title={src.label}
                >
                  <div className="flex items-center gap-2 truncate">
                    <span className={`${src.color} shrink-0`}>{src.icon}</span>
                    {!sidebarCollapsed && <span className="truncate">{src.label}</span>}
                  </div>
                  {!sidebarCollapsed && (
                    <span
                      className={`text-[10px] px-1.5 py-0.2 rounded-full font-mono ${
                        isSelected
                          ? 'bg-[var(--accent-color)] text-white font-bold'
                          : 'bg-[var(--bg-tertiary)] text-[var(--text-muted)]'
                      }`}
                    >
                      {src.count}
                    </span>
                  )}
                </button>
              )
            })}
          </div>
        </div>

        {/* Sync Controls buttons */}
        {!sidebarCollapsed && (
          <div className="p-2 rounded-xl bg-[var(--bg-tertiary)]/40 border border-[var(--border-color)] space-y-1.5">
            <button
              onClick={syncAll}
              disabled={isSyncing}
              className="w-full flex items-center justify-center gap-1.5 py-1.5 px-2 rounded-lg text-xs font-bold text-white accent-bg shadow-xs hover:opacity-90 active:scale-95 transition-all disabled:opacity-50"
            >
              <RefreshCw size={12} className={isSyncing ? 'animate-spin' : ''} />
              <span>Tout Synchroniser</span>
            </button>
            <div className="grid grid-cols-2 gap-1 text-[10px]">
              <button
                onClick={syncLinear}
                disabled={isSyncing}
                className="py-1 px-1.5 rounded bg-[var(--bg-secondary)] hover:bg-[var(--border-color)] text-[var(--text-secondary)] hover:text-[var(--text-primary)] border border-[var(--border-color)] font-semibold truncate transition-colors"
                title="Sync Linear CLI"
              >
                ◆ Linear
              </button>
              <button
                onClick={syncGithub}
                disabled={isSyncing}
                className="py-1 px-1.5 rounded bg-[var(--bg-secondary)] hover:bg-[var(--border-color)] text-[var(--text-secondary)] hover:text-[var(--text-primary)] border border-[var(--border-color)] font-semibold truncate transition-colors"
                title="Sync GitHub CLI"
              >
                🐙 GitHub
              </button>
            </div>
          </div>
        )}

        {/* Workflow Stages */}
        <div>
          {!sidebarCollapsed && (
            <div className="px-2 pb-1 text-[10px] font-bold uppercase tracking-wider text-[var(--text-muted)]">
              Agentic Workflow
            </div>
          )}
          <div className="space-y-0.5">
            {workflowItems.map(item => {
              const isActive = statusFilter === item.status && !assigneeFilter && !priorityFilter && !labelFilter
              return (
                <button
                  key={item.status || 'all'}
                  onClick={() => {
                    setStatusFilter(item.status)
                    setAssigneeFilter(null)
                    setPriorityFilter(null)
                    setLabelFilter(null)
                  }}
                  className={`w-full flex items-center justify-between px-2.5 py-1.5 rounded-md text-xs font-medium transition-all ${
                    isActive
                      ? 'bg-[var(--accent-light)] accent-text font-semibold'
                      : 'text-[var(--text-secondary)] hover:bg-[var(--bg-tertiary)] hover:text-[var(--text-primary)]'
                  }`}
                  title={item.label}
                >
                  <div className="flex items-center gap-2 truncate">
                    <span className={`${item.color} shrink-0`}>{item.icon}</span>
                    {!sidebarCollapsed && (
                      <div className="flex items-center gap-1.5 truncate">
                        <span className="truncate">{item.label}</span>
                        {item.stageLabel && (
                          <span className={`text-[9px] font-bold px-1 py-0 rounded border ${item.stageColor}`}>
                            {item.stageLabel}
                          </span>
                        )}
                      </div>
                    )}
                  </div>
                  {!sidebarCollapsed && (
                    <span
                      className={`text-[10px] px-1.5 py-0.2 rounded-full font-mono ${
                        isActive
                          ? 'bg-[var(--accent-color)] text-white font-bold'
                          : 'bg-[var(--bg-tertiary)] text-[var(--text-muted)]'
                      }`}
                    >
                      {item.count}
                    </span>
                  )}
                </button>
              )
            })}
          </div>
        </div>

        {/* Quick Filters */}
        <div>
          {!sidebarCollapsed && (
            <div className="px-2 pb-1 text-[10px] font-bold uppercase tracking-wider text-[var(--text-muted)]">
              {t.nav.filters}
            </div>
          )}
          <div className="space-y-0.5">
            <button
              onClick={() => setAssigneeFilter(isMyTasksActive ? null : settings.userName)}
              className={`w-full flex items-center justify-between px-2.5 py-1.5 rounded-md text-xs font-medium transition-all ${
                isMyTasksActive
                  ? 'bg-[var(--accent-light)] accent-text font-semibold'
                  : 'text-[var(--text-secondary)] hover:bg-[var(--bg-tertiary)] hover:text-[var(--text-primary)]'
              }`}
              title={t.nav.myTasks}
            >
              <div className="flex items-center gap-2.5 truncate">
                <User size={15} className="text-cyan-400 shrink-0" />
                {!sidebarCollapsed && <span className="truncate">{t.nav.myTasks}</span>}
              </div>
            </button>
            <button
              onClick={() => setPriorityFilter(isUrgentActive ? null : 'urgent')}
              className={`w-full flex items-center justify-between px-2.5 py-1.5 rounded-md text-xs font-medium transition-all ${
                isUrgentActive
                  ? 'bg-[var(--accent-light)] accent-text font-semibold'
                  : 'text-[var(--text-secondary)] hover:bg-[var(--bg-tertiary)] hover:text-[var(--text-primary)]'
              }`}
              title={t.nav.urgentHigh}
            >
              <div className="flex items-center gap-2.5 truncate">
                <Flame size={15} className="text-rose-400 shrink-0" />
                {!sidebarCollapsed && <span className="truncate">{t.nav.urgentHigh}</span>}
              </div>
            </button>
          </div>
        </div>

        {/* Labels Section */}
        {availableLabels.length > 0 && (
          <div>
            {!sidebarCollapsed && (
              <div className="px-2 pb-1 text-[10px] font-bold uppercase tracking-wider text-[var(--text-muted)]">
                {t.nav.labels}
              </div>
            )}
            <div className="space-y-0.5">
              {availableLabels.slice(0, 6).map(lbl => {
                const isSelected = labelFilter === lbl
                return (
                  <button
                    key={lbl}
                    onClick={() => setLabelFilter(isSelected ? null : lbl)}
                    className={`w-full flex items-center gap-2 px-2.5 py-1 rounded-md text-xs transition-all ${
                      isSelected
                        ? 'bg-[var(--accent-light)] accent-text font-semibold'
                        : 'text-[var(--text-secondary)] hover:bg-[var(--bg-tertiary)] hover:text-[var(--text-primary)]'
                    }`}
                    title={lbl}
                  >
                    <Tag size={12} className="shrink-0 text-slate-400" />
                    {!sidebarCollapsed && <span className="truncate">#{lbl}</span>}
                  </button>
                )
              })}
            </div>
          </div>
        )}
      </div>

      {/* Footer Profile & Settings */}
      <div className="p-2 border-t border-[var(--border-color)] space-y-1">
        <button
          onClick={() => setIsProfileOpen(true)}
          className="w-full flex items-center gap-2.5 p-2 rounded-lg hover:bg-[var(--bg-tertiary)] text-[var(--text-primary)] transition-colors group text-left"
          title={t.nav.settings}
        >
          <div className="w-7 h-7 rounded-full flex items-center justify-center font-bold text-xs accent-bg text-white shadow shrink-0">
            {settings.userName ? settings.userName.substring(0, 2).toUpperCase() : 'SF'}
          </div>
          {!sidebarCollapsed && (
            <div className="flex-1 min-w-0">
              <div className="text-xs font-semibold truncate text-[var(--text-primary)]">
                {settings.userName || 'Sylvain Ferry'}
              </div>
              <div className="text-[10px] text-[var(--text-muted)] truncate flex items-center gap-1 font-mono">
                <Bot size={11} className="text-amber-400" />
                {settings.aiProvider.toUpperCase()} | Linear+GH
              </div>
            </div>
          )}
          {!sidebarCollapsed && (
            <Settings size={15} className="text-[var(--text-muted)] group-hover:text-[var(--text-primary)] transition-transform group-hover:rotate-45" />
          )}
        </button>

        {!sidebarCollapsed && (
          <button
            onClick={() => reseedDemo()}
            className="w-full flex items-center gap-2 px-2.5 py-1 text-[10px] text-[var(--text-muted)] hover:text-[var(--text-secondary)] hover:bg-[var(--bg-tertiary)] rounded transition-colors"
            title={t.nav.reseedDemo}
          >
            <RotateCcw size={11} />
            <span className="truncate">{t.nav.reseedDemo}</span>
          </button>
        )}
      </div>
    </aside>
  )
}
