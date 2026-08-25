import React, { useEffect, useMemo, useState } from 'react'
import {
  Target,
  Route,
  Compass,
  HelpCircle,
  CalendarDays,
  ExternalLink,
  Plus,
  Check,
  Trash2,
  Terminal as TerminalIcon,
  AlertTriangle,
  Save,
} from 'lucide-react'
import { useApp } from '../context/AppContext'
import {
  buildEpicRows,
  placementIssues,
  placementOf,
  HORIZON_META,
  MATURITY_META,
  PLACEMENT_META,
  PRIORITY_META,
  type EpicRow,
  type Horizon,
  type HorizonTab,
} from '../lib/roadmap'
import type { EpicHorizon, EpicMeta, EpicTodo } from '../types'

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
    addToast,
  } = useApp()

  const [tab, setTab] = useState<HorizonTab>('now')
  const [epicMeta, setEpicMeta] = useState<EpicMeta[]>([])
  const [selectedKey, setSelectedKey] = useState<string | null>(null)
  const [onlyIssues, setOnlyIssues] = useState(false)

  // Le cadrage n'est enregistré qu'à la demande : écrire à chaque frappe
  // enverrait une requête par caractère.
  const [draftDescription, setDraftDescription] = useState('')
  const [draftDirty, setDraftDirty] = useState(false)
  const [newTodo, setNewTodo] = useState('')
  const [creatingTodoId, setCreatingTodoId] = useState<string | null>(null)

  useEffect(() => {
    if (!currentProject?.id) {
      setEpicMeta([])
      return
    }
    fetchProjectEpics(currentProject.id).then(setEpicMeta)
  }, [currentProject?.id, fetchProjectEpics])

  const rows = useMemo(() => buildEpicRows(tasks, currentProject, epicMeta), [tasks, currentProject, epicMeta])

  const counts = useMemo(
    () => ({
      now: rows.filter(r => r.horizon === 'now').length,
      next: rows.filter(r => r.horizon === 'next').length,
      later: rows.filter(r => r.horizon === 'later').length,
      unclassified: rows.filter(r => !r.horizon).length,
    }),
    [rows]
  )

  const operational = tab === 'now' || tab === 'next'
  const horizonOfTab: Horizon = tab === 'next' ? 'next' : tab === 'later' ? 'later' : 'now'

  const visibleRows = useMemo(() => {
    const list = tab === 'unclassified' ? rows.filter(r => !r.horizon) : rows.filter(r => r.horizon === tab)
    if (operational && onlyIssues) {
      return list.filter(r => placementIssues(r, horizonOfTab).length > 0)
    }
    return list
  }, [rows, tab, operational, onlyIssues, horizonOfTab])

  const selected: EpicRow | null = visibleRows.find(r => r.key === selectedKey) || visibleRows[0] || null

  useEffect(() => {
    setDraftDescription(selected?.meta?.description || '')
    setDraftDirty(false)
    setNewTodo('')
  }, [selected?.key, selected?.meta?.description])

  const jiraBase = (currentProject?.trackerUrl || settings.jiraUrl || '').replace(/\/+$/, '')
  const epicUrl = (key: string) => (jiraBase ? `${jiraBase}/browse/${key}` : undefined)

  const activeSprints = (currentProject?.sprints || []).filter(s => s.state === 'active')
  const futureSprints = (currentProject?.sprints || []).filter(s => s.state === 'future')

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

      <div className="flex-1 flex min-h-0 min-w-0 overflow-hidden">
        {/* Épics de l'horizon courant */}
        <div className="flex-1 overflow-y-auto p-3 min-w-0 space-y-2">
          {visibleRows.length === 0 ? (
            <div className="h-full flex flex-col items-center justify-center gap-2 text-center px-6">
              <Compass size={26} className="text-[var(--text-muted)]" />
              <p className="text-sm font-semibold">Aucun épic ici</p>
              <p className="text-[11px] text-[var(--text-muted)] max-w-sm">
                {tab === 'unclassified'
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
                    {(['now', 'next', 'later'] as EpicHorizon[]).map(h => {
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

        {/* Panneau : vérification des sprints en NOW/NEXT, cadrage en LATER */}
        {selected && (
          <aside className="flex flex-col min-h-0 shrink border-l border-[var(--border-color)] bg-[var(--bg-secondary)]"
            style={{ width: 470, maxWidth: '38%', minWidth: 360 }}>
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
                    className="ml-auto text-[10px] font-semibold inline-flex items-center gap-1 hover:underline"
                    style={{ color: 'var(--status-info)' }}>
                    Ouvrir dans Jira <ExternalLink size={10} />
                  </a>
                )}
              </div>
              <h2 className="text-[16px] font-bold leading-[1.25]">{selected.title}</h2>
            </div>

            <div className="flex-1 overflow-y-auto px-4 pt-3.5 pb-7 flex flex-col gap-4">
              {operational ? (
                <>
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
                      {selected.open.map(task => {
                        const state = placementOf(task, selected, horizonOfTab)
                        const meta = PLACEMENT_META[state]
                        return (
                          <div key={task.id} className="px-2.5 py-2 rounded-lg bg-[var(--bg-primary)] border"
                            style={{ borderColor: state === 'ok' ? 'var(--border-color)' : meta.border }}>
                            <div className="flex items-center gap-2">
                              <button type="button" onClick={() => setSelectedTask(task)}
                                className="text-[10.5px] font-mono font-bold hover:underline cursor-pointer"
                                style={{ color: 'var(--status-info)' }}>
                                {task.key}
                              </button>
                              <span className="text-[9px] px-1 rounded font-mono ml-auto shrink-0 truncate max-w-[150px]"
                                style={{ color: meta.color, background: meta.bg, border: `1px solid ${meta.border}` }}>
                                {task.sprint || meta.label}
                              </span>
                              <button type="button" onClick={() => setChatTask(task)}
                                className="p-0.5 rounded text-[var(--text-muted)] hover:text-cyan-300 cursor-pointer shrink-0"
                                title={`Terminal de ${task.key}`}>
                                <TerminalIcon size={12} />
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
                        )
                      })}
                    </div>
                  </div>
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
                    <textarea
                      rows={8}
                      value={draftDescription}
                      onChange={e => {
                        setDraftDescription(e.target.value)
                        setDraftDirty(true)
                      }}
                      placeholder="Le problème, le périmètre, ce qui est hors périmètre… Ce cadrage vit dans Taskacao."
                      className="w-full px-3 py-2 text-[12px] rounded-xl bg-[var(--bg-primary)] border border-[var(--border-color)] text-[var(--text-primary)] focus:outline-none focus:border-[var(--accent-color)] leading-relaxed resize-y"
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
