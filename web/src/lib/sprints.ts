import type { TrackerSprint } from '../types'

export interface SprintTimelineConfig {
  durationDays: number
  startDate: string
}

/** Formate une date en YYYY-MM-DD */
export const formatDateISO = (d: Date): string => {
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}-${m}-${day}`
}

/** Récupère le lundi de la semaine d'une date */
export const getMonday = (d: Date): Date => {
  const date = new Date(d)
  const day = date.getDay()
  const diff = date.getDate() - day + (day === 0 ? -6 : 1)
  date.setDate(diff)
  date.setHours(0, 0, 0, 0)
  return date
}

/** Formate une date en français lisible (ex: 1 sept. 2026) */
export const formatDateFR = (dateStr?: string): string => {
  if (!dateStr) return ''
  const d = new Date(dateStr)
  if (isNaN(d.getTime())) return dateStr
  return d.toLocaleDateString('fr-FR', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
  })
}

/** Calcule les dates consécutives pour une liste de sprints */
export const calculateSprintDates = (
  sprints: TrackerSprint[],
  startDateStr: string,
  durationDays: number
): TrackerSprint[] => {
  let currentStart = new Date(startDateStr)
  if (isNaN(currentStart.getTime())) {
    currentStart = getMonday(new Date())
  }

  const today = new Date()
  today.setHours(0, 0, 0, 0)

  return sprints.map((sprint, idx) => {
    const start = new Date(currentStart)
    const end = new Date(start)
    end.setDate(end.getDate() + Math.max(1, durationDays) - 1)
    end.setHours(23, 59, 59, 999)

    let state = sprint.state || 'future'
    if (today > end) {
      state = 'closed'
    } else if (today >= start && today <= end) {
      state = 'active'
    } else {
      state = 'future'
    }

    const nextStart = new Date(end)
    nextStart.setDate(nextStart.getDate() + 1)
    nextStart.setHours(0, 0, 0, 0)
    currentStart = nextStart

    return {
      ...sprint,
      name: sprint.name || `Sprint ${idx + 1}`,
      startDate: formatDateISO(start),
      endDate: formatDateISO(end),
      state,
    }
  })
}

/** Génère une liste de sprints par défaut */
export const generateDefaultSprints = (
  count: number = 4,
  startDateStr?: string,
  durationDays: number = 14
): TrackerSprint[] => {
  const start = startDateStr ? new Date(startDateStr) : getMonday(new Date())
  const placeholders: TrackerSprint[] = Array.from({ length: count }, (_, i) => ({
    id: `sprint-${i + 1}`,
    name: `Sprint ${i + 1}`,
    state: 'future',
  }))
  return calculateSprintDates(placeholders, formatDateISO(start), durationDays)
}

/** Calcule les jours restants ou le délai jusqu'à un sprint */
export const getSprintRelativeInfo = (sprint: TrackerSprint): { label: string; type: 'current' | 'future' | 'past' } => {
  if (!sprint.startDate || !sprint.endDate) {
    return { label: sprint.state === 'active' ? 'En cours' : sprint.state === 'closed' ? 'Terminé' : 'À venir', type: 'future' }
  }

  const today = new Date()
  today.setHours(0, 0, 0, 0)
  const start = new Date(sprint.startDate)
  const end = new Date(sprint.endDate)
  end.setHours(23, 59, 59, 999)

  if (today >= start && today <= end) {
    const diffMs = end.getTime() - today.getTime()
    const daysLeft = Math.ceil(diffMs / (1000 * 60 * 60 * 24))
    return {
      label: daysLeft <= 1 ? "Dernier jour du sprint !" : `En cours (${daysLeft} jours restants)`,
      type: 'current',
    }
  }

  if (today < start) {
    const diffMs = start.getTime() - today.getTime()
    const daysUntil = Math.ceil(diffMs / (1000 * 60 * 60 * 24))
    return {
      label: daysUntil === 1 ? "Débute demain" : `Débute dans ${daysUntil} jours`,
      type: 'future',
    }
  }

  return {
    label: "Sprint terminé",
    type: 'past',
  }
}
