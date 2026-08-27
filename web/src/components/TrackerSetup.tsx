import React, { useState } from 'react'
import { Check, Key, Globe, Mail, Loader2, ShieldCheck, X, AlertCircle } from 'lucide-react'
import { useApp } from '../context/AppContext'

/**
 * Premier démarrage : ce qu'il faut savoir avant que quoi que ce soit fonctionne.
 *
 * Sans site, sans e-mail et sans jeton, la synchronisation ne ramène rien, les
 * équipes restent vides et aucune écriture ne part. Jusqu'ici on l'apprenait en
 * synchronisant pour rien. Cet écran demande les trois valeurs, les vérifie
 * auprès de l'instance, et dit qui elles désignent avant d'enregistrer quoi que
 * ce soit.
 *
 * Le jeton peut rester hors de la base. Une base est un fichier qu'on copie,
 * qu'on sauvegarde et qu'on transmet ; un fichier d'environnement à côté n'est
 * pas plus secret en théorie, mais c'est celui qu'on sait ne pas envoyer.
 */
export const TrackerSetup: React.FC<{ onClose: () => void }> = ({ onClose }) => {
  const { checkTrackerCredentials, saveTrackerCredentials, settings, addToast } = useApp()

  const [siteUrl, setSiteUrl] = useState(settings.jiraUrl || '')
  const [email, setEmail] = useState(settings.jiraEmail || '')
  const [token, setToken] = useState('')
  const [storeInFile, setStoreInFile] = useState(true)
  const [isChecking, setIsChecking] = useState(false)
  const [isSaving, setIsSaving] = useState(false)
  const [check, setCheck] = useState<{
    ok: boolean
    error?: string
    identity?: { displayName: string; email?: string; siteUrl: string }
    projects?: { id: string; name: string }[]
  } | null>(null)

  const runCheck = async () => {
    setIsChecking(true)
    setCheck(await checkTrackerCredentials(siteUrl, email, token))
    setIsChecking(false)
  }

  const save = async () => {
    setIsSaving(true)
    const saved = await saveTrackerCredentials(siteUrl, email, token, storeInFile)
    setIsSaving(false)
    if (saved) {
      addToast({
        type: 'success',
        title: 'Tracker configuré',
        description: storeInFile
          ? 'Le jeton est écrit dans le fichier de configuration, hors de la base.'
          : 'Le jeton est enregistré dans la base.',
      })
      onClose()
    }
  }

  const fieldClass =
    'w-full pl-8 pr-3 py-2 text-xs rounded-xl bg-[var(--bg-tertiary)] border border-[var(--border-color)] text-[var(--text-primary)] focus:outline-none focus:border-[var(--accent-color)]'

  return (
    <div className="fixed top-0 left-0 h-[var(--app-h)] w-[var(--app-w)] z-60 flex items-center justify-center p-4 bg-black/70 backdrop-blur-xs">
      <div className="relative w-full max-w-lg rounded-2xl bg-[var(--bg-secondary)] border border-[var(--border-color)] shadow-2xl overflow-hidden flex flex-col max-h-[calc(var(--app-h)*0.9)]">
        <div className="flex items-start justify-between px-5 py-4 border-b border-[var(--border-color)]">
          <div className="min-w-0">
            <h2 className="text-sm font-bold text-[var(--text-primary)]">Connecter votre tracker</h2>
            <p className="text-[11px] text-[var(--text-secondary)] mt-0.5 leading-relaxed">
              Sans ces trois valeurs, la synchronisation ne ramène rien et aucune écriture ne part.
              Elles sont vérifiées avant d'être enregistrées.
            </p>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="p-1.5 rounded-lg text-[var(--text-muted)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-tertiary)] cursor-pointer shrink-0"
            title="Configurer plus tard"
          >
            <X size={16} />
          </button>
        </div>

        <div className="p-5 space-y-3 overflow-y-auto">
          <div>
            <label className="block text-[10px] font-bold uppercase tracking-wider text-[var(--text-muted)] mb-1">
              Site Jira
            </label>
            <div className="relative">
              <input
                type="text"
                value={siteUrl}
                onChange={e => setSiteUrl(e.target.value)}
                placeholder="mon-org.atlassian.net"
                className={fieldClass}
              />
              <Globe size={13} className="absolute left-2.5 top-2.5 text-[var(--accent-color)]" />
            </div>
          </div>

          <div>
            <label className="block text-[10px] font-bold uppercase tracking-wider text-[var(--text-muted)] mb-1">
              E-mail Atlassian
            </label>
            <div className="relative">
              <input
                type="email"
                value={email}
                onChange={e => setEmail(e.target.value)}
                placeholder="prenom.nom@exemple.com"
                className={fieldClass}
              />
              <Mail size={13} className="absolute left-2.5 top-2.5 text-[var(--accent-color)]" />
            </div>
          </div>

          <div>
            <label className="block text-[10px] font-bold uppercase tracking-wider text-[var(--text-muted)] mb-1">
              Jeton d'API
            </label>
            <div className="relative">
              <input
                type="password"
                value={token}
                onChange={e => setToken(e.target.value)}
                placeholder={settings.jiraApiTokenSet ? 'Déjà configuré, laissez vide pour le garder' : 'Collez le jeton'}
                className={fieldClass}
              />
              <Key size={13} className="absolute left-2.5 top-2.5 text-[var(--accent-color)]" />
            </div>
            <span className="text-[9.5px] text-[var(--text-muted)] block mt-1">
              À créer sur id.atlassian.com, section jetons d'API.
            </span>
          </div>

          <label className="flex items-start gap-2 text-[10.5px] text-[var(--text-secondary)] cursor-pointer">
            <input
              type="checkbox"
              checked={storeInFile}
              onChange={e => setStoreInFile(e.target.checked)}
              className="mt-0.5 accent-[var(--accent-color)]"
            />
            <span>
              Écrire le jeton dans un fichier de configuration plutôt que dans la base. La base est
              un fichier qu'on copie et qu'on sauvegarde ; le fichier de configuration, lui, est
              celui qu'on sait ne pas transmettre.
            </span>
          </label>

          {check && (
            <div
              className="p-3 rounded-xl border text-[11px] leading-relaxed"
              style={{
                background: check.ok ? 'rgb(var(--status-ok-rgb) / 0.1)' : 'rgb(var(--status-danger-rgb) / 0.1)',
                borderColor: check.ok ? 'rgb(var(--status-ok-rgb) / 0.35)' : 'rgb(var(--status-danger-rgb) / 0.35)',
                color: check.ok ? 'var(--status-ok)' : 'var(--status-danger)',
              }}
            >
              {check.ok ? (
                <>
                  <div className="flex items-center gap-1.5 font-bold">
                    <ShieldCheck size={13} />
                    Connecté comme {check.identity?.displayName}
                  </div>
                  {check.projects && check.projects.length > 0 && (
                    <div className="mt-1 text-[var(--text-secondary)]">
                      {check.projects.length} projet(s) visibles, dont{' '}
                      <span className="font-mono">
                        {check.projects.slice(0, 5).map(p => p.id).join(', ')}
                      </span>
                    </div>
                  )}
                </>
              ) : (
                <div className="flex items-start gap-1.5">
                  <AlertCircle size={13} className="shrink-0 mt-0.5" />
                  <span>{check.error}</span>
                </div>
              )}
            </div>
          )}
        </div>

        <div className="flex items-center justify-between gap-2 px-5 py-3.5 border-t border-[var(--border-color)] bg-[var(--bg-tertiary)]/40">
          <button
            type="button"
            onClick={onClose}
            className="px-3 py-1.5 rounded-xl text-xs font-medium text-[var(--text-secondary)] hover:text-[var(--text-primary)] cursor-pointer"
          >
            Plus tard
          </button>
          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={runCheck}
              disabled={isChecking || !siteUrl.trim() || !email.trim()}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-xl text-xs font-semibold bg-[var(--bg-tertiary)] border border-[var(--border-color)] text-[var(--text-secondary)] hover:text-[var(--text-primary)] disabled:opacity-40 cursor-pointer"
            >
              {isChecking ? <Loader2 size={13} className="animate-spin" /> : <ShieldCheck size={13} />}
              Vérifier
            </button>
            <button
              type="button"
              onClick={save}
              disabled={isSaving || !check?.ok}
              title={check?.ok ? 'Enregistrer ces accès' : "Vérifiez d'abord les accès"}
              className="flex items-center gap-1.5 px-4 py-1.5 rounded-xl text-xs font-bold text-white accent-bg shadow-xs hover:opacity-90 disabled:opacity-40 cursor-pointer"
            >
              {isSaving ? <Loader2 size={13} className="animate-spin" /> : <Check size={13} />}
              Enregistrer
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
