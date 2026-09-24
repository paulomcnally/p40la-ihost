import { useState, useEffect } from 'react'
import { api } from '../api'
import { useI18nStore } from '../stores/i18nStore'
import { Icon } from './Icons'
import type { Auto, ServiceAuto } from '../types'

interface AddPolicyModalProps {
  serviceId: number
  onSaved: () => void
  onCancel: () => void
  policy?: ServiceAuto | null
}

export default function AddPolicyModal({ serviceId, onSaved, onCancel, policy }: AddPolicyModalProps) {
  const { t } = useI18nStore()
  const isEditing = !!policy
  const [autos, setAutos] = useState<Auto[]>([])
  const [search, setSearch] = useState('')
  const [selectedAutoId, setSelectedAutoId] = useState<number | null>(isEditing ? policy!.auto_id : null)
  const [coverageType, setCoverageType] = useState<'daños_a_terceros' | 'full_cover'>(policy?.coverage_type ?? 'daños_a_terceros')
  const [policyNumber, setPolicyNumber] = useState(policy?.policy_number ?? '')
  const [certificate, setCertificate] = useState(policy?.certificate ?? '')
  const [insurerNumber, setInsurerNumber] = useState(policy?.insurer_number ?? '')
  const [loading, setLoading] = useState(true)
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    let cancelled = false
    const load = async () => {
      const [all, associated] = await Promise.all([
        api.autos.list(),
        api.services.listAutos(serviceId),
      ])
      if (cancelled) return
      const associatedIds = new Set((associated || []).map(a => a.auto_id))
      // En edición el auto actual queda fijo; el resto no se filtra.
      const available = isEditing
        ? (all || [])
        : (all || []).filter(a => !associatedIds.has(a.id))
      setAutos(available)
      setLoading(false)
    }
    load()
    return () => { cancelled = true }
  }, [serviceId, isEditing])

  const filtered = autos.filter(a =>
    `${a.brand} ${a.model} ${a.year} ${a.placa}`.toLowerCase().includes(search.toLowerCase())
  )

  const handleSubmit = async () => {
    if (!selectedAutoId) return
    setSubmitting(true)
    const body = {
      coverage_type: coverageType,
      policy_number: policyNumber,
      certificate: certificate || undefined,
      insurer_number: insurerNumber,
    }
    try {
      if (isEditing && policy) {
        await api.autos.updateService(policy.auto_id, serviceId, body)
      } else {
        await api.autos.addService(selectedAutoId, { service_id: serviceId, ...body })
      }
      onSaved()
    } catch {
      setSubmitting(false)
    }
  }

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4" onClick={onCancel}>
      <div className="bg-card rounded-ios shadow-ios w-full max-w-lg max-h-[80vh] flex flex-col" onClick={e => e.stopPropagation()}>
        <div className="p-4 border-b border-border">
          <h3 className="text-lg font-bold">{isEditing ? t('bills.polizas_edit') : t('bills.polizas_add')}</h3>
          {!isEditing && (
            <input
              type="text"
              placeholder={t('bills.polizas_search')}
              value={search}
              onChange={e => setSearch(e.target.value)}
              className="w-full mt-3 px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px]"
            />
          )}
        </div>
        {!isEditing && (
          <div className="flex-1 overflow-y-auto p-4">
            {loading ? (
              <p className="text-text-secondary text-center py-4">{t('app.loading')}</p>
            ) : filtered.length === 0 ? (
              <p className="text-text-secondary text-center py-4">{t('bills.polizas_auto_empty')}</p>
            ) : (
              <div className="space-y-2">
                {filtered.map(auto => (
                  <button
                    key={auto.id}
                    onClick={() => setSelectedAutoId(auto.id)}
                    className={`w-full flex items-center gap-3 p-3 rounded-ios-sm border text-left transition-colors ${
                      selectedAutoId === auto.id
                        ? 'border-primary bg-primary/5'
                        : 'border-border hover:border-primary/50'
                    }`}
                  >
                    <div className="w-9 h-9 rounded-ios bg-primary/10 text-primary flex items-center justify-center flex-shrink-0">
                      <Icon name={auto.icon || 'vehicle'} className="w-5 h-5" />
                    </div>
                    <div className="flex-1 min-w-0">
                      <p className="font-medium text-sm truncate">{auto.brand} {auto.model} {auto.year}</p>
                      <p className="text-xs text-text-secondary truncate">{auto.placa} · {auto.color}</p>
                    </div>
                  </button>
                ))}
              </div>
            )}
          </div>
        )}
        {selectedAutoId && (
          <div className="p-4 border-t border-border">
            <p className="text-sm font-medium mb-2">{t('bills.polizas_coverage')}</p>
            <div className="flex gap-2 mb-4">
              <button
                onClick={() => setCoverageType('daños_a_terceros')}
                className={`flex-1 px-3 py-2 rounded-ios-sm border text-sm font-medium transition-colors ${
                  coverageType === 'daños_a_terceros'
                    ? 'border-amber-400 bg-amber-50 text-amber-700 dark:border-amber-600 dark:bg-amber-900/30 dark:text-amber-300'
                    : 'border-border hover:border-amber-300 dark:hover:border-amber-700'
                }`}
              >
                {t('bills.polizas_third_party')}
              </button>
              <button
                onClick={() => setCoverageType('full_cover')}
                className={`flex-1 px-3 py-2 rounded-ios-sm border text-sm font-medium transition-colors ${
                  coverageType === 'full_cover'
                    ? 'border-green-400 bg-green-50 text-green-700 dark:border-green-600 dark:bg-green-900/30 dark:text-green-300'
                    : 'border-border hover:border-green-300 dark:hover:border-green-700'
                }`}
              >
                {t('bills.polizas_full_cover')}
              </button>
            </div>
            <div className="space-y-3 mb-4">
              <div>
                <label className="block text-sm font-medium mb-1">{t('bills.polizas_policy_number')}</label>
                <input
                  type="text"
                  value={policyNumber}
                  onChange={e => setPolicyNumber(e.target.value)}
                  placeholder="POL-2026-001"
                  className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px]"
                  required
                />
              </div>
              <div>
                <label className="block text-sm font-medium mb-1">{t('bills.polizas_certificate')}</label>
                <input
                  type="text"
                  value={certificate}
                  onChange={e => setCertificate(e.target.value)}
                  placeholder="CERT-123456"
                  className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px]"
                />
              </div>
              <div>
                <label className="block text-sm font-medium mb-1">{t('bills.polizas_insurer')}</label>
                <input
                  type="text"
                  value={insurerNumber}
                  onChange={e => setInsurerNumber(e.target.value)}
                  placeholder="1800 9911"
                  className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px]"
                  required
                />
              </div>
            </div>
            <div className="flex justify-end gap-3">
              <button
                onClick={onCancel}
                className="px-4 py-2 bg-bg text-text rounded-ios-sm hover:bg-border transition-colors min-h-[44px]"
              >
                {t('app.cancel')}
              </button>
              <button
                onClick={handleSubmit}
                disabled={submitting}
                className="px-4 py-2 bg-primary text-white rounded-ios-sm hover:bg-primary-hover disabled:opacity-50 transition-colors min-h-[44px]"
              >
                {submitting ? t('app.saving') : t('app.save')}
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}