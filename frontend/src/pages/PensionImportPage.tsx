import { useEffect, useMemo, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { useI18nStore } from '../stores/i18nStore'
import { usePageTitle } from '../hooks/usePageTitle'
import { useCurrencyFormatStore } from '../stores/currencyFormatStore'
import { api } from '../api'
import { Icon } from '../components/Icons'
import Select from '../components/Select'
import LoadingSpinner from '../components/LoadingSpinner'
import { useToast } from '../components/Toast'
import type {
  Child,
  PensionCategory,
  PensionCategoryKeyword,
  PensionImportBatch,
  PensionImportLine,
  PensionImportPreview,
} from '../types'

const inputCls = 'w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px]'
const NEW_CATEGORY = '__new__'

type RowView = PensionImportLine & {
  category_currency: string
  amount_to_record: number
  excluded: boolean
}

function currentPeriod(): { year: number; month: number } {
  const now = new Date()
  return { year: now.getFullYear(), month: now.getMonth() + 1 }
}

function roundShare(total: number, share: number): number {
  return Math.round(total * share) / 100
}

function periodLabel(year: number, month: number, lang: string): string {
  const date = new Date(year, month - 1, 1)
  return date.toLocaleDateString(lang === 'en' ? 'en-US' : 'es-ES', { month: 'long', year: 'numeric' })
}

export default function PensionImportPage() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const { t, lang } = useI18nStore()
  const { showToast } = useToast()
  const formatMoney = useCurrencyFormatStore(s => s.formatMoney)
  usePageTitle(t('pensionImport.title'))

  const initial = currentPeriod()
  const urlYear = Number(searchParams.get('year')) || initial.year
  const urlMonth = Number(searchParams.get('month')) || initial.month

  const [children, setChildren] = useState<Child[]>([])
  const [categories, setCategories] = useState<PensionCategory[]>([])
  const [childId, setChildId] = useState<number | ''>('')
  const [file, setFile] = useState<File | null>(null)
  const [text, setText] = useState('')
  const [preview, setPreview] = useState<PensionImportPreview | null>(null)
  const [rows, setRows] = useState<PensionImportLine[]>([])
  const [excluded, setExcluded] = useState<number[]>([])
  const [loading, setLoading] = useState(false)
  const [confirming, setConfirming] = useState(false)
  const [dirty, setDirty] = useState(false)
  const [showReset, setShowReset] = useState(false)
  const [createFor, setCreateFor] = useState<PensionImportLine | null>(null)
  const [newCategoryName, setNewCategoryName] = useState('')
  const [newCategoryShare, setNewCategoryShare] = useState<50 | 100>(50)
  const [newCategoryCurrency, setNewCategoryCurrency] = useState('NIO')
  const [creatingCategory, setCreatingCategory] = useState(false)
  const [loadingData, setLoadingData] = useState(true)
  const [batches, setBatches] = useState<PensionImportBatch[]>([])
  const [keywords, setKeywords] = useState<PensionCategoryKeyword[]>([])
  const [showKeywords, setShowKeywords] = useState(false)
  const [newKeyword, setNewKeyword] = useState('')
  const [newKeywordCategory, setNewKeywordCategory] = useState<number | ''>('')
  const [savingKeyword, setSavingKeyword] = useState(false)

  const loadLookups = async () => {
    const [childrenData, categoriesData, keywordsData, batchesData] = await Promise.all([
      api.children.list(),
      api.pensionCategories.list(),
      api.pensionImport.keywords.list(),
      api.pensionImport.batches(),
    ])
    setChildren(childrenData || [])
    setCategories(categoriesData || [])
    setKeywords(keywordsData || [])
    setBatches((batchesData || []).slice(0, 5))
    if ((childrenData || []).length === 1) setChildId(childrenData![0].id)
  }

  useEffect(() => {
    loadLookups().finally(() => setLoadingData(false))
  }, [])

  const categoryOptions = useMemo(
    () => [
      ...categories.map(c => ({ value: c.id, label: `${c.name} · ${c.share_percent}%` })),
      { value: NEW_CATEGORY, label: t('pensionImport.new_category_option') },
    ],
    [categories, t],
  )
  const childOptions = children.map(c => ({ value: c.id, label: `${c.first_name} ${c.last_name}` }))
  const shareOptions = [
    { value: 50, label: t('pensionImport.share_shared') },
    { value: 100, label: t('pensionImport.share_full') },
  ]
  const currencyOptions = useMemo(() => {
    const codes = new Set(categories.map(c => c.currency).filter(Boolean))
    codes.add('NIO')
    codes.add('USD')
    return [...codes].map(code => ({ value: code, label: code }))
  }, [categories])

  const rowsWithAmounts: RowView[] = useMemo(() => rows.map(r => {
    const category = categories.find(c => c.id === (r.chosen_category_id ?? r.suggested_category_id))
    const share = category?.share_percent ?? r.share_percent ?? 50
    return {
      ...r,
      share_percent: share,
      category_currency: category?.currency ?? r.currency ?? 'NIO',
      amount_to_record: roundShare(r.amount_total, share),
      category_name: category?.name ?? r.category_name ?? '',
      excluded: excluded.includes(r.line_no),
    }
  }), [rows, categories, excluded])

  const includedRows = rowsWithAmounts.filter(r => !r.excluded)

  const categoryCounts = useMemo(() => {
    const counts = new Map<number, number>()
    for (const r of includedRows) {
      const id = r.chosen_category_id ?? r.suggested_category_id
      if (id) counts.set(id, (counts.get(id) ?? 0) + 1)
    }
    return counts
  }, [includedRows])

  const rowError = (r: RowView): string | null => {
    const id = r.chosen_category_id ?? r.suggested_category_id
    if (!id) return t('pensionImport.error_missing_category')
    if (!r.amount_total || r.amount_total <= 0) return t('pensionImport.error_invalid_amount')
    if (r.currency && r.category_currency && r.currency.toUpperCase() !== r.category_currency.toUpperCase()) {
      return t('pensionImport.error_currency')
    }
    if ((categoryCounts.get(id) ?? 0) > 1) return t('pensionImport.error_duplicate')
    return null
  }

  const hasErrors = includedRows.some(r => rowError(r) !== null)
  const exactShared = includedRows
    .filter(r => r.share_percent === 50)
    .reduce((sum, r) => sum + r.amount_total * 0.5, 0)
  const computedShared = includedRows
    .filter(r => r.share_percent === 50)
    .reduce((sum, r) => sum + r.amount_to_record, 0)
  const computedGrand = includedRows.reduce((sum, r) => sum + r.amount_to_record, 0)
  const declared = preview?.validation.declared_reimbursable_total ?? 0
  const hasDeclared = declared > 0
  const difference = hasDeclared ? declared - computedShared : 0
  const totalMismatch = hasDeclared && (
    declared < Math.min(exactShared, computedShared) - 0.0001 ||
    declared > Math.max(exactShared, computedShared) + 0.0001
  )
  const canConfirm = !hasErrors && !totalMismatch && includedRows.length > 0
  const currency = includedRows[0]?.category_currency || includedRows[0]?.currency || 'NIO'
  const money = (value: number, code = currency) => `${code} ${formatMoney(value)}`
  const period = preview?.period
  const childName = children.find(c => c.id === childId)

  const reset = () => {
    setPreview(null)
    setRows([])
    setExcluded([])
    setFile(null)
    setText('')
    setDirty(false)
    setShowReset(false)
  }

  const handlePreview = async () => {
    if (!file && !text.trim()) {
      showToast(t('pensionImport.select_file_or_text'), 'error')
      return
    }
    setLoading(true)
    try {
      const data = file
        ? await api.pensionImport.previewFile(file, { year: urlYear, month: urlMonth, child_id: childId || undefined })
        : await api.pensionImport.previewText({ text, year: urlYear, month: urlMonth, child_id: childId || undefined })
      if (!data) return
      setPreview(data)
      setRows(data.lines)
      setExcluded([])
      setDirty(false)
    } catch (err: any) {
      const code = err?.message || ''
      if (code.toLowerCase().includes('pdf') && code.toLowerCase().includes('texto')) {
        showToast(t('pensionImport.pdf_without_text'), 'error')
      } else {
        showToast(code || t('errors.generic'), 'error')
      }
    } finally {
      setLoading(false)
    }
  }

  const updateRow = (lineNo: number, patch: Partial<PensionImportLine>) => {
    setRows(prev => prev.map(r => r.line_no === lineNo ? { ...r, ...patch } : r))
    setDirty(true)
  }

  const toggleExcluded = (lineNo: number) => {
    setExcluded(prev => prev.includes(lineNo) ? prev.filter(n => n !== lineNo) : [...prev, lineNo])
    setDirty(true)
  }

  const openCreateCategory = (row: PensionImportLine) => {
    setCreateFor(row)
    setNewCategoryName(row.concept)
    setNewCategoryShare(50)
    setNewCategoryCurrency(row.currency || 'NIO')
  }

  const handleCreateKeyword = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!newKeywordCategory || !newKeyword.trim()) return
    setSavingKeyword(true)
    try {
      await api.pensionImport.keywords.create({ pension_category_id: Number(newKeywordCategory), keyword: newKeyword })
      setNewKeyword('')
      await loadLookups()
      showToast(t('pensionImport.keyword_saved'), 'success')
    } catch (err: any) {
      showToast(err?.message || t('errors.generic'), 'error')
    } finally {
      setSavingKeyword(false)
    }
  }

  const handleDeleteKeyword = async (id: number) => {
    try {
      await api.pensionImport.keywords.delete(id)
      await loadLookups()
    } catch (err: any) {
      showToast(err?.message || t('errors.generic'), 'error')
    }
  }

  const handleCreateCategory = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!createFor) return
    setCreatingCategory(true)
    try {
      const created = await api.pensionCategories.create({
        name: newCategoryName,
        description: '',
        auto_generate: false,
        share_percent: newCategoryShare,
        currency: newCategoryCurrency,
      })
      if (created) {
        await loadLookups()
        updateRow(createFor.line_no, {
          chosen_category_id: created.id,
          suggested_category_id: created.id,
          share_percent: created.share_percent,
          currency: created.currency,
          match_reason: 'manual',
          confidence: 1,
        })
        showToast(t('pensionImport.category_created'), 'success')
      }
      setCreateFor(null)
    } catch (err: any) {
      showToast(err?.message || t('categorias.save_error'), 'error')
    } finally {
      setCreatingCategory(false)
    }
  }

  const handleConfirm = async () => {
    if (!preview || !childId) {
      showToast(t('pensionImport.select_child'), 'error')
      return
    }
    setConfirming(true)
    try {
      const result = await api.pensionImport.confirm({
        batch_id: preview.batch_id,
        child_id: Number(childId),
        rows: includedRows.map(r => ({
          line_no: r.line_no,
          action: 'create',
          category_id: Number(r.chosen_category_id ?? r.suggested_category_id),
          amount_total: r.amount_total,
          status: 'pending',
        })),
      })
      if (result) {
        showToast(t('pensionImport.confirmed').replace('{created}', String(result.created)), 'success')
        navigate(`/pension/registros?year=${preview.period.year}&month=${preview.period.month}`, { replace: true })
      }
    } catch (err: any) {
      showToast(err?.message || t('pensionImport.confirm_error'), 'error')
    } finally {
      setConfirming(false)
    }
  }

  if (loadingData) return <LoadingSpinner />

  return (
    <div className="space-y-4 sm:space-y-5">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div>
          <h2 className="text-xl sm:text-2xl font-bold">{t('pensionImport.title')}</h2>
          <p className="text-sm text-text-secondary mt-1">{t('pensionImport.subtitle')}</p>
        </div>
        <div className="flex items-center gap-2">
          <button
            type="button"
            onClick={() => setShowKeywords(true)}
            className="inline-flex items-center gap-2 px-3 py-2 bg-card rounded-ios shadow-ios text-sm text-primary hover:bg-bg transition-colors min-h-[44px]"
          >
            <Icon name="key" className="w-4 h-4" />
            {t('pensionImport.keywords')}
          </button>
          <a
            href={api.pensionImport.templateUrl}
            className="inline-flex items-center gap-2 px-3 py-2 bg-card rounded-ios shadow-ios text-sm text-primary hover:bg-bg transition-colors min-h-[44px]"
          >
            <Icon name="save" className="w-4 h-4" />
            {t('pensionImport.download_template')}
          </a>
        </div>
      </div>

      {!preview && (
        <div className="bg-card rounded-ios shadow-ios p-4 sm:p-6 space-y-5">
          <div>
            <label className="block text-sm font-medium mb-1">{t('pensionImport.file')}</label>
            <input
              type="file"
              accept=".pdf,.txt,.csv,application/pdf,text/plain,text/csv"
              onChange={(e) => setFile(e.target.files?.[0] ?? null)}
              className="block w-full text-sm text-text-secondary file:mr-4 file:py-2 file:px-4 file:rounded-ios-sm file:border-0 file:bg-primary/10 file:text-primary min-h-[44px]"
            />
            <p className="text-xs text-text-secondary mt-1">{t('pensionImport.file_hint')}</p>
          </div>

          <div>
            <label className="block text-sm font-medium mb-1">{t('pensionImport.or_paste')}</label>
            <textarea
              value={text}
              onChange={(e) => setText(e.target.value)}
              rows={6}
              placeholder={t('pensionImport.paste_placeholder')}
              className={`${inputCls} resize-y font-mono text-sm`}
            />
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium mb-1">{t('pensionImport.period')}</label>
              <div className="px-3 py-2 bg-bg rounded-ios-sm text-sm">{periodLabel(urlYear, urlMonth, lang)}</div>
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">{t('pensionImport.child')}</label>
              <Select
                options={childOptions}
                value={childId}
                onChange={(v) => setChildId(Number(v))}
                placeholder={t('pensionImport.select_child')}
                searchable
              />
            </div>
          </div>

          <div className="flex justify-end gap-3 pt-4 border-t border-border">
            <button
              type="button"
              onClick={() => navigate('/pension/registros')}
              className="px-4 py-2 bg-bg text-text rounded-ios-sm hover:bg-border transition-colors flex items-center gap-2 min-h-[44px]"
            >
              <Icon name="cancel" className="w-4 h-4" />
              {t('app.cancel')}
            </button>
            <button
              type="button"
              onClick={handlePreview}
              disabled={loading}
              className="px-4 py-2 bg-primary text-white rounded-ios-sm hover:bg-primary-hover disabled:opacity-50 transition-colors flex items-center gap-2 min-h-[44px]"
            >
              <Icon name="search" className="w-4 h-4" />
              {loading ? t('app.loading') : t('pensionImport.preview')}
            </button>
          </div>
        </div>
      )}

      {!preview && batches.length > 0 && (
        <div className="bg-card rounded-ios shadow-ios p-4 sm:p-5">
          <h3 className="text-sm font-semibold mb-3">{t('pensionImport.recent')}</h3>
          <div className="space-y-2">
            {batches.map((b) => (
              <div key={b.id} className="flex flex-wrap items-center justify-between gap-2 text-sm border-b border-border last:border-0 pb-2 last:pb-0">
                <span className="text-text-secondary">
                  {periodLabel(b.period_year, b.period_month, lang)} · {b.filename || t('pensionImport.pasted_text')}
                </span>
                <span className="text-xs text-text-secondary">
                  {b.status === 'confirmed'
                    ? t('pensionImport.batch_confirmed').replace('{created}', String(b.created_count)).replace('{skipped}', String(b.skipped_count))
                    : t('pensionImport.batch_preview')}
                </span>
              </div>
            ))}
          </div>
        </div>
      )}

      {preview && (
        <>
          <div className={`rounded-ios p-4 sm:p-5 border ${canConfirm ? 'bg-card border-emerald-500/25' : 'bg-card border-amber-500/40'}`}>
            <div className="flex items-start gap-3">
              <Icon name={canConfirm ? 'check' : 'warning'} className={`w-5 h-5 mt-0.5 ${canConfirm ? 'text-emerald-600' : 'text-amber-500'}`} />
              <div className="flex-1 min-w-0 text-sm">
                <p className="font-semibold">{canConfirm ? t('pensionImport.summary_ok') : t('pensionImport.summary_invalid')}</p>

                {hasDeclared ? (
                  <p className="mt-1">
                    <span className="text-text-secondary">{t('pensionImport.document_reimbursement')}: </span>
                    <strong>{money(declared)}</strong>{' '}
                    {totalMismatch ? (
                      <span className="text-amber-600 dark:text-amber-400 font-medium">
                        · {t('pensionImport.mismatch')} {money(Math.abs(difference))}
                      </span>
                    ) : (
                      <span className="text-emerald-600 dark:text-emerald-400">· {t('pensionImport.matches')}</span>
                    )}
                  </p>
                ) : (
                  <p className="mt-1 text-amber-600 dark:text-amber-400">{t('pensionImport.no_control_line')}</p>
                )}

                <p className="mt-2 font-medium">
                  {t('pensionImport.will_import')
                    .replace('{count}', String(includedRows.length))
                    .replace('{total}', money(computedGrand))}
                </p>

                <div className="mt-1 text-text-secondary">
                  {period ? periodLabel(period.year, period.month, lang) : ''}
                  {childName ? ` · ${childName.first_name} ${childName.last_name}` : ''}
                </div>

                {preview.period_mismatch && preview.document_period && (
                  <p className="mt-2 text-amber-600 dark:text-amber-400">
                    {t('pensionImport.period_mismatch')
                      .replace('{document}', periodLabel(preview.document_period.year, preview.document_period.month, lang))
                      .replace('{selected}', periodLabel(urlYear, urlMonth, lang))}
                  </p>
                )}

                {preview.validation.errors.some(e => e.code === 'total_mismatch') && !totalMismatch && (
                  <p className="mt-2 text-amber-600 dark:text-amber-400">{t('pensionImport.total_mismatch')}</p>
                )}

                {preview.control_lines.length > 0 && (
                  <details className="mt-2">
                    <summary className="cursor-pointer text-xs text-text-secondary hover:text-text">
                      {t('pensionImport.control_lines')}
                    </summary>
                    <ul className="mt-1 space-y-0.5 text-xs text-text-secondary">
                      {preview.control_lines.map((c) => (
                        <li key={c.line_no} className="font-mono">
                          {c.raw_text} · {money(c.amount)}
                        </li>
                      ))}
                    </ul>
                  </details>
                )}

                <p className="mt-2 text-xs text-text-secondary">{t('pensionImport.summary_hint')}</p>
              </div>
            </div>
          </div>

          <div className="space-y-3">
            {rowsWithAmounts.map(row => {
              const error = row.excluded ? null : rowError(row)
              const confidence = row.confidence || 0
              const level = row.match_reason === 'manual' || confidence >= 0.85 ? 'high' : confidence >= 0.6 ? 'medium' : 'low'
              const chipCls = level === 'high'
                ? 'bg-emerald-600/10 text-emerald-700 border-emerald-600/30 dark:text-emerald-300'
                : level === 'medium'
                  ? 'bg-amber-500/10 text-amber-700 border-amber-500/30 dark:text-amber-300'
                  : 'bg-red-500/10 text-red-600 border-red-500/30 dark:text-red-300'
              const reasonLabel = row.matched_keyword
                ? t('pensionImport.matched_by').replace('{keyword}', row.matched_keyword)
                : row.match_reason === 'name'
                  ? t('pensionImport.matched_name')
                  : row.match_reason === 'manual'
                    ? t('pensionImport.matched_manual')
                    : t('pensionImport.matched_fuzzy')

              return (
                <div key={row.line_no} className={`bg-card rounded-ios shadow-ios p-4 ${row.excluded ? 'opacity-50' : ''} ${error ? 'border border-amber-500/40' : 'border border-transparent'}`}>
                  <div className="flex flex-wrap items-start justify-between gap-2">
                    <div className="min-w-0 flex-1">
                      <p className="font-semibold truncate">{row.concept || row.raw_text}</p>
                      <p className="text-[11px] text-text-secondary mt-0.5">
                        {t('pensionImport.original_line')}: <span className="font-mono">{row.raw_text}</span>
                      </p>
                    </div>
                    <div className="flex items-center gap-2">
                      <span className={`text-[11px] px-2 py-0.5 rounded-full border ${chipCls}`}>
                        {Math.round(confidence * 100)}% · {reasonLabel}
                      </span>
                      <button
                        type="button"
                        onClick={() => toggleExcluded(row.line_no)}
                        className="text-xs text-text-secondary hover:text-text px-2 py-1 rounded-ios-sm hover:bg-bg min-h-[32px]"
                      >
                        {row.excluded ? t('pensionImport.include') : t('pensionImport.exclude')}
                      </button>
                    </div>
                  </div>

                  {!row.excluded && (
                    <>
                      {error && (
                        <p className="text-xs text-amber-600 dark:text-amber-400 mt-2 flex items-center gap-1">
                          <Icon name="warning" className="w-3.5 h-3.5" /> {error}
                        </p>
                      )}
                      {!error && typeof row.previous_amount === 'number' && (
                        <p className="text-xs text-text-secondary mt-0.5">
                          {t('pensionImport.previous')}: {money(row.previous_amount, row.category_currency)}
                          {typeof row.variation_pct === 'number' && Math.abs(row.variation_pct) >= 0.01 && (
                            <span className="ml-1">({row.variation_pct > 0 ? '+' : ''}{row.variation_pct.toFixed(1)}%)</span>
                          )}
                        </p>
                      )}
                      {typeof row.existing_amount === 'number' && (
                        <p className="text-xs text-amber-600 dark:text-amber-400 mt-1">
                          {t('pensionImport.already_exists').replace('{amount}', money(row.existing_amount, row.category_currency))}
                        </p>
                      )}
                    </>
                  )}

                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 mt-3">
                    <div>
                      <label className="block text-xs font-medium mb-1">{t('pensionImport.category')}</label>
                      <Select
                        options={categoryOptions}
                        value={row.chosen_category_id ?? row.suggested_category_id ?? ''}
                        onChange={(v) => {
                          if (v === NEW_CATEGORY) {
                            openCreateCategory(row)
                            return
                          }
                          updateRow(row.line_no, { chosen_category_id: Number(v), suggested_category_id: Number(v) })
                        }}
                        placeholder={t('pensionImport.select_category')}
                        searchable
                      />
                    </div>
                    <div>
                      <label className="block text-xs font-medium mb-1">{t('pensionImport.amount_total')}</label>
                      <input
                        type="number"
                        step="0.01"
                        min="0"
                        value={row.amount_total}
                        onChange={(e) => updateRow(row.line_no, { amount_total: Number(e.target.value) })}
                        className={inputCls}
                        disabled={row.excluded}
                      />
                    </div>
                  </div>

                  <p className="text-sm mt-2">
                    <span className="text-text-secondary">{formatMoney(row.amount_total)} × {row.share_percent}% = </span>
                    <strong>{formatMoney(row.amount_to_record)}</strong>
                    <span className="text-text-secondary"> {row.category_currency}</span>
                    {row.rounding_adjustment !== 0 && (
                      <span className="text-xs text-text-secondary ml-2">
                        ({t('pensionImport.rounding')} {row.rounding_adjustment > 0 ? '+' : ''}{row.rounding_adjustment.toFixed(3)})
                      </span>
                    )}
                  </p>

                  {!row.excluded && !(row.chosen_category_id ?? row.suggested_category_id) && (
                    <button
                      type="button"
                      onClick={() => openCreateCategory(row)}
                      className="mt-3 w-full px-3 py-2 bg-primary/10 text-primary rounded-ios-sm hover:bg-primary/15 transition-colors flex items-center justify-center gap-2 min-h-[44px]"
                    >
                      <Icon name="plus" className="w-4 h-4" />
                      {t('pensionImport.create_category')}
                    </button>
                  )}
                </div>
              )
            })}
          </div>

          <div className="flex flex-wrap items-center justify-between gap-3">
            <button
              type="button"
              onClick={() => dirty ? setShowReset(true) : reset()}
              className="px-4 py-2 text-text-secondary hover:text-text rounded-ios-sm hover:bg-bg transition-colors flex items-center gap-2 min-h-[44px]"
            >
              <Icon name="refresh" className="w-4 h-4" />
              {t('pensionImport.start_over')}
            </button>
            <button
              type="button"
              onClick={handleConfirm}
              disabled={confirming || !canConfirm}
              className="px-4 py-2 bg-primary text-white rounded-ios-sm hover:bg-primary-hover disabled:opacity-50 transition-colors flex items-center gap-2 min-h-[44px]"
            >
              <Icon name="check" className="w-4 h-4" />
              {confirming
                ? t('app.saving')
                : t('pensionImport.confirm_count')
                    .replace('{count}', String(includedRows.length))
                    .replace('{total}', money(computedGrand))}
            </button>
          </div>
          {hasErrors && (
            <p className="text-sm text-amber-600 dark:text-amber-400 text-right">{t('pensionImport.fix_errors')}</p>
          )}
        </>
      )}

      {showReset && (
        <div className="fixed inset-0 bg-black/40 flex items-center justify-center z-[60] p-4">
          <div className="bg-card rounded-ios shadow-ios-lg w-full max-w-sm p-5" onClick={(e) => e.stopPropagation()}>
            <h3 className="text-base font-semibold mb-2">{t('pensionImport.reset_title')}</h3>
            <p className="text-sm text-text-secondary mb-4">{t('pensionImport.reset_message')}</p>
            <div className="flex justify-end gap-3">
              <button type="button" onClick={() => setShowReset(false)} className="px-4 py-2 bg-bg text-text rounded-ios-sm hover:bg-border transition-colors min-h-[44px]">
                {t('app.cancel')}
              </button>
              <button type="button" onClick={reset} className="px-4 py-2 bg-primary text-white rounded-ios-sm hover:bg-primary-hover transition-colors min-h-[44px]">
                {t('pensionImport.start_over')}
              </button>
            </div>
          </div>
        </div>
      )}

      {showKeywords && (
        <div className="fixed inset-0 bg-black/40 flex items-center justify-center z-[60] p-4">
          <div className="bg-card rounded-ios shadow-ios-lg w-full max-w-lg max-h-[90vh] overflow-y-auto" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between px-5 pt-4">
              <h3 className="text-base sm:text-lg font-semibold">{t('pensionImport.keywords')}</h3>
              <button onClick={() => setShowKeywords(false)} className="w-8 h-8 rounded-full flex items-center justify-center hover:bg-bg transition-colors">
                <Icon name="cancel" className="w-4 h-4" />
              </button>
            </div>
            <div className="p-5 space-y-3">
              <p className="text-xs text-text-secondary">{t('pensionImport.keywords_hint')}</p>
              <div className="max-h-56 overflow-y-auto space-y-1">
                {keywords.map((k) => {
                  const category = categories.find(c => c.id === k.pension_category_id)
                  return (
                    <div key={k.id} className="flex items-center justify-between gap-2 text-sm bg-bg rounded-ios-sm px-3 py-2">
                      <span className="font-mono text-xs">{k.keyword}</span>
                      <span className="text-xs text-text-secondary truncate flex-1 text-right">{category?.name ?? '—'}</span>
                      <button onClick={() => handleDeleteKeyword(k.id)} className="w-7 h-7 rounded-full flex items-center justify-center hover:bg-card text-red-500" aria-label={t('app.delete')}>
                        <Icon name="delete" className="w-3.5 h-3.5" />
                      </button>
                    </div>
                  )
                })}
              </div>
              <form onSubmit={handleCreateKeyword} className="space-y-3 pt-3 border-t border-border">
                <div>
                  <label className="block text-xs font-medium mb-1">{t('pensionImport.category')}</label>
                  <Select
                    options={categoryOptions.filter(o => o.value !== NEW_CATEGORY)}
                    value={newKeywordCategory}
                    onChange={(v) => setNewKeywordCategory(Number(v))}
                    placeholder={t('pensionImport.select_category')}
                    searchable
                  />
                </div>
                <div>
                  <label className="block text-xs font-medium mb-1">{t('pensionImport.keyword')}</label>
                  <input value={newKeyword} onChange={(e) => setNewKeyword(e.target.value)} className={inputCls} placeholder={t('pensionImport.keyword_placeholder')} />
                </div>
                <button type="submit" disabled={savingKeyword || !newKeywordCategory || !newKeyword.trim()} className="w-full px-4 py-2 bg-primary text-white rounded-ios-sm hover:bg-primary-hover disabled:opacity-50 transition-colors min-h-[44px]">
                  {savingKeyword ? t('app.saving') : t('app.create')}
                </button>
              </form>
            </div>
          </div>
        </div>
      )}

      {createFor && (
        <div className="fixed inset-0 bg-black/40 flex items-center justify-center z-[60] p-4">
          <div className="bg-card rounded-ios shadow-ios-lg w-full max-w-md max-h-[90vh] overflow-y-auto" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between px-5 pt-4">
              <h3 className="text-base sm:text-lg font-semibold">{t('pensionImport.create_category')}</h3>
              <button onClick={() => setCreateFor(null)} className="w-8 h-8 rounded-full flex items-center justify-center hover:bg-bg transition-colors">
                <Icon name="cancel" className="w-4 h-4" />
              </button>
            </div>
            <form onSubmit={handleCreateCategory} className="p-5 space-y-4">
              <div>
                <label className="block text-sm font-medium mb-1">{t('categorias.name')} *</label>
                <input value={newCategoryName} onChange={(e) => setNewCategoryName(e.target.value)} className={inputCls} required />
              </div>
              <div>
                <label className="block text-sm font-medium mb-1">{t('pensionImport.share')}</label>
                <Select options={shareOptions} value={newCategoryShare} onChange={(v) => setNewCategoryShare(Number(v) as 50 | 100)} />
              </div>
              <div>
                <label className="block text-sm font-medium mb-1">{t('pensionImport.currency')}</label>
                <Select options={currencyOptions} value={newCategoryCurrency} onChange={(v) => setNewCategoryCurrency(String(v))} searchable />
              </div>
              <div className="flex justify-end gap-3 pt-4 border-t border-border">
                <button type="button" onClick={() => setCreateFor(null)} className="px-4 py-2 bg-bg text-text rounded-ios-sm hover:bg-border transition-colors min-h-[44px]">
                  {t('app.cancel')}
                </button>
                <button type="submit" disabled={creatingCategory} className="px-4 py-2 bg-primary text-white rounded-ios-sm hover:bg-primary-hover disabled:opacity-50 transition-colors min-h-[44px]">
                  {creatingCategory ? t('app.saving') : t('app.create')}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}
