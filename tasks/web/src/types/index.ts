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

export interface TaskActivity {
  id: string
  taskId: string
  skillId: string
  skillName: string
  action: string
  status: 'running' | 'completed' | 'failed'
  summary: string
  output: string
  steps: string[]
  createdAt: string
}

export interface Task {
  id: string
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

export type ViewMode = 'board' | 'list'

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
