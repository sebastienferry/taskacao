---
name: create-pr
description: Effectue la revue finale, génère des commits conventionnels et publie la Pull Request sur GitHub (ou fusionne localement si aucun remote n'est configuré).
---
# Skill : Review, Pull Request & Local Merge

## Objectif
Revoir les changements, valider la qualité du code, commiter avec des messages conventionnels et publier la Pull Request (ou fusionner localement si aucun remote n'est configuré).

## Instructions
1. Effectuer un 'git status' et 'git diff' pour inspecter tous les changements sur la branche.
2. S'assurer que les modifications sont commitées (ex: feat(scope): ... ou fix(scope): ...).
3. Vérifier les remotes via 'git remote' :
   - **Si remote présent** : Pousser la branche ('git push -u origin <branch>') et créer la Pull Request ('gh pr create' ou 'glab mr create').
   - **Si aucun remote configuré** : Basculer sur la branche principale ('git checkout main') et fusionner la branche ('git merge --no-ff <branch>').
4. Produire un compte-rendu clair des actions réalisées.

