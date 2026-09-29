import { useEffect, useState, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAppStore } from '../stores/appStore'
import { useCurrencyFormatStore } from '../stores/currencyFormatStore'
import { useI18nStore } from '../stores/i18nStore'
import { usePageTitle } from '../hooks/usePageTitle'
import { api } from '../api'
import { Icon } from '../components/Icons'
import CreateMenu from '../components/CreateMenu'
import CardMenu from '../components/CardMenu'
import DeleteModal from '../components/DeleteModal'
import LoadingSpinner from '../components/LoadingSpinner'
import IconPickerModal from '../components/IconPickerModal'
import Select, { type SelectOption } from '../components/Select'
import MultiSelect from '../components/MultiSelect'
import { useToast } from '../components/Toast'
import type { BudgetMonthView, BudgetCategoryRow, BudgetMonthGroup, Currency, CategoryGroup, BudgetCategory, RecurringRule, Account, Service } from '../types'

interface CategoryAmount {
  currencyId: number
  symbol: string
  assigned: number
  activity: number
  available: number
}

export default function BudgetPage() {
  const navigate = useNavigate()
  const { t } = useI18nStore()
  const { currencies, loadAll } = useAppStore()
  const formatMoney = useCurrencyFormatStore(s => s.formatMoney)
  const { showToast } = useToast()

  const now = new Date()
  const [year, setYear] = useState(now.getFullYear())
  const [month, setMonth] = useState(now.getMonth() + 1)
  const [view, setView] = useState<BudgetMonthView | null>(null)
  const [loading, setLoading] = useState(true)
  const [configOpen, setConfigOpen] = useState(false)
  const [groupForm, setGroupForm] = useState<{ id?: number; name: string; icon: string } | null>(null)
  const [catForm, setCatForm] = useState<{ id?: number; groupId: number; name: string; icon: string; target?: string; serviceIds?: number[]; accountId?: number | null } | null>(null)
  const [iconPickerFor, setIconPickerFor] = useState<'group' | 'category' | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<{ kind: 'group' | 'category'; id: number; name: string } | null>(null)
  const [services, setServices] = useState<Service[]>([])
  const [accounts, setAccounts] = useState<Account[]>([])

  usePageTitle(t('budget.title'))

  const symbolOf = (id: number) => currencies.find((c) => c.id === id)?.symbol || ''

  const loadView = useCallback(async (y: number, m: number) => {
    setLoading(true)
    try {
      const data = await api.budget.monthView(y, m)
      setView(data)
    } catch (err) {
      showToast((err as Error).message, 'error')
    } finally {
      setLoading(false)
    }
  }, [showToast])

  useEffect(() => {
    loadAll()
  }, [loadAll])

  useEffect(() => {
    loadView(year, month)
  }, [year, month, loadView])

  useEffect(() => {
    api.services.list().then((list) => setServices(list || [])).catch(() => setServices([]))
    api.budget.accounts.list().then((list) => setAccounts(list || [])).catch(() => setAccounts([]))
  }, [])

  const changeMonth = (delta: number) => {
    const d = new Date(year, month - 1 + delta, 1)
    setYear(d.getFullYear())
    setMonth(d.getMonth() + 1)
  }

  const saveGroup = async () => {
    if (!groupForm || !groupForm.name.trim()) return
    try {
      if (groupForm.id) {
        await api.budget.categoryGroups.update(groupForm.id, { name: groupForm.name, icon: groupForm.icon })
      } else {
        await api.budget.categoryGroups.create({ name: groupForm.name, icon: groupForm.icon })
      }
      setGroupForm(null)
      await loadView(year, month)
    } catch (err) {
      showToast((err as Error).message, 'error')
    }
  }

  const saveCategory = async () => {
    if (!catForm || !catForm.name.trim()) return
    try {
      const target = catForm.target ? parseFloat(catForm.target) : null
      const body = {
        category_group_id: catForm.groupId,
        name: catForm.name,
        icon: catForm.icon,
        target_amount: target,
        service_ids: catForm.serviceIds || [],
        account_id: (catForm.serviceIds && catForm.serviceIds.length > 0) ? (catForm.accountId || null) : null,
      }
      if (catForm.id) {
        await api.budget.categories.update(catForm.id, body)
      } else {
        await api.budget.categories.create(body)
      }
      setCatForm(null)
      await loadView(year, month)
    } catch (err) {
      showToast((err as Error).message, 'error')
    }
  }

  const handleDelete = async () => {
    if (!deleteTarget) return
    try {
      if (deleteTarget.kind === 'group') {
        await api.budget.categoryGroups.delete(deleteTarget.id)
      } else {
        await api.budget.categories.delete(deleteTarget.id)
      }
      setDeleteTarget(null)
      await loadView(year, month)
    } catch (err) {
      showToast((err as Error).message, 'error')
    }
  }

  if (loading && !view) return <LoadingSpinner />

  if (currencies.length === 0) {
    return (
      <div className="bg-card rounded-ios shadow-ios p-8 sm:p-12 text-center max-w-md mx-auto mt-8">
        <div className="w-16 h-16 mx-auto mb-5 text-primary opacity-80">
          <Icon name="savings" className="w-full h-full" />
        </div>
        <h3 className="text-lg sm:text-xl font-semibold mb-2">{t('budget.no_currencies')}</h3>
        <button
          onClick={() => navigate('/settings/currency')}
          className="inline-flex items-center justify-center gap-2 px-6 py-3 bg-card border-2 border-dashed border-border rounded-ios text-primary font-semibold hover:border-primary hover:bg-primary/5 transition-colors"
        >
          <Icon name="plus" className="w-5 h-5" />
          {t('budget.no_currencies_cta')}
        </button>
      </div>
    )
  }

  const monthName = new Date(year, month - 1, 1).toLocaleString('es', { month: 'long', year: 'numeric' })

  const createOptions = [
    { label: t('budget.configure'), icon: 'tag', onClick: () => setConfigOpen(true) },
    { label: t('budget.transactions'), icon: 'tag', onClick: () => navigate('/budget/transacciones') },
    { label: t('budget.accounts'), icon: 'bank', onClick: () => navigate('/budget/transacciones') },
  ]

  return (
    <div>
      <div className="flex items-center justify-between mb-4 sm:mb-5">
        <h2 className="text-xl sm:text-2xl font-bold">{t('budget.title')}</h2>
        <CreateMenu options={createOptions} />
      </div>

      {/* Selector de mes */}
      <div className="flex items-center justify-between bg-card rounded-ios shadow-ios p-3 mb-4">
        <button
          onClick={() => changeMonth(-1)}
          className="w-10 h-10 rounded-full flex items-center justify-center hover:bg-bg transition-colors min-h-[44px]"
          title={t('budget.previous_month')}
        >
          <Icon name="chevron" className="w-5 h-5 rotate-180" />
        </button>
        <span className="font-semibold capitalize">{monthName}</span>
        <button
          onClick={() => changeMonth(1)}
          className="w-10 h-10 rounded-full flex items-center justify-center hover:bg-bg transition-colors min-h-[44px]"
          title={t('budget.next_month')}
        >
          <Icon name="chevron" className="w-5 h-5" />
        </button>
      </div>

      {/* Totales por moneda */}
      {view && (view.currency_totals || []).length > 0 && (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3 mb-4">
          {(view.currency_totals || []).map((total) => (
            <div key={total.currency_id} className="bg-card rounded-ios shadow-ios p-4">
              <div className="text-xs uppercase tracking-wide text-text-secondary font-semibold mb-1">
                {total.code}
              </div>
              <div className="flex items-baseline justify-between">
                <span className="text-sm text-text-secondary">{t('budget.unassigned')}</span>
                <span className={`font-semibold ${total.unassigned < 0 ? 'text-red-500' : 'text-text'}`}>
                  {formatMoney(total.unassigned, total.symbol)}
                </span>
              </div>
              <div className="flex items-baseline justify-between text-sm mt-1">
                <span className="text-text-secondary">{t('budget.income')}</span>
                <span>{formatMoney(total.total_income, total.symbol)}</span>
              </div>
            </div>
          ))}
        </div>
      )}

      <p className="text-xs text-text-secondary mb-3 flex items-center gap-1">
        <Icon name="info" className="w-3.5 h-3.5" />
        {t('budget.reset_note')}
      </p>

      {!view || (view.groups || []).length === 0 ? (
        <div className="bg-card rounded-ios shadow-ios p-8 sm:p-12 text-center max-w-md mx-auto">
          <div className="w-16 h-16 mx-auto mb-5 text-primary opacity-80">
            <Icon name="wallet" className="w-full h-full" />
          </div>
          <h3 className="text-lg sm:text-xl font-semibold mb-2">{t('budget.empty')}</h3>
          <p className="text-text-secondary mb-6">{t('budget.empty_subtitle')}</p>
          <button
            onClick={() => setGroupForm({ name: '', icon: 'tag' })}
            className="inline-flex items-center justify-center gap-2 px-6 py-3 bg-card border-2 border-dashed border-border rounded-ios text-primary font-semibold hover:border-primary hover:bg-primary/5 transition-colors"
          >
            <Icon name="plus" className="w-5 h-5" />
            {t('budget.create_group')}
          </button>
        </div>
      ) : (
        <div className="space-y-4">
          {(view.groups || []).map((group) => (
            <BudgetGroupTable
              key={group.id}
              group={group}
              currencies={currencies}
              formatMoney={formatMoney}
              symbolOf={symbolOf}
              year={year}
              month={month}
              services={services}
              accounts={accounts}
              onAssigned={() => loadView(year, month)}
              onAddCategory={(groupId) => setCatForm({ groupId, name: '', icon: 'tag', target: '', serviceIds: [], accountId: null })}
              onEditGroup={(g) => setGroupForm({ id: g.id, name: g.name, icon: g.icon })}
              onDeleteGroup={(g) => setDeleteTarget({ kind: 'group', id: g.id, name: g.name })}
              onEditCategory={(cat) => setCatForm({ id: cat.id, groupId: cat.category_group_id, name: cat.name, icon: cat.icon, target: cat.target_amount != null ? String(cat.target_amount) : '', serviceIds: cat.service_ids || [], accountId: cat.account_id || null })}
              onDeleteCategory={(cat) => setDeleteTarget({ kind: 'category', id: cat.id, name: cat.name })}
            />
          ))}
          <button
            onClick={() => setGroupForm({ name: '', icon: 'tag' })}
            className="w-full flex items-center justify-center gap-2 px-4 py-3 bg-card border-2 border-dashed border-border rounded-ios text-primary font-semibold hover:border-primary hover:bg-primary/5 transition-colors min-h-[44px]"
          >
            <Icon name="plus" className="w-5 h-5" />
            {t('budget.create_group')}
          </button>
        </div>
      )}

      {groupForm && (
        <GroupFormModal
          form={groupForm}
          onChange={setGroupForm}
          onIconPick={() => setIconPickerFor('group')}
          onSave={saveGroup}
          onClose={() => setGroupForm(null)}
        />
      )}

      {catForm && (
        <CategoryFormModal
          form={catForm}
          onChange={setCatForm}
          onIconPick={() => setIconPickerFor('category')}
          onSave={saveCategory}
          onClose={() => setCatForm(null)}
          services={services}
          accounts={accounts}
        />
      )}

      {iconPickerFor && (
        <IconPickerModal
          isOpen
          zIndex="z-[70]"
          selectedIcon={iconPickerFor === 'group' ? groupForm?.icon || 'tag' : catForm?.icon || 'tag'}
          onSelect={(key) => {
            if (iconPickerFor === 'group' && groupForm) setGroupForm({ ...groupForm, icon: key })
            if (iconPickerFor === 'category' && catForm) setCatForm({ ...catForm, icon: key })
            setIconPickerFor(null)
          }}
          onClose={() => setIconPickerFor(null)}
        />
      )}

      {deleteTarget && (
        <DeleteModal
          title={deleteTarget.kind === 'group' ? t('budget.delete_group') : t('budget.delete_category')}
          subtitle={`${deleteTarget.name}${deleteTarget.kind === 'category' ? ` — ${t('budget.archive_category')}` : ''}`}
          onConfirm={handleDelete}
          onCancel={() => setDeleteTarget(null)}
        />
      )}

      {configOpen && (
        <BudgetConfigModal
          onClose={() => setConfigOpen(false)}
          onChanged={() => loadView(year, month)}
          services={services}
          accounts={accounts}
        />
      )}
    </div>
  )
}

function categoryAmounts(row: BudgetCategoryRow, symbolOf: (id: number) => string): CategoryAmount[] {
  const ids = new Set<number>()
  Object.keys(row.assigned || {}).forEach((k) => ids.add(Number(k)))
  Object.keys(row.activity || {}).forEach((k) => ids.add(Number(k)))
  return Array.from(ids).sort((a, b) => a - b).map((id) => ({
    currencyId: id,
    symbol: symbolOf(id),
    assigned: row.assigned?.[id] || 0,
    activity: row.activity?.[id] || 0,
    available: row.available?.[id] ?? (row.assigned?.[id] || 0) - (row.activity?.[id] || 0),
  }))
}

function BudgetGroupTable({ group, currencies, formatMoney, symbolOf, year, month, services, accounts, onAssigned, onAddCategory, onEditGroup, onDeleteGroup, onEditCategory, onDeleteCategory }: {
  group: BudgetMonthGroup
  currencies: Currency[]
  formatMoney: (n: number, s?: string) => string
  symbolOf: (id: number) => string
  year: number
  month: number
  services: Service[]
  accounts: Account[]
  onAssigned: () => void
  onAddCategory: (groupId: number) => void
  onEditGroup: (g: BudgetMonthGroup) => void
  onDeleteGroup: (g: BudgetMonthGroup) => void
  onEditCategory: (cat: BudgetCategoryRow) => void
  onDeleteCategory: (cat: BudgetCategoryRow) => void
}) {
  const { t } = useI18nStore()
  const [assignTarget, setAssignTarget] = useState<BudgetCategoryRow | null>(null)

  return (
    <div className="bg-card rounded-ios shadow-ios overflow-hidden">
      <div className="relative flex items-center gap-2 px-4 py-3 border-b border-border">
        <Icon name={group.icon || 'tag'} className="w-4 h-4 text-primary" />
        <h3 className="font-semibold flex-1">{group.name}</h3>
        <CardMenu
          options={[
            { label: t('budget.create_category'), icon: 'plus', onClick: () => onAddCategory(group.id) },
            { label: t('budget.edit_group'), icon: 'edit', onClick: () => onEditGroup(group) },
            { label: t('budget.delete_group'), icon: 'delete', danger: true, onClick: () => onDeleteGroup(group) },
          ]}
        />
      </div>

      {/* Cabecera de columnas */}
      <div className="hidden sm:grid grid-cols-4 gap-2 px-4 py-2 text-xs uppercase tracking-wide text-text-secondary font-semibold border-b border-border">
        <span>{t('budget.category')}</span>
        <span className="text-right">{t('budget.assigned')}</span>
        <span className="text-right">{t('budget.activity')}</span>
        <span className="text-right">{t('budget.available')}</span>
      </div>

      {group.categories.length === 0 ? (
        <p className="px-4 py-4 text-sm text-text-secondary">{t('budget.category_empty', '')}</p>
      ) : (
        <div>
          {group.categories.map((cat) => {
            const amounts = categoryAmounts(cat, symbolOf)
            const main = amounts[0]
            return (
              <div
                key={cat.id}
                className="relative flex flex-col sm:grid sm:grid-cols-4 gap-1 sm:gap-2 items-start sm:items-center px-4 py-3 border-b border-border last:border-b-0 hover:bg-bg transition-colors"
              >
                <button
                  onClick={() => setAssignTarget(cat)}
                  className="contents"
                >
                  <span className="flex items-center gap-2 text-sm font-medium min-w-0 sm:pr-8">
                    <Icon name={cat.icon || 'tag'} className="w-4 h-4 text-text-secondary flex-shrink-0" />
                    <span className="truncate">{cat.name}</span>
                    {(cat.service_names && cat.service_names.length > 0) && (
                      <span className="inline-flex items-center gap-1 text-[10px] px-1.5 py-0.5 rounded-full bg-primary/10 text-primary whitespace-nowrap min-w-0">
                        <Icon name="services" className="w-3 h-3 flex-shrink-0" />
                        <span className="truncate">{cat.service_names.length === 1 ? cat.service_names[0] : `${cat.service_names.length} ${t('budget.n_services')}`}</span>
                      </span>
                    )}
                    {cat.recurring_rule && (
                      <Icon
                        name="refresh"
                        className={`w-3.5 h-3.5 ${cat.recurring_rule.active ? 'text-primary' : 'text-text-secondary'}`}
                      />
                    )}
                    {main && main.assigned > 0 && main.activity === 0 && (
                      <span className="text-[10px] px-1.5 py-0.5 rounded-full bg-bg text-text-secondary">
                        {t('budget.unused')}
                      </span>
                    )}
                  </span>
                  {amounts.length === 0 ? (
                    <>
                      <span className="sm:text-right text-text-secondary text-sm sm:hidden">
                        {t('budget.assign')}
                      </span>
                      <span className="hidden sm:block sm:text-right text-text-secondary text-sm" />
                      <span className="hidden sm:block sm:text-right text-text-secondary text-sm" />
                      <span className="hidden sm:block sm:text-right text-text-secondary text-sm">—</span>
                    </>
                  ) : amounts.length === 1 ? (
                    <>
                      <span className="sm:text-right text-sm sm:col-span-1">
                        {formatMoney(main.assigned, main.symbol)}
                      </span>
                      <span className="sm:text-right text-sm sm:col-span-1">
                        {formatMoney(main.activity, main.symbol)}
                      </span>
                      <span className={`sm:text-right text-sm font-medium ${main.available < 0 ? 'text-red-500' : ''}`}>
                        {formatMoney(main.available, main.symbol)}
                      </span>
                    </>
                  ) : (
                    <div className="sm:col-span-3 flex flex-col gap-0.5 text-right text-xs text-text-secondary">
                      {amounts.map((a) => (
                        <span key={a.currencyId}>
                          {a.symbol}{formatMoney(a.assigned, '')} / {formatMoney(a.activity, '')} / {formatMoney(a.available, '')}
                        </span>
                      ))}
                    </div>
                  )}
                </button>
                <CardMenu
                  options={[
                    { label: t('budget.assign'), icon: 'edit', onClick: () => setAssignTarget(cat) },
                    { label: t('budget.edit_category'), icon: 'edit', onClick: () => onEditCategory(cat) },
                    { label: t('budget.delete_category'), icon: 'delete', danger: true, onClick: () => onDeleteCategory(cat) },
                  ]}
                />
              </div>
            )
          })}
        </div>
      )}

      {assignTarget && (
        <AssignModal
          category={assignTarget}
          year={year}
          month={month}
          currencies={currencies}
          formatMoney={formatMoney}
          onClose={() => setAssignTarget(null)}
          onSaved={() => { setAssignTarget(null); onAssigned() }}
          onEdit={() => { setAssignTarget(null); onEditCategory(assignTarget) }}
          onDelete={() => { setAssignTarget(null); onDeleteCategory(assignTarget) }}
        />
      )}
    </div>
  )
}

function AssignModal({ category, year, month, currencies, formatMoney, onClose, onSaved, onEdit, onDelete }: {
  category: BudgetCategoryRow
  year: number
  month: number
  currencies: Currency[]
  formatMoney: (n: number, s?: string) => string
  onClose: () => void
  onSaved: () => void
  onEdit?: () => void
  onDelete?: () => void
}) {
  const { t } = useI18nStore()
  const { showToast } = useToast()
  const [amount, setAmount] = useState('')
  const [currencyId, setCurrencyId] = useState(currencies[0]?.id || 0)
  const [makeRecurring, setMakeRecurring] = useState(false)
  const [saving, setSaving] = useState(false)
  const [history, setHistory] = useState<Array<{ id: number; payee: string; date: string; outflow: number; inflow: number; currency_code?: string }>>([])
  const [rule, setRule] = useState<RecurringRule | null>(category.recurring_rule || null)

  useEffect(() => {
    api.budget.transactions.listByCategory(category.id, year, month)
      .then((list) => setHistory(list || []))
      .catch(() => setHistory([]))
  }, [category.id, year, month])

  const handleSave = async () => {
    const value = parseFloat(amount)
    if (isNaN(value) || value <= 0) {
      showToast(t('budget.assign_required', 'Ingresá un monto válido'), 'error')
      return
    }
    setSaving(true)
    try {
      await api.budget.assign(year, month, category.id, {
        amount: value,
        currency_id: currencyId,
        make_recurring: makeRecurring,
      })
      onSaved()
    } catch (err) {
      showToast((err as Error).message, 'error')
    } finally {
      setSaving(false)
    }
  }

  const toggleRule = async () => {
    if (!rule) return
    setSaving(true)
    try {
      const updated = await api.budget.toggleRecurring(rule.id, !rule.active)
      if (updated) {
        setRule(updated)
        showToast(updated.active ? t('budget.resume') : t('budget.pause'))
      }
    } catch (err) {
      showToast((err as Error).message, 'error')
    } finally {
      setSaving(false)
    }
  }

  const deleteRule = async () => {
    if (!rule) return
    setSaving(true)
    try {
      await api.budget.deleteRecurringRule(rule.id)
      setRule(null)
      showToast(t('budget.recurring_deleted', 'Regla recurrente eliminada'))
    } catch (err) {
      showToast((err as Error).message, 'error')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
      <div className="bg-card rounded-ios shadow-ios w-full max-w-sm max-h-[90vh] overflow-y-auto">
        <div className="flex items-center justify-between px-4 py-3 border-b border-border">
          <h3 className="font-bold truncate">{category.name}</h3>
          <div className="flex items-center gap-1">
            {onEdit && (
              <button onClick={onEdit} className="w-9 h-9 rounded-full flex items-center justify-center hover:bg-bg transition-colors" title={t('budget.edit_category')}>
                <Icon name="edit" className="w-5 h-5" />
              </button>
            )}
            {onDelete && (
              <button onClick={onDelete} className="w-9 h-9 rounded-full flex items-center justify-center hover:bg-bg transition-colors" title={t('budget.delete_category')}>
                <Icon name="delete" className="w-5 h-5" />
              </button>
            )}
            <button onClick={onClose} className="w-9 h-9 rounded-full flex items-center justify-center hover:bg-bg transition-colors" title={t('app.close')}>
              <Icon name="cancel" className="w-5 h-5" />
            </button>
          </div>
        </div>

        <div className="p-4 space-y-4">
          <div>
            <label className="block text-sm font-medium mb-1">{t('budget.assign')}</label>
            <input
              type="number"
              step="0.01"
              min="0"
              value={amount}
              onChange={(e) => setAmount(e.target.value)}
              placeholder="0.00"
              className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px]"
            />
          </div>

          <div>
            <label className="block text-sm font-medium mb-1">{t('budget.currency')}</label>
            <Select
              value={currencyId}
              onChange={(v) => setCurrencyId(Number(v))}
              options={currencies.map((c) => ({ value: c.id, label: `${c.code} (${c.symbol})` }))}
              zIndex="z-[80]"
            />
          </div>

          <label className="flex items-center gap-2 text-sm cursor-pointer">
            <input
              type="checkbox"
              checked={makeRecurring}
              onChange={(e) => setMakeRecurring(e.target.checked)}
              className="w-4 h-4"
            />
            {t('budget.make_recurring')}
          </label>

          {rule && (
            <div className="bg-bg rounded-ios-sm p-3 space-y-2">
              <div className="flex items-center justify-between text-sm">
                <span className="flex items-center gap-1.5">
                  <Icon name="refresh" className="w-3.5 h-3.5 text-primary" />
                  {t('budget.recurring')}
                </span>
                <span className={rule.active ? 'text-primary font-medium' : 'text-text-secondary'}>
                  {rule.active ? t('budget.recurring') : t('budget.recurring_paused')}
                </span>
              </div>
              <div className="text-sm text-text-secondary">
                {formatMoney(rule.amount, currencies.find((c) => c.id === currencyId)?.symbol)} · {rule.start_month}
              </div>
              <div className="flex gap-2">
                <button
                  onClick={toggleRule}
                  disabled={saving}
                  className="flex-1 px-3 py-2 rounded-ios-sm bg-card border border-border text-sm hover:bg-bg transition-colors min-h-[44px]"
                >
                  {rule.active ? t('budget.pause') : t('budget.resume')}
                </button>
                <button
                  onClick={deleteRule}
                  disabled={saving}
                  className="px-3 py-2 rounded-ios-sm bg-card border border-red-500/30 text-red-500 text-sm hover:bg-red-500/10 transition-colors min-h-[44px]"
                >
                  <Icon name="delete" className="w-4 h-4" />
                </button>
              </div>
            </div>
          )}

          {history.length > 0 && (
            <div>
              <div className="text-sm font-medium mb-1">{t('budget.activity')}</div>
              <div className="space-y-1">
                {history.map((tx) => (
                  <div key={tx.id} className="flex items-center justify-between text-sm text-text-secondary">
                    <span className="truncate">{tx.payee || tx.date}</span>
                    <span className={tx.outflow > 0 ? 'text-red-500' : 'text-primary'}>
                      {tx.outflow > 0 ? `-${formatMoney(tx.outflow, tx.currency_code)}` : `+${formatMoney(tx.inflow, tx.currency_code)}`}
                    </span>
                  </div>
                ))}
              </div>
            </div>
          )}

          <button
            onClick={handleSave}
            disabled={saving}
            className="w-full px-4 py-3 bg-primary text-white rounded-ios-sm font-semibold hover:opacity-90 transition-opacity min-h-[44px]"
          >
            {saving ? t('app.saving') : t('budget.assign')}
          </button>
        </div>
      </div>
    </div>
  )
}

function BudgetConfigModal({ onClose, onChanged, services, accounts }: { onClose: () => void; onChanged: () => void; services: Service[]; accounts: Account[] }) {
  const { t } = useI18nStore()
  const { showToast } = useToast()
  const [groups, setGroups] = useState<CategoryGroup[]>([])
  const [loading, setLoading] = useState(true)
  const [groupForm, setGroupForm] = useState<{ id?: number; name: string; icon: string } | null>(null)
  const [catForm, setCatForm] = useState<{ id?: number; groupId: number; name: string; icon: string; target?: string; serviceIds?: number[]; accountId?: number | null } | null>(null)
  const [iconPickerFor, setIconPickerFor] = useState<'group' | 'category' | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<{ kind: 'group' | 'category'; id: number; name: string } | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const list = await api.budget.categoryGroups.list()
      setGroups(list || [])
    } catch (err) {
      showToast((err as Error).message, 'error')
    } finally {
      setLoading(false)
    }
  }, [showToast])

  useEffect(() => {
    load()
  }, [load])

  const saveGroup = async () => {
    if (!groupForm || !groupForm.name.trim()) return
    try {
      if (groupForm.id) {
        await api.budget.categoryGroups.update(groupForm.id, { name: groupForm.name, icon: groupForm.icon })
      } else {
        await api.budget.categoryGroups.create({ name: groupForm.name, icon: groupForm.icon })
      }
      setGroupForm(null)
      await load()
      onChanged()
    } catch (err) {
      showToast((err as Error).message, 'error')
    }
  }

  const saveCategory = async () => {
    if (!catForm || !catForm.name.trim()) return
    try {
      const target = catForm.target ? parseFloat(catForm.target) : null
      const body = {
        category_group_id: catForm.groupId,
        name: catForm.name,
        icon: catForm.icon,
        target_amount: target,
        service_ids: catForm.serviceIds || [],
        account_id: (catForm.serviceIds && catForm.serviceIds.length > 0) ? (catForm.accountId || null) : null,
      }
      if (catForm.id) {
        await api.budget.categories.update(catForm.id, body)
      } else {
        await api.budget.categories.create(body)
      }
      setCatForm(null)
      await load()
      onChanged()
    } catch (err) {
      showToast((err as Error).message, 'error')
    }
  }

  const handleDelete = async () => {
    if (!deleteTarget) return
    try {
      if (deleteTarget.kind === 'group') {
        await api.budget.categoryGroups.delete(deleteTarget.id)
      } else {
        await api.budget.categories.delete(deleteTarget.id)
      }
      setDeleteTarget(null)
      await load()
      onChanged()
    } catch (err) {
      showToast((err as Error).message, 'error')
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
      <div className="bg-card rounded-ios shadow-ios w-full max-w-lg max-h-[90vh] flex flex-col">
        <div className="flex items-center justify-between px-4 py-3 border-b border-border">
          <h3 className="font-bold">{t('budget.groups')}</h3>
          <button onClick={onClose} className="w-9 h-9 rounded-full flex items-center justify-center hover:bg-bg transition-colors" title={t('app.close')}>
            <Icon name="cancel" className="w-5 h-5" />
          </button>
        </div>

        <div className="flex-1 overflow-y-auto p-4 space-y-4">
          {loading ? <LoadingSpinner /> : groups.length === 0 ? (
            <p className="text-center text-text-secondary text-sm py-6">{t('budget.empty')}</p>
          ) : (
            groups.map((group) => (
              <div key={group.id} className="border border-border rounded-ios-sm overflow-hidden">
                <div className="flex items-center justify-between px-3 py-2 bg-bg">
                  <span className="flex items-center gap-2 font-medium text-sm">
                    <Icon name={group.icon || 'tag'} className="w-4 h-4 text-primary" />
                    {group.name}
                  </span>
                  <div className="flex items-center gap-1">
                    <button
                      onClick={() => setGroupForm({ id: group.id, name: group.name, icon: group.icon })}
                      className="w-8 h-8 rounded-full flex items-center justify-center hover:bg-border transition-colors"
                      title={t('app.edit')}
                    >
                      <Icon name="edit" className="w-4 h-4" />
                    </button>
                    <button
                      onClick={() => setDeleteTarget({ kind: 'group', id: group.id, name: group.name })}
                      className="w-8 h-8 rounded-full flex items-center justify-center hover:bg-border transition-colors"
                      title={t('app.delete')}
                    >
                      <Icon name="delete" className="w-4 h-4" />
                    </button>
                  </div>
                </div>
                <div className="divide-y divide-border">
                  {group.categories.map((cat) => (
                    <div key={cat.id} className="flex items-center justify-between px-3 py-2">
                      <span className="flex items-center gap-2 text-sm">
                        <Icon name={cat.icon || 'tag'} className="w-4 h-4 text-text-secondary" />
                        {cat.name}
                      </span>
                      <div className="flex items-center gap-1">
                        <button
                          onClick={() => setCatForm({ id: cat.id, groupId: group.id, name: cat.name, icon: cat.icon, target: cat.target_amount != null ? String(cat.target_amount) : '', serviceIds: cat.service_ids || [], accountId: cat.account_id || null })}
                          className="w-8 h-8 rounded-full flex items-center justify-center hover:bg-border transition-colors"
                          title={t('app.edit')}
                        >
                          <Icon name="edit" className="w-4 h-4" />
                        </button>
                        <button
                          onClick={() => setDeleteTarget({ kind: 'category', id: cat.id, name: cat.name })}
                          className="w-8 h-8 rounded-full flex items-center justify-center hover:bg-border transition-colors"
                          title={t('app.delete')}
                        >
                          <Icon name="delete" className="w-4 h-4" />
                        </button>
                      </div>
                    </div>
                  ))}
                  <button
                    onClick={() => setCatForm({ groupId: group.id, name: '', icon: 'tag', target: '', serviceIds: [], accountId: null })}
                    className="w-full flex items-center gap-2 px-3 py-2 text-sm text-primary hover:bg-bg transition-colors"
                  >
                    <Icon name="plus" className="w-4 h-4" />
                    {t('budget.create_category')}
                  </button>
                </div>
              </div>
            ))
          )}

          <button
            onClick={() => setGroupForm({ name: '', icon: 'tag' })}
            className="w-full flex items-center justify-center gap-2 px-4 py-3 bg-card border-2 border-dashed border-border rounded-ios text-primary font-semibold hover:border-primary hover:bg-primary/5 transition-colors min-h-[44px]"
          >
            <Icon name="plus" className="w-5 h-5" />
            {t('budget.create_group')}
          </button>
        </div>
      </div>

      {groupForm && (
        <GroupFormModal
          form={groupForm}
          onChange={setGroupForm}
          onIconPick={() => setIconPickerFor('group')}
          onSave={saveGroup}
          onClose={() => setGroupForm(null)}
        />
      )}

      {catForm && (
        <CategoryFormModal
          form={catForm}
          onChange={setCatForm}
          onIconPick={() => setIconPickerFor('category')}
          onSave={saveCategory}
          onClose={() => setCatForm(null)}
          services={services}
          accounts={accounts}
        />
      )}

      {iconPickerFor && (
        <IconPickerModal
          isOpen
          zIndex="z-[70]"
          selectedIcon={iconPickerFor === 'group' ? groupForm?.icon || 'tag' : catForm?.icon || 'tag'}
          onSelect={(key) => {
            if (iconPickerFor === 'group' && groupForm) setGroupForm({ ...groupForm, icon: key })
            if (iconPickerFor === 'category' && catForm) setCatForm({ ...catForm, icon: key })
            setIconPickerFor(null)
          }}
          onClose={() => setIconPickerFor(null)}
        />
      )}

      {deleteTarget && (
        <DeleteModal
          title={deleteTarget.kind === 'group' ? t('budget.delete_group') : t('budget.delete_category')}
          subtitle={`${deleteTarget.name}${deleteTarget.kind === 'category' ? ` — ${t('budget.archive_category')}` : ''}`}
          onConfirm={handleDelete}
          onCancel={() => setDeleteTarget(null)}
        />
      )}
    </div>
  )
}

function GroupFormModal({ form, onChange, onIconPick, onSave, onClose }: {
  form: { id?: number; name: string; icon: string }
  onChange: (f: { id?: number; name: string; icon: string }) => void
  onIconPick: () => void
  onSave: () => void
  onClose: () => void
}) {
  const { t } = useI18nStore()
  return (
    <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/40 p-4">
      <div className="bg-card rounded-ios shadow-ios w-full max-w-sm p-4 space-y-4">
        <h3 className="font-bold">{form.id ? t('budget.edit_group') : t('budget.create_group')}</h3>
        <div>
          <label className="block text-sm font-medium mb-1">{t('budget.group_name')}</label>
          <input
            type="text"
            value={form.name}
            onChange={(e) => onChange({ ...form, name: e.target.value })}
            placeholder={t('budget.group_name')}
            className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px]"
          />
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">{t('budget.icon')}</label>
          <button
            type="button"
            onClick={onIconPick}
            className="w-12 h-12 rounded-ios-sm border-2 border-border text-text-secondary flex items-center justify-center hover:border-primary/50 transition-colors"
          >
            <Icon name={form.icon || 'tag'} className="w-5 h-5" />
          </button>
        </div>
        <div className="flex gap-2">
          <button onClick={onClose} className="flex-1 px-4 py-3 bg-card border border-border rounded-ios-sm text-sm hover:bg-bg transition-colors min-h-[44px]">
            {t('app.cancel')}
          </button>
          <button onClick={onSave} className="flex-1 px-4 py-3 bg-primary text-white rounded-ios-sm font-semibold hover:opacity-90 transition-opacity min-h-[44px]">
            {t('app.save')}
          </button>
        </div>
      </div>
    </div>
  )
}

function CategoryFormModal({ form, onChange, onIconPick, onSave, onClose, services, accounts }: {
  form: { id?: number; groupId: number; name: string; icon: string; target?: string; serviceIds?: number[]; accountId?: number | null }
  onChange: (f: { id?: number; groupId: number; name: string; icon: string; target?: string; serviceIds?: number[]; accountId?: number | null }) => void
  onIconPick: () => void
  onSave: () => void
  onClose: () => void
  services: Service[]
  accounts: Account[]
}) {
  const { t } = useI18nStore()
  const serviceOptions: SelectOption[] = services.map((s) => ({ value: s.id, label: s.name }))
  const accountOptions = accounts.map((a) => ({ value: a.id, label: `${a.name} (${a.currency_code || ''})` }))
  const linkedServices = form.serviceIds || []
  return (
    <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/40 p-4">
      <div className="bg-card rounded-ios shadow-ios w-full max-w-sm p-4 space-y-4 max-h-[90vh] overflow-y-auto">
        <h3 className="font-bold">{form.id ? t('budget.edit_category') : t('budget.create_category')}</h3>
        <div>
          <label className="block text-sm font-medium mb-1">{t('budget.category_name')}</label>
          <input
            type="text"
            value={form.name}
            onChange={(e) => onChange({ ...form, name: e.target.value })}
            placeholder={t('budget.category_name')}
            className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px] bg-card text-text"
          />
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">{t('budget.target_amount')}</label>
          <input
            type="number"
            step="0.01"
            min="0"
            value={form.target ?? ''}
            onChange={(e) => onChange({ ...form, target: e.target.value })}
            placeholder="0.00"
            className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px] bg-card text-text"
          />
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">{t('budget.link_service')}</label>
          <MultiSelect
            value={linkedServices}
            onChange={(vals) => {
              const ids = vals.map(Number)
              onChange({ ...form, serviceIds: ids, accountId: ids.length > 0 ? form.accountId : null })
            }}
            options={serviceOptions}
            placeholder={t('budget.no_service')}
            searchable={services.length > 8}
            zIndex="z-[80]"
          />
          <p className="text-xs text-text-secondary mt-1">{t('budget.link_service_hint')}</p>
        </div>
        {linkedServices.length > 0 ? (
          <div>
            <label className="block text-sm font-medium mb-1">{t('budget.link_account')}</label>
            <Select
              value={form.accountId || 0}
              onChange={(v) => onChange({ ...form, accountId: Number(v) || null })}
              options={accountOptions}
              placeholder={t('budget.link_account_required')}
              searchable={accounts.length > 8}
              zIndex="z-[80]"
            />
          </div>
        ) : null}
        <div>
          <label className="block text-sm font-medium mb-1">{t('budget.icon')}</label>
          <button
            type="button"
            onClick={onIconPick}
            className="w-12 h-12 rounded-ios-sm border-2 border-border text-text-secondary flex items-center justify-center hover:border-primary/50 transition-colors"
          >
            <Icon name={form.icon || 'tag'} className="w-5 h-5" />
          </button>
        </div>
        <div className="flex gap-2">
          <button onClick={onClose} className="flex-1 px-4 py-3 bg-card border border-border rounded-ios-sm text-sm hover:bg-bg transition-colors min-h-[44px]">
            {t('app.cancel')}
          </button>
          <button onClick={onSave} className="flex-1 px-4 py-3 bg-primary text-white rounded-ios-sm font-semibold hover:opacity-90 transition-opacity min-h-[44px]">
            {t('app.save')}
          </button>
        </div>
      </div>
    </div>
  )
}