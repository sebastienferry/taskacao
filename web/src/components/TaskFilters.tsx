import React from 'react'
import { Flame, Calendar, Layers } from 'lucide-react'
import { useApp } from '../context/AppContext'
import type { Priority } from '../types'

/**
 * Les filtres de tri transversaux, posés dans la barre d'outils de chaque vue
 * plutôt que dans l'en-tête global, pour rester à côté du contenu qu'ils
 * filtrent. Un seul composant partagé par le board et la liste : deux copies
 * finiraient par diverger.
 *
 * Sprint et Équipe n'apparaissent que si le tracker alimente réellement ces
 * champs, ce que dit l'endpoint des facettes. Un projet local ou GitHub n'affiche
 * donc que la priorité.
 */
export const TaskFilters: React.FC = () => {
  const {
    priorityFilter,
    setPriorityFilter,
    taskFacets,
    sprintFilter,
    setSprintFilter,
    teamFilter,
    setTeamFilter,
    t,
  } = useApp()

  const selectClass =
    'text-[11px] font-medium bg-[var(--bg-secondary)] text-[var(--text-secondary)] border border-[var(--border-color)] rounded-md px-2 py-1 focus:outline-none focus:border-[var(--accent-color)] cursor-pointer'

  return (
    <div className="flex items-center gap-2 shrink-0">
      <div className="flex items-center gap-1">
        <Flame size={12} className={priorityFilter ? 'text-rose-400' : 'text-[var(--text-muted)]'} />
        <select
          value={priorityFilter || ''}
          onChange={e => setPriorityFilter((e.target.value || null) as Priority | null)}
          title="Filtrer par priorité"
          className={selectClass}
        >
          <option value="">Toutes priorités</option>
          <option value="urgent">{t.priority.urgent}</option>
          <option value="high">{t.priority.high}</option>
          <option value="medium">{t.priority.medium}</option>
          <option value="low">{t.priority.low}</option>
        </select>
      </div>

      {taskFacets.sprints.length > 0 && (
        <div className="flex items-center gap-1">
          <Calendar size={12} className={sprintFilter ? 'text-cyan-400' : 'text-[var(--text-muted)]'} />
          <select
            value={sprintFilter || ''}
            onChange={e => setSprintFilter(e.target.value || null)}
            title="Filtrer par sprint"
            className={`${selectClass} max-w-[170px]`}
          >
            <option value="">Tous sprints</option>
            {taskFacets.sprints.map(sprint => (
              <option key={sprint} value={sprint}>{sprint}</option>
            ))}
          </select>
        </div>
      )}

      {taskFacets.teams.length > 0 && (
        <div className="flex items-center gap-1">
          <Layers size={12} className={teamFilter ? 'text-violet-400' : 'text-[var(--text-muted)]'} />
          <select
            value={teamFilter || ''}
            onChange={e => setTeamFilter(e.target.value || null)}
            title="Filtrer par équipe"
            className={`${selectClass} max-w-[170px]`}
          >
            <option value="">Toutes équipes</option>
            {taskFacets.teams.map(team => (
              <option key={team} value={team}>{team}</option>
            ))}
          </select>
        </div>
      )}
    </div>
  )
}
