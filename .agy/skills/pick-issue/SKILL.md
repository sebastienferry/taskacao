---
name: pick-issue
description: Sélectionne la tâche prioritaire, avance son statut à travers le cycle de développement complet et crée la PR.
---
# Skill : Auto-Pilot Orchestrator (Pick Issue)

## Objectif
Prendre en charge une tâche depuis la file d'attente, exécuter de manière autonome le cycle de clarification, spécification, codage et ouverture de PR en synchronisant chaque étape via le handler local TaskFlow.

## Instructions
1. Identifier la tâche cible spécifiée ou prendre la plus prioritaire.
2. Basculer ou créer la branche Git dédiée dans le worktree.
3. Exécuter séquentiellement et transitionner après chaque étape via le handler local TaskFlow :
   - `/clarify-issue` (si non cadrée) ➔ `curl -s -X POST "${TASKFLOW_API_URL:-http://127.0.0.1:8090}/api/tasks/${TASKFLOW_TASK_KEY:-$TASKFLOW_TASK_ID}/stage" -H "Content-Type: application/json" -d '{"stage":"clarified","note":"..."}'`
   - `/specify-issue` (si non spécifiée) ➔ `curl -s -X POST "${TASKFLOW_API_URL:-http://127.0.0.1:8090}/api/tasks/${TASKFLOW_TASK_KEY:-$TASKFLOW_TASK_ID}/stage" -H "Content-Type: application/json" -d '{"stage":"specified","branch":"...","note":"..."}'`
   - `/code-issue` (implémentation et tests) ➔ `curl -s -X POST "${TASKFLOW_API_URL:-http://127.0.0.1:8090}/api/tasks/${TASKFLOW_TASK_KEY:-$TASKFLOW_TASK_ID}/stage" -H "Content-Type: application/json" -d '{"stage":"implemented","note":"..."}'`
   - `/create-pr` (validation finale et soumission) ➔ `curl -s -X POST "${TASKFLOW_API_URL:-http://127.0.0.1:8090}/api/tasks/${TASKFLOW_TASK_KEY:-$TASKFLOW_TASK_ID}/stage" -H "Content-Type: application/json" -d '{"stage":"reviewed","prUrl":"...","note":"..."}'`
4. Mettre à jour le statut et consigner les artefacts d'exécution.
