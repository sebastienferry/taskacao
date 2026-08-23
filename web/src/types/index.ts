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

export type TaskSource = 'linear' | 'github' | 'local'

export type ActivityStatus = 'queued' | 'pending' | 'running' | 'completed' | 'failed' | 'canceled'

export interface TaskActivity {
  id: string
  taskId: string
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

export interface TaskMessage {
  id: string
  taskId: string
  role: 'user' | 'assistant' | 'system'
  content: string
  activityId?: string
  skillId?: string
  steps?: string[]
  createdAt: string
}

export interface ActivityStats {
  total: number
  queued: number
  running: number
  completed: number
  failed: number
  canceled: number
}

export interface Project {
  id: string
  name: string
  slug: string
  description: string
  icon: string
  color: AccentColor | string
  repoPath: string
  gitRemoteUrl?: string
  linearTeam: string
  githubRepo: string
  issueTracker: IssueTracker
  trackerUrl?: string
  isDefault: boolean
  taskCount?: number
  stageMapping?: Record<WorkflowStage, string>
  skillOverrides?: Record<string, string>
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
  source?: TaskSource
  externalUrl?: string
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

export type ViewMode = 'board' | 'list' | 'activities' | 'sync'

export type BoardGroupingMode = 'workflow' | 'status'

export type WorkflowStage = 'new' | 'clarified' | 'specified' | 'implemented' | 'reviewed' | 'finished'

export type DetailMode = 'modal' | 'panel'

export type AIProvider = 'agy' | 'vibe' | 'claude' | 'custom'

export type IssueTracker = 'linear' | 'github' | 'local'

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
  promptClarify: string
  promptSpecify: string
  promptImplement: string
  promptCreatePr: string
  promptPick: string
  updatedAt: string
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
