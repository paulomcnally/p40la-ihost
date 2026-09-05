import { useEffect, useState, useCallback, useRef } from 'react'
import { useSearchParams, useNavigate } from 'react-router-dom'
import { useI18nStore } from '../stores/i18nStore'
import { useCurrencyFormatStore } from '../stores/currencyFormatStore'
import { usePageTitle } from '../hooks/usePageTitle'
import { api } from '../api'
import { Icon } from '../components/Icons'
import CreateMenu from '../components/CreateMenu'
import Select from '../components/Select'
import LoadingSpinner from '../components/LoadingSpinner'
import { useToast } from '../components/Toast'
import type { SupportRecord, SalaryPayment, Child, PensionCategory } from '../types'

const PAYMENT_METHODS = ['bank_transfer', 'cash', 'check', 'mobile', 'other']

function formatDateTimeLocal(dt?: string): string {
  if (!dt) return ''
  const d = new Date(dt)
  if (isNaN(d.getTime())) return ''
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

function formatDate(dt?: string): string {
  if (!dt) return '—'
  const d = new Date(dt)
  if (isNaN(d.getTime())) return '—'
  return d.toLocaleDateString()
}

function statusIcon(status: string): string {
  if (status === 'paid') return 'check'
  if (status === 'rejected') return 'cancel'
  return 'clock'
}

function statusColor(status: string): string {
  if (status === 'paid') return 'text-emerald-600 dark:text-emerald-400'
  if (status === 'rejected') return 'text-red-500 dark:text-red-400'
  return 'text-amber-500 dark:text-amber-400'
}

function ModalShell({ title, onClose, children, wide }: {
  title: string
  onClose: () => void
  children: React.ReactNode
  wide?: boolean
}) {
  return (
    <div className="fixed inset-0 bg-black/40 flex items-center justify-center z-50 p-4">
      <div
        className={`bg-card rounded-ios shadow-ios-lg w-full ${wide ? 'max-w-lg' : 'max-w-md'} max-h-[90vh] overflow-y-auto`}
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between px-5 pt-4">
          <h3 className="text-base sm:text-lg font-semibold">{title}</h3>
          <button
            onClick={onClose}
            className="w-8 h-8 rounded-full flex items-center justify-center hover:bg-bg transition-colors"
          >
            <Icon name="cancel" className="w-4 h-4" />
          </button>
        </div>
        <div className="p-5">{children}</div>
      </div>
    </div>
  )
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div>
      <label className="block text-sm font-medium mb-1">{label}</label>
      {children}
    </div>
  )
}

const inputCls = 'w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px]'

function EmptyCard({ title, subtitle, actionLabel, onAction }: {
  title: string
  subtitle: string
  actionLabel: string
  onAction: () => void
}) {
  return (
    <div className="bg-card rounded-ios shadow-ios p-8 sm:p-12 text-center max-w-md mx-auto mt-8">
      <div className="w-16 h-16 mx-auto mb-5 text-primary opacity-80">
        <Icon name="calendar" className="w-full h-full" />
      </div>
      <h3 className="text-lg sm:text-xl font-semibold mb-2">{title}</h3>
      <p className="text-text-secondary mb-6">{subtitle}</p>
      <button
        onClick={onAction}
        className="inline-flex items-center justify-center gap-2 px-6 py-3 bg-card border-2 border-dashed border-border rounded-ios text-primary font-semibold hover:border-primary hover:bg-primary/5 transition-colors min-w-48"
      >
        <Icon name="plus" className="w-5 h-5" />
        {actionLabel}
      </button>
    </div>
  )
}

export default function RegistrosPage() {
  const navigate = useNavigate()
  const { t } = useI18nStore()
  const formatMoney = useCurrencyFormatStore(s => s.formatMoney)
  const { showToast } = useToast()
  const [searchParams, setSearchParams] = useSearchParams()
  const now = new Date()
  const initialYear = searchParams.get('year') ? parseInt(searchParams.get('year')!, 10) : now.getFullYear()
  const initialMonth = searchParams.get('month') ? parseInt(searchParams.get('month')!, 10) : now.getMonth() + 1

  const [year, setYear] = useState(initialYear)
  const [month, setMonth] = useState(initialMonth)
  const [loading, setLoading] = useState(true)

  const [records, setRecords] = useState<SupportRecord[]>([])
  const [salaryPayments, setSalaryPayments] = useState<SalaryPayment[]>([])
  const [closing, setClosing] = useState<{ closed: boolean; closed_at: string | null }>({ closed: false, closed_at: null })
  const [children, setChildren] = useState<Child[]>([])
  const [categories, setCategories] = useState<PensionCategory[]>([])

  // Crear registro manual
  const [showCreate, setShowCreate] = useState(false)
  const [newChildId, setNewChildId] = useState<number | string>('')
  const [newCategoryId, setNewCategoryId] = useState<number | string>('')
  const [newAmount, setNewAmount] = useState('')
  const [newCurrency, setNewCurrency] = useState('NIO')
  const [creating, setCreating] = useState(false)

  // Edición inline
  const [editingId, setEditingId] = useState<number | null>(null)
  const [editCategoryId, setEditCategoryId] = useState<number | string>('')
  const [editAmount, setEditAmount] = useState('')
  const [updating, setUpdating] = useState(false)

  // Modales
  const [payRecord, setPayRecord] = useState<SupportRecord | null>(null)
  const [rejectRecord, setRejectRecord] = useState<SupportRecord | null>(null)
  const [salaryModal, setSalaryModal] = useState<SalaryPayment | null>(null)
  const [showCloseConfirm, setShowCloseConfirm] = useState(false)
  const [showReopenModal, setShowReopenModal] = useState(false)
  const [busy, setBusy] = useState(false)

  // Pay modal state
  const [payPaidAt, setPayPaidAt] = useState('')
  const [payMethod, setPayMethod] = useState<string | number>('')
  const [payReference, setPayReference] = useState('')
  const [payNotes, setPayNotes] = useState('')
  const [proofFile, setProofFile] = useState<File | null>(null)
  const [uploading, setUploading] = useState(false)
  const [showConversion, setShowConversion] = useState(false)
  const [origAmount, setOrigAmount] = useState('')
  const [origCurrency, setOrigCurrency] = useState('')
  const [exchangeRate, setExchangeRate] = useState('')
  const proofInputRef = useRef<HTMLInputElement>(null)

  // Salary modal state
  const [salaryReceivedAt, setSalaryReceivedAt] = useState('')
  const [salaryReceivedAmount, setSalaryReceivedAmount] = useState('')
  const [salaryNotes, setSalaryNotes] = useState('')

  // Reject modal state
  const [rejectReason, setRejectReason] = useState('')

  // Reopen modal state
  const [reopenWord, setReopenWord] = useState('')

  // Generación mensual (SPEC-051)
  const [generating, setGenerating] = useState(false)

  const isClosed = closing.closed
  const periodLabel = `${t(`months.${month}`)} ${year}`
  usePageTitle(periodLabel)

  const loadData = useCallback(async () => {
    setLoading(true)
    try {
      const [recordsData, salaryData, closingData, childrenData, categoriesData] = await Promise.all([
        api.pensionRecords.list({ year, month }),
        api.pensionSalaryPayments.list({ year, month }),
        api.pensionClosing.status(year, month),
        api.children.list(),
        api.pensionCategories.list(),
      ])
      setRecords(recordsData || [])
      setSalaryPayments(salaryData || [])
      if (closingData) setClosing({ closed: !!closingData.closed, closed_at: closingData.closed_at })
      setChildren(childrenData || [])
      setCategories(categoriesData || [])
    } catch (err: any) {
      showToast(err?.message || t('errors.generic'), 'error')
    } finally {
      setLoading(false)
    }
  }, [year, month, showToast, t])

  useEffect(() => {
    loadData()
  }, [loadData])

  const updateMonthYear = (m: number, y: number) => {
    setMonth(m)
    setYear(y)
    setSearchParams({ year: String(y), month: String(m) }, { replace: true })
  }
  const prevMonth = () => (month === 1 ? updateMonthYear(12, year - 1) : updateMonthYear(month - 1, year))
  const nextMonth = () => (month === 12 ? updateMonthYear(1, year + 1) : updateMonthYear(month + 1, year))

  const childOptions = children.map((c) => ({ value: c.id, label: `${c.first_name} ${c.last_name}` }))
  const categoryOptions = categories.map((c) => ({ value: c.id, label: c.name }))
  const pmOptions = PAYMENT_METHODS.map((pm) => ({ value: pm, label: t(`registros.paymentMethods.${pm}`) }))

  const totalAmount = records.reduce((s, r) => s + Number(r.amount), 0)
  const paidAmount = records.filter((r) => r.status === 'paid').reduce((s, r) => s + Number(r.amount), 0)
  const pendingAmount = records.filter((r) => r.status === 'pending').reduce((s, r) => s + Number(r.amount), 0)
  const rejectedAmount = records.filter((r) => r.status === 'rejected').reduce((s, r) => s + Number(r.amount), 0)
  const baseCurrency = records[0]?.currency || 'NIO'

  const openCreateForm = () => {
    if (children.length === 0) {
      showToast(t('registros.no_children'), 'error')
      navigate('/pension/hijos')
      return
    }
    if (categories.length === 0) {
      showToast(t('registros.no_categories'), 'error')
      navigate('/pension/categorias')
      return
    }
    setShowCreate(true)
  }

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!newChildId || !newCategoryId || !newAmount) return
    setCreating(true)
    try {
      await api.pensionRecords.create({
        child_id: Number(newChildId),
        pension_category_id: Number(newCategoryId),
        year,
        month,
        amount: parseFloat(newAmount),
        currency: newCurrency || 'NIO',
      })
      showToast(t('registros.created'), 'success')
      setShowCreate(false)
      setNewChildId('')
      setNewCategoryId('')
      setNewAmount('')
      loadData()
    } catch (err: any) {
      showToast(err?.message || t('registros.save_error'), 'error')
    } finally {
      setCreating(false)
    }
  }

  const startEdit = (record: SupportRecord) => {
    setEditingId(record.id)
    setEditCategoryId(record.pension_category_id)
    setEditAmount(String(Number(record.amount)))
  }
  const cancelEdit = () => {
    setEditingId(null)
    setEditCategoryId('')
    setEditAmount('')
  }
  const handleEdit = async (record: SupportRecord) => {
    if (!editCategoryId || !editAmount) return
    setUpdating(true)
    try {
      await api.pensionRecords.update(record.id, {
        amount: parseFloat(editAmount),
        pension_category_id: Number(editCategoryId),
      })
      showToast(t('registros.updated'), 'success')
      cancelEdit()
      loadData()
    } catch (err: any) {
      showToast(err?.message || t('registros.save_error'), 'error')
    } finally {
      setUpdating(false)
    }
  }

  const openPayModal = (record: SupportRecord) => {
    setPayRecord(record)
    setPayPaidAt(formatDateTimeLocal(new Date().toISOString()))
    setPayMethod('')
    setPayReference('')
    setPayNotes('')
    setProofFile(null)
    setShowConversion(false)
    setOrigAmount('')
    setOrigCurrency('')
    setExchangeRate('')
  }
  const closePayModal = () => setPayRecord(null)

  const handleProofFile = async (file: File) => {
    if (!payRecord) return
    setProofFile(file)
    setUploading(true)
    try {
      await api.pensionRecords.uploadProof(payRecord.id, file)
      showToast(t('registros.upload_proof'), 'success')
    } catch (err: any) {
      setProofFile(null)
      showToast(err?.message || t('registros.save_error'), 'error')
    } finally {
      setUploading(false)
    }
  }

  const handleMarkPaid = async () => {
    if (!payRecord) return
    setBusy(true)
    try {
      const body: any = {
        paid_at: payPaidAt || undefined,
        payment_method: payMethod ? String(payMethod) : undefined,
        payment_reference: payReference || undefined,
        evidence_notes: payNotes || undefined,
      }
      if (showConversion && origAmount && origCurrency) {
        body.original_amount = parseFloat(origAmount)
        body.original_currency = origCurrency.toUpperCase()
        body.exchange_rate = exchangeRate ? parseFloat(exchangeRate) : undefined
      }
      await api.pensionRecords.markPaid(payRecord.id, body)
      showToast(t('registros.paid'), 'success')
      closePayModal()
      loadData()
    } catch (err: any) {
      showToast(err?.message || t('registros.save_error'), 'error')
    } finally {
      setBusy(false)
    }
  }

  const handleMarkPending = async (record: SupportRecord) => {
    try {
      await api.pensionRecords.markPending(record.id)
      showToast(t('registros.pending'), 'success')
      loadData()
    } catch (err: any) {
      showToast(err?.message || t('registros.save_error'), 'error')
    }
  }

  const handleMarkRejected = async () => {
    if (!rejectRecord || !rejectReason.trim()) return
    setBusy(true)
    try {
      await api.pensionRecords.markRejected(rejectRecord.id, rejectReason.trim())
      showToast(t('registros.rejected'), 'success')
      setRejectRecord(null)
      setRejectReason('')
      loadData()
    } catch (err: any) {
      showToast(err?.message || t('registros.save_error'), 'error')
    } finally {
      setBusy(false)
    }
  }

  const openSalaryModal = (sp: SalaryPayment) => {
    setSalaryModal(sp)
    setSalaryReceivedAt(formatDateTimeLocal(new Date().toISOString()))
    setSalaryReceivedAmount(String(Number(sp.amount)))
    setSalaryNotes('')
  }
  const closeSalaryModal = () => setSalaryModal(null)

  const handleMarkSalaryReceived = async () => {
    if (!salaryModal || !salaryReceivedAt) return
    setBusy(true)
    try {
      await api.pensionSalaryPayments.markReceived(salaryModal.id, {
        received_at: salaryReceivedAt,
        received_amount: salaryReceivedAmount ? parseFloat(salaryReceivedAmount) : undefined,
        notes: salaryNotes || undefined,
      })
      showToast(t('salaryPayments.received'), 'success')
      closeSalaryModal()
      loadData()
    } catch (err: any) {
      showToast(err?.message || t('registros.save_error'), 'error')
    } finally {
      setBusy(false)
    }
  }

  const handleMarkSalaryPending = async (sp: SalaryPayment) => {
    try {
      await api.pensionSalaryPayments.markPending(sp.id)
      loadData()
    } catch (err: any) {
      showToast(err?.message || t('registros.save_error'), 'error')
    }
  }

  const handleGenerateMonth = async () => {
    setGenerating(true)
    try {
      const res = await api.pensionGenerate.generate(year, month)
      showToast(t('registros.generated'), 'success')
      if (res && (res.created_salary_payments > 0 || res.created_support_records > 0)) {
        loadData()
      }
    } catch (err: any) {
      showToast(err?.message || t('registros.save_error'), 'error')
    } finally {
      setGenerating(false)
    }
  }

  const handleCloseMonth = async () => {
    setBusy(true)
    try {
      await api.pensionClosing.close(year, month)
      showToast(t('closing.closeSuccess'), 'success')
      setShowCloseConfirm(false)
      loadData()
    } catch (err: any) {
      showToast(err?.message || t('registros.save_error'), 'error')
    } finally {
      setBusy(false)
    }
  }

  const handleReopenMonth = async () => {
    setBusy(true)
    try {
      await api.pensionClosing.reopen(year, month)
      showToast(t('closing.reopenSuccess'), 'success')
      setShowReopenModal(false)
      setReopenWord('')
      loadData()
    } catch (err: any) {
      showToast(err?.message || t('registros.save_error'), 'error')
    } finally {
      setBusy(false)
    }
  }

  const confirmWord = t('closing.reopenModalConfirmWord')
  const isReopenValid = reopenWord.trim().toLowerCase() === confirmWord.toLowerCase()

  const menuOptions = [
    { label: t('registros.create_manual'), icon: 'plus', onClick: openCreateForm },
    ...(!isClosed
      ? [{ label: t('registros.generate'), icon: 'refresh', onClick: handleGenerateMonth }]
      : []),
    ...(records.length > 0 && !isClosed
      ? [{ label: t('registros.close_month'), icon: 'lock', onClick: () => setShowCloseConfirm(true) }]
      : []),
    ...(isClosed
      ? [{ label: t('registros.reopen_month'), icon: 'refresh', onClick: () => setShowReopenModal(true) }]
      : []),
  ]

  return (
    <div>
      {/* Header con selector de mes y menú de acciones */}
      <div className="flex items-center justify-between mb-4 sm:mb-5">
        <h2 className="text-xl sm:text-2xl font-bold">{t('registros.title')}</h2>
        <div className="flex items-center gap-2">
          <div className="flex items-center gap-1 bg-card rounded-ios shadow-ios px-2 py-1">
            <button onClick={prevMonth} className="w-8 h-8 rounded-full flex items-center justify-center hover:bg-bg transition-colors" aria-label="prev">
              <Icon name="chevron" className="w-4 h-4 rotate-180" />
            </button>
            <span className="text-sm font-semibold min-w-28 text-center whitespace-nowrap">{periodLabel}</span>
            <button onClick={nextMonth} className="w-8 h-8 rounded-full flex items-center justify-center hover:bg-bg transition-colors" aria-label="next">
              <Icon name="chevron" className="w-4 h-4" />
            </button>
          </div>
          {isClosed && (
            <span className="text-xs px-2.5 py-1 rounded-full font-medium bg-emerald-600/10 text-emerald-700 border border-emerald-600/30 dark:text-emerald-300 dark:border-emerald-500/30">
              <Icon name="lock" className="w-3 h-3 inline mr-1" />
              {t('registros.closed')}
            </span>
          )}
          <CreateMenu options={menuOptions} />
        </div>
      </div>

      {/* Formulario crear registro manual */}
      {showCreate && !isClosed && (
        <div className="bg-card rounded-ios shadow-ios p-4 sm:p-5 mb-4">
          <form onSubmit={handleCreate} className="space-y-4">
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <Field label={t('registros.child')}>
                <Select options={childOptions} value={newChildId} onChange={setNewChildId} placeholder={t('registros.select_child')} />
              </Field>
              <Field label={t('registros.category')}>
                <Select options={categoryOptions} value={newCategoryId} onChange={setNewCategoryId} placeholder={t('registros.select_category')} />
              </Field>
              <Field label={t('registros.amount')}>
                <input type="number" step="0.01" min="0" required value={newAmount} onChange={(e) => setNewAmount(e.target.value)} className={inputCls} />
              </Field>
              <Field label={t('registros.currency')}>
                <input type="text" value={newCurrency} onChange={(e) => setNewCurrency(e.target.value)} maxLength={3} className={inputCls} />
              </Field>
            </div>
            <div className="flex justify-end gap-3 pt-4 border-t border-border">
              <button type="button" onClick={() => setShowCreate(false)} className="px-4 py-2 bg-bg text-text rounded-ios-sm hover:bg-border transition-colors flex items-center gap-2 min-h-[44px]">
                <Icon name="cancel" className="w-4 h-4" />
                {t('app.cancel')}
              </button>
              <button type="submit" disabled={creating || !newChildId || !newCategoryId || !newAmount} className="px-4 py-2 bg-primary text-white rounded-ios-sm hover:bg-primary-hover disabled:opacity-50 transition-colors flex items-center gap-2 min-h-[44px]">
                <Icon name="save" className="w-4 h-4" />
                {creating ? t('app.loading') : t('app.save')}
              </button>
            </div>
          </form>
        </div>
      )}

      {/* Sección de salarios del mes */}
      {salaryPayments.length > 0 && (
        <div className="mb-4">
          <h3 className="text-sm font-semibold text-text-secondary uppercase tracking-wide mb-2">{t('salaryPayments.title')}</h3>
          <div className="space-y-2">
            {salaryPayments.map((sp) => (
              <SalaryCard
                key={sp.id}
                sp={sp}
                isClosed={isClosed}
                t={t}
                onMarkReceived={() => openSalaryModal(sp)}
                onMarkPending={() => handleMarkSalaryPending(sp)}
              />
            ))}
          </div>
        </div>
      )}

      {/* Cards resumen */}
      {records.length > 0 && (
        <div className="grid grid-cols-2 lg:grid-cols-4 gap-3 sm:gap-4 mb-4">
          <SummaryCard label={t('registros.total_due')} amount={totalAmount} currency={baseCurrency} color="text-white" />
          <SummaryCard label={t('registros.total_paid')} amount={paidAmount} currency={baseCurrency} color="text-emerald-600 dark:text-emerald-400" />
          <SummaryCard label={t('registros.total_pending')} amount={pendingAmount} currency={baseCurrency} color="text-amber-500 dark:text-amber-400" />
          <SummaryCard label={t('registros.total_rejected')} amount={rejectedAmount} currency={baseCurrency} color="text-red-500 dark:text-red-400" />
        </div>
      )}

      {/* Listado de registros */}
      {loading ? (
        <LoadingSpinner text={t('app.loading')} />
      ) : records.length === 0 ? (
        <EmptyCard
          title={t('registros.empty')}
          subtitle={t('registros.subtitle')}
          actionLabel={t('registros.create')}
          onAction={openCreateForm}
        />
      ) : (
        <div className="space-y-2 sm:space-y-3">
          {records.map((record) => (
            <div key={record.id} className="bg-card rounded-ios shadow-ios p-4">
              {editingId === record.id ? (
                <div className="space-y-3">
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                    <Field label={t('registros.category')}>
                      <Select options={categoryOptions} value={editCategoryId} onChange={setEditCategoryId} placeholder={t('registros.select_category')} />
                    </Field>
                    <Field label={t('registros.amount')}>
                      <input type="number" step="0.01" min="0" value={editAmount} onChange={(e) => setEditAmount(e.target.value)} className={inputCls} />
                    </Field>
                  </div>
                  <div className="flex justify-end gap-3">
                    <button onClick={cancelEdit} className="px-3 py-2 bg-bg text-text rounded-ios-sm hover:bg-border transition-colors min-h-[44px]">
                      {t('app.cancel')}
                    </button>
                    <button onClick={() => handleEdit(record)} disabled={updating || !editCategoryId || !editAmount} className="px-4 py-2 bg-primary text-white rounded-ios-sm hover:bg-primary-hover disabled:opacity-50 transition-colors min-h-[44px]">
                      {updating ? t('app.loading') : t('app.save')}
                    </button>
                  </div>
                </div>
              ) : (
                <RecordCard
                  record={record}
                  isClosed={isClosed}
                  t={t}
                  onEdit={() => startEdit(record)}
                  onPay={() => openPayModal(record)}
                  onReject={() => { setRejectRecord(record); setRejectReason('') }}
                  onMarkPending={() => handleMarkPending(record)}
                />
              )}
            </div>
          ))}
        </div>
      )}

      {/* Modal Marcar Pagado */}
      {payRecord && (
        <ModalShell title={t('registros.mark_paid')} onClose={closePayModal}>
          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium mb-1">{t('registros.proof')}</label>
              <input
                ref={proofInputRef}
                type="file"
                accept=".pdf,image/png,image/jpeg,image/webp"
                className="hidden"
                onChange={(e) => {
                  const f = e.target.files?.[0]
                  if (f) handleProofFile(f)
                }}
              />
              <button
                type="button"
                onClick={() => proofInputRef.current?.click()}
                disabled={uploading}
                className={`w-full border-2 border-dashed rounded-ios-sm px-4 py-3 text-sm transition-colors min-h-[44px] ${
                  proofFile
                    ? 'border-emerald-600/50 bg-emerald-600/10 text-emerald-700 dark:text-emerald-300 dark:border-emerald-500/40'
                    : 'border-border text-text-secondary hover:border-primary'
                }`}
              >
                <Icon name={proofFile ? 'check' : 'upload'} className="w-4 h-4 inline mr-2" />
                {uploading ? t('registros.analyzing_proof') : proofFile ? proofFile.name : t('registros.upload_proof_hint')}
              </button>
              {payRecord.proof_file_name && (
                <a href={api.pensionRecords.proofUrl(payRecord.id)} target="_blank" rel="noopener noreferrer" className="inline-flex items-center gap-1 text-sm text-primary hover:underline mt-2">
                  <Icon name="pdf" className="w-4 h-4" />
                  {payRecord.proof_file_name}
                </a>
              )}
            </div>

            <button
              type="button"
              onClick={() => setShowConversion(!showConversion)}
              className="text-sm text-primary hover:underline flex items-center gap-1"
            >
              <Icon name={showConversion ? 'chevron' : 'plus'} className={`w-3 h-3 ${showConversion ? 'rotate-90' : ''}`} />
              {t('registros.currency_conversion')}
            </button>
            {showConversion && (
              <div className="grid grid-cols-2 sm:grid-cols-3 gap-3">
                <Field label={t('registros.amount')}>
                  <input type="number" step="0.01" min="0" value={origAmount} onChange={(e) => setOrigAmount(e.target.value)} className={inputCls} placeholder="USD" />
                </Field>
                <Field label={t('registros.currency')}>
                  <input type="text" value={origCurrency} onChange={(e) => setOrigCurrency(e.target.value)} maxLength={3} className={inputCls} placeholder="USD" />
                </Field>
                <Field label="Tasa">
                  <input type="number" step="0.0001" value={exchangeRate} onChange={(e) => setExchangeRate(e.target.value)} className={inputCls} placeholder="36.50" />
                </Field>
              </div>
            )}

            <Field label={t('registros.paid_at')}>
              <input type="datetime-local" value={payPaidAt} onChange={(e) => setPayPaidAt(e.target.value)} className={inputCls} />
            </Field>
            <Field label={t('registros.payment_method')}>
              <Select options={pmOptions} value={payMethod} onChange={setPayMethod} placeholder={t('registros.select_payment_method')} />
            </Field>
            <Field label={t('registros.payment_reference')}>
              <input type="text" value={payReference} onChange={(e) => setPayReference(e.target.value)} className={inputCls} placeholder={t('registros.payment_reference_placeholder')} />
            </Field>
            <Field label={t('registros.evidence_notes')}>
              <textarea value={payNotes} onChange={(e) => setPayNotes(e.target.value)} rows={2} className={`${inputCls} resize-none`} placeholder={t('registros.evidence_notes_placeholder')} />
            </Field>
            <div className="flex justify-end gap-3 pt-4 border-t border-border">
              <button onClick={closePayModal} className="px-4 py-2 bg-bg text-text rounded-ios-sm hover:bg-border transition-colors min-h-[44px]">
                {t('app.cancel')}
              </button>
              <button onClick={handleMarkPaid} disabled={busy || uploading} className="px-4 py-2 bg-emerald-600 text-white rounded-ios-sm hover:bg-emerald-700 disabled:opacity-50 transition-colors min-h-[44px]">
                {busy ? t('app.loading') : t('registros.mark_paid')}
              </button>
            </div>
          </div>
        </ModalShell>
      )}

      {/* Modal Marcar Salario Recibido */}
      {salaryModal && (
        <ModalShell title={t('salaryPayments.markReceived')} onClose={closeSalaryModal}>
          <div className="space-y-4">
            <Field label={t('salaryPayments.receivedAmount')}>
              <input type="number" step="0.01" min="0" value={salaryReceivedAmount} onChange={(e) => setSalaryReceivedAmount(e.target.value)} className={inputCls} />
              {(() => {
                const expected = Number(salaryModal.amount)
                const received = salaryReceivedAmount ? parseFloat(salaryReceivedAmount) : 0
                const diff = received - expected
                if (!salaryReceivedAmount || diff === 0) return null
                return (
                  <p className={`text-xs mt-1 ${diff > 0 ? 'text-emerald-600 dark:text-emerald-400' : 'text-red-500 dark:text-red-400'}`}>
                    {diff > 0 ? '+' : ''}{formatMoney(diff)} {salaryModal.currency}
                  </p>
                )
              })()}
            </Field>
            <Field label={t('salaryPayments.receivedAt')}>
              <input type="datetime-local" required value={salaryReceivedAt} onChange={(e) => setSalaryReceivedAt(e.target.value)} className={inputCls} />
            </Field>
            <Field label={t('salaryPayments.notes')}>
              <textarea value={salaryNotes} onChange={(e) => setSalaryNotes(e.target.value)} rows={3} className={`${inputCls} resize-none`} />
            </Field>
            <div className="flex justify-end gap-3 pt-4 border-t border-border">
              <button onClick={closeSalaryModal} className="px-4 py-2 bg-bg text-text rounded-ios-sm hover:bg-border transition-colors min-h-[44px]">
                {t('app.cancel')}
              </button>
              <button onClick={handleMarkSalaryReceived} disabled={busy || !salaryReceivedAt} className="px-4 py-2 bg-emerald-600 text-white rounded-ios-sm hover:bg-emerald-700 disabled:opacity-50 transition-colors min-h-[44px]">
                {busy ? t('app.loading') : t('salaryPayments.markReceived')}
              </button>
            </div>
          </div>
        </ModalShell>
      )}

      {/* Modal Rechazar */}
      {rejectRecord && (
        <ModalShell title={t('registros.mark_rejected')} onClose={() => setRejectRecord(null)}>
          <div className="space-y-4">
            <Field label={t('registros.rejection_reason')}>
              <textarea value={rejectReason} onChange={(e) => setRejectReason(e.target.value)} rows={3} className={`${inputCls} resize-none`} placeholder={t('registros.rejection_reason_placeholder')} autoFocus />
            </Field>
            <div className="flex justify-end gap-3 pt-4 border-t border-border">
              <button onClick={() => setRejectRecord(null)} className="px-4 py-2 bg-bg text-text rounded-ios-sm hover:bg-border transition-colors min-h-[44px]">
                {t('app.cancel')}
              </button>
              <button onClick={handleMarkRejected} disabled={busy || !rejectReason.trim()} className="px-4 py-2 bg-red-600 text-white rounded-ios-sm hover:bg-red-700 disabled:opacity-50 transition-colors min-h-[44px]">
                {busy ? t('app.loading') : t('registros.mark_rejected')}
              </button>
            </div>
          </div>
        </ModalShell>
      )}

      {/* Modal confirmar cierre de mes */}
      {showCloseConfirm && (
        <ModalShell title={t('registros.close_month')} onClose={() => setShowCloseConfirm(false)}>
          <p className="text-text-secondary text-sm mb-5">{t('closing.confirmClose')}</p>
          <div className="flex justify-end gap-3">
            <button onClick={() => setShowCloseConfirm(false)} className="px-4 py-2 bg-bg text-text rounded-ios-sm hover:bg-border transition-colors min-h-[44px]">
              {t('app.cancel')}
            </button>
            <button onClick={handleCloseMonth} disabled={busy} className="px-4 py-2 bg-primary text-white rounded-ios-sm hover:bg-primary-hover disabled:opacity-50 transition-colors min-h-[44px]">
              {busy ? t('app.loading') : t('registros.close_month')}
            </button>
          </div>
        </ModalShell>
      )}

      {/* Modal reabrir mes (confirmación por palabra) */}
      {showReopenModal && (
        <ModalShell title={t('closing.reopenModalTitle')} onClose={() => { setShowReopenModal(false); setReopenWord('') }}>
          <p className="text-text-secondary text-sm mb-4">{t('closing.reopenModalMessage').replace('{period}', periodLabel)}</p>
          <div className="mb-4">
            <label className="block text-sm font-medium mb-1">{t('closing.reopenModalInstruction').replace('{word}', confirmWord)}</label>
            <input
              type="text"
              value={reopenWord}
              onChange={(e) => setReopenWord(e.target.value)}
              className={inputCls}
              placeholder={confirmWord}
              autoFocus
            />
          </div>
          <div className="flex justify-end gap-3">
            <button onClick={() => { setShowReopenModal(false); setReopenWord('') }} className="px-4 py-2 bg-bg text-text rounded-ios-sm hover:bg-border transition-colors min-h-[44px]">
              {t('app.cancel')}
            </button>
            <button onClick={handleReopenMonth} disabled={busy || !isReopenValid} className="px-4 py-2 bg-primary text-white rounded-ios-sm hover:bg-primary-hover disabled:opacity-50 transition-colors min-h-[44px]">
              {busy ? t('app.loading') : t('registros.reopen_month')}
            </button>
          </div>
        </ModalShell>
      )}
    </div>
  )
}

function SummaryCard({ label, amount, currency, color }: {
  label: string
  amount: number
  currency: string
  color: string
}) {
  const formatMoney = useCurrencyFormatStore(s => s.formatMoney)
  return (
    <div className="bg-card rounded-ios shadow-ios p-3 sm:p-4">
      <p className="text-text-secondary text-xs font-medium uppercase tracking-wide">{label}</p>
      <p className={`text-lg sm:text-xl font-bold mt-1 ${color}`}>{currency} {formatMoney(amount)}</p>
    </div>
  )
}

function SalaryCard({ sp, isClosed, t, onMarkReceived, onMarkPending }: {
  sp: SalaryPayment
  isClosed: boolean
  t: (key: string, fallback?: string) => string
  onMarkReceived: () => void
  onMarkPending: () => void
}) {
  const formatMoney = useCurrencyFormatStore(s => s.formatMoney)
  const received = sp.status === 'received'
  return (
    <div className="bg-card rounded-ios shadow-ios p-4">
      <div className="flex items-center justify-between gap-3">
        <div className="flex items-center gap-3">
          <div className={`w-10 h-10 rounded-ios flex items-center justify-center ${received ? 'bg-emerald-600/10' : 'bg-amber-500/10'}`}>
            <Icon name={received ? 'check' : 'clock'} className={`w-5 h-5 ${received ? 'text-emerald-600 dark:text-emerald-400' : 'text-amber-500 dark:text-amber-400'}`} />
          </div>
          <div>
            <p className="text-sm font-semibold">{sp.employer}</p>
            <p className="text-xs text-text-secondary">{sp.currency} {formatMoney(Number(sp.amount))}</p>
          </div>
        </div>
        {received ? (
          <button
            onClick={onMarkPending}
            disabled={isClosed}
            className={`text-xs transition-colors min-h-[44px] px-2 ${isClosed ? 'text-text-secondary opacity-50 cursor-not-allowed' : 'text-text-secondary hover:text-amber-500 dark:hover:text-amber-400'}`}
          >
            <Icon name="refresh" className="w-3 h-3 inline mr-1" />
            {t('salaryPayments.markPending')}
          </button>
        ) : (
          <button
            onClick={onMarkReceived}
            disabled={isClosed}
            className={`px-3 py-2 rounded-ios-sm text-sm font-medium transition-colors min-h-[44px] ${
              isClosed
                ? 'bg-bg text-text-secondary opacity-50 cursor-not-allowed'
                : 'bg-emerald-600 text-white hover:bg-emerald-700'
            }`}
          >
            {t('salaryPayments.markReceived')}
          </button>
        )}
      </div>
      {received && (
        <div className="mt-3 pt-3 border-t border-border grid grid-cols-2 sm:grid-cols-4 gap-3 text-xs">
          {sp.received_amount != null && Number(sp.received_amount) !== Number(sp.amount) && (
            <div>
              <p className="text-text-secondary">{t('salaryPayments.receivedAmount')}</p>
              <p className={`mt-0.5 font-medium ${Number(sp.received_amount) > Number(sp.amount) ? 'text-emerald-600 dark:text-emerald-400' : 'text-red-500 dark:text-red-400'}`}>
                {sp.currency} {formatMoney(Number(sp.received_amount))}
              </p>
            </div>
          )}
          <div>
            <p className="text-text-secondary">{t('salaryPayments.receivedAt')}</p>
            <p className="text-text mt-0.5">{formatDate(sp.received_at)}</p>
          </div>
          {sp.notes && (
            <div>
              <p className="text-text-secondary">{t('salaryPayments.notes')}</p>
              <p className="text-text mt-0.5 truncate" title={sp.notes}>{sp.notes}</p>
            </div>
          )}
        </div>
      )}
    </div>
  )
}

function RecordCard({ record, isClosed, t, onEdit, onPay, onReject, onMarkPending }: {
  record: SupportRecord
  isClosed: boolean
  t: (key: string, fallback?: string) => string
  onEdit: () => void
  onPay: () => void
  onReject: () => void
  onMarkPending: () => void
}) {
  const formatMoney = useCurrencyFormatStore(s => s.formatMoney)
  return (
    <div>
      <div className="flex items-center justify-between gap-3">
        <div className="flex items-center gap-3">
          <div className={`w-10 h-10 rounded-ios flex items-center justify-center ${
            record.status === 'paid' ? 'bg-emerald-600/10' : record.status === 'rejected' ? 'bg-red-500/10' : 'bg-amber-500/10'
          }`}>
            <Icon name={statusIcon(record.status)} className={`w-5 h-5 ${statusColor(record.status)}`} />
          </div>
          <div>
            <p className="text-sm font-semibold">{record.category_name}</p>
            <p className="text-xs text-text-secondary">{record.child_name}</p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <p className="font-semibold text-sm">{record.currency} {formatMoney(Number(record.amount))}</p>
          {record.status === 'pending' ? (
            !isClosed ? (
              <div className="flex items-center gap-1">
                <button onClick={onEdit} className="w-8 h-8 rounded-full flex items-center justify-center text-text-secondary hover:text-primary transition-colors" title={t('app.edit')}>
                  <Icon name="edit" className="w-4 h-4" />
                </button>
                <button onClick={onPay} className="px-3 py-1.5 bg-emerald-600 text-white rounded-ios-sm text-xs font-medium hover:bg-emerald-700 transition-colors min-h-[36px]">
                  {t('registros.mark_paid')}
                </button>
                <button onClick={onReject} className="w-8 h-8 rounded-full flex items-center justify-center text-text-secondary hover:text-red-500 dark:hover:text-red-400 transition-colors" title={t('registros.mark_rejected')}>
                  <Icon name="cancel" className="w-4 h-4" />
                </button>
              </div>
            ) : (
              <span className="text-xs text-text-secondary italic">{t('registros.closed_short')}</span>
            )
          ) : (
            <button
              onClick={onMarkPending}
              disabled={isClosed}
              className={`text-xs transition-colors min-h-[44px] px-2 ${isClosed ? 'text-text-secondary opacity-50 cursor-not-allowed' : 'text-text-secondary hover:text-amber-500 dark:hover:text-amber-400'}`}
            >
              <Icon name="refresh" className="w-3 h-3 inline mr-1" />
              {t('registros.mark_pending')}
            </button>
          )}
        </div>
      </div>

      {record.status === 'paid' && (
        <div className="mt-3 pt-3 border-t border-border grid grid-cols-2 sm:grid-cols-4 gap-3 text-xs">
          <div>
            <p className="text-text-secondary">{t('registros.paid_at')}</p>
            <p className="text-text mt-0.5">{formatDate(record.paid_at)}</p>
          </div>
          <div>
            <p className="text-text-secondary">{t('registros.payment_method')}</p>
            <p className="text-text mt-0.5">
              {record.payment_method ? t(`registros.paymentMethods.${record.payment_method}`) : '—'}
            </p>
          </div>
          <div>
            <p className="text-text-secondary">{t('registros.payment_reference')}</p>
            <p className="text-text mt-0.5">{record.payment_reference || '—'}</p>
          </div>
          <div>
            <p className="text-text-secondary">{t('registros.proof')}</p>
            {record.proof_file_name ? (
              <a href={`/api/pension/records/${record.id}/proof`} target="_blank" rel="noopener noreferrer" className="text-primary hover:underline inline-flex items-center gap-1 mt-0.5">
                <Icon name="pdf" className="w-3 h-3" />
                {record.proof_file_name}
              </a>
            ) : (
              <p className="text-text mt-0.5">—</p>
            )}
          </div>
          {record.original_amount != null && record.original_currency && record.original_currency !== record.currency && (
            <div className="col-span-2 sm:col-span-4">
              <p className="text-text-secondary">{t('registros.currency_conversion')}</p>
              <p className="text-text mt-0.5">
                {record.original_currency} {formatMoney(Number(record.original_amount))} × {Number(record.exchange_rate || 0).toFixed(4)} = {record.currency} {formatMoney(Number(record.amount))}
              </p>
            </div>
          )}
          {record.evidence_notes && (
            <div className="col-span-2 sm:col-span-4">
              <p className="text-text-secondary">{t('registros.evidence_notes')}</p>
              <p className="text-text mt-0.5">{record.evidence_notes}</p>
            </div>
          )}
        </div>
      )}

      {record.status === 'rejected' && (
        <div className="mt-3 pt-3 border-t border-border">
          <div className="bg-red-500/5 border border-red-500/20 rounded-ios-sm px-4 py-3">
            <p className="text-red-600 dark:text-red-400 font-medium text-xs mb-1">{t('registros.rejection_reason')}</p>
            <p className="text-text text-sm">{record.notes || '—'}</p>
          </div>
        </div>
      )}
    </div>
  )
}