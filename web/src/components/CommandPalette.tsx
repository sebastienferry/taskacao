import React, { useState, useEffect, useRef } from 'react'
import {
  Search,
  Plus,
  Columns,
  ListFilter,
  Activity,
  Sun,
  Moon,
  Globe,
  Settings,
  RotateCcw,
  ArrowRight,
  Sparkles,
  HelpCircle,
  FileCode,
  Flame,
  ShieldCheck,
  Layers,
} from 'lucide-react'
import { useApp } from '../context/AppContext'

export const CommandPalette: React.FC = () => {
  const {
    isCommandPaletteOpen,
    setIsCommandPaletteOpen,
    tasks,
    projects,
    setSelectedProjectId,
    setIsProjectModalOpen,
    setEditingProject,
    setActiveView,
    setIsQuickAddOpen,
    setIsProfileOpen,
    setSelectedTask,
    settings,
    updateSettings,
    reseedDemo,
    skills,
    runSkill,
    t,
  } = useApp()

  const [query, setQuery] = useState('')
  const [selectedIndex, setSelectedIndex] = useState(0)
  const inputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    if (isCommandPaletteOpen) {
      setQuery('')
      setSelectedIndex(0)
      setTimeout(() => {
        inputRef.current?.focus()
      }, 50)
    }
  }, [isCommandPaletteOpen])

  useEffect(() => {
    if (!isCommandPaletteOpen) return
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        setIsCommandPaletteOpen(false)
      }
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [isCommandPaletteOpen, setIsCommandPaletteOpen])

  if (!isCommandPaletteOpen) return null

  const getSkillIcon = (iconName: string) => {
    switch (iconName) {
      case 'HelpCircle':
        return <HelpCircle size={16} className="text-amber-400" />
      case 'FileCode':
        return <FileCode size={16} className="text-blue-400" />
      case 'Flame':
        return <Flame size={16} className="text-indigo-400" />
      case 'ShieldCheck':
        return <ShieldCheck size={16} className="text-purple-400" />
      default:
        return <Sparkles size={16} className="text-emerald-400" />
    }
  }

  // Built-in actions list
  const generalActions = [
    {
      id: 'create_task',
      title: t.commandPalette.createTask,
      icon: <Plus size={16} className="text-emerald-400" />,
      shortcut: 'N',
      action: () => {
        setIsCommandPaletteOpen(false)
        setIsQuickAddOpen(true)
      },
    },
    {
      id: 'switch_board',
      title: t.commandPalette.switchBoard,
      icon: <Columns size={16} className="text-indigo-400" />,
      shortcut: 'B',
      action: () => {
        setActiveView('board')
        setIsCommandPaletteOpen(false)
      },
    },
    {
      id: 'switch_list',
      title: t.commandPalette.switchList,
      icon: <ListFilter size={16} className="text-blue-400" />,
      shortcut: 'L',
      action: () => {
        setActiveView('list')
        setIsCommandPaletteOpen(false)
      },
    },
    {
      id: 'switch_activities',
      title: t.nav.activities,
      icon: <Activity size={16} className="text-cyan-400" />,
      shortcut: 'A',
      action: () => {
        setActiveView('activities')
        setIsCommandPaletteOpen(false)
      },
    },
    {
      id: 'switch_sync',
      title: t.nav.sync || 'Synchronisation',
      icon: <RotateCcw size={16} className="text-indigo-400" />,
      shortcut: 'S',
      action: () => {
        setActiveView('sync')
        setIsCommandPaletteOpen(false)
      },
    },
    {
      id: 'new_project',
      title: '📁 Créer un nouveau projet...',
      icon: <Plus size={16} className="text-indigo-400" />,
      shortcut: 'Shift+P',
      action: () => {
        setEditingProject(null)
        setIsProjectModalOpen(true)
        setIsCommandPaletteOpen(false)
      },
    },
    ...projects.map(p => ({
      id: `switch_proj_${p.id}`,
      title: `Basculer sur le projet: ${p.name}${p.linearTeam ? ` (${p.linearTeam})` : ''}`,
      icon: <Layers size={16} className={`text-${p.color || 'indigo'}-400`} />,
      shortcut: p.slug.substring(0, 3).toUpperCase(),
      action: () => {
        setSelectedProjectId(p.id)
        setIsCommandPaletteOpen(false)
      },
    })),
    {
      id: 'switch_proj_all',
      title: 'Voir tous les projets combinés',
      icon: <Layers size={16} className="text-slate-400" />,
      shortcut: 'ALL',
      action: () => {
        setSelectedProjectId('all')
        setIsCommandPaletteOpen(false)
      },
    },
    {
      id: 'toggle_theme',
      title: `${t.commandPalette.toggleTheme} (${settings.theme === 'dark' ? 'Light' : 'Dark'})`,
      icon: settings.theme === 'dark' ? <Sun size={16} className="text-amber-400" /> : <Moon size={16} className="text-indigo-400" />,
      shortcut: 'T',
      action: () => {
        updateSettings({ theme: settings.theme === 'dark' ? 'light' : 'dark' })
        setIsCommandPaletteOpen(false)
      },
    },
    {
      id: 'toggle_lang',
      title: `${t.commandPalette.changeLanguage} (${settings.language.toUpperCase()})`,
      icon: <Globe size={16} className="text-cyan-400" />,
      shortcut: 'G',
      action: () => {
        updateSettings({ language: settings.language === 'fr' ? 'en' : 'fr' })
        setIsCommandPaletteOpen(false)
      },
    },
    {
      id: 'open_profile',
      title: t.commandPalette.openProfile,
      icon: <Settings size={16} className="text-slate-400" />,
      shortcut: 'P',
      action: () => {
        setIsCommandPaletteOpen(false)
        setIsProfileOpen(true)
      },
    },
    {
      id: 'reseed_demo',
      title: t.commandPalette.reseed,
      icon: <RotateCcw size={16} className="text-rose-400" />,
      shortcut: 'R',
      action: () => {
        reseedDemo()
        setIsCommandPaletteOpen(false)
      },
    },
  ]

  // Dynamic Skill Actions
  const skillActions = skills.map(sk => ({
    id: `skill_${sk.id}`,
    title: `⚡ Lancer ${sk.name} (${sk.command})`,
    icon: getSkillIcon(sk.icon),
    shortcut: sk.command.replace('/', ''),
    action: () => {
      setIsCommandPaletteOpen(false)
      // Pick first task matching this skill input or first active task
      const targetTask = tasks.find(t => t.status === sk.inputStatus) || tasks[0]
      if (targetTask) {
        runSkill(targetTask.id, sk.id)
        setSelectedTask(targetTask)
      }
    },
  }))

  const filteredGeneral = generalActions.filter(a =>
    a.title.toLowerCase().includes(query.toLowerCase())
  )

  const filteredSkills = skillActions.filter(s =>
    s.title.toLowerCase().includes(query.toLowerCase())
  )

  const matchingTasks = tasks.filter(t =>
    t.key.toLowerCase().includes(query.toLowerCase()) ||
    t.title.toLowerCase().includes(query.toLowerCase()) ||
    (t.description && t.description.toLowerCase().includes(query.toLowerCase())) ||
    (t.labels && t.labels.some(l => l.toLowerCase().includes(query.toLowerCase())))
  ).slice(0, 8)

  const allItems: { type: 'action' | 'skill' | 'task'; data: any }[] = [
    ...filteredGeneral.map(a => ({ type: 'action' as const, data: a })),
    ...filteredSkills.map(s => ({ type: 'skill' as const, data: s })),
    ...matchingTasks.map(tk => ({ type: 'task' as const, data: tk })),
  ]

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      setSelectedIndex(prev => (prev + 1) % Math.max(1, allItems.length))
    } else if (e.key === 'ArrowUp') {
      e.preventDefault()
      setSelectedIndex(prev => (prev - 1 + allItems.length) % Math.max(1, allItems.length))
    } else if (e.key === 'Enter') {
      e.preventDefault()
      const selected = allItems[selectedIndex]
      if (selected) {
        if (selected.type === 'action' || selected.type === 'skill') {
          selected.data.action()
        } else {
          setIsCommandPaletteOpen(false)
          setSelectedTask(selected.data)
        }
      }
    } else if (e.key === 'Escape') {
      setIsCommandPaletteOpen(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center pt-20 p-4 bg-black/60 backdrop-blur-xs animate-in fade-in duration-150">
      <div className="relative w-full max-w-xl rounded-2xl bg-[var(--bg-secondary)] border border-[var(--border-color)] shadow-2xl overflow-hidden flex flex-col max-h-[70vh]">
        {/* Search Header */}
        <div className="flex items-center gap-3 px-4 py-3.5 border-b border-[var(--border-color)] bg-[var(--bg-tertiary)]/30">
          <Search size={18} className="text-[var(--text-muted)] shrink-0" />
          <input
            ref={inputRef}
            type="text"
            value={query}
            onChange={e => {
              setQuery(e.target.value)
              setSelectedIndex(0)
            }}
            onKeyDown={handleKeyDown}
            placeholder={t.commandPalette.searchPlaceholder}
            className="w-full text-sm font-medium bg-transparent border-none text-[var(--text-primary)] placeholder-[var(--text-muted)] focus:outline-none"
          />
          <kbd className="px-2 py-0.5 text-[10px] font-mono text-[var(--text-muted)] bg-[var(--bg-secondary)] border border-[var(--border-color)] rounded">
            ESC
          </kbd>
        </div>

        {/* Results List */}
        <div className="overflow-y-auto p-2 space-y-3 flex-1">
          {allItems.length === 0 ? (
            <div className="py-8 text-center text-xs text-[var(--text-muted)]">
              {t.commandPalette.noResults} "{query}"
            </div>
          ) : (
            <>
              {/* Skills section */}
              {filteredSkills.length > 0 && (
                <div>
                  <div className="px-2 pb-1 text-[10px] font-bold uppercase tracking-wider text-amber-400 flex items-center gap-1">
                    <Sparkles size={12} />
                    {t.commandPalette.skillsSection}
                  </div>
                  <div className="space-y-0.5">
                    {filteredSkills.map((sk, idx) => {
                      const itemIdx = filteredGeneral.length + idx
                      const isSelected = selectedIndex === itemIdx
                      return (
                        <div
                          key={sk.id}
                          onClick={() => sk.action()}
                          onMouseEnter={() => setSelectedIndex(itemIdx)}
                          className={`flex items-center justify-between px-3 py-2 rounded-xl text-xs font-medium cursor-pointer transition-colors ${
                            isSelected
                              ? 'bg-[var(--accent-color)] text-white'
                              : 'text-[var(--text-primary)] hover:bg-[var(--bg-tertiary)]'
                          }`}
                        >
                          <div className="flex items-center gap-2.5">
                            <span>{sk.icon}</span>
                            <span>{sk.title}</span>
                          </div>
                          {sk.shortcut && (
                            <kbd
                              className={`px-1.5 py-0.5 text-[10px] font-mono rounded ${
                                isSelected
                                  ? 'bg-white/20 text-white'
                                  : 'bg-[var(--bg-tertiary)] text-[var(--text-muted)] border border-[var(--border-color)]'
                              }`}
                            >
                              {sk.shortcut}
                            </kbd>
                          )}
                        </div>
                      )
                    })}
                  </div>
                </div>
              )}

              {/* General Actions */}
              {filteredGeneral.length > 0 && (
                <div>
                  <div className="px-2 pb-1 text-[10px] font-bold uppercase tracking-wider text-[var(--text-muted)]">
                    {t.commandPalette.general}
                  </div>
                  <div className="space-y-0.5">
                    {filteredGeneral.map((action, idx) => {
                      const isSelected = selectedIndex === idx
                      return (
                        <div
                          key={action.id}
                          onClick={() => action.action()}
                          onMouseEnter={() => setSelectedIndex(idx)}
                          className={`flex items-center justify-between px-3 py-2 rounded-xl text-xs font-medium cursor-pointer transition-colors ${
                            isSelected
                              ? 'bg-[var(--accent-color)] text-white'
                              : 'text-[var(--text-primary)] hover:bg-[var(--bg-tertiary)]'
                          }`}
                        >
                          <div className="flex items-center gap-2.5">
                            <span>{action.icon}</span>
                            <span>{action.title}</span>
                          </div>
                          {action.shortcut && (
                            <kbd
                              className={`px-1.5 py-0.5 text-[10px] font-mono rounded ${
                                isSelected
                                  ? 'bg-white/20 text-white'
                                  : 'bg-[var(--bg-tertiary)] text-[var(--text-muted)] border border-[var(--border-color)]'
                              }`}
                            >
                              {action.shortcut}
                            </kbd>
                          )}
                        </div>
                      )
                    })}
                  </div>
                </div>
              )}

              {/* Tasks Direct Jump */}
              {matchingTasks.length > 0 && (
                <div>
                  <div className="px-2 pb-1 text-[10px] font-bold uppercase tracking-wider text-[var(--text-muted)]">
                    {t.commandPalette.tasksSection}
                  </div>
                  <div className="space-y-0.5">
                    {matchingTasks.map((task, idx) => {
                      const itemIdx = filteredGeneral.length + filteredSkills.length + idx
                      const isSelected = selectedIndex === itemIdx
                      return (
                        <div
                          key={task.id}
                          onClick={() => {
                            setIsCommandPaletteOpen(false)
                            setSelectedTask(task)
                          }}
                          onMouseEnter={() => setSelectedIndex(itemIdx)}
                          className={`flex items-center justify-between px-3 py-2 rounded-xl text-xs cursor-pointer transition-colors ${
                            isSelected
                              ? 'bg-[var(--accent-color)] text-white'
                              : 'text-[var(--text-primary)] hover:bg-[var(--bg-tertiary)]'
                          }`}
                        >
                          <div className="flex items-center gap-2.5 min-w-0">
                            <span className="font-mono text-[11px] font-bold opacity-80 shrink-0">
                              {task.key}
                            </span>
                            <span className="truncate font-medium">{task.title}</span>
                          </div>
                          <ArrowRight size={14} className="opacity-60 shrink-0 ml-2" />
                        </div>
                      )
                    })}
                  </div>
                </div>
              )}
            </>
          )}
        </div>

        {/* Footer shortcuts helper */}
        <div className="px-4 py-2 border-t border-[var(--border-color)] bg-[var(--bg-tertiary)]/20 flex items-center justify-between text-[11px] text-[var(--text-muted)]">
          <div className="flex items-center gap-3">
            <span>
              <kbd className="font-mono bg-[var(--bg-secondary)] px-1 py-0.2 rounded border border-[var(--border-color)]">↑↓</kbd>{' '}
              {t.commandPalette.hintNavigate}
            </span>
            <span>
              <kbd className="font-mono bg-[var(--bg-secondary)] px-1 py-0.2 rounded border border-[var(--border-color)]">↵</kbd>{' '}
              {t.commandPalette.hintSelect}
            </span>
          </div>
          <span>
            <kbd className="font-mono bg-[var(--bg-secondary)] px-1 py-0.2 rounded border border-[var(--border-color)]">ESC</kbd>{' '}
            {t.commandPalette.hintClose}
          </span>
        </div>
      </div>
    </div>
  )
}
