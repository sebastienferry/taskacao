---
name: pick-issue
description: Sélectionne la tâche prioritaire, avance son statut à travers le cycle de développement complet et crée la PR.
---
# Skill : Auto-Pilot Orchestrator (Pick Issue)

## Objectif
Prendre en charge une tâche depuis la file d'attente, exécuter de manière autonome le cycle de clarification, spécification, codage et ouverture de PR.

## Instructions
1. Identifier la tâche cible spécifiée ou prendre la plus prioritaire.
2. Basculer ou créer la branche Git dédiée dans le worktree.
3. Exécuter séquentiellement :
   - /clarify-issue (si non cadrée)
   - /specify-issue (si non spécifiée)
   - /code-issue (implémentation et tests)
   - /create-pr (validation finale et soumission)
4. Mettre à jour le statut et consigner les artefacts d'exécution.
