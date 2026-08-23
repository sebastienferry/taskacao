import React, { useRef } from 'react'
import {
  Search,
  Plus,
  Command,
  Columns,
  ListFilter,
  Activity,
  X,
  Sun,
  Moon,
  Globe,
  SlidersHorizontal,
  RefreshCw,
  Type,
  Eye,
  EyeOff,
  GitBranch,
} from 'lucide-react'
import { useApp } from '../context/AppContext'
import type { Status, Priority, Density } from '../types'

export const Header: React.FC = () => {
  const {
    currentProject,
    setIsProjectModalOpen,
    setEditingProject,
    tasks,
    gitStatus,
    activeJobCount,
    searchQuery,
    setSearchQuery,
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
    hideDone,
    toggleHideDone,
    setIsQuickAddOpen,
    setIsCommandPaletteOpen,
    setIsProfileOpen,
    setIsBranchModalOpen,
    isSyncing,
    settings,
    updateSettings,
    t,
  } = useApp()

  const searchInputRef = useRef<HTMLInputElement>(null)

  const hasActiveFilters = Boolean(statusFilter || priorityFilter || labelFilter || assigneeFilter || searchQuery)

  const clearAllFilters = () => {
    setSearchQuery('')
    setStatusFilter(null)
    setPriorityFilter(null)
    setLabelFilter(null)
    setAssigneeFilter(null)
  }

  const toggleTheme = () => {
    const nextTheme = settings.theme === 'dark' ? 'light' : 'dark'
    updateSettings({ theme: nextTheme })
  }

  const toggleLanguage = () => {
    const nextLang = settings.language === 'fr' ? 'en' : 'fr'
    updateSettings({ language: nextLang })
  }

  const cycleDensity = () => {
    const nextDensity: Density =
      settings.density === 'compact' ? 'standard' :
      settings.density === 'standard' ? 'comfortable' : 'compact'
    updateSettings({ density: nextDensity })
  }

  const isMac = typeof window !== 'undefined' && navigator.platform.toUpperCase().indexOf('MAC') >= 0

  return (
    <header
      className="h-14 border-b border-[var(--header-border)] px-4 flex items-center justify-between gap-3 shrink-0 z-10 select-none text-slate-100 shadow-xs"
      style={{
        background: 'linear-gradient(90deg, var(--header-accent-tint) 0%, var(--header-bg) 50%)',
      }}
    >
      {/* Left: Active Project Indicator Badge */}
      {currentProject && (
        <div
          onClick={() => {
            setEditingProject(currentProject)
            setIsProjectModalOpen(true)
          }}
          className={`hidden md:flex items-center gap-2 px-2.5 py-1 rounded-xl bg-[#131d31]/90 border border-slate-700/60 hover:border-[var(--accent-color)]/60 text-xs font-bold text-slate-200 transition-all cursor-pointer shadow-xs shrink-0 group`}
          title={`Espace de travail actif : ${currentProject.name} — Cliquer pour configurer`}
        >
          <span className={`w-2 h-2 rounded-full accent-bg shadow-[0_0_8px_var(--accent-glow)]`} />
          <span className="truncate max-w-[140px]">{currentProject.name}</span>
          <span className="text-[10px] font-mono px-1.5 py-0.2 rounded-md bg-[var(--accent-light)] accent-text font-bold">
            {tasks.length}
          </span>
        </div>
      )}

      {/* Center: Global Search Bar */}
      <div className="flex-1 max-w-xl relative flex items-center">
        <div className="absolute left-3 text-slate-400 pointer-events-none flex items-center">
          <Search size={15} />
        </div>
        <input
          id="global-search-input"
          ref={searchInputRef}
          type="text"
          value={searchQuery}
          onChange={e => setSearchQuery(e.target.value)}
          placeholder={t.header.searchPlaceholder}
          className="w-full pl-9 pr-14 py-1.5 text-xs rounded-lg bg-[var(--header-input-bg)] border border-[var(--header-input-border)] text-slate-100 placeholder-slate-400 focus:outline-none focus:border-[var(--accent-color)] focus:ring-1 focus:ring-[var(--accent-color)] transition-all"
        />
        {searchQuery ? (
          <button
            onClick={() => setSearchQuery('')}
            className="absolute right-3 p-0.5 text-slate-400 hover:text-white transition-colors cursor-pointer"
            title="Effacer la recherche (Esc)"
          >
            <X size={14} />
          </button>
        ) : (
          <kbd className="absolute right-2.5 px-1.5 py-0.5 text-[10px] font-mono text-slate-400 bg-[#0c1220] border border-slate-700/60 rounded pointer-events-none shadow-xs">
            /
          </kbd>
        )}
      </div>

      {/* Middle: Active Filter Chips */}
      <div className="hidden lg:flex items-center gap-1.5 overflow-x-auto max-w-xs">
        {statusFilter && (
          <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[11px] font-bold bg-[var(--accent-light)] accent-text border border-[var(--accent-color)]/30">
            {t.status[statusFilter as Status]}
            <button onClick={() => setStatusFilter(null)} className="hover:opacity-75 cursor-pointer">
              <X size={11} />
            </button>
          </span>
        )}
        {priorityFilter && (
          <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[11px] font-bold bg-rose-500/20 text-rose-300 border border-rose-500/40">
            {t.priority[priorityFilter as Priority]}
            <button onClick={() => setPriorityFilter(null)} className="hover:opacity-75 cursor-pointer">
              <X size={11} />
            </button>
          </span>
        )}
        {labelFilter && (
          <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[11px] font-bold bg-cyan-500/20 text-cyan-300 border border-cyan-500/40">
            #{labelFilter}
            <button onClick={() => setLabelFilter(null)} className="hover:opacity-75 cursor-pointer">
              <X size={11} />
            </button>
          </span>
        )}
        {assigneeFilter && (
          <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[11px] font-bold bg-emerald-500/20 text-emerald-300 border border-emerald-500/40">
            {assigneeFilter}
            <button onClick={() => setAssigneeFilter(null)} className="hover:opacity-75 cursor-pointer">
              <X size={11} />
            </button>
          </span>
        )}
        {hasActiveFilters && (
          <button
            onClick={clearAllFilters}
            className="text-[11px] text-slate-400 hover:text-white underline transition-colors px-1 cursor-pointer"
          >
            {t.header.clearFilters}
          </button>
        )}
      </div>

      {/* Right Controls */}
      <div className="flex items-center gap-2">
        {/* Active Git Branch Badge */}
        {gitStatus?.branch && (
          <div
            onClick={() => setIsBranchModalOpen(true)}
            className="hidden sm:flex items-center gap-1.5 px-2.5 py-1 rounded-lg text-[11px] font-mono font-semibold bg-[#131d31] border border-slate-700/60 text-slate-300 hover:text-white hover:border-cyan-500/50 hover:bg-cyan-500/15 cursor-pointer transition-all shadow-xs"
            title={`Branche Git active : ${gitStatus.branch} (${gitStatus.isClean ? 'Clean' : '*' + (gitStatus.modifiedCount + gitStatus.untrackedCount)}) - Cliquer pour changer de branche`}
          >
            <GitBranch size={13} className="text-cyan-400 shrink-0" />
            <span className="text-cyan-300 font-bold truncate max-w-[160px]">{gitStatus.branch}</span>
            {gitStatus.isClean ? (
              <span className="w-1.5 h-1.5 rounded-full bg-emerald-400" title="Arbre propre" />
            ) : (
              <span className="text-[10px] text-amber-400 font-bold" title="Fichiers modifiés">
                *{(gitStatus.modifiedCount + gitStatus.untrackedCount)}
              </span>
            )}
          </div>
        )}

        {/* View Mode Switcher (Board / List / Activities / Sync) */}
        <div className="flex items-center bg-[#131d31] p-0.5 rounded-lg border border-slate-700/60">
          <button
            onClick={() => setActiveView('board')}
            className={`flex items-center gap-1.5 px-2.5 py-1 rounded-md text-xs font-medium transition-all cursor-pointer ${
              activeView === 'board'
                ? 'bg-[#1e293b] text-white shadow-xs font-bold border border-slate-600/50'
                : 'text-slate-400 hover:text-white'
            }`}
            title={t.header.boardView}
          >
            <Columns size={14} />
            <span className="hidden sm:inline">{t.header.boardView}</span>
          </button>
          <button
            onClick={() => setActiveView('list')}
            className={`flex items-center gap-1.5 px-2.5 py-1 rounded-md text-xs font-medium transition-all cursor-pointer ${
              activeView === 'list'
                ? 'bg-[#1e293b] text-white shadow-xs font-bold border border-slate-600/50'
                : 'text-slate-400 hover:text-white'
            }`}
            title={t.header.listView}
          >
            <ListFilter size={14} />
            <span className="hidden sm:inline">{t.header.listView}</span>
          </button>
          <button
            onClick={() => setActiveView('activities')}
            className={`flex items-center gap-1.5 px-2.5 py-1 rounded-md text-xs font-medium transition-all cursor-pointer ${
              activeView === 'activities'
                ? 'bg-[#1e293b] text-cyan-300 shadow-xs font-bold border border-slate-600/50'
                : 'text-slate-400 hover:text-white'
            }`}
            title={t.header.activitiesView}
          >
            <Activity size={14} className={activeJobCount > 0 ? 'text-cyan-400 animate-pulse' : ''} />
            <span className="hidden sm:inline">{t.header.activitiesView}</span>
            {activeJobCount > 0 && (
              <span className="w-2 h-2 rounded-full bg-cyan-400 animate-ping" />
            )}
          </button>
          <button
            onClick={() => setActiveView('sync')}
            className={`flex items-center gap-1.5 px-2.5 py-1 rounded-md text-xs font-medium transition-all cursor-pointer ${
              activeView === 'sync'
                ? 'bg-[#1e293b] text-indigo-300 shadow-xs font-bold border border-slate-600/50'
                : 'text-slate-400 hover:text-white'
            }`}
            title={t.nav.sync || "Synchronisation"}
          >
            <RefreshCw size={13} className={isSyncing ? 'animate-spin text-indigo-400' : ''} />
            <span className="hidden sm:inline">{t.nav.sync || "Synchro"}</span>
          </button>
        </div>

        {/* Hide / Show Done Stories Toggle */}
        <button
          onClick={toggleHideDone}
          className={`flex items-center gap-1 px-2.5 py-1.5 rounded-lg text-xs font-semibold border transition-all cursor-pointer ${
            hideDone
              ? 'bg-[#131d31] border-slate-700/60 text-slate-400 hover:text-white'
              : 'bg-emerald-500/15 border-emerald-500/40 text-emerald-300 hover:bg-emerald-500/25'
          }`}
          title={hideDone ? t.header.showDone : t.header.hideDone}
        >
          {hideDone ? <EyeOff size={13} className="text-slate-400" /> : <Eye size={13} className="text-emerald-400" />}
          <span className="hidden xl:inline text-[11px]">
            {hideDone ? 'Terminées masquées' : `Terminées (${tasks.filter(t => t.status === 'done' || t.status === 'finished').length})`}
          </span>
        </button>

        {/* Command Palette Button */}
        <button
          onClick={() => setIsCommandPaletteOpen(true)}
          className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-xs font-medium bg-[#131d31] hover:bg-[#1e293b] text-slate-300 hover:text-white border border-slate-700/60 transition-all cursor-pointer"
          title={t.header.commandPalette}
        >
          <Command size={14} />
          <kbd className="hidden sm:inline-block px-1 py-0.2 text-[10px] font-mono text-slate-400 bg-[#0c1220] rounded border border-slate-700/50">
            {isMac ? '⌘K' : 'Ctrl+K'}
          </kbd>
        </button>

        {/* Language quick toggle */}
        <button
          onClick={toggleLanguage}
          className="flex items-center gap-1 px-2 py-1.5 rounded-lg text-xs font-bold text-slate-300 hover:text-white hover:bg-[#1e293b] bg-[#131d31] border border-slate-700/60 transition-all cursor-pointer"
          title={`Langue: ${settings.language.toUpperCase()}`}
        >
          <Globe size={14} />
          <span>{settings.language.toUpperCase()}</span>
        </button>

        {/* Theme quick toggle */}
        <button
          onClick={toggleTheme}
          className="p-1.5 rounded-lg text-slate-300 hover:text-white hover:bg-[#1e293b] bg-[#131d31] border border-slate-700/60 transition-all cursor-pointer"
          title={settings.theme === 'dark' ? 'Passer en mode clair' : 'Passer en mode sombre'}
        >
          {settings.theme === 'dark' ? <Sun size={15} /> : <Moon size={15} />}
        </button>

        {/* Density quick toggle */}
        <button
          onClick={cycleDensity}
          className="flex items-center gap-1 px-2 py-1.5 rounded-lg text-xs font-mono font-bold text-slate-300 hover:text-white hover:bg-[#1e293b] bg-[#131d31] border border-slate-700/60 transition-all cursor-pointer"
          title={`Taille d'affichage : ${settings.density === 'compact' ? 'Compacte (13px)' : settings.density === 'standard' ? 'Standard (14px)' : 'Confortable (15.5px)'} - Cliquer pour changer`}
        >
          <Type size={13} className="text-cyan-400" />
          <span className="text-[10px] uppercase font-bold">{settings.density.slice(0, 3)}</span>
        </button>

        {/* Settings modal trigger */}
        <button
          onClick={() => setIsProfileOpen(true)}
          className="p-1.5 rounded-lg text-slate-300 hover:text-white hover:bg-[#1e293b] bg-[#131d31] border border-slate-700/60 transition-all cursor-pointer"
          title="Configuration & Paramètres"
        >
          <SlidersHorizontal size={15} />
        </button>

        {/* Quick Add Button */}
        <button
          onClick={() => setIsQuickAddOpen(true)}
          className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-semibold text-white shadow-md accent-bg hover:opacity-90 active:scale-95 transition-all cursor-pointer"
          title={`${t.header.quickAdd} (N)`}
        >
          <Plus size={15} />
          <span className="hidden sm:inline">{t.header.quickAdd}</span>
        </button>
      </div>
    </header>
  )
}
