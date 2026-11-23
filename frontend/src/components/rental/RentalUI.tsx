import type { ReactNode } from 'react'
import { Icon } from '../Icons'
import { formatCurrency } from '../../utils/currency'

export function Money({ amount, currency }: { amount: number; currency: string }) {
  return <span className="tabular-nums">{currency} {formatCurrency(amount)}</span>
}

export function Badge({ tone, children }: { tone: 'success' | 'warning' | 'danger' | 'neutral' | 'info'; children: ReactNode }) {
  const tones: Record<string, string> = {
    success: 'bg-success/10 text-success',
    warning: 'bg-warning/10 text-warning',
    danger: 'bg-danger/10 text-danger',
    neutral: 'bg-bg text-text-secondary',
    info: 'bg-primary/10 text-primary',
  }
  return <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${tones[tone]}`}>{children}</span>
}

export function Field({ label, hint, error, children }: { label: string; hint?: string; error?: string; children: ReactNode }) {
  return (
    <div>
      <label className="block text-sm font-medium mb-1">{label}</label>
      {children}
      {hint && !error && <p className="text-xs text-text-secondary mt-1">{hint}</p>}
      {error && <p className="text-sm text-danger mt-1">{error}</p>}
    </div>
  )
}

export function TextInput(props: React.InputHTMLAttributes<HTMLInputElement>) {
  const { className = '', ...rest } = props
  return (
    <input
      {...rest}
      className={`w-full px-3 py-2 border border-border rounded-ios-sm bg-card text-text placeholder:text-text-secondary focus:outline-none focus:border-primary min-h-[44px] disabled:opacity-60 disabled:cursor-not-allowed ${className}`}
    />
  )
}

export function TextArea(props: React.TextareaHTMLAttributes<HTMLTextAreaElement>) {
  const { className = '', ...rest } = props
  return (
    <textarea
      {...rest}
      className={`w-full px-3 py-2 border border-border rounded-ios-sm bg-card text-text placeholder:text-text-secondary focus:outline-none focus:border-primary ${className}`}
    />
  )
}

export function Modal({ open, title, onClose, children, footer, zIndex = 'z-[60]' }: {
  open: boolean
  title: string
  onClose: () => void
  children: ReactNode
  footer?: ReactNode
  zIndex?: string
}) {
  if (!open) return null
  return (
    <div className={`fixed inset-0 bg-black/40 flex items-center justify-center ${zIndex} p-4`}>
      <div className="bg-card rounded-ios shadow-ios-lg w-full max-w-lg max-h-[90vh] overflow-y-auto">
        <div className="flex items-center justify-between px-5 py-4 border-b border-border">
          <h3 className="font-semibold text-base">{title}</h3>
          <button
            type="button"
            onClick={onClose}
            aria-label="Cerrar"
            className="p-2 rounded-full hover:bg-bg transition-colors text-text-secondary"
          >
            <Icon name="cancel" className="w-5 h-5" />
          </button>
        </div>
        <div className="p-5 space-y-4">{children}</div>
        {footer && <div className="px-5 py-4 border-t border-border flex justify-end gap-2">{footer}</div>}
      </div>
    </div>
  )
}

export function PrimaryButton(props: React.ButtonHTMLAttributes<HTMLButtonElement>) {
  const { className = '', children, ...rest } = props
  return (
    <button
      {...rest}
      className={`inline-flex items-center justify-center gap-2 px-4 py-2 rounded-ios-sm bg-primary text-white font-medium hover:bg-primary-hover disabled:opacity-50 transition-colors min-h-[44px] ${className}`}
    >
      {children}
    </button>
  )
}

export function SecondaryButton(props: React.ButtonHTMLAttributes<HTMLButtonElement>) {
  const { className = '', children, ...rest } = props
  return (
    <button
      {...rest}
      className={`inline-flex items-center justify-center gap-2 px-4 py-2 rounded-ios-sm bg-bg text-text font-medium hover:bg-border disabled:opacity-50 transition-colors min-h-[44px] ${className}`}
    >
      {children}
    </button>
  )
}

export function SectionCard({ title, action, children, subtitle }: {
  title: string
  subtitle?: string
  action?: ReactNode
  children: ReactNode
}) {
  return (
    <section className="bg-card rounded-ios shadow-ios p-4 sm:p-5">
      <div className="flex items-center justify-between gap-3 mb-3">
        <div>
          <h3 className="font-semibold">{title}</h3>
          {subtitle && <p className="text-sm text-text-secondary mt-0.5">{subtitle}</p>}
        </div>
        {action}
      </div>
      {children}
    </section>
  )
}

export function EmptyBlock({ title, subtitle, action }: { title: string; subtitle?: string; action?: ReactNode }) {
  return (
    <div className="border-2 border-dashed border-border rounded-ios p-6 text-center">
      <p className="font-medium">{title}</p>
      {subtitle && <p className="text-sm text-text-secondary mt-1">{subtitle}</p>}
      {action && <div className="mt-4 flex justify-center">{action}</div>}
    </div>
  )
}
