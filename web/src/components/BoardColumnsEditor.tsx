import React, { useEffect, useMemo, useState } from 'react'
import { Plus, X, ArrowUp, ArrowDown, RefreshCw, Kanban, Tag, GitPullRequest, Eye, EyeOff } from 'lucide-react'
import { useApp } from '../context/AppContext'
import type { Project, TrackerBoard, TrackerColumn, WorkflowStage } from '../types'

/**
 * Éditeur des colonnes du board, sur le modèle de Jira : des colonnes, les
 * statuts du tracker qu'on y dépose, et — l'ajout de Taskacao — les étapes du
 * workflow agentique déposées de la même façon.
 *
 * Tout se fait au glisser-déposer, y compris pour déplacer un statut déjà placé
 * d'une colonne à l'autre : un menu « ajouter » ne permettait que de placer les
 * statuts encore libres, donc rien après un import qui les affecte tous.
 *
 * Un statut n'appartient qu'à une colonne, sinon un ticket apparaîtrait deux
 * fois sur le board. Une étape du workflow, elle, peut viser plusieurs colonnes.
 */

const WORKFLOW_STAGES: { id: WorkflowStage; label: string }[] = [
  { id: 'new', label: '#new' },
  { id: 'clarified', label: '#clarified' },
  { id: 'specified', label: '#specified' },
  { id: 'implemented', label: '#implemented' },
  { id: 'reviewed', label: '#reviewed' },
  { id: 'finished', label: '#finished' },
]

const DRAG_STATUS = 'application/x-taskacao-status'
const DRAG_STAGE = 'application/x-taskacao-stage'

interface Props {
  project: Project | null
  columns: TrackerColumn[]
  onColumnsChange: (columns: TrackerColumn[]) => void
  stageColumns: Record<string, string[]>
  onStageColumnsChange: (mapping: Record<string, string[]>) => void
}

type DragPayload =
  | { kind: 'status'; value: string }
  | { kind: 'stage'; value: WorkflowStage }
  | null

export const BoardColumnsEditor: React.FC<Props> = ({
  project,
  columns,
  onColumnsChange,
  stageColumns,
  onStageColumnsChange,
}) => {
  const { listProjectBoards, importProjectBoardColumns, fetchProjectTrackerStatuses } = useApp()

  const [boards, setBoards] = useState<TrackerBoard[]>([])
  const [selectedBoard, setSelectedBoard] = useState(project?.boardId || '')
  const [statuses, setStatuses] = useState<string[]>([])
  const [newColumnName, setNewColumnName] = useState('')
  const [isImporting, setIsImporting] = useState(false)
  const [dragging, setDragging] = useState<DragPayload>(null)
  const [dropTarget, setDropTarget] = useState<string | null>(null)

  useEffect(() => {
    if (!project?.id) return
    fetchProjectTrackerStatuses(project.id).then(setStatuses)
    listProjectBoards(project.id).then(list => {
      setBoards(list)
      if (!project.boardId && list.length > 0) {
        setSelectedBoard((list.find(b => b.type === 'scrum') || list[0]).id)
      }
    })
  }, [project?.id])

  const assignedStatuses = useMemo(
    () => new Set(columns.flatMap(c => c.statuses.map(s => s.toLowerCase()))),
    [columns]
  )
  const freeStatuses = statuses.filter(st => !assignedStatuses.has(st.toLowerCase()))

  const refreshFromBoard = async () => {
    if (!project?.id || !selectedBoard) return
    setIsImporting(true)
    const updated = await importProjectBoardColumns(project.id, selectedBoard)
    setIsImporting(false)
    if (updated?.trackerColumns) {
      onColumnsChange(updated.trackerColumns)
      fetchProjectTrackerStatuses(project.id).then(setStatuses)
    }
  }

  // ----- glisser-déposer -----
  const startDragStatus = (e: React.DragEvent, status: string) => {
    e.dataTransfer.setData(DRAG_STATUS, status)
    e.dataTransfer.setData('text/plain', status)
    e.dataTransfer.effectAllowed = 'move'
    setDragging({ kind: 'status', value: status })
  }

  const startDragStage = (e: React.DragEvent, stage: WorkflowStage) => {
    e.dataTransfer.setData(DRAG_STAGE, stage)
    e.dataTransfer.setData('text/plain', stage)
    e.dataTransfer.effectAllowed = 'copy'
    setDragging({ kind: 'stage', value: stage })
  }

  const endDrag = () => {
    setDragging(null)
    setDropTarget(null)
  }

  const dropOnColumn = (e: React.DragEvent, columnName: string) => {
    e.preventDefault()
    const status = e.dataTransfer.getData(DRAG_STATUS)
    const stage = e.dataTransfer.getData(DRAG_STAGE) as WorkflowStage

    if (status) {
      // Exclusif : le statut quitte sa colonne précédente.
      onColumnsChange(
        columns.map(col => ({
          ...col,
          statuses:
            col.name === columnName
              ? Array.from(new Set([...col.statuses, status]))
              : col.statuses.filter(st => st.toLowerCase() !== status.toLowerCase()),
        }))
      )
    } else if (stage) {
      const current = stageColumns[stage] || []
      if (!current.includes(columnName)) {
        onStageColumnsChange({ ...stageColumns, [stage]: [...current, columnName] })
      }
    }
    endDrag()
  }

  const dropOnPool = (e: React.DragEvent) => {
    e.preventDefault()
    const status = e.dataTransfer.getData(DRAG_STATUS)
    if (status) {
      onColumnsChange(
        columns.map(col => ({
          ...col,
          statuses: col.statuses.filter(st => st.toLowerCase() !== status.toLowerCase()),
        }))
      )
    }
    endDrag()
  }

  const removeStatus = (columnName: string, status: string) => {
    onColumnsChange(
      columns.map(col =>
        col.name === columnName
          ? { ...col, statuses: col.statuses.filter(st => st !== status) }
          : col
      )
    )
  }

  const removeStage = (columnName: string, stage: WorkflowStage) => {
    const kept = (stageColumns[stage] || []).filter(c => c !== columnName)
    onStageColumnsChange({ ...stageColumns, [stage]: kept })
  }

  // ----- colonnes -----
  const addColumn = () => {
    const name = newColumnName.trim()
    if (!name || columns.some(c => c.name.toLowerCase() === name.toLowerCase())) {
      setNewColumnName('')
      return
    }
    onColumnsChange([...columns, { name, statuses: [] }])
    setNewColumnName('')
  }

  const removeColumn = (index: number) => {
    const removed = columns[index]
    onColumnsChange(columns.filter((_, idx) => idx !== index))
    const cleaned: Record<string, string[]> = {}
    Object.entries(stageColumns).forEach(([stage, cols]) => {
      const kept = (cols || []).filter(c => c !== removed.name)
      if (kept.length > 0) cleaned[stage] = kept
    })
    onStageColumnsChange(cleaned)
  }

  const moveColumn = (index: number, delta: number) => {
    const target = index + delta
    if (target < 0 || target >= columns.length) return
    const next = [...columns]
    const [moved] = next.splice(index, 1)
    next.splice(target, 0, moved)
    onColumnsChange(next)
  }

  const renameColumn = (index: number, name: string) => {
    const previous = columns[index].name
    onColumnsChange(columns.map((col, idx) => (idx === index ? { ...col, name } : col)))
    const renamed: Record<string, string[]> = {}
    Object.entries(stageColumns).forEach(([stage, cols]) => {
      renamed[stage] = (cols || []).map(c => (c === previous ? name : c))
    })
    onStageColumnsChange(renamed)
  }

  const stagesOfColumn = (columnName: string): WorkflowStage[] =>
    WORKFLOW_STAGES.filter(st => (stageColumns[st.id] || []).includes(columnName)).map(st => st.id)

  const chipBase =
    'inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-[10px] font-mono border cursor-grab active:cursor-grabbing select-none'

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between gap-2">
        <label className="text-[10px] font-bold uppercase tracking-wider text-[var(--text-muted)]">
          Colonnes du board
        </label>
        <div className="flex items-center gap-1.5">
          <select
            value={selectedBoard}
            onChange={e => setSelectedBoard(e.target.value)}
            className="text-[11px] bg-[var(--bg-tertiary)] text-[var(--text-secondary)] border border-[var(--border-color)] rounded-md px-2 py-1 focus:outline-none focus:border-[var(--accent-color)] cursor-pointer max-w-[170px]"
          >
            <option value="">Board du tracker…</option>
            {boards.map(b => (
              <option key={b.id} value={b.id}>{b.name} ({b.type})</option>
            ))}
          </select>
          <button
            type="button"
            onClick={refreshFromBoard}
            disabled={!selectedBoard || isImporting}
            className="flex items-center gap-1 px-2 py-1 rounded-md text-[11px] font-bold text-[var(--text-primary)] bg-[var(--bg-tertiary)] border border-[var(--border-color)] hover:border-[var(--accent-color)]/50 disabled:opacity-40 transition-colors cursor-pointer"
            title="Reprendre les colonnes de ce board. La synchro le fait aussi automatiquement."
          >
            <RefreshCw size={11} className={isImporting ? 'animate-spin' : ''} />
            <span>{isImporting ? 'Lecture…' : 'Détecter'}</span>
          </button>
        </div>
      </div>

      <p className="text-[10px] text-[var(--text-muted)] -mt-1">
        Les colonnes sont détectées à chaque synchro. Glisse un statut du tracker ou une étape du workflow dans une colonne ; un statut n'appartient qu'à une colonne, une étape peut en viser plusieurs.
      </p>

      {/* Réservoirs de pastilles à glisser */}
      <div
        onDragOver={e => { e.preventDefault(); setDropTarget('pool') }}
        onDragLeave={() => setDropTarget(null)}
        onDrop={dropOnPool}
        className={`p-2.5 rounded-xl border space-y-2 transition-colors ${
          dropTarget === 'pool'
            ? 'border-[var(--accent-color)] bg-[var(--accent-light)]/20'
            : 'border-[var(--border-color)] bg-[var(--bg-tertiary)]/40'
        }`}
      >
        <div className="flex items-center gap-1.5 flex-wrap">
          <span className="inline-flex items-center gap-1 text-[9px] uppercase tracking-wider text-[var(--text-muted)] mr-1">
            <GitPullRequest size={10} /> Statuts libres
          </span>
          {freeStatuses.length === 0 ? (
            <span className="text-[10px] text-[var(--text-muted)]">
              Tous les statuts connus sont affectés. Glisse-en un ici pour le libérer.
            </span>
          ) : (
            freeStatuses.map(st => (
              <span
                key={st}
                draggable
                onDragStart={e => startDragStatus(e, st)}
                onDragEnd={endDrag}
                className={`${chipBase} bg-[var(--bg-primary)] text-[var(--text-secondary)] border-[var(--border-color)] hover:border-[var(--accent-color)]/50`}
                title="Glisser dans une colonne"
              >
                {st}
              </span>
            ))
          )}
        </div>

        <div className="flex items-center gap-1.5 flex-wrap pt-1.5 border-t border-[var(--border-color)]/50">
          <span className="inline-flex items-center gap-1 text-[9px] uppercase tracking-wider text-[var(--text-muted)] mr-1">
            <Tag size={10} /> Workflow agentique
          </span>
          {WORKFLOW_STAGES.map(stage => (
            <span
              key={stage.id}
              draggable
              onDragStart={e => startDragStage(e, stage.id)}
              onDragEnd={endDrag}
              className={`${chipBase} bg-emerald-500/10 text-emerald-300 border-emerald-500/30 hover:border-emerald-400`}
              title="Glisser dans une ou plusieurs colonnes"
            >
              {stage.label}
            </span>
          ))}
        </div>
      </div>

      {columns.length === 0 ? (
        <p className="text-[10px] text-[var(--text-muted)]">
          Aucune colonne : le board utilise les statuts génériques. Lance une synchro, clique sur Détecter, ou crée une colonne à la main.
        </p>
      ) : (
        <div className="space-y-2">
          {columns.map((col, index) => {
            const isTarget = dropTarget === col.name
            return (
              <div
                key={`${col.name}-${index}`}
                onDragOver={e => { e.preventDefault(); setDropTarget(col.name) }}
                onDragLeave={() => setDropTarget(prev => (prev === col.name ? null : prev))}
                onDrop={e => dropOnColumn(e, col.name)}
                className={`p-2.5 rounded-xl border space-y-2 transition-colors ${
                  isTarget
                    ? 'border-[var(--accent-color)] bg-[var(--accent-light)]/20 ring-2 ring-[var(--accent-glow)]'
                    : col.hidden
                      ? 'border-amber-500/30 bg-[var(--bg-tertiary)]/30 opacity-70'
                      : 'border-[var(--border-color)] bg-[var(--bg-tertiary)]/60'
                }`}
              >
                <div className="flex items-center gap-1.5">
                  <Kanban size={13} className="text-[var(--accent-color)] shrink-0" />
                  <input
                    type="text"
                    value={col.name}
                    onChange={e => renameColumn(index, e.target.value)}
                    className="flex-1 px-2 py-1 text-xs font-bold rounded-lg bg-[var(--bg-primary)] border border-[var(--border-color)] text-[var(--text-primary)] focus:outline-none focus:border-[var(--accent-color)]"
                  />
                  <button type="button" onClick={() => moveColumn(index, -1)} disabled={index === 0}
                    className="p-1 rounded text-[var(--text-muted)] hover:text-[var(--text-primary)] disabled:opacity-30 cursor-pointer" title="Monter">
                    <ArrowUp size={12} />
                  </button>
                  <button type="button" onClick={() => moveColumn(index, 1)} disabled={index === columns.length - 1}
                    className="p-1 rounded text-[var(--text-muted)] hover:text-[var(--text-primary)] disabled:opacity-30 cursor-pointer" title="Descendre">
                    <ArrowDown size={12} />
                  </button>
                  <button
                    type="button"
                    onClick={() =>
                      onColumnsChange(
                        columns.map((c, idx) => (idx === index ? { ...c, hidden: !c.hidden } : c))
                      )
                    }
                    className={`p-1 rounded cursor-pointer ${
                      col.hidden ? 'text-amber-400' : 'text-[var(--text-muted)] hover:text-[var(--text-primary)]'
                    }`}
                    title={col.hidden ? 'Colonne masquée sur le board : réafficher' : 'Masquer cette colonne sur le board'}
                  >
                    {col.hidden ? <EyeOff size={12} /> : <Eye size={12} />}
                  </button>
                  <button type="button" onClick={() => removeColumn(index)}
                    className="p-1 rounded text-[var(--text-muted)] hover:text-rose-400 cursor-pointer" title="Supprimer la colonne">
                    <X size={12} />
                  </button>
                </div>

                <div className="flex flex-wrap items-center gap-1 min-h-[22px]">
                  {col.statuses.length === 0 && (
                    <span className="text-[10px] text-[var(--text-muted)] italic">
                      Aucun statut : dépose-en un ici
                    </span>
                  )}
                  {col.statuses.map(st => (
                    <span
                      key={st}
                      draggable
                      onDragStart={e => startDragStatus(e, st)}
                      onDragEnd={endDrag}
                      className={`${chipBase} bg-[var(--accent-light)] accent-text border-[var(--accent-color)]/30`}
                      title="Glisser vers une autre colonne, ou vers les statuts libres"
                    >
                      {st}
                      <button type="button" onClick={() => removeStatus(col.name, st)} className="hover:opacity-70 cursor-pointer" title="Retirer">
                        <X size={9} />
                      </button>
                    </span>
                  ))}
                </div>

                <div className="flex flex-wrap items-center gap-1 pt-1.5 border-t border-[var(--border-color)]/50 min-h-[22px]">
                  <span className="text-[9px] uppercase tracking-wider text-[var(--text-muted)] mr-1">Workflow</span>
                  {stagesOfColumn(col.name).length === 0 && (
                    <span className="text-[10px] text-[var(--text-muted)] italic">
                      Aucune étape affectée
                    </span>
                  )}
                  {stagesOfColumn(col.name).map(stage => (
                    <span
                      key={stage}
                      draggable
                      onDragStart={e => startDragStage(e, stage)}
                      onDragEnd={endDrag}
                      className={`${chipBase} bg-emerald-500/20 text-emerald-300 border-emerald-500/40 font-bold`}
                    >
                      #{stage}
                      <button type="button" onClick={() => removeStage(col.name, stage)} className="hover:opacity-70 cursor-pointer" title="Retirer de cette colonne">
                        <X size={9} />
                      </button>
                    </span>
                  ))}
                </div>
              </div>
            )
          })}
        </div>
      )}

      {dragging?.kind === 'status' && (
        <p className="text-[10px] text-[var(--text-muted)]">
          Dépose « {dragging.value} » dans une colonne, ou dans les statuts libres pour l'en retirer.
        </p>
      )}

      <div className="flex items-center gap-2">
        <input
          type="text"
          value={newColumnName}
          onChange={e => setNewColumnName(e.target.value)}
          onKeyDown={e => {
            if (e.key === 'Enter') {
              e.preventDefault()
              addColumn()
            }
          }}
          placeholder="Nouvelle colonne (ex. PEER REVIEW)"
          className="flex-1 px-2.5 py-1.5 text-xs rounded-xl bg-[var(--bg-tertiary)] border border-[var(--border-color)] text-[var(--text-primary)] focus:outline-none focus:border-[var(--accent-color)]"
        />
        <button
          type="button"
          onClick={addColumn}
          disabled={!newColumnName.trim()}
          className="flex items-center gap-1 px-2.5 py-1.5 rounded-xl text-xs font-bold text-white accent-bg shadow-xs hover:opacity-90 disabled:opacity-40 transition-all cursor-pointer shrink-0"
        >
          <Plus size={12} />
          <span>Colonne</span>
        </button>
      </div>
    </div>
  )
}
