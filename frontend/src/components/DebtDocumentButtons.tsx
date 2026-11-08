import { api } from '../api'
import { Icon } from './Icons'
import { useI18nStore } from '../stores/i18nStore'
import type { DebtBill } from '../types'

interface DebtDocumentButtonsProps {
  bill: DebtBill
  className?: string
}

function openDocument(bill: DebtBill, kind: 'invoice' | 'receipt') {
  const url = kind === 'invoice' ? api.debts.invoiceUrl(bill.id) : api.debts.receiptUrl(bill.id)
  window.open(url, '_blank', 'noopener,noreferrer')
}

export default function DebtDocumentButtons({ bill, className = 'flex items-center gap-1' }: DebtDocumentButtonsProps) {
  const { t } = useI18nStore()

  if (!bill.has_invoice && !bill.has_receipt) return null

  return (
    <span className={className}>
      {bill.has_invoice && (
        <button
          onClick={() => openDocument(bill, 'invoice')}
          className="w-8 h-8 rounded-full flex items-center justify-center text-primary hover:bg-primary/10 transition-colors"
          title={t('deudas.view_invoice')}
          aria-label={t('deudas.view_invoice')}
        >
          <Icon name="bill" className="w-4 h-4" />
        </button>
      )}
      {bill.has_receipt && (
        <button
          onClick={() => openDocument(bill, 'receipt')}
          className="w-8 h-8 rounded-full flex items-center justify-center text-primary hover:bg-primary/10 transition-colors"
          title={t('deudas.view_receipt')}
          aria-label={t('deudas.view_receipt')}
        >
          <Icon name="pdf" className="w-4 h-4" />
        </button>
      )}
    </span>
  )
}
