import React from 'react'
import { Flame, Calendar, Layers, Pin } from 'lucide-react'
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
    pinnedOnly,
    setPinnedOnly,
    pinnedTasks,
    t,
  } = useApp()

  // Même palette que les cartes et la vue liste, du plus urgent au moins urgent.
  const PRIORITY_ORDER: Priority[] = ['urgent', 'high', 'medium', 'low']
  const PRIORITY_DOTS: Record<Priority, { color: string; label: string }> = {
    urgent: { color: 'var(--status-danger)', label: t.priority.urgent },
    high: { color: 'var(--status-warn)', label: t.priority.high },
    medium: { color: 'var(--status-info)', label: t.priority.medium },
    low: { color: 'var(--text-muted)', label: t.priority.low },
  }

  const selectClass =
    'text-[11px] font-medium bg-[var(--bg-secondary)] text-[var(--text-secondary)] border border-[var(--border-color)] rounded-md px-2 py-1 focus:outline-none focus:border-[var(--accent-color)] cursor-pointer'

  return (
    <div className="flex items-center gap-2 shrink-0">
      {/* Épinglés : le retour immédiat aux chantiers en cours quand le board en
          porte trois cents. Le filtre est tenu par le serveur, sur la colonne
          indexée, donc il vaut aussi pour la recherche et les autres filtres. */}
      <button
        type="button"
        onClick={() => setPinnedOnly(!pinnedOnly)}
        disabled={!pinnedOnly && pinnedTasks.length === 0}
        className={`flex items-center gap-1 px-2 py-1 rounded-md text-[11px] font-medium border transition-colors cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed ${
          pinnedOnly
            ? 'accent-text bg-[var(--accent-light)] border-[var(--accent-color)]/50 font-bold'
            : 'bg-[var(--bg-secondary)] text-[var(--text-secondary)] border-[var(--border-color)] hover:text-[var(--text-primary)]'
        }`}
        title={
          pinnedTasks.length === 0
            ? "Aucun ticket épinglé pour l'instant"
            : pinnedOnly
              ? 'Afficher tous les tickets'
              : `N'afficher que les ${pinnedTasks.length} ticket(s) épinglé(s)`
        }
      >
        <Pin size={12} />
        <span>Épinglés</span>
        {pinnedTasks.length > 0 && (
          <span className="font-mono text-[10px] opacity-70">{pinnedTasks.length}</span>
        )}
      </button>

      {/* Priorité en pastilles plutôt qu'en liste déroulante : un select natif ne
          sait afficher que du texte, et la couleur est justement l'information.
          Un clic filtre, un second clic sur la pastille active l'enlève. */}
      <div className="flex items-center gap-1">
        <Flame size={12} className={priorityFilter ? 'text-rose-400' : 'text-[var(--text-muted)]'} />
        <div className="flex items-center gap-1 px-1 py-0.5 rounded-md bg-[var(--bg-secondary)] border border-[var(--border-color)]">
          {PRIORITY_ORDER.map(level => {
            const isActive = priorityFilter === level
            return (
              <button
                key={level}
                type="button"
                onClick={() => setPriorityFilter(isActive ? null : level)}
                title={isActive ? `Retirer le filtre ${PRIORITY_DOTS[level].label}` : `Filtrer : ${PRIORITY_DOTS[level].label}`}
                aria-pressed={isActive}
                className={`w-4 h-4 rounded-full flex items-center justify-center transition-all cursor-pointer ${
                  isActive
                    ? 'ring-2 ring-[var(--accent-color)] scale-110'
                    : priorityFilter
                      ? 'opacity-40 hover:opacity-100'
                      : 'hover:scale-110'
                }`}
              >
                <span
                  className="w-2.5 h-2.5 rounded-full ring-1 ring-black/10"
                  style={{ backgroundColor: PRIORITY_DOTS[level].color }}
                />
              </button>
            )
          })}
        </div>
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
