import type { LookupOption } from '../components/LookupField'
import type { EpicMeta, TrackerSprint, Project } from '../types'

export const isProjectCompatible = (
  p1: Project | null | undefined,
  p2: Project | null | undefined
): boolean => {
  if (!p1 || !p2) return false
  if (p1.id === p2.id) return false
  const t1 = (p1.issueTracker || 'local').toLowerCase().trim()
  const t2 = (p2.issueTracker || 'local').toLowerCase().trim()
  if (t1 === t2 || t1 === 'local' || t2 === 'local') return true
  if ((p1.githubRepo || t1 === 'github') && (p2.githubRepo || t2 === 'github')) return true
  return false
}

/**
 * Sources de recherche pour les champs de type lookup.
 *
 * Deux familles de valeurs cohabitent dans l'outil. Celles qui vivent dans le
 * tracker et qu'aucune liste locale ne contient (les personnes, les équipes de
 * l'instance) sont cherchées par un appel réseau. Celles que la synchronisation a
 * déjà ramenées (les sprints du board, les épics du projet) sont filtrées ici,
 * sans réseau : un projet porte cent quarante épics, ce qui rend une liste
 * déroulante inutilisable, mais reste minuscule à filtrer en mémoire.
 *
 * Toutes renvoient une promesse, pour que le champ de recherche traite les deux
 * familles de la même façon.
 */

/** Nombre de propositions rendues sans frappe : au delà, la liste ne se lit plus. */
const DEFAULT_LIMIT = 40

const matches = (haystack: string, query: string): boolean =>
  haystack.toLowerCase().includes(query.trim().toLowerCase())

/**
 * Épics du projet, cherchés par clé ou par titre. Les épics terminés sont exclus
 * par défaut : on ne rattache pas un ticket à un chantier clos, mais l'épic
 * courant d'un ticket reste proposé par l'appelant s'il est hors liste.
 */
export const epicLookup =
  (epics: EpicMeta[], options?: { includeClosed?: boolean }) =>
  async (query: string): Promise<LookupOption[]> => {
    const pool = options?.includeClosed ? epics : epics.filter(epic => !epic.closed)
    const found = pool.filter(epic => {
      if (!query.trim()) return true
      return matches(epic.key, query) || matches(epic.title || '', query)
    })
    return found.slice(0, DEFAULT_LIMIT).map(epic => ({
      id: epic.key,
      label: epic.key,
      sublabel: epic.title || undefined,
    }))
  }

/**
 * Sprints du board, cherchés par nom. Les sprints clos sont écartés, ainsi que
 * ceux dépourvus d'identifiant : l'API Agile ne déplace un ticket que par
 * identifiant, donc un sprint sans le sien ne serait pas applicable.
 */
export const sprintLookup =
  (sprints: TrackerSprint[]) =>
  async (query: string): Promise<LookupOption[]> => {
    const pool = sprints.filter(sprint => sprint.id && sprint.state !== 'closed')
    const found = pool.filter(sprint => (query.trim() ? matches(sprint.name, query) : true))
    return found.slice(0, DEFAULT_LIMIT).map(sprint => ({
      id: sprint.id as string,
      label: sprint.name,
      sublabel: sprint.state === 'active' ? 'sprint en cours' : undefined,
    }))
  }

/**
 * Valeurs déjà présentes sur les tickets, pour les filtres : un sprint, une
 * équipe ou une personne que le board porte réellement. Le compteur donne le
 * poids de chaque valeur, ce qui aide à choisir.
 */
export const valueLookup =
  (values: string[], counts?: Record<string, number>) =>
  async (query: string): Promise<LookupOption[]> => {
    const found = values.filter(value => (query.trim() ? matches(value, query) : true))
    return found.slice(0, DEFAULT_LIMIT).map(value => ({
      id: value,
      label: value,
      sublabel: counts?.[value] ? `${counts[value]} ticket(s)` : undefined,
    }))
  }
