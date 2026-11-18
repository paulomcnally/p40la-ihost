// SPEC-137: paleta canónica de tarjetas. Debe coincidir con
// internal/modules/bank/creditcard/model.go (Colors).
export const CARD_COLORS = [
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

export type CardColor = (typeof CARD_COLORS)[number]

export function isCardColor(value: string): value is CardColor {
  return (CARD_COLORS as readonly string[]).includes(value)
}
