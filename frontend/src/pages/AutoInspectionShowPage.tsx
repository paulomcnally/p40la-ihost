import { useCallback, useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { api } from '../api'
import { useI18nStore } from '../stores/i18nStore'
import { usePageTitle } from '../hooks/usePageTitle'
import { Icon } from '../components/Icons'
import LoadingSpinner from '../components/LoadingSpinner'
import CardMenu from '../components/CardMenu'
import DeleteModal from '../components/DeleteModal'
import InspectionModal from '../components/InspectionModal'
import AutoAttachmentsField from '../components/AutoAttachmentsField'
import type { AutoInspection } from '../types'

export default function AutoInspectionShowPage() {
  const { id, inspectionId } = useParams()
  const navigate = useNavigate()
  const { t } = useI18nStore()
  const [inspection, setInspection] = useState<AutoInspection | null>(null)
  const [loading, setLoading] = useState(true)
  const [editOpen, setEditOpen] = useState(false)
  const [deleteOpen, setDeleteOpen] = useState(false)

  usePageTitle(t('autos.inspections.detail_title'))

  const loadDetail = useCallback(async () => {
    const data = await api.autos.getInspection(Number(id), Number(inspectionId))
    setInspection(data)
  }, [id, inspectionId])

  useEffect(() => {
    loadDetail().finally(() => setLoading(false))
  }, [loadDetail])

  const handleDelete = useCallback(async () => {
    await api.autos.deleteInspection(Number(id), Number(inspectionId))
    setDeleteOpen(false)
    navigate(`/autos/${id}?tab=inspecciones`)
  }, [id, inspectionId, navigate])

  const handleUploadFiles = async (files: File[]) => {
    if (!inspection) return
    for (const file of files) {
      await api.autos.uploadInspectionAttachment(Number(id), inspection.id, file)
    }
    await loadDetail()
  }

  const handleDeleteAttachment = async (attachmentId: number) => {
    if (!inspection) return
    await api.autos.deleteInspectionAttachment(Number(id), inspection.id, attachmentId)
    await loadDetail()
  }

  if (loading) return <LoadingSpinner />
  if (!inspection) {
    return (
      <div className="text-center text-text-secondary p-8">
        <p className="font-medium">{t('autos.inspections.not_found')}</p>
        <p className="text-sm mt-1">{t('autos.inspections.not_found_hint')}</p>
      </div>
    )
  }

  return (
    <div>
      <div className="relative bg-card rounded-ios shadow-ios p-4 sm:p-6 mb-6">
        <CardMenu
          options={[
            { label: t('app.edit'), icon: 'edit', onClick: () => setEditOpen(true) },
            { label: t('app.delete'), icon: 'delete', danger: true, onClick: () => setDeleteOpen(true) },
          ]}
        />
        <div className="pr-10">
          <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
            <h2 className="text-xl sm:text-2xl font-bold break-words">{inspection.performed_at}</h2>
            <span className="text-xs font-semibold px-2 py-0.5 rounded-full bg-primary/10 text-primary">
              {t('autos.inspections.cycle_year')} {inspection.inspection_year}
            </span>
            <span
              className={`text-xs font-semibold px-2 py-0.5 rounded-full ${
                inspection.result === 'approved'
                  ? 'bg-success/10 text-success'
                  : 'bg-danger/10 text-danger'
              }`}
            >
              {inspection.result === 'approved'
                ? t('autos.inspections.result_approved')
                : t('autos.inspections.result_rejected')}
            </span>
          </div>
          <p className="text-text-secondary mt-1 break-words">{inspection.institution_name}</p>
          {inspection.notes && (
            <p className="text-sm text-text-secondary mt-2 break-words">{inspection.notes}</p>
          )}
        </div>
      </div>

      <div className="bg-card rounded-ios shadow-ios p-4 sm:p-6">
        <AutoAttachmentsField
          attachments={inspection.attachments}
          attachmentUrl={(attachment) =>
            api.autos.inspectionAttachmentUrl(Number(id), inspection.id, attachment.id)
          }
          onUpload={handleUploadFiles}
          onDelete={(attachment) => handleDeleteAttachment(attachment.id)}
        />
      </div>

      {editOpen && (
        <InspectionModal
          autoId={Number(id)}
          autoYear={inspection.inspection_year}
          inspection={inspection}
          onSaved={() => {
            setEditOpen(false)
            loadDetail()
          }}
          onCancel={() => setEditOpen(false)}
        />
      )}

      {deleteOpen && (
        <DeleteModal
          title={t('autos.inspections.delete_title')}
          subtitle={t('autos.inspections.delete_subtitle').replace('{year}', String(inspection.inspection_year))}
          onConfirm={handleDelete}
          onCancel={() => setDeleteOpen(false)}
        />
      )}
    </div>
  )
}
