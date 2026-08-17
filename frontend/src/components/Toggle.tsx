interface ToggleProps {
  checked: boolean
  onChange: (checked: boolean) => void
  disabled?: boolean
}

// Toggle de estilo iOS (mismo patrón de ServiceFormPage).
export default function Toggle({ checked, onChange, disabled }: ToggleProps) {
  return (
    <label className={`relative inline-flex items-center cursor-pointer ${disabled ? 'opacity-50 pointer-events-none' : ''}`}>
      <input
        type="checkbox"
        checked={checked}
        onChange={(e) => onChange(e.target.checked)}
        className="sr-only peer"
      />
      <div className="w-11 h-6 bg-border peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full after:content-[''] after:absolute after:top-0.5 after:left-0.5 after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-success"></div>
    </label>
  )
}