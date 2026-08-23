import React, { useState, useEffect } from 'react'
import {
  X,
  User,
  Mail,
  Palette,
  Sun,
  Moon,
  Globe,
  Check,
  RotateCcw,
  Sliders,
  PanelRight,
  Square,
  Code2,
} from 'lucide-react'
import { useApp } from '../context/AppContext'
import type { AccentColor, Theme, Language, Density, ViewMode, DetailMode } from '../types'

export const ProfileModal: React.FC = () => {
  const {
    isProfileOpen,
    setIsProfileOpen,
    settings,
    updateSettings,
    reseedDemo,
    t,
  } = useApp()

  // Appearance & User
  const [userName, setUserName] = useState(settings.userName)
  const [userEmail, setUserEmail] = useState(settings.userEmail)
  const [theme, setTheme] = useState<Theme>(settings.theme)
  const [accentColor, setAccentColor] = useState<AccentColor>(settings.accentColor)
  const [language, setLanguage] = useState<Language>(settings.language)
  const [density, setDensity] = useState<Density>(settings.density)
  const [defaultView, setDefaultView] = useState<ViewMode>(settings.defaultView)
  const [detailMode, setDetailMode] = useState<DetailMode>(settings.detailMode || 'panel')
  const [editorCommand, setEditorCommand] = useState(settings.editorCommand || 'code')

  useEffect(() => {
    if (isProfileOpen) {
      setUserName(settings.userName)
      setUserEmail(settings.userEmail)
      setTheme(settings.theme)
      setAccentColor(settings.accentColor)
      setLanguage(settings.language)
      setDensity(settings.density)
      setDefaultView(settings.defaultView)
      setDetailMode(settings.detailMode || 'panel')
      setEditorCommand(settings.editorCommand || 'code')
    }
  }, [isProfileOpen, settings])

  useEffect(() => {
    if (!isProfileOpen) return
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        setIsProfileOpen(false)
      }
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [isProfileOpen, setIsProfileOpen])

  if (!isProfileOpen) return null

  const accents: { id: AccentColor; name: string; hex: string }[] = [
    { id: 'indigo', name: t.profileModal.accents.indigo, hex: '#6366f1' },
    { id: 'violet', name: t.profileModal.accents.violet, hex: '#8b5cf6' },
    { id: 'emerald', name: t.profileModal.accents.emerald, hex: '#10b981' },
    { id: 'amber', name: t.profileModal.accents.amber, hex: '#f59e0b' },
    { id: 'rose', name: t.profileModal.accents.rose, hex: '#f43f5e' },
    { id: 'cyan', name: t.profileModal.accents.cyan, hex: '#06b6d4' },
    { id: 'blue', name: t.profileModal.accents.blue, hex: '#3b82f6' },
    { id: 'orange', name: t.profileModal.accents.orange, hex: '#f97316' },
    { id: 'neon-cyan', name: t.profileModal.accents['neon-cyan'], hex: '#00f0ff' },
    { id: 'neon-purple', name: t.profileModal.accents['neon-purple'], hex: '#d946ef' },
    { id: 'neon-green', name: t.profileModal.accents['neon-green'], hex: '#10f070' },
    { id: 'neon-amber', name: t.profileModal.accents['neon-amber'], hex: '#ffd000' },
  ]

  const densities: { id: Density; label: string; desc: string }[] = [
    { id: 'compact', label: 'Compact', desc: '13px font, padding réduit' },
    { id: 'standard', label: 'Standard', desc: '14px font, équilibre optimal' },
    { id: 'comfortable', label: 'Confortable', desc: '15px font, grands espacements' },
  ]

  const handleSave = async () => {
    await updateSettings({
      userName: userName.trim(),
      userEmail: userEmail.trim(),
      theme,
      accentColor,
      language,
      density,
      defaultView,
      detailMode,
      editorCommand: editorCommand.trim() || 'code',
    })
    setIsProfileOpen(false)
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-xs animate-in fade-in duration-200">
      <div className="relative w-full max-w-xl rounded-2xl bg-[var(--bg-secondary)] border border-[var(--border-color)] shadow-2xl overflow-hidden flex flex-col max-h-[90vh]">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-[var(--border-color)] bg-[var(--bg-tertiary)]/30 shrink-0">
          <div className="flex items-center gap-2.5">
            <div className="w-8 h-8 rounded-xl accent-bg text-white flex items-center justify-center shadow">
              <Sliders size={16} />
            </div>
            <div>
              <h3 className="text-sm font-bold text-[var(--text-primary)]">
                {t.profileModal.title}
              </h3>
              <p className="text-[11px] text-[var(--text-muted)]">
                Personnalisez vos préférences d'affichage et votre profil
              </p>
            </div>
          </div>
          <button
            onClick={() => setIsProfileOpen(false)}
            className="p-1.5 rounded-lg text-[var(--text-muted)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-tertiary)] transition-colors cursor-pointer"
          >
            <X size={17} />
          </button>
        </div>

        {/* Modal Form Body */}
        <div className="p-6 overflow-y-auto space-y-6 flex-1 text-xs">
          {/* User info */}
          <div className="space-y-3">
            <div className="text-[11px] font-bold uppercase tracking-wider text-[var(--text-muted)] flex items-center gap-1.5">
              <User size={13} className="text-cyan-400" />
              <span>{t.profileModal.userSection}</span>
            </div>
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <div>
                <label className="block text-[11px] font-medium text-[var(--text-secondary)] mb-1">
                  {t.profileModal.name}
                </label>
                <div className="relative">
                  <input
                    type="text"
                    value={userName}
                    onChange={e => setUserName(e.target.value)}
                    className="w-full pl-8 pr-3 py-2 rounded-xl bg-[var(--bg-tertiary)] border border-[var(--border-color)] text-[var(--text-primary)] focus:outline-none focus:border-[var(--accent-color)]"
                  />
                  <User size={14} className="absolute left-2.5 top-2.5 text-[var(--text-muted)]" />
                </div>
              </div>

              <div>
                <label className="block text-[11px] font-medium text-[var(--text-secondary)] mb-1">
                  {t.profileModal.email}
                </label>
                <div className="relative">
                  <input
                    type="email"
                    value={userEmail}
                    onChange={e => setUserEmail(e.target.value)}
                    className="w-full pl-8 pr-3 py-2 rounded-xl bg-[var(--bg-tertiary)] border border-[var(--border-color)] text-[var(--text-primary)] focus:outline-none focus:border-[var(--accent-color)]"
                  />
                  <Mail size={14} className="absolute left-2.5 top-2.5 text-[var(--text-muted)]" />
                </div>
              </div>
            </div>
          </div>

          {/* Accent Color */}
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <div className="text-[11px] font-bold uppercase tracking-wider text-[var(--text-muted)] flex items-center gap-1.5">
                <Palette size={13} />
                <span>{t.profileModal.accentColor}</span>
              </div>
              <span className="text-[11px] font-semibold accent-text capitalize">
                {accentColor}
              </span>
            </div>

            <div className="grid grid-cols-4 sm:grid-cols-8 gap-2">
              {accents.map(acc => {
                const isSelected = accentColor === acc.id
                return (
                  <button
                    key={acc.id}
                    type="button"
                    onClick={() => setAccentColor(acc.id)}
                    className={`h-10 rounded-xl flex flex-col items-center justify-center relative transition-all duration-150 cursor-pointer ${
                      isSelected
                        ? 'ring-2 ring-white scale-105 shadow-md'
                        : 'opacity-85 hover:opacity-100 hover:scale-102'
                    }`}
                    style={{ backgroundColor: acc.hex }}
                    title={acc.name}
                  >
                    {isSelected && <Check size={16} className="text-white drop-shadow" />}
                  </button>
                )
              })}
            </div>
          </div>

          {/* Theme & Language */}
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div className="space-y-2">
              <label className="block text-[11px] font-bold uppercase tracking-wider text-[var(--text-muted)]">
                {t.profileModal.theme}
              </label>
              <div className="grid grid-cols-2 gap-2">
                <button
                  type="button"
                  onClick={() => setTheme('dark')}
                  className={`flex items-center justify-center gap-2 py-2 rounded-xl border font-medium transition-all cursor-pointer ${
                    theme === 'dark'
                      ? 'bg-[var(--accent-light)] border-[var(--accent-color)] accent-text shadow-xs'
                      : 'bg-[var(--bg-tertiary)] border-[var(--border-color)] text-[var(--text-secondary)] hover:text-[var(--text-primary)]'
                  }`}
                >
                  <Moon size={14} />
                  <span>{t.profileModal.themes.dark}</span>
                </button>

                <button
                  type="button"
                  onClick={() => setTheme('light')}
                  className={`flex items-center justify-center gap-2 py-2 rounded-xl border font-medium transition-all cursor-pointer ${
                    theme === 'light'
                      ? 'bg-[var(--accent-light)] border-[var(--accent-color)] accent-text shadow-xs'
                      : 'bg-[var(--bg-tertiary)] border-[var(--border-color)] text-[var(--text-secondary)] hover:text-[var(--text-primary)]'
                  }`}
                >
                  <Sun size={14} />
                  <span>{t.profileModal.themes.light}</span>
                </button>
              </div>
            </div>

            <div className="space-y-2">
              <label className="block text-[11px] font-bold uppercase tracking-wider text-[var(--text-muted)]">
                {t.profileModal.language}
              </label>
              <div className="grid grid-cols-2 gap-2">
                <button
                  type="button"
                  onClick={() => setLanguage('fr')}
                  className={`flex items-center justify-center gap-1.5 py-2 rounded-xl border font-medium transition-all cursor-pointer ${
                    language === 'fr'
                      ? 'bg-[var(--accent-light)] border-[var(--accent-color)] accent-text shadow-xs'
                      : 'bg-[var(--bg-tertiary)] border-[var(--border-color)] text-[var(--text-secondary)] hover:text-[var(--text-primary)]'
                  }`}
                >
                  <Globe size={14} />
                  <span>FR</span>
                </button>

                <button
                  type="button"
                  onClick={() => setLanguage('en')}
                  className={`flex items-center justify-center gap-1.5 py-2 rounded-xl border font-medium transition-all cursor-pointer ${
                    language === 'en'
                      ? 'bg-[var(--accent-light)] border-[var(--accent-color)] accent-text shadow-xs'
                      : 'bg-[var(--bg-tertiary)] border-[var(--border-color)] text-[var(--text-secondary)] hover:text-[var(--text-primary)]'
                  }`}
                >
                  <Globe size={14} />
                  <span>EN</span>
                </button>
              </div>
            </div>
          </div>

          {/* Story Detail Mode (Right Panel vs Modal) */}
          <div className="space-y-2">
            <label className="block text-[11px] font-bold uppercase tracking-wider text-[var(--text-muted)]">
              {t.profileModal.detailMode}
            </label>
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
              <button
                type="button"
                onClick={() => setDetailMode('panel')}
                className={`flex items-center gap-3 p-3 rounded-xl border text-left transition-all cursor-pointer ${
                  detailMode === 'panel'
                    ? 'bg-[var(--accent-light)] border-[var(--accent-color)] accent-text ring-2 ring-[var(--accent-glow)]'
                    : 'bg-[var(--bg-tertiary)] border-[var(--border-color)] text-[var(--text-secondary)] hover:border-[var(--text-muted)]'
                }`}
              >
                <div className="w-8 h-8 rounded-lg bg-[var(--bg-secondary)] flex items-center justify-center text-indigo-400 border border-[var(--border-color)] shrink-0">
                  <PanelRight size={18} />
                </div>
                <div className="truncate">
                  <div className="font-bold text-xs">{t.profileModal.detailModes.panel}</div>
                  <div className="text-[10px] text-[var(--text-muted)] opacity-80">Glissement latéral à droite</div>
                </div>
              </button>

              <button
                type="button"
                onClick={() => setDetailMode('modal')}
                className={`flex items-center gap-3 p-3 rounded-xl border text-left transition-all cursor-pointer ${
                  detailMode === 'modal'
                    ? 'bg-[var(--accent-light)] border-[var(--accent-color)] accent-text ring-2 ring-[var(--accent-glow)]'
                    : 'bg-[var(--bg-tertiary)] border-[var(--border-color)] text-[var(--text-secondary)] hover:border-[var(--text-muted)]'
                }`}
              >
                <div className="w-8 h-8 rounded-lg bg-[var(--bg-secondary)] flex items-center justify-center text-purple-400 border border-[var(--border-color)] shrink-0">
                  <Square size={18} />
                </div>
                <div className="truncate">
                  <div className="font-bold text-xs">{t.profileModal.detailModes.modal}</div>
                  <div className="text-[10px] text-[var(--text-muted)] opacity-80">Boîte de dialogue au centre</div>
                </div>
              </button>
            </div>
          </div>

          {/* Density */}
          <div className="space-y-2">
            <label className="block text-[11px] font-bold uppercase tracking-wider text-[var(--text-muted)]">
              {t.profileModal.density}
            </label>
            <div className="grid grid-cols-1 sm:grid-cols-3 gap-2">
              {densities.map(d => {
                const isSelected = density === d.id
                return (
                  <button
                    key={d.id}
                    type="button"
                    onClick={() => setDensity(d.id)}
                    className={`p-2.5 rounded-xl border text-left transition-all cursor-pointer ${
                      isSelected
                        ? 'bg-[var(--accent-light)] border-[var(--accent-color)] accent-text shadow-xs'
                        : 'bg-[var(--bg-tertiary)] border-[var(--border-color)] text-[var(--text-secondary)] hover:border-[var(--text-muted)]'
                    }`}
                  >
                    <div className="font-bold text-xs">{d.label}</div>
                    <div className="text-[10px] opacity-75 mt-0.5">{d.desc}</div>
                  </button>
                )
              })}
            </div>
          </div>

          {/* Default Code Editor */}
          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <label className="block text-[11px] font-bold uppercase tracking-wider text-[var(--text-muted)] flex items-center gap-1.5">
                <Code2 size={13} className="text-[var(--accent-color)]" />
                <span>Éditeur de code (Ouvrir le dossier / code)</span>
              </label>
              <span className="text-[10px] text-[var(--text-muted)] font-mono">Défaut : code (VS Code)</span>
            </div>
            
            {/* Quick Presets */}
            <div className="grid grid-cols-3 sm:grid-cols-6 gap-1.5">
              {[
                { cmd: 'code', label: 'VS Code' },
                { cmd: 'cursor', label: 'Cursor' },
                { cmd: 'zed', label: 'Zed' },
                { cmd: 'subl', label: 'Sublime' },
                { cmd: 'idea', label: 'IntelliJ' },
                { cmd: 'webstorm', label: 'WebStorm' },
              ].map(preset => {
                const isSelected = editorCommand === preset.cmd
                return (
                  <button
                    key={preset.cmd}
                    type="button"
                    onClick={() => setEditorCommand(preset.cmd)}
                    className={`py-1.5 px-2 rounded-lg text-xs font-semibold border transition-all cursor-pointer text-center ${
                      isSelected
                        ? 'bg-[var(--accent-light)] border-[var(--accent-color)] accent-text shadow-xs'
                        : 'bg-[var(--bg-tertiary)] border-[var(--border-color)] text-[var(--text-secondary)] hover:text-[var(--text-primary)] hover:border-[var(--text-muted)]'
                    }`}
                  >
                    {preset.label}
                  </button>
                )
              })}
            </div>

            {/* Custom Command Input */}
            <div className="flex items-center gap-2 pt-1">
              <input
                type="text"
                value={editorCommand}
                onChange={e => setEditorCommand(e.target.value)}
                placeholder="Ex: code, cursor, zed"
                className="w-full px-3 py-1.5 text-xs font-mono rounded-xl bg-[var(--bg-tertiary)] border border-[var(--border-color)] text-[var(--text-primary)] focus:outline-none focus:border-[var(--accent-color)] transition-all"
              />
            </div>
          </div>

          {/* Demo Reset */}
          <div className="pt-2">
            <button
              type="button"
              onClick={() => {
                reseedDemo()
                setIsProfileOpen(false)
              }}
              className="w-full flex items-center justify-center gap-2 py-2 px-3 rounded-xl text-xs font-semibold text-rose-400 bg-rose-500/10 hover:bg-rose-500/20 border border-rose-500/30 transition-colors cursor-pointer"
            >
              <RotateCcw size={14} />
              <span>{t.profileModal.reseedBtn}</span>
            </button>
          </div>
        </div>

        {/* Footer */}
        <div className="flex items-center justify-end gap-2.5 px-6 py-4 border-t border-[var(--border-color)] bg-[var(--bg-tertiary)]/40 shrink-0">
          <button
            type="button"
            onClick={() => setIsProfileOpen(false)}
            className="px-4 py-2 rounded-xl text-xs font-medium text-[var(--text-secondary)] hover:bg-[var(--bg-tertiary)] transition-colors cursor-pointer"
          >
            {t.taskModal.cancel}
          </button>
          <button
            type="button"
            onClick={handleSave}
            className="px-5 py-2 rounded-xl text-xs font-semibold text-white accent-bg shadow hover:opacity-90 active:scale-95 transition-all cursor-pointer"
          >
            {t.profileModal.save}
          </button>
        </div>
      </div>
    </div>
  )
}
