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
import type { Child, PensionCategory, PensionCategoryKeyword, PensionImportBatch, PensionImportLine, PensionImportPreview } from '../types'

const inputCls = 'w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px]'

function currentPeriod(): { year: number; month: number } {
  const now = new Date()
  return { year: now.getFullYear(), month: now.getMonth() + 1 }
}

function roundShare(total: number, share: number): number {
  return Math.round(total * share) / 100
}

export default function PensionImportPage() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const { t } = useI18nStore()
  const { showToast } = useToast()
  const formatMoney = useCurrencyFormatStore(s => s.formatMoney)
  usePageTitle(t('pensionImport.title'))

  const initial = currentPeriod()
  const year = Number(searchParams.get('year')) || initial.year
  const month = Number(searchParams.get('month')) || initial.month

  const [children, setChildren] = useState<Child[]>([])
  const [categories, setCategories] = useState<PensionCategory[]>([])
  const [childId, setChildId] = useState<number | ''>('')
  const [file, setFile] = useState<File | null>(null)
  const [text, setText] = useState('')
  const [preview, setPreview] = useState<PensionImportPreview | null>(null)
  const [rows, setRows] = useState<PensionImportLine[]>([])
  const [loading, setLoading] = useState(false)
  const [confirming, setConfirming] = useState(false)
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
    () => categories.map(c => ({ value: c.id, label: `${c.name} · ${c.share_percent}%` })),
    [categories],
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

  const rowsWithAmounts = useMemo(() => rows.map(r => {
    const category = categories.find(c => c.id === (r.chosen_category_id ?? r.suggested_category_id))
    const share = category?.share_percent ?? r.share_percent ?? 50
    return {
      ...r,
      share_percent: share,
      category_currency: category?.currency ?? r.currency ?? 'NIO',
      amount_to_record: roundShare(r.amount_total, share),
      category_name: category?.name ?? r.category_name ?? '',
    }
  }), [rows, categories])

  const rowError = (r: PensionImportLine): string | null => {
    if (!(r.chosen_category_id ?? r.suggested_category_id)) return t('pensionImport.error_missing_category')
    if (!r.amount_total || r.amount_total <= 0) return t('pensionImport.error_invalid_amount')
    const recordCurrency = (r as PensionImportLine & { category_currency?: string }).category_currency
    if (r.currency && recordCurrency && r.currency.toUpperCase() !== recordCurrency.toUpperCase()) {
      return t('pensionImport.error_currency')
    }
    return null
  }
  const hasErrors = rowsWithAmounts.some(r => rowError(r) !== null)
  const computedShared = rowsWithAmounts
    .filter(r => r.share_percent === 50)
    .reduce((sum, r) => sum + r.amount_to_record, 0)
  const computedGrand = rowsWithAmounts.reduce((sum, r) => sum + r.amount_to_record, 0)
  const declared = preview?.validation.declared_reimbursable_total ?? 0
  const totalMismatch = declared > 0 && Math.abs(declared - computedShared) > 0.0001
  const serverBlocking = (preview?.validation.errors ?? []).some(e => e.code !== 'total_mismatch')
  const canConfirm = !hasErrors && !totalMismatch && !serverBlocking

  const handlePreview = async () => {
    if (!file && !text.trim()) {
      showToast(t('pensionImport.select_file_or_text'), 'error')
      return
    }
    setLoading(true)
    try {
      const data = file
        ? await api.pensionImport.previewFile(file, { year, month, child_id: childId || undefined })
        : await api.pensionImport.previewText({ text, year, month, child_id: childId || undefined })
      if (!data) return
      setPreview(data)
      setRows(data.lines)
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
        rows: rowsWithAmounts.map(r => ({
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
              <div className="px-3 py-2 bg-bg rounded-ios-sm text-sm">{String(month).padStart(2, '0')}/{year}</div>
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
                  {String(b.period_month).padStart(2, '0')}/{b.period_year} · {b.filename || t('pensionImport.pasted_text')}
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
          <div className={`rounded-ios shadow-ios p-4 sm:p-5 border ${canConfirm ? 'bg-emerald-600/5 border-emerald-600/30' : 'bg-amber-500/5 border-amber-500/30'}`}>
            <div className="flex items-start gap-3">
              <Icon name={canConfirm ? 'check' : 'warning'} className={`w-5 h-5 mt-0.5 ${canConfirm ? 'text-emerald-600' : 'text-amber-500'}`} />
              <div className="flex-1 text-sm">
                <p className="font-semibold">
                  {canConfirm
                    ? t('pensionImport.summary_ok')
                    : t('pensionImport.summary_invalid')}
                </p>
                <div className="mt-2 grid grid-cols-1 sm:grid-cols-3 gap-2 text-text-secondary">
                  <span>{t('pensionImport.declared')}: <strong className="text-text">{formatMoney(declared)}</strong></span>
                  <span>{t('pensionImport.computed_shared')}: <strong className="text-text">{formatMoney(computedShared)}</strong></span>
                  <span>{t('pensionImport.computed_grand')}: <strong className="text-text">{formatMoney(computedGrand)}</strong></span>
                </div>
                {preview.validation.errors.length > 0 && (
                  <ul className="mt-2 space-y-1 text-amber-700 dark:text-amber-300">
                    {preview.validation.errors.map((e, i) => (
                      <li key={i}>{e.line_no ? `#${e.line_no}: ` : ''}{e.message}</li>
                    ))}
                  </ul>
                )}
                {totalMismatch && (
                  <p className="mt-2 text-amber-700 dark:text-amber-300">{t('pensionImport.total_mismatch')}</p>
                )}
                <p className="mt-2 text-xs text-text-secondary">{t('pensionImport.summary_hint')}</p>
              </div>
            </div>
          </div>

          <div className="space-y-3">
            {rowsWithAmounts.map(row => {
              const error = rowError(row)
              return (
                <div key={row.line_no} className={`bg-card rounded-ios shadow-ios p-4 ${error ? 'border border-amber-500/40' : ''}`}>
                  <div className="flex flex-wrap items-start justify-between gap-3">
                    <div className="min-w-0 flex-1">
                      <p className="font-semibold truncate">{row.concept || row.raw_text}</p>
                      <p className="text-xs text-text-secondary mt-0.5 font-mono truncate">{row.raw_text}</p>
                      {error && (
                        <p className="text-xs text-amber-600 dark:text-amber-400 mt-1 flex items-center gap-1">
                          <Icon name="warning" className="w-3.5 h-3.5" /> {error}
                        </p>
                      )}
                      {row.match_reason && !error && (
                        <p className="text-xs text-text-secondary mt-1">
                          {t('pensionImport.match')}: {row.match_reason} · {Math.round((row.confidence || 0) * 100)}%
                          {typeof row.previous_amount === 'number' && (
                            <span className="ml-2">{t('pensionImport.previous')}: {formatMoney(row.previous_amount)}</span>
                          )}
                          {typeof row.variation_pct === 'number' && Math.abs(row.variation_pct) >= 0.01 && (
                            <span className="ml-2">({row.variation_pct > 0 ? '+' : ''}{row.variation_pct.toFixed(1)}%)</span>
                          )}
                        </p>
                      )}
                    </div>
                    <div className="text-right">
                      <p className="text-xs text-text-secondary">{t('pensionImport.amount_total')}</p>
                      <p className="text-sm font-medium">{formatMoney(row.amount_total)}</p>
                      <p className="text-base font-bold text-primary mt-1">{formatMoney(row.amount_to_record)}</p>
                      <p className="text-xs text-text-secondary">{row.share_percent}% · {(row as any).category_currency || row.currency}</p>
                      {row.rounding_adjustment !== 0 && (
                        <p className="text-[11px] text-text-secondary">{t('pensionImport.rounding')}: {row.rounding_adjustment.toFixed(3)}</p>
                      )}
                    </div>
                  </div>

                  <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 mt-3">
                    <div>
                      <label className="block text-xs font-medium mb-1">{t('pensionImport.category')}</label>
                      <Select
                        options={categoryOptions}
                        value={row.chosen_category_id ?? row.suggested_category_id ?? ''}
                        onChange={(v) => updateRow(row.line_no, {
                          chosen_category_id: Number(v),
                          suggested_category_id: Number(v),
                        })}
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
                      />
                    </div>
                    <div className="flex items-end">
                      <button
                        type="button"
                        onClick={() => openCreateCategory(row)}
                        className="w-full px-3 py-2 bg-bg text-text rounded-ios-sm hover:bg-border transition-colors flex items-center justify-center gap-2 min-h-[44px]"
                      >
                        <Icon name="plus" className="w-4 h-4" />
                        {t('pensionImport.create_category')}
                      </button>
                    </div>
                  </div>
                </div>
              )
            })}
          </div>

          <div className="flex flex-wrap justify-end gap-3">
            <button
              type="button"
              onClick={() => { setPreview(null); setRows([]); setFile(null); setText('') }}
              className="px-4 py-2 bg-bg text-text rounded-ios-sm hover:bg-border transition-colors flex items-center gap-2 min-h-[44px]"
            >
              <Icon name="cancel" className="w-4 h-4" />
              {t('pensionImport.start_over')}
            </button>
            <button
              type="button"
              onClick={handleConfirm}
              disabled={confirming || !canConfirm}
              className="px-4 py-2 bg-primary text-white rounded-ios-sm hover:bg-primary-hover disabled:opacity-50 transition-colors flex items-center gap-2 min-h-[44px]"
            >
              <Icon name="check" className="w-4 h-4" />
              {confirming ? t('app.saving') : t('pensionImport.confirm')}
            </button>
          </div>
          {hasErrors && (
            <p className="text-sm text-amber-600 dark:text-amber-400 text-right">{t('pensionImport.fix_errors')}</p>
          )}
        </>
      )}

      {showKeywords && (
        <div className="fixed inset-0 bg-black/40 flex items-center justify-center z-50 p-4">
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
                    options={categoryOptions}
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
        <div className="fixed inset-0 bg-black/40 flex items-center justify-center z-50 p-4">
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
