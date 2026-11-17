interface ToggleProps {
  checked: boolean
  onChange: (checked: boolean) => void
  disabled?: boolean
  ariaLabel?: string
}

// Toggle de estilo iOS. Único control válido para booleanos de entidad/configuración
// (ver docs/project-rules.md §4.10 y AGENTS.md). No usar <input type="checkbox"> nativo.
export default function Toggle({ checked, onChange, disabled, ariaLabel }: ToggleProps) {
  return (
    <label className={`relative inline-flex items-center cursor-pointer ${disabled ? 'opacity-50 pointer-events-none' : ''}`}>
      <input
        type="checkbox"
        role="switch"
        checked={checked}
        onChange={(e) => onChange(e.target.checked)}
        disabled={disabled}
        aria-label={ariaLabel}
        className="sr-only peer"
      />
      <div className="w-11 h-6 bg-border rounded-full peer transition-colors peer-focus-visible:ring-2 peer-focus-visible:ring-primary/50 peer-focus-visible:ring-offset-2 peer-checked:bg-success after:content-[''] after:absolute after:top-0.5 after:left-0.5 after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:after:translate-x-full"></div>
    </label>
  )
}
