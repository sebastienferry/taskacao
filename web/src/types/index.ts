export type Priority = 'urgent' | 'high' | 'medium' | 'low'

export type Status = 
  | 'to_clarify'    // A clarifier (Label: New)
  | 'to_specify'    // A spécifier (Label: Clarified)
  | 'to_implement'  // A implémenter (Label: Specified)
  | 'to_test'       // A tester (Label: Implemented)
  | 'to_close'      // A fermer (Label: Reviewed)
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
  linearTeam: string
  githubRepo: string
  issueTracker: IssueTracker
  isDefault: boolean
  taskCount?: number
  createdAt: string
  updatedAt: string
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
  isClean: boolean
  filesChanged: number
  insertions: number
  deletions: number
  files: GitDiffFile[]
  rawDiff: string
  prUrl?: string
  error?: string
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

export type Theme = 'dark' | 'light' | 'system'

export type Language = 'fr' | 'en'

export type Density = 'compact' | 'standard' | 'comfortable'

export type ViewMode = 'board' | 'list' | 'activities' | 'sync'

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

export interface ProjectGitInitResult {
  repoPath: string
  isGitRepo: boolean
  branch: string
  message: string
  initialized: boolean
}
