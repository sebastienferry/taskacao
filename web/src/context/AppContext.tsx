import React, { createContext, useContext, useState, useEffect, useCallback, useMemo, useRef, type ReactNode } from 'react'
import type { Task, Status, Priority, UserSettings, ViewMode, ToastMessage, Skill, TaskActivity, ActivityStats, CliStatus, TaskSource, Project, GitDiffResult, GitStatusInfo } from '../types'
import { translations, type TranslationSchema } from '../locales/translations'

interface AppContextType {
  projects: Project[]
  selectedProjectId: string | 'all'
  setSelectedProjectId: (id: string | 'all') => void
  currentProject: Project | null
  createProject: (data: Partial<Project>) => Promise<Project | null>
  updateProject: (id: string, updates: Partial<Project>) => Promise<Project | null>
  deleteProject: (id: string) => Promise<boolean>
  fetchProjects: () => Promise<void>
  isProjectModalOpen: boolean
  setIsProjectModalOpen: (open: boolean) => void
  editingProject: Project | null
  setEditingProject: (p: Project | null) => void
  tasks: Task[]
  skills: Skill[]
  cliStatuses: CliStatus[]
  gitStatus: GitStatusInfo | null
  isFetchingGitStatus: boolean
  fetchGitStatus: (targetPathOrProject?: string) => Promise<GitStatusInfo | null>
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
  createTask: (task: { title: string; description?: string; status?: Status; priority?: Priority; labels?: string[]; assignee?: string; dueDate?: string | null; source?: TaskSource; externalUrl?: string; projectId?: string }) => Promise<Task | null>
  updateTask: (id: string, updates: Partial<Task>) => Promise<Task | null>
  convertTask: (id: string, target: 'linear' | 'github') => Promise<Task | null>
  moveTask: (id: string, newStatus: Status, newPosition: number) => Promise<void>
  deleteTask: (id: string) => Promise<boolean>
  runSkill: (taskId: string, skillId: string, prompt?: string) => Promise<TaskActivity | null>
  syncAll: () => Promise<void>
  syncLinear: (team?: string) => Promise<void>
  syncGithub: (repo?: string) => Promise<void>
  fetchCliStatus: () => Promise<void>
  reseedDemo: () => Promise<void>
  refreshTasks: () => Promise<void>
  activities: TaskActivity[]
  activityStats: ActivityStats
  selectedActivity: TaskActivity | null
  setSelectedActivity: (activity: TaskActivity | null) => void
  activeJobCount: number
  fetchActivities: () => Promise<void>
  fetchActivityStats: () => Promise<void>
  retryActivity: (id: string) => Promise<void>
  cancelActivity: (id: string) => Promise<void>
  deleteActivity: (id: string) => Promise<void>
  clearCompletedActivities: () => Promise<void>
  availableLabels: string[]
  availableAssignees: string[]
  diffTask: Task | null
  setDiffTask: (task: Task | null) => void
  fetchGitDiff: (taskId: string) => Promise<GitDiffResult | null>
  checkoutTaskBranch: (taskId: string) => Promise<boolean>
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
  const [diffTask, setDiffTask] = useState<Task | null>(null)
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

  // Projects State
  const [projects, setProjects] = useState<Project[]>([])
  const [selectedProjectId, setSelectedProjectIdState] = useState<string | 'all'>(() => {
    try {
      return localStorage.getItem('fretzee_selected_project_id') || 'all'
    } catch {
      return 'all'
    }
  })
  const setSelectedProjectId = useCallback((id: string | 'all') => {
    setSelectedProjectIdState(id)
    try {
      localStorage.setItem('fretzee_selected_project_id', id)
    } catch {}
  }, [])
  const [isProjectModalOpen, setIsProjectModalOpen] = useState(false)
  const [editingProject, setEditingProject] = useState<Project | null>(null)

  const currentProject = useMemo(() => {
    if (selectedProjectId === 'all') return null
    return projects.find(p => p.id === selectedProjectId || p.slug === selectedProjectId) || null
  }, [projects, selectedProjectId])

  // Git Status State
  const [gitStatus, setGitStatus] = useState<GitStatusInfo | null>(null)
  const [isFetchingGitStatus, setIsFetchingGitStatus] = useState(false)

  // Activities & Queue State
  const [activities, setActivities] = useState<TaskActivity[]>([])
  const [activityStats, setActivityStats] = useState<ActivityStats>({
    total: 0,
    queued: 0,
    running: 0,
    completed: 0,
    failed: 0,
    canceled: 0,
  })
  const [selectedActivity, setSelectedActivity] = useState<TaskActivity | null>(null)
  const prevActiveActivitiesRef = useRef<Map<string, string>>(new Map())

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

    body.setAttribute('data-accent', settings.accentColor || 'indigo')
    body.setAttribute('data-density', settings.density || 'standard')
  }, [settings.theme, settings.accentColor, settings.density])

  const fetchSettings = useCallback(async () => {
    try {
      const res = await fetch(`${API_BASE}/settings`)
      if (res.ok) {
        const data: UserSettings = await res.json()
        setSettings(data)
      }
    } catch (err) {
      console.warn('Failed to load settings from server', err)
    }
  }, [])

  const fetchSkills = useCallback(async () => {
    try {
      const res = await fetch(`${API_BASE}/skills`)
      if (res.ok) {
        const data: Skill[] = await res.json()
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
        const data: CliStatus[] = await res.json()
        setCliStatuses(data)
      }
    } catch (err) {
      console.warn('Failed to load CLI statuses', err)
    }
  }, [])

  const fetchProjects = useCallback(async () => {
    try {
      const res = await fetch(`${API_BASE}/projects`)
      if (res.ok) {
        const data: Project[] = await res.json()
        setProjects(data || [])
      }
    } catch (err) {
      console.warn('Failed to load projects', err)
    }
  }, [])

  const fetchActivities = useCallback(async () => {
    try {
      const res = await fetch(`${API_BASE}/activities`)
      if (res.ok) {
        const data: TaskActivity[] = await res.json()
        setActivities(data)
      }
    } catch (err) {
      console.warn('Failed to load activities', err)
    }
  }, [])

  const fetchActivityStats = useCallback(async () => {
    try {
      const res = await fetch(`${API_BASE}/activities/stats`)
      if (res.ok) {
        const data: ActivityStats = await res.json()
        setActivityStats(data)
      }
    } catch (err) {
      console.warn('Failed to load activity stats', err)
    }
  }, [])

  const fetchGitStatus = useCallback(async (targetPathOrProject?: string): Promise<GitStatusInfo | null> => {
    try {
      setIsFetchingGitStatus(true)
      const params = new URLSearchParams()
      if (targetPathOrProject) {
        params.append('path', targetPathOrProject)
      } else if (currentProject && currentProject.repoPath) {
        params.append('path', currentProject.repoPath)
      } else if (settings.repoPath) {
        params.append('path', settings.repoPath)
      }

      const res = await fetch(`${API_BASE}/git-status?${params.toString()}`)
      if (res.ok) {
        const data: GitStatusInfo = await res.json()
        setGitStatus(data)
        return data
      }
      return null
    } catch (err) {
      console.warn('Failed to load git status', err)
      return null
    } finally {
      setIsFetchingGitStatus(false)
    }
  }, [currentProject, settings.repoPath])

  const fetchTasks = useCallback(async () => {
    try {
      setIsLoading(true)
      const params = new URLSearchParams()
      if (selectedProjectId && selectedProjectId !== 'all') {
        params.append('projectId', selectedProjectId)
      }
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
  }, [searchQuery, statusFilter, priorityFilter, labelFilter, selectedProjectId])

  useEffect(() => {
    fetchSettings()
    fetchSkills()
    fetchCliStatus()
    fetchProjects()
    fetchActivities()
    fetchActivityStats()
    fetchGitStatus()
  }, [fetchSettings, fetchSkills, fetchCliStatus, fetchProjects, fetchActivities, fetchActivityStats, fetchGitStatus])

  useEffect(() => {
    fetchTasks()
  }, [fetchTasks])

  // Active Job Count (queued or running)
  const activeJobCount = activities.filter(
    a => a.status === 'queued' || a.status === 'pending' || a.status === 'running'
  ).length

  // Smart background polling for queue execution & tasks
  useEffect(() => {
    const pollInterval = activeJobCount > 0 ? 2500 : 20000

    const interval = setInterval(async () => {
      try {
        const [actRes, statsRes] = await Promise.all([
          fetch(`${API_BASE}/activities`),
          fetch(`${API_BASE}/activities/stats`),
        ])

        if (actRes.ok) {
          const newActivities: TaskActivity[] = await actRes.json()
          setActivities(newActivities)

          // Detect finished activities
          const prevMap = prevActiveActivitiesRef.current
          let needTaskRefresh = false

          newActivities.forEach(act => {
            const prevStatus = prevMap.get(act.id)
            if (prevStatus && (prevStatus === 'queued' || prevStatus === 'pending' || prevStatus === 'running')) {
              if (act.status === 'completed') {
                needTaskRefresh = true
                addToast({
                  type: 'success',
                  title: t.toasts.skillCompleted,
                  description: `${act.skillName} (${act.taskKey || 'Tâche'}) terminée avec succès !`,
                })
              } else if (act.status === 'failed') {
                needTaskRefresh = true
                addToast({
                  type: 'error',
                  title: t.toasts.error,
                  description: `Échec de ${act.skillName} (${act.taskKey || 'Tâche'})`,
                })
              }
            }
          })

          // Update tracking map
          const nextMap = new Map<string, string>()
          newActivities.forEach(a => nextMap.set(a.id, a.status))
          prevActiveActivitiesRef.current = nextMap

          if (needTaskRefresh) {
            const taskRes = await fetch(`${API_BASE}/tasks`)
            if (taskRes.ok) {
              const freshTasks = await taskRes.json()
              setTasks(freshTasks)
              setSelectedTask(curr => {
                if (!curr) return null
                return freshTasks.find((t: Task) => t.id === curr.id) || curr
              })
            }
          }
        }

        if (statsRes.ok) {
          const newStats = await statsRes.json()
          setActivityStats(newStats)
        }
      } catch (err) {
        console.warn('Queue polling error', err)
      }
    }, pollInterval)

    return () => clearInterval(interval)
  }, [activeJobCount, t, addToast])

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
    try {
      const res = await fetch(`${API_BASE}/sync/all`, { method: 'POST' })
      if (!res.ok) {
        const errData = await res.json()
        throw new Error(errData.error || 'Global sync failed')
      }
      const data = await res.json()
      if (data.activity) {
        setActivities(prev => [data.activity, ...prev.filter(a => a.id !== data.activity.id)])
      }
      fetchActivityStats()
      addToast({
        type: 'info',
        title: 'Synchronisation globale lancée',
        description: 'La tâche a été ajoutée à la file d\'attente (consultable dans Activités).',
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

  const syncLinear = async (team?: string) => {
    setIsSyncing(true)
    try {
      const res = await fetch(`${API_BASE}/sync/linear`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ team: team || settings.linearTeam || 'FRE' }),
      })
      if (!res.ok) {
        const errData = await res.json()
        throw new Error(errData.error || 'Linear sync failed')
      }
      const data = await res.json()
      if (data.activity) {
        setActivities(prev => [data.activity, ...prev.filter(a => a.id !== data.activity.id)])
      }
      fetchActivityStats()
      addToast({
        type: 'info',
        title: 'Synchronisation Linear lancée',
        description: `Équipe ${team || settings.linearTeam || 'FRE'} — Suivi en direct dans Activités.`,
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

  const syncGithub = async (repo?: string) => {
    setIsSyncing(true)
    try {
      const res = await fetch(`${API_BASE}/sync/github`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ repo: repo || settings.githubRepo || 'sebastienferry/fretzee-studio' }),
      })
      if (!res.ok) {
        const errData = await res.json()
        throw new Error(errData.error || 'GitHub sync failed')
      }
      const data = await res.json()
      if (data.activity) {
        setActivities(prev => [data.activity, ...prev.filter(a => a.id !== data.activity.id)])
      }
      fetchActivityStats()
      addToast({
        type: 'info',
        title: 'Synchronisation GitHub lancée',
        description: `Repo ${repo || settings.githubRepo || 'sebastienferry/fretzee-studio'} — Suivi dans Activités.`,
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

  const createProject = async (data: Partial<Project>): Promise<Project | null> => {
    try {
      const res = await fetch(`${API_BASE}/projects`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data),
      })
      if (!res.ok) {
        const errData = await res.json()
        throw new Error(errData.error || 'Failed to create project')
      }
      const created: Project = await res.json()
      await fetchProjects()
      setSelectedProjectId(created.id)
      addToast({
        type: 'success',
        title: 'Projet créé',
        description: `Le projet ${created.name} a été créé avec succès.`,
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

  const updateProject = async (id: string, updates: Partial<Project>): Promise<Project | null> => {
    try {
      const res = await fetch(`${API_BASE}/projects/${encodeURIComponent(id)}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(updates),
      })
      if (!res.ok) {
        const errData = await res.json()
        throw new Error(errData.error || 'Failed to update project')
      }
      const updated: Project = await res.json()
      await fetchProjects()
      addToast({
        type: 'success',
        title: 'Projet mis à jour',
        description: `Le projet ${updated.name} a été actualisé.`,
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

  const deleteProject = async (id: string): Promise<boolean> => {
    try {
      const res = await fetch(`${API_BASE}/projects/${encodeURIComponent(id)}`, {
        method: 'DELETE',
      })
      if (!res.ok) {
        const errData = await res.json()
        throw new Error(errData.error || 'Failed to delete project')
      }
      if (selectedProjectId === id) {
        setSelectedProjectId('all')
      }
      await fetchProjects()
      await fetchTasks()
      addToast({
        type: 'warning',
        title: 'Projet supprimé',
        description: 'Les tâches ont été réassignées au projet principal.',
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
    projectId?: string
  }): Promise<Task | null> => {
    try {
      const defaultProj = taskData.projectId || (selectedProjectId !== 'all' ? selectedProjectId : (projects[0]?.id || 'fretzee-studio'))
      const res = await fetch(`${API_BASE}/tasks`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          ...taskData,
          projectId: defaultProj,
        }),
      })
      if (!res.ok) throw new Error('Creation failed')
      const created: Task = await res.json()
      setTasks(prev => [created, ...prev])
      fetchProjects()
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
      const res = await fetch(`${API_BASE}/tasks/${encodeURIComponent(id)}`, {
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
      const res = await fetch(`${API_BASE}/tasks/${encodeURIComponent(id)}/convert`, {
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
      const res = await fetch(`${API_BASE}/tasks/${encodeURIComponent(id)}/move`, {
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
      title: t.toasts.skillQueued,
      description: `Moteur: ${settings.aiProvider.toUpperCase()} (${skillId}) - Poussée en file d'attente`,
    })

    try {
      const res = await fetch(`${API_BASE}/tasks/${encodeURIComponent(taskId)}/run-skill`, {
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
      setActivities(prev => [activity, ...prev.filter(a => a.id !== activity.id)])
      await fetchActivityStats()

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

  const retryActivity = async (id: string) => {
    try {
      const res = await fetch(`${API_BASE}/activities/${id}/retry`, { method: 'POST' })
      if (!res.ok) throw new Error('Retry failed')
      addToast({
        type: 'info',
        title: t.toasts.activityRetried,
      })
      await fetchActivities()
      await fetchActivityStats()
    } catch (err: any) {
      addToast({
        type: 'error',
        title: t.toasts.error,
        description: err.message,
      })
    }
  }

  const cancelActivity = async (id: string) => {
    try {
      const res = await fetch(`${API_BASE}/activities/${id}/cancel`, { method: 'POST' })
      if (!res.ok) throw new Error('Cancel failed')
      addToast({
        type: 'warning',
        title: t.toasts.activityCanceled,
      })
      await fetchActivities()
      await fetchActivityStats()
    } catch (err: any) {
      addToast({
        type: 'error',
        title: t.toasts.error,
        description: err.message,
      })
    }
  }

  const deleteActivity = async (id: string) => {
    try {
      const res = await fetch(`${API_BASE}/activities/${id}`, { method: 'DELETE' })
      if (!res.ok) throw new Error('Delete failed')
      setActivities(prev => prev.filter(a => a.id !== id))
      if (selectedActivity && selectedActivity.id === id) {
        setSelectedActivity(null)
      }
      addToast({
        type: 'warning',
        title: t.toasts.activityDeleted,
      })
      await fetchActivityStats()
    } catch (err: any) {
      addToast({
        type: 'error',
        title: t.toasts.error,
        description: err.message,
      })
    }
  }

  const clearCompletedActivities = async () => {
    try {
      const res = await fetch(`${API_BASE}/activities`, { method: 'DELETE' })
      if (!res.ok) throw new Error('Clear failed')
      addToast({
        type: 'info',
        title: t.toasts.activitiesCleared,
      })
      await fetchActivities()
      await fetchActivityStats()
    } catch (err: any) {
      addToast({
        type: 'error',
        title: t.toasts.error,
        description: err.message,
      })
    }
  }

  const deleteTask = async (id: string): Promise<boolean> => {
    try {
      const res = await fetch(`${API_BASE}/tasks/${encodeURIComponent(id)}`, {
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

  const fetchGitDiff = useCallback(async (taskId: string): Promise<GitDiffResult | null> => {
    try {
      const res = await fetch(`${API_BASE}/tasks/${encodeURIComponent(taskId)}/git-diff`)
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const data: GitDiffResult = await res.json()
      return data
    } catch (err: any) {
      console.error('Failed to fetch git diff', err)
      return null
    }
  }, [])

  const checkoutTaskBranch = useCallback(async (taskId: string): Promise<boolean> => {
    try {
      const res = await fetch(`${API_BASE}/tasks/${encodeURIComponent(taskId)}/checkout-branch`, {
        method: 'POST',
      })
      if (!res.ok) {
        const errData = await res.json()
        throw new Error(errData.error || 'Failed to switch branch')
      }
      const data = await res.json()
      addToast({
        type: 'success',
        title: 'Branche Git active',
        description: data.message || `Bascule effectuée sur ${data.branch}`,
      })
      await fetchTasks()
      return true
    } catch (err: any) {
      addToast({
        type: 'error',
        title: 'Erreur Git checkout',
        description: err.message,
      })
      return false
    }
  }, [addToast, fetchTasks])

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
        if (diffTask) {
          setDiffTask(null)
        } else if (isCommandPaletteOpen) {
          setIsCommandPaletteOpen(false)
        } else if (isQuickAddOpen) {
          setIsQuickAddOpen(false)
        } else if (selectedTask) {
          setSelectedTask(null)
        } else if (selectedActivity) {
          setSelectedActivity(null)
        } else if (isProfileOpen) {
          setIsProfileOpen(false)
        } else if (searchQuery) {
          setSearchQuery('')
        }
      }
    }

    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [isCommandPaletteOpen, isQuickAddOpen, selectedTask, selectedActivity, isProfileOpen, searchQuery, diffTask])

  return (
    <AppContext.Provider
      value={{
        projects,
        selectedProjectId,
        setSelectedProjectId,
        currentProject,
        createProject,
        updateProject,
        deleteProject,
        fetchProjects,
        isProjectModalOpen,
        setIsProjectModalOpen,
        editingProject,
        setEditingProject,
        tasks: filteredTasks,
        skills,
        cliStatuses,
        gitStatus,
        isFetchingGitStatus,
        fetchGitStatus,
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
        diffTask,
        setDiffTask,
        fetchGitDiff,
        checkoutTaskBranch,
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
        activities,
        activityStats,
        selectedActivity,
        setSelectedActivity,
        activeJobCount,
        fetchActivities,
        fetchActivityStats,
        retryActivity,
        cancelActivity,
        deleteActivity,
        clearCompletedActivities,
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
