import { useCallback, useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { api } from '../api'
import { useI18nStore } from '../stores/i18nStore'
import { usePageTitle } from '../hooks/usePageTitle'
import { useToast } from '../components/Toast'
import { Icon } from '../components/Icons'
import CardMenu from '../components/CardMenu'
import DeleteModal from '../components/DeleteModal'
import LoadingSpinner from '../components/LoadingSpinner'
import Select from '../components/Select'
import HomeWorkRequestAttachmentField from '../components/home/HomeWorkRequestAttachmentField'
import {
  Badge,
  Field,
  Modal,
  PrimaryButton,
  SecondaryButton,
  SectionCard,
} from '../components/rental/RentalUI'
import type { HomeMaintenanceArea, HomeMaintenanceKind } from '../types/homeMaintenance'
import type { HomeWorkRequest } from '../types/homeWorkRequest'

const KINDS: HomeMaintenanceKind[] = ['corrective', 'preventive']
const AREAS: HomeMaintenanceArea[] = [
  'plumbing', 'electrical', 'roof', 'paint', 'carpentry',
  'appliances', 'garden', 'cleaning', 'other',
]

function statusTone(status: HomeWorkRequest['status']): 'info' | 'success' | 'neutral' {
  if (status === 'converted') return 'success'
  if (status === 'dismissed') return 'neutral'
  return 'info'
}

export default function HomeWorkRequestShowPage() {
  const navigate = useNavigate()
  const { id, workRequestId } = useParams()
  const homeId = Number(id)
  const recordId = Number(workRequestId)
  const { t } = useI18nStore()
  const { showToast } = useToast()

  const [homeName, setHomeName] = useState('')
  const [record, setRecord] = useState<HomeWorkRequest | null>(null)
  const [loading, setLoading] = useState(true)
  const [deleteRecord, setDeleteRecord] = useState(false)
  const [convertOpen, setConvertOpen] = useState(false)
  const [convertKind, setConvertKind] = useState<HomeMaintenanceKind>('corrective')
  const [convertArea, setConvertArea] = useState<HomeMaintenanceArea>('other')
  const [converting, setConverting] = useState(false)
  const [statusSaving, setStatusSaving] = useState(false)

  usePageTitle(t('homeWorkRequest.detail_title'))

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const [data, home] = await Promise.all([
        api.homeWorkRequest.get(homeId, recordId),
        api.homes.get(homeId),
      ])
      setRecord(data)
      setHomeName(home?.name ?? '')
    } catch {
      setRecord(null)
    } finally {
      setLoading(false)
    }
  }, [homeId, recordId])

  useEffect(() => {
    load()
  }, [load])

  const backToHome = () => navigate(`/home/${homeId}?tab=reports`)

  const handleDeleteRecord = useCallback(async () => {
    try {
      await api.homeWorkRequest.delete(homeId, recordId)
      showToast(t('homeWorkRequest.deleted_ok'), 'success')
      setDeleteRecord(false)
      backToHome()
    } catch (err) {
      showToast(err instanceof Error ? err.message : t('errors.generic'), 'error')
    }
  }, [homeId, recordId, showToast, t])

  const handleStatus = useCallback(async (status: 'reported' | 'dismissed') => {
    setStatusSaving(true)
    try {
      const updated = await api.homeWorkRequest.setStatus(homeId, recordId, status)
      if (updated) setRecord(updated)
      showToast(
        status === 'dismissed' ? t('homeWorkRequest.dismissed_ok') : t('homeWorkRequest.reactivated_ok'),
        'success',
      )
    } catch (err) {
      showToast(err instanceof Error ? err.message : t('errors.generic'), 'error')
    } finally {
      setStatusSaving(false)
    }
  }, [homeId, recordId, showToast, t])

  const handleConvert = useCallback(async () => {
    setConverting(true)
    try {
      const result = await api.homeWorkRequest.convert(homeId, recordId, {
        kind: convertKind,
        area: convertArea,
      })
      if (result?.work_request) setRecord(result.work_request)
      setConvertOpen(false)
      showToast(t('homeWorkRequest.converted_ok'), 'success')
    } catch (err) {
      showToast(err instanceof Error ? err.message : t('errors.generic'), 'error')
    } finally {
      setConverting(false)
    }
  }, [homeId, recordId, convertKind, convertArea, showToast, t])

  if (loading) return <LoadingSpinner />

  if (!record) {
    return (
      <div className="bg-card rounded-ios shadow-ios p-8 text-center max-w-md mx-auto mt-8">
        <h3 className="text-lg font-semibold mb-2">{t('homeWorkRequest.not_found')}</h3>
        <p className="text-sm text-text-secondary mb-4">{t('homeWorkRequest.not_found_hint')}</p>
        <PrimaryButton onClick={backToHome}>{t('homeWorkRequest.back_to_home')}</PrimaryButton>
      </div>
    )
  }

  const menu = [
    {
      label: t('app.edit'),
      icon: 'edit',
      onClick: () => navigate(`/home/${homeId}/workrequests/${record.id}/edit`),
    },
    ...(record.status === 'reported'
      ? [{
          label: t('homeWorkRequest.convert'),
          icon: 'wrench',
          onClick: () => setConvertOpen(true),
        }, {
          label: t('homeWorkRequest.dismiss'),
          icon: 'cancel',
          onClick: () => handleStatus('dismissed'),
        }]
      : []),
    ...(record.status === 'dismissed'
      ? [{
          label: t('homeWorkRequest.reactivate'),
          icon: 'recycling',
          onClick: () => handleStatus('reported'),
        }]
      : []),
    {
      label: t('app.delete'),
      icon: 'delete',
      danger: true,
      onClick: () => setDeleteRecord(true),
    },
  ]

  return (
    <div className="space-y-4 sm:space-y-5">
      <div className="relative bg-card rounded-ios shadow-ios p-4 sm:p-6">
        <CardMenu options={menu} />
        <div className="flex items-start gap-3">
          <div className="w-12 h-12 rounded-ios bg-primary/10 text-primary flex items-center justify-center shrink-0">
            <Icon name="warning" className="w-6 h-6" />
          </div>
          <div className="min-w-0 pr-10">
            <h2 className="text-xl font-bold break-words">{record.title}</h2>
            <div className="flex flex-wrap items-center gap-2 mt-2">
              <Badge tone={statusTone(record.status)}>{t(`homeWorkRequest.status_${record.status}`)}</Badge>
              <span className="text-xs px-2 py-0.5 rounded-full bg-bg text-text-secondary">
                {t(`homeWorkRequest.category_${record.category}`)}
              </span>
              <span className={`text-xs px-2 py-0.5 rounded-full ${record.priority === 'high' ? 'bg-warning/10 text-warning' : 'bg-bg text-text-secondary'}`}>
                {t(`homeWorkRequest.priority_${record.priority}`)}
              </span>
            </div>
            {homeName && <p className="text-sm text-text-secondary mt-1">{homeName}</p>}
          </div>
        </div>
      </div>

      <SectionCard title={t('homeWorkRequest.detail_section')}>
        <dl className="grid grid-cols-1 sm:grid-cols-2 gap-4 text-sm">
          <div>
            <dt className="text-text-secondary">{t('homeWorkRequest.reported_at')}</dt>
            <dd className="font-medium mt-0.5">{record.reported_at}</dd>
          </div>
          <div>
            <dt className="text-text-secondary">{t('homeWorkRequest.category')}</dt>
            <dd className="font-medium mt-0.5">{t(`homeWorkRequest.category_${record.category}`)}</dd>
          </div>
          <div>
            <dt className="text-text-secondary">{t('homeWorkRequest.priority')}</dt>
            <dd className="font-medium mt-0.5">{t(`homeWorkRequest.priority_${record.priority}`)}</dd>
          </div>
          <div>
            <dt className="text-text-secondary">{t('homeWorkRequest.status')}</dt>
            <dd className="font-medium mt-0.5">{t(`homeWorkRequest.status_${record.status}`)}</dd>
          </div>
        </dl>
        <div className="mt-4">
          <p className="text-sm text-text-secondary">{t('homeWorkRequest.description')}</p>
          <p className="text-sm mt-1 whitespace-pre-wrap break-words">
            {record.description || t('homeWorkRequest.no_description')}
          </p>
        </div>
        {record.maintenance_id && (
          <div className="mt-4 pt-4 border-t border-border">
            <p className="text-sm text-text-secondary">{t('homeWorkRequest.linked_maintenance')}</p>
            <button
              type="button"
              onClick={() => navigate(`/home/${homeId}/maintenances/${record.maintenance_id}`)}
              className="mt-2 inline-flex items-center gap-2 text-primary font-medium hover:underline min-h-[44px]"
            >
              <Icon name="wrench" className="w-4 h-4" />
              {t('homeWorkRequest.open_maintenance')}
            </button>
            {record.converted_at && (
              <p className="text-xs text-text-secondary mt-1">
                {t('homeWorkRequest.converted_at')}: {new Date(record.converted_at).toLocaleString()}
              </p>
            )}
          </div>
        )}
      </SectionCard>

      <SectionCard title={t('homeWorkRequest.evidence')} subtitle={t('homeWorkRequest.evidence_subtitle')}>
        <HomeWorkRequestAttachmentField
          homeId={homeId}
          workRequestId={record.id}
          attachments={record.attachments ?? []}
          onChange={load}
        />
      </SectionCard>

      {deleteRecord && (
        <DeleteModal
          title={t('homeWorkRequest.delete_title')}
          subtitle={t('homeWorkRequest.delete_subtitle').replace('{title}', record.title)}
          onConfirm={handleDeleteRecord}
          onCancel={() => setDeleteRecord(false)}
        />
      )}

      <Modal
        open={convertOpen}
        title={t('homeWorkRequest.convert_title')}
        onClose={() => { if (!converting) setConvertOpen(false) }}
        footer={(
          <>
            <SecondaryButton type="button" onClick={() => setConvertOpen(false)} disabled={converting}>
              {t('app.cancel')}
            </SecondaryButton>
            <PrimaryButton type="button" onClick={handleConvert} disabled={converting}>
              {converting ? t('app.saving') : t('homeWorkRequest.convert')}
            </PrimaryButton>
          </>
        )}
      >
        <p className="text-sm text-text-secondary">{t('homeWorkRequest.convert_hint')}</p>
        <Field label={t('homeMaintenance.kind')}>
          <Select
            options={KINDS.map((value) => ({ value, label: t(`homeMaintenance.kind_${value}`) }))}
            value={convertKind}
            onChange={(value) => setConvertKind(String(value) as HomeMaintenanceKind)}
            placeholder={t('homeMaintenance.kind')}
            emptyLabel={t('app.no_results')}
          />
        </Field>
        <Field label={t('homeMaintenance.area')}>
          <Select
            options={AREAS.map((value) => ({ value, label: t(`homeMaintenance.area_${value}`) }))}
            value={convertArea}
            onChange={(value) => setConvertArea(String(value) as HomeMaintenanceArea)}
            placeholder={t('homeMaintenance.area')}
            searchable
            searchPlaceholder={t('app.search')}
            emptyLabel={t('app.no_results')}
          />
        </Field>
      </Modal>

      {statusSaving && <span className="sr-only">{t('app.saving')}</span>}
    </div>
  )
}
