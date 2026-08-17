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

type ModalState = 'select' | 'analyzing' | 'result'

interface BillFormData {
  year: number
  month: number
  amount: string
  invoice_number: string
}

interface PendingFile {
  file: File
  status: 'pending' | 'ok' | 'error'
  error?: string
  analyzerUsed?: string
  fileHash?: string
  formData: BillFormData
}

const MAX_SIZE = 10 * 1024 * 1024
const ALLOWED_EXTS = ['pdf', 'png', 'jpg', 'jpeg']

function formatSize(bytes: number): string {
  if (bytes >= 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
  if (bytes >= 1024) return `${Math.round(bytes / 1024)} KB`
  return `${bytes} B`
}

function validateFile(file: File): string | null {
  const ext = file.name.split('.').pop()?.toLowerCase()
  if (!ext || !ALLOWED_EXTS.includes(ext)) return 'Formato no soportado. Usá PDF, PNG o JPG.'
  if (file.size > MAX_SIZE) return 'Archivo demasiado grande (máx 10MB).'
  return null
}

export default function UploadBillModal({ isOpen, serviceId, frequency, onClose, onSaved }: UploadBillModalProps) {
  const { showToast } = useToast()
  const [state, setState] = useState<ModalState>('select')
  const [files, setFiles] = useState<PendingFile[]>([])
  const [progress, setProgress] = useState(0)
  const [saving, setSaving] = useState(false)
  const fileRef = useRef<HTMLInputElement>(null)
  const modalRef = useRef<HTMLDivElement>(null)

  const defaultFormData = (): BillFormData => ({
    year: new Date().getFullYear(),
    month: new Date().getMonth() + 1,
    amount: '',
    invoice_number: '',
  })

  useEffect(() => {
    if (isOpen) {
      setState('select')
      setFiles([])
      setProgress(0)
      setSaving(false)
    }
  }, [isOpen])

  if (!isOpen) return null

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const selected = e.target.files ? Array.from(e.target.files) : []
    setFiles(selected.map((f) => ({ file: f, status: 'pending', formData: defaultFormData() })))
    setProgress(0)
    if (fileRef.current) fileRef.current.value = ''
  }

  const addFiles = (newFiles: File[]) => {
    setFiles((prev) => [
      ...prev,
      ...newFiles.map((f) => ({ file: f, status: 'pending' as const, formData: defaultFormData() })),
    ])
    setProgress(0)
  }

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault()
    const dropped = e.dataTransfer.files ? Array.from(e.dataTransfer.files) : []
    if (dropped.length > 0) addFiles(dropped)
  }

  const analyzeOne = async (index: number) => {
    const entry = files[index]
    try {
      const res = await api.bills.uploadAndAnalyze(serviceId, entry.file)
      if (!res?.extracted) {
        throw new Error('No se pudieron extraer datos del documento')
      }
      const ex = res.extracted
      const formData: BillFormData = {
        year: ex.year || new Date().getFullYear(),
        month: frequency === 'yearly' ? 0 : ex.month || new Date().getMonth() + 1,
        amount: ex.amount != null ? String(ex.amount) : '',
        invoice_number: ex.invoice_number || '',
      }
      setFiles((prev) =>
        prev.map((p, i) =>
          i === index
            ? {
                ...p,
                status: 'ok' as const,
                analyzerUsed: res.analyzer_used || '',
                fileHash: res.file_hash || '',
                formData,
              }
            : p,
        ),
      )
    } catch (err: unknown) {
      const message = (err as { message?: string })?.message || 'Error al analizar el documento'
      setFiles((prev) =>
        prev.map((p, i) => (i === index ? { ...p, status: 'error' as const, error: message } : p)),
      )
    }
  }

  const handleAnalyze = async () => {
    if (files.length === 0) {
      showToast('Seleccioná al menos un archivo para analizar', 'error')
      return
    }
    setState('analyzing')
    for (let i = 0; i < files.length; i++) {
      setProgress(i + 1)
      await analyzeOne(i)
    }
    setState('result')
  }

  const updateFormData = (index: number, patch: Partial<BillFormData>) => {
    setFiles((prev) =>
      prev.map((p, i) => (i === index ? { ...p, formData: { ...p.formData, ...patch } } : p)),
    )
  }

  const handleSave = async () => {
    setSaving(true)
    let created = 0
    let updated = 0
    let duplicates = 0
    let errors = 0

    for (let i = 0; i < files.length; i++) {
      const entry = files[i]
      if (entry.status !== 'ok') {
        if (entry.status === 'error') errors++
        continue
      }
      const amount = parseFloat(entry.formData.amount)
      if (
        isNaN(amount) || amount < 0 ||
        !entry.formData.year || entry.formData.year < 2000 || entry.formData.year > 2100
      ) {
        errors++
        continue
      }
      const month = frequency === 'yearly' ? 0 : entry.formData.month
      try {
        const result = await api.bills.createBillFromExtracted(serviceId, {
          amount,
          invoice_number: entry.formData.invoice_number,
          year: entry.formData.year,
          month,
          file_hash: entry.fileHash || undefined,
        })
        if (result?.duplicate) duplicates++
        else if (result?.updated) updated++
        else created++
      } catch {
        errors++
      }
    }
    setSaving(false)

    const parts: string[] = []
    if (created > 0) parts.push(`${created} creadas`)
    if (updated > 0) parts.push(`${updated} actualizadas`)
    if (duplicates > 0) parts.push(`${duplicates} ya importadas`)
    if (errors > 0) parts.push(`${errors} con error`)
    const summary = parts.length > 0 ? parts.join(', ') : 'Sin cambios'
    showToast(summary, errors > 0 && created === 0 && updated === 0 && duplicates === 0 ? 'error' : 'success')
    onSaved()
    onClose()
  }

  const months = Array.from({ length: 12 }, (_, i) => i + 1)

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
      <div ref={modalRef} className="bg-card rounded-ios shadow-ios w-full max-w-2xl mx-auto max-h-[85vh] flex flex-col">
        <div className="p-4 border-b border-border flex items-center justify-between">
          <h3 className="text-lg font-bold">Subir facturas</h3>
          <button
            onClick={onClose}
            disabled={saving}
            className="w-8 h-8 rounded-full flex items-center justify-center hover:bg-bg transition-colors disabled:opacity-50"
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
                onDragOver={(e) => e.preventDefault()}
                onDrop={handleDrop}
                className="border-2 border-dashed border-border rounded-ios p-8 text-center cursor-pointer hover:border-primary hover:bg-primary/5 transition-colors"
              >
                <Icon name="upload" className="w-10 h-10 mx-auto mb-3 text-text-secondary" />
                <p className="text-sm font-medium">
                  {files.length > 0 ? `${files.length} archivo(s) seleccionado(s)` : 'Seleccionar archivos'}
                </p>
                <p className="text-xs text-text-secondary mt-1">PDF, PNG o JPG (máx 10MB c/u) — podés elegir varios</p>
                <input
                  ref={fileRef}
                  type="file"
                  multiple
                  accept=".pdf,.png,.jpg,.jpeg"
                  onChange={handleFileChange}
                  className="hidden"
                />
              </div>

              {files.length > 0 && (
                <div className="space-y-2">
                  {files.map((f, i) => {
                    const err = validateFile(f.file)
                    return (
                      <div key={i} className="flex items-center justify-between gap-3 px-3 py-2 bg-bg rounded-ios-sm border border-border">
                        <div className="flex items-center gap-2 min-w-0">
                          <Icon name="pdf" className="w-4 h-4 text-text-secondary shrink-0" />
                          <span className="text-sm truncate">{f.file.name}</span>
                        </div>
                        <div className="flex items-center gap-2 shrink-0">
                          <span className="text-xs text-text-secondary">{formatSize(f.file.size)}</span>
                          {err ? (
                            <span className="text-xs text-danger">Formato inválido</span>
                          ) : (
                            <span className="text-xs text-success">OK</span>
                          )}
                        </div>
                      </div>
                    )
                  })}
                </div>
              )}

              <button
                onClick={handleAnalyze}
                disabled={files.length === 0}
                className="w-full flex items-center justify-center gap-2 px-4 py-2.5 bg-primary text-white rounded-ios-sm hover:bg-primary-hover disabled:opacity-50 transition-colors min-h-[44px]"
              >
                <Icon name="search" className="w-4 h-4" />
                {files.length > 0 ? `Analizar ${files.length} factura${files.length > 1 ? 's' : ''}` : 'Analizar factura'}
              </button>
            </>
          )}

          {state === 'analyzing' && (
            <div className="text-center py-10">
              <div className="w-10 h-10 mx-auto mb-4 rounded-full border-4 border-primary/30 border-t-primary animate-spin" />
              <p className="text-sm text-text-secondary">
                Analizando documento {Math.min(progress, files.length)} de {files.length}...
              </p>
            </div>
          )}

          {state === 'result' && (
            <>
              <div className="space-y-4">
                {files.map((f, i) => (
                  <div key={i} className="border border-border rounded-ios-sm p-4 space-y-3">
                    <div className="flex items-center justify-between gap-3">
                      <div className="flex items-center gap-2 min-w-0">
                        <Icon name="pdf" className="w-4 h-4 text-text-secondary shrink-0" />
                        <span className="text-sm font-medium truncate">{f.file.name}</span>
                      </div>
                      <span className="text-xs shrink-0">
                        {f.status === 'ok' && <span className="text-success">Datos extraídos</span>}
                        {f.status === 'error' && <span className="text-danger">Error</span>}
                        {f.status === 'pending' && <span className="text-text-secondary">Pendiente</span>}
                      </span>
                    </div>

                    {f.analyzerUsed && (
                      <div className="flex items-center gap-2 px-3 py-2 bg-primary/10 text-primary rounded-ios-sm text-sm">
                        <Icon name="pdf" className="w-4 h-4" />
                        Analizado con: {f.analyzerUsed}
                      </div>
                    )}

                    {f.status === 'ok' && (
                      <div className="space-y-3">
                        <div className="grid grid-cols-2 gap-3">
                          <div>
                            <label className="block text-sm font-medium mb-1">Año</label>
                            <input
                              type="number"
                              value={f.formData.year}
                              onChange={(e) => updateFormData(i, { year: Number(e.target.value) })}
                              className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px]"
                            />
                          </div>
                          {frequency !== 'yearly' ? (
                            <div>
                              <label className="block text-sm font-medium mb-1">Mes</label>
                              <select
                                value={f.formData.month}
                                onChange={(e) => updateFormData(i, { month: Number(e.target.value) })}
                                className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary bg-card min-h-[44px]"
                              >
                                {months.map((m) => (
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
                            value={f.formData.amount}
                            onChange={(e) => updateFormData(i, { amount: e.target.value })}
                            className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px]"
                          />
                        </div>
                        <div>
                          <label className="block text-sm font-medium mb-1">Número de factura</label>
                          <input
                            type="text"
                            value={f.formData.invoice_number}
                            onChange={(e) => updateFormData(i, { invoice_number: e.target.value })}
                            className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px]"
                          />
                        </div>
                      </div>
                    )}

                    {f.status === 'error' && f.error && (
                      <p className="text-sm text-danger">{f.error}</p>
                    )}
                  </div>
                ))}
              </div>

              <div className="flex justify-end gap-3 pt-2">
                <button
                  onClick={() => setState('select')}
                  disabled={saving}
                  className="px-4 py-2 bg-bg text-text rounded-ios-sm hover:bg-border transition-colors min-h-[44px] text-sm disabled:opacity-50"
                >
                  Volver
                </button>
                <button
                  onClick={handleSave}
                  disabled={saving}
                  className="px-4 py-2 bg-primary text-white rounded-ios-sm hover:bg-primary-hover disabled:opacity-50 transition-colors flex items-center gap-2 min-h-[44px] text-sm"
                >
                  {saving ? (
                    <div className="w-4 h-4 rounded-full border-2 border-white/30 border-t-white animate-spin" />
                  ) : (
                    <Icon name="save" className="w-4 h-4" />
                  )}
                  {saving ? 'Guardando...' : 'Guardar facturas'}
                </button>
              </div>
            </>
          )}
        </div>
      </div>
    </div>
  )
}