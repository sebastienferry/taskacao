import React, { createContext, useContext, useState, useEffect, useCallback, useMemo, useRef, type ReactNode } from 'react'
import type { SkillEditorEntry, TTYLaunchResult, Task, Status, Priority, UserSettings, ViewMode, BoardGroupingMode, WorkflowStage, ToastMessage, Skill, TaskActivity, ActivityStats, CliStatus, TaskSource, Project, TrackerBoard, TaskComment, TerminalSession, EpicMeta, EpicHorizon, EpicTodo, GitDiffResult, GitStatusInfo, GitBranchesInfo, DailyDigest } from '../types'
import { translations, type TranslationSchema } from '../locales/translations'
import { resolveAccentAttribute } from '../lib/accents'

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
  fetchGitStatus: (targetPathOrProject?: string, isManual?: boolean) => Promise<GitStatusInfo | null>
  gitBranches: GitBranchesInfo | null
  fetchGitBranches: (projectIdOrPath?: string) => Promise<GitBranchesInfo | null>
  switchGitBranch: (branch: string, create?: boolean, projectIdOrPath?: string) => Promise<boolean>
  isBranchModalOpen: boolean
  setIsBranchModalOpen: (open: boolean) => void
  isLoading: boolean
  isSkillRunning: boolean
  isSyncing: boolean
  runningSkillId: string | null
  error: string | null
  activeView: ViewMode
  setActiveView: (view: ViewMode) => void
  boardGrouping: BoardGroupingMode
  setBoardGrouping: (mode: BoardGroupingMode) => void
  searchQuery: string
  setSearchQuery: (query: string) => void
  statusFilter: Status | null
  setStatusFilter: (status: Status | null) => void
  priorityFilter: Priority | null
  setPriorityFilter: (priority: Priority | null) => void
  labelFilter: string | null
  taskFacets: { sprints: string[]; teams: string[] }
  sprintFilter: string | null
  setSprintFilter: (sprint: string | null) => void
  teamFilter: string | null
  setTeamFilter: (team: string | null) => void
  setLabelFilter: (label: string | null) => void
  assigneeFilter: string | null
  setAssigneeFilter: (assignee: string | null) => void
  sourceFilter: 'all' | TaskSource
  setSourceFilter: (source: 'all' | TaskSource) => void
  /** Filters the board on a parent work item key (epic, or parent story). */
  parentFilter: string | null
  setParentFilter: (parentKey: string | null) => void
  /** Distinct parents present in the loaded tasks, most populated first. */
  availableParents: { key: string; title: string; type: string; count: number }[] 
  /**
   * Resolves the display name of a workflow skill, honouring the project's
   * `skillOverrides`. Pass `projectId` to resolve against a specific project —
   * a task's project is not necessarily the one selected in the sidebar.
   */
  skillLabel: (skillId: string, fallback?: string, projectId?: string) => string
  /**
   * Resolves the slash command of a workflow skill. A project override is
   * treated as the command to invoke, normalised with a leading slash, so
   * renaming a skill also changes the command shown and run.
   */
  skillCommand: (skillId: string, fallback: string, projectId?: string) => string
  /** Docked workspace terminal on the right side of the app. */
  isTerminalPanelOpen: boolean
  setIsTerminalPanelOpen: (open: boolean) => void
  toggleTerminalPanel: () => void
  /** True when the selected project is a personal board, the only kind the digest is served for. */
  isDigestAvailable: boolean
  /** Daily digest of the active project: task sections plus an optional AI agenda. */
  dailyDigest: DailyDigest | null
  isDigestLoading: boolean
  isDigestEnriching: boolean
  fetchDailyDigest: (date?: string, assignee?: string) => Promise<DailyDigest | null>
  generateDailyDigest: (opts?: { date?: string; assignee?: string; enrich?: boolean }) => Promise<DailyDigest | null>
  sidebarCollapsed: boolean
  setSidebarCollapsed: (collapsed: boolean | ((prev: boolean) => boolean)) => void
  selectedTask: Task | null
  setSelectedTask: (task: Task | null) => void
  chatTask: Task | null
  setChatTask: (task: Task | null) => void
  /**
   * Session de terminal ouverte par son identifiant, sans passer par une tâche
   * chargée. Une exécution autonome crée sa session côté serveur, et elle doit
   * pouvoir s'ouvrir même quand la tâche est filtrée ou appartient à un autre
   * projet.
   */
  terminalSessionOverride: string | null
  openTerminalSession: (sessionId: string) => void
  openTerminalForTask: (taskId: string) => void
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
  moveTaskToTrackerStatus: (id: string, status: string) => Promise<Task | null>
  getTaskComments: (id: string) => Promise<TaskComment[]>
  listTerminalSessions: () => Promise<TerminalSession[]>
  resetTerminalSession: (sessionId: string) => Promise<void>
  postTaskComment: (id: string, body: string) => Promise<TaskComment[] | null>
  listProjectBoards: (projectId: string) => Promise<TrackerBoard[]>
  importProjectBoardColumns: (projectId: string, boardId: string) => Promise<Project | null>
  fetchProjectTrackerStatuses: (projectId: string) => Promise<string[]>
  fetchProjectEpics: (projectId: string) => Promise<EpicMeta[]>
  saveEpicMeta: (projectId: string, key: string, patch: { horizon?: EpicHorizon | ''; description?: string; todos?: EpicTodo[] }) => Promise<EpicMeta | null>
  createStoryFromEpicTodo: (projectId: string, epicKey: string, todoId: string) => Promise<{ epic: EpicMeta | null; storyKey: string } | null>
  pendingHorizonPushes: (projectId: string) => Promise<EpicMeta[]>
  pushPendingHorizons: (projectId: string) => Promise<number>
  setTaskEpic: (taskId: string, epicKey: string) => Promise<Task | null>
  createStoryUnderEpic: (projectId: string, epicKey: string, title: string) => Promise<string>
  createEpic: (projectId: string, title: string, horizon?: EpicHorizon | '') => Promise<EpicMeta | null>
  moveTasksToEpic: (projectId: string, taskIds: string[], targetEpicKey: string, newEpicTitle?: string) => Promise<string>
  advanceTask: (taskId: string, auto?: boolean) => Promise<{ mode: string; skillId?: string; label?: string } | null>
  // Pas interactif en cours : la tâche dont la session TTY attend d'être clôturée.
  pendingInteractive: { taskId: string; taskKey: string; skillId: string; label: string } | null
  /** Tickets épinglés : la barre de bascule rapide entre chantiers en cours. */
  pinnedTasks: Task[]
  isPinned: (taskId: string) => boolean
  togglePin: (taskId: string) => Promise<void>
  hotSwitch: (taskId: string) => void
  /** Éditeur de skills : les cinq pas du workflow du projet courant. */
  /** Démarre l'agent du projet dans la session d'une tâche. */
  startTaskAgent: (taskId: string, force?: boolean) => Promise<TTYLaunchResult | null>
  /** Tape l'appel d'une skill dans l'agent déjà démarré. */
  injectTaskSkill: (taskId: string, skillId: string) => Promise<TTYLaunchResult | null>
  fetchSkillEditor: () => Promise<SkillEditorEntry[]>
  saveSkillContent: (skillId: string, content: string) => Promise<SkillEditorEntry | null>
  resetSkillContent: (skillId: string) => Promise<SkillEditorEntry | null>
  importSkillFromRepo: (skillId: string) => Promise<SkillEditorEntry | null>
  launchInteractiveStep: (task: Task, skillId: string, label: string) => Promise<void>
  confirmInteractiveStep: (note?: string) => Promise<void>
  dismissInteractiveStep: () => void
  convertTask: (id: string, target: 'linear' | 'github') => Promise<Task | null>
  moveTask: (id: string, newStatus: Status, newPosition: number) => Promise<void>
  moveTaskWorkflowStage: (taskId: string, targetStage: WorkflowStage) => Promise<Task | null>
  deleteTask: (id: string) => Promise<boolean>
  runSkill: (taskId: string, skillId: string, prompt?: string) => Promise<TaskActivity | null>
  syncAll: () => Promise<void>
  syncLinear: (team?: string) => Promise<void>
  syncGithub: (repo?: string) => Promise<void>
  syncJira: (projectKey?: string) => Promise<void>
  syncCurrentProject: () => Promise<void>
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
  cleanLocalBranches: (projectIdOrPath?: string) => Promise<boolean>
  deleteGitBranch: (branch: string, deleteRemote?: boolean, projectIdOrPath?: string) => Promise<boolean>
  openInEditor: (options?: { taskId?: string; projectId?: string; path?: string; editorCommand?: string }) => Promise<boolean>
}

const defaultSettings: UserSettings = {
  id: 1,
  theme: 'dark',
  accentColor: 'orange',
  language: 'fr',
  density: 'standard',
  defaultView: 'board',
  detailMode: 'panel',
  userName: 'Developer',
  userEmail: 'dev@example.com',
  userAvatar: '',
  aiProvider: 'agy',
  aiCommandTemplate: 'agy -p "{prompt}"',
  repoPath: '',
  issueTracker: 'local',
  linearTeam: '',
  githubRepo: '',
  jiraProject: '',
  jiraUrl: '',
  specFramework: 'speckit',
  promptClarify: '',
  promptSpecify: '',
  promptImplement: '',
  promptCreatePr: '',
  promptPick: '',
  editorCommand: 'code',
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

  // The docked terminal keeps its open state and width across reloads: it is a
  // workspace tool, not a transient modal.
  const [isTerminalPanelOpen, setIsTerminalPanelOpenState] = useState<boolean>(() => {
    try {
      return localStorage.getItem('taskacao_terminal_panel_open') === 'true'
    } catch {
      return false
    }
  })

  const setIsTerminalPanelOpen = useCallback((open: boolean) => {
    setIsTerminalPanelOpenState(open)
    try {
      localStorage.setItem('taskacao_terminal_panel_open', String(open))
    } catch {
      // private mode / blocked storage: the panel just won't be remembered
    }
  }, [])

  const toggleTerminalPanel = useCallback(() => {
    setIsTerminalPanelOpenState(prev => {
      const next = !prev
      try {
        localStorage.setItem('taskacao_terminal_panel_open', String(next))
      } catch {
        // ignore
      }
      return next
    })
  }, [])
  const [boardGrouping, setBoardGroupingState] = useState<BoardGroupingMode>(() => {
    try {
      return (localStorage.getItem('taskacao_board_grouping') as BoardGroupingMode) || 'workflow'
    } catch {
      return 'workflow'
    }
  })

  const setBoardGrouping = useCallback((mode: BoardGroupingMode) => {
    setBoardGroupingState(mode)
    try {
      localStorage.setItem('taskacao_board_grouping', mode)
    } catch {}
  }, [])
  const [searchQuery, setSearchQuery] = useState('')
  const [statusFilter, setStatusFilterState] = useState<Status | null>(null)
  const [priorityFilter, setPriorityFilterState] = useState<Priority | null>(null)
  const [labelFilter, setLabelFilterState] = useState<string | null>(null)
  const [taskFacets, setTaskFacets] = useState<{ sprints: string[]; teams: string[] }>({ sprints: [], teams: [] })
  const [sprintFilter, setSprintFilterState] = useState<string | null>(null)
  const [teamFilter, setTeamFilterState] = useState<string | null>(null)
  const [assigneeFilter, setAssigneeFilterState] = useState<string | null>(null)
  const [sourceFilter, setSourceFilter] = useState<'all' | TaskSource>('all')
  const [parentFilter, setParentFilter] = useState<string | null>(null)
  const [dailyDigest, setDailyDigest] = useState<DailyDigest | null>(null)
  const [isDigestLoading, setIsDigestLoading] = useState(false)
  const [isDigestEnriching, setIsDigestEnriching] = useState(false)
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false)
  const [selectedTask, setSelectedTask] = useState<Task | null>(null)
  // chatTask désigne la tâche dont le PTY est affiché. Il vit dans le panneau
  // latéral ancré, pas dans une modale : on garde le board visible à côté du
  // terminal, et une session par tâche reste accessible d'un clic.
  const [chatTask, setChatTaskState] = useState<Task | null>(null)
  // Le pas interactif attend une confirmation humaine : la skill du dépôt ne
  // produit que du texte dans le terminal, elle ne touche jamais au ticket.
  const [pendingInteractive, setPendingInteractive] = useState<{
    taskId: string
    taskKey: string
    skillId: string
    label: string
  } | null>(null)

  const [terminalSessionOverride, setTerminalSessionOverride] = useState<string | null>(null)
  const [pinnedTasks, setPinnedTasks] = useState<Task[]>([])

  const setChatTask = useCallback((task: Task | null) => {
    setChatTaskState(task)
    if (task) {
      setTerminalSessionOverride(null)
      setIsTerminalPanelOpenState(true)
      try {
        localStorage.setItem('taskacao_terminal_panel_open', 'true')
      } catch {}
    }
  }, [])

  // Ouvre une session par son identifiant. C'est le chemin qui marche toujours :
  // il ne dépend pas de la présence de la tâche dans la liste courante.
  const openTerminalSession = useCallback((sessionId: string) => {
    setChatTaskState(null)
    setTerminalSessionOverride(sessionId)
    setIsTerminalPanelOpenState(true)
    try {
      localStorage.setItem('taskacao_terminal_panel_open', 'true')
    } catch {}
  }, [])

  const [diffTask, setDiffTask] = useState<Task | null>(null)
  const [hideDone, setHideDoneState] = useState<boolean>(() => {
    try {
      return localStorage.getItem('taskacao_hide_done') === 'true'
    } catch {
      return false
    }
  })

  const setHideDone = useCallback((val: boolean | ((prev: boolean) => boolean)) => {
    setHideDoneState(prev => {
      const next = typeof val === 'function' ? val(prev) : val
      try {
        localStorage.setItem('taskacao_hide_done', String(next))
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
      return localStorage.getItem('taskacao_selected_project_id') || 'all'
    } catch {
      return 'all'
    }
  })
  const setSelectedProjectId = useCallback((id: string | 'all') => {
    setSelectedProjectIdState(id)
    try {
      localStorage.setItem('taskacao_selected_project_id', id)
    } catch {}
  }, [])

  // Les filtres sont mémorisés par projet : sprint et équipe n'ont de sens que
  // dans le projet où ils ont été choisis, et on retrouve son contexte de
  // travail en revenant sur un projet ou après un rechargement.
  const filterStorageKey = (projectId: string) => `taskacao_filters_${projectId || 'all'}`

  const readStoredFilters = (projectId: string): Record<string, string | null> => {
    try {
      return JSON.parse(localStorage.getItem(filterStorageKey(projectId)) || '{}') || {}
    } catch {
      return {}
    }
  }

  // L'écriture se fait dans les setters et non dans un effet : au changement de
  // projet, un effet verrait encore les filtres de l'ancien projet et les
  // écrirait sous la clé du nouveau.
  const persistFilter = useCallback((patch: Record<string, string | null>) => {
    try {
      const current = readStoredFilters(selectedProjectId)
      const merged = { ...current, ...patch }
      localStorage.setItem(filterStorageKey(selectedProjectId), JSON.stringify(merged))
    } catch {
      // stockage indisponible : les filtres restent simplement non mémorisés
    }
  }, [selectedProjectId])

  const setStatusFilter = useCallback((value: Status | null) => {
    setStatusFilterState(value)
    persistFilter({ status: value })
  }, [persistFilter])

  const setPriorityFilter = useCallback((value: Priority | null) => {
    setPriorityFilterState(value)
    persistFilter({ priority: value })
  }, [persistFilter])

  const setLabelFilter = useCallback((value: string | null) => {
    setLabelFilterState(value)
    persistFilter({ label: value })
  }, [persistFilter])

  const setSprintFilter = useCallback((value: string | null) => {
    setSprintFilterState(value)
    persistFilter({ sprint: value })
  }, [persistFilter])

  const setTeamFilter = useCallback((value: string | null) => {
    setTeamFilterState(value)
    persistFilter({ team: value })
  }, [persistFilter])

  const setAssigneeFilter = useCallback((value: string | null) => {
    setAssigneeFilterState(value)
    persistFilter({ assignee: value })
  }, [persistFilter])

  const [isProjectModalOpen, setIsProjectModalOpen] = useState(false)
  const [editingProject, setEditingProject] = useState<Project | null>(null)

  const currentProject = useMemo(() => {
    if (selectedProjectId === 'all') return null
    return projects.find(p => p.id === selectedProjectId || p.slug === selectedProjectId) || null
  }, [projects, selectedProjectId])

  // Git Status & Branches State
  const [gitStatus, setGitStatus] = useState<GitStatusInfo | null>(null)
  const [gitBranches, setGitBranches] = useState<GitBranchesInfo | null>(null)
  const [isFetchingGitStatus, setIsFetchingGitStatus] = useState(false)
  const [isBranchModalOpen, setIsBranchModalOpen] = useState(false)

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

  const t = useMemo(() => translations[settings.language] || translations.fr, [settings.language])

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

    // Accent per project: the selected project's color drives the whole accent
    // scale (index.css [data-accent=…]). "All projects" keeps the brand orange.
    const accent = resolveAccentAttribute(currentProject?.color)
    body.setAttribute('data-accent', accent)
    root.setAttribute('data-accent', accent)

    const density = settings.density || 'standard'
    root.classList.remove('density-compact', 'density-standard', 'density-comfortable')
    body.classList.remove('density-compact', 'density-standard', 'density-comfortable')

    root.classList.add(`density-${density}`)
    body.classList.add(`density-${density}`)

    root.setAttribute('data-density', density)
    body.setAttribute('data-density', density)

    // Direct root font-size scaling for instantaneous global rem scaling
    if (density === 'compact') {
      root.style.fontSize = '12.5px'
    } else if (density === 'comfortable') {
      root.style.fontSize = '15.5px'
    } else {
      root.style.fontSize = '14px'
    }
  }, [settings.theme, settings.density, currentProject?.color])

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
        const projectList = data || []
        setProjects(projectList)

        // Ensure a valid project is actively selected, preserving 'all' or active project with tasks
        setSelectedProjectIdState(prev => {
          if (prev === 'all') {
            return 'all'
          }
          if (prev && projectList.some(p => p.id === prev || p.slug === prev)) {
            return prev
          }
          try {
            const stored = localStorage.getItem('taskacao_selected_project_id')
            if (stored === 'all') return 'all'
            if (stored && projectList.some(p => p.id === stored || p.slug === stored)) {
              return stored
            }
          } catch {}

          // Prioritize project with tasks or 'all'
          const projWithTasks = projectList.find(p => (p.taskCount || 0) > 0)
          if (projWithTasks) {
            try {
              localStorage.setItem('taskacao_selected_project_id', projWithTasks.id)
            } catch {}
            return projWithTasks.id
          }
          return 'all'
        })
      }
    } catch (err) {
      console.warn('Failed to load projects', err)
    }
  }, [])

  const fetchActivities = useCallback(async () => {
    try {
      const params = new URLSearchParams()
      if (selectedProjectId && selectedProjectId !== 'all') {
        params.append('projectId', selectedProjectId)
      }
      const res = await fetch(`${API_BASE}/activities?${params.toString()}`)
      if (res.ok) {
        const data: TaskActivity[] = await res.json()
        setActivities(data)
      }
    } catch (err) {
      console.warn('Failed to load activities', err)
    }
  }, [selectedProjectId])

  const fetchActivityStats = useCallback(async () => {
    try {
      const params = new URLSearchParams()
      if (selectedProjectId && selectedProjectId !== 'all') {
        params.append('projectId', selectedProjectId)
      }
      const res = await fetch(`${API_BASE}/activities/stats?${params.toString()}`)
      if (res.ok) {
        const data: ActivityStats = await res.json()
        setActivityStats(data)
      }
    } catch (err) {
      console.warn('Failed to load activity stats', err)
    }
  }, [selectedProjectId])

  const fetchGitStatus = useCallback(async (targetPathOrProject?: string, isManual = false): Promise<GitStatusInfo | null> => {
    try {
      if (isManual) setIsFetchingGitStatus(true)
      const params = new URLSearchParams()
      if (targetPathOrProject) {
        params.append('path', targetPathOrProject)
      } else if (currentProject?.repoPath) {
        params.append('path', currentProject.repoPath)
      } else if (settings.repoPath) {
        params.append('path', settings.repoPath)
      }

      const res = await fetch(`${API_BASE}/git-status?${params.toString()}`)
      if (res.ok) {
        const data: GitStatusInfo = await res.json()
        setGitStatus(prev => {
          if (
            prev &&
            prev.branch === data.branch &&
            prev.isClean === data.isClean &&
            prev.modifiedCount === data.modifiedCount &&
            prev.untrackedCount === data.untrackedCount &&
            prev.repoPath === data.repoPath
          ) {
            return prev
          }
          return data
        })
        return data
      }
      return null
    } catch (err) {
      console.warn('Failed to load git status', err)
      return null
    } finally {
      if (isManual) setIsFetchingGitStatus(false)
    }
  }, [currentProject?.repoPath, settings.repoPath])

  // Projet et filtres actifs, en un seul endroit : le rafraîchissement de fond
  // après une synchro ou une skill doit interroger exactement la même liste,
  // sinon il ramène tout le board et perd le contexte de travail.
  const buildTaskQuery = useCallback(() => {
    const params = new URLSearchParams()
    if (selectedProjectId && selectedProjectId !== 'all') {
      params.append('projectId', selectedProjectId)
    }
    if (searchQuery) params.append('q', searchQuery)
    if (statusFilter) params.append('status', statusFilter)
    if (priorityFilter) params.append('priority', priorityFilter)
    if (labelFilter) params.append('label', labelFilter)
    if (sprintFilter) params.append('sprint', sprintFilter)
    if (teamFilter) params.append('team', teamFilter)
    return params.toString()
  }, [selectedProjectId, searchQuery, statusFilter, priorityFilter, labelFilter, sprintFilter, teamFilter])

  const fetchTasks = useCallback(async () => {
    try {
      setIsLoading(true)
      const res = await fetch(`${API_BASE}/tasks?${buildTaskQuery()}`)
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const data: Task[] = await res.json()
      setTasks(data)
      setError(null)
    } catch (err: any) {
      setError(err.message || 'Failed to fetch tasks')
    } finally {
      setIsLoading(false)
    }
  }, [buildTaskQuery])

  // Restauration à l'ouverture et à chaque changement de projet. Les setters
  // bruts sont utilisés ici : réécrire ce qu'on vient de lire serait inutile.
  useEffect(() => {
    const stored = readStoredFilters(selectedProjectId)
    setStatusFilterState((stored.status as Status | null) ?? null)
    setPriorityFilterState((stored.priority as Priority | null) ?? null)
    setLabelFilterState(stored.label ?? null)
    setSprintFilterState(stored.sprint ?? null)
    setTeamFilterState(stored.team ?? null)
    setAssigneeFilterState(stored.assignee ?? null)
  }, [selectedProjectId])

  const fetchTaskFacets = useCallback(async () => {
    try {
      const params = new URLSearchParams()
      if (selectedProjectId && selectedProjectId !== 'all') {
        params.append('projectId', selectedProjectId)
      }
      const res = await fetch(`${API_BASE}/tasks/facets?${params.toString()}`)
      if (!res.ok) return
      const data = await res.json()
      setTaskFacets({ sprints: data?.sprints || [], teams: data?.teams || [] })
    } catch {
      // A tracker that feeds neither field simply leaves the filters hidden.
    }
  }, [selectedProjectId])

  useEffect(() => {
    fetchTaskFacets()
  }, [fetchTaskFacets, tasks.length])

  // Un filtre mémorisé peut ne plus exister : sprint clos, équipe renommée. Sans
  // ce garde-fou, le tableau paraîtrait vide avec un sélecteur qui n'affiche
  // rien de sélectionné.
  useEffect(() => {
    if (sprintFilter && taskFacets.sprints.length > 0 && !taskFacets.sprints.includes(sprintFilter)) {
      setSprintFilter(null)
    }
    if (teamFilter && taskFacets.teams.length > 0 && !taskFacets.teams.includes(teamFilter)) {
      setTeamFilter(null)
    }
  }, [taskFacets, sprintFilter, teamFilter, setSprintFilter, setTeamFilter])

  // Initial load on mount
  useEffect(() => {
    fetchSettings()
    fetchSkills()
    fetchCliStatus()
    fetchProjects()
  }, [fetchSettings, fetchSkills, fetchCliStatus, fetchProjects])

  // Data reload on filter / project change
  useEffect(() => {
    fetchTasks()
    fetchActivities()
    fetchActivityStats()
  }, [fetchTasks, fetchActivities, fetchActivityStats])

  // Git status reload when active repo path changes
  useEffect(() => {
    fetchGitStatus()
  }, [fetchGitStatus])

  // Active Job Count (queued or running)
  const activeJobCount = activities.filter(
    a => a.status === 'queued' || a.status === 'pending' || a.status === 'running'
  ).length

  // Smart background polling for queue execution & tasks
  useEffect(() => {
    const pollInterval = activeJobCount > 0 ? 3000 : 25000

    const interval = setInterval(async () => {
      try {
        const params = new URLSearchParams()
        if (selectedProjectId && selectedProjectId !== 'all') {
          params.append('projectId', selectedProjectId)
        }
        const [actRes, statsRes] = await Promise.all([
          fetch(`${API_BASE}/activities?${params.toString()}`),
          fetch(`${API_BASE}/activities/stats?${params.toString()}`),
        ])

        if (actRes.ok) {
          const newActivities: TaskActivity[] = await actRes.json()

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

          // Only update activities state if changed
          setActivities(prev => {
            if (
              prev.length === newActivities.length &&
              prev.every(
                (a, i) =>
                  a.id === newActivities[i]?.id &&
                  a.status === newActivities[i]?.status &&
                  a.output?.length === newActivities[i]?.output?.length
              )
            ) {
              return prev
            }
            return newActivities
          })

          if (needTaskRefresh) {
            // Même requête que le chargement normal : sans les paramètres, ce
            // rafraîchissement remplaçait la liste par tout le board, tous
            // projets et tous filtres confondus.
            const taskRes = await fetch(`${API_BASE}/tasks?${buildTaskQuery()}`)
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
          const newStats: ActivityStats = await statsRes.json()
          setActivityStats(prev => {
            if (
              prev.total === newStats.total &&
              prev.queued === newStats.queued &&
              prev.running === newStats.running &&
              prev.completed === newStats.completed &&
              prev.failed === newStats.failed &&
              prev.canceled === newStats.canceled
            ) {
              return prev
            }
            return newStats
          })
        }
      } catch (err) {
        console.warn('Queue polling error', err)
      }
    }, pollInterval)

    return () => clearInterval(interval)
  }, [activeJobCount, selectedProjectId, buildTaskQuery, t, addToast])

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
      const activeProj = selectedProjectId !== 'all' ? projects.find(p => p.id === selectedProjectId) : (projects.find(p => p.isDefault) || projects[0])
      const res = await fetch(`${API_BASE}/sync/all`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ projectId: activeProj?.id }),
      })
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
        description: activeProj ? `Projet ${activeProj.name} — Suivi dans Activités.` : 'La tâche a été ajoutée à la file d\'attente.',
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
      const activeProj = selectedProjectId !== 'all' ? projects.find(p => p.id === selectedProjectId) : (projects.find(p => p.isDefault) || projects[0])
      const targetTeam = team || activeProj?.linearTeam || settings.linearTeam || ''
      const res = await fetch(`${API_BASE}/sync/linear`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ team: targetTeam, projectId: activeProj?.id }),
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
        description: targetTeam ? `Équipe ${targetTeam} (${activeProj?.name || ''}) — Suivi en direct dans Activités.` : 'Synchronisation Linear en cours...',
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
      const activeProj = selectedProjectId !== 'all' ? projects.find(p => p.id === selectedProjectId) : (projects.find(p => p.isDefault) || projects[0])
      const targetRepo = repo || activeProj?.githubRepo || settings.githubRepo || ''
      const res = await fetch(`${API_BASE}/sync/github`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ repo: targetRepo, projectId: activeProj?.id }),
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
        description: targetRepo ? `Dépôt ${targetRepo} (${activeProj?.name || ''}) — Suivi dans Activités.` : 'Synchronisation GitHub en cours...',
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

  const syncJira = async (projectKey?: string) => {
    setIsSyncing(true)
    try {
      const activeProj = selectedProjectId !== 'all' ? projects.find(p => p.id === selectedProjectId) : (projects.find(p => p.isDefault) || projects[0])
      const targetKey = projectKey || activeProj?.jiraProject || settings.jiraProject || ''
      const res = await fetch(`${API_BASE}/sync/jira`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ projectKey: targetKey, projectId: activeProj?.id }),
      })
      if (!res.ok) {
        const errData = await res.json()
        throw new Error(errData.error || 'Jira sync failed')
      }
      const data = await res.json()
      if (data.activity) {
        setActivities(prev => [data.activity, ...prev.filter(a => a.id !== data.activity.id)])
      }
      fetchActivityStats()
      addToast({
        type: 'info',
        title: 'Synchronisation Jira lancée',
        description: targetKey ? `Projet Jira ${targetKey} (${activeProj?.name || ''}) — Suivi dans Activités.` : 'Synchronisation Jira en cours...',
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

  const syncCurrentProject = async () => {
    const activeProj = currentProject || (projects.find(p => p.isDefault) || projects[0])
    const tracker = activeProj?.issueTracker || 'local'
    if (tracker === 'linear') {
      await syncLinear(activeProj?.linearTeam)
    } else if (tracker === 'github') {
      await syncGithub(activeProj?.githubRepo)
    } else if (tracker === 'jira') {
      await syncJira(activeProj?.jiraProject)
    } else {
      await fetchTasks()
      addToast({
        type: 'info',
        title: 'Projet local à jour',
        description: `Tâches locales de ${activeProj?.name || 'ce projet'} rechargées depuis SQLite.`,
      })
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
      const defaultProj = taskData.projectId || (selectedProjectId !== 'all' ? selectedProjectId : (projects[0]?.id || 'default'))
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
      setTasks(prev => prev.map(t => (t.id === id || t.key === id || t.id === updated.id || t.key === updated.key ? updated : t)))
      if (selectedTask && (selectedTask.id === id || selectedTask.key === id || selectedTask.id === updated.id)) {
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

  // Déplacement par colonne de board : la transition dans le tracker est
  // synchrone, la carte a déjà bougé côté interface et doit revenir en place si
  // le tracker refuse.
  const moveTaskToTrackerStatus = async (id: string, status: string): Promise<Task | null> => {
    try {
      const res = await fetch(`${API_BASE}/tasks/${encodeURIComponent(id)}/tracker-status`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ status }),
      })
      if (!res.ok) {
        const errData = await res.json().catch(() => ({}))
        throw new Error(errData.error || 'Transition refusée par le tracker')
      }
      const updated: Task = await res.json()
      setTasks(prev => prev.map(t => (t.id === updated.id ? updated : t)))
      addToast({
        type: 'success',
        title: 'Ticket déplacé',
        description: `${updated.key} est passé en « ${status} »`,
      })
      return updated
    } catch (err: any) {
      addToast({
        type: 'error',
        title: 'Déplacement impossible',
        description: err.message,
      })
      return null
    }
  }

  // Les commentaires vivent dans le tracker quand il y en a un : on les relit à
  // la demande plutôt que de les recopier en base, ce qui divergerait.
  // Les sessions PTY survivent à leurs spectateurs : la liste dit ce qui tourne
  // encore et permet d'y revenir.
  const listTerminalSessions = async (): Promise<TerminalSession[]> => {
    try {
      const res = await fetch(`${API_BASE}/terminal/sessions`)
      if (!res.ok) return []
      return (await res.json()) || []
    } catch {
      return []
    }
  }

  const resetTerminalSession = async (sessionId: string): Promise<void> => {
    try {
      await fetch(`${API_BASE}/terminal/reset`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ sessionId }),
      })
      addToast({ type: 'info', title: 'Session terminée', description: sessionId })
    } catch (err: any) {
      addToast({ type: 'error', title: 'Session non terminée', description: err.message })
    }
  }

  const getTaskComments = async (id: string): Promise<TaskComment[]> => {
    try {
      const res = await fetch(`${API_BASE}/tasks/${encodeURIComponent(id)}/comments`)
      if (!res.ok) {
        const errData = await res.json().catch(() => ({}))
        throw new Error(errData.error || 'Commentaires indisponibles')
      }
      return (await res.json()) || []
    } catch (err: any) {
      addToast({ type: 'error', title: 'Commentaires', description: err.message })
      return []
    }
  }

  const postTaskComment = async (id: string, body: string): Promise<TaskComment[] | null> => {
    try {
      const res = await fetch(`${API_BASE}/tasks/${encodeURIComponent(id)}/comments`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ body }),
      })
      if (!res.ok) {
        const errData = await res.json().catch(() => ({}))
        throw new Error(errData.error || 'Publication refusée')
      }
      const comments: TaskComment[] = await res.json()
      addToast({ type: 'success', title: 'Commentaire publié' })
      return comments
    } catch (err: any) {
      addToast({ type: 'error', title: 'Commentaire non publié', description: err.message })
      return null
    }
  }

  const listProjectBoards = async (projectId: string): Promise<TrackerBoard[]> => {
    try {
      const res = await fetch(`${API_BASE}/projects/${encodeURIComponent(projectId)}/boards`)
      if (!res.ok) {
        const errData = await res.json().catch(() => ({}))
        throw new Error(errData.error || 'Boards indisponibles')
      }
      return (await res.json()) || []
    } catch (err: any) {
      addToast({ type: 'error', title: 'Boards du tracker', description: err.message })
      return []
    }
  }

  const importProjectBoardColumns = async (projectId: string, boardId: string): Promise<Project | null> => {
    try {
      const res = await fetch(`${API_BASE}/projects/${encodeURIComponent(projectId)}/board-columns`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ boardId }),
      })
      if (!res.ok) {
        const errData = await res.json().catch(() => ({}))
        throw new Error(errData.error || 'Import des colonnes impossible')
      }
      const proj: Project = await res.json()
      setProjects(prev => prev.map(p => (p.id === proj.id ? proj : p)))
      addToast({
        type: 'success',
        title: 'Colonnes importées',
        description: `${proj.trackerColumns?.length || 0} colonnes reprises du board`,
      })
      return proj
    } catch (err: any) {
      addToast({ type: 'error', title: 'Import des colonnes', description: err.message })
      return null
    }
  }

  // Méta-épics : l'horizon est une décision produit, la description et la TODO
  // du travail de cadrage. Rien de tout ça n'existe côté tracker pour un épic,
  // donc Taskacao le porte.
  const fetchProjectEpics = async (projectId: string): Promise<EpicMeta[]> => {
    try {
      const res = await fetch(`${API_BASE}/projects/${encodeURIComponent(projectId)}/epics`)
      if (!res.ok) return []
      return (await res.json()) || []
    } catch {
      return []
    }
  }

  const saveEpicMeta = async (
    projectId: string,
    key: string,
    patch: { horizon?: EpicHorizon | ''; description?: string; todos?: EpicTodo[] }
  ): Promise<EpicMeta | null> => {
    try {
      const res = await fetch(`${API_BASE}/projects/${encodeURIComponent(projectId)}/epics`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ key, ...patch }),
      })
      const data = await res.json().catch(() => ({}))
      if (!res.ok) throw new Error(data.error || 'Enregistrement refusé')
      // Pas de bandeau ici : la poussée du label part en arrière-plan et un
      // message par clic rendrait le triage insupportable. Un échec se voit
      // dans le compteur « à pousser vers Jira ».
      return data.epic || null
    } catch (err: any) {
      addToast({ type: 'error', title: 'Épic non enregistré', description: err.message })
      return null
    }
  }

  // Une ligne de TODO devient une story dans le tracker, sous son épic. La
  // tâche est insérée localement par le serveur, donc un rafraîchissement suffit
  // à la voir apparaître sur le board.
  const createStoryFromEpicTodo = async (
    projectId: string,
    epicKey: string,
    todoId: string
  ): Promise<{ epic: EpicMeta | null; storyKey: string } | null> => {
    try {
      const res = await fetch(
        `${API_BASE}/projects/${encodeURIComponent(projectId)}/epics/${encodeURIComponent(epicKey)}/story`,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ todoId }),
        }
      )
      const data = await res.json().catch(() => ({}))
      if (!res.ok) throw new Error(data.error || 'Création refusée')
      addToast({
        type: 'success',
        title: 'Story créée',
        description: `${data.storyKey} rattachée à ${epicKey}`,
      })
      fetchTasks()
      return { epic: data.epic || null, storyKey: data.storyKey || '' }
    } catch (err: any) {
      addToast({ type: 'error', title: 'Story non créée', description: err.message })
      return null
    }
  }

  // Rattrapage : les épics classés avant que le miroir en label existe, et ceux
  // dont la poussée a échoué, restent invisibles dans Jira jusqu'à ce qu'on les
  // pousse. C'est explicite, une édition de ticket par épic n'est pas anodine.
  // Rattacher un ticket existant à un épic, ou l'en détacher avec une clé vide.
  const setTaskEpic = async (taskId: string, epicKey: string): Promise<Task | null> => {
    try {
      const res = await fetch(`${API_BASE}/tasks/${encodeURIComponent(taskId)}/epic`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ epicKey }),
      })
      const data = await res.json().catch(() => ({}))
      if (!res.ok) throw new Error(data.error || 'Rattachement refusé')
      const updated: Task = data
      setTasks(prev => prev.map(t => (t.id === updated.id ? updated : t)))
      addToast({
        type: 'success',
        title: epicKey ? `${updated.key} rattaché à ${epicKey}` : `${updated.key} détaché de son épic`,
      })
      return updated
    } catch (err: any) {
      addToast({ type: 'error', title: 'Rattachement impossible', description: err.message })
      return null
    }
  }

  const createStoryUnderEpic = async (projectId: string, epicKey: string, title: string): Promise<string> => {
    try {
      const res = await fetch(
        `${API_BASE}/projects/${encodeURIComponent(projectId)}/epics/${encodeURIComponent(epicKey)}/story`,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ title }),
        }
      )
      const data = await res.json().catch(() => ({}))
      if (!res.ok) throw new Error(data.error || 'Création refusée')
      addToast({ type: 'success', title: 'Story créée', description: `${data.storyKey} sous ${epicKey}` })
      fetchTasks()
      return data.storyKey || ''
    } catch (err: any) {
      addToast({ type: 'error', title: 'Story non créée', description: err.message })
      return ''
    }
  }

  const createEpic = async (projectId: string, title: string, horizon?: EpicHorizon | ''): Promise<EpicMeta | null> => {
    try {
      const res = await fetch(`${API_BASE}/projects/${encodeURIComponent(projectId)}/epics/create`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ title, horizon: horizon || '' }),
      })
      const data = await res.json().catch(() => ({}))
      if (!res.ok) throw new Error(data.error || 'Création refusée')
      addToast({ type: 'success', title: `Épic ${data.key} créé` })
      return data
    } catch (err: any) {
      addToast({ type: 'error', title: 'Épic non créé', description: err.message })
      return null
    }
  }

  // Le geste de coupe : un lot de tickets part vers un autre épic, créé à la
  // volée si on ne donne qu'un intitulé.
  const moveTasksToEpic = async (
    projectId: string,
    taskIds: string[],
    targetEpicKey: string,
    newEpicTitle?: string
  ): Promise<string> => {
    try {
      const res = await fetch(`${API_BASE}/projects/${encodeURIComponent(projectId)}/epics/move`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ taskIds, targetEpicKey, newEpicTitle: newEpicTitle || '' }),
      })
      const data = await res.json().catch(() => ({}))
      if (!res.ok) throw new Error(data.error || 'Déplacement refusé')
      const failures: string[] = data.failures || []
      addToast({
        type: failures.length ? 'error' : 'success',
        title: `${data.moved} ticket(s) déplacé(s) vers ${data.targetEpicKey}`,
        description: failures.length ? `${failures.length} échec(s)` : undefined,
      })
      fetchTasks()
      return data.targetEpicKey || ''
    } catch (err: any) {
      addToast({ type: 'error', title: 'Déplacement impossible', description: err.message })
      return ''
    }
  }

  // Un pas du workflow, ou la chaîne autonome. Le serveur décide du pas depuis
  // l'étape de la tâche : l'interface ne fait qu'ouvrir le terminal quand le pas
  // est interactif.
  const advanceTask = async (
    taskId: string,
    auto?: boolean
  ): Promise<{ mode: string; skillId?: string; label?: string } | null> => {
    try {
      const res = await fetch(`${API_BASE}/tasks/${encodeURIComponent(taskId)}/advance`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ auto: Boolean(auto) }),
      })
      const data = await res.json().catch(() => ({}))
      if (!res.ok) throw new Error(data.error || 'Avance refusée')
      if (data.mode === 'queued') {
        addToast({
          type: 'success',
          title: 'Étape lancée',
          description: `${data.label || ''}${data.label ? '. ' : ''}Console ouvrable depuis la tâche pendant le run.`,
        })
        fetchActivities()
      } else if (data.mode === 'auto') {
        addToast({
          type: 'success',
          title: 'Chaîne autonome lancée',
          description: "L'agent s'arrêtera à l'étape de revue. Sa console est ouvrable pendant le run.",
        })
        fetchActivities()
      }
      return data
    } catch (err: any) {
      addToast({ type: 'error', title: 'Avance impossible', description: err.message })
      return null
    }
  }

  // Lance un pas interactif : ouvre le terminal de la tâche, attend que la
  // session PTY existe vraiment, puis y écrit la commande. L'attente est une
  // vraie boucle et non un délai fixe : la création du worktree peut prendre
  // plusieurs secondes, et un envoi trop tôt échouait en silence.
  const startTaskAgent = async (taskId: string, force?: boolean): Promise<TTYLaunchResult | null> => {
    try {
      const res = await fetch(`${API_BASE}/tasks/${encodeURIComponent(taskId)}/tty-agent`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ force: Boolean(force) }),
      })
      const data = await res.json().catch(() => ({}))
      if (!res.ok) throw new Error(data.error || 'Démarrage impossible')
      addToast({
        type: 'success',
        title: data.agentLaunched ? `Agent ${data.provider || ''} démarré` : `Agent ${data.provider || ''} déjà en cours`,
        description: data.launchCommand || data.cwd,
      })
      return data as TTYLaunchResult
    } catch (err: any) {
      addToast({ type: 'error', title: 'Agent non démarré', description: err.message, duration: 8000 })
      return null
    }
  }

  const injectTaskSkill = async (taskId: string, skillId: string): Promise<TTYLaunchResult | null> => {
    try {
      const res = await fetch(`${API_BASE}/tasks/${encodeURIComponent(taskId)}/tty-skill`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ skillId }),
      })
      const data = await res.json().catch(() => ({}))
      if (!res.ok) throw new Error(data.error || 'Injection impossible')
      addToast({ type: 'success', title: 'Skill lancée', description: data.call })
      return data as TTYLaunchResult
    } catch (err: any) {
      addToast({ type: 'error', title: 'Skill non lancée', description: err.message, duration: 8000 })
      return null
    }
  }

  // Un pas interactif ouvre la console de la tâche et arme la confirmation. Le
  // démarrage de l'agent et le lancement de la skill sont deux boutons de cette
  // console : enchaîner les trois tout seul supposait de deviner quand l'invite
  // de l'agent était prête, ce qui ne marchait pas d'un moteur à l'autre.
  const launchInteractiveStep = async (task: Task, skillId: string, label: string): Promise<void> => {
    setChatTask(task)
    setPendingInteractive({ taskId: task.id, taskKey: task.key, skillId, label })
    addToast({
      type: 'info',
      title: `${label} : console ouverte`,
      description: `Démarre l'agent puis lance ${skillId} depuis la barre de la console.`,
      duration: 7000,
    })
  }

  // Clôture le pas interactif : c'est ici que le ticket bouge enfin (label
  // d'étape, statut, transition sur le tracker), le serveur faisant le même
  // travail que pour une skill autonome.
  const confirmInteractiveStep = async (note?: string): Promise<void> => {
    if (!pendingInteractive) return
    const { taskId, taskKey, skillId, label } = pendingInteractive
    try {
      const res = await fetch(`${API_BASE}/tasks/${encodeURIComponent(taskId)}/advance/confirm`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ skillId, note: note || '' }),
      })
      const data = await res.json().catch(() => ({}))
      if (!res.ok) throw new Error(data.error || 'Confirmation refusée')
      setPendingInteractive(null)
      addToast({ type: 'success', title: `${label} confirmée`, description: `${taskKey} avance dans le workflow` })
      fetchTasks()
      fetchActivities()
    } catch (err: any) {
      addToast({ type: 'error', title: 'Confirmation impossible', description: err.message })
    }
  }

  // Ouvre la console d'une tâche : la tâche chargée quand elle est là, pour
  // l'étiquette et le worktree, sa session sinon.
  const openTerminalForTask = (taskId: string) => {
    const task = tasks.find(t => t.id === taskId)
    if (task) {
      setChatTask(task)
      return
    }
    openTerminalSession(`task-${taskId}`)
  }

  // Épingles. Elles vivent côté serveur : elles survivent au rechargement, et la
  // barre affiche un ticket même quand les filtres courants le cachent.
  const fetchPins = useCallback(async () => {
    try {
      const res = await fetch(`${API_BASE}/tasks/pins`)
      if (!res.ok) return
      setPinnedTasks((await res.json()) || [])
    } catch {
      // Serveur momentanément absent : la barre garde son dernier état.
    }
  }, [])

  useEffect(() => {
    fetchPins()
  }, [fetchPins])

  const isPinned = (taskId: string) => pinnedTasks.some(t => t.id === taskId)

  const togglePin = async (taskId: string) => {
    try {
      const res = await fetch(`${API_BASE}/tasks/${encodeURIComponent(taskId)}/pin`, { method: 'POST' })
      const data = await res.json().catch(() => ({}))
      if (!res.ok) throw new Error(data.error || 'Épinglage refusé')
      await fetchPins()
    } catch (err: any) {
      addToast({ type: 'error', title: 'Épinglage impossible', description: err.message })
    }
  }

  // Bascule à chaud : la console du ticket prend la main dans le panneau, et le
  // ticket devient celui sur lequel on travaille. C'est le geste qu'on répète
  // vingt fois par jour quand trois chantiers avancent en parallèle.
  const hotSwitch = (taskId: string) => {
    const task = pinnedTasks.find(t => t.id === taskId) || tasks.find(t => t.id === taskId)
    if (task) {
      setChatTask(task)
      return
    }
    openTerminalSession(`task-${taskId}`)
  }

  const dismissInteractiveStep = () => setPendingInteractive(null)

  // Éditeur de skills. Le contenu vit en base, par projet, et le serveur
  // régénère les SKILL.md du dépôt à chaque enregistrement : l'éditeur est la
  // source, les fichiers sont le produit.
  const skillEditorBase = () => {
    const pid = currentProject?.id
    if (!pid) return ''
    return `${API_BASE}/projects/${encodeURIComponent(pid)}/skill-editor`
  }

  const fetchSkillEditor = async (): Promise<SkillEditorEntry[]> => {
    const base = skillEditorBase()
    if (!base) return []
    try {
      const res = await fetch(base)
      if (!res.ok) throw new Error('Lecture des skills impossible')
      return (await res.json()) || []
    } catch (err: any) {
      addToast({ type: 'error', title: 'Skills indisponibles', description: err.message })
      return []
    }
  }

  const skillEditorAction = async (
    path: string,
    init: RequestInit,
    successTitle: string
  ): Promise<SkillEditorEntry | null> => {
    const base = skillEditorBase()
    if (!base) return null
    try {
      const res = await fetch(`${base}${path}`, init)
      const data = await res.json().catch(() => ({}))
      if (!res.ok) throw new Error(data.error || 'Action refusée')
      const entry = data as SkillEditorEntry
      addToast({
        type: 'success',
        title: successTitle,
        description: entry.paths?.length
          ? `${entry.paths.length} fichier(s) régénéré(s) dans le dépôt`
          : 'Enregistré en base, dépôt non accessible',
      })
      return entry
    } catch (err: any) {
      addToast({ type: 'error', title: 'Skill non enregistrée', description: err.message })
      return null
    }
  }

  const saveSkillContent = (skillId: string, content: string) =>
    skillEditorAction(
      `/${encodeURIComponent(skillId)}`,
      {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ content }),
      },
      'Skill enregistrée'
    )

  const resetSkillContent = (skillId: string) =>
    skillEditorAction(`/${encodeURIComponent(skillId)}/reset`, { method: 'POST' }, 'Modèle intégré restauré')

  const importSkillFromRepo = (skillId: string) =>
    skillEditorAction(`/${encodeURIComponent(skillId)}/import`, { method: 'POST' }, 'Contenu du dépôt importé')

  const pendingHorizonPushes = async (projectId: string): Promise<EpicMeta[]> => {
    try {
      const res = await fetch(`${API_BASE}/projects/${encodeURIComponent(projectId)}/epics/push-horizons`)
      if (!res.ok) return []
      return (await res.json()) || []
    } catch {
      return []
    }
  }

  const pushPendingHorizons = async (projectId: string): Promise<number> => {
    try {
      const res = await fetch(`${API_BASE}/projects/${encodeURIComponent(projectId)}/epics/push-horizons`, { method: 'POST' })
      const data = await res.json().catch(() => ({}))
      if (!res.ok) throw new Error(data.error || 'Poussée refusée')
      const failures: string[] = data.failures || []
      addToast({
        type: failures.length ? 'error' : 'success',
        title: `${data.pushed || 0} horizons poussés dans Jira`,
        description: failures.length ? `${failures.length} échec(s) : ${failures.slice(0, 2).join(' ; ')}` : undefined,
      })
      return data.pushed || 0
    } catch (err: any) {
      addToast({ type: 'error', title: 'Poussée impossible', description: err.message })
      return 0
    }
  }

  const fetchProjectTrackerStatuses = async (projectId: string): Promise<string[]> => {
    try {
      const res = await fetch(`${API_BASE}/projects/${encodeURIComponent(projectId)}/tracker-statuses`)
      if (!res.ok) return []
      return (await res.json()) || []
    } catch {
      return []
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
      const activeProj = selectedProjectId !== 'all' ? projects.find(p => p.id === selectedProjectId) : projects[0]
      const teamLabel = activeProj?.linearTeam || settings.linearTeam || 'Linear'
      const repoLabel = activeProj?.githubRepo || settings.githubRepo || 'GitHub'
      addToast({
        type: 'success',
        title: target === 'linear' ? 'Exporté vers Linear' : 'Exporté vers GitHub',
        description: `${updated.key} (${target === 'linear' ? teamLabel : repoLabel}) créé avec succès !`,
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

  const moveTaskWorkflowStage = async (taskId: string, targetStage: WorkflowStage): Promise<Task | null> => {
    const task = tasks.find(t => t.id === taskId || t.key === taskId)
    if (!task) return null

    const currentLabels = task.labels || []
    const cleanLabels = currentLabels.filter(
      l => !['untouched', 'new', 'clarified', 'specified', 'implemented', 'reviewed', 'finished', 'closed', 'New', 'Clarified', 'Specified', 'Implemented', 'Reviewed', 'Finished', 'Untouched'].some(wl => wl.toLowerCase() === l.toLowerCase())
    )
    cleanLabels.push(targetStage)

    // Determine mapped status using project stageMapping if configured
    let mappedStatus: Status = task.status
    const proj = projects.find(p => p.id === task.projectId) || currentProject
    if (proj?.stageMapping && proj.stageMapping[targetStage]) {
      mappedStatus = proj.stageMapping[targetStage] as Status
    } else {
      if (targetStage === 'new' || (targetStage as any) === 'untouched') mappedStatus = 'to_clarify'
      else if (targetStage === 'clarified') mappedStatus = 'to_specify'
      else if (targetStage === 'specified') mappedStatus = 'to_implement'
      else if (targetStage === 'implemented') mappedStatus = 'to_test'
      else if (targetStage === 'reviewed') mappedStatus = 'to_close'
      else if (targetStage === 'finished') mappedStatus = 'finished'
    }

    const updated = await updateTask(task.id, {
      labels: cleanLabels,
      status: mappedStatus,
    })

    if (updated) {
      addToast({
        type: 'info',
        title: 'Étape Workflow Agentic mise à jour',
        description: `${updated.key} ➔ #${targetStage}`,
      })
    }
    return updated
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

  // Filter tasks by active source filter (all / linear / github / jira / local)
  // then by the active parent (epic or parent story), when one is selected.
  const filteredTasks = React.useMemo(() => {
    let out = sourceFilter === 'all'
      ? tasks
      : tasks.filter(t => (t.source || 'local') === sourceFilter)
    if (parentFilter) {
      out = out.filter(t => t.parentKey === parentFilter)
    }
    return out
  }, [tasks, sourceFilter, parentFilter])

  // The daily digest reads as a brief for one person, so it is served only for
  // a selected project of type "personal" — never for a delivery project, and
  // never for the "all projects" view.
  const isDigestAvailable = currentProject?.projectType === 'personal'

  const digestProjectId = useCallback((): string | null => {
    return currentProject?.projectType === 'personal' ? currentProject.id : null
  }, [currentProject])

  // Switching to a delivery project while the digest is open would leave an
  // empty view behind: fall back to the board.
  useEffect(() => {
    if (activeView === 'digest' && !isDigestAvailable) {
      setActiveView('board')
    }
  }, [activeView, isDigestAvailable])

  const fetchDailyDigest = useCallback(async (date?: string, assignee?: string): Promise<DailyDigest | null> => {
    const pid = digestProjectId()
    if (!pid) return null
    setIsDigestLoading(true)
    try {
      const params = new URLSearchParams()
      if (date) params.set('date', date)
      if (assignee) params.set('assignee', assignee)
      const qs = params.toString() ? `?${params.toString()}` : ''
      const res = await fetch(`${API_BASE}/projects/${encodeURIComponent(pid)}/daily-digest${qs}`)
      if (!res.ok) {
        const err = await res.json().catch(() => ({}))
        throw new Error(err.error || 'Digest indisponible')
      }
      const data: DailyDigest = await res.json()
      setDailyDigest(data)
      return data
    } catch (err: any) {
      addToast({ type: 'error', title: 'Digest indisponible', description: err.message })
      return null
    } finally {
      setIsDigestLoading(false)
    }
  }, [digestProjectId])

  const generateDailyDigest = useCallback(async (
    opts?: { date?: string; assignee?: string; enrich?: boolean }
  ): Promise<DailyDigest | null> => {
    const pid = digestProjectId()
    if (!pid) return null
    const enrich = Boolean(opts?.enrich)
    if (enrich) setIsDigestEnriching(true)
    else setIsDigestLoading(true)
    try {
      const res = await fetch(`${API_BASE}/projects/${encodeURIComponent(pid)}/daily-digest`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ date: opts?.date || '', assignee: opts?.assignee || '', enrich }),
      })
      if (!res.ok) {
        const err = await res.json().catch(() => ({}))
        throw new Error(err.error || 'Génération du digest impossible')
      }
      const data: DailyDigest = await res.json()
      setDailyDigest(data)
      if (enrich) {
        addToast({
          type: data.aiStatus === 'completed' ? 'success' : 'error',
          title: data.aiStatus === 'completed' ? 'Agenda récupéré' : 'Agenda indisponible',
          description: data.aiStatus === 'completed'
            ? `Agenda du ${data.date} ajouté au digest.`
            : (data.aiError || "L'agent n'a rien renvoyé."),
        })
      }
      return data
    } catch (err: any) {
      addToast({ type: 'error', title: 'Digest', description: err.message })
      return null
    } finally {
      setIsDigestEnriching(false)
      setIsDigestLoading(false)
    }
  }, [digestProjectId])

  // A project can rename any workflow skill through `skillOverrides`
  // (skillId -> custom label). Every place that shows a skill name goes through
  // this resolver, otherwise the setting would be write-only.
  const resolveSkillOverride = useCallback(
    (skillId: string, projectId?: string): string => {
      const proj = projectId
        ? projects.find(p => p.id === projectId) || currentProject
        : currentProject
      return (proj?.skillOverrides?.[skillId] || '').trim()
    },
    [projects, currentProject]
  )

  const skillLabel = useCallback(
    (skillId: string, fallback?: string, projectId?: string): string => {
      const override = resolveSkillOverride(skillId, projectId)
      // An override written as a command ("/clarify-workitem") reads badly as a
      // label, so strip the slash for display purposes.
      if (override) return override.replace(/^\//, '')
      if (fallback && fallback.trim() !== '') return fallback
      const known = skills.find(s => s.id === skillId)
      return known?.name || skillId
    },
    [resolveSkillOverride, skills]
  )

  const skillCommand = useCallback(
    (skillId: string, fallback: string, projectId?: string): string => {
      const override = resolveSkillOverride(skillId, projectId)
      if (!override) return fallback
      // Accept both "clarify-workitem" and "/clarify-workitem".
      return '/' + override.replace(/^\//, '')
    },
    [resolveSkillOverride]
  )

  // Distinct parents across the loaded tasks, ordered by how much work hangs
  // under each one. Drives the sidebar "Epics / Parents" filter.
  const availableParents = React.useMemo(() => {
    const byKey = new Map<string, { key: string; title: string; type: string; count: number }>()
    for (const t of tasks) {
      if (!t.parentKey) continue
      const existing = byKey.get(t.parentKey)
      if (existing) {
        existing.count += 1
        if (!existing.title && t.parentTitle) existing.title = t.parentTitle
      } else {
        byKey.set(t.parentKey, {
          key: t.parentKey,
          title: t.parentTitle || '',
          type: t.parentType || '',
          count: 1,
        })
      }
    }
    return Array.from(byKey.values()).sort(
      (a, b) => b.count - a.count || a.key.localeCompare(b.key)
    )
  }, [tasks])

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

  const fetchGitBranches = useCallback(async (projectIdOrPath?: string): Promise<GitBranchesInfo | null> => {
    try {
      const target = projectIdOrPath || selectedProjectId || ''
      const res = await fetch(`${API_BASE}/git/branches?projectId=${encodeURIComponent(target)}`)
      if (!res.ok) throw new Error('Failed to fetch branches')
      const data: GitBranchesInfo = await res.json()
      setGitBranches(data)
      return data
    } catch (err) {
      console.error('Failed to fetch git branches', err)
      return null
    }
  }, [selectedProjectId])

  const switchGitBranch = useCallback(async (branch: string, create: boolean = false, projectIdOrPath?: string): Promise<boolean> => {
    try {
      const target = projectIdOrPath || selectedProjectId || ''
      const res = await fetch(`${API_BASE}/git/checkout`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          branch,
          create,
          projectId: target,
        }),
      })
      if (!res.ok) {
        const errData = await res.json()
        throw new Error(errData.error || 'Erreur lors du changement de branche')
      }
      const data = await res.json()
      if (data.status) {
        setGitStatus(data.status)
      }
      await fetchGitBranches(target)
      await fetchGitStatus()
      addToast({
        type: 'success',
        title: 'Branche Git active',
        description: data.message || `Bascule effectuée sur '${branch}'`,
      })
      return true
    } catch (err: any) {
      addToast({
        type: 'error',
        title: 'Échec de bascule de branche',
        description: err.message,
      })
      return false
    }
  }, [selectedProjectId, fetchGitBranches, fetchGitStatus, addToast])

  const cleanLocalBranches = useCallback(async (projectIdOrPath?: string): Promise<boolean> => {
    try {
      const target = projectIdOrPath || selectedProjectId || ''
      const res = await fetch(`${API_BASE}/git/branches/clean`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ projectId: target }),
      })
      if (!res.ok) {
        const errData = await res.json()
        throw new Error(errData.error || 'Erreur lors du nettoyage des branches')
      }
      const data = await res.json()
      await fetchGitBranches(target)
      await fetchGitStatus()
      addToast({
        type: 'success',
        title: 'Nettoyage des branches locales',
        description: data.message || `${data.deletedBranches?.length || 0} branches supprimées.`,
      })
      return true
    } catch (err: any) {
      addToast({
        type: 'error',
        title: 'Échec du nettoyage',
        description: err.message,
      })
      return false
    }
  }, [selectedProjectId, fetchGitBranches, fetchGitStatus, addToast])

  const deleteGitBranch = useCallback(async (branch: string, deleteRemote: boolean = false, projectIdOrPath?: string): Promise<boolean> => {
    try {
      const target = projectIdOrPath || selectedProjectId || ''
      const res = await fetch(`${API_BASE}/git/branches/delete`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          branch,
          deleteRemote,
          projectId: target,
        }),
      })
      if (!res.ok) {
        const errData = await res.json()
        throw new Error(errData.error || 'Erreur lors de la suppression de la branche')
      }
      const data = await res.json()
      await fetchGitBranches(target)
      await fetchGitStatus()
      addToast({
        type: 'success',
        title: 'Branche supprimée',
        description: data.message || `Branche '${branch}' supprimée.`,
      })
      return true
    } catch (err: any) {
      addToast({
        type: 'error',
        title: 'Échec de la suppression',
        description: err.message,
      })
      return false
    }
  }, [selectedProjectId, fetchGitBranches, fetchGitStatus, addToast])

  const openInEditor = useCallback(async (options?: { taskId?: string; projectId?: string; path?: string; editorCommand?: string }): Promise<boolean> => {
    try {
      const res = await fetch(`${API_BASE}/open-editor`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          taskId: options?.taskId,
          projectId: options?.projectId || (currentProject?.id !== 'default' ? currentProject?.id : undefined),
          path: options?.path,
          editorCommand: options?.editorCommand || settings.editorCommand || 'code',
        }),
      })
      const text = await res.text()
      let data: any = {}
      try {
        data = JSON.parse(text)
      } catch {
        if (res.status === 404) {
          throw new Error("Route /api/open-editor non trouvée (404). Veuillez relancer le serveur Go (cmd/server).")
        }
        throw new Error(text || `Erreur HTTP ${res.status}`)
      }
      if (!res.ok) {
        throw new Error(data.error || "Impossible d'ouvrir l'éditeur")
      }
      addToast({
        type: 'success',
        title: 'Éditeur ouvert',
        description: `Dossier ouvert dans ${data.editor || 'l\'éditeur'} (${data.path || ''})`,
      })
      return true
    } catch (err: any) {
      addToast({
        type: 'error',
        title: 'Erreur éditeur',
        description: err.message,
      })
      return false
    }
  }, [currentProject, settings.editorCommand, addToast])

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

      // Ctrl/Cmd + ` toggles the docked terminal, as in an IDE.
      if ((e.metaKey || e.ctrlKey) && (e.key === '`' || e.code === 'Backquote')) {
        e.preventDefault()
        toggleTerminalPanel()
        return
      }

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

      if (!isInputActive && !e.metaKey && !e.ctrlKey && !e.altKey) {
        const key = e.key.toLowerCase()
        if (key === 'b') {
          e.preventDefault()
          setActiveView('board')
          return
        }
        if (key === 'l') {
          e.preventDefault()
          setActiveView('list')
          return
        }
        if (key === 'a') {
          e.preventDefault()
          setActiveView('activities')
          return
        }
        if (key === 's' && !e.shiftKey) {
          e.preventDefault()
          setActiveView('sync')
          return
        }
        if (key === 'o') {
          e.preventDefault()
          openInEditor()
          return
        }
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
  }, [isCommandPaletteOpen, isQuickAddOpen, selectedTask, selectedActivity, isProfileOpen, searchQuery, diffTask, toggleTerminalPanel])

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
        gitBranches,
        fetchGitBranches,
        switchGitBranch,
        isBranchModalOpen,
        setIsBranchModalOpen,
        isLoading,
        isSkillRunning,
        isSyncing,
        runningSkillId,
        error,
        activeView,
        setActiveView,
        boardGrouping,
        setBoardGrouping,
        moveTaskWorkflowStage,
        searchQuery,
        setSearchQuery,
        statusFilter,
        setStatusFilter,
        priorityFilter,
        setPriorityFilter,
        labelFilter,
        taskFacets,
        sprintFilter,
        setSprintFilter,
        teamFilter,
        setTeamFilter,
        setLabelFilter,
        assigneeFilter,
        setAssigneeFilter,
        sourceFilter,
        setSourceFilter,
        parentFilter,
        setParentFilter,
        availableParents,
        skillLabel,
        skillCommand,
        isTerminalPanelOpen,
        setIsTerminalPanelOpen,
        toggleTerminalPanel,
        isDigestAvailable,
        dailyDigest,
        isDigestLoading,
        isDigestEnriching,
        fetchDailyDigest,
        generateDailyDigest,
        sidebarCollapsed,
        setSidebarCollapsed,
        selectedTask,
        setSelectedTask,
        chatTask,
        terminalSessionOverride,
        openTerminalSession,
        openTerminalForTask,
        setChatTask,
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
        moveTaskToTrackerStatus,
        getTaskComments,
        listTerminalSessions,
        resetTerminalSession,
        postTaskComment,
        listProjectBoards,
        importProjectBoardColumns,
        fetchProjectTrackerStatuses,
        fetchProjectEpics,
        saveEpicMeta,
        createStoryFromEpicTodo,
        pendingHorizonPushes,
        pushPendingHorizons,
        setTaskEpic,
        createStoryUnderEpic,
        createEpic,
        moveTasksToEpic,
        advanceTask,
        pendingInteractive,
        pinnedTasks,
        isPinned,
        togglePin,
        hotSwitch,
        startTaskAgent,
        injectTaskSkill,
        fetchSkillEditor,
        saveSkillContent,
        resetSkillContent,
        importSkillFromRepo,
        launchInteractiveStep,
        confirmInteractiveStep,
        dismissInteractiveStep,
        convertTask,
        moveTask,
        deleteTask,
        runSkill,
        syncAll,
        syncLinear,
        syncGithub,
        syncJira,
        syncCurrentProject,
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
        cleanLocalBranches,
        deleteGitBranch,
        openInEditor,
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
