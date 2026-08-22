import React, { useEffect } from 'react'
import { CheckCircle2, AlertTriangle, AlertCircle, Info, X } from 'lucide-react'
import { useApp } from '../context/AppContext'
import type { ToastMessage } from '../types'

const ToastItem: React.FC<{ toast: ToastMessage; onRemove: (id: string) => void }> = ({ toast, onRemove }) => {
  useEffect(() => {
    const timer = setTimeout(() => {
      onRemove(toast.id)
    }, toast.duration || 3500)
    return () => clearTimeout(timer)
  }, [toast.id, toast.duration, onRemove])

  const getIcon = (type: ToastMessage['type']) => {
    switch (type) {
      case 'success':
        return <CheckCircle2 size={16} className="text-emerald-400 shrink-0" />
      case 'warning':
        return <AlertTriangle size={16} className="text-amber-400 shrink-0" />
      case 'error':
        return <AlertCircle size={16} className="text-rose-400 shrink-0" />
      default:
        return <Info size={16} className="text-blue-400 shrink-0" />
    }
  }

  return (
    <div
      className="pointer-events-auto flex items-start gap-2.5 p-3 rounded-xl bg-[var(--bg-secondary)] border border-[var(--border-color)] shadow-xl text-xs animate-in slide-in-from-bottom-2 fade-in duration-200"
    >
      <div className="mt-0.5">{getIcon(toast.type)}</div>
      <div className="flex-1 min-w-0">
        <div className="font-semibold text-[var(--text-primary)]">
          {toast.title}
        </div>
        {toast.description && (
          <div className="text-[11px] text-[var(--text-muted)] mt-0.5 line-clamp-2">
            {toast.description}
          </div>
        )}
      </div>
      <button
        onClick={() => onRemove(toast.id)}
        className="p-0.5 text-[var(--text-muted)] hover:text-[var(--text-primary)] rounded transition-colors"
      >
        <X size={14} />
      </button>
    </div>
  )
}

export const ToastContainer: React.FC = () => {
  const { toasts, removeToast } = useApp()

  if (toasts.length === 0) return null

  return (
    <div className="fixed bottom-4 right-4 z-50 flex flex-col gap-2 max-w-sm pointer-events-none">
      {toasts.map(toast => (
        <ToastItem key={toast.id} toast={toast} onRemove={removeToast} />
      ))}
    </div>
  )
}
