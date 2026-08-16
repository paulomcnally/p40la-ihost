import { useState, useRef, useEffect } from 'react'
import { api } from '../api'
import { Icon } from './Icons'
import { useToast } from './Toast'

interface UploadBillModalProps {
  isOpen: boolean
  serviceId: number
  frequency: 'monthly' | 'yearly'
  onClose: () => void
  onSaved: () => void
}

type ModalState = 'select' | 'analyzing' | 'result' | 'error'

export default function UploadBillModal({ isOpen, serviceId, frequency, onClose, onSaved }: UploadBillModalProps) {
  const { showToast } = useToast()
  const [state, setState] = useState<ModalState>('select')
  const [file, setFile] = useState<File | null>(null)
  const [analyzerUsed, setAnalyzerUsed] = useState('')
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)
  const [formData, setFormData] = useState({
    year: new Date().getFullYear(),
    month: new Date().getMonth() + 1,
    amount: '',
    invoice_number: '',
  })
  const fileRef = useRef<HTMLInputElement>(null)
  const modalRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (isOpen) {
      setState('select')
      setFile(null)
      setError('')
      setAnalyzerUsed('')
      setFormData({
        year: new Date().getFullYear(),
        month: new Date().getMonth() + 1,
        amount: '',
        invoice_number: '',
      })
    }
  }, [isOpen])

  if (!isOpen) return null

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const selected = e.target.files?.[0] || null
    setError('')
    if (selected) {
      const ext = selected.name.split('.').pop()?.toLowerCase()
      if (ext && !['pdf', 'png', 'jpg', 'jpeg'].includes(ext)) {
        setError('Formato no soportado. Usá PDF, PNG o JPG.')
        setFile(null)
        return
      }
      setFile(selected)
    } else {
      setFile(null)
    }
  }

  const handleAnalyze = async () => {
    if (!file) {
      setError('Seleccioná un archivo para analizar')
      return
    }
    setState('analyzing')
    setError('')
    try {
      const res = await api.bills.uploadAndAnalyze(serviceId, file)
      if (!res?.extracted) {
        throw new Error('No se pudieron extraer datos del documento')
      }
      const ex = res.extracted
      setFormData({
        year: ex.year || new Date().getFullYear(),
        month: frequency === 'yearly' ? 0 : ex.month || new Date().getMonth() + 1,
        amount: ex.amount != null ? String(ex.amount) : '',
        invoice_number: ex.invoice_number || '',
      })
      setAnalyzerUsed(res.analyzer_used || '')
      setState('result')
    } catch (err: unknown) {
      const message = (err as { message?: string })?.message || 'Error al analizar el documento'
      setError(message)
      setState('error')
    }
  }

  const handleSave = async () => {
    setSaving(true)
    setError('')
    try {
      const amount = parseFloat(formData.amount)
      if (isNaN(amount) || amount < 0) {
        throw new Error('Monto inválido')
      }
      if (!formData.year || formData.year < 2000 || formData.year > 2100) {
        throw new Error('Año inválido')
      }
      const month = frequency === 'yearly' ? 0 : formData.month
      const result = await api.bills.createBillFromExtracted(serviceId, {
        amount,
        invoice_number: formData.invoice_number,
        year: formData.year,
        month,
      })
      showToast(result?.updated ? 'Factura actualizada con datos del analizador' : 'Factura guardada', 'success')
      onSaved()
      onClose()
    } catch (err: unknown) {
      const message = (err as { message?: string })?.message || 'Error al guardar la factura'
      setError(message)
    } finally {
      setSaving(false)
    }
  }

  const months = Array.from({ length: 12 }, (_, i) => i + 1)

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
      <div ref={modalRef} className="bg-card rounded-ios shadow-ios w-full max-w-md mx-auto max-h-[85vh] flex flex-col">
        <div className="p-4 border-b border-border flex items-center justify-between">
          <h3 className="text-lg font-bold">Subir factura</h3>
          <button
            onClick={onClose}
            className="w-8 h-8 rounded-full flex items-center justify-center hover:bg-bg transition-colors"
            aria-label="Cerrar"
          >
            <Icon name="cancel" className="w-5 h-5 text-text-secondary" />
          </button>
        </div>

        <div className="p-4 overflow-y-auto flex-1 space-y-4">
          {state === 'select' && (
            <>
              <div
                onClick={() => fileRef.current?.click()}
                className="border-2 border-dashed border-border rounded-ios p-8 text-center cursor-pointer hover:border-primary hover:bg-primary/5 transition-colors"
              >
                <Icon name="upload" className="w-10 h-10 mx-auto mb-3 text-text-secondary" />
                <p className="text-sm font-medium">{file ? file.name : 'Seleccionar archivo'}</p>
                <p className="text-xs text-text-secondary mt-1">PDF, PNG o JPG (máx 10MB)</p>
                <input
                  ref={fileRef}
                  type="file"
                  accept=".pdf,.png,.jpg,.jpeg"
                  onChange={handleFileChange}
                  className="hidden"
                />
              </div>
              {error && <p className="text-sm text-danger">{error}</p>}
              <button
                onClick={handleAnalyze}
                disabled={!file}
                className="w-full flex items-center justify-center gap-2 px-4 py-2.5 bg-primary text-white rounded-ios-sm hover:bg-primary-hover disabled:opacity-50 transition-colors min-h-[44px]"
              >
                <Icon name="search" className="w-4 h-4" />
                Analizar factura
              </button>
            </>
          )}

          {state === 'analyzing' && (
            <div className="text-center py-10">
              <div className="w-10 h-10 mx-auto mb-4 rounded-full border-4 border-primary/30 border-t-primary animate-spin" />
              <p className="text-sm text-text-secondary">Analizando documento...</p>
            </div>
          )}

          {(state === 'result' || state === 'error') && (
            <>
              {analyzerUsed && (
                <div className="flex items-center gap-2 px-3 py-2 bg-primary/10 text-primary rounded-ios-sm text-sm">
                  <Icon name="pdf" className="w-4 h-4" />
                  Analizado con: {analyzerUsed}
                </div>
              )}

              {state === 'result' && (
                <div className="space-y-4">
                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <label className="block text-sm font-medium mb-1">Año</label>
                      <input
                        type="number"
                        value={formData.year}
                        onChange={(e) => setFormData(prev => ({ ...prev, year: Number(e.target.value) }))}
                        className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px]"
                      />
                    </div>
                    {frequency !== 'yearly' ? (
                      <div>
                        <label className="block text-sm font-medium mb-1">Mes</label>
                        <select
                          value={formData.month}
                          onChange={(e) => setFormData(prev => ({ ...prev, month: Number(e.target.value) }))}
                          className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary bg-card min-h-[44px]"
                        >
                          {months.map(m => (
                            <option key={m} value={m}>{m}</option>
                          ))}
                        </select>
                      </div>
                    ) : (
                      <div className="flex items-center px-3 py-2 bg-bg rounded-ios-sm border border-border">
                        <span className="text-sm text-text-secondary">Anual</span>
                      </div>
                    )}
                  </div>
                  <div>
                    <label className="block text-sm font-medium mb-1">Monto</label>
                    <input
                      type="number"
                      step="0.01"
                      value={formData.amount}
                      onChange={(e) => setFormData(prev => ({ ...prev, amount: e.target.value }))}
                      className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px]"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium mb-1">Número de factura</label>
                    <input
                      type="text"
                      value={formData.invoice_number}
                      onChange={(e) => setFormData(prev => ({ ...prev, invoice_number: e.target.value }))}
                      className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px]"
                    />
                  </div>
                </div>
              )}

              {error && <p className="text-sm text-danger">{error}</p>}

              {state === 'result' && (
                <div className="flex justify-end gap-3 pt-2">
                  <button
                    onClick={() => setState('select')}
                    disabled={saving}
                    className="px-4 py-2 bg-bg text-text rounded-ios-sm hover:bg-border transition-colors min-h-[44px] text-sm"
                  >
                    Otra factura
                  </button>
                  <button
                    onClick={handleSave}
                    disabled={saving}
                    className="px-4 py-2 bg-primary text-white rounded-ios-sm hover:bg-primary-hover disabled:opacity-50 transition-colors flex items-center gap-2 min-h-[44px] text-sm"
                  >
                    <Icon name="save" className="w-4 h-4" />
                    Guardar factura
                  </button>
                </div>
              )}

              {state === 'error' && (
                <div className="flex justify-end pt-2">
                  <button
                    onClick={() => setState('select')}
                    className="px-4 py-2 bg-primary text-white rounded-ios-sm hover:bg-primary-hover transition-colors min-h-[44px] text-sm"
                  >
                    Intentar de nuevo
                  </button>
                </div>
              )}
            </>
          )}
        </div>
      </div>
    </div>
  )
}
