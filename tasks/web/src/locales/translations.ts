export interface TranslationSchema {
  app: {
    title: string
    tagline: string
  }
  nav: {
    allTasks: string
    backlog: string
    toClarify: string
    specified: string
    inProgress: string
    toValidate: string
    done: string
    views: string
    board: string
    list: string
    filters: string
    myTasks: string
    urgentHigh: string
    labels: string
    settings: string
    reseedDemo: string
    toggleSidebar: string
    syncLinear: string
    syncGithub: string
  }
  header: {
    searchPlaceholder: string
    quickAdd: string
    commandPalette: string
    boardView: string
    listView: string
    filterStatus: string
    filterPriority: string
    clearFilters: string
    activeFilter: string
    activeFilters: string
    syncing: string
    syncNow: string
    hideDone: string
    showDone: string
  }
  board: {
    emptyColumn: string
    addTask: string
    dragHint: string
  }
  list: {
    columns: {
      key: string
      title: string
      status: string
      priority: string
      labels: string
      assignee: string
      dueDate: string
      source: string
      actions: string
    }
    empty: string
  }
  status: {
    to_clarify: string
    to_specify: string
    to_implement: string
    to_test: string
    to_close: string
    backlog: string
    specified: string
    in_progress: string
    to_validate: string
    done: string
  }
  priority: {
    urgent: string
    high: string
    medium: string
    low: string
  }
  skills: {
    title: string
    subtitle: string
    pipeline: string
    runSkill: string
    running: string
    nextStep: string
    autoPilot: string
    clarify: string
    specify: string
    implement: string
    createPr: string
    history: string
    noHistory: string
    branch: string
    pullRequest: string
    viewPr: string
    output: string
    stepsTitle: string
    skillSuccess: string
  }
  taskModal: {
    createTitle: string
    editTitle: string
    titlePlaceholder: string
    descPlaceholder: string
    status: string
    priority: string
    labels: string
    addLabel: string
    assignee: string
    dueDate: string
    cancel: string
    save: string
    create: string
    delete: string
    deleteConfirm: string
    created: string
    updated: string
    switchToModal: string
    switchToPanel: string
  }
  convert: {
    bannerTitle: string
    bannerDesc: string
    btnLinear: string
    btnGithub: string
    converting: string
    success: string
  }
  quickAdd: {
    title: string
    placeholder: string
    hint: string
    status: string
    priority: string
    tracker: string
  }
  commandPalette: {
    searchPlaceholder: string
    general: string
    createTask: string
    switchBoard: string
    switchList: string
    toggleTheme: string
    changeLanguage: string
    openProfile: string
    reseed: string
    syncLinear: string
    syncGithub: string
    tasksSection: string
    skillsSection: string
    noResults: string
    hintNavigate: string
    hintSelect: string
    hintClose: string
  }
  profileModal: {
    title: string
    subtitle: string
    tabs: {
      appearance: string
      aiConfig: string
      tracker: string
    }
    userSection: string
    name: string
    email: string
    appearanceSection: string
    accentColor: string
    theme: string
    themes: {
      dark: string
      light: string
      system: string
    }
    language: string
    languages: {
      fr: string
      en: string
    }
    density: string
    densities: {
      compact: string
      standard: string
      comfortable: string
    }
    defaultView: string
    detailMode: string
    detailModes: {
      modal: string
      panel: string
    }
    accents: {
      indigo: string
      violet: string
      emerald: string
      amber: string
      rose: string
      cyan: string
      blue: string
      orange: string
    }
    ai: {
      title: string
      engine: string
      engineDesc: string
      cmdTemplate: string
      repoPath: string
      repoPathDesc: string
      promptsTitle: string
      promptClarify: string
      promptSpecify: string
      promptImplement: string
      promptCreatePr: string
      cliStatusTitle: string
    }
    tracker: {
      title: string
      selectTracker: string
      linearTeam: string
      githubRepo: string
      syncLinearBtn: string
      syncGithubBtn: string
    }
    save: string
    reseedBtn: string
  }
  toasts: {
    taskCreated: string
    taskUpdated: string
    taskMoved: string
    taskDeleted: string
    settingsSaved: string
    demoReseeded: string
    skillStarted: string
    skillCompleted: string
    syncSuccess: string
    error: string
  }
}

export const translations: Record<'fr' | 'en', TranslationSchema> = {
  fr: {
    app: {
      title: 'Fretzee Tasks',
      tagline: 'Gestionnaire de tâches agentique',
    },
    nav: {
      allTasks: 'Toutes les tâches',
      backlog: 'Backlog',
      toClarify: 'À clarifier',
      specified: 'Spécifié',
      inProgress: 'En cours',
      toValidate: 'À valider',
      done: 'Terminé',
      views: 'Vues',
      board: 'Tableau Kanban',
      list: 'Vue Liste',
      filters: 'Filtres rapides',
      myTasks: 'Mes tâches',
      urgentHigh: 'Priorité Haute',
      labels: 'Étiquettes',
      settings: 'Profil & Préférences',
      reseedDemo: 'Réinitialiser démo',
      toggleSidebar: 'Replier / Déplier le menu',
      syncLinear: 'Synchroniser Linear',
      syncGithub: 'Synchroniser GitHub',
    },
    header: {
      searchPlaceholder: 'Rechercher des tâches... (Appuyez sur "/" pour cibler)',
      quickAdd: 'Ajouter une tâche',
      commandPalette: 'Palette de commandes',
      boardView: 'Tableau',
      listView: 'Liste',
      filterStatus: 'Statut',
      filterPriority: 'Priorité',
      clearFilters: 'Effacer les filtres',
      activeFilter: 'filtre actif',
      activeFilters: 'filtres actifs',
      syncing: 'Synchronisation...',
      syncNow: 'Synchroniser',
      hideDone: 'Masquer les tâches terminées',
      showDone: 'Afficher les tâches terminées',
    },
    board: {
      emptyColumn: 'Aucune tâche dans cette colonne',
      addTask: 'Ajouter une tâche',
      dragHint: 'Glisser-déposer pour changer de statut ou réordonner',
    },
    list: {
      columns: {
        key: 'Clé',
        title: 'Titre',
        status: 'Statut',
        priority: 'Priorité',
        labels: 'Étiquettes',
        assignee: 'Assigné à',
        dueDate: 'Échéance',
        source: 'Source',
        actions: 'Actions',
      },
      empty: 'Aucune tâche ne correspond à votre recherche ou filtre.',
    },
    status: {
      to_clarify: 'À clarifier',
      to_specify: 'À spécifier',
      to_implement: 'À implémenter',
      to_test: 'À tester',
      to_close: 'À fermer',
      backlog: 'À clarifier',
      specified: 'À spécifier',
      in_progress: 'À implémenter',
      to_validate: 'À tester',
      done: 'À fermer',
    },
    priority: {
      urgent: 'Urgent',
      high: 'Haute',
      medium: 'Moyenne',
      low: 'Basse',
    },
    skills: {
      title: 'Agent Copilot & Skills',
      subtitle: 'Exécutez les skills du workflow pour faire avancer la tâche',
      pipeline: 'Pipeline de développement',
      runSkill: 'Lancer la skill',
      running: 'Exécution en cours via CLI...',
      nextStep: 'Étape suivante',
      autoPilot: 'Auto-Pilot (/pick-issue)',
      clarify: 'Clarifier (/clarify-issue)',
      specify: 'Spécifier Speckit (/specify-issue)',
      implement: 'Coder (/code-issue)',
      createPr: 'Créer PR (/create-pr)',
      history: 'Historique des exécutions & Artefacts',
      noHistory: 'Aucune exécution de skill pour le moment sur cette tâche.',
      branch: 'Branche Git',
      pullRequest: 'Pull Request',
      viewPr: 'Voir la PR sur GitHub',
      output: 'Rapport & Artefact généré',
      stepsTitle: 'Étapes exécutées',
      skillSuccess: 'Skill exécutée avec succès !',
    },
    taskModal: {
      createTitle: 'Nouvelle Tâche',
      editTitle: 'Détails & Agent Copilot',
      titlePlaceholder: 'Titre de la tâche...',
      descPlaceholder: 'Description, contexte technique, critères d\'acceptation...',
      status: 'Statut',
      priority: 'Priorité',
      labels: 'Étiquettes (séparées par des virgules ou Entrée)',
      addLabel: 'Ajouter une étiquette...',
      assignee: 'Assigné à',
      dueDate: 'Date d\'échéance',
      cancel: 'Annuler',
      save: 'Enregistrer',
      create: 'Créer la tâche',
      delete: 'Supprimer la tâche',
      deleteConfirm: 'Êtes-vous sûr de vouloir supprimer cette tâche ? Cette action est irréversible.',
      created: 'Créée le',
      updated: 'Modifiée le',
      switchToModal: 'Passer en modale centrée',
      switchToPanel: 'Passer en panneau latéral droit',
    },
    convert: {
      bannerTitle: 'Tâche locale (non synchronisée)',
      bannerDesc: 'Cette tâche est enregistrée uniquement en local dans SQLite.',
      btnLinear: 'Exporter vers Linear',
      btnGithub: 'Exporter vers GitHub',
      converting: 'Exportation vers le tracker distant...',
      success: 'Issue créée sur le tracker distant !',
    },
    quickAdd: {
      title: 'Ajout rapide',
      placeholder: 'Titre de la tâche (ex: FRE-110 Créer endpoint GraphQL...)',
      hint: 'Appuyez sur Entrée pour créer immédiatement',
      status: 'Statut initial',
      priority: 'Priorité',
      tracker: 'Destination / Tracker',
    },
    commandPalette: {
      searchPlaceholder: 'Tapez une commande, skill ou tâche...',
      general: 'Actions générales',
      createTask: 'Créer une nouvelle tâche',
      switchBoard: 'Passer en vue Kanban (Tableau)',
      switchList: 'Passer en vue Liste',
      toggleTheme: 'Basculer le thème (Sombre / Clair)',
      changeLanguage: 'Changer la langue (FR / EN)',
      openProfile: 'Ouvrir les Préférences & Profil',
      reseed: 'Réinitialiser la base de données de démo',
      syncLinear: 'Synchroniser avec Linear CLI',
      syncGithub: 'Synchroniser avec GitHub CLI',
      tasksSection: 'Accès direct aux tâches',
      skillsSection: 'Skills Agentiques (/skills)',
      noResults: 'Aucun résultat trouvé pour',
      hintNavigate: 'pour naviguer',
      hintSelect: 'pour choisir',
      hintClose: 'pour fermer',
    },
    profileModal: {
      title: 'Configuration & Paramètres',
      subtitle: 'Personnalisez le moteur IA (agy, vibe, claude), Linear/GitHub CLI et l\'apparence',
      tabs: {
        appearance: 'Apparence & Profil',
        aiConfig: 'Moteur IA & Prompts',
        tracker: 'Linear & GitHub CLI',
      },
      userSection: 'Informations utilisateur',
      name: 'Nom d\'utilisateur',
      email: 'Adresse e-mail',
      appearanceSection: 'Apparence & Ergonomie',
      accentColor: 'Couleur d\'accent',
      theme: 'Thème général',
      themes: {
        dark: 'Sombre',
        light: 'Clair',
        system: 'Système',
      },
      language: 'Langue de l\'interface',
      languages: {
        fr: 'Français (FR)',
        en: 'English (EN)',
      },
      density: 'Taille d\'affichage / Densité',
      densities: {
        compact: 'Compacte (Plus d\'éléments à l\'écran)',
        standard: 'Standard (Équilibrée)',
        comfortable: 'Confortable (Espaces généreux)',
      },
      defaultView: 'Vue par défaut',
      detailMode: 'Affichage des détails de tâche',
      detailModes: {
        modal: 'Modale centrée',
        panel: 'Panneau latéral droit (Right Panel)',
      },
      accents: {
        indigo: 'Indigo Royal',
        violet: 'Violet Electrique',
        emerald: 'Émeraude Vif',
        amber: 'Ambre Chaud',
        rose: 'Rose Bonbon',
        cyan: 'Cyan Néon',
        blue: 'Bleu Océan',
        orange: 'Orange Sunset',
      },
      ai: {
        title: 'Moteur IA & Exécution Shell',
        engine: 'Agent IA sélectionné',
        engineDesc: 'Choisissez l\'outil CLI qui exécutera les skills en arrière-plan',
        cmdTemplate: 'Template de commande Shell CLI',
        repoPath: 'Répertoire du projet cible (CWD)',
        repoPathDesc: 'Emplacement du repo dans lequel l\'agent exécutera les commandes',
        promptsTitle: 'Personnalisation des Prompts par Skill',
        promptClarify: 'Prompt /clarify-issue (Cadrage & questions)',
        promptSpecify: 'Prompt /specify-issue (Spécification Speckit)',
        promptImplement: 'Prompt /code-issue (Implémentation & tests)',
        promptCreatePr: 'Prompt /create-pr (Génération Pull Request)',
        cliStatusTitle: 'Statut des CLI Locales',
      },
      tracker: {
        title: 'Intégration Issue Tracker',
        selectTracker: 'Tracker Actif',
        linearTeam: 'Clé d\'équipe Linear (Team Key)',
        githubRepo: 'Repository GitHub (owner/repo)',
        syncLinearBtn: 'Importer & Synchroniser Linear',
        syncGithubBtn: 'Importer & Synchroniser GitHub',
      },
      save: 'Enregistrer la configuration',
      reseedBtn: 'Réinitialiser le jeu de test démo',
    },
    toasts: {
      taskCreated: 'Tâche créée avec succès !',
      taskUpdated: 'Tâche mise à jour !',
      taskMoved: 'Statut de la tâche mis à jour',
      taskDeleted: 'Tâche supprimée',
      settingsSaved: 'Paramètres enregistrés avec succès',
      demoReseeded: 'Base de données réinitialisée avec succès !',
      skillStarted: 'Exécution du process IA via shell en cours...',
      skillCompleted: 'Skill exécutée avec succès !',
      syncSuccess: 'Synchronisation des tickets réussie !',
      error: 'Une erreur est survenue',
    },
  },
  en: {
    app: {
      title: 'Fretzee Tasks',
      tagline: 'Agentic Task Workflow Manager',
    },
    nav: {
      allTasks: 'All Tasks',
      backlog: 'Backlog',
      toClarify: 'To Clarify',
      specified: 'Specified',
      inProgress: 'In Progress',
      toValidate: 'To Validate',
      done: 'Done',
      views: 'Views',
      board: 'Kanban Board',
      list: 'List View',
      filters: 'Quick Filters',
      myTasks: 'My Tasks',
      urgentHigh: 'High Priority',
      labels: 'Labels',
      settings: 'Profile & Preferences',
      reseedDemo: 'Reset Demo Data',
      toggleSidebar: 'Toggle Sidebar',
      syncLinear: 'Sync Linear',
      syncGithub: 'Sync GitHub',
    },
    header: {
      searchPlaceholder: 'Search tasks... (Press "/" to focus)',
      quickAdd: 'Add Task',
      commandPalette: 'Command Palette',
      boardView: 'Board',
      listView: 'List',
      filterStatus: 'Status',
      filterPriority: 'Priority',
      clearFilters: 'Clear filters',
      activeFilter: 'active filter',
      activeFilters: 'active filters',
      syncing: 'Syncing...',
      syncNow: 'Sync Issues',
      hideDone: 'Hide completed tasks',
      showDone: 'Show completed tasks',
    },
    board: {
      emptyColumn: 'No tasks in this column',
      addTask: 'Add task',
      dragHint: 'Drag and drop to update status or reorder',
    },
    list: {
      columns: {
        key: 'Key',
        title: 'Title',
        status: 'Status',
        priority: 'Priority',
        labels: 'Labels',
        assignee: 'Assignee',
        dueDate: 'Due Date',
        source: 'Source',
        actions: 'Actions',
      },
      empty: 'No tasks match your search or filters.',
    },
    status: {
      to_clarify: 'To Clarify',
      to_specify: 'To Specify',
      to_implement: 'To Implement',
      to_test: 'To Test',
      to_close: 'To Close',
      backlog: 'To Clarify',
      specified: 'To Specify',
      in_progress: 'To Implement',
      to_validate: 'To Test',
      done: 'To Close',
    },
    priority: {
      urgent: 'Urgent',
      high: 'High',
      medium: 'Medium',
      low: 'Low',
    },
    skills: {
      title: 'Agent Copilot & Skills',
      subtitle: 'Run workflow skills to advance this task automatically',
      pipeline: 'Development Pipeline',
      runSkill: 'Run Skill',
      running: 'Executing via CLI shell...',
      nextStep: 'Next Step',
      autoPilot: 'Auto-Pilot (/pick-issue)',
      clarify: 'Clarify (/clarify-issue)',
      specify: 'Specify Speckit (/specify-issue)',
      implement: 'Implement (/code-issue)',
      createPr: 'Create PR (/create-pr)',
      history: 'Execution History & Artifacts',
      noHistory: 'No skill runs recorded yet on this task.',
      branch: 'Git Branch',
      pullRequest: 'Pull Request',
      viewPr: 'View PR on GitHub',
      output: 'Report & Generated Artifact',
      stepsTitle: 'Executed Steps',
      skillSuccess: 'Skill executed successfully!',
    },
    taskModal: {
      createTitle: 'New Task',
      editTitle: 'Details & Agent Copilot',
      titlePlaceholder: 'Task title...',
      descPlaceholder: 'Description, technical context, acceptance criteria...',
      status: 'Status',
      priority: 'Priority',
      labels: 'Labels (separated by commas or Enter)',
      addLabel: 'Add label...',
      assignee: 'Assignee',
      dueDate: 'Due date',
      cancel: 'Cancel',
      save: 'Save Changes',
      create: 'Create Task',
      delete: 'Delete Task',
      deleteConfirm: 'Are you sure you want to delete this task? This action cannot be undone.',
      created: 'Created at',
      updated: 'Updated at',
      switchToModal: 'Switch to centered modal',
      switchToPanel: 'Switch to right sliding panel',
    },
    convert: {
      bannerTitle: 'Local task (unsynced)',
      bannerDesc: 'This task is stored only in local SQLite.',
      btnLinear: 'Export to Linear',
      btnGithub: 'Export to GitHub',
      converting: 'Exporting to remote tracker...',
      success: 'Issue created on remote tracker!',
    },
    quickAdd: {
      title: 'Quick Add Task',
      placeholder: 'Task title (e.g. FRE-110 Create GraphQL endpoint...)',
      hint: 'Press Enter to create immediately',
      status: 'Initial status',
      priority: 'Priority',
      tracker: 'Destination / Tracker',
    },
    commandPalette: {
      searchPlaceholder: 'Type a command, skill or task...',
      general: 'General Actions',
      createTask: 'Create new task',
      switchBoard: 'Switch to Kanban Board',
      switchList: 'Switch to List View',
      toggleTheme: 'Toggle Theme (Dark / Light)',
      changeLanguage: 'Switch Language (FR / EN)',
      openProfile: 'Open Profile & Preferences',
      reseed: 'Reset Demo Database',
      syncLinear: 'Sync with Linear CLI',
      syncGithub: 'Sync with GitHub CLI',
      tasksSection: 'Jump to Task',
      skillsSection: 'Agentic Skills (/skills)',
      noResults: 'No results found for',
      hintNavigate: 'to navigate',
      hintSelect: 'to select',
      hintClose: 'to close',
    },
    profileModal: {
      title: 'Configuration & Settings',
      subtitle: 'Configure AI engine (agy, vibe, claude), Linear/GitHub CLI and appearance',
      tabs: {
        appearance: 'Appearance & Profile',
        aiConfig: 'AI Engine & Prompts',
        tracker: 'Linear & GitHub CLI',
      },
      userSection: 'User Information',
      name: 'User Name',
      email: 'Email Address',
      appearanceSection: 'Appearance & Ergonomics',
      accentColor: 'Accent Color',
      theme: 'Theme Mode',
      themes: {
        dark: 'Dark',
        light: 'Light',
        system: 'System',
      },
      language: 'Interface Language',
      languages: {
        fr: 'Français (FR)',
        en: 'English (EN)',
      },
      density: 'Display Density / Scaling',
      densities: {
        compact: 'Compact (More content on screen)',
        standard: 'Standard (Balanced)',
        comfortable: 'Comfortable (Spacious)',
      },
      defaultView: 'Default View',
      detailMode: 'Story Detail View Style',
      detailModes: {
        modal: 'Centered Modal',
        panel: 'Right Sliding Panel (Drawer)',
      },
      accents: {
        indigo: 'Royal Indigo',
        violet: 'Electric Violet',
        emerald: 'Vivid Emerald',
        amber: 'Warm Amber',
        rose: 'Candy Rose',
        cyan: 'Neon Cyan',
        blue: 'Ocean Blue',
        orange: 'Sunset Orange',
      },
      ai: {
        title: 'AI Engine & Shell Execution',
        engine: 'Selected AI Agent',
        engineDesc: 'Choose the CLI tool that will execute skills in the background',
        cmdTemplate: 'Shell CLI Command Template',
        repoPath: 'Target Project Directory (CWD)',
        repoPathDesc: 'Workspace directory where the agent will run commands',
        promptsTitle: 'Custom Skill Prompts',
        promptClarify: 'Prompt /clarify-issue',
        promptSpecify: 'Prompt /specify-issue',
        promptImplement: 'Prompt /code-issue',
        promptCreatePr: 'Prompt /create-pr',
        cliStatusTitle: 'Local CLI Tools Status',
      },
      tracker: {
        title: 'Issue Tracker Integration',
        selectTracker: 'Active Tracker',
        linearTeam: 'Linear Team Key',
        githubRepo: 'GitHub Repository (owner/repo)',
        syncLinearBtn: 'Import & Sync Linear',
        syncGithubBtn: 'Import & Sync GitHub',
      },
      save: 'Save Configuration',
      reseedBtn: 'Reset Demo Dataset',
    },
    toasts: {
      taskCreated: 'Task created successfully!',
      taskUpdated: 'Task updated successfully!',
      taskMoved: 'Task status updated',
      taskDeleted: 'Task deleted',
      settingsSaved: 'Settings saved successfully',
      demoReseeded: 'Demo database reset successfully!',
      skillStarted: 'Executing AI process via shell...',
      skillCompleted: 'Skill executed successfully!',
      syncSuccess: 'Issues synchronized successfully!',
      error: 'An error occurred',
    },
  },
}
