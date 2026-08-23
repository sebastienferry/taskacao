import React, { useState, useEffect, useMemo } from 'react'
import {
  X,
  GitBranch,
  Search,
  Check,
  Plus,
  RefreshCw,
  FolderGit2,
  Cloud,
  Loader2,
  ArrowRight,
} from 'lucide-react'
import { useApp } from '../context/AppContext'

export const BranchSwitcherModal: React.FC = () => {
  const {
    isBranchModalOpen,
    setIsBranchModalOpen,
    gitBranches,
    fetchGitBranches,
    switchGitBranch,
    gitStatus,
    currentProject,
    addToast,
  } = useApp()

  const [search, setSearch] = useState('')
  const [isRefreshing, setIsRefreshing] = useState(false)
  const [switchingBranch, setSwitchingBranch] = useState<string | null>(null)

  useEffect(() => {
    if (isBranchModalOpen) {
      setSearch('')
      setSwitchingBranch(null)
      fetchGitBranches()
    }
  }, [isBranchModalOpen, fetchGitBranches])

  const handleRefresh = async () => {
    setIsRefreshing(true)
    try {
      await fetchGitBranches()
      addToast({
        type: 'info',
        title: 'Branches actualisées',
        description: 'La liste des branches locales et distantes a été mise à jour.',
      })
    } finally {
      setIsRefreshing(false)
    }
  }

  const handleSwitch = async (branchName: string, create: boolean = false) => {
    if (switchingBranch) return
    setSwitchingBranch(branchName)
    try {
      const success = await switchGitBranch(branchName, create)
      if (success) {
        setIsBranchModalOpen(false)
      }
    } finally {
      setSwitchingBranch(null)
    }
  }

  const currentBranch = gitBranches?.currentBranch || gitStatus?.branch || 'main'

  // Filtered branches
  const filteredBranches = useMemo(() => {
    if (!gitBranches?.branches) return []
    const q = search.trim().toLowerCase()
    if (!q) return gitBranches.branches

    return gitBranches.branches.filter(
      b =>
        b.name.toLowerCase().includes(q) ||
        (b.commit && b.commit.toLowerCase().includes(q)) ||
        (b.message && b.message.toLowerCase().includes(q))
    )
  }, [gitBranches?.branches, search])

  // Split into local vs remote
  const localBranches = useMemo(
    () => filteredBranches.filter(b => !b.isRemote),
    [filteredBranches]
  )

  const remoteBranches = useMemo(
    () => filteredBranches.filter(b => b.isRemote),
    [filteredBranches]
  )

  // Check if search exact matches existing
  const exactMatchExists = useMemo(() => {
    const q = search.trim().toLowerCase()
    if (!q || !gitBranches?.branches) return true
    return gitBranches.branches.some(b => b.name.toLowerCase() === q)
  }, [gitBranches?.branches, search])

  const canCreateBranch = search.trim().length > 0 && !exactMatchExists

  if (!isBranchModalOpen) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-xs animate-in fade-in duration-200">
      <div
        className="relative w-full max-w-lg rounded-2xl bg-[var(--bg-secondary)] border border-[var(--border-color)] shadow-2xl overflow-hidden flex flex-col max-h-[85vh] animate-in zoom-in-95 duration-150"
        onClick={e => e.stopPropagation()}
      >
        {/* Header */}
        <div className="p-4 border-b border-[var(--border-color)] flex items-center justify-between bg-[var(--bg-primary)]/50">
          <div className="flex items-center gap-2.5">
            <div className="w-8 h-8 rounded-xl bg-cyan-500/15 border border-cyan-500/30 flex items-center justify-center text-cyan-400">
              <GitBranch size={16} />
            </div>
            <div>
              <h2 className="text-sm font-bold text-[var(--text-primary)] flex items-center gap-2">
                <span>Bascule de branche Git</span>
                {currentProject && (
                  <span className="text-[10px] font-mono font-medium px-2 py-0.5 rounded-full bg-[var(--accent-light)] accent-text">
                    {currentProject.name}
                  </span>
                )}
              </h2>
              <p className="text-[11px] text-[var(--text-muted)] font-mono truncate max-w-[320px]">
                {gitBranches?.repoPath || currentProject?.repoPath || 'Dépôt Git actif'}
              </p>
            </div>
          </div>

          <div className="flex items-center gap-1.5">
            <button
              onClick={handleRefresh}
              disabled={isRefreshing}
              className="p-1.5 rounded-lg text-[var(--text-secondary)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-tertiary)] transition-colors cursor-pointer disabled:opacity-50"
              title="Rafraîchir les branches"
            >
              <RefreshCw size={14} className={isRefreshing ? 'animate-spin text-cyan-400' : ''} />
            </button>
            <button
              onClick={() => setIsBranchModalOpen(false)}
              className="p-1.5 rounded-lg text-[var(--text-muted)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-tertiary)] transition-colors cursor-pointer"
            >
              <X size={16} />
            </button>
          </div>
        </div>

        {/* Current Active Branch Highlight */}
        <div className="px-4 py-2.5 bg-cyan-500/10 border-b border-cyan-500/20 flex items-center justify-between">
          <div className="flex items-center gap-2 min-w-0">
            <span className="text-[10px] font-bold uppercase tracking-wider text-cyan-400 shrink-0">
              Branche Active :
            </span>
            <span className="text-xs font-mono font-bold text-cyan-300 dark:text-cyan-200 truncate">
              {currentBranch}
            </span>
          </div>

          <div className="flex items-center gap-1.5 shrink-0">
            {gitStatus?.isClean ? (
              <span className="inline-flex items-center gap-1 text-[10px] font-medium text-emerald-400 bg-emerald-500/15 border border-emerald-500/30 px-2 py-0.5 rounded-md">
                <Check size={10} />
                <span>Arbre propre</span>
              </span>
            ) : (
              <span className="inline-flex items-center gap-1 text-[10px] font-medium text-amber-400 bg-amber-500/15 border border-amber-500/30 px-2 py-0.5 rounded-md">
                <span>{(gitStatus?.modifiedCount || 0) + (gitStatus?.untrackedCount || 0)} modifiés</span>
              </span>
            )}
          </div>
        </div>

        {/* Search Input */}
        <div className="p-3 border-b border-[var(--border-color)] bg-[var(--bg-primary)]">
          <div className="relative">
            <Search size={14} className="absolute left-3 top-2.5 text-[var(--text-muted)]" />
            <input
              type="text"
              value={search}
              onChange={e => setSearch(e.target.value)}
              placeholder="Filtrer ou créer une branche (ex: feature/mon-dev)..."
              className="w-full pl-9 pr-8 py-1.5 text-xs rounded-xl bg-[var(--bg-secondary)] border border-[var(--border-color)] text-[var(--text-primary)] placeholder-[var(--text-muted)] focus:outline-none focus:border-[var(--accent-color)] font-mono text-[12px]"
              autoFocus
              onKeyDown={e => {
                if (e.key === 'Enter') {
                  if (canCreateBranch) {
                    handleSwitch(search.trim(), true)
                  } else if (localBranches.length > 0) {
                    handleSwitch(localBranches[0].name)
                  }
                }
              }}
            />
            {search && (
              <button
                onClick={() => setSearch('')}
                className="absolute right-2.5 top-2.5 text-[var(--text-muted)] hover:text-[var(--text-primary)] cursor-pointer"
              >
                <X size={13} />
              </button>
            )}
          </div>

          {/* Quick create action banner if branch does not exist */}
          {canCreateBranch && (
            <button
              onClick={() => handleSwitch(search.trim(), true)}
              disabled={switchingBranch !== null}
              className="mt-2 w-full p-2 rounded-xl bg-[var(--accent-color)]/15 border border-[var(--accent-color)]/40 hover:bg-[var(--accent-color)]/25 flex items-center justify-between text-xs accent-text font-medium transition-colors cursor-pointer"
            >
              <div className="flex items-center gap-2 font-mono">
                <Plus size={14} className="text-[var(--accent-color)]" />
                <span>Créer et basculer sur : <strong>{search.trim()}</strong></span>
              </div>
              <ArrowRight size={13} />
            </button>
          )}
        </div>

        {/* Branch List */}
        <div className="p-2 overflow-y-auto flex-1 divide-y divide-[var(--border-color)]/40 space-y-3">
          {/* Local Branches */}
          <div>
            <div className="px-2 py-1 flex items-center justify-between">
              <span className="text-[10px] font-bold uppercase tracking-wider text-[var(--text-muted)]">
                Branches Locales ({localBranches.length})
              </span>
            </div>

            <div className="mt-1 space-y-1">
              {localBranches.length === 0 ? (
                <div className="p-3 text-center text-xs text-[var(--text-muted)]">
                  Aucune branche locale trouvée
                </div>
              ) : (
                localBranches.map(branch => {
                  const isCurrent = branch.name === currentBranch
                  const isSwitching = switchingBranch === branch.name

                  return (
                    <div
                      key={`local-${branch.name}`}
                      onClick={() => !isCurrent && handleSwitch(branch.name)}
                      className={`group p-2 rounded-xl flex items-center justify-between gap-2 text-xs transition-all ${
                        isCurrent
                          ? 'bg-cyan-500/10 border border-cyan-500/30 text-cyan-300 font-semibold'
                          : 'hover:bg-[var(--bg-tertiary)] border border-transparent text-[var(--text-primary)] cursor-pointer'
                      }`}
                    >
                      <div className="flex items-center gap-2.5 min-w-0">
                        <FolderGit2
                          size={14}
                          className={isCurrent ? 'text-cyan-400 shrink-0' : 'text-[var(--text-muted)] group-hover:text-[var(--text-primary)] shrink-0'}
                        />
                        <div className="min-w-0">
                          <div className="font-mono text-[12px] truncate flex items-center gap-1.5">
                            <span>{branch.name}</span>
                            {isCurrent && (
                              <span className="text-[9px] font-sans font-bold px-1.5 py-0.2 rounded bg-cyan-500/20 text-cyan-300 border border-cyan-500/30">
                                active
                              </span>
                            )}
                          </div>
                          {branch.message && (
                            <p className="text-[10px] text-[var(--text-muted)] truncate max-w-[340px]">
                              {branch.message}
                            </p>
                          )}
                        </div>
                      </div>

                      <div className="flex items-center gap-1.5 shrink-0">
                        {branch.commit && (
                          <span className="text-[10px] font-mono text-[var(--text-muted)] bg-[var(--bg-primary)] px-1.5 py-0.5 rounded border border-[var(--border-color)]">
                            {branch.commit}
                          </span>
                        )}

                        {isSwitching ? (
                          <Loader2 size={13} className="animate-spin text-cyan-400" />
                        ) : isCurrent ? (
                          <Check size={14} className="text-cyan-400" />
                        ) : (
                          <button
                            type="button"
                            className="opacity-0 group-hover:opacity-100 px-2 py-0.5 rounded-md text-[10px] font-medium bg-[var(--accent-color)] text-white transition-opacity"
                          >
                            Bascule
                          </button>
                        )}
                      </div>
                    </div>
                  )
                })
              )}
            </div>
          </div>

          {/* Remote Branches */}
          {remoteBranches.length > 0 && (
            <div className="pt-2">
              <div className="px-2 py-1 flex items-center justify-between">
                <span className="text-[10px] font-bold uppercase tracking-wider text-[var(--text-muted)]">
                  Branches Distantes / Remote ({remoteBranches.length})
                </span>
              </div>

              <div className="mt-1 space-y-1">
                {remoteBranches.map(branch => {
                  const isSwitching = switchingBranch === branch.name

                  return (
                    <div
                      key={`remote-${branch.name}`}
                      onClick={() => handleSwitch(branch.name)}
                      className="group p-2 rounded-xl flex items-center justify-between gap-2 text-xs hover:bg-[var(--bg-tertiary)] border border-transparent text-[var(--text-secondary)] hover:text-[var(--text-primary)] cursor-pointer transition-all"
                    >
                      <div className="flex items-center gap-2.5 min-w-0">
                        <Cloud size={14} className="text-[var(--text-muted)] group-hover:text-blue-400 shrink-0" />
                        <div className="min-w-0">
                          <div className="font-mono text-[12px] truncate text-[var(--text-primary)]">
                            {branch.name}
                          </div>
                          {branch.message && (
                            <p className="text-[10px] text-[var(--text-muted)] truncate max-w-[340px]">
                              {branch.message}
                            </p>
                          )}
                        </div>
                      </div>

                      <div className="flex items-center gap-1.5 shrink-0">
                        {isSwitching ? (
                          <Loader2 size={13} className="animate-spin text-cyan-400" />
                        ) : (
                          <button
                            type="button"
                            className="opacity-0 group-hover:opacity-100 px-2 py-0.5 rounded-md text-[10px] font-medium bg-[var(--bg-primary)] border border-[var(--border-color)] hover:border-[var(--accent-color)] text-[var(--text-primary)] transition-opacity"
                          >
                            Checkout
                          </button>
                        )}
                      </div>
                    </div>
                  )
                })}
              </div>
            </div>
          )}
        </div>

        {/* Footer info */}
        <div className="p-3 border-t border-[var(--border-color)] bg-[var(--bg-primary)]/70 flex items-center justify-between text-[11px] text-[var(--text-muted)]">
          <span>
            {localBranches.length + remoteBranches.length} branches au total
          </span>
          <span className="font-mono text-[10px]">
            Appuyez sur <kbd className="px-1 py-0.5 rounded bg-[var(--bg-tertiary)] border border-[var(--border-color)]">Entrée</kbd> pour basculer
          </span>
        </div>
      </div>
    </div>
  )
}
