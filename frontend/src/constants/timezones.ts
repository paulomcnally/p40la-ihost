export interface TimeZoneOption {
  value: string
  label: string
  offset: string
}

const FALLBACK_ZONES = [
  'UTC',
  'America/Argentina/Buenos_Aires',
  'America/Bogota',
  'America/Caracas',
  'America/Chicago',
  'America/Denver',
  'America/Guatemala',
  'America/Havana',
  'America/Lima',
  'America/Los_Angeles',
  'America/Managua',
  'America/Mexico_City',
  'America/New_York',
  'America/Panama',
  'America/Santiago',
  'America/Sao_Paulo',
  'America/Tegucigalpa',
  'Asia/Kolkata',
  'Asia/Shanghai',
  'Asia/Tokyo',
  'Australia/Sydney',
  'Europe/Berlin',
  'Europe/London',
  'Europe/Madrid',
  'Europe/Paris',
  'Pacific/Auckland',
]

export function getBrowserTimeZone(): string {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC'
  } catch {
    return 'UTC'
  }
}

interface IntlWithSupportedValues {
  supportedValuesOf?: (key: string) => string[]
}

export function getSupportedTimeZones(): string[] {
  const intl = Intl as IntlWithSupportedValues
  if (typeof intl.supportedValuesOf === 'function') {
    try {
      const zones = intl.supportedValuesOf('timeZone')
      if (zones.length > 0) return zones
    } catch {
      // fall through a la lista estática
    }
  }
  return FALLBACK_ZONES
}

function zoneOffsetLabel(tz: string): string {
  try {
    const parts = new Intl.DateTimeFormat('en', { timeZone: tz, timeZoneName: 'shortOffset' }).formatToParts(new Date())
    const offset = parts.find((p) => p.type === 'timeZoneName')?.value
    if (offset && offset !== 'GMT') return offset
  } catch {
    // zona inválida para Intl
  }
  return 'UTC'
}

export function getTimeZoneOptions(): TimeZoneOption[] {
  return getSupportedTimeZones()
    .map((z) => {
      const offset = zoneOffsetLabel(z)
      return { value: z, label: `(${offset}) ${z}`, offset }
    })
    .sort((a, b) => a.offset.localeCompare(b.offset) || a.value.localeCompare(b.value))
}

export function isValidTimeZone(tz: string): boolean {
  if (!tz) return false
  try {
    Intl.DateTimeFormat('en', { timeZone: tz })
    return true
  } catch {
    return false
  }
}

export function timeZoneShortLabel(tz: string): string {
  if (!tz) return 'UTC'
  const options = getTimeZoneOptions()
  return options.find((o) => o.value === tz)?.offset ?? 'UTC'
}