import type { Project, Task, WorkflowStage } from '../types'

/**
 * Étape du workflow agentique d'une tâche, et skill qui en découle.
 *
 * Deux sources possibles. Les labels du ticket, historiquement la seule, restent
 * la trace écrite dans le tracker. Et depuis que les colonnes du board portent
 * une affectation d'étapes, la colonne où se trouve le ticket décide : c'est
 * elle qui reflète l'état réel du travail côté équipe, alors qu'un label peut
 * n'avoir jamais été posé.
 */

export const WORKFLOW_ORDER: WorkflowStage[] = [
  'new',
  'clarified',
  'specified',
  'implemented',
  'reviewed',
  'finished',
]

/** Étape déduite des labels, puis du statut interne en repli. */
export const stageFromLabels = (task: Task): WorkflowStage => {
  const labels = (task.labels || []).map(l => l.toLowerCase())
  if (labels.includes('finished') || labels.includes('closed') || labels.includes('done')) return 'finished'
  if (labels.includes('reviewed')) return 'reviewed'
  if (labels.includes('implemented')) return 'implemented'
  if (labels.includes('specified')) return 'specified'
  if (labels.includes('clarified')) return 'clarified'
  if (labels.includes('new') || labels.includes('untouched')) return 'new'

  if (task.status === 'finished' || task.status === 'done') return 'finished'
  if (task.status === 'to_close') return 'reviewed'
  if (task.status === 'to_test' || task.status === 'to_validate') return 'implemented'
  if (task.status === 'to_implement' || task.status === 'in_progress') return 'specified'
  if (task.status === 'to_specify' || task.status === 'specified') return 'clarified'
  return 'new'
}

/** Colonne du board contenant la tâche, d'après son statut de tracker. */
export const columnOfTask = (task: Task, project?: Project | null): string | null => {
  const status = (task.trackerStatus || '').toLowerCase()
  if (!status || !project?.trackerColumns?.length) return null
  const column = project.trackerColumns.find(col =>
    col.statuses.some(st => st.toLowerCase() === status)
  )
  return column?.name || null
}

/**
 * Étape affectée à la colonne de la tâche. Quand une colonne en porte
 * plusieurs, la moins avancée gagne : c'est l'étape encore à faire dans cette
 * colonne, donc celle qui doit être proposée.
 */
export const stageFromColumn = (task: Task, project?: Project | null): WorkflowStage | null => {
  const column = columnOfTask(task, project)
  if (!column) return null
  const mapping = project?.stageColumns || {}
  for (const stage of WORKFLOW_ORDER) {
    if ((mapping[stage] || []).includes(column)) return stage
  }
  return null
}

/** Étape retenue : la colonne d'abord, les labels en repli. */
export const resolveTaskStage = (task: Task, project?: Project | null): WorkflowStage => {
  return stageFromColumn(task, project) || stageFromLabels(task)
}

/**
 * Skill à proposer pour une étape : l'étape nomme ce qui est déjà acquis, donc
 * la skill est l'étape suivante. Une tâche terminée n'en propose aucune.
 */
export const skillForStage = (stage: WorkflowStage): string | null => {
  switch (stage) {
    case 'new':
      return 'clarify'
    case 'clarified':
      return 'specify'
    case 'specified':
      return 'implement'
    case 'implemented':
    case 'reviewed':
      return 'create_pr'
    case 'finished':
    default:
      return null
  }
}
