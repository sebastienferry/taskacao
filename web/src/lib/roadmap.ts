import type { EpicMeta, Priority, Project, Task, TrackerSprint, WorkflowStage } from '../types'
import { WORKFLOW_ORDER, resolveTaskStage } from './workflow'

/**
 * Agrégation des épics pour la vue Roadmap.
 *
 * Taskacao n'importe pas les épics comme cartes — ce sont des conteneurs, portés
 * par les tickets sous forme de `parentKey` / `parentTitle`. Un épic est donc
 * reconstruit ici depuis ses enfants, et tout ce que la vue affiche est déduit
 * d'eux : c'est la seule source disponible, et elle a l'avantage d'être toujours
 * à jour après une synchro.
 */

export type Horizon = 'now' | 'next' | 'later'
/** Onglet de la roadmap : les trois horizons, plus les épics pas encore arbitrés. */
export type HorizonTab = Horizon | 'unclassified'
export type Maturity = 'Draft' | 'Clarified' | 'Specified' | 'Ready'

export interface EpicRow {
  key: string
  title: string
  /** Équipe la plus représentée chez les enfants, faute d'équipe portée par l'épic. */
  squad: string
  /** Classification retenue par l'utilisateur, vide si l'épic n'est pas arbitré. */
  horizon: Horizon | ''
  /** Classification que les données suggèrent, pour proposer un arbitrage. */
  suggested: Horizon
  maturity: Maturity
  priority: Priority
  tasks: Task[]
  /** Enfants encore ouverts : ceux dont le placement en sprint est à vérifier. */
  open: Task[]
  /** Ouverts dans un sprint actif. */
  inActiveSprint: Task[]
  /** Ouverts dans un sprint futur. */
  inFutureSprint: Task[]
  /** Ouverts dans un sprint clos ou inconnu du board : anomalie à corriger. */
  inStaleSprint: Task[]
  /** Ouverts sans aucun sprint. */
  unscheduled: Task[]
  sprints: string[]
  meta?: EpicMeta
}

const PRIORITY_RANK: Record<Priority, number> = { urgent: 4, high: 3, medium: 2, low: 1 }

/**
 * Couleurs de la vue : uniquement des variables globales de l'app, jamais de
 * valeur en dur. Le design system Equativ vit déjà dans index.css — accent,
 * couleurs de signal, thèmes clair et sombre, variantes par projet — donc figer
 * la palette du design ici priverait la vue du thème et de l'accent choisis.
 *
 * L'accent porte NOW, puisque c'est l'orange de la marque par défaut et qu'un
 * projet peut légitimement en changer.
 */
export const HORIZON_META: Record<Horizon, { label: string; hint: string; color: string; bg: string; border: string }> = {
  now: {
    label: 'NOW',
    hint: 'Sprint en cours',
    color: 'var(--accent-color)',
    bg: 'var(--accent-light)',
    border: 'rgb(var(--accent-rgb) / 0.45)',
  },
  next: {
    label: 'NEXT',
    hint: 'Sprints à venir',
    color: 'var(--status-info)',
    bg: 'rgb(var(--status-info-rgb) / 0.13)',
    border: 'rgb(var(--status-info-rgb) / 0.4)',
  },
  later: {
    label: 'LATER',
    hint: 'Cadrage',
    color: 'var(--status-warn)',
    bg: 'rgb(var(--status-warn-rgb) / 0.12)',
    border: 'rgb(var(--status-warn-rgb) / 0.32)',
  },
}

export const MATURITY_META: Record<Maturity, { pct: number; color: string; bg: string; border: string }> = {
  Draft: { pct: 20, color: 'var(--text-muted)', bg: 'var(--bg-tertiary)', border: 'var(--border-color)' },
  Clarified: { pct: 50, color: 'var(--status-warn)', bg: 'rgb(var(--status-warn-rgb) / 0.14)', border: 'rgb(var(--status-warn-rgb) / 0.34)' },
  Specified: { pct: 78, color: 'var(--status-info)', bg: 'rgb(var(--status-info-rgb) / 0.12)', border: 'rgb(var(--status-info-rgb) / 0.32)' },
  Ready: { pct: 100, color: 'var(--status-ok)', bg: 'rgb(var(--status-ok-rgb) / 0.13)', border: 'rgb(var(--status-ok-rgb) / 0.32)' },
}

export const PRIORITY_META: Record<Priority, { label: string; color: string; bg: string }> = {
  urgent: { label: 'Critical', color: 'var(--status-danger)', bg: 'rgb(var(--status-danger-rgb) / 0.13)' },
  high: { label: 'High', color: 'var(--accent-color)', bg: 'var(--accent-light)' },
  medium: { label: 'Medium', color: 'var(--status-info)', bg: 'rgb(var(--status-info-rgb) / 0.12)' },
  low: { label: 'Low', color: 'var(--text-muted)', bg: 'var(--bg-tertiary)' },
}

const isOpen = (task: Task): boolean => task.status !== 'finished' && task.status !== 'done'

/**
 * Maturité de l'épic = étape la moins avancée parmi ses enfants ouverts. Un épic
 * n'est « Ready » que si aucun de ses tickets n'attend encore un cadrage ou une
 * spécification.
 */
const maturityOf = (open: Task[], project?: Project | null): Maturity => {
  if (open.length === 0) return 'Ready'
  let lowest = WORKFLOW_ORDER.length - 1
  open.forEach(task => {
    const index = WORKFLOW_ORDER.indexOf(resolveTaskStage(task, project))
    if (index >= 0 && index < lowest) lowest = index
  })
  const stage: WorkflowStage = WORKFLOW_ORDER[lowest]
  if (stage === 'new') return 'Draft'
  if (stage === 'clarified') return 'Clarified'
  if (stage === 'specified') return 'Specified'
  return 'Ready'
}

/**
 * Classification suggérée, jamais imposée : du travail dans un sprint actif
 * ressemble à du NOW, dans un sprint futur à du NEXT, et le reste à du cadrage.
 * L'utilisateur tranche, la suggestion ne sert qu'à proposer un arbitrage en un
 * clic sur les épics encore non classés.
 */
const suggestHorizon = (inActive: Task[], inFuture: Task[]): Horizon => {
  if (inActive.length > 0) return 'now'
  if (inFuture.length > 0) return 'next'
  return 'later'
}

const sprintStateIndex = (sprints?: TrackerSprint[]): Map<string, string> => {
  const map = new Map<string, string>()
  ;(sprints || []).forEach(sp => {
    const name = (sp.name || '').trim().toLowerCase()
    if (name) map.set(name, (sp.state || '').toLowerCase())
  })
  return map
}

export const buildEpicRows = (
  tasks: Task[],
  project?: Project | null,
  epicMeta?: EpicMeta[]
): EpicRow[] => {
  const metaByKey = new Map<string, EpicMeta>()
  ;(epicMeta || []).forEach(m => metaByKey.set(m.key, m))
  const states = sprintStateIndex(project?.sprints)

  const byKey = new Map<string, Task[]>()
  tasks.forEach(task => {
    const key = (task.parentKey || '').trim()
    if (!key) return
    const list = byKey.get(key)
    if (list) list.push(task)
    else byKey.set(key, [task])
  })

  const rows: EpicRow[] = []
  byKey.forEach((children, key) => {
    const open = children.filter(isOpen)

    const inActiveSprint: Task[] = []
    const inFutureSprint: Task[] = []
    const inStaleSprint: Task[] = []
    const unscheduled: Task[] = []
    open.forEach(task => {
      const sprint = (task.sprint || '').trim()
      if (!sprint) {
        unscheduled.push(task)
        return
      }
      // Un sprint que le board ne connaît plus (clos, ou hors des sprints
      // rapatriés) est une anomalie autant qu'un ticket sans sprint.
      switch (states.get(sprint.toLowerCase())) {
        case 'active':
          inActiveSprint.push(task)
          break
        case 'future':
          inFutureSprint.push(task)
          break
        default:
          inStaleSprint.push(task)
      }
    })

    const teamCounts = new Map<string, number>()
    children.forEach(t => {
      const team = (t.team || '').trim()
      if (team) teamCounts.set(team, (teamCounts.get(team) || 0) + 1)
    })
    const squad = Array.from(teamCounts.entries()).sort((a, b) => b[1] - a[1])[0]?.[0] || '—'

    let priority: Priority = 'low'
    children.forEach(t => {
      if ((PRIORITY_RANK[t.priority] || 0) > (PRIORITY_RANK[priority] || 0)) priority = t.priority
    })

    const meta = metaByKey.get(key)
    rows.push({
      key,
      title: children.find(t => (t.parentTitle || '').trim())?.parentTitle || key,
      squad,
      horizon: (meta?.horizon as Horizon | '') || '',
      suggested: suggestHorizon(inActiveSprint, inFutureSprint),
      maturity: maturityOf(open, project),
      priority,
      tasks: children,
      open,
      inActiveSprint,
      inFutureSprint,
      inStaleSprint,
      unscheduled,
      sprints: Array.from(
        new Set([...inActiveSprint, ...inFutureSprint, ...inStaleSprint].map(t => (t.sprint || '').trim()).filter(Boolean))
      ).sort(),
      meta,
    })
  })

  // Le plus gros chantier ouvert en premier : c'est celui qui demande le plus
  // d'attention à la revue de sprint.
  return rows.sort((a, b) => {
    if (b.open.length !== a.open.length) return b.open.length - a.open.length
    return a.key.localeCompare(b.key, undefined, { numeric: true })
  })
}

/**
 * Anomalies de placement pour un horizon opérationnel : dans NOW on attend un
 * sprint actif, dans NEXT un sprint futur. Tout le reste doit se voir.
 */
export const placementIssues = (row: EpicRow, horizon: Horizon): Task[] => {
  if (horizon === 'now') {
    return [...row.unscheduled, ...row.inStaleSprint, ...row.inFutureSprint]
  }
  if (horizon === 'next') {
    return [...row.unscheduled, ...row.inStaleSprint]
  }
  return []
}

/** État du placement d'un ticket au regard de l'horizon visé. */
export type PlacementState = 'ok' | 'other-horizon' | 'stale' | 'missing'

export const placementOf = (task: Task, row: EpicRow, horizon: Horizon): PlacementState => {
  if (row.unscheduled.includes(task)) return 'missing'
  if (row.inStaleSprint.includes(task)) return 'stale'
  if (horizon === 'now') return row.inActiveSprint.includes(task) ? 'ok' : 'other-horizon'
  if (horizon === 'next') return row.inFutureSprint.includes(task) ? 'ok' : 'other-horizon'
  return 'ok'
}

export const PLACEMENT_META: Record<PlacementState, { label: string; color: string; bg: string; border: string }> = {
  ok: {
    label: 'placé',
    color: 'var(--status-ok)',
    bg: 'rgb(var(--status-ok-rgb) / 0.13)',
    border: 'rgb(var(--status-ok-rgb) / 0.32)',
  },
  'other-horizon': {
    label: 'autre horizon',
    color: 'var(--status-info)',
    bg: 'rgb(var(--status-info-rgb) / 0.12)',
    border: 'rgb(var(--status-info-rgb) / 0.32)',
  },
  stale: {
    label: 'sprint clos',
    color: 'var(--status-warn)',
    bg: 'rgb(var(--status-warn-rgb) / 0.14)',
    border: 'rgb(var(--status-warn-rgb) / 0.34)',
  },
  missing: {
    label: 'sans sprint',
    color: 'var(--status-danger)',
    bg: 'rgb(var(--status-danger-rgb) / 0.13)',
    border: 'rgb(var(--status-danger-rgb) / 0.32)',
  },
}
