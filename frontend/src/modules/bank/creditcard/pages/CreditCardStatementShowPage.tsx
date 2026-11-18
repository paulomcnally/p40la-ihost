import { useCallback, useEffect, useMemo, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { useI18nStore } from '../../../../stores/i18nStore'
import { usePageTitle } from '../../../../hooks/usePageTitle'
import { useToast } from '../../../../components/Toast'
import { useCurrencyFormatStore } from '../../../../stores/currencyFormatStore'
import { api } from '../../../../api'
import { Icon } from '../../../../components/Icons'
import CardMenu from '../../../../components/CardMenu'
import DeleteModal from '../../../../components/DeleteModal'
import LoadingSpinner from '../../../../components/LoadingSpinner'
import Select from '../../../../components/Select'
import type { CategoryGroup } from '../../../../types'
import { creditCardsApi } from '../api'
import TransactionFormModal, { type TransactionInput } from '../components/TransactionFormModal'
import type { CardStatement, CardTransaction, CreditCard } from '../types'
import {
  TRANSACTION_TYPES,
  formatISODate,
  statementStatusLabel,
  transactionTypeLabel,
} from '../utils'

export default function CreditCardStatementShowPage() {
  const { id, cardId, statementId } = useParams()
  const bankId = Number(id)
  const navigate = useNavigate()
  const { t, lang } = useI18nStore()
  const { showToast } = useToast()
  const formatMoney = useCurrencyFormatStore((state) => state.formatMoney)

  const [card, setCard] = useState<CreditCard | null>(null)
  const [statement, setStatement] = useState<CardStatement | null>(null)
  const [transactions, setTransactions] = useState<CardTransaction[]>([])
  const [categoryGroups, setCategoryGroups] = useState<CategoryGroup[]>([])
  const [loading, setLoading] = useState(true)
  const [deleteStatement, setDeleteStatement] = useState(false)
  const [deleteTransaction, setDeleteTransaction] = useState<CardTransaction | null>(null)
  const [editing, setEditing] = useState<CardTransaction | null>(null)
  const [modalOpen, setModalOpen] = useState(false)
  const [query, setQuery] = useState('')
  const [typeFilter, setTypeFilter] = useState('')

  usePageTitle(
    statement ? `${t('creditCards.statements')} ${formatISODate(statement.period_end, lang)}` : t('creditCards.title'),
  )

  const load = useCallback(async () => {
    try {
      const [cardDetail, statementDetail, transactionList, groups] = await Promise.all([
        creditCardsApi.get(Number(cardId)),
        creditCardsApi.getStatement(Number(statementId)),
        creditCardsApi.transactions(Number(statementId)),
        api.budget.categoryGroups.list(),
      ])
      setCard(cardDetail)
      setStatement(statementDetail)
      setTransactions(transactionList || [])
      setCategoryGroups(groups || [])
    } catch (err) {
      showToast(err instanceof Error ? err.message : t('errors.generic'), 'error')
    } finally {
      setLoading(false)
    }
  }, [cardId, statementId, showToast, t])

  useEffect(() => {
    load()
  }, [load])

  const filtered = useMemo(() => {
    const term = query.trim().toLowerCase()
    return transactions.filter((tx) => {
      if (typeFilter && tx.type !== typeFilter) return false
      if (term && !tx.concept.toLowerCase().includes(term)) return false
      return true
    })
  }, [query, transactions, typeFilter])

  const handleSaveTransaction = useCallback(
    async (input: TransactionInput) => {
      try {
        if (editing) {
          await creditCardsApi.updateTransaction(editing.id, input)
          showToast(t('creditCards.transaction_updated'), 'success')
        } else {
          await creditCardsApi.createTransaction(Number(statementId), input)
          showToast(t('creditCards.transaction_created'), 'success')
        }
        setModalOpen(false)
        setEditing(null)
        load()
      } catch (err) {
        showToast(err instanceof Error ? err.message : t('errors.generic'), 'error')
      }
    },
    [editing, load, showToast, statementId, t],
  )

  const handleDeleteTransaction = useCallback(async () => {
    if (!deleteTransaction) return
    try {
      await creditCardsApi.removeTransaction(deleteTransaction.id)
      setDeleteTransaction(null)
      showToast(t('creditCards.transaction_deleted'), 'success')
      load()
    } catch (err) {
      showToast(err instanceof Error ? err.message : t('errors.generic'), 'error')
    }
  }, [deleteTransaction, load, showToast, t])

  const handleDeleteStatement = useCallback(async () => {
    try {
      await creditCardsApi.removeStatement(Number(statementId))
      showToast(t('creditCards.statement_deleted'), 'success')
      navigate(`/bancos/${bankId}/tarjetas/${cardId}`, { replace: true })
    } catch (err) {
      showToast(err instanceof Error ? err.message : t('errors.generic'), 'error')
    }
  }, [bankId, cardId, navigate, showToast, statementId, t])

  if (loading) return <LoadingSpinner />

  if (!statement) {
    return (
      <p className="text-sm text-text-secondary text-center py-10">{t('creditCards.statement_not_found')}</p>
    )
  }

  const reconciliation = statement.reconciliation

  return (
    <div>
      <div className="relative bg-card rounded-ios shadow-ios p-4 sm:p-6 mb-6">
        <CardMenu
          options={[
            {
              label: t('app.edit'),
              icon: 'edit',
              onClick: () =>
                navigate(`/bancos/${bankId}/tarjetas/${cardId}/estados/${statement.id}/edit`),
            },
            {
              label: t('app.delete'),
              icon: 'delete',
              danger: true,
              onClick: () => setDeleteStatement(true),
            },
          ]}
        />
        <div className="pr-10">
          <p className="text-xs text-text-secondary uppercase tracking-wide">{t('creditCards.statement_period')}</p>
          <h2 className="text-xl sm:text-2xl font-bold">{formatISODate(statement.period_end, lang)}</h2>
          <p className="text-sm text-text-secondary mt-1">
            {card?.alias || ''}
            {card ? ` · •••• ${card.last4}` : ''}
          </p>
          <div className="flex flex-wrap items-center gap-2 mt-2">
            <span
              className={`px-2 py-0.5 rounded-full text-xs font-medium ${
                statement.status === 'paid'
                  ? 'bg-success/10 text-success'
                  : statement.status === 'partial'
                    ? 'bg-warning/10 text-warning'
                    : 'bg-bg text-text-secondary'
              }`}
            >
              {statementStatusLabel(statement.status, t)}
            </span>
            <span className="px-2 py-0.5 rounded-full text-xs font-medium bg-bg text-text-secondary">
              {statement.source === 'import' ? t('creditCards.source_import') : t('creditCards.source_manual')}
            </span>
            {statement.source_module_id && (
              <span className="text-xs text-text-secondary font-mono">{statement.source_module_id}</span>
            )}
          </div>
        </div>

        <div className="mt-4 grid grid-cols-2 lg:grid-cols-4 gap-3">
          <Metric label={t('creditCards.previous_balance')} value={formatMoney(statement.previous_balance)} sub={formatMoney(statement.previous_balance_usd) + ' USD'} />
          <Metric label={t('creditCards.new_balance')} value={formatMoney(statement.new_balance)} sub={formatMoney(statement.new_balance_usd) + ' USD'} />
          <Metric label={t('creditCards.min_payment')} value={formatMoney(statement.min_payment)} sub={formatMoney(statement.min_payment_usd) + ' USD'} />
          <Metric label={t('creditCards.total_payment')} value={formatMoney(statement.total_payment)} sub={formatMoney(statement.total_payment_usd) + ' USD'} />
        </div>
        <div className="mt-3 grid grid-cols-1 sm:grid-cols-2 gap-2 text-sm text-text-secondary">
          <p>
            {t('creditCards.statement_due_min')}: <strong className="text-text">{formatISODate(statement.min_payment_due_date, lang)}</strong>
          </p>
          <p>
            {t('creditCards.statement_due_full')}: <strong className="text-text">{formatISODate(statement.full_payment_due_date, lang)}</strong>
          </p>
        </div>
      </div>

      {reconciliation && (
        <div
          className={`rounded-ios p-4 mb-6 border ${
            reconciliation.balanced
              ? 'bg-success/10 border-success/30 text-success'
              : 'bg-warning/10 border-warning/30 text-warning'
          }`}
        >
          <p className="font-semibold flex items-center gap-2">
            <Icon name={reconciliation.balanced ? 'check' : 'warning'} className="w-5 h-5" />
            {reconciliation.balanced ? t('creditCards.reconciliation_balanced') : t('creditCards.reconciliation_unbalanced')}
          </p>
          <div className="mt-2 grid grid-cols-1 sm:grid-cols-3 gap-2 text-sm">
            <p>{t('creditCards.reconciliation_expected')}: {formatMoney(reconciliation.expected_local)}</p>
            <p>{t('creditCards.reconciliation_actual')}: {formatMoney(reconciliation.actual_local)}</p>
            <p>{t('creditCards.reconciliation_difference')}: {formatMoney(reconciliation.difference_local)}</p>
          </div>
        </div>
      )}

      {statement.summary && statement.summary.length > 0 && (
        <section className="bg-card rounded-ios shadow-ios p-4 sm:p-5 mb-6">
          <h3 className="text-sm font-semibold text-text-secondary uppercase tracking-wide mb-2">
            {t('creditCards.summary')}
          </h3>
          <div className="divide-y divide-border">
            {statement.summary.map((item) => (
              <div key={item.type} className="flex items-center justify-between gap-3 py-2 text-sm">
                <span>{transactionTypeLabel(item.type, t)}</span>
                <span className="font-mono text-right">
                  {formatMoney(item.amount_local)}
                  <span className="text-text-secondary"> / {formatMoney(item.amount_usd)} USD</span>
                </span>
              </div>
            ))}
          </div>
        </section>
      )}

      <div className="flex flex-col sm:flex-row sm:items-center gap-3 mb-3">
        <h3 className="text-sm font-semibold text-text-secondary uppercase tracking-wide flex-1">
          {t('creditCards.transactions')}
        </h3>
        <div className="flex flex-col sm:flex-row gap-2 sm:items-center">
          <input
            type="search"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder={t('creditCards.search_concept')}
            className="px-3 py-2 border border-border rounded-ios-sm bg-card text-text min-h-[44px] text-sm"
          />
          <div className="min-w-40">
            <Select
              options={[
                { value: '', label: t('creditCards.all_types') },
                ...TRANSACTION_TYPES.map((value) => ({ value, label: transactionTypeLabel(value, t) })),
              ]}
              value={typeFilter}
              onChange={(value) => setTypeFilter(String(value))}
              searchable={false}
            />
          </div>
          <button
            onClick={() => {
              setEditing(null)
              setModalOpen(true)
            }}
            className="inline-flex items-center justify-center gap-2 px-4 py-2 bg-primary text-white rounded-ios-sm hover:opacity-90 min-h-[44px] text-sm"
          >
            <Icon name="plus" className="w-4 h-4" />
            {t('creditCards.add_transaction')}
          </button>
        </div>
      </div>

      {filtered.length === 0 ? (
        <div className="bg-card rounded-ios shadow-ios p-8 text-center max-w-md mx-auto">
          <div className="w-12 h-12 mx-auto mb-3 text-primary opacity-80">
            <Icon name="credit" className="w-full h-full" />
          </div>
          <h4 className="text-lg font-semibold mb-2">{t('creditCards.empty_transactions_title')}</h4>
          <p className="text-text-secondary">{t('creditCards.empty_transactions_subtitle')}</p>
        </div>
      ) : (
        <div className="space-y-2">
          {filtered.map((tx) => (
            <div key={tx.id} className="bg-card rounded-ios shadow-ios p-3 sm:p-4 relative">
              <CardMenu
                options={[
                  {
                    label: t('app.edit'),
                    icon: 'edit',
                    onClick: () => {
                      setEditing(tx)
                      setModalOpen(true)
                    },
                  },
                  {
                    label: t('app.delete'),
                    icon: 'delete',
                    danger: true,
                    onClick: () => setDeleteTransaction(tx),
                  },
                ]}
              />
              <div className="pr-8 flex flex-col sm:flex-row sm:items-center gap-2">
                <div className="min-w-0 flex-1">
                  <p className="font-medium truncate">{tx.concept}</p>
                  <p className="text-xs text-text-secondary">
                    {formatISODate(tx.posted_at, lang)}
                    {tx.category_name ? ` · ${tx.category_name}` : ''}
                  </p>
                </div>
                <span className="text-xs px-2 py-0.5 rounded-full bg-bg text-text-secondary self-start sm:self-auto">
                  {transactionTypeLabel(tx.type, t)}
                </span>
                <div className="text-right font-mono text-sm">
                  {tx.amount_local !== 0 && <p>{formatMoney(tx.amount_local)}</p>}
                  {tx.amount_usd !== 0 && (
                    <p className="text-text-secondary">{formatMoney(tx.amount_usd)} USD</p>
                  )}
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {modalOpen && (
        <TransactionFormModal
          transaction={editing}
          categoryGroups={categoryGroups}
          onSave={handleSaveTransaction}
          onClose={() => {
            setModalOpen(false)
            setEditing(null)
          }}
        />
      )}

      {deleteStatement && (
        <DeleteModal
          title={t('creditCards.delete_statement_title')}
          subtitle={t('creditCards.delete_statement_subtitle').replace(
            '{date}',
            formatISODate(statement.period_end, lang),
          )}
          onConfirm={handleDeleteStatement}
          onCancel={() => setDeleteStatement(false)}
        />
      )}
      {deleteTransaction && (
        <DeleteModal
          title={t('creditCards.delete_transaction_title')}
          subtitle={t('creditCards.delete_transaction_subtitle').replace('{name}', deleteTransaction.concept)}
          onConfirm={handleDeleteTransaction}
          onCancel={() => setDeleteTransaction(null)}
        />
      )}
    </div>
  )
}

function Metric({ label, value, sub }: { label: string; value: string; sub?: string }) {
  return (
    <div className="bg-bg rounded-ios-sm p-3">
      <p className="text-xs text-text-secondary uppercase tracking-wide">{label}</p>
      <p className="text-sm sm:text-base font-semibold mt-0.5">{value}</p>
      {sub && <p className="text-xs text-text-secondary">{sub}</p>}
    </div>
  )
}
