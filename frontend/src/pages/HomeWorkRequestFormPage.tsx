import { useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { api } from '../api'
import { useI18nStore } from '../stores/i18nStore'
import { usePageTitle } from '../hooks/usePageTitle'
import { useToast } from '../components/Toast'
import { Icon } from '../components/Icons'
import LoadingSpinner from '../components/LoadingSpinner'
import Select from '../components/Select'
import { Field, PrimaryButton, SecondaryButton, TextArea, TextInput } from '../components/rental/RentalUI'
import type {
  HomeWorkRequestCategory,
  HomeWorkRequestInput,
  HomeWorkRequestPriority,
} from '../types/homeWorkRequest'

const CATEGORIES: HomeWorkRequestCategory[] = ['repair', 'improvement', 'maintenance', 'other']
const PRIORITIES: HomeWorkRequestPriority[] = ['low', 'normal', 'high']

function todayISO(): string {
  const now = new Date()
  const offset = now.getTimezoneOffset()
  return new Date(now.getTime() - offset * 60 * 1000).toISOString().slice(0, 10)
}

export default function HomeWorkRequestFormPage() {
  const navigate = useNavigate()
  const { id, workRequestId } = useParams()
  const homeId = Number(id)
  const isEdit = !!workRequestId
  const { t } = useI18nStore()
  const { showToast } = useToast()

  const [homeName, setHomeName] = useState('')
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [category, setCategory] = useState<HomeWorkRequestCategory>('repair')
  const [priority, setPriority] = useState<HomeWorkRequestPriority>('normal')
  const [reportedAt, setReportedAt] = useState(todayISO())
  const [loading, setLoading] = useState(true)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')

  usePageTitle(isEdit ? t('homeWorkRequest.edit_title') : t('homeWorkRequest.new'))

  useEffect(() => {
    const tasks: Promise<unknown>[] = [
      api.homes.get(homeId).then((home) => setHomeName(home?.name ?? '')),
    ]
    if (isEdit) {
      tasks.push(
        api.homeWorkRequest.get(homeId, Number(workRequestId)).then((record) => {
          if (!record) return
          setTitle(record.title)
          setDescription(record.description)
          setCategory(record.category)
          setPriority(record.priority)
          setReportedAt(record.reported_at)
        }),
      )
    }
    Promise.all(tasks).finally(() => setLoading(false))
  }, [homeId, isEdit, workRequestId])

  const categoryOptions = CATEGORIES.map((value) => ({
    value,
    label: t(`homeWorkRequest.category_${value}`),
  }))
  const priorityOptions = PRIORITIES.map((value) => ({
    value,
    label: t(`homeWorkRequest.priority_${value}`),
  }))

  const backToHome = () => navigate(`/home/${homeId}?tab=reports`)

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault()
    setError('')
    if (!title.trim()) {
      setError(t('homeWorkRequest.title_required'))
      return
    }
    if (!reportedAt) {
      setError(t('homeWorkRequest.date_required'))
      return
    }
    if (reportedAt > todayISO()) {
      setError(t('homeWorkRequest.date_future'))
      return
    }

    setSubmitting(true)
    try {
      const body: HomeWorkRequestInput = {
        title: title.trim(),
        description: description.trim(),
        category,
        priority,
        reported_at: reportedAt,
      }
      const saved = isEdit
        ? await api.homeWorkRequest.update(homeId, Number(workRequestId), body)
        : await api.homeWorkRequest.create(homeId, body)
      if (!saved) throw new Error(t('errors.generic'))
      showToast(t('homeWorkRequest.saved_ok'), 'success')
      navigate(`/home/${homeId}/workrequests/${saved.id}`)
    } catch (err) {
      setError(err instanceof Error ? err.message : t('errors.generic'))
    } finally {
      setSubmitting(false)
    }
  }

  if (loading) return <LoadingSpinner />

  return (
    <form onSubmit={handleSubmit} className="max-w-2xl mx-auto space-y-4">
      <div className="bg-card rounded-ios shadow-ios p-4 sm:p-6">
        <div className="flex items-center gap-3">
          <div className="w-11 h-11 rounded-ios bg-primary/10 text-primary flex items-center justify-center shrink-0">
            <Icon name="warning" className="w-5 h-5" />
          </div>
          <div className="min-w-0">
            <h2 className="text-lg font-bold truncate">
              {isEdit ? t('homeWorkRequest.edit_title') : t('homeWorkRequest.new')}
            </h2>
            {homeName && <p className="text-sm text-text-secondary truncate">{homeName}</p>}
          </div>
        </div>
      </div>

      <div className="bg-card rounded-ios shadow-ios p-4 sm:p-6 space-y-4">
        <Field label={t('homeWorkRequest.title')}>
          <TextInput
            value={title}
            onChange={(event) => setTitle(event.target.value)}
            placeholder={t('homeWorkRequest.title_placeholder')}
            maxLength={200}
            required
          />
        </Field>

        <Field label={t('homeWorkRequest.description')}>
          <TextArea
            value={description}
            onChange={(event) => setDescription(event.target.value)}
            placeholder={t('homeWorkRequest.description_placeholder')}
            rows={4}
            maxLength={2000}
          />
        </Field>

        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <Field label={t('homeWorkRequest.category')}>
            <Select
              options={categoryOptions}
              value={category}
              onChange={(value) => setCategory(String(value) as HomeWorkRequestCategory)}
              placeholder={t('homeWorkRequest.category')}
              emptyLabel={t('app.no_results')}
            />
          </Field>
          <Field label={t('homeWorkRequest.priority')}>
            <Select
              options={priorityOptions}
              value={priority}
              onChange={(value) => setPriority(String(value) as HomeWorkRequestPriority)}
              placeholder={t('homeWorkRequest.priority')}
              emptyLabel={t('app.no_results')}
            />
          </Field>
        </div>

        <Field label={t('homeWorkRequest.reported_at')}>
          <TextInput
            type="date"
            value={reportedAt}
            max={todayISO()}
            onChange={(event) => setReportedAt(event.target.value)}
            required
          />
        </Field>

        {error && <p className="text-sm text-danger">{error}</p>}

        <div className="flex flex-col sm:flex-row gap-3 pt-2">
          <PrimaryButton type="submit" disabled={submitting}>
            <Icon name="save" className="w-4 h-4" />
            {submitting ? t('app.saving') : t('app.save')}
          </PrimaryButton>
          <SecondaryButton type="button" onClick={backToHome} disabled={submitting}>
            {t('app.cancel')}
          </SecondaryButton>
        </div>
      </div>
    </form>
  )
}
