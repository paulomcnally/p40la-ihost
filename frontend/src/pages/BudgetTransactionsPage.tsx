import { useEffect, useState, useCallback } from 'react'
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
import Select from '../components/Select'
import { useToast } from '../components/Toast'
import type { BudgetTransaction, Account, BudgetCategory } from '../types'

export default function BudgetTransactionsPage() {
  const { t } = useI18nStore()
  const { currencies, loadAll } = useAppStore()
  const formatMoney = useCurrencyFormatStore(s => s.formatMoney)
  const { showToast } = useToast()
  usePageTitle(t('budget.transactions'))

  const now = new Date()
  const [year, setYear] = useState(now.getFullYear())
  const [month, setMonth] = useState(now.getMonth() + 1)
  const [transactions, setTransactions] = useState<BudgetTransaction[]>([])
  const [accounts, setAccounts] = useState<Account[]>([])
  const [categories, setCategories] = useState<BudgetCategory[]>([])
  const [loading, setLoading] = useState(true)
  const [formOpen, setFormOpen] = useState(false)
  const [editTarget, setEditTarget] = useState<BudgetTransaction | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<BudgetTransaction | null>(null)
  const [accountsOpen, setAccountsOpen] = useState(false)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const [txList, accList, groups] = await Promise.all([
        api.budget.transactions.list(year, month),
        api.budget.accounts.list(),
        api.budget.categoryGroups.list(),
      ])
      setTransactions(txList || [])
      setAccounts(accList || [])
      setCategories(groups?.flatMap((g) => g.categories) || [])
    } catch (err) {
      showToast((err as Error).message, 'error')
    } finally {
      setLoading(false)
    }
  }, [year, month, showToast])

  useEffect(() => {
    loadAll()
  }, [loadAll])

  useEffect(() => {
    load()
  }, [load])

  const changeMonth = (delta: number) => {
    const d = new Date(year, month - 1 + delta, 1)
    setYear(d.getFullYear())
    setMonth(d.getMonth() + 1)
  }

  const handleDelete = async () => {
    if (!deleteTarget) return
    try {
      await api.budget.transactions.delete(deleteTarget.id)
      setDeleteTarget(null)
      await load()
    } catch (err) {
      showToast((err as Error).message, 'error')
    }
  }

  const createOptions = [
    { label: t('budget.create_transaction'), icon: 'plus', onClick: () => { setEditTarget(null); setFormOpen(true) } },
    { label: t('budget.accounts'), icon: 'bank', onClick: () => setAccountsOpen(true) },
  ]

  if (loading) return <LoadingSpinner />

  if (currencies.length === 0) {
    return (
      <div className="bg-card rounded-ios shadow-ios p-8 sm:p-12 text-center max-w-md mx-auto mt-8">
        <div className="w-16 h-16 mx-auto mb-5 text-primary opacity-80">
          <Icon name="savings" className="w-full h-full" />
        </div>
        <h3 className="text-lg sm:text-xl font-semibold mb-2">{t('budget.no_currencies')}</h3>
      </div>
    )
  }

  const monthName = new Date(year, month - 1, 1).toLocaleString('es', { month: 'long', year: 'numeric' })

  return (
    <div>
      <div className="flex items-center justify-between mb-4 sm:mb-5">
        <h2 className="text-xl sm:text-2xl font-bold">{t('budget.transactions')}</h2>
        <CreateMenu options={createOptions} />
      </div>

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

      {accounts.length === 0 ? (
        <div className="bg-card rounded-ios shadow-ios p-8 sm:p-12 text-center max-w-md mx-auto">
          <div className="w-16 h-16 mx-auto mb-5 text-primary opacity-80">
            <Icon name="bank" className="w-full h-full" />
          </div>
          <h3 className="text-lg sm:text-xl font-semibold mb-2">{t('budget.empty_accounts')}</h3>
          <p className="text-text-secondary mb-6">{t('budget.empty_accounts_subtitle')}</p>
          <button
            onClick={() => setAccountsOpen(true)}
            className="inline-flex items-center justify-center gap-2 px-6 py-3 bg-card border-2 border-dashed border-border rounded-ios text-primary font-semibold hover:border-primary hover:bg-primary/5 transition-colors"
          >
            <Icon name="plus" className="w-5 h-5" />
            {t('budget.create_account')}
          </button>
        </div>
      ) : transactions.length === 0 ? (
        <div className="bg-card rounded-ios shadow-ios p-8 sm:p-12 text-center max-w-md mx-auto">
          <div className="w-16 h-16 mx-auto mb-5 text-primary opacity-80">
            <Icon name="tag" className="w-full h-full" />
          </div>
          <h3 className="text-lg sm:text-xl font-semibold mb-2">{t('budget.empty_transactions')}</h3>
          <p className="text-text-secondary mb-6">{t('budget.empty_transactions_subtitle')}</p>
          <button
            onClick={() => { setEditTarget(null); setFormOpen(true) }}
            className="inline-flex items-center justify-center gap-2 px-6 py-3 bg-card border-2 border-dashed border-border rounded-ios text-primary font-semibold hover:border-primary hover:bg-primary/5 transition-colors"
          >
            <Icon name="plus" className="w-5 h-5" />
            {t('budget.create_transaction')}
          </button>
        </div>
      ) : (
        <>
          {/* Mobile: cards */}
          <div className="sm:hidden space-y-3">
            {transactions.map((tx) => (
              <div key={tx.id} className="bg-card rounded-ios shadow-ios p-4 relative">
                <CardMenu
                  options={[
                    { label: t('app.edit'), icon: 'edit', onClick: () => { setEditTarget(tx); setFormOpen(true) } },
                    { label: t('app.delete'), icon: 'delete', danger: true, onClick: () => setDeleteTarget(tx) },
                  ]}
                />
                <div className="flex items-center justify-between mb-1">
                  <span className="font-medium truncate">{tx.payee || tx.date}</span>
                  <span className={`font-semibold ${tx.outflow > 0 ? 'text-red-500' : 'text-primary'}`}>
                    {tx.outflow > 0 ? `-${formatMoney(tx.outflow, tx.currency_code)}` : `+${formatMoney(tx.inflow, tx.currency_code)}`}
                  </span>
                </div>
                <div className="text-sm text-text-secondary flex items-center gap-2 flex-wrap">
                  <span>{tx.account_name}</span>
                  <span>·</span>
                  <span>{tx.category_name || t('budget.uncategorized')}</span>
                  <span>·</span>
                  <span>{tx.date}</span>
                  {tx.cleared && <span className="text-primary text-xs">✓</span>}
                </div>
              </div>
            ))}
          </div>

          {/* Desktop: tabla */}
          <div className="hidden sm:block bg-card rounded-ios shadow-ios overflow-x-auto">
            <table className="w-full min-w-[720px]">
              <thead>
                <tr className="border-b border-border">
                  <th className="text-left px-4 py-3 text-xs font-semibold text-text-secondary uppercase">{t('budget.date')}</th>
                  <th className="text-left px-4 py-3 text-xs font-semibold text-text-secondary uppercase">{t('budget.payee')}</th>
                  <th className="text-left px-4 py-3 text-xs font-semibold text-text-secondary uppercase">{t('budget.category')}</th>
                  <th className="text-left px-4 py-3 text-xs font-semibold text-text-secondary uppercase">{t('budget.accounts')}</th>
                  <th className="text-left px-4 py-3 text-xs font-semibold text-text-secondary uppercase">{t('budget.memo')}</th>
                  <th className="text-right px-4 py-3 text-xs font-semibold text-text-secondary uppercase">{t('budget.outflow')}</th>
                  <th className="text-right px-4 py-3 text-xs font-semibold text-text-secondary uppercase">{t('budget.inflow')}</th>
                  <th className="text-left px-4 py-3 text-xs font-semibold text-text-secondary uppercase">{t('budget.cleared')}</th>
                  <th className="w-10"></th>
                </tr>
              </thead>
              <tbody>
                {transactions.map((tx) => (
                  <tr key={tx.id} className="border-b border-border last:border-b-0 hover:bg-bg/50">
                    <td className="px-4 py-3 text-sm whitespace-nowrap">{tx.date}</td>
                    <td className="px-4 py-3 text-sm font-medium">{tx.payee || '-'}</td>
                    <td className="px-4 py-3 text-sm">{tx.category_name || <span className="text-text-secondary">{t('budget.uncategorized')}</span>}</td>
                    <td className="px-4 py-3 text-sm text-text-secondary">{tx.account_name}</td>
                    <td className="px-4 py-3 text-sm text-text-secondary">{tx.memo || '-'}</td>
                    <td className="px-4 py-3 text-sm text-right text-red-500">
                      {tx.outflow > 0 ? formatMoney(tx.outflow, tx.currency_code) : ''}
                    </td>
                    <td className="px-4 py-3 text-sm text-right text-primary">
                      {tx.inflow > 0 ? formatMoney(tx.inflow, tx.currency_code) : ''}
                    </td>
                    <td className="px-4 py-3 text-sm">{tx.cleared ? '✓' : ''}</td>
                    <td className="px-2 py-3">
                      <CardMenu
                        options={[
                          { label: t('app.edit'), icon: 'edit', onClick: () => { setEditTarget(tx); setFormOpen(true) } },
                          { label: t('app.delete'), icon: 'delete', danger: true, onClick: () => setDeleteTarget(tx) },
                        ]}
                      />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </>
      )}

      {formOpen && (
        <TransactionFormModal
          edit={editTarget}
          accounts={accounts}
          categories={categories}
          currencies={currencies}
          defaultDate={`${year}-${String(month).padStart(2, '0')}-01`}
          onClose={() => { setFormOpen(false); setEditTarget(null) }}
          onSaved={() => { setFormOpen(false); setEditTarget(null); load() }}
        />
      )}

      {accountsOpen && (
        <AccountsModal
          accounts={accounts}
          currencies={currencies}
          onClose={() => setAccountsOpen(false)}
          onChanged={load}
        />
      )}

      {deleteTarget && (
        <DeleteModal
          title={t('app.confirm')}
          subtitle={`${t('budget.transactions')}: ${deleteTarget.payee || deleteTarget.date}`}
          onConfirm={handleDelete}
          onCancel={() => setDeleteTarget(null)}
        />
      )}
    </div>
  )
}

function TransactionFormModal({ edit, accounts, categories, currencies, defaultDate, onClose, onSaved }: {
  edit: BudgetTransaction | null
  accounts: Account[]
  categories: BudgetCategory[]
  currencies: { id: number; code: string; symbol: string }[]
  defaultDate: string
  onClose: () => void
  onSaved: () => void
}) {
  const { t } = useI18nStore()
  const { showToast } = useToast()
  const [form, setForm] = useState({
    account_id: edit?.account_id || accounts[0]?.id || 0,
    category_id: edit?.category_id != null ? edit.category_id : 0,
    currency_id: edit?.currency_id || currencies[0]?.id || 0,
    date: edit?.date || defaultDate,
    payee: edit?.payee || '',
    memo: edit?.memo || '',
    outflow: edit?.outflow ? String(edit.outflow) : '',
    inflow: edit?.inflow ? String(edit.inflow) : '',
    cleared: edit?.cleared ?? false,
  })
  const [saving, setSaving] = useState(false)

  const handleSave = async () => {
    const outflow = parseFloat(form.outflow) || 0
    const inflow = parseFloat(form.inflow) || 0
    if (outflow === 0 && inflow === 0) {
      showToast(t('budget.assign_required', 'Ingresá un monto'), 'error')
      return
    }
    const body = {
      account_id: form.account_id,
      category_id: form.category_id ? form.category_id : null,
      currency_id: form.currency_id,
      date: form.date,
      payee: form.payee,
      memo: form.memo,
      outflow,
      inflow,
      cleared: form.cleared,
    }
    setSaving(true)
    try {
      if (edit) {
        await api.budget.transactions.update(edit.id, body)
      } else {
        await api.budget.transactions.create(body)
      }
      onSaved()
    } catch (err) {
      showToast((err as Error).message, 'error')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
      <div className="bg-card rounded-ios shadow-ios w-full max-w-md max-h-[90vh] overflow-y-auto">
        <div className="flex items-center justify-between px-4 py-3 border-b border-border">
          <h3 className="font-bold">{edit ? t('app.edit') : t('budget.create_transaction')}</h3>
          <button onClick={onClose} className="w-9 h-9 rounded-full flex items-center justify-center hover:bg-bg transition-colors" title={t('app.close')}>
            <Icon name="cancel" className="w-5 h-5" />
          </button>
        </div>

        <div className="p-4 space-y-4">
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-sm font-medium mb-1">{t('budget.date')}</label>
              <input
                type="date"
                value={form.date}
                onChange={(e) => setForm({ ...form, date: e.target.value })}
                className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px]"
              />
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">{t('budget.payee')}</label>
              <input
                type="text"
                value={form.payee}
                onChange={(e) => setForm({ ...form, payee: e.target.value })}
                placeholder={t('budget.payee')}
                className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px]"
              />
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium mb-1">{t('budget.accounts')}</label>
            <Select
              value={form.account_id}
              onChange={(v) => setForm({ ...form, account_id: Number(v) })}
              options={accounts.map((a) => ({ value: a.id, label: a.name }))}
            />
          </div>

          <div>
            <label className="block text-sm font-medium mb-1">{t('budget.category')}</label>
            <Select
              value={form.category_id}
              onChange={(v) => setForm({ ...form, category_id: Number(v) })}
              options={[
                { value: 0, label: t('budget.uncategorized') },
                ...categories.map((c) => ({ value: c.id, label: c.name })),
              ]}
            />
          </div>

          <div>
            <label className="block text-sm font-medium mb-1">{t('budget.currency')}</label>
            <Select
              value={form.currency_id}
              onChange={(v) => setForm({ ...form, currency_id: Number(v) })}
              options={currencies.map((c) => ({ value: c.id, label: `${c.code} (${c.symbol})` }))}
            />
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-sm font-medium mb-1">{t('budget.outflow')}</label>
              <input
                type="number"
                step="0.01"
                min="0"
                value={form.outflow}
                onChange={(e) => setForm({ ...form, outflow: e.target.value })}
                placeholder="0.00"
                className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px]"
              />
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">{t('budget.inflow')}</label>
              <input
                type="number"
                step="0.01"
                min="0"
                value={form.inflow}
                onChange={(e) => setForm({ ...form, inflow: e.target.value })}
                placeholder="0.00"
                className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px]"
              />
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium mb-1">{t('budget.memo')}</label>
            <input
              type="text"
              value={form.memo}
              onChange={(e) => setForm({ ...form, memo: e.target.value })}
              placeholder={t('budget.memo')}
              className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px]"
            />
          </div>

          <label className="flex items-center gap-2 text-sm cursor-pointer">
            <input
              type="checkbox"
              checked={form.cleared}
              onChange={(e) => setForm({ ...form, cleared: e.target.checked })}
              className="w-4 h-4"
            />
            {t('budget.cleared')}
          </label>

          <button
            onClick={handleSave}
            disabled={saving}
            className="w-full px-4 py-3 bg-primary text-white rounded-ios-sm font-semibold hover:opacity-90 transition-opacity min-h-[44px]"
          >
            {saving ? t('app.saving') : t('app.save')}
          </button>
        </div>
      </div>
    </div>
  )
}

function AccountsModal({ accounts, currencies, onClose, onChanged }: {
  accounts: Account[]
  currencies: { id: number; code: string; symbol: string }[]
  onClose: () => void
  onChanged: () => void
}) {
  const { t } = useI18nStore()
  const { showToast } = useToast()
  const [form, setForm] = useState<Account | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<Account | null>(null)
  const [local, setLocal] = useState<Account[]>(accounts)

  const refresh = async () => {
    const list = await api.budget.accounts.list()
    setLocal(list || [])
    onChanged()
  }

  const handleDelete = async () => {
    if (!deleteTarget) return
    try {
      await api.budget.accounts.delete(deleteTarget.id)
      setDeleteTarget(null)
      await refresh()
    } catch (err) {
      showToast((err as Error).message, 'error')
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
      <div className="bg-card rounded-ios shadow-ios w-full max-w-md max-h-[90vh] flex flex-col">
        <div className="flex items-center justify-between px-4 py-3 border-b border-border">
          <h3 className="font-bold">{t('budget.accounts')}</h3>
          <button onClick={onClose} className="w-9 h-9 rounded-full flex items-center justify-center hover:bg-bg transition-colors" title={t('app.close')}>
            <Icon name="cancel" className="w-5 h-5" />
          </button>
        </div>

        <div className="flex-1 overflow-y-auto p-4 space-y-3">
          {local.length === 0 ? (
            <p className="text-center text-text-secondary text-sm py-6">{t('budget.empty_accounts')}</p>
          ) : (
            local.map((a) => (
              <div key={a.id} className="flex items-center justify-between border border-border rounded-ios-sm px-3 py-2">
                <div>
                  <div className="font-medium text-sm">{a.name}</div>
                  <div className="text-xs text-text-secondary capitalize">
                    {t(`budget.account_types_${a.type}`)} · {a.currency_code}
                  </div>
                </div>
                <div className="flex items-center gap-1">
                  <button
                    onClick={() => setForm(a)}
                    className="w-8 h-8 rounded-full flex items-center justify-center hover:bg-border transition-colors"
                    title={t('app.edit')}
                  >
                    <Icon name="edit" className="w-4 h-4" />
                  </button>
                  <button
                    onClick={() => setDeleteTarget(a)}
                    className="w-8 h-8 rounded-full flex items-center justify-center hover:bg-border transition-colors"
                    title={t('app.delete')}
                  >
                    <Icon name="delete" className="w-4 h-4" />
                  </button>
                </div>
              </div>
            ))
          )}

          <button
            onClick={() => setForm({ id: 0, name: '', type: 'checking', currency_id: currencies[0]?.id || 0, starting_balance: 0, created_at: '', updated_at: '' } as Account)}
            className="w-full flex items-center justify-center gap-2 px-4 py-3 bg-card border-2 border-dashed border-border rounded-ios text-primary font-semibold hover:border-primary hover:bg-primary/5 transition-colors min-h-[44px]"
          >
            <Icon name="plus" className="w-5 h-5" />
            {t('budget.create_account')}
          </button>
        </div>
      </div>

      {form && (
        <AccountFormModal
          edit={form}
          currencies={currencies}
          onClose={() => setForm(null)}
          onSaved={async () => {
            setForm(null)
            await refresh()
          }}
        />
      )}

      {deleteTarget && (
        <DeleteModal
          title={t('budget.delete') ?? t('app.confirm')}
          subtitle={deleteTarget.name}
          onConfirm={handleDelete}
          onCancel={() => setDeleteTarget(null)}
        />
      )}
    </div>
  )
}

function AccountFormModal({ edit, currencies, onClose, onSaved }: {
  edit: Account
  currencies: { id: number; code: string; symbol: string }[]
  onClose: () => void
  onSaved: () => void
}) {
  const { t } = useI18nStore()
  const { showToast } = useToast()
  const [name, setName] = useState(edit.name)
  const [type, setType] = useState(edit.type)
  const [currencyId, setCurrencyId] = useState(edit.currency_id || currencies[0]?.id || 0)
  const [balance, setBalance] = useState(edit.starting_balance ? String(edit.starting_balance) : '')
  const [saving, setSaving] = useState(false)

  const save = async () => {
    setSaving(true)
    try {
      const body = { name, type, currency_id: currencyId, starting_balance: parseFloat(balance) || 0 }
      if (edit.id) {
        await api.budget.accounts.update(edit.id, body)
      } else {
        await api.budget.accounts.create(body)
      }
      onSaved()
    } catch (err) {
      showToast((err as Error).message, 'error')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/40 p-4">
      <div className="bg-card rounded-ios shadow-ios w-full max-w-sm p-4 space-y-4">
        <h3 className="font-bold">{edit.id ? t('budget.edit_account') : t('budget.create_account')}</h3>

        <div>
          <label className="block text-sm font-medium mb-1">{t('budget.account_name')}</label>
          <input
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder={t('budget.account_name')}
            className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px]"
          />
        </div>

        <div>
          <label className="block text-sm font-medium mb-1">{t('budget.account_type')}</label>
          <Select
            value={type}
            onChange={(v) => setType(v as Account['type'])}
            options={(['checking', 'savings', 'credit_card', 'cash'] as const).map((tt) => ({
              value: tt,
              label: t(`budget.account_types_${tt}`),
            }))}
          />
        </div>

        <div>
          <label className="block text-sm font-medium mb-1">{t('budget.currency')}</label>
          <Select
            value={currencyId}
            onChange={(v) => setCurrencyId(Number(v))}
            options={currencies.map((c) => ({ value: c.id, label: `${c.code} (${c.symbol})` }))}
          />
        </div>

        <div>
          <label className="block text-sm font-medium mb-1">{t('budget.starting_balance')}</label>
          <input
            type="number"
            step="0.01"
            min="0"
            value={balance}
            onChange={(e) => setBalance(e.target.value)}
            placeholder="0.00"
            className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px]"
          />
        </div>

        <div className="flex gap-2">
          <button onClick={onClose} className="flex-1 px-4 py-3 bg-card border border-border rounded-ios-sm text-sm hover:bg-bg transition-colors min-h-[44px]">
            {t('app.cancel')}
          </button>
          <button onClick={save} disabled={saving} className="flex-1 px-4 py-3 bg-primary text-white rounded-ios-sm font-semibold hover:opacity-90 transition-opacity min-h-[44px]">
            {saving ? t('app.saving') : t('app.save')}
          </button>
        </div>
      </div>
    </div>
  )
}