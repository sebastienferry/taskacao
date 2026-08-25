export type Priority = 'urgent' | 'high' | 'medium' | 'low'

export type Status = 
  | 'to_clarify'    // A clarifier (Label: #new)
  | 'to_specify'    // A spécifier (Label: #clarified)
  | 'to_implement'  // A implémenter (Label: #specified)
  | 'to_test'       // A tester (Label: #implemented)
  | 'to_close'      // En revue / PR (Label: #reviewed)
  | 'finished'      // Terminé (Label: #finished)
  | 'backlog'
  | 'specified'
  | 'in_progress'
  | 'to_validate'
  | 'done'

export type TaskSource = 'linear' | 'github' | 'jira' | 'local'

export type ActivityStatus = 'queued' | 'pending' | 'running' | 'completed' | 'failed' | 'canceled'

export interface TaskActivity {
  id: string
  taskId: string
  projectId?: string
  taskKey?: string
  taskTitle?: string
  skillId: string
  skillName: string
  action: string
  status: ActivityStatus
  summary: string
  output: string
  steps: string[]
  prompt?: string
  createdAt: string
  startedAt?: string
  completedAt?: string
  error?: string
  duration?: string
}

export interface ActivityStats {
  total: number
  queued: number
  running: number
  completed: number
  failed: number
  canceled: number
}

export interface TrackerColumn {
  name: string
  statuses: string[]
  /** Colonne retirée du board sans perdre son affectation de statuts. */
  hidden?: boolean
}

export interface TerminalSession {
  id: string
  cwd: string
  clients: number
  createdAt: string
  lastActiveAt: string
  historyBytes: number
}

export interface TaskComment {
  id: string
  taskId?: string
  author: string
  body: string
  createdAt?: string
  source: string
}

export interface TrackerSprint {
  name: string
  /** « active », « future » ou « closed » : ce qui sépare NOW de NEXT. */
  state: string
}

export type EpicHorizon = 'now' | 'next' | 'later'

export interface EpicTodo {
  id: string
  text: string
  done: boolean
  /** Ticket créé depuis cette ligne de TODO, s'il existe. */
  storyKey?: string
}

export interface EpicMeta {
  projectId: string
  key: string
  /** Chaîne vide = épic non encore classé. */
  horizon: EpicHorizon | ''
  description: string
  todos: EpicTodo[]
  updatedAt: string
}

export interface TrackerBoard {
  id: string
  name: string
  type: string
}

export interface Project {
  id: string
  name: string
  slug: string
  description: string
  icon: string
  color: AccentColor | string
  repoPath: string
  /**
   * Répertoires de travail connus du projet. Alimentée automatiquement :
   * dès qu'un ticket épingle un nouveau CWD, le chemin est enregistré ici.
   */
  repoPaths?: string[]
  /**
   * Chaque tâche travaille dans son propre worktree Git isolé, ou directement
   * dans le clone si l'option est désactivée. Vrai par défaut.
   */
  useWorktrees?: boolean
  /** Board du tracker retenu pour ce projet. */
  boardId?: string
  /**
   * Colonnes du board, à la façon de Jira : un nom et les statuts du tracker
   * que la colonne regroupe. Importables depuis le board, puis modifiables.
   */
  trackerColumns?: TrackerColumn[]
  /** Sprints du board avec leur état, rafraîchis par la synchro. */
  sprints?: TrackerSprint[]
  /** Étape du workflow agentique -> colonnes concernées (une ou plusieurs). */
  stageColumns?: Record<string, string[]>
  gitRemoteUrl?: string
  linearTeam: string
  githubRepo: string
  /** Jira project key used as `acli --project`, e.g. "PE". */
  jiraProject?: string
  issueTracker: IssueTracker
  /** Linear project URL, or the Jira base URL (e.g. https://acme.atlassian.net). */
  trackerUrl?: string
  /**
   * "standard" for a delivery project, "personal" for a personal board. The
   * daily digest is only served for a personal project.
   */
  projectType?: ProjectType
  isDefault: boolean
  taskCount?: number
  stageMapping?: Record<WorkflowStage, string>
  skillOverrides?: Record<string, string>
  aiProvider?: AIProvider
  aiCommandTemplate?: string
  specFramework?: SpecFramework
  createdAt: string
  updatedAt: string
}

export interface DetectedStatus {
  id: string
  name: string
  type?: string
  color?: string
  source?: string
}

export interface Task {
  id: string
  projectId?: string
  key: string
  title: string
  description: string
  status: Status
  priority: Priority
  labels: string[]
  assignee: string
  assigneeAvatar?: string
  position: number
  dueDate?: string | null
  branchName?: string
  prUrl?: string
  /** Répertoire de travail propre au ticket. Vide = hérite du projet, puis du réglage global. */
  repoPath?: string
  /** Statut brut du tracker, tel qu'il l'écrit (« Dev Test », « To Merge »…). */
  trackerStatus?: string
  /** Sprint / itération du tracker (champ Sprint côté Jira). */
  sprint?: string
  /** Équipe du tracker (champ Team côté Jira). */
  team?: string
  source?: TaskSource
  externalUrl?: string
  /** Tracker work item type. Only "Task" and "Story" are imported. */
  issueType?: string
  /**
   * Parent work item — an epic, or a parent story for a sub-task — carried as a
   * property of the task rather than as a card of its own.
   */
  parentKey?: string
  parentTitle?: string
  parentType?: string
  activities?: TaskActivity[]
  createdAt: string
  updatedAt: string
}

export interface GitDiffFile {
  path: string
  oldPath?: string
  status: 'modified' | 'added' | 'deleted' | 'renamed' | string
  additions: number
  deletions: number
  diff: string
}

export interface GitDiffResult {
  taskKey: string
  branch: string
  baseBranch: string
  repoPath: string
  worktreePath?: string
  isClean: boolean
  filesChanged: number
  insertions: number
  deletions: number
  files: GitDiffFile[]
  rawDiff: string
  prUrl?: string
  error?: string
}

export interface WorktreeInfo {
  taskKey: string
  branch: string
  worktreePath: string
  exists: boolean
  mainRepoPath: string
}

export interface GitStatusInfo {
  repoPath: string
  isGitRepo: boolean
  branch: string
  baseBranch?: string
  isClean: boolean
  modifiedCount: number
  untrackedCount: number
  ahead: number
  behind: number
  remoteName?: string
  remoteUrl?: string
  latestCommit?: string
  error?: string
}

export interface Skill {
  id: string
  name: string
  command: string
  description: string
  inputStatus: Status
  outputStatus: Status
  icon: string
  color: string
  steps: string[]
}

export type AccentColor = 
  | 'indigo'
  | 'violet'
  | 'emerald'
  | 'amber'
  | 'rose'
  | 'cyan'
  | 'blue'
  | 'orange'
  | 'neon-cyan'
  | 'neon-purple'
  | 'neon-green'
  | 'neon-amber'

export type Theme = 'dark' | 'light' | 'system'

export type Language = 'fr' | 'en'

export type Density = 'compact' | 'standard' | 'comfortable'

export type ViewMode = 'board' | 'list' | 'roadmap' | 'activities' | 'sync' | 'digest'

export type BoardGroupingMode = 'workflow' | 'status'

export type WorkflowStage = 'new' | 'clarified' | 'specified' | 'implemented' | 'reviewed' | 'finished'

export type DetailMode = 'modal' | 'panel'

export type AIProvider = 'agy' | 'vibe' | 'claude' | 'gemini' | 'codex' | 'cursor' | 'custom'

export type IssueTracker = 'linear' | 'github' | 'jira' | 'local'

export type ProjectType = 'standard' | 'personal'

/**
 * Spec-Driven Design frameworks Taskacao can scaffold into a project.
 * - `speckit`  : GitHub Spec Kit (`specify` CLI, `.specify/` + `specs/`)
 * - `openspec` : OpenSpec (`openspec` CLI, `openspec/changes/` + `openspec/specs/`)
 */
export type SpecFramework = 'speckit' | 'openspec'

export interface SpecFrameworkStatus {
  framework: SpecFramework
  frameworkLabel: string
  repoPath: string
  cliAvailable: boolean
  cliCommand: string
  initialized: boolean
  markerPaths?: string[]
  installHint?: string
}

export interface SpecFrameworkStep {
  label: string
  command: string
  success: boolean
  skipped: boolean
  output?: string
  error?: string
}

export interface SpecFrameworkInstallResult {
  framework: SpecFramework
  frameworkLabel: string
  repoPath: string
  installed: boolean
  alreadyInit: boolean
  version?: string
  markerPaths?: string[]
  steps: SpecFrameworkStep[]
  message: string
  error?: string
}

export interface UserSettings {
  id: number
  theme: Theme
  accentColor: AccentColor
  language: Language
  density: Density
  defaultView: ViewMode
  detailMode: DetailMode
  userName: string
  userEmail: string
  userAvatar: string
  aiProvider: AIProvider
  aiCommandTemplate: string
  repoPath: string
  issueTracker: IssueTracker
  linearTeam: string
  githubRepo: string
  jiraProject?: string
  jiraUrl?: string
  /** Identifiants de l'API Jira, requis pour importer Sprint et Team. */
  jiraEmail?: string
  /**
   * Jamais renvoyé par l'API. En écriture, une chaîne vide conserve le jeton
   * existant et la sentinelle `__clear__` l'efface.
   */
  jiraApiToken?: string
  /** Un jeton est configuré, en base ou par variable d'environnement. */
  jiraApiTokenSet?: boolean
  /** Le jeton vient de TASKACAO_JIRA_API_TOKEN et prime sur la base. */
  jiraApiTokenFromEnv?: boolean
  promptClarify: string
  promptSpecify: string
  promptImplement: string
  promptCreatePr: string
  promptPick: string
  editorCommand: string
  specFramework?: SpecFramework
  updatedAt: string
}

export interface DigestTaskRef {
  key: string
  title: string
  status: Status
  priority: Priority
  issueType?: string
  assignee?: string
  parentKey?: string
  parentTitle?: string
  externalUrl?: string
  branchName?: string
  prUrl?: string
  dueDate?: string
  ageDays: number
  isStale: boolean
  /** The tracker did not expose real dates at sync time. */
  datesUnknown?: boolean
  daysToDue?: number
}

export interface DigestEpicGroup {
  parentKey: string
  parentTitle?: string
  openCount: number
  doneCount: number
}

export interface DigestStats {
  totalOpen: number
  urgent: number
  high: number
  stale: number
  overdue: number
  awaitingReview: number
  doneLast7Days: number
  openDateUnknown: number
  closedDateUnknown: number
}

export type DigestAIStatus = 'none' | 'queued' | 'running' | 'completed' | 'failed'

export interface DailyDigest {
  projectId: string
  projectName: string
  date: string
  /** Narrows the digest to one person; empty means the whole project. */
  assignee: string
  /** Every assignee present in the project's tasks, in the tracker's spelling. */
  assignees?: string[]
  focus: DigestTaskRef[]
  watch: DigestTaskRef[]
  stale: DigestTaskRef[]
  dueSoon: DigestTaskRef[]
  awaitingReview: DigestTaskRef[]
  recentlyDone: DigestTaskRef[]
  byEpic: DigestEpicGroup[]
  stats: DigestStats
  /** Markdown agenda produced by the project's AI agent (meetings). */
  agenda?: string
  aiStatus: DigestAIStatus
  aiError?: string
  aiActivityId?: string
  aiUpdatedAt?: string
  markdown: string
  generatedAt: string
}

export interface CliStatus {
  tool: string
  available: boolean
  path: string
  authStatus: string
  details: string
}

export interface ToastMessage {
  id: string
  type: 'success' | 'info' | 'warning' | 'error'
  title: string
  description?: string
  duration?: number
}

export interface InstalledSkillInfo {
  id: string
  name: string
  installed: boolean
  path: string
  description: string
}

export interface ProjectSkillsStatus {
  projectId: string
  projectName: string
  repoPath: string
  pathExists: boolean
  isGitRepo?: boolean
  gitBranch?: string
  installedAll: boolean
  specFramework?: SpecFramework
  worktreesCount?: number
  worktreePaths?: string[]
  skills: InstalledSkillInfo[]
}

export interface GitBranchItem {
  name: string
  isCurrent: boolean
  isRemote: boolean
  commit?: string
  message?: string
}

export interface GitBranchesInfo {
  repoPath: string
  currentBranch: string
  branches: GitBranchItem[]
}

export interface ProjectGitInitResult {
  repoPath: string
  isGitRepo: boolean
  branch: string
  message: string
  initialized: boolean
}
