import { useCallback, useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useI18nStore } from '../../../../stores/i18nStore'
import { useToast } from '../../../../components/Toast'
import { useCurrencyFormatStore } from '../../../../stores/currencyFormatStore'
import { Icon } from '../../../../components/Icons'
import CardMenu from '../../../../components/CardMenu'
import CreateMenu from '../../../../components/CreateMenu'
import DeleteModal from '../../../../components/DeleteModal'
import LoadingSpinner from '../../../../components/LoadingSpinner'
import { creditCardsApi } from '../api'
import type { CreditCard } from '../types'
import { networkLabel } from '../utils'

export default function CreditCardsTab({ bankId }: { bankId: number }) {
  const navigate = useNavigate()
  const { t } = useI18nStore()
  const { showToast } = useToast()
  const formatMoney = useCurrencyFormatStore((state) => state.formatMoney)

  const [cards, setCards] = useState<CreditCard[]>([])
  const [loading, setLoading] = useState(true)
  const [deleteTarget, setDeleteTarget] = useState<CreditCard | null>(null)

  const load = useCallback(async () => {
    try {
      const list = await creditCardsApi.listByBank(bankId)
      setCards(list || [])
    } catch (err) {
      showToast(err instanceof Error ? err.message : t('errors.generic'), 'error')
    } finally {
      setLoading(false)
    }
  }, [bankId, showToast, t])

  useEffect(() => {
    load()
  }, [load])

  const handleDelete = useCallback(async () => {
    if (!deleteTarget) return
    try {
      await creditCardsApi.remove(deleteTarget.id)
      setDeleteTarget(null)
      showToast(t('creditCards.card_deleted'), 'success')
      load()
    } catch (err) {
      showToast(err instanceof Error ? err.message : t('errors.generic'), 'error')
    }
  }, [deleteTarget, load, showToast, t])

  if (loading) return <LoadingSpinner />

  return (
    <div>
      <div className="flex items-center justify-between gap-3 mb-3">
        <h3 className="text-sm font-semibold text-text-secondary uppercase tracking-wide">
          {t('creditCards.title')}
        </h3>
        <CreateMenu
          title={t('creditCards.create')}
          options={[
            {
              label: t('creditCards.create'),
              icon: 'credit',
              onClick: () => navigate(`/bancos/${bankId}/tarjetas/new`),
            },
          ]}
        />
      </div>

      {cards.length === 0 ? (
        <div className="bg-card rounded-ios shadow-ios p-8 text-center max-w-md mx-auto mt-4">
          <div className="w-14 h-14 mx-auto mb-4 text-primary opacity-80">
            <Icon name="credit" className="w-full h-full" />
          </div>
          <h4 className="text-lg font-semibold mb-2">{t('creditCards.empty_title')}</h4>
          <p className="text-text-secondary mb-5">{t('creditCards.empty_subtitle')}</p>
          <button
            onClick={() => navigate(`/bancos/${bankId}/tarjetas/new`)}
            className="inline-flex items-center justify-center gap-2 px-5 py-2.5 bg-primary text-white rounded-ios-sm hover:opacity-90 min-h-[44px]"
          >
            <Icon name="plus" className="w-4 h-4" />
            {t('creditCards.create')}
          </button>
        </div>
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 sm:gap-4">
          {cards.map((card) => (
            <div
              key={card.id}
              onClick={() => navigate(`/bancos/${bankId}/tarjetas/${card.id}`)}
              className="bg-card rounded-ios shadow-ios p-4 relative cursor-pointer hover:shadow-ios-lg transition-shadow"
            >
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
                    onClick: () => setDeleteTarget(card),
                  },
                ]}
              />
              <div className="flex items-start gap-3 pr-8">
                <div className="w-10 h-10 rounded-ios bg-primary/10 text-primary flex items-center justify-center shrink-0">
                  <Icon name={card.icon || 'credit'} className="w-5 h-5" />
                </div>
                <div className="min-w-0">
                  <h4 className="font-semibold truncate">{card.alias}</h4>
                  <p className="text-sm text-text-secondary font-mono truncate">•••• {card.last4}</p>
                  <p className="text-xs text-text-secondary mt-0.5">
                    {networkLabel(card.network, t)}
                    {!card.active ? ` · ${t('creditCards.status_inactive')}` : ''}
                  </p>
                </div>
              </div>
              <div className="mt-3 grid grid-cols-2 gap-2 text-sm">
                <div>
                  <p className="text-xs text-text-secondary uppercase tracking-wide">
                    {t('creditCards.limit_label')} {card.limit_currency === 'USD' ? 'USD' : (card.currency_code || 'NIO')}
                  </p>
                  <p className="font-medium">
                    {card.limit_currency === 'USD'
                      ? (card.credit_limit_usd > 0 ? formatMoney(card.credit_limit_usd) : '—')
                      : (card.credit_limit > 0 ? formatMoney(card.credit_limit) : '—')}
                  </p>
                </div>
                <div>
                  <p className="text-xs text-text-secondary uppercase tracking-wide">
                    {t('creditCards.available')} {card.limit_currency === 'USD' ? 'USD' : (card.currency_code || 'NIO')}
                  </p>
                  <p className="font-medium">
                    {card.limit_currency === 'USD'
                      ? (card.credit_limit_usd > 0 ? formatMoney(card.available_usd) : '—')
                      : (card.credit_limit > 0 ? formatMoney(card.available_local) : '—')}
                  </p>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {deleteTarget && (
        <DeleteModal
          title={t('creditCards.delete_title')}
          subtitle={t('creditCards.delete_subtitle').replace('{name}', deleteTarget.alias)}
          onConfirm={handleDelete}
          onCancel={() => setDeleteTarget(null)}
        />
      )}
    </div>
  )
}
