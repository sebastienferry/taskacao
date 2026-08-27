import React, { useMemo, useRef, useState } from 'react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { Bold, Italic, Code, Link2, List, ListOrdered, Quote, Heading2, Eye, Pencil, CheckSquare } from 'lucide-react'

/**
 * Rendu et édition du Markdown, pour les descriptions et les commentaires.
 *
 * Les descriptions que produisent les skills, celles que le tracker renvoie et
 * celles qu'on écrit à la main sont du Markdown depuis toujours : elles étaient
 * simplement affichées telles quelles, dièses et astérisques compris.
 *
 * Le rendu passe par react-markdown, qui construit des noeuds React au lieu
 * d'injecter du HTML : un ticket peut contenir n'importe quoi, y compris du HTML
 * collé depuis un mail, et rien de tout cela ne doit s'exécuter ici.
 */

/** Rendu seul, pour un commentaire ou une description déjà écrite. */
export const MarkdownView: React.FC<{ children: string; className?: string; compact?: boolean }> = ({
  children,
  className,
  compact,
}) => {
  const text = (children || '').trim()
  if (!text) return null

  return (
    <div className={`markdown-body ${compact ? 'markdown-compact' : ''} ${className || ''}`}>
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        components={{
          // Un lien d'un ticket mène ailleurs : il s'ouvre à côté, jamais en
          // remplaçant l'application.
          a: props => <a {...props} target="_blank" rel="noreferrer" />,
        }}
      >
        {text}
      </ReactMarkdown>
    </div>
  )
}

type Snippet = {
  icon: React.ReactNode
  title: string
  /** Texte inséré avant la sélection. */
  before: string
  /** Texte inséré après la sélection, vide pour un préfixe de ligne. */
  after?: string
  /** Contenu posé quand rien n'est sélectionné. */
  placeholder?: string
  /** Le préfixe s'applique à chaque ligne sélectionnée (listes, citations). */
  perLine?: boolean
  shortcut?: string
}

const SNIPPETS: Snippet[] = [
  { icon: <Bold size={12} />, title: 'Gras', before: '**', after: '**', placeholder: 'texte', shortcut: 'b' },
  { icon: <Italic size={12} />, title: 'Italique', before: '_', after: '_', placeholder: 'texte', shortcut: 'i' },
  { icon: <Code size={12} />, title: 'Code', before: '`', after: '`', placeholder: 'code' },
  { icon: <Link2 size={12} />, title: 'Lien', before: '[', after: '](url)', placeholder: 'libellé', shortcut: 'k' },
  { icon: <Heading2 size={12} />, title: 'Titre', before: '## ', perLine: true, placeholder: 'Titre' },
  { icon: <List size={12} />, title: 'Liste', before: '- ', perLine: true, placeholder: 'élément' },
  { icon: <ListOrdered size={12} />, title: 'Liste numérotée', before: '1. ', perLine: true, placeholder: 'élément' },
  { icon: <CheckSquare size={12} />, title: 'Case à cocher', before: '- [ ] ', perLine: true, placeholder: 'à faire' },
  { icon: <Quote size={12} />, title: 'Citation', before: '> ', perLine: true, placeholder: 'citation' },
]

/**
 * Éditeur Markdown : une zone de saisie, une barre de mise en forme et un
 * aperçu. L'aperçu est un onglet et non un second panneau, parce que ces champs
 * vivent dans une fiche déjà dense et dans un commentaire de quelques lignes.
 */
export const MarkdownEditor: React.FC<{
  value: string
  onChange: (value: string) => void
  placeholder?: string
  rows?: number
  minHeight?: number
  maxHeight?: number
  disabled?: boolean
  /** Rendu au dessus de l'onglet Aperçu, pour un bouton d'envoi par exemple. */
  actions?: React.ReactNode
  onKeyDown?: (e: React.KeyboardEvent<HTMLTextAreaElement>) => void
}> = ({ value, onChange, placeholder, minHeight = 120, maxHeight, disabled, actions, onKeyDown }) => {
  const [isPreview, setIsPreview] = useState(false)
  const textareaRef = useRef<HTMLTextAreaElement>(null)

  const applySnippet = (snippet: Snippet) => {
    const textarea = textareaRef.current
    if (!textarea || disabled) return

    const start = textarea.selectionStart
    const end = textarea.selectionEnd
    const selected = value.slice(start, end)

    let inserted: string
    let nextStart: number
    let nextEnd: number

    if (snippet.perLine) {
      // Le préfixe se pose en tête de chaque ligne, et sur la ligne courante
      // quand rien n'est sélectionné.
      const lineStart = value.lastIndexOf('\n', Math.max(0, start - 1)) + 1
      const target = selected || snippet.placeholder || ''
      const body = selected ? value.slice(lineStart, end) : target
      inserted = body
        .split('\n')
        .map(line => (line.startsWith(snippet.before) ? line : snippet.before + line))
        .join('\n')
      const from = selected ? lineStart : start
      const to = selected ? end : start
      onChange(value.slice(0, from) + inserted + value.slice(to))
      nextStart = from
      nextEnd = from + inserted.length
    } else {
      const target = selected || snippet.placeholder || ''
      inserted = snippet.before + target + (snippet.after || '')
      onChange(value.slice(0, start) + inserted + value.slice(end))
      // Sans sélection, le curseur se pose sur le mot posé, prêt à être remplacé.
      nextStart = start + snippet.before.length
      nextEnd = nextStart + target.length
    }

    requestAnimationFrame(() => {
      textarea.focus()
      textarea.setSelectionRange(nextStart, nextEnd)
    })
  }

  const shortcuts = useMemo(() => SNIPPETS.filter(s => s.shortcut), [])

  return (
    <div className="rounded-xl border border-[var(--border-color)] bg-[var(--bg-tertiary)] overflow-hidden">
      <div className="flex items-center gap-0.5 px-1.5 py-1 border-b border-[var(--border-color)] bg-[var(--bg-secondary)]/60">
        <button
          type="button"
          onClick={() => setIsPreview(false)}
          className={`flex items-center gap-1 px-2 py-0.5 rounded text-[10.5px] font-bold cursor-pointer ${
            isPreview ? 'text-[var(--text-muted)] hover:text-[var(--text-primary)]' : 'text-[var(--accent-color)] bg-[var(--accent-light)]'
          }`}
        >
          <Pencil size={11} />
          Écrire
        </button>
        <button
          type="button"
          onClick={() => setIsPreview(true)}
          className={`flex items-center gap-1 px-2 py-0.5 rounded text-[10.5px] font-bold cursor-pointer ${
            isPreview ? 'text-[var(--accent-color)] bg-[var(--accent-light)]' : 'text-[var(--text-muted)] hover:text-[var(--text-primary)]'
          }`}
        >
          <Eye size={11} />
          Aperçu
        </button>

        {!isPreview && (
          <div className="flex items-center gap-0.5 ml-2 pl-2 border-l border-[var(--border-color)]">
            {SNIPPETS.map(snippet => (
              <button
                key={snippet.title}
                type="button"
                disabled={disabled}
                onClick={() => applySnippet(snippet)}
                title={snippet.shortcut ? `${snippet.title} (Ctrl/Cmd+${snippet.shortcut.toUpperCase()})` : snippet.title}
                className="p-1 rounded text-[var(--text-muted)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-tertiary)] cursor-pointer disabled:opacity-40"
              >
                {snippet.icon}
              </button>
            ))}
          </div>
        )}

        {actions && <div className="ml-auto flex items-center gap-1.5">{actions}</div>}
      </div>

      {isPreview ? (
        <div className="px-3 py-2 overflow-auto" style={{ minHeight, maxHeight }}>
          {value.trim() ? (
            <MarkdownView>{value}</MarkdownView>
          ) : (
            <span className="text-[11px] text-[var(--text-muted)] italic">Rien à afficher pour l'instant.</span>
          )}
        </div>
      ) : (
        <textarea
          ref={textareaRef}
          value={value}
          disabled={disabled}
          placeholder={placeholder}
          onChange={e => onChange(e.target.value)}
          onKeyDown={e => {
            // Les raccourcis d'usage, ceux qu'on tape sans y penser.
            if ((e.metaKey || e.ctrlKey) && !e.altKey) {
              const snippet = shortcuts.find(s => s.shortcut === e.key.toLowerCase())
              if (snippet) {
                e.preventDefault()
                applySnippet(snippet)
                return
              }
            }
            onKeyDown?.(e)
          }}
          style={{ minHeight, maxHeight }}
          className="w-full px-3 py-2 text-[13px] bg-transparent text-[var(--text-primary)] focus:outline-none leading-relaxed resize-y font-[inherit]"
        />
      )}
    </div>
  )
}
