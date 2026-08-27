import React, { useEffect, useMemo, useRef, useState } from 'react'
import {
  Target,
  CalendarRange,
  Route,
  Compass,
  HelpCircle,
  EyeOff,
  Eye,
  CalendarDays,
  ExternalLink,
  Plus,
  Check,
  Trash2,
  Terminal as TerminalIcon,
  AlertTriangle,
  Save,
  X,
  Filter,
  Scissors,
} from 'lucide-react'
import { useApp } from '../context/AppContext'
import { LookupField } from './LookupField'
import { MarkdownEditor } from './Markdown'
import { sprintLookup } from '../lib/lookups'
import {
  buildEpicRows,
  placementIssues,
  placementOf,
  belongsToProjectKey,
  HORIZON_META,
  MATURITY_META,
  PLACEMENT_META,
  PRIORITY_META,
  type EpicRow,
  type Horizon,
  type HorizonTab,
  tasksBySprintOrder,
  sprintLabelOf,
} from '../lib/roadmap'
import type { EpicHorizon, EpicMeta, EpicTodo, EpicRequiredField } from '../types'

/**
 * Roadmap des épics, d'après le design « Roadmap Epics.dc.html ».
 *
 * Deux métiers, pas un seul. NOW et NEXT sont opérationnels : on y vérifie que
 * les stories d'un épic sont bien dans un sprint — actif pour NOW, à venir pour
 * NEXT — et tout ce qui ne l'est pas doit sauter aux yeux. LATER est du design
 * d'épic : description et TODO se travaillent avant qu'il y ait des stories.
 *
 * La classification est une décision, pas une déduction : elle est stockée par
 * épic. Les données ne font que la suggérer, pour arbitrer en un clic les épics
 * qui n'ont pas encore été rangés.
 */

const TABS: { id: HorizonTab; label: string; icon: React.ReactNode }[] = [
  { id: 'now', label: 'NOW', icon: <Target size={14} /> },
  { id: 'next', label: 'NEXT', icon: <Route size={14} /> },
  { id: 'later', label: 'LATER', icon: <Compass size={14} /> },
  { id: 'unclassified', label: 'Non classés', icon: <HelpCircle size={14} /> },
  // Sans onglet dédié, un épic masqué serait injoignable : on ne pourrait plus
  // le reclasser.
  { id: 'hidden', label: 'Masqués', icon: <EyeOff size={14} /> },
]

export const RoadmapView: React.FC = () => {
  const {
    tasks,
    currentProject,
    settings,
    setSelectedTask,
    setChatTask,
    fetchProjectEpics,
    saveEpicMeta,
    createStoryFromEpicTodo,
    pendingHorizonPushes,
    pushPendingHorizons,
    setTaskEpic,
    createStoryUnderEpic,
    createEpic,
    moveTasksToEpic,
    setTaskSprint,
    setTasksSprint,
    assigneeFilter,
    setAssigneeFilter,
    sprintFilter,
    setSprintFilter,
    teamFilter,
    setTeamFilter,
    labelFilter,
    setLabelFilter,
    pinnedOnly,
    setPinnedOnly,
    searchQuery,
    setSearchQuery,
    fetchEpicRequiredFields,
    activeJobCount,
    addToast,
  } = useApp()

  const [tab, setTab] = useState<HorizonTab>('now')
  const [epicMeta, setEpicMeta] = useState<EpicMeta[]>([])
  const [selectedKey, setSelectedKey] = useState<string | null>(null)
  const [onlyIssues, setOnlyIssues] = useState(false)
  // Les épics terminés et ceux d'un autre projet Jira sont écartés par défaut :
  // une roadmap sert à décider de ce qui reste à faire chez soi.
  const [showClosed, setShowClosed] = useState(false)
  const [onlyProjectKey, setOnlyProjectKey] = useState(true)

  // Le cadrage n'est enregistré qu'à la demande : écrire à chaque frappe
  // enverrait une requête par caractère.
  const [draftDescription, setDraftDescription] = useState('')
  const [draftDirty, setDraftDirty] = useState(false)
  const [newTodo, setNewTodo] = useState('')
  const [creatingTodoId, setCreatingTodoId] = useState<string | null>(null)
  const [pendingPushes, setPendingPushes] = useState(0)
  const [isPushing, setIsPushing] = useState(false)
  const pendingTimer = useRef<number | null>(null)
  // Prototypage de l'épic : créer une story a la volée, ou y pousser un ticket
  // existant retrouvé par sa clé ou son titre.
  const [newStory, setNewStory] = useState('')
  const [attachQuery, setAttachQuery] = useState('')
  const [busyKey, setBusyKey] = useState<string | null>(null)
  // Découpe : les stories cochées partent vers un autre épic, existant ou créé
  // pour l'occasion.
  const [checked, setChecked] = useState<Record<string, boolean>>({})
  // Curseur de coupe : rang dans la liste ordonnée par sprint à partir duquel
  // les tickets quittent l'épic. 0 veut dire « tout couper », la longueur de la
  // liste veut dire « ne rien couper ».
  const [cutAt, setCutAt] = useState<number>(-1)
  const [draggingCut, setDraggingCut] = useState(false)
  // Champs que l'instance impose pour créer un épic, et la valeur retenue. PE
  // exige « Epic Type » : sans lui, Jira refuse la création en 400.
  const [epicFields, setEpicFields] = useState<EpicRequiredField[]>([])
  const [epicFieldValues, setEpicFieldValues] = useState<Record<string, string>>({})
  // Les lignes de la liste, pour retrouver la frontière la plus proche du
  // pointeur pendant le glissement. Les hauteurs varient avec les titres, donc
  // un calcul par pas fixe se décalerait.
  const rowRefs = useRef<(HTMLDivElement | null)[]>([])
  const [moveTarget, setMoveTarget] = useState('')
  const [newEpicTitle, setNewEpicTitle] = useState('')

  // Largeur du panneau de droite. C'est là que le travail se fait (les tickets de
  // l'épic, la découpe), alors que la colonne de gauche n'a qu'à lister : la
  // répartition par défaut donne donc la place au panneau, et la poignée permet
  // de la régler. La valeur est mémorisée par navigateur.
  // Sprints proposables : ceux du board du projet, actifs et futurs. Un sprint
  // clos n'accueille plus de travail, et l'API Agile déplace par identifiant, donc
  // ceux qui n'en portent pas (import antérieur) sont écartés.
  const sprintOptions = useMemo(
    () =>
      (currentProject?.sprints || [])
        .filter(sp => sp.id && sp.state !== 'closed')
        .map(sp => ({ id: sp.id as string, name: sp.name, state: sp.state })),
    [currentProject?.sprints]
  )
  const [sprintTarget, setSprintTarget] = useState<{ id: string; name: string }>({ id: '', name: '' })

  // Épics et sprints se cherchent au clavier : cette vue en liste cent quarante
  // et dix, et l'épic cible d'une découpe se choisissait dans un menu déroulant
  // de tout le projet.
  const searchSprint = useMemo(() => sprintLookup(currentProject?.sprints || []), [currentProject?.sprints])

  // Ce qui restreint la liste de tickets sur laquelle la roadmap est construite.
  const activeFilterChips = useMemo(() => {
    const chips: { label: string; clear: () => void }[] = []
    if (assigneeFilter) {
      chips.push({
        label: assigneeFilter === '__unassigned__' ? 'non assigné' : assigneeFilter,
        clear: () => setAssigneeFilter(null),
      })
    }
    if (sprintFilter) chips.push({ label: sprintFilter, clear: () => setSprintFilter(null) })
    if (teamFilter) chips.push({ label: teamFilter, clear: () => setTeamFilter(null) })
    if (labelFilter) chips.push({ label: `#${labelFilter.replace(/^#/, '')}`, clear: () => setLabelFilter(null) })
    if (pinnedOnly) chips.push({ label: 'épinglés seulement', clear: () => setPinnedOnly(false) })
    if (searchQuery) chips.push({ label: `« ${searchQuery} »`, clear: () => setSearchQuery('') })
    return chips
  }, [
    assigneeFilter,
    sprintFilter,
    teamFilter,
    labelFilter,
    pinnedOnly,
    searchQuery,
    setAssigneeFilter,
    setSprintFilter,
    setTeamFilter,
    setLabelFilter,
    setPinnedOnly,
    setSearchQuery,
  ])

  const PANEL_MIN = 420
  const LIST_MIN = 280
  const [panelWidth, setPanelWidth] = useState<number>(() => {
    const stored = Number(localStorage.getItem('taskacao_roadmap_panel_width') || '')
    return Number.isFinite(stored) && stored >= PANEL_MIN ? stored : 720
  })
  const splitRef = useRef<HTMLDivElement>(null)
  const [isDraggingSplit, setIsDraggingSplit] = useState(false)

  const startSplitDrag = (e: React.PointerEvent) => {
    e.preventDefault()
    const container = splitRef.current
    if (!container) return
    setIsDraggingSplit(true)

    const onMove = (ev: PointerEvent) => {
      const rect = container.getBoundingClientRect()
      const next = Math.round(rect.right - ev.clientX)
      const max = Math.max(PANEL_MIN, rect.width - LIST_MIN)
      setPanelWidth(Math.min(max, Math.max(PANEL_MIN, next)))
    }
    const onUp = () => {
      setIsDraggingSplit(false)
      window.removeEventListener('pointermove', onMove)
      window.removeEventListener('pointerup', onUp)
      setPanelWidth(current => {
        try {
          localStorage.setItem('taskacao_roadmap_panel_width', String(current))
        } catch {
          // stockage indisponible : la largeur vaut pour cette session
        }
        return current
      })
    }
    window.addEventListener('pointermove', onMove)
    window.addEventListener('pointerup', onUp)
  }

  useEffect(() => {
    if (!currentProject?.id) {
      setEpicMeta([])
      return
    }
    fetchProjectEpics(currentProject.id).then(setEpicMeta)
    pendingHorizonPushes(currentProject.id).then(list => setPendingPushes(list.length))
    // activeJobCount en dépendance : les écritures Jira (rattachement, découpe,
    // labels d'horizon) passent par la file, donc la liste des épics n'est à
    // jour qu'une fois la file vidée.
  }, [currentProject?.id, fetchProjectEpics, pendingHorizonPushes, activeJobCount])

  const allRows = useMemo(() => buildEpicRows(tasks, currentProject, epicMeta), [tasks, currentProject, epicMeta])

  const projectKey = (currentProject?.jiraProject || '').trim().toUpperCase()

  const rows = useMemo(() => {
    let list = allRows
    if (!showClosed) list = list.filter(r => !r.closed)
    if (onlyProjectKey && projectKey) list = list.filter(r => belongsToProjectKey(r, projectKey))
    return list
  }, [allRows, showClosed, onlyProjectKey, projectKey])

  const closedCount = allRows.filter(r => r.closed).length
  const foreignCount = projectKey ? allRows.filter(r => !belongsToProjectKey(r, projectKey)).length : 0

  const counts = useMemo(
    () => ({
      now: rows.filter(r => r.horizon === 'now').length,
      next: rows.filter(r => r.horizon === 'next').length,
      later: rows.filter(r => r.horizon === 'later').length,
      unclassified: rows.filter(r => !r.horizon).length,
      hidden: rows.filter(r => r.horizon === 'hidden').length,
    }),
    [rows]
  )

  const operational = tab === 'now' || tab === 'next'
  // Les onglets « non classés » et « masqués » n'ont pas d'horizon propre : le
  // panneau y montre le cadrage, pas la vérification de sprint.
  const horizonOfTab: Horizon =
    tab === 'next' ? 'next' : tab === 'later' ? 'later' : tab === 'hidden' ? 'hidden' : 'now'

  const visibleRows = useMemo(() => {
    const list = tab === 'unclassified' ? rows.filter(r => !r.horizon) : rows.filter(r => r.horizon === tab)
    if (operational && onlyIssues) {
      return list.filter(r => placementIssues(r, horizonOfTab).length > 0)
    }
    return list
  }, [rows, tab, operational, onlyIssues, horizonOfTab])

  useEffect(() => {
    if (!currentProject?.id) {
      setEpicFields([])
      return
    }
    fetchEpicRequiredFields(currentProject.id).then(fields => {
      setEpicFields(fields)
      // Le dernier choix est réutilisé : le champ est métier, mais on ne va pas
      // le redemander à chaque épic d'une même session de découpe.
      const storeKey = `taskacao_epic_fields_${currentProject.id}`
      let stored: Record<string, string> = {}
      try {
        stored = JSON.parse(localStorage.getItem(storeKey) || '{}') || {}
      } catch {
        stored = {}
      }
      const next: Record<string, string> = {}
      fields.forEach(f => {
        if (stored[f.id] && f.options.some(o => o.id === stored[f.id])) next[f.id] = stored[f.id]
      })
      setEpicFieldValues(next)
    })
  }, [currentProject?.id, fetchEpicRequiredFields])

  const setEpicFieldValue = (fieldId: string, optionId: string) => {
    const next = { ...epicFieldValues, [fieldId]: optionId }
    setEpicFieldValues(next)
    if (currentProject?.id) {
      try {
        localStorage.setItem(`taskacao_epic_fields_${currentProject.id}`, JSON.stringify(next))
      } catch {
        // stockage indisponible : le choix vaut pour cette session seulement
      }
    }
  }

  // Tous les champs imposés sont-ils renseignés ? Sinon la création échouera
  // côté Jira, autant désactiver le bouton et le dire.
  const missingEpicField = epicFields.find(f => !epicFieldValues[f.id])

  // Les champs imposés par l'instance, rendus partout où un épic peut naître.
  const epicFieldSelectors =
    epicFields.length === 0 ? null : (
      <div className="flex items-center gap-1.5 flex-wrap">
        {epicFields.map(field => (
          <label key={field.id} className="flex items-center gap-1 text-[10px] text-[var(--text-muted)]">
            <span>{field.name}</span>
            <select
              value={epicFieldValues[field.id] || ''}
              onChange={e => setEpicFieldValue(field.id, e.target.value)}
              className="px-1.5 py-1 text-[10px] rounded-lg bg-[var(--bg-primary)] border text-[var(--text-primary)] focus:outline-none cursor-pointer"
              style={{
                borderColor: epicFieldValues[field.id] ? 'var(--border-color)' : 'var(--status-danger)',
              }}
            >
              <option value="">— à choisir —</option>
              {field.options.map(opt => (
                <option key={opt.id} value={opt.id}>{opt.value}</option>
              ))}
            </select>
          </label>
        ))}
      </div>
    )

  const selected: EpicRow | null = visibleRows.find(r => r.key === selectedKey) || visibleRows[0] || null

  // Les tickets de l'épic dans l'ordre chronologique de leur sprint : c'est cet
  // ordre que la liste affiche et que le curseur de coupe découpe.
  const orderedOpen = useMemo(
    () => (selected ? tasksBySprintOrder(selected.open, currentProject) : []),
    [selected?.key, selected?.open, currentProject?.id, currentProject?.sprints]
  )

  // Glissement du cran de coupe : on cherche la frontière de ligne la plus proche
  // du pointeur, ce qui évite tout calcul de pas et supporte des lignes de
  // hauteurs différentes.
  useEffect(() => {
    if (!draggingCut) return

    const onMove = (e: PointerEvent) => {
      // Bornée à la liste courante : le tableau de refs garde des entrées d'un
      // épic précédent plus long, qui décaleraient la frontière trouvée.
      const rows = rowRefs.current.slice(0, orderedOpen.length).filter(Boolean) as HTMLDivElement[]
      if (rows.length === 0) return

      let best = 0
      let bestDistance = Number.POSITIVE_INFINITY
      rows.forEach((row, index) => {
        const rect = row.getBoundingClientRect()
        const distance = Math.abs(rect.top - e.clientY)
        if (distance < bestDistance) {
          bestDistance = distance
          best = index
        }
      })

      // Sous la dernière ligne, la coupe n'a plus d'objet : on la relâche.
      const last = rows[rows.length - 1].getBoundingClientRect()
      if (e.clientY > last.bottom) {
        setCutAt(-1)
        return
      }
      setCutAt(best)
    }

    const onUp = () => setDraggingCut(false)
    window.addEventListener('pointermove', onMove)
    window.addEventListener('pointerup', onUp)
    return () => {
      window.removeEventListener('pointermove', onMove)
      window.removeEventListener('pointerup', onUp)
    }
  }, [draggingCut, orderedOpen.length])

  // Les tickets à couper : ceux cochés à la main, ou ceux situés au delà du
  // curseur. Les deux gestes alimentent la même action.
  const cutIds = useMemo(() => {
    const manual = Object.entries(checked).filter(([, on]) => on).map(([id]) => id)
    if (manual.length > 0) return manual
    if (cutAt >= 0) return orderedOpen.slice(cutAt).map(t => t.id)
    return []
  }, [checked, cutAt, orderedOpen])

  useEffect(() => {
    setDraftDescription(selected?.meta?.description || '')
    setDraftDirty(false)
    setNewTodo('')
    setChecked({})
    setCutAt(-1)
    rowRefs.current = []
    setMoveTarget('')
    setNewEpicTitle('')
  }, [selected?.key, selected?.meta?.description])

  const attachCandidates = useMemo(() => {
    const q = attachQuery.trim().toLowerCase()
    if (!q || !selected) return []
    return tasks
      .filter(t => t.parentKey !== selected.key)
      .filter(t => t.key.toLowerCase().includes(q) || t.title.toLowerCase().includes(q))
      .slice(0, 6)
  }, [attachQuery, tasks, selected])

  const jiraBase = (currentProject?.trackerUrl || settings.jiraUrl || '').replace(/\/+$/, '')
  const epicUrl = (key: string) => (jiraBase ? `${jiraBase}/browse/${key}` : undefined)

  const activeSprints = (currentProject?.sprints || []).filter(s => s.state === 'active')
  const futureSprints = (currentProject?.sprints || []).filter(s => s.state === 'future')

  // Le compteur « à pousser » interroge Jira : on ne le rafraîchit pas à chaque
  // clic, mais une fois le triage retombé.
  const schedulePendingRefresh = () => {
    if (pendingTimer.current) window.clearTimeout(pendingTimer.current)
    pendingTimer.current = window.setTimeout(() => {
      if (!currentProject?.id) return
      pendingHorizonPushes(currentProject.id).then(list => setPendingPushes(list.length))
    }, 4000)
  }

  const persist = async (
    key: string,
    patch: { horizon?: EpicHorizon | ''; description?: string; todos?: EpicTodo[] }
  ) => {
    if (!currentProject?.id) {
      addToast({
        type: 'error',
        title: 'Aucun projet sélectionné',
        description: 'Choisis un projet pour classer ses épics.',
      })
      return
    }
    const saved = await saveEpicMeta(currentProject.id, key, patch)
    if (saved) {
      setEpicMeta(prev => [...prev.filter(m => m.key !== saved.key), saved])
      if (patch.horizon !== undefined) schedulePendingRefresh()
    }
  }

  const todosOf = (row: EpicRow | null): EpicTodo[] => row?.meta?.todos || []

  const addTodo = (row: EpicRow) => {
    const text = newTodo.trim()
    if (!text) return
    setNewTodo('')
    persist(row.key, { todos: [...todosOf(row), { id: '', text, done: false }] })
  }

  return (
    <div className="flex-1 flex flex-col min-h-0 overflow-hidden bg-[var(--bg-primary)] text-[var(--text-primary)]">
      {/* Barre d'outils : classification, et en mode opérationnel les sprints visés */}
      <div className="flex items-center justify-between gap-3 flex-wrap px-4 py-2 border-b border-[var(--border-color)] bg-[var(--bg-secondary)]/50 shrink-0">
        <div className="flex items-center gap-3 flex-wrap min-w-0">
          <div className="flex items-center p-0.5 rounded-lg bg-[var(--bg-tertiary)] border border-[var(--border-color)]">
            {TABS.map(t => {
              const active = tab === t.id
              const meta = t.id === 'unclassified' ? null : HORIZON_META[t.id as Horizon]
              return (
                <button
                  key={t.id}
                  type="button"
                  onClick={() => setTab(t.id)}
                  className="flex items-center gap-1.5 px-2.5 py-1 rounded-md text-[11px] transition-colors cursor-pointer"
                  style={{
                    fontWeight: active ? 700 : 500,
                    background: active ? meta?.color || 'var(--text-muted)' : 'transparent',
                    color: active ? '#fff' : 'var(--text-secondary)',
                  }}
                  title={meta ? meta.hint : 'Épics pas encore arbitrés'}
                >
                  {t.icon}
                  <span>{t.label}</span>
                  <span
                    className="text-[10px] font-mono px-1.5 rounded-full"
                    style={{
                      background: active ? 'rgb(255 255 255 / 0.25)' : 'var(--bg-secondary)',
                      color: active ? '#fff' : 'var(--text-muted)',
                    }}
                  >
                    {counts[t.id]}
                  </span>
                </button>
              )
            })}
          </div>

          {pendingPushes > 0 && (
            <button
              type="button"
              disabled={isPushing}
              onClick={async () => {
                if (!currentProject?.id) return
                setIsPushing(true)
                await pushPendingHorizons(currentProject.id)
                const rest = await pendingHorizonPushes(currentProject.id)
                setPendingPushes(rest.length)
                setIsPushing(false)
              }}
              className="flex items-center gap-1.5 px-2.5 py-1 rounded-md text-[11px] font-semibold cursor-pointer border disabled:opacity-50"
              style={{
                background: 'rgb(var(--status-warn-rgb) / 0.14)',
                borderColor: 'rgb(var(--status-warn-rgb) / 0.34)',
                color: 'var(--status-warn)',
              }}
              title="Ces épics sont classés dans Taskacao mais leur épic Jira ne porte pas encore le label roadmap"
            >
              {isPushing ? 'Poussée…' : `${pendingPushes} à pousser vers Jira`}
            </button>
          )}

          {closedCount > 0 && (
            <button
              type="button"
              onClick={() => setShowClosed(v => !v)}
              className="flex items-center gap-1.5 px-2.5 py-1 rounded-md text-[11px] font-semibold cursor-pointer border"
              style={{
                background: showClosed ? 'var(--accent-light)' : 'var(--bg-tertiary)',
                borderColor: showClosed ? 'rgb(var(--accent-rgb) / 0.4)' : 'var(--border-color)',
                color: showClosed ? 'var(--accent-color)' : 'var(--text-secondary)',
              }}
              title={`${closedCount} épics terminés côté tracker`}
            >
              {showClosed ? <Eye size={12} /> : <EyeOff size={12} />}
              {showClosed ? `${closedCount} terminés affichés` : `${closedCount} terminés masqués`}
            </button>
          )}

          {projectKey && foreignCount > 0 && (
            <button
              type="button"
              onClick={() => setOnlyProjectKey(v => !v)}
              className="flex items-center gap-1.5 px-2.5 py-1 rounded-md text-[11px] font-semibold cursor-pointer border font-mono"
              style={{
                background: onlyProjectKey ? 'rgb(var(--status-info-rgb) / 0.12)' : 'var(--bg-tertiary)',
                borderColor: onlyProjectKey ? 'rgb(var(--status-info-rgb) / 0.32)' : 'var(--border-color)',
                color: onlyProjectKey ? 'var(--status-info)' : 'var(--text-secondary)',
              }}
              title={`${foreignCount} épics venant d'un autre projet Jira, portés par des tickets ${projectKey}`}
            >
              {onlyProjectKey ? `${projectKey}- uniquement` : `tous projets (+${foreignCount})`}
            </button>
          )}

          {/* Les filtres globaux amputent aussi cette vue, puisqu'elle est bâtie sur
              la même liste de tickets que le board. Sans les afficher ici, un
              filtre resté actif ailleurs fait « disparaître » des tickets sans
              explication : cette vue n'a pas de barre de filtres à elle. */}
          {activeFilterChips.length > 0 && (
            <div className="flex items-center gap-1.5 flex-wrap">
              {activeFilterChips.map(chip => (
                <button
                  key={chip.label}
                  type="button"
                  onClick={chip.clear}
                  className="flex items-center gap-1 px-2 py-1 rounded-md text-[10.5px] font-semibold cursor-pointer border"
                  style={{
                    background: 'rgb(var(--status-warn-rgb) / 0.14)',
                    borderColor: 'rgb(var(--status-warn-rgb) / 0.34)',
                    color: 'var(--status-warn)',
                  }}
                  title={`Filtre actif : ${chip.label}. Cliquer pour l'enlever.`}
                >
                  <Filter size={10} />
                  {chip.label}
                  <X size={10} />
                </button>
              ))}
            </div>
          )}

          {epicFieldSelectors}

          <button
            type="button"
            disabled={busyKey === 'epic' || !currentProject?.id || Boolean(missingEpicField)}
            onClick={async () => {
              const title = window.prompt('Titre du nouvel épic ?')
              if (!title?.trim() || !currentProject?.id) return
              setBusyKey('epic')
              const created = await createEpic(
                currentProject.id,
                title.trim(),
                tab === 'unclassified' || tab === 'hidden' ? '' : (tab as EpicHorizon),
                epicFieldValues
              )
              if (created) {
                setEpicMeta(prev => [...prev.filter(m => m.key !== created.key), created])
                setSelectedKey(created.key)
              }
              setBusyKey(null)
            }}
            className="flex items-center gap-1.5 px-2.5 py-1 rounded-md text-[11px] font-bold text-white accent-bg cursor-pointer disabled:opacity-40"
            title={
              missingEpicField
                ? `${missingEpicField.name} est obligatoire sur ce projet : choisis une valeur d'abord`
                : 'Créer un épic vide, utilisable comme cible pour découper un épic trop gros'
            }
          >
            <Plus size={12} /> Épic
          </button>

          {operational && (
            <button
              type="button"
              onClick={() => setOnlyIssues(v => !v)}
              className="flex items-center gap-1.5 px-2.5 py-1 rounded-md text-[11px] font-semibold transition-colors cursor-pointer border"
              style={{
                background: onlyIssues ? 'rgb(var(--status-danger-rgb) / 0.14)' : 'var(--bg-tertiary)',
                borderColor: onlyIssues ? 'rgb(var(--status-danger-rgb) / 0.4)' : 'var(--border-color)',
                color: onlyIssues ? 'var(--status-danger)' : 'var(--text-secondary)',
              }}
              title="Ne garder que les épics dont une story n'est pas dans le bon sprint"
            >
              <AlertTriangle size={12} />
              À corriger
            </button>
          )}
        </div>

        {operational && (
          <div className="flex items-center gap-1.5 px-2.5 py-1 rounded-lg text-[11px] bg-[var(--bg-tertiary)] border border-[var(--border-color)] shrink-0">
            <CalendarDays size={13} style={{ color: HORIZON_META[horizonOfTab].color }} />
            <span className="text-[var(--text-muted)]">{tab === 'now' ? 'Sprints actifs' : 'Sprints à venir'} :</span>
            {(tab === 'now' ? activeSprints : futureSprints).length === 0 ? (
              <span className="font-mono" style={{ color: 'var(--status-warn)' }}>aucun connu — lance une synchro</span>
            ) : (
              <span className="font-mono font-bold truncate max-w-[320px]">
                {(tab === 'now' ? activeSprints : futureSprints).map(s => s.name).join(' · ')}
              </span>
            )}
          </div>
        )}
      </div>

      <div className="flex-1 flex min-h-0 min-w-0 overflow-hidden" ref={splitRef}>
        {/* Épics de l'horizon courant */}
        <div className="flex-1 overflow-y-auto p-3 min-w-0 space-y-2">
          {visibleRows.length === 0 ? (
            <div className="h-full flex flex-col items-center justify-center gap-2 text-center px-6">
              <Compass size={26} className="text-[var(--text-muted)]" />
              <p className="text-sm font-semibold">Aucun épic ici</p>
              <p className="text-[11px] text-[var(--text-muted)] max-w-sm">
                {tab === 'hidden'
                  ? 'Aucun épic masqué. Classe en HIDDEN le tout-venant qui n’a pas vocation à apparaître dans la roadmap.'
                  : tab === 'unclassified'
                  ? 'Tous les épics sont classés. Les nouveaux apparaîtront ici après une synchro.'
                  : onlyIssues
                    ? 'Aucune anomalie de placement sur cet horizon.'
                    : 'Classe des épics depuis l’onglet « Non classés » pour les voir apparaître ici.'}
              </p>
            </div>
          ) : (
            visibleRows.map(row => {
              const isSel = selected?.key === row.key
              const issues = placementIssues(row, horizonOfTab)
              const mat = MATURITY_META[row.maturity]
              const prio = PRIORITY_META[row.priority]
              return (
                <div
                  key={row.key}
                  onClick={() => setSelectedKey(row.key)}
                  className="rounded-xl border p-2.5 cursor-pointer transition-colors"
                  style={{
                    background: isSel ? 'var(--accent-light)' : 'var(--bg-secondary)',
                    borderColor: isSel ? 'rgb(var(--accent-rgb) / 0.45)' : 'var(--border-color)',
                  }}
                >
                  <div className="flex items-center gap-2 flex-wrap">
                    <span className="text-[11px] font-mono font-bold" style={{ color: 'var(--accent-color)' }}>{row.key}</span>
                    <span className="text-[9.5px] px-1 rounded font-mono truncate max-w-[150px] bg-[var(--bg-tertiary)] text-[var(--text-muted)] border border-[var(--border-color)]" title={row.squad}>
                      {row.squad}
                    </span>
                    <span className="text-[9.5px] px-1 rounded font-bold" style={{ color: prio.color, background: prio.bg }}>
                      {prio.label}
                    </span>
                    <span className="text-[9px] font-bold px-1.5 rounded uppercase tracking-[.06em]"
                      style={{ color: mat.color, background: mat.bg, border: `1px solid ${mat.border}` }}>
                      {row.maturity}
                    </span>

                    {operational && (
                      issues.length > 0 ? (
                        <span className="ml-auto text-[10px] font-bold px-1.5 py-0.5 rounded inline-flex items-center gap-1"
                          style={{ color: 'var(--status-danger)', background: 'rgb(var(--status-danger-rgb) / 0.13)', border: '1px solid rgb(var(--status-danger-rgb) / 0.32)' }}>
                          <AlertTriangle size={10} />
                          {issues.length} à corriger
                        </span>
                      ) : (
                        <span className="ml-auto text-[10px] font-bold px-1.5 py-0.5 rounded inline-flex items-center gap-1"
                          style={{ color: 'var(--status-ok)', background: 'rgb(var(--status-ok-rgb) / 0.13)', border: '1px solid rgb(var(--status-ok-rgb) / 0.32)' }}>
                          <Check size={10} />
                          tout placé
                        </span>
                      )
                    )}

                    {tab === 'later' && (
                      <span className="ml-auto text-[10px] font-mono text-[var(--text-muted)]">
                        {todosOf(row).filter(t => t.done).length}/{todosOf(row).length} todos
                      </span>
                    )}
                  </div>

                  <div className="text-[12px] font-semibold leading-snug mt-1.5">{row.title}</div>

                  <div className="flex items-center gap-2 mt-2 flex-wrap text-[10px] font-mono text-[var(--text-muted)]">
                    <span>{row.open.length} ouverts / {row.tasks.length}</span>
                    {row.inActiveSprint.length > 0 && (
                      <span style={{ color: 'var(--status-ok)' }}>{row.inActiveSprint.length} sprint actif</span>
                    )}
                    {row.inFutureSprint.length > 0 && (
                      <span style={{ color: 'var(--status-info)' }}>{row.inFutureSprint.length} sprint futur</span>
                    )}
                    {row.inStaleSprint.length > 0 && (
                      <span style={{ color: 'var(--status-warn)' }}>{row.inStaleSprint.length} sprint clos</span>
                    )}
                    {row.unscheduled.length > 0 && (
                      <span style={{ color: 'var(--status-danger)' }}>{row.unscheduled.length} sans sprint</span>
                    )}
                  </div>

                  {/* Classification : un clic, et la suggestion est mise en avant */}
                  <div className="flex items-center gap-1.5 mt-2">
                    {(['now', 'next', 'later', 'hidden'] as EpicHorizon[]).map(h => {
                      const active = row.horizon === h
                      const isSuggestion = !row.horizon && row.suggested === h
                      return (
                        <button
                          key={h}
                          type="button"
                          onClick={e => {
                            e.stopPropagation()
                            persist(row.key, { horizon: h })
                          }}
                          className="px-1.5 py-0.5 rounded text-[9.5px] font-bold uppercase tracking-[.06em] border transition-colors cursor-pointer"
                          style={{
                            color: active ? '#fff' : HORIZON_META[h].color,
                            background: active ? HORIZON_META[h].color : isSuggestion ? HORIZON_META[h].bg : 'transparent',
                            borderColor: active || isSuggestion ? HORIZON_META[h].border : 'var(--border-color)',
                          }}
                          title={isSuggestion ? `Suggéré d'après les sprints : ${HORIZON_META[h].label}` : `Classer en ${HORIZON_META[h].label}`}
                        >
                          {HORIZON_META[h].label}
                          {isSuggestion && ' ?'}
                        </button>
                      )
                    })}
                  </div>
                </div>
              )
            })
          )}
        </div>

        {/* Poignée de répartition : la liste des épics n'a pas besoin de la moitié
            de l'écran, le panneau de travail oui. */}
        {selected && (
          <div
            role="separator"
            aria-orientation="vertical"
            aria-label="Répartition entre la liste des épics et le panneau"
            onPointerDown={startSplitDrag}
            onDoubleClick={() => setPanelWidth(720)}
            title="Glisser pour répartir, double-clic pour revenir à la largeur par défaut"
            className="w-1.5 shrink-0 cursor-col-resize transition-colors"
            style={{ background: isDraggingSplit ? 'var(--accent-color)' : 'var(--border-color)' }}
          />
        )}

        {/* Panneau : vérification des sprints en NOW/NEXT, cadrage en LATER */}
        {selected && (
          <aside className="flex flex-col min-h-0 shrink-0 bg-[var(--bg-secondary)]"
            style={{ width: panelWidth }}>
            <div className="px-4 pt-3.5 pb-3 shrink-0 border-b border-[var(--border-color)]">
              <div className="flex items-center gap-2 mb-1.5 flex-wrap">
                <span className="text-[11px] font-mono font-bold" style={{ color: 'var(--accent-color)' }}>{selected.key}</span>
                {selected.horizon && (
                  <span className="text-[9px] font-bold px-1.5 rounded-full uppercase tracking-[.08em]"
                    style={{
                      color: HORIZON_META[selected.horizon].color,
                      background: HORIZON_META[selected.horizon].bg,
                      border: `1px solid ${HORIZON_META[selected.horizon].border}`,
                    }}>
                    {HORIZON_META[selected.horizon].label}
                  </span>
                )}
                {epicUrl(selected.key) && (
                  <a href={epicUrl(selected.key)} target="_blank" rel="noreferrer"
                    className="ml-auto inline-flex items-center gap-1 px-2.5 py-1 rounded-lg text-xs font-semibold transition-colors"
                    style={{
                      color: 'var(--status-info)',
                      background: 'rgb(var(--status-info-rgb) / 0.12)',
                      border: '1px solid rgb(var(--status-info-rgb) / 0.32)',
                    }}
                    title={`Ouvrir ${selected.key} dans Jira`}>
                    <ExternalLink size={13} />
                    <span>Jira</span>
                  </a>
                )}
              </div>
              <h2 className="text-[16px] font-bold leading-[1.25]">{selected.title}</h2>
              {selected.meta?.status && (
                <div className="mt-1.5 text-[10px] font-mono" style={{ color: selected.closed ? 'var(--status-ok)' : 'var(--text-muted)' }}>
                  Épic {selected.closed ? 'terminé' : 'ouvert'} · {selected.meta.status}
                </div>
              )}
            </div>

            <div className="flex-1 overflow-y-auto px-4 pt-3.5 pb-7 flex flex-col gap-4">
              {operational ? (
                <>
                  {/* Prototypage : ajouter une story a la volée, ou pousser un
                      ticket existant dans l'épic. */}
                  <div>
                    <div className="text-[10px] font-bold uppercase tracking-[.08em] text-[var(--text-muted)] mb-1.5">
                      Composer l'épic
                    </div>

                    <div className="flex items-center gap-2">
                      <input
                        type="text"
                        value={newStory}
                        onChange={e => setNewStory(e.target.value)}
                        onKeyDown={async e => {
                          if (e.key === 'Enter' && newStory.trim() && currentProject?.id) {
                            e.preventDefault()
                            setBusyKey('new')
                            await createStoryUnderEpic(currentProject.id, selected.key, newStory.trim())
                            setNewStory('')
                            setBusyKey(null)
                          }
                        }}
                        placeholder={`Nouvelle story sous ${selected.key}…`}
                        className="flex-1 px-2.5 py-1.5 text-xs rounded-xl bg-[var(--bg-primary)] border border-[var(--border-color)] text-[var(--text-primary)] focus:outline-none focus:border-[var(--accent-color)]"
                      />
                      <button
                        type="button"
                        disabled={!newStory.trim() || busyKey === 'new' || !currentProject?.id}
                        onClick={async () => {
                          setBusyKey('new')
                          await createStoryUnderEpic(currentProject!.id, selected.key, newStory.trim())
                          setNewStory('')
                          setBusyKey(null)
                        }}
                        className="flex items-center gap-1 px-2.5 py-1.5 rounded-xl text-xs font-bold text-white accent-bg disabled:opacity-40 cursor-pointer shrink-0"
                      >
                        <Plus size={12} /> {busyKey === 'new' ? '…' : 'Créer'}
                      </button>
                    </div>

                    <div className="mt-2">
                      <input
                        type="text"
                        value={attachQuery}
                        onChange={e => setAttachQuery(e.target.value)}
                        placeholder="Pousser un ticket existant : clé ou titre…"
                        className="w-full px-2.5 py-1.5 text-xs rounded-xl bg-[var(--bg-primary)] border border-[var(--border-color)] text-[var(--text-primary)] focus:outline-none focus:border-[var(--accent-color)]"
                      />
                      {attachCandidates.length > 0 && (
                        <div className="mt-1.5 flex flex-col gap-1">
                          {attachCandidates.map(candidate => (
                            <button
                              key={candidate.id}
                              type="button"
                              disabled={busyKey === candidate.id}
                              onClick={async () => {
                                setBusyKey(candidate.id)
                                await setTaskEpic(candidate.id, selected.key)
                                setBusyKey(null)
                                setAttachQuery('')
                              }}
                              className="flex items-center gap-2 px-2 py-1.5 rounded-lg text-left bg-[var(--bg-primary)] border border-[var(--border-color)] hover:border-[var(--accent-color)]/50 cursor-pointer disabled:opacity-50"
                            >
                              <span className="text-[10px] font-mono font-bold shrink-0" style={{ color: 'var(--status-info)' }}>
                                {candidate.key}
                              </span>
                              <span className="text-[11px] truncate flex-1 text-[var(--text-secondary)]">{candidate.title}</span>
                              {candidate.parentKey && (
                                <span className="text-[9px] font-mono shrink-0 text-[var(--text-muted)]" title="Épic actuel, qui sera remplacé">
                                  {candidate.parentKey} →
                                </span>
                              )}
                              <Plus size={11} className="shrink-0 text-[var(--text-muted)]" />
                            </button>
                          ))}
                        </div>
                      )}
                    </div>
                  </div>

                  <div className="grid grid-cols-2 gap-2">
                    <div className="px-2.5 py-2 rounded-lg bg-[var(--bg-primary)] border border-[var(--border-color)]">
                      <div className="text-[9px] font-bold uppercase tracking-[.08em] text-[var(--text-muted)]">
                        {tab === 'now' ? 'Dans un sprint actif' : 'Dans un sprint à venir'}
                      </div>
                      <div className="text-[15px] font-bold mt-0.5" style={{ color: 'var(--status-ok)' }}>
                        {tab === 'now' ? selected.inActiveSprint.length : selected.inFutureSprint.length}
                        <span className="text-[11px] font-normal text-[var(--text-muted)]"> / {selected.open.length} ouverts</span>
                      </div>
                    </div>
                    <div className="px-2.5 py-2 rounded-lg bg-[var(--bg-primary)] border border-[var(--border-color)]">
                      <div className="text-[9px] font-bold uppercase tracking-[.08em] text-[var(--text-muted)]">À corriger</div>
                      <div className="text-[15px] font-bold mt-0.5"
                        style={{ color: placementIssues(selected, horizonOfTab).length ? 'var(--status-danger)' : 'var(--status-ok)' }}>
                        {placementIssues(selected, horizonOfTab).length}
                      </div>
                    </div>
                  </div>

                  <div>
                    <div className="text-[10px] font-bold uppercase tracking-[.08em] text-[var(--text-muted)] mb-1.5">
                      Stories et sprints ({selected.open.length} ouvertes)
                    </div>
                    <div className="flex flex-col gap-1.5">
                      {selected.open.length === 0 && (
                        <p className="text-[11px] text-[var(--text-muted)]">
                          Aucune story ouverte : cet épic ressemble plutôt à du LATER.
                        </p>
                      )}
                      {orderedOpen.map((task, index) => {
                        const state = placementOf(task, selected, horizonOfTab)
                        const meta = PLACEMENT_META[state]
                        const beyondCut = cutAt >= 0 && index >= cutAt
                        const sprintLabel = sprintLabelOf(task)
                        const startsSprint = index === 0 || sprintLabelOf(orderedOpen[index - 1]) !== sprintLabel
                        return (
                          <React.Fragment key={task.id}>
                            {startsSprint && (
                              <div className="flex items-center gap-1.5 mt-1 first:mt-0 pl-[20px]">
                                <span className="text-[9px] font-bold uppercase tracking-[.08em] text-[var(--text-muted)]">
                                  {sprintLabel}
                                </span>
                                <span className="flex-1 h-px bg-[var(--border-color)]" />
                              </div>
                            )}
                          <div
                            ref={el => { rowRefs.current[index] = el }}
                            className="grid gap-1.5"
                            style={{ gridTemplateColumns: '14px 1fr' }}
                          >
                            {/* Gouttière de coupe : le cran est à la frontière
                                haute de la ligne, donc l'alignement suit les
                                lignes quel que soit leur hauteur. */}
                            <div className="relative">
                              <span
                                className="absolute left-1/2 top-0 bottom-0 w-px -translate-x-1/2"
                                style={{ background: beyondCut ? 'var(--accent-color)' : 'var(--border-color)' }}
                              />
                              <button
                                type="button"
                                onPointerDown={e => {
                                  e.preventDefault()
                                  setCutAt(index)
                                  setDraggingCut(true)
                                }}
                                onKeyDown={e => {
                                  if (e.key === 'ArrowUp') { e.preventDefault(); setCutAt(Math.max(0, index - 1)) }
                                  if (e.key === 'ArrowDown') {
                                    e.preventDefault()
                                    const next = index + 1
                                    setCutAt(next >= orderedOpen.length ? -1 : next)
                                  }
                                }}
                                className="absolute left-1/2 -translate-x-1/2 -translate-y-1/2 rounded-full cursor-ns-resize"
                                style={{
                                  top: 0,
                                  width: cutAt === index ? 11 : 7,
                                  height: cutAt === index ? 11 : 7,
                                  background: cutAt === index ? 'var(--accent-color)' : 'var(--bg-secondary)',
                                  border: `1px solid ${cutAt === index ? 'var(--accent-color)' : 'var(--border-color)'}`,
                                }}
                                title={`Couper ici : ${orderedOpen.length - index} ticket(s) partent`}
                                aria-label={`Couper avant ${task.key}`}
                              />
                            </div>

                          <div className="px-2.5 py-2 rounded-lg bg-[var(--bg-primary)] border"
                            style={{
                              borderColor: beyondCut
                                ? 'var(--accent-color)'
                                : state === 'ok'
                                  ? 'var(--border-color)'
                                  : meta.border,
                              opacity: cutAt >= 0 && !beyondCut ? 0.55 : 1,
                            }}>
                            <div className="flex items-center gap-2">
                              <button
                                type="button"
                                onClick={() => setChecked(prev => ({ ...prev, [task.id]: !prev[task.id] }))}
                                className="w-3.5 h-3.5 rounded shrink-0 flex items-center justify-center cursor-pointer"
                                style={{
                                  background: checked[task.id] ? 'var(--accent-color)' : 'transparent',
                                  border: `1px solid ${checked[task.id] ? 'var(--accent-color)' : 'var(--border-color)'}`,
                                }}
                                title="Sélectionner pour déplacer vers un autre épic"
                              >
                                {checked[task.id] && <Check size={10} className="text-white" />}
                              </button>
                              <button type="button" onClick={() => setSelectedTask(task)}
                                className="text-[10.5px] font-mono font-bold hover:underline cursor-pointer"
                                style={{ color: 'var(--status-info)' }}>
                                {task.key}
                              </button>
                              {sprintOptions.length > 0 ? (
                                <div className="ml-auto shrink-0 w-[165px]">
                                  <LookupField
                                    value={task.sprint || ''}
                                    icon={<CalendarRange size={10} />}
                                    placeholder={meta.label}
                                    clearLabel="Backlog (aucun sprint)"
                                    onSearch={searchSprint}
                                    onPick={option => setTaskSprint(task.id, option?.id || '', option?.label)}
                                  />
                                </div>
                              ) : (
                                <span className="text-[9px] px-1 rounded font-mono ml-auto shrink-0 truncate max-w-[150px]"
                                  style={{ color: meta.color, background: meta.bg, border: `1px solid ${meta.border}` }}>
                                  {task.sprint || meta.label}
                                </span>
                              )}
                              <button type="button" onClick={() => setChatTask(task)}
                                className="p-0.5 rounded text-[var(--text-muted)] hover:text-cyan-300 cursor-pointer shrink-0"
                                title={`Terminal de ${task.key}`}>
                                <TerminalIcon size={12} />
                              </button>
                              <button
                                type="button"
                                disabled={busyKey === task.id}
                                onClick={async () => {
                                  setBusyKey(task.id)
                                  await setTaskEpic(task.id, '')
                                  setBusyKey(null)
                                }}
                                className="p-0.5 rounded text-[var(--text-muted)] hover:text-rose-400 cursor-pointer shrink-0 disabled:opacity-50"
                                title={`Retirer ${task.key} de l'épic`}>
                                <X size={12} />
                              </button>
                            </div>
                            <div className="text-[11px] mt-1 leading-snug text-[var(--text-secondary)]">{task.title}</div>
                            {state !== 'ok' && (
                              <div className="text-[9.5px] mt-1 font-mono" style={{ color: meta.color }}>
                                {state === 'missing' && 'Aucun sprint : à placer'}
                                {state === 'stale' && 'Sprint clos ou inconnu du board'}
                                {state === 'other-horizon' &&
                                  (tab === 'now' ? 'Dans un sprint futur, pas actif' : 'Dans un sprint actif, pas futur')}
                              </div>
                            )}
                          </div>
                          </div>
                          </React.Fragment>
                        )
                      })}
                    </div>
                  </div>

                  {/* Résumé de la coupe. Le curseur lui-même est vertical, dans
                      la gouttière de la liste : couper se lit alors comme une
                      ligne tracée entre deux tickets, et non comme un rail
                      horizontal détaché de ce qu'il découpe. */}
                  {orderedOpen.length > 1 && (
                    <div className="flex items-center gap-2 px-2.5 py-1.5 rounded-xl bg-[var(--bg-primary)] border border-[var(--border-color)]">
                      <Scissors size={11} style={{ color: cutAt >= 0 ? 'var(--accent-color)' : 'var(--text-muted)' }} />
                      <span className="text-[10px] text-[var(--text-secondary)]">
                        {cutAt < 0
                          ? 'Fais glisser un cran dans la marge pour couper'
                          : `${orderedOpen.length - cutAt} ticket(s) à partir de ${sprintLabelOf(orderedOpen[cutAt])}`}
                      </span>
                      {cutAt >= 0 && (
                        <button
                          type="button"
                          onClick={() => setCutAt(-1)}
                          className="ml-auto text-[10px] font-bold text-[var(--text-muted)] hover:text-[var(--text-primary)] cursor-pointer"
                        >
                          Annuler la coupe
                        </button>
                      )}
                    </div>
                  )}

                  {/* Planification : la même sélection sert à replanifier, ce qui
                      est l'autre moitié du travail sur un épic trop gros. Couper
                      change de contenant, changer de sprint change la date. */}
                  {cutIds.length > 0 && sprintOptions.length > 0 && (
                    <div className="p-2.5 rounded-xl border border-[var(--border-color)] bg-[var(--bg-primary)]">
                      <div className="text-[10px] font-bold uppercase tracking-[.08em] mb-1.5 text-[var(--text-muted)]">
                        Replanifier {cutIds.length} ticket(s) dans…
                      </div>
                      <div className="flex items-center gap-2">
                        <div className="flex-1">
                          <LookupField
                            value={sprintTarget.name}
                            icon={<CalendarRange size={11} />}
                            placeholder="sprint cible..."
                            clearLabel="Backlog (retirer du sprint)"
                            onSearch={searchSprint}
                            onPick={option =>
                              setSprintTarget({ id: option?.id || '', name: option?.label || 'Backlog' })
                            }
                          />
                        </div>
                        <button
                          type="button"
                          disabled={!sprintTarget.name || busyKey === 'sprint'}
                          onClick={async () => {
                            setBusyKey('sprint')
                            await setTasksSprint(
                              currentProject!.id,
                              cutIds,
                              sprintTarget.id,
                              sprintTarget.name
                            )
                            setChecked({})
                            setCutAt(-1)
                            setSprintTarget({ id: '', name: '' })
                            setBusyKey(null)
                          }}
                          className="px-2.5 py-1.5 rounded-lg text-[11px] font-bold cursor-pointer shrink-0 disabled:opacity-40"
                          style={{ color: 'var(--status-ok)', background: 'rgb(var(--status-ok-rgb) / 0.12)', border: '1px solid rgb(var(--status-ok-rgb) / 0.32)' }}
                          title="Déplacer la sélection dans ce sprint, via la file d'activités"
                        >
                          {busyKey === 'sprint' ? '…' : 'Replanifier'}
                        </button>
                      </div>
                    </div>
                  )}

                  {/* Découpe : les stories cochées, ou celles au delà du
                      curseur, quittent l'épic pour un autre. */}
                  {cutIds.length > 0 && (
                    <div className="p-2.5 rounded-xl border" style={{ background: 'var(--accent-light)', borderColor: 'rgb(var(--accent-rgb) / 0.4)' }}>
                      <div className="text-[10px] font-bold uppercase tracking-[.08em] mb-1.5" style={{ color: 'var(--accent-color)' }}>
                        Couper {cutIds.length} ticket(s) vers…
                      </div>
                      <div className="flex items-center gap-2">
                        <div className="flex-1">
                          <LookupField
                            value={moveTarget}
                            icon={<Target size={11} />}
                            placeholder="épic existant..."
                            allowClear={false}
                            emptyHint="Aucun épic ne correspond."
                            onSearch={async query => {
                              const q = query.trim().toLowerCase()
                              return allRows
                                .filter(r => r.key !== selected.key && !r.closed)
                                .filter(r => !q || r.key.toLowerCase().includes(q) || r.title.toLowerCase().includes(q))
                                .slice(0, 40)
                                .map(r => ({ id: r.key, label: r.key, sublabel: r.title }))
                            }}
                            onPick={option => setMoveTarget(option?.id || '')}
                          />
                        </div>
                        <button
                          type="button"
                          disabled={!moveTarget || busyKey === 'move'}
                          onClick={async () => {
                            setBusyKey('move')
                            await moveTasksToEpic(currentProject!.id, cutIds, moveTarget)
                            setChecked({})
                            setCutAt(-1)
                            setBusyKey(null)
                          }}
                          className="px-2.5 py-1.5 rounded-lg text-[11px] font-bold text-white accent-bg disabled:opacity-40 cursor-pointer shrink-0"
                        >
                          Déplacer
                        </button>
                      </div>
                      {epicFieldSelectors && <div className="mt-2">{epicFieldSelectors}</div>}
                      <div className="flex items-center gap-2 mt-2">
                        <input
                          type="text"
                          value={newEpicTitle}
                          onChange={e => setNewEpicTitle(e.target.value)}
                          placeholder="…ou vers un nouvel épic : son titre"
                          className="flex-1 px-2 py-1.5 text-[11px] rounded-lg bg-[var(--bg-primary)] border border-[var(--border-color)] text-[var(--text-primary)] focus:outline-none focus:border-[var(--accent-color)]"
                        />
                        <button
                          type="button"
                          disabled={!newEpicTitle.trim() || busyKey === 'move' || Boolean(missingEpicField)}
                          onClick={async () => {
                            setBusyKey('move')
                            // La découpe est mise en file : l'épic est créé par
                            // l'activité, donc la liste se recharge quand la
                            // file se vide, pas ici.
                            await moveTasksToEpic(currentProject!.id, cutIds, '', newEpicTitle.trim(), epicFieldValues)
                            setChecked({})
                            setCutAt(-1)
                            setNewEpicTitle('')
                            setBusyKey(null)
                          }}
                          title={
                            missingEpicField
                              ? `Jira impose « ${missingEpicField.name} » à la création d'un épic : choisissez une valeur ci-dessus.`
                              : 'Créer l\'épic cible et y déplacer les tickets sélectionnés'
                          }
                          className="px-2.5 py-1.5 rounded-lg text-[11px] font-bold cursor-pointer shrink-0 disabled:opacity-40"
                          style={{ color: 'var(--status-info)', background: 'rgb(var(--status-info-rgb) / 0.12)', border: '1px solid rgb(var(--status-info-rgb) / 0.32)' }}
                        >
                          {busyKey === 'move' ? '…' : 'Créer et couper'}
                        </button>
                      </div>
                      {missingEpicField && (
                        <p className="mt-1 text-[9.5px] leading-snug" style={{ color: 'var(--status-danger)' }}>
                          Jira impose « {missingEpicField.name} » pour créer un épic : sans valeur, la création est
                          refusée en 400.
                        </p>
                      )}
                    </div>
                  )}

</>
              ) : (
                <>
                  {/* LATER : le cadrage se fait ici, description et TODO */}
                  <div>
                    <div className="flex items-center justify-between mb-1.5">
                      <span className="text-[10px] font-bold uppercase tracking-[.08em] text-[var(--text-muted)]">Description</span>
                      {draftDirty && (
                        <button
                          type="button"
                          onClick={() => {
                            persist(selected.key, { description: draftDescription })
                            setDraftDirty(false)
                          }}
                          className="flex items-center gap-1 px-2 py-0.5 rounded text-[10px] font-bold text-white accent-bg cursor-pointer"
                        >
                          <Save size={10} /> Enregistrer
                        </button>
                      )}
                    </div>
                    <MarkdownEditor
                      value={draftDescription}
                      onChange={value => {
                        setDraftDescription(value)
                        setDraftDirty(true)
                      }}
                      minHeight={160}
                      placeholder="Le problème, le périmètre, ce qui est hors périmètre… Ce cadrage vit dans Taskacao."
                    />
                  </div>

                  <div>
                    <div className="text-[10px] font-bold uppercase tracking-[.08em] text-[var(--text-muted)] mb-1.5">
                      TODO ({todosOf(selected).filter(t => t.done).length}/{todosOf(selected).length})
                    </div>
                    <div className="flex flex-col gap-1.5">
                      {todosOf(selected).map(todo => (
                        <div key={todo.id} className="flex items-start gap-2 px-2.5 py-2 rounded-lg bg-[var(--bg-primary)] border"
                          style={{ borderColor: todo.done ? 'rgb(var(--accent-rgb) / 0.4)' : 'var(--border-color)' }}>
                          <button
                            type="button"
                            onClick={() =>
                              persist(selected.key, {
                                todos: todosOf(selected).map(t => (t.id === todo.id ? { ...t, done: !t.done } : t)),
                              })
                            }
                            className="w-3.5 h-3.5 mt-0.5 rounded shrink-0 flex items-center justify-center cursor-pointer"
                            style={{
                              background: todo.done ? 'var(--accent-color)' : 'transparent',
                              border: `1px solid ${todo.done ? 'var(--accent-color)' : 'var(--border-color)'}`,
                            }}
                            title={todo.done ? 'Rouvrir' : 'Cocher'}
                          >
                            {todo.done && <Check size={10} className="text-white" />}
                          </button>
                          <span className="text-[11.5px] leading-snug flex-1"
                            style={{
                              color: todo.done ? 'var(--text-muted)' : 'var(--text-primary)',
                              textDecoration: todo.done ? 'line-through' : 'none',
                            }}>
                            {todo.text}
                          </span>
                          {todo.storyKey ? (
                            <button
                              type="button"
                              onClick={() => {
                                const created = tasks.find(t => t.key === todo.storyKey)
                                if (created) setSelectedTask(created)
                              }}
                              className="text-[9.5px] font-mono font-bold px-1.5 py-0.5 rounded shrink-0 cursor-pointer"
                              style={{
                                color: 'var(--status-ok)',
                                background: 'rgb(var(--status-ok-rgb) / 0.13)',
                                border: '1px solid rgb(var(--status-ok-rgb) / 0.32)',
                              }}
                              title={`Story créée : ${todo.storyKey}`}
                            >
                              {todo.storyKey}
                            </button>
                          ) : (
                            <button
                              type="button"
                              disabled={creatingTodoId === todo.id}
                              onClick={async () => {
                                setCreatingTodoId(todo.id)
                                const result = await createStoryFromEpicTodo(currentProject!.id, selected.key, todo.id)
                                setCreatingTodoId(null)
                                if (result?.epic) {
                                  setEpicMeta(prev => [...prev.filter(m => m.key !== result.epic!.key), result.epic!])
                                }
                              }}
                              className="text-[9.5px] font-mono font-bold px-1.5 py-0.5 rounded shrink-0 cursor-pointer disabled:opacity-50"
                              style={{
                                color: 'var(--status-info)',
                                background: 'rgb(var(--status-info-rgb) / 0.12)',
                                border: '1px solid rgb(var(--status-info-rgb) / 0.32)',
                              }}
                              title={`Créer une story Jira sous ${selected.key}`}
                            >
                              {creatingTodoId === todo.id ? '…' : 'Créer story'}
                            </button>
                          )}
                          <button
                            type="button"
                            onClick={() => persist(selected.key, { todos: todosOf(selected).filter(t => t.id !== todo.id) })}
                            className="p-0.5 rounded text-[var(--text-muted)] hover:text-rose-400 cursor-pointer shrink-0"
                            title="Retirer"
                          >
                            <Trash2 size={11} />
                          </button>
                        </div>
                      ))}
                      {todosOf(selected).length === 0 && (
                        <p className="text-[11px] text-[var(--text-muted)]">
                          Aucune ligne. Écris ici ce qu'il faudra faire, avant même de créer des stories.
                        </p>
                      )}
                    </div>

                    <div className="flex items-center gap-2 mt-2">
                      <input
                        type="text"
                        value={newTodo}
                        onChange={e => setNewTodo(e.target.value)}
                        onKeyDown={e => {
                          if (e.key === 'Enter') {
                            e.preventDefault()
                            addTodo(selected)
                          }
                        }}
                        placeholder="Ajouter une ligne de TODO…"
                        className="flex-1 px-2.5 py-1.5 text-xs rounded-xl bg-[var(--bg-primary)] border border-[var(--border-color)] text-[var(--text-primary)] focus:outline-none focus:border-[var(--accent-color)]"
                      />
                      <button type="button" onClick={() => addTodo(selected)} disabled={!newTodo.trim()}
                        className="flex items-center gap-1 px-2.5 py-1.5 rounded-xl text-xs font-bold text-white accent-bg disabled:opacity-40 cursor-pointer shrink-0">
                        <Plus size={12} /> Ajouter
                      </button>
                    </div>
                  </div>

                  {selected.tasks.length > 0 && (
                    <div>
                      <div className="text-[10px] font-bold uppercase tracking-[.08em] text-[var(--text-muted)] mb-1.5">
                        Tickets déjà créés ({selected.tasks.length})
                      </div>
                      <div className="flex flex-wrap gap-1.5">
                        {selected.tasks.map(task => (
                          <button key={task.id} type="button" onClick={() => setSelectedTask(task)}
                            className="text-[10px] font-mono px-1.5 py-0.5 rounded bg-[var(--bg-primary)] border border-[var(--border-color)] text-[var(--text-secondary)] hover:text-[var(--text-primary)] cursor-pointer"
                            title={task.title}>
                            {task.key}
                          </button>
                        ))}
                      </div>
                    </div>
                  )}
                </>
              )}
            </div>
          </aside>
        )}
      </div>
    </div>
  )
}
