// SPEC-118: códigos canónicos de color de autos.
// Debe coincidir exactamente con services.AutoColors (internal/services/auto.go).
export const AUTO_COLORS = [
  'white',
  'black',
  'gray',
  'silver',
  'red',
  'blue',
  'green',
  'yellow',
  'orange',
  'brown',
  'beige',
  'gold',
  'burgundy',
  'light_blue',
  'pink',
  'purple',
] as const

export type AutoColor = (typeof AUTO_COLORS)[number]

export function isAutoColor(value: string): value is AutoColor {
  return (AUTO_COLORS as readonly string[]).includes(value)
}
