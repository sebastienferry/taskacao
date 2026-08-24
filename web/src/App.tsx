import React from 'react'
import { AppProvider, useApp } from './context/AppContext'
import { Sidebar } from './components/Sidebar'
import { Header } from './components/Header'
import { BoardView } from './components/BoardView'
import { ListView } from './components/ListView'
import { ActivitiesView } from './components/ActivitiesView'
import { SyncView } from './components/SyncView'
import { DigestView } from './components/DigestView'
import { QuickAddModal } from './components/QuickAddModal'
import { TaskDetailModal } from './components/TaskDetailModal'
import { TaskChatDrawer } from './components/TaskChatDrawer'
import { CommandPalette } from './components/CommandPalette'
import { ProfileModal } from './components/ProfileModal'
import { ProjectModal } from './components/ProjectModal'
import { GitDiffModal } from './components/GitDiffModal'
import { BranchSwitcherModal } from './components/BranchSwitcherModal'
import { StatusBar } from './components/StatusBar'
import { WorkspaceTerminalPanel } from './components/WorkspaceTerminalPanel'
import { ToastContainer } from './components/ToastContainer'
import { Loader2 } from 'lucide-react'

const MainContent: React.FC = () => {
  const { activeView, isLoading, error, tasks } = useApp()

  return (
    <div className="flex flex-col h-screen w-screen overflow-hidden bg-[var(--bg-primary)]">
      <div className="flex flex-1 overflow-hidden">
        {/* Navigation Sidebar */}
        <Sidebar />

        {/* Main Workspace Area */}
        <div className="flex-1 flex flex-col h-full overflow-hidden">
          <Header />

          {/* Dynamic View Body */}
          <main className="flex-1 flex flex-col overflow-hidden relative">
            {isLoading && tasks.length === 0 ? (
              <div className="flex-1 flex flex-col items-center justify-center gap-3 text-[var(--text-muted)]">
                <Loader2 size={28} className="animate-spin text-[var(--accent-color)]" />
                <span className="text-xs font-medium">Chargement des tâches...</span>
              </div>
            ) : error ? (
              <div className="flex-1 flex flex-col items-center justify-center p-6 text-center text-rose-400">
                <p className="text-sm font-semibold mb-2">Erreur de connexion</p>
                <p className="text-xs text-[var(--text-muted)] max-w-md">{error}</p>
              </div>
            ) : activeView === 'board' ? (
              <BoardView />
            ) : activeView === 'list' ? (
              <ListView />
            ) : activeView === 'sync' ? (
              <SyncView />
            ) : activeView === 'digest' ? (
              <DigestView />
            ) : (
              <ActivitiesView />
            )}
          </main>
        </div>

        {/* Docked workspace CLI, side by side with the views */}
        <WorkspaceTerminalPanel />
      </div>

      {/* Global Bottom Status Bar with CWD Git Branch, Project, Engine & Live Jobs */}
      <StatusBar />

      {/* Global Modals & Overlays */}
      <QuickAddModal />
      <TaskDetailModal />
      <TaskChatDrawer />
      <ProjectModal />
      <GitDiffModal />
      <BranchSwitcherModal />
      <CommandPalette />
      <ProfileModal />
      <ToastContainer />
    </div>
  )
}

export function App() {
  return (
    <AppProvider>
      <MainContent />
    </AppProvider>
  )
}

export default App
