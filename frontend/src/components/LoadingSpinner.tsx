interface LoadingSpinnerProps {
  text?: string
  className?: string
}

export default function LoadingSpinner({ text, className = '' }: LoadingSpinnerProps) {
  return (
    <div className={`flex flex-col items-center justify-center py-16 ${className}`}>
      <div className="w-8 h-8 border-2 border-border border-t-primary rounded-full animate-spin mb-3" />
      {text && <p className="text-sm text-text-secondary">{text}</p>}
    </div>
  )
}
