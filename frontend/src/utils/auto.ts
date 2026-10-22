import type { Auto } from '../types'

type Translate = (key: string, fallback?: string) => string

// autoColorLabel traduce el código canónico de color (SPEC-118). Devuelve el
// código tal cual si no existe traducción y cadena vacía si no hay color.
export function autoColorLabel(t: Translate, code?: string): string {
  if (!code) return ''
  return t(`autos.colors.${code}`, code)
}

// autoSubtitle arma "año · color · placa" omitiendo las partes vacías, para
// que los campos opcionales no dejen separadores colgantes (SPEC-118 REQ-007).
export function autoSubtitle(t: Translate, auto: Pick<Auto, 'year' | 'color' | 'placa'>): string {
  return [auto.year ? String(auto.year) : '', autoColorLabel(t, auto.color), (auto.placa || '').trim()]
    .filter(Boolean)
    .join(' · ')
}
