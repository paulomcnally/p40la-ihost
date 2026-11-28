import type { HomeMaintenanceArea, HomeMaintenanceKind } from './homeMaintenance'

export type HomeWorkRequestStatus = 'reported' | 'converted' | 'dismissed'

export type HomeWorkRequestCategory = 'repair' | 'improvement' | 'maintenance' | 'other'

export type HomeWorkRequestPriority = 'low' | 'normal' | 'high'

export type HomeWorkRequestMediaType = 'image' | 'video'

export interface HomeWorkRequestAttachment {
  id: number
  work_request_id: number
  media_type: HomeWorkRequestMediaType
  file_name: string
  mime_type: string
  size_bytes: number
  created_at: string
}

export interface HomeWorkRequest {
  id: number
  home_id: number
  title: string
  description: string
  category: HomeWorkRequestCategory
  priority: HomeWorkRequestPriority
  status: HomeWorkRequestStatus
  reported_at: string
  maintenance_id: number | null
  converted_at: string | null
  attachment_count: number
  attachments?: HomeWorkRequestAttachment[]
  created_at: string
  updated_at: string
}

export interface HomeWorkRequestInput {
  title: string
  description: string
  category: HomeWorkRequestCategory
  priority: HomeWorkRequestPriority
  reported_at: string
}

export interface HomeWorkRequestList {
  items: HomeWorkRequest[]
  pending_count: number
  truncated: boolean
}

export interface HomeWorkRequestConvertInput {
  kind: HomeMaintenanceKind
  area: HomeMaintenanceArea
}

export interface HomeWorkRequestConvertResult {
  maintenance_id: number
  work_request: HomeWorkRequest
}
