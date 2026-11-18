import { useCallback, useEffect, useState } from 'react'
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
import type { Bank } from '../../../../types'
import { creditCardsApi } from '../api'
import type { CardStatement, CreditCard } from '../types'
import {
  formatISODate,
  networkLabel,
  statementStatusLabel,
  todayISO,
} from '../utils'

export default function CreditCardShowPage() {
  const { id, cardId } = useParams()
  const bankId = Number(id)
  const navigate = useNavigate()
  const { t, lang } = useI18nStore()
  const { showToast } = useToast()
  const formatMoney = useCurrencyFormatStore((state) => state.formatMoney)

  const [bank, setBank] = useState<Bank | null>(null)
  const [card, setCard] = useState<CreditCard | null>(null)
  const [statements, setStatements] = useState<CardStatement[]>([])
  const [loading, setLoading] = useState(true)
  const [deleteCard, setDeleteCard] = useState(false)
  const [deleteStatement, setDeleteStatement] = useState<CardStatement | null>(null)

  usePageTitle(card?.alias || t('creditCards.title'))

  const load = useCallback(async () => {
    try {
      const [bankDetail, cardDetail, statementList] = await Promise.all([
        api.banks.get(bankId),
        creditCardsApi.get(Number(cardId)),
        creditCardsApi.statements(Number(cardId)),
      ])
      setBank(bankDetail)
      setCard(cardDetail)
      setStatements(statementList || [])
    } catch (err) {
      showToast(err instanceof Error ? err.message : t('errors.generic'), 'error')
    } finally {
      setLoading(false)
    }
  }, [bankId, cardId, showToast, t])

  useEffect(() => {
    load()
  }, [load])

  const handleDeleteCard = useCallback(async () => {
    try {
      await creditCardsApi.remove(Number(cardId))
      showToast(t('creditCards.card_deleted'), 'success')
      navigate(`/bancos/${bankId}?tab=tarjetas`, { replace: true })
    } catch (err) {
      showToast(err instanceof Error ? err.message : t('errors.generic'), 'error')
    }
  }, [bankId, cardId, navigate, showToast, t])

  const handleDeleteStatement = useCallback(async () => {
    if (!deleteStatement) return
    try {
      await creditCardsApi.removeStatement(deleteStatement.id)
      setDeleteStatement(null)
      showToast(t('creditCards.statement_deleted'), 'success')
      load()
    } catch (err) {
      showToast(err instanceof Error ? err.message : t('errors.generic'), 'error')
    }
  }, [deleteStatement, load, showToast, t])

  if (loading) return <LoadingSpinner />

  if (!card) {
    return <p className="text-sm text-text-secondary text-center py-10">{t('creditCards.not_found')}</p>
  }

  const overdue = (statement: CardStatement) =>
    statement.status !== 'paid' &&
    statement.new_balance > 0 &&
    statement.min_payment_due_date < todayISO()

  return (
    <div>
      <div className="relative bg-card rounded-ios shadow-ios p-4 sm:p-6 mb-6">
        <CardMenu
          options={[
            {
              label: t('app.edit'),
              icon: 'edit',
              onClick: () => navigate(`/bancos/${bankId}/tarjetas/${card.id}/edit`),
            },
            {
              label: t('app.delete'),
              icon: 'delete',
              danger: true,
              onClick: () => setDeleteCard(true),
            },
          ]}
        />
        <div className="pr-10 flex items-start gap-4">
          <div className="w-12 h-12 rounded-ios bg-primary/10 text-primary flex items-center justify-center shrink-0">
            <Icon name={card.icon || 'credit'} className="w-6 h-6" />
          </div>
          <div className="min-w-0">
            <h2 className="text-xl sm:text-2xl font-bold truncate">{card.alias}</h2>
            <p className="text-sm text-text-secondary font-mono">
              •••• {card.last4} · {networkLabel(card.network, t)}
            </p>
            <p className="text-sm text-text-secondary mt-1">
              {bank?.name || card.bank_name || ''}
              {card.holder ? ` · ${card.holder}` : ''}
            </p>
            <span
              className={`inline-block mt-2 px-2 py-0.5 rounded-full text-xs font-medium ${
                card.active ? 'bg-success/10 text-success' : 'bg-bg text-text-secondary'
              }`}
            >
              {card.active ? t('creditCards.status_active') : t('creditCards.status_inactive')}
            </span>
          </div>
        </div>

        <div className="mt-4 grid grid-cols-2 sm:grid-cols-4 gap-3">
          <Metric
            label={`${t('creditCards.limit_label')} ${card.currency_code || 'NIO'}`}
            value={card.credit_limit > 0 ? formatMoney(card.credit_limit) : '—'}
          />
          <Metric
            label={`${t('creditCards.limit_label')} USD`}
            value={card.credit_limit_usd > 0 ? formatMoney(card.credit_limit_usd) : '—'}
          />
          <Metric
            label={`${t('creditCards.available')} ${card.currency_code || 'NIO'}`}
            value={card.credit_limit > 0 ? formatMoney(card.available_local) : '—'}
          />
          <Metric
            label={`${t('creditCards.available')} USD`}
            value={card.credit_limit_usd > 0 ? formatMoney(card.available_usd) : '—'}
          />
        </div>
        {card.limit_currency ? (
          <p className="text-xs text-text-secondary mt-2">
            {t('creditCards.limit_currency')}: {card.limit_currency}
            {card.exchange_rate > 0 ? ` · ${t('creditCards.exchange_rate')}: ${card.exchange_rate}` : ''}
          </p>
        ) : null}
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4 mb-6">
        <section className="bg-card rounded-ios shadow-ios p-4 sm:p-5">
          <h3 className="text-sm font-semibold text-text-secondary uppercase tracking-wide mb-2">
            {t('creditCards.info')}
          </h3>
          <InfoRow label={t('creditCards.statement_day')} value={String(card.statement_day)} />
          <InfoRow label={t('creditCards.payment_due_day')} value={String(card.payment_due_day)} />
          {card.bonus_due_day > 0 ? (
            <InfoRow label={t('creditCards.bonus_due_day')} value={String(card.bonus_due_day)} />
          ) : null}
          {card.category ? (
            <InfoRow label={t('creditCards.category')} value={t(`creditCards.category_${card.category}`)} />
          ) : null}
          <InfoRow
            label={t('creditCards.min_payment_pct')}
            value={card.min_payment_pct > 0 ? `${card.min_payment_pct}%` : '—'}
          />
          <InfoRow label={t('creditCards.holder')} value={card.holder} />
        </section>
        <section className="bg-card rounded-ios shadow-ios p-4 sm:p-5">
          <h3 className="text-sm font-semibold text-text-secondary uppercase tracking-wide mb-2">
            {t('creditCards.rates')}
          </h3>
          <InfoRow label={t('creditCards.interest_current_rate')} value={`${card.interest_current_rate}%`} />
          <InfoRow label={t('creditCards.late_surcharge')} value={`${card.interest_late_rate}%`} />
          <InfoRow label={t('creditCards.late_total')} value={`${card.late_total_rate}%`} />
          <InfoRow label={t('creditCards.interest_current_rate_usd')} value={`${card.interest_current_rate_usd}%`} />
          <InfoRow label={t('creditCards.late_surcharge_usd')} value={`${card.interest_late_rate_usd}%`} />
          <InfoRow label={`${t('creditCards.late_total')} USD`} value={`${card.late_total_rate_usd}%`} />
        </section>
        <section className="bg-card rounded-ios shadow-ios p-4 sm:p-5">
          <h3 className="text-sm font-semibold text-text-secondary uppercase tracking-wide mb-2">
            {t('creditCards.notes')}
          </h3>
          <p className="text-sm text-text whitespace-pre-wrap">{card.notes || t('banks.no_notes')}</p>
        </section>
      </div>

      <div className="flex items-center justify-between gap-3 mb-3">
        <h3 className="text-sm font-semibold text-text-secondary uppercase tracking-wide">
          {t('creditCards.statements')}
        </h3>
        <button
          onClick={() => navigate(`/bancos/${bankId}/tarjetas/${card.id}/importar`)}
          className="inline-flex items-center gap-2 px-4 py-2 bg-primary text-white rounded-ios-sm hover:opacity-90 min-h-[44px] text-sm"
        >
          <Icon name="upload" className="w-4 h-4" />
          {t('creditCards.import')}
        </button>
      </div>

      {statements.length === 0 ? (
        <div className="bg-card rounded-ios shadow-ios p-8 text-center max-w-md mx-auto mt-4">
          <div className="w-14 h-14 mx-auto mb-4 text-primary opacity-80">
            <Icon name="pdf" className="w-full h-full" />
          </div>
          <h4 className="text-lg font-semibold mb-2">{t('creditCards.empty_statements_title')}</h4>
          <p className="text-text-secondary mb-5">{t('creditCards.empty_statements_subtitle')}</p>
          <div className="flex flex-col sm:flex-row gap-2 justify-center">
            <button
              onClick={() => navigate(`/bancos/${bankId}/tarjetas/${card.id}/importar`)}
              className="inline-flex items-center justify-center gap-2 px-4 py-2.5 bg-primary text-white rounded-ios-sm hover:opacity-90 min-h-[44px]"
            >
              <Icon name="upload" className="w-4 h-4" />
              {t('creditCards.import')}
            </button>
            <button
              onClick={() => navigate(`/bancos/${bankId}/tarjetas/${card.id}/estados/new`)}
              className="inline-flex items-center justify-center gap-2 px-4 py-2.5 bg-bg text-text rounded-ios-sm hover:bg-border min-h-[44px]"
            >
              <Icon name="plus" className="w-4 h-4" />
              {t('creditCards.create_statement')}
            </button>
          </div>
        </div>
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 sm:gap-4">
          {statements.map((statement) => (
            <div
              key={statement.id}
              onClick={() => navigate(`/bancos/${bankId}/tarjetas/${card.id}/estados/${statement.id}`)}
              className="bg-card rounded-ios shadow-ios p-4 relative cursor-pointer hover:shadow-ios-lg transition-shadow"
            >
              <CardMenu
                options={[
                  {
                    label: t('app.edit'),
                    icon: 'edit',
                    onClick: () =>
                      navigate(`/bancos/${bankId}/tarjetas/${card.id}/estados/${statement.id}/edit`),
                  },
                  {
                    label: t('app.delete'),
                    icon: 'delete',
                    danger: true,
                    onClick: () => setDeleteStatement(statement),
                  },
                ]}
              />
              <div className="pr-8">
                <p className="font-semibold">{formatISODate(statement.period_end, lang)}</p>
                <p className="text-xs text-text-secondary">
                  {t('creditCards.statement_due_min')}: {formatISODate(statement.min_payment_due_date, lang)}
                </p>
                <p className="text-xs text-text-secondary">
                  {t('creditCards.statement_due_full')}: {formatISODate(statement.full_payment_due_date, lang)}
                </p>
              </div>
              <div className="mt-3 flex items-end justify-between gap-2">
                <div>
                  <p className="text-xs text-text-secondary uppercase tracking-wide">{t('creditCards.new_balance')}</p>
                  <p className="font-medium">{formatMoney(statement.new_balance)}</p>
                  <p className="text-sm text-text-secondary">{formatMoney(statement.new_balance_usd)} USD</p>
                </div>
                <span
                  className={`px-2 py-0.5 rounded-full text-xs font-medium ${
                    overdue(statement)
                      ? 'bg-danger/10 text-danger'
                      : statement.status === 'paid'
                        ? 'bg-success/10 text-success'
                        : 'bg-bg text-text-secondary'
                  }`}
                >
                  {overdue(statement) ? t('creditCards.overdue') : statementStatusLabel(statement.status, t)}
                </span>
              </div>
            </div>
          ))}
        </div>
      )}

      {deleteCard && (
        <DeleteModal
          title={t('creditCards.delete_title')}
          subtitle={t('creditCards.delete_subtitle').replace('{name}', card.alias)}
          onConfirm={handleDeleteCard}
          onCancel={() => setDeleteCard(false)}
        />
      )}
      {deleteStatement && (
        <DeleteModal
          title={t('creditCards.delete_statement_title')}
          subtitle={t('creditCards.delete_statement_subtitle').replace(
            '{date}',
            formatISODate(deleteStatement.period_end, lang),
          )}
          onConfirm={handleDeleteStatement}
          onCancel={() => setDeleteStatement(null)}
        />
      )}
    </div>
  )
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div className="bg-bg rounded-ios-sm p-3">
      <p className="text-xs text-text-secondary uppercase tracking-wide">{label}</p>
      <p className="text-sm sm:text-base font-semibold mt-0.5">{value}</p>
    </div>
  )
}

function InfoRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between gap-3 py-2 border-b border-border last:border-b-0">
      <span className="text-xs text-text-secondary uppercase tracking-wide">{label}</span>
      <span className="text-sm text-text text-right break-words">{value || '—'}</span>
    </div>
  )
}
