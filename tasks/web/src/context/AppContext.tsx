import React, { createContext, useContext, useState, useEffect, useCallback, type ReactNode } from 'react'
import type { Task, Status, Priority, UserSettings, ViewMode, ToastMessage, Skill, TaskActivity, CliStatus, TaskSource } from '../types'
import { translations, type TranslationSchema } from '../locales/translations'

interface AppContextType {
  tasks: Task[]
  skills: Skill[]
  cliStatuses: CliStatus[]
  isLoading: boolean
  isSkillRunning: boolean
  isSyncing: boolean
  runningSkillId: string | null
  error: string | null
  activeView: ViewMode
  setActiveView: (view: ViewMode) => void
  searchQuery: string
  setSearchQuery: (query: string) => void
  statusFilter: Status | null
  setStatusFilter: (status: Status | null) => void
  priorityFilter: Priority | null
  setPriorityFilter: (priority: Priority | null) => void
  labelFilter: string | null
  setLabelFilter: (label: string | null) => void
  assigneeFilter: string | null
  setAssigneeFilter: (assignee: string | null) => void
  sourceFilter: 'all' | TaskSource
  setSourceFilter: (source: 'all' | TaskSource) => void
  sidebarCollapsed: boolean
  setSidebarCollapsed: (collapsed: boolean | ((prev: boolean) => boolean)) => void
  selectedTask: Task | null
  setSelectedTask: (task: Task | null) => void
  hideDone: boolean
  setHideDone: (hide: boolean | ((prev: boolean) => boolean)) => void
  toggleHideDone: () => void
  isQuickAddOpen: boolean
  setIsQuickAddOpen: (open: boolean) => void
  quickAddInitialStatus: Status
  setQuickAddInitialStatus: (status: Status) => void
  isCommandPaletteOpen: boolean
  setIsCommandPaletteOpen: (open: boolean) => void
  isProfileOpen: boolean
  setIsProfileOpen: (open: boolean) => void
  settings: UserSettings
  updateSettings: (newSettings: Partial<UserSettings>) => Promise<void>
  t: TranslationSchema
  toasts: ToastMessage[]
  addToast: (toast: Omit<ToastMessage, 'id'>) => void
  removeToast: (id: string) => void
  createTask: (task: { title: string; description?: string; status?: Status; priority?: Priority; labels?: string[]; assignee?: string; dueDate?: string | null; source?: TaskSource; externalUrl?: string }) => Promise<Task | null>
  updateTask: (id: string, updates: Partial<Task>) => Promise<Task | null>
  convertTask: (id: string, target: 'linear' | 'github') => Promise<Task | null>
  moveTask: (id: string, newStatus: Status, newPosition: number) => Promise<void>
  deleteTask: (id: string) => Promise<boolean>
  runSkill: (taskId: string, skillId: string, prompt?: string) => Promise<TaskActivity | null>
  syncAll: () => Promise<void>
  syncLinear: () => Promise<void>
  syncGithub: () => Promise<void>
  fetchCliStatus: () => Promise<void>
  reseedDemo: () => Promise<void>
  refreshTasks: () => Promise<void>
  availableLabels: string[]
  availableAssignees: string[]
}

const defaultSettings: UserSettings = {
  id: 1,
  theme: 'dark',
  accentColor: 'indigo',
  language: 'fr',
  density: 'standard',
  defaultView: 'board',
  detailMode: 'panel',
  userName: 'Sylvain Ferry',
  userEmail: 'sylvain@fretzee.com',
  userAvatar: '',
  aiProvider: 'agy',
  aiCommandTemplate: 'agy -p "{prompt}"',
  repoPath: '/Users/sferry/Sources/fretzee-studio',
  issueTracker: 'linear',
  linearTeam: 'FRE',
  githubRepo: 'sebastienferry/fretzee-studio',
  promptClarify: '',
  promptSpecify: '',
  promptImplement: '',
  promptCreatePr: '',
  promptPick: '',
  updatedAt: new Date().toISOString(),
}

const AppContext = createContext<AppContextType | undefined>(undefined)

const API_BASE = '/api'

export const AppProvider: React.FC<{ children: ReactNode }> = ({ children }) => {
  const [tasks, setTasks] = useState<Task[]>([])
  const [skills, setSkills] = useState<Skill[]>([])
  const [cliStatuses, setCliStatuses] = useState<CliStatus[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [isSkillRunning, setIsSkillRunning] = useState(false)
  const [isSyncing, setIsSyncing] = useState(false)
  const [runningSkillId, setRunningSkillId] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [activeView, setActiveView] = useState<ViewMode>('board')
  const [searchQuery, setSearchQuery] = useState('')
  const [statusFilter, setStatusFilter] = useState<Status | null>(null)
  const [priorityFilter, setPriorityFilter] = useState<Priority | null>(null)
  const [labelFilter, setLabelFilter] = useState<string | null>(null)
  const [assigneeFilter, setAssigneeFilter] = useState<string | null>(null)
  const [sourceFilter, setSourceFilter] = useState<'all' | TaskSource>('all')
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false)
  const [selectedTask, setSelectedTask] = useState<Task | null>(null)
  const [hideDone, setHideDoneState] = useState<boolean>(() => {
    try {
      return localStorage.getItem('fretzee_hide_done') === 'true'
    } catch {
      return false
    }
  })

  const setHideDone = useCallback((val: boolean | ((prev: boolean) => boolean)) => {
    setHideDoneState(prev => {
      const next = typeof val === 'function' ? val(prev) : val
      try {
        localStorage.setItem('fretzee_hide_done', String(next))
      } catch {}
      return next
    })
  }, [])

  const toggleHideDone = useCallback(() => {
    setHideDone(prev => !prev)
  }, [setHideDone])

  const [isQuickAddOpen, setIsQuickAddOpen] = useState(false)
  const [quickAddInitialStatus, setQuickAddInitialStatus] = useState<Status>('backlog')
  const [isCommandPaletteOpen, setIsCommandPaletteOpen] = useState(false)
  const [isProfileOpen, setIsProfileOpen] = useState(false)
  const [settings, setSettings] = useState<UserSettings>(defaultSettings)
  const [toasts, setToasts] = useState<ToastMessage[]>([])

  const addToast = useCallback((toast: Omit<ToastMessage, 'id'>) => {
    const id = Math.random().toString(36).substring(2, 9)
    const newToast: ToastMessage = { ...toast, id, duration: toast.duration || 3500 }
    setToasts(prev => [...prev, newToast])
  }, [])

  const removeToast = useCallback((id: string) => {
    setToasts(prev => prev.filter(t => t.id !== id))
  }, [])

  const t = translations[settings.language] || translations.fr

  useEffect(() => {
    const root = document.documentElement
    const body = document.body

    if (settings.theme === 'light') {
      root.classList.add('light')
      root.classList.remove('dark')
    } else {
      root.classList.add('dark')
      root.classList.remove('light')
    }

    root.classList.remove('density-compact', 'density-standard', 'density-comfortable')
    root.classList.add(`density-${settings.density}`)

    body.classList.remove('density-compact', 'density-standard', 'density-comfortable')
    body.classList.add(`density-${settings.density}`)
  }, [settings.theme, settings.accentColor, settings.density])

  const fetchSettings = useCallback(async () => {
    try {
      const res = await fetch(`${API_BASE}/settings`)
      if (res.ok) {
        const data = await res.json()
        setSettings(data)
        if (data.defaultView) {
          setActiveView(data.defaultView)
        }
      }
    } catch (err) {
      console.warn('Failed to load settings from server', err)
    }
  }, [])

  const fetchSkills = useCallback(async () => {
    try {
      const res = await fetch(`${API_BASE}/skills`)
      if (res.ok) {
        const data = await res.json()
        setSkills(data)
      }
    } catch (err) {
      console.warn('Failed to load skills from server', err)
    }
  }, [])

  const fetchCliStatus = useCallback(async () => {
    try {
      const res = await fetch(`${API_BASE}/cli-status`)
      if (res.ok) {
        const data = await res.json()
        setCliStatuses(data)
      }
    } catch (err) {
      console.warn('Failed to load CLI status', err)
    }
  }, [])

  const fetchTasks = useCallback(async () => {
    try {
      setIsLoading(true)
      const params = new URLSearchParams()
      if (searchQuery) params.append('q', searchQuery)
      if (statusFilter) params.append('status', statusFilter)
      if (priorityFilter) params.append('priority', priorityFilter)
      if (labelFilter) params.append('label', labelFilter)

      const res = await fetch(`${API_BASE}/tasks?${params.toString()}`)
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const data: Task[] = await res.json()
      setTasks(data)
      setError(null)
    } catch (err: any) {
      setError(err.message || 'Failed to fetch tasks')
    } finally {
      setIsLoading(false)
    }
  }, [searchQuery, statusFilter, priorityFilter, labelFilter])

  useEffect(() => {
    fetchSettings()
    fetchSkills()
    fetchCliStatus()
  }, [fetchSettings, fetchSkills, fetchCliStatus])

  useEffect(() => {
    fetchTasks()
  }, [fetchTasks])

  // Periodic background auto-refresh every 30 seconds
  useEffect(() => {
    const interval = setInterval(() => {
      fetch(`${API_BASE}/tasks`)
        .then(res => res.json())
        .then((data: Task[]) => {
          if (Array.isArray(data)) {
            setTasks(data)
          }
        })
        .catch(() => {})
    }, 30000)

    return () => clearInterval(interval)
  }, [])

  const updateSettings = async (newSettings: Partial<UserSettings>) => {
    const merged = { ...settings, ...newSettings }
    setSettings(merged)
    try {
      const res = await fetch(`${API_BASE}/settings`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(merged),
      })
      if (res.ok) {
        const saved = await res.json()
        setSettings(saved)
        addToast({
          type: 'success',
          title: t.toasts.settingsSaved,
        })
        fetchCliStatus()
      }
    } catch (err) {
      addToast({
        type: 'error',
        title: t.toasts.error,
        description: String(err),
      })
    }
  }

  const syncAll = async () => {
    setIsSyncing(true)
    addToast({
      type: 'info',
      title: 'Synchronisation globale en cours...',
      description: `Linear (${settings.linearTeam || 'FRE'}) + GitHub (${settings.githubRepo || 'fretzee/studio'})`,
    })

    try {
      const res = await fetch(`${API_BASE}/sync/all`, { method: 'POST' })
      if (!res.ok) {
        const errData = await res.json()
        throw new Error(errData.error || 'Global sync failed')
      }
      const data = await res.json()
      setTasks(data.tasks || [])
      addToast({
        type: 'success',
        title: t.toasts.syncSuccess,
        description: data.message,
      })
    } catch (err: any) {
      addToast({
        type: 'error',
        title: t.toasts.error,
        description: err.message,
      })
    } finally {
      setIsSyncing(false)
    }
  }

  const syncLinear = async () => {
    setIsSyncing(true)
    addToast({
      type: 'info',
      title: 'Synchronisation Linear CLI...',
      description: `Équipe: ${settings.linearTeam || 'FRE'}`,
    })

    try {
      const res = await fetch(`${API_BASE}/sync/linear`, { method: 'POST' })
      if (!res.ok) {
        const errData = await res.json()
        throw new Error(errData.error || 'Linear sync failed')
      }
      const data = await res.json()
      setTasks(data.tasks || [])
      addToast({
        type: 'success',
        title: t.toasts.syncSuccess,
        description: data.message,
      })
    } catch (err: any) {
      addToast({
        type: 'error',
        title: t.toasts.error,
        description: err.message,
      })
    } finally {
      setIsSyncing(false)
    }
  }

  const syncGithub = async () => {
    setIsSyncing(true)
    addToast({
      type: 'info',
      title: 'Synchronisation GitHub CLI...',
      description: `Repo: ${settings.githubRepo || 'fretzee/studio'}`,
    })

    try {
      const res = await fetch(`${API_BASE}/sync/github`, { method: 'POST' })
      if (!res.ok) {
        const errData = await res.json()
        throw new Error(errData.error || 'GitHub sync failed')
      }
      const data = await res.json()
      setTasks(data.tasks || [])
      addToast({
        type: 'success',
        title: t.toasts.syncSuccess,
        description: data.message,
      })
    } catch (err: any) {
      addToast({
        type: 'error',
        title: t.toasts.error,
        description: err.message,
      })
    } finally {
      setIsSyncing(false)
    }
  }

  const createTask = async (taskData: {
    title: string
    description?: string
    status?: Status
    priority?: Priority
    labels?: string[]
    assignee?: string
    dueDate?: string | null
    source?: TaskSource
    externalUrl?: string
  }): Promise<Task | null> => {
    try {
      const res = await fetch(`${API_BASE}/tasks`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(taskData),
      })
      if (!res.ok) throw new Error('Creation failed')
      const created: Task = await res.json()
      setTasks(prev => [created, ...prev])
      addToast({
        type: 'success',
        title: t.toasts.taskCreated,
        description: `${created.key}: ${created.title} (${(created.source || 'local').toUpperCase()})`,
      })
      return created
    } catch (err: any) {
      addToast({
        type: 'error',
        title: t.toasts.error,
        description: err.message,
      })
      return null
    }
  }

  const updateTask = async (id: string, updates: Partial<Task>): Promise<Task | null> => {
    try {
      const res = await fetch(`${API_BASE}/tasks/${id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(updates),
      })
      if (!res.ok) throw new Error('Update failed')
      const updated: Task = await res.json()
      setTasks(prev => prev.map(t => (t.id === id ? updated : t)))
      if (selectedTask && selectedTask.id === id) {
        setSelectedTask(updated)
      }
      addToast({
        type: 'success',
        title: t.toasts.taskUpdated,
        description: `${updated.key} ${t.toasts.taskUpdated.toLowerCase()}`,
      })
      return updated
    } catch (err: any) {
      addToast({
        type: 'error',
        title: t.toasts.error,
        description: err.message,
      })
      return null
    }
  }

  const convertTask = async (id: string, target: 'linear' | 'github'): Promise<Task | null> => {
    try {
      const res = await fetch(`${API_BASE}/tasks/${id}/convert`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ target }),
      })
      if (!res.ok) {
        const errData = await res.json()
        throw new Error(errData.error || 'Conversion failed')
      }
      const updated: Task = await res.json()
      setTasks(prev => prev.map(t => (t.id === id ? updated : t)))
      if (selectedTask && selectedTask.id === id) {
        setSelectedTask(updated)
      }
      addToast({
        type: 'success',
        title: target === 'linear' ? 'Exporté vers Linear' : 'Exporté vers GitHub',
        description: `${updated.key} (${target === 'linear' ? 'Linear ' + (settings.linearTeam || 'FRE') : 'GitHub ' + (settings.githubRepo || '')}) créé avec succès !`,
      })
      return updated
    } catch (err: any) {
      addToast({
        type: 'error',
        title: 'Erreur d\'export',
        description: err.message,
      })
      return null
    }
  }


  const moveTask = async (id: string, newStatus: Status, newPosition: number) => {
    setTasks(prev => {
      return prev.map(t => {
        if (t.id === id) {
          return { ...t, status: newStatus, position: newPosition }
        }
        return t
      })
    })

    try {
      const res = await fetch(`${API_BASE}/tasks/${id}/move`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ status: newStatus, position: newPosition }),
      })
      if (!res.ok) throw new Error('Move failed')
      const updated: Task = await res.json()
      setTasks(prev => prev.map(t => (t.id === id ? updated : t)))
      addToast({
        type: 'info',
        title: t.toasts.taskMoved,
        description: `${updated.key} ➔ ${t.status[newStatus]}`,
      })
    } catch (err: any) {
      fetchTasks()
      addToast({
        type: 'error',
        title: t.toasts.error,
        description: err.message,
      })
    }
  }

  const runSkill = async (taskId: string, skillId: string, prompt?: string): Promise<TaskActivity | null> => {
    setIsSkillRunning(true)
    setRunningSkillId(skillId)
    addToast({
      type: 'info',
      title: t.toasts.skillStarted,
      description: `Moteur: ${settings.aiProvider.toUpperCase()} (${skillId})`,
    })

    try {
      const res = await fetch(`${API_BASE}/tasks/${taskId}/run-skill`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ skillId, prompt }),
      })
      if (!res.ok) {
        const errorData = await res.json()
        throw new Error(errorData.error || 'Skill execution failed')
      }

      const data = await res.json()
      const updatedTask: Task = data.task
      const activity: TaskActivity = data.activity

      setTasks(prev => prev.map(t => (t.id === taskId ? updatedTask : t)))
      if (selectedTask && selectedTask.id === taskId) {
        setSelectedTask(updatedTask)
      }

      addToast({
        type: 'success',
        title: t.skills.skillSuccess,
        description: `${updatedTask.key} ➔ ${t.status[updatedTask.status]}`,
      })

      return activity
    } catch (err: any) {
      addToast({
        type: 'error',
        title: t.toasts.error,
        description: err.message,
      })
      return null
    } finally {
      setIsSkillRunning(false)
      setRunningSkillId(null)
    }
  }

  const deleteTask = async (id: string): Promise<boolean> => {
    try {
      const res = await fetch(`${API_BASE}/tasks/${id}`, {
        method: 'DELETE',
      })
      if (!res.ok) throw new Error('Delete failed')
      setTasks(prev => prev.filter(t => t.id !== id))
      if (selectedTask && selectedTask.id === id) {
        setSelectedTask(null)
      }
      addToast({
        type: 'warning',
        title: t.toasts.taskDeleted,
      })
      return true
    } catch (err: any) {
      addToast({
        type: 'error',
        title: t.toasts.error,
        description: err.message,
      })
      return false
    }
  }

  const reseedDemo = async () => {
    try {
      setIsLoading(true)
      const res = await fetch(`${API_BASE}/seed`, { method: 'POST' })
      if (!res.ok) throw new Error('Reseed failed')
      const data = await res.json()
      setTasks(data.tasks || [])
      addToast({
        type: 'success',
        title: t.toasts.demoReseeded,
      })
    } catch (err: any) {
      addToast({
        type: 'error',
        title: t.toasts.error,
        description: err.message,
      })
    } finally {
      setIsLoading(false)
    }
  }

  // Filter tasks by active source filter (all / linear / github / local)
  const filteredTasks = sourceFilter === 'all' 
    ? tasks 
    : tasks.filter(t => (t.source || 'local') === sourceFilter)

  const availableLabels = Array.from(
    new Set(tasks.flatMap(t => t.labels || []).filter(Boolean))
  )

  const availableAssignees = Array.from(
    new Set(tasks.map(t => t.assignee).filter(Boolean))
  )

  // Global Keyboard Shortcuts
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault()
        setIsCommandPaletteOpen(prev => !prev)
        return
      }

      const activeTag = (document.activeElement?.tagName || '').toLowerCase()
      const isInputActive = activeTag === 'input' || activeTag === 'textarea' || activeTag === 'select'

      if (e.key === '/' && !isInputActive) {
        e.preventDefault()
        const searchInput = document.getElementById('global-search-input') as HTMLInputElement
        if (searchInput) {
          searchInput.focus()
          searchInput.select()
        }
        return
      }

      if ((e.key === 'n' || e.key === 'N' || e.key === 'c' || e.key === 'C') && !isInputActive && !e.metaKey && !e.ctrlKey) {
        e.preventDefault()
        setIsQuickAddOpen(true)
        return
      }

      if (e.key === 'Escape') {
        if (isCommandPaletteOpen) {
          setIsCommandPaletteOpen(false)
        } else if (isQuickAddOpen) {
          setIsQuickAddOpen(false)
        } else if (selectedTask) {
          setSelectedTask(null)
        } else if (isProfileOpen) {
          setIsProfileOpen(false)
        } else if (searchQuery) {
          setSearchQuery('')
        }
      }
    }

    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [isCommandPaletteOpen, isQuickAddOpen, selectedTask, isProfileOpen, searchQuery])

  return (
    <AppContext.Provider
      value={{
        tasks: filteredTasks,
        skills,
        cliStatuses,
        isLoading,
        isSkillRunning,
        isSyncing,
        runningSkillId,
        error,
        activeView,
        setActiveView,
        searchQuery,
        setSearchQuery,
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
        selectedTask,
        setSelectedTask,
        hideDone,
        setHideDone,
        toggleHideDone,
        isQuickAddOpen,
        setIsQuickAddOpen,
        quickAddInitialStatus,
        setQuickAddInitialStatus,
        isCommandPaletteOpen,
        setIsCommandPaletteOpen,
        isProfileOpen,
        setIsProfileOpen,
        settings,
        updateSettings,
        t,
        toasts,
        addToast,
        removeToast,
        createTask,
        updateTask,
        convertTask,
        moveTask,
        deleteTask,
        runSkill,
        syncAll,
        syncLinear,
        syncGithub,
        fetchCliStatus,
        reseedDemo,
        refreshTasks: fetchTasks,
        availableLabels,
        availableAssignees,
      }}
    >
      {children}
    </AppContext.Provider>
  )
}

export const useApp = () => {
  const context = useContext(AppContext)
  if (!context) {
    throw new Error('useApp must be used within an AppProvider')
  }
  return context
}
