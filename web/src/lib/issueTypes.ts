/**
 * Étiquette de type de ticket.
 *
 * Un board qui porte sept types (story, tâche, bug, corrective action, dette
 * technique, amélioration, expertise) ne se lit pas si toutes les cartes se
 * ressemblent. La couleur porte l'information, le libellé la confirme.
 *
 * Les types connus ont leur couleur, choisie par convention plutôt que par goût :
 * le rouge pour ce qui est cassé, l'ambre pour ce qui corrige, le bleu pour la
 * demande, le gris pour l'ordinaire. Un type que l'instance a inventé reçoit une
 * couleur stable dérivée de son nom : elle ne veut rien dire, mais elle ne change
 * pas d'un rendu à l'autre, ce qui suffit à distinguer deux types côte à côte.
 */

export interface IssueTypeStyle {
  /** Libellé court affiché sur la carte, le nom complet restant en infobulle. */
  short: string
  color: string
  background: string
  border: string
}

/** Couleurs des types courants, par convention de sens. */
const KNOWN: Record<string, { rgb: string; short?: string }> = {
  story: { rgb: '59 130 246' },
  task: { rgb: '148 163 184' },
  'technical task': { rgb: '148 163 184', short: 'Tech task' },
  bug: { rgb: '244 63 94' },
  'bug-exp': { rgb: '244 63 94' },
  defect: { rgb: '244 63 94' },
  'corrective action': { rgb: '245 158 11', short: 'Corrective' },
  'technical debt': { rgb: '168 85 247', short: 'Tech debt' },
  improvement: { rgb: '16 185 129', short: 'Improv.' },
  expertise: { rgb: '6 182 212' },
  initiative: { rgb: '217 70 239' },
  epic: { rgb: '124 58 237' },
  'platform feedback': { rgb: '14 165 233', short: 'Feedback' },
  'internal documentation': { rgb: '100 116 139', short: 'Doc' },
  vulnerability: { rgb: '239 68 68', short: 'Vuln.' },
  'user story': { rgb: '59 130 246', short: 'Story' },
}

/** Palette de repli, parcourue par un hachage du nom du type. */
const FALLBACK_RGB = [
  '99 102 241',
  '236 72 153',
  '20 184 166',
  '234 88 12',
  '132 204 22',
  '129 140 248',
]

const hashOf = (value: string): number => {
  let hash = 0
  for (let i = 0; i < value.length; i += 1) {
    hash = (hash * 31 + value.charCodeAt(i)) % 100000
  }
  return hash
}

export const issueTypeStyle = (rawType: string): IssueTypeStyle => {
  const type = (rawType || '').trim()
  const key = type.toLowerCase()
  const known = KNOWN[key]
  const rgb = known ? known.rgb : FALLBACK_RGB[hashOf(key) % FALLBACK_RGB.length]

  return {
    // Un nom long mangerait la ligne du titre : il est raccourci par convention
    // quand on en connaît une, sinon tronqué proprement.
    short: known?.short || (type.length > 14 ? `${type.slice(0, 13)}…` : type),
    color: `rgb(${rgb})`,
    background: `rgb(${rgb} / 0.14)`,
    border: `rgb(${rgb} / 0.32)`,
  }
}
