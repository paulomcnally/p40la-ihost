import { useEffect, useState, useCallback } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { useAppStore } from '../stores/appStore'
import { useI18nStore } from '../stores/i18nStore'
import { api } from '../api'
import { Icon } from '../components/Icons'
import CreateMenu from '../components/CreateMenu'
import CardMenu from '../components/CardMenu'
import DeleteModal from '../components/DeleteModal'
import type { Bill, Service } from '../types'

const MONTHS = [
  '', 'January', 'February', 'March', 'April', 'May', 'June',
  'July', 'August', 'September', 'October', 'November', 'December'
]

export default function BillsPage() {
  const navigate = useNavigate()
  const { serviceId } = useParams()
  const { t } = useI18nStore()
  const { currencies } = useAppStore()
  const [bills, setBills] = useState<Bill[]>([])
  const [service, setService] = useState<Service | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<number | null>(null)

  useEffect(() => {
    const load = async () => {
      if (!serviceId) return
      const [svc, billList] = await Promise.all([
        api.services.get(Number(serviceId)),
        api.bills.list(Number(serviceId)),
      ])
      setService(svc)
      setBills(billList || [])
    }
    load()
  }, [serviceId])

  const handleDelete = useCallback(async () => {
    if (!deleteTarget) return
    await api.bills.delete(deleteTarget)
    setDeleteTarget(null)
    if (serviceId) {
      const list = await api.bills.list(Number(serviceId))
      setBills(list || [])
    }
  }, [deleteTarget, serviceId])

  if (!service) return <div className="text-center py-8 text-text-secondary">Loading...</div>

  const currency = currencies.find(c => c.id === service.currency_id)

  return (
    <div>
      <div className="flex items-center gap-3 mb-5">
        <button
          onClick={() => navigate('/services')}
          className="flex items-center gap-1 text-text-secondary hover:text-text transition-colors"
        >
          <Icon name="back" className="w-4 h-4" />
          {t('menu.services')}
        </button>
      </div>
      <div className="flex items-center justify-between mb-5">
        <div>
          <h2 className="text-2xl font-bold">{service.name}</h2>
          <p className="text-sm text-text-secondary">{t('bills.subtitle')}</p>
        </div>
        <CreateMenu options={[
          { label: t('bills.create'), icon: 'plus', onClick: () => navigate(`/bills/new?service=${serviceId}`) },
        ]} />
      </div>
      {bills.length === 0 ? (
        <div className="bg-card rounded-ios shadow-ios p-12 text-center max-w-md mx-auto">
          <div className="w-16 h-16 mx-auto mb-5 text-primary opacity-80">
            <Icon name="bill" className="w-full h-full" />
          </div>
          <h3 className="text-xl font-semibold mb-2">{t('bills.empty')}</h3>
          <button
            onClick={() => navigate(`/bills/new?service=${serviceId}`)}
            className="inline-flex items-center justify-center gap-2 px-6 py-3 bg-card border-2 border-dashed border-border rounded-ios text-primary font-semibold hover:border-primary hover:bg-primary/5 transition-colors"
          >
            <Icon name="plus" className="w-5 h-5" />
            {t('bills.create')}
          </button>
        </div>
      ) : (
        <div className="bg-card rounded-ios shadow-ios">
          <table className="w-full">
            <thead>
              <tr className="border-b border-border">
                <th className="text-left px-4 py-3 text-xs font-semibold text-text-secondary uppercase">{t('bills.year')}</th>
                <th className="text-left px-4 py-3 text-xs font-semibold text-text-secondary uppercase">{t('bills.month')}</th>
                <th className="text-left px-4 py-3 text-xs font-semibold text-text-secondary uppercase">{t('bills.amount')}</th>
                <th className="text-left px-4 py-3 text-xs font-semibold text-text-secondary uppercase">{t('bills.invoice_number')}</th>
                <th className="text-left px-4 py-3 text-xs font-semibold text-text-secondary uppercase">{t('bills.status')}</th>
                <th className="text-left px-4 py-3 text-xs font-semibold text-text-secondary uppercase">{t('bills.drive_url')}</th>
                <th className="w-10"></th>
              </tr>
            </thead>
            <tbody>
              {bills.map(bill => (
                <tr key={bill.id} className="border-b border-border last:border-b-0 hover:bg-bg/50">
                  <td className="px-4 py-3 text-sm">{bill.year}</td>
                  <td className="px-4 py-3 text-sm">{bill.month === 0 ? t('bills.annual') : t(`months.${bill.month}`, MONTHS[bill.month])}</td>
                  <td className="px-4 py-3 text-sm font-medium">{currency?.symbol}{bill.amount.toFixed(2)}</td>
                  <td className="px-4 py-3 text-sm">{bill.invoice_number || '-'}</td>
                  <td className="px-4 py-3">
                    <span className={`text-xs font-semibold px-2.5 py-1 rounded-full ${
                      bill.status === 'paid' ? 'bg-success/20 text-green-800' : 'bg-warning/20 text-yellow-800'
                    }`}>
                      {t(`bills.status_${bill.status}`)}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-sm">
                    {bill.drive_url ? (
                      <a href={bill.drive_url} target="_blank" rel="noopener noreferrer" className="text-primary hover:underline">
                        Drive
                      </a>
                    ) : '-'}
                  </td>
                  <td className="px-4 py-3 relative">
                    <CardMenu
                      options={[
                        { label: t('app.edit'), icon: 'edit', onClick: () => navigate(`/bills/edit/${bill.id}?service=${serviceId}`) },
                        { label: t('app.delete'), icon: 'delete', danger: true, onClick: () => setDeleteTarget(bill.id) },
                      ]}
                    />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      {deleteTarget && (
        <DeleteModal
          title={t('app.confirm')}
          subtitle={`${t('bills.title')} #${deleteTarget}`}
          onConfirm={handleDelete}
          onCancel={() => setDeleteTarget(null)}
        />
      )}
    </div>
  )
}
