export function calcularEdad(birthDate: string, now: Date = new Date()): number {
  const [y, m, d] = birthDate.split('-').map(Number)
  if (!y || !m || !d) return 0

  const birth = new Date(y, m - 1, d)
  let age = now.getFullYear() - birth.getFullYear()
  const monthDiff = now.getMonth() - birth.getMonth()
  if (monthDiff < 0 || (monthDiff === 0 && now.getDate() < birth.getDate())) {
    age -= 1
  }
  return age > 0 ? age : 0
}