---
name: specify-issue
description: Rédige la spécification technique selon la norme Open Feature (Feature Flags, Evaluation Contexts, Hooks, Variations & Lifecycle).
---
# Skill : Specify Issue (Open Feature SDD)

## Objectif
Générer une spécification technique standardisée selon le framework Open Feature Spec-Driven Design.

## Instructions
1. Vérifier les réponses de clarification et le contexte du projet.
2. Créer ou basculer sur la branche Git de travail au format <KEY>-<titre-slug>.
3. Rédiger la spécification technique OpenFeature complète incluant :
   - Définition des Feature Flags (Flag Key, Type: boolean/string/number/object, Default Value, Variations)
   - Evaluation Context & Règles de ciblage (Attributs utilisateur, tenant, environnement)
   - Intégration OpenFeature SDK (Provider, Evaluation Hooks, Fallbacks de sécurité)
   - Cycle de vie du Flag (Création -> Rollout progressif -> Dépréciation & Nettoyage de code)
   - Plan de tests et scénarios de validation (Given / When / Then)
