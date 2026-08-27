import React, { useState, useEffect, useCallback } from 'react'
import {
  X,
  GitBranch,
  GitPullRequest,
  ExternalLink,
  Copy,
  Check,
  RefreshCw,
  FileCode,
  FolderGit2,
  FilePlus2,
  FileMinus2,
  FileEdit,
  Loader2,
  AlertCircle,
  CheckCircle2,
  Code2
} from 'lucide-react'
import { useApp } from '../context/AppContext'
import type { GitDiffResult } from '../types'

export const GitDiffModal: React.FC = () => {
  const { diffTask, setDiffTask, fetchGitDiff, checkoutTaskBranch, addToast } = useApp()

  const [diffResult, setDiffResult] = useState<GitDiffResult | null>(null)
  const [isLoading, setIsLoading] = useState(false)
  const [copiedPatch, setCopiedPatch] = useState(false)
  const [copiedPath, setCopiedPath] = useState<string | null>(null)
  const [selectedFilePath, setSelectedFilePath] = useState<string | null>(null)

  const loadDiff = useCallback(async () => {
    if (!diffTask) return
    setIsLoading(true)
    const res = await fetchGitDiff(diffTask.id)
    setDiffResult(res)
    if (res?.files && res.files.length > 0) {
      setSelectedFilePath(res.files[0].path)
    }
    setIsLoading(false)
  }, [diffTask, fetchGitDiff])

  useEffect(() => {
    if (diffTask) {
      loadDiff()
    } else {
      setDiffResult(null)
      setSelectedFilePath(null)
    }
  }, [diffTask, loadDiff])

  useEffect(() => {
    if (!diffTask) return
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        setDiffTask(null)
      }
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [diffTask, setDiffTask])

  if (!diffTask) return null

  const handleCopyPatch = () => {
    if (!diffResult?.rawDiff) return
    navigator.clipboard?.writeText(diffResult.rawDiff)
    setCopiedPatch(true)
    setTimeout(() => setCopiedPatch(false), 2000)
    addToast({
      type: 'success',
      title: 'Patch diff copié',
      description: 'Le code diff a été copié dans votre presse-papiers.',
    })
  }

  const handleCopyFilePath = (path: string, e: React.MouseEvent) => {
    e.stopPropagation()
    navigator.clipboard?.writeText(path)
    setCopiedPath(path)
    setTimeout(() => setCopiedPath(null), 2000)
  }

  const renderFileStatusIcon = (status: string) => {
    switch (status) {
      case 'added':
        return <FilePlus2 size={13} className="text-emerald-400 shrink-0" />
      case 'deleted':
        return <FileMinus2 size={13} className="text-rose-400 shrink-0" />
      case 'renamed':
        return <FileEdit size={13} className="text-amber-400 shrink-0" />
      default:
        return <FileCode size={13} className="text-blue-400 shrink-0" />
    }
  }

  const renderDiffLines = (diffContent: string) => {
    const lines = diffContent.split('\n')
    let startIndex = 0
    for (let i = 0; i < lines.length; i++) {
      if (lines[i].startsWith('@@')) {
        startIndex = i
        break
      }
    }

    const contentLines = startIndex > 0 ? lines.slice(startIndex) : lines

    return (
      <div className="font-mono text-[11px] leading-5 overflow-x-auto py-1">
        {contentLines.map((line, idx) => {
          let lineBg = ''
          let textStyle = 'text-[var(--text-secondary)]'
          let signBg = ''

          if (line.startsWith('@@')) {
            lineBg = 'bg-cyan-500/10 text-cyan-300 font-semibold'
            textStyle = 'text-cyan-300'
          } else if (line.startsWith('+')) {
            lineBg = 'bg-emerald-500/15 text-emerald-300'
            textStyle = 'text-emerald-300'
            signBg = 'text-emerald-400 font-bold'
          } else if (line.startsWith('-')) {
            lineBg = 'bg-rose-500/15 text-rose-300'
            textStyle = 'text-rose-300'
            signBg = 'text-rose-400 font-bold'
          }

          return (
            <div
              key={idx}
              className={`flex items-start px-3 py-0.5 hover:bg-white/5 transition-colors ${lineBg}`}
            >
              <span className={`w-4 select-none shrink-0 font-bold text-[11px] ${signBg}`}>
                {line.startsWith('+') ? '+' : line.startsWith('-') ? '-' : line.startsWith('@') ? '@' : ' '}
              </span>
              <pre className={`whitespace-pre font-mono flex-1 pl-1 ${textStyle}`}>
                {line.startsWith('+') || line.startsWith('-') ? line.substring(1) : line}
              </pre>
            </div>
          )
        })}
      </div>
    )
  }

  return (
    <div className="fixed top-0 left-0 h-[var(--app-h)] w-[var(--app-w)] z-50 flex items-center justify-center p-3 sm:p-6 bg-black/75 backdrop-blur-xs animate-in fade-in duration-200">
      <div className="relative w-full max-w-6xl h-[calc(var(--app-h)*0.9)] rounded-2xl bg-[var(--bg-secondary)] border border-[var(--border-color)] shadow-2xl flex flex-col overflow-hidden">
        {/* Header */}
        <div className="flex flex-wrap items-center justify-between gap-3 px-5 py-3.5 border-b border-[var(--border-color)] bg-[var(--bg-tertiary)]/50 shrink-0">
          <div className="flex items-center gap-3 min-w-0">
            <div className="w-8 h-8 rounded-xl bg-indigo-500/20 text-indigo-400 border border-indigo-500/30 flex items-center justify-center shadow-xs shrink-0">
              <Code2 size={16} />
            </div>

            <div className="flex flex-col min-w-0">
              <div className="flex items-center gap-2 flex-wrap">
                <span className="font-mono text-xs font-bold text-indigo-400 bg-indigo-500/10 px-2 py-0.5 rounded border border-indigo-500/20">
                  {diffTask.key}
                </span>
                <h3 className="text-sm font-bold text-[var(--text-primary)] truncate max-w-md">
                  {diffTask.title}
                </h3>
              </div>

              {diffResult && (
                <div className="flex items-center gap-2 text-[10px] font-mono text-[var(--text-muted)] mt-0.5 flex-wrap">
                  <span className="inline-flex items-center gap-1 text-indigo-300">
                    <GitBranch size={11} />
                    <span>{diffResult.branch || diffTask.branchName || 'active-branch'}</span>
                  </span>
                  <span>vs</span>
                  <span className="text-slate-400">{diffResult.baseBranch || 'main'}</span>

                  {diffResult.filesChanged > 0 && (
                    <>
                      <span>•</span>
                      <span className="text-[var(--text-secondary)] font-bold">
                        {diffResult.filesChanged} fichier{diffResult.filesChanged > 1 ? 's' : ''}
                      </span>
                      <span className="text-emerald-400 font-bold">+{diffResult.insertions}</span>
                      <span className="text-rose-400 font-bold">-{diffResult.deletions}</span>
                    </>
                  )}

                  {diffResult.worktreePath && (
                    <span className="inline-flex items-center gap-1 text-[9px] font-mono px-1.5 py-0.5 rounded bg-emerald-500/10 text-emerald-300 border border-emerald-500/20" title={`Worktree isolé : ${diffResult.worktreePath}`}>
                      <span>🌳 Worktree isolé</span>
                    </span>
                  )}
                </div>
              )}
            </div>
          </div>

            {/* Action buttons */}
          <div className="flex items-center gap-2 shrink-0">
            <button
              type="button"
              onClick={async () => {
                if (diffTask) {
                  await checkoutTaskBranch(diffTask.id)
                  await loadDiff()
                }
              }}
              className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-xl text-xs font-semibold bg-indigo-600/20 hover:bg-indigo-600/30 text-indigo-300 border border-indigo-500/30 transition-colors"
              title="Basculez votre environnement de travail local sur cette branche"
            >
              <GitBranch size={13} />
              <span>Checkout branche</span>
            </button>

            {diffTask.prUrl && (
              <a
                href={diffTask.prUrl}
                target="_blank"
                rel="noreferrer"
                className={`inline-flex items-center gap-1.5 px-3 py-1.5 rounded-xl text-xs font-semibold shadow-xs transition-all ${
                  diffTask.prUrl.includes('gitlab')
                    ? 'bg-orange-500/20 text-orange-300 hover:bg-orange-500/30 border border-orange-500/40'
                    : 'bg-purple-500/20 text-purple-300 hover:bg-purple-500/30 border border-purple-500/40'
                }`}
                title="Ouvrir la Pull / Merge Request"
              >
                <GitPullRequest size={13} />
                <span>{diffTask.prUrl.includes('gitlab') ? 'GitLab MR' : 'GitHub PR'}</span>
                <ExternalLink size={11} />
              </a>
            )}

            <button
              type="button"
              onClick={handleCopyPatch}
              disabled={!diffResult?.rawDiff}
              className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-xl text-xs font-semibold bg-[var(--bg-primary)] hover:bg-[var(--bg-tertiary)] text-[var(--text-secondary)] hover:text-[var(--text-primary)] border border-[var(--border-color)] transition-colors disabled:opacity-50"
              title="Copier le diff brut (patch)"
            >
              {copiedPatch ? (
                <>
                  <Check size={13} className="text-emerald-400" />
                  <span className="text-emerald-400">Patch copié !</span>
                </>
              ) : (
                <>
                  <Copy size={13} />
                  <span>Copier diff</span>
                </>
              )}
            </button>

            <button
              type="button"
              onClick={loadDiff}
              disabled={isLoading}
              className="p-1.5 rounded-xl bg-[var(--bg-primary)] hover:bg-[var(--bg-tertiary)] text-[var(--text-muted)] hover:text-[var(--text-primary)] border border-[var(--border-color)] transition-colors"
              title="Rafraîchir le diff"
            >
              <RefreshCw size={14} className={isLoading ? 'animate-spin text-indigo-400' : ''} />
            </button>

            <button
              type="button"
              onClick={() => setDiffTask(null)}
              className="p-1.5 rounded-xl hover:bg-[var(--bg-tertiary)] text-[var(--text-muted)] hover:text-[var(--text-primary)] transition-colors ml-1"
              title="Fermer (ESC)"
            >
              <X size={18} />
            </button>
          </div>
        </div>

        {/* Content Body */}
        <div className="flex-1 flex min-h-0 overflow-hidden">
          {isLoading ? (
            <div className="flex-1 flex flex-col items-center justify-center p-12 text-center">
              <Loader2 size={32} className="animate-spin text-[var(--accent-color)] mb-3" />
              <p className="text-sm font-semibold text-[var(--text-primary)]">
                Analyse du diff Git en cours...
              </p>
              <p className="text-xs text-[var(--text-muted)] mt-1 font-mono">
                Comparaison de la branche et du répertoire local
              </p>
            </div>
          ) : diffResult?.error ? (
            <div className="flex-1 flex flex-col items-center justify-center p-12 text-center">
              <AlertCircle size={36} className="text-amber-400 mb-3" />
              <h4 className="text-sm font-bold text-[var(--text-primary)] mb-1">
                Impossible de charger le diff
              </h4>
              <p className="text-xs text-[var(--text-muted)] max-w-md leading-relaxed font-mono">
                {diffResult.error}
              </p>
              <div className="mt-4 flex items-center gap-2">
                <button
                  type="button"
                  onClick={loadDiff}
                  className="px-4 py-2 rounded-xl text-xs font-semibold accent-bg text-white shadow hover:opacity-90 transition-all"
                >
                  Réessayer
                </button>
              </div>
            </div>
          ) : !diffResult?.files || diffResult.files.length === 0 ? (
            <div className="flex-1 flex flex-col items-center justify-center p-12 text-center">
              <div className="w-12 h-12 rounded-2xl bg-emerald-500/15 text-emerald-400 border border-emerald-500/30 flex items-center justify-center shadow-md mb-3">
                <CheckCircle2 size={24} />
              </div>
              <h4 className="text-sm font-bold text-[var(--text-primary)] mb-1">
                Aucune modification détectée (Dépôt propre)
              </h4>
              <p className="text-xs text-[var(--text-muted)] max-w-md leading-relaxed">
                La branche <code className="text-indigo-300 font-mono font-semibold">{diffResult?.branch || diffTask.branchName || 'active'}</code> est identique à <code className="text-slate-300 font-mono font-semibold">{diffResult?.baseBranch || 'main'}</code> ou toutes les modifications ont déjà été commitées et mergées.
              </p>
              {diffTask.prUrl && (
                <div className="mt-4">
                  <a
                    href={diffTask.prUrl}
                    target="_blank"
                    rel="noreferrer"
                    className="inline-flex items-center gap-1.5 px-4 py-2 rounded-xl text-xs font-semibold bg-purple-500/20 text-purple-300 hover:bg-purple-500/30 border border-purple-500/40 transition-all shadow-xs"
                  >
                    <GitPullRequest size={13} />
                    <span>Consulter la Pull Request en ligne ↗</span>
                  </a>
                </div>
              )}
            </div>
          ) : (
            <>
              {/* Left Sidebar: File List */}
              <div className="w-72 border-r border-[var(--border-color)] bg-[var(--bg-tertiary)]/20 flex flex-col shrink-0 overflow-hidden">
                <div className="p-3 border-b border-[var(--border-color)] flex items-center justify-between">
                  <span className="text-[10px] font-bold uppercase tracking-wider text-[var(--text-muted)] flex items-center gap-1.5">
                    <FolderGit2 size={13} className="text-indigo-400" />
                    Fichiers modifiés ({diffResult.files.length})
                  </span>
                </div>

                <div className="flex-1 overflow-y-auto p-2 space-y-1">
                  {diffResult.files.map(file => {
                    const isSelected = selectedFilePath === file.path
                    const fileName = file.path.split('/').pop() || file.path
                    const dirName = file.path.includes('/') ? file.path.substring(0, file.path.lastIndexOf('/')) : ''

                    return (
                      <button
                        key={file.path}
                        type="button"
                        onClick={() => {
                          setSelectedFilePath(file.path)
                          const el = document.getElementById(`file-diff-${file.path}`)
                          if (el) el.scrollIntoView({ behavior: 'smooth', block: 'start' })
                        }}
                        className={`w-full flex items-center justify-between p-2 rounded-xl text-left transition-all group ${
                          isSelected
                            ? 'bg-[var(--accent-light)] border border-[var(--accent-color)]/50 shadow-xs'
                            : 'hover:bg-[var(--bg-tertiary)] border border-transparent'
                        }`}
                      >
                        <div className="flex items-center gap-2 min-w-0">
                          {renderFileStatusIcon(file.status)}
                          <div className="flex flex-col min-w-0">
                            <span className={`text-xs font-mono font-semibold truncate ${
                              isSelected ? 'text-[var(--text-primary)] font-bold' : 'text-[var(--text-secondary)] group-hover:text-[var(--text-primary)]'
                            }`}>
                              {fileName}
                            </span>
                            {dirName && (
                              <span className="text-[9px] font-mono text-[var(--text-muted)] truncate">
                                {dirName}
                              </span>
                            )}
                          </div>
                        </div>

                        <div className="flex items-center gap-1 shrink-0 text-[10px] font-mono font-bold">
                          {file.additions > 0 && <span className="text-emerald-400">+{file.additions}</span>}
                          {file.deletions > 0 && <span className="text-rose-400">-{file.deletions}</span>}
                        </div>
                      </button>
                    )
                  })}
                </div>
              </div>

              {/* Main Diff Area */}
              <div className="flex-1 overflow-y-auto p-4 space-y-5 bg-[var(--bg-primary)]">
                {diffResult.files.map(file => {
                  const isCopied = copiedPath === file.path

                  return (
                    <div
                      key={file.path}
                      id={`file-diff-${file.path}`}
                      className="rounded-2xl border border-[var(--border-color)] bg-[var(--bg-secondary)] overflow-hidden shadow-xs"
                    >
                      {/* File Card Header */}
                      <div className="flex items-center justify-between px-4 py-2.5 bg-[var(--bg-tertiary)]/70 border-b border-[var(--border-color)]">
                        <div className="flex items-center gap-2 min-w-0">
                          {renderFileStatusIcon(file.status)}
                          <span className="font-mono text-xs font-bold text-[var(--text-primary)] truncate">
                            {file.path}
                          </span>
                          <span className="text-[10px] font-mono px-1.5 py-0.2 rounded bg-[var(--bg-primary)] text-[var(--text-muted)] uppercase border border-[var(--border-color)]">
                            {file.status}
                          </span>
                        </div>

                        <div className="flex items-center gap-2">
                          <div className="flex items-center gap-1 text-[11px] font-mono font-bold">
                            {file.additions > 0 && (
                              <span className="text-emerald-400 bg-emerald-500/10 px-1.5 py-0.2 rounded">
                                +{file.additions}
                              </span>
                            )}
                            {file.deletions > 0 && (
                              <span className="text-rose-400 bg-rose-500/10 px-1.5 py-0.2 rounded">
                                -{file.deletions}
                              </span>
                            )}
                          </div>

                          <button
                            type="button"
                            onClick={e => handleCopyFilePath(file.path, e)}
                            className="p-1 rounded text-[var(--text-muted)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-tertiary)] transition-colors"
                            title="Copier le chemin du fichier"
                          >
                            {isCopied ? <Check size={12} className="text-emerald-400" /> : <Copy size={12} />}
                          </button>
                        </div>
                      </div>

                      {/* Diff Content */}
                      <div className="bg-[var(--bg-primary)]">
                        {renderDiffLines(file.diff)}
                      </div>
                    </div>
                  )
                })}
              </div>
            </>
          )}
        </div>
      </div>
    </div>
  )
}
