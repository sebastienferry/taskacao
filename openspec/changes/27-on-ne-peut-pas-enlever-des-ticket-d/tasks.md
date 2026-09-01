# Implementation Checklist (Tasks)

## Phase 1: Menu contextuel des cartes (`TaskCard.tsx`)
- [ ] 1.1 Ajouter la vérification de la présence de `task.sprint` dans `TaskCard.tsx`.
- [ ] 1.2 Ajouter l'entrée "Retirer du sprint" (avec icône et handler `setTaskSprint(task.id, '', '')`) dans le menu déroulant `MoreHorizontal`.
- [ ] 1.3 Vérifier que le menu contextuel se ferme correctement et déclenche la notification toast.

## Phase 2: Actions individuelles dans la Vue Sprint (`SprintTimelineView.tsx`)
- [ ] 2.1 En mode d'affichage compact (vue cartes sans backlog ouvert), ajouter le bouton/icône "Retirer du sprint" sur chaque carte de ticket dans la colonne d'un sprint.
- [ ] 2.2 En mode d'affichage compact (vue chips sans backlog ouvert), ajouter un bouton de retrait rapide sur chaque puce de ticket.
- [ ] 2.3 Vérifier et uniformiser le comportement des boutons de retrait dans le mode d'affichage planification (backlog ouvert).

## Phase 3: Action groupée / Sélection multiple (`SprintTimelineView.tsx`)
- [ ] 3.1 Ajouter un bouton "Retirer du sprint" / "Renvoyer au backlog" dans la barre d'action groupée lorsque des tâches affectées à un sprint sont sélectionnées.
- [ ] 3.2 Lier cette action à l'appel `setTasksSprint(currentProject.id, selectedIds, '', '')`.
- [ ] 3.3 Réinitialiser les cases à cocher `checkedTaskIds` après l'exécution du retrait groupé.

## Phase 4: Persistence Backend & Synchronization
- [ ] 4.1 Valider le bon fonctionnement de `POST /api/tasks/{id}/sprint` lors de l'envoi de `sprintId: ""` et `sprintName: ""`.
- [ ] 4.2 Valider le bon fonctionnement de `POST /api/projects/{id}/sprint-move` lors du déplacement par lot vers le backlog (`sprintId: ""`).
- [ ] 4.3 Vérifier la création des activités `TrackerOpSetSprint` ciblées sur "backlog" dans la file de synchro.

## Phase 5: Verification & Tests
- [ ] 5.1 Exécuter la suite de tests backend (`go test ./...`).
- [ ] 5.2 Valider le build frontend (`npm run build` dans `web/`).
- [ ] 5.3 Valider la spécification OpenSpec avec `openspec validate 27-on-ne-peut-pas-enlever-des-ticket-d --strict`.
