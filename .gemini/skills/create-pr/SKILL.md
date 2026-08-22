---
name: create-pr
description: Effectue la revue finale, génère des commits conventionnels et publie la Pull Request sur GitHub.
---
# Skill : Review & Create PR

## Objectif
Revoir les changements, valider la qualité du code, commiter avec des messages conventionnels et créer la Pull Request.

## Instructions
1. Effectuer un git diff pour inspecter tous les changements réalisés.
2. Vérifier l'absence d'erreurs, de secrets ou de code mort.
3. Créer un commit conventionnel (ex: feat(scope): ... ou fix(scope): ...).
4. Pousser la branche sur le dépôt distant (git push -u origin <branch>).
5. Créer la Pull Request via gh pr create avec un résumé détaillé et le lien vers le ticket.
