---
name: specify-issue
description: Rédige la spécification technique selon GitHub Spec Kit (spec.md, plan.md, tasks.md sous specs/), avec user stories, architecture et critères d'acceptation Gherkin.
---
# Skill : Specify Issue (GitHub Spec Kit SDD)

## Objectif
Produire la spécification exécutable d'un ticket selon GitHub Spec Kit, dans le
répertoire `specs/` du projet.

## Pré-requis
Le projet doit être initialisé avec Spec Kit (répertoire `.specify/` présent).
Sinon, lancer l'installation Spec Kit depuis la configuration du projet Taskacao,
ou exécuter `specify init --here` à la racine du dépôt.

## Instructions
1. Lire `.specify/memory/constitution.md` pour respecter les principes du projet.
2. Créer ou basculer sur la branche Git de travail au format <KEY>-<titre-slug>.
3. Écrire `specs/<KEY>-<titre-slug>/spec.md` (le quoi et le pourquoi, sans
   choix d'implémentation) :
   - Contexte, User Stories priorisées et périmètre exclu
   - Exigences fonctionnelles numérotées et critères d'acceptation Given / When / Then
   - Points à clarifier marqués explicitement plutôt que devinés
4. Écrire `plan.md` (le comment) : pile technique, architecture, contrats de données
   et diagrammes de flux Mermaid.
5. Écrire `tasks.md` : la checklist ordonnée et vérifiable de mise en œuvre.
6. Si les commandes Spec Kit sont disponibles dans l'agent, utiliser
   `/speckit.specify`, `/speckit.plan` puis `/speckit.tasks`
   au lieu de rédiger les fichiers à la main.
7. Publier le résumé de la spécification en commentaire du ticket.
