import React, { useRef } from 'react'
import {
  Search,
  Plus,
  Command,
  Columns,
  ListFilter,
  X,
  Sun,
  Moon,
  Globe,
  SlidersHorizontal,
  RefreshCw,
  Bot,
  Type,
  Eye,
  EyeOff
} from 'lucide-react'
import { useApp } from '../context/AppContext'
import type { Status, Priority, Density } from '../types'

export const Header: React.FC = () => {
  const {
    tasks,
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
    syncLinear,
    syncGithub,
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

  const handleQuickSync = () => {
    if (settings.issueTracker === 'github') {
      syncGithub()
    } else {
      syncLinear()
    }
  }

  const isMac = typeof window !== 'undefined' && navigator.platform.toUpperCase().indexOf('MAC') >= 0

  return (
    <header className="h-14 border-b border-[var(--border-color)] bg-[var(--bg-secondary)] px-4 flex items-center justify-between gap-3 shrink-0 z-10 select-none">
      {/* Left / Center: Global Search Bar */}
      <div className="flex-1 max-w-xl relative flex items-center">
        <div className="absolute left-3 text-[var(--text-muted)] pointer-events-none flex items-center">
          <Search size={15} />
        </div>
        <input
          id="global-search-input"
          ref={searchInputRef}
          type="text"
          value={searchQuery}
          onChange={e => setSearchQuery(e.target.value)}
          placeholder={t.header.searchPlaceholder}
          className="w-full pl-9 pr-14 py-1.5 text-xs rounded-lg bg-[var(--bg-tertiary)] border border-[var(--border-color)] text-[var(--text-primary)] placeholder-[var(--text-muted)] focus:outline-none focus:border-[var(--accent-color)] focus:ring-1 focus:ring-[var(--accent-color)] transition-all"
        />
        {searchQuery ? (
          <button
            onClick={() => setSearchQuery('')}
            className="absolute right-3 p-0.5 text-[var(--text-muted)] hover:text-[var(--text-primary)] transition-colors"
            title="Effacer la recherche (Esc)"
          >
            <X size={14} />
          </button>
        ) : (
          <kbd className="absolute right-2.5 px-1.5 py-0.5 text-[10px] font-mono text-[var(--text-muted)] bg-[var(--bg-secondary)] border border-[var(--border-color)] rounded pointer-events-none shadow-xs">
            /
          </kbd>
        )}
      </div>

      {/* Middle: Active Filter Chips */}
      <div className="hidden lg:flex items-center gap-1.5 overflow-x-auto max-w-xs">
        {statusFilter && (
          <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[11px] font-medium bg-[var(--accent-light)] accent-text">
            {t.status[statusFilter as Status]}
            <button onClick={() => setStatusFilter(null)} className="hover:opacity-75">
              <X size={11} />
            </button>
          </span>
        )}
        {priorityFilter && (
          <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[11px] font-medium bg-rose-500/15 text-rose-400">
            {t.priority[priorityFilter as Priority]}
            <button onClick={() => setPriorityFilter(null)} className="hover:opacity-75">
              <X size={11} />
            </button>
          </span>
        )}
        {labelFilter && (
          <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[11px] font-medium bg-cyan-500/15 text-cyan-400">
            #{labelFilter}
            <button onClick={() => setLabelFilter(null)} className="hover:opacity-75">
              <X size={11} />
            </button>
          </span>
        )}
        {assigneeFilter && (
          <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[11px] font-medium bg-emerald-500/15 text-emerald-400">
            {assigneeFilter}
            <button onClick={() => setAssigneeFilter(null)} className="hover:opacity-75">
              <X size={11} />
            </button>
          </span>
        )}
        {hasActiveFilters && (
          <button
            onClick={clearAllFilters}
            className="text-[11px] text-[var(--text-muted)] hover:text-[var(--text-primary)] underline transition-colors px-1"
          >
            {t.header.clearFilters}
          </button>
        )}
      </div>

      {/* Right Controls */}
      <div className="flex items-center gap-2">
        {/* Sync Issues button */}
        <button
          onClick={handleQuickSync}
          disabled={isSyncing}
          className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-xs font-semibold bg-[var(--bg-tertiary)] hover:bg-[var(--border-color)] text-[var(--text-secondary)] hover:text-[var(--text-primary)] border border-[var(--border-color)] transition-all disabled:opacity-50"
          title={`Synchroniser les issues avec ${settings.issueTracker.toUpperCase()}`}
        >
          <RefreshCw size={13} className={isSyncing ? 'animate-spin text-amber-400' : ''} />
          <span className="hidden sm:inline">
            {isSyncing ? t.header.syncing : `${t.header.syncNow} (${settings.issueTracker.toUpperCase()})`}
          </span>
        </button>

        {/* AI Engine Badge */}
        <div
          onClick={() => setIsProfileOpen(true)}
          className="cursor-pointer hidden md:flex items-center gap-1 px-2 py-1 rounded-md text-[11px] font-mono font-bold bg-[var(--accent-light)] accent-text border border-[var(--accent-color)]/40 hover:opacity-80 transition-opacity"
          title={`Moteur IA actif : ${settings.aiProvider.toUpperCase()}`}
        >
          <Bot size={13} />
          <span>{settings.aiProvider.toUpperCase()}</span>
        </div>

        {/* View Mode Switcher (Board / List) */}
        <div className="flex items-center bg-[var(--bg-tertiary)] p-0.5 rounded-lg border border-[var(--border-color)]">
          <button
            onClick={() => setActiveView('board')}
            className={`flex items-center gap-1.5 px-2.5 py-1 rounded-md text-xs font-medium transition-all ${
              activeView === 'board'
                ? 'bg-[var(--bg-secondary)] text-[var(--text-primary)] shadow-xs'
                : 'text-[var(--text-muted)] hover:text-[var(--text-primary)]'
            }`}
            title={t.header.boardView}
          >
            <Columns size={14} />
            <span className="hidden sm:inline">{t.header.boardView}</span>
          </button>
          <button
            onClick={() => setActiveView('list')}
            className={`flex items-center gap-1.5 px-2.5 py-1 rounded-md text-xs font-medium transition-all ${
              activeView === 'list'
                ? 'bg-[var(--bg-secondary)] text-[var(--text-primary)] shadow-xs'
                : 'text-[var(--text-muted)] hover:text-[var(--text-primary)]'
            }`}
            title={t.header.listView}
          >
            <ListFilter size={14} />
            <span className="hidden sm:inline">{t.header.listView}</span>
          </button>
        </div>

        {/* Hide / Show Done Stories Toggle */}
        <button
          onClick={toggleHideDone}
          className={`flex items-center gap-1 px-2.5 py-1.5 rounded-lg text-xs font-semibold border transition-all ${
            hideDone
              ? 'bg-[var(--bg-tertiary)] border-[var(--border-color)] text-[var(--text-muted)] hover:text-[var(--text-primary)]'
              : 'bg-emerald-500/10 border-emerald-500/30 text-emerald-400 hover:bg-emerald-500/20'
          }`}
          title={hideDone ? t.header.showDone : t.header.hideDone}
        >
          {hideDone ? <EyeOff size={13} className="text-slate-400" /> : <Eye size={13} className="text-emerald-400" />}
          <span className="hidden xl:inline text-[11px]">
            {hideDone ? 'Terminées masquées' : `Terminées (${tasks.filter(t => t.status === 'done').length})`}
          </span>
        </button>

        {/* Command Palette Button */}
        <button
          onClick={() => setIsCommandPaletteOpen(true)}
          className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-xs font-medium bg-[var(--bg-tertiary)] hover:bg-[var(--border-color)] text-[var(--text-secondary)] hover:text-[var(--text-primary)] border border-[var(--border-color)] transition-all"
          title={t.header.commandPalette}
        >
          <Command size={14} />
          <kbd className="hidden sm:inline-block px-1 py-0.2 text-[10px] font-mono text-[var(--text-muted)] bg-[var(--bg-secondary)] rounded">
            {isMac ? '⌘K' : 'Ctrl+K'}
          </kbd>
        </button>

        {/* Language quick toggle */}
        <button
          onClick={toggleLanguage}
          className="flex items-center gap-1 px-2 py-1.5 rounded-lg text-xs font-bold text-[var(--text-secondary)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-tertiary)] border border-[var(--border-color)] transition-all"
          title={`Langue: ${settings.language.toUpperCase()}`}
        >
          <Globe size={14} />
          <span>{settings.language.toUpperCase()}</span>
        </button>

        {/* Theme quick toggle */}
        <button
          onClick={toggleTheme}
          className="p-1.5 rounded-lg text-[var(--text-secondary)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-tertiary)] border border-[var(--border-color)] transition-all"
          title={settings.theme === 'dark' ? 'Passer en mode clair' : 'Passer en mode sombre'}
        >
          {settings.theme === 'dark' ? <Sun size={15} /> : <Moon size={15} />}
        </button>

        {/* Density quick toggle */}
        <button
          onClick={cycleDensity}
          className="flex items-center gap-1 px-2 py-1.5 rounded-lg text-xs font-mono font-bold text-[var(--text-secondary)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-tertiary)] border border-[var(--border-color)] transition-all"
          title={`Taille d'affichage : ${settings.density === 'compact' ? 'Compacte (13px)' : settings.density === 'standard' ? 'Standard (14px)' : 'Confortable (15.5px)'} - Cliquer pour changer`}
        >
          <Type size={13} className="text-cyan-400" />
          <span className="text-[10px] uppercase font-bold">{settings.density.slice(0, 3)}</span>
        </button>

        {/* Settings modal trigger */}
        <button
          onClick={() => setIsProfileOpen(true)}
          className="p-1.5 rounded-lg text-[var(--text-secondary)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-tertiary)] border border-[var(--border-color)] transition-all"
          title="Configuration & Paramètres"
        >
          <SlidersHorizontal size={15} />
        </button>

        {/* Quick Add Button */}
        <button
          onClick={() => setIsQuickAddOpen(true)}
          className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-semibold text-white shadow-md accent-bg hover:opacity-90 active:scale-95 transition-all"
          title={`${t.header.quickAdd} (N)`}
        >
          <Plus size={15} />
          <span className="hidden sm:inline">{t.header.quickAdd}</span>
        </button>
      </div>
    </header>
  )
}
