export interface IconDefinition {
  key: string
  label: string
  category: string
}

export const iconCatalog: IconDefinition[] = [
  // Utilidades
  { key: 'internet', label: 'Internet', category: 'Utilidades' },
  { key: 'water', label: 'Agua', category: 'Utilidades' },
  { key: 'electricity', label: 'Electricidad', category: 'Utilidades' },
  { key: 'gas', label: 'Gas', category: 'Utilidades' },
  { key: 'phone', label: 'Teléfono', category: 'Utilidades' },
  { key: 'tv', label: 'TV/Cable', category: 'Utilidades' },
  { key: 'trash', label: 'Basura', category: 'Utilidades' },
  { key: 'delete', label: 'Basura', category: 'Utilidades' },
  { key: 'recycling', label: 'Reciclaje', category: 'Utilidades' },
  { key: 'sewer', label: 'Alcantarillado', category: 'Utilidades' },
  { key: 'wifi', label: 'WiFi', category: 'Utilidades' },
  { key: 'cable', label: 'Cable', category: 'Utilidades' },
  { key: 'satellite', label: 'Satélite', category: 'Utilidades' },
  // Hogar
  { key: 'home', label: 'Casa', category: 'Hogar' },
  { key: 'apartment', label: 'Apartamento', category: 'Hogar' },
  { key: 'building', label: 'Edificio', category: 'Hogar' },
  { key: 'key', label: 'Llave', category: 'Hogar' },
  { key: 'lock', label: 'Cerradura', category: 'Hogar' },
  { key: 'door', label: 'Puerta', category: 'Hogar' },
  { key: 'window', label: 'Ventana', category: 'Hogar' },
  { key: 'garden', label: 'Jardín', category: 'Hogar' },
  { key: 'garage', label: 'Garaje', category: 'Hogar' },
  { key: 'parking', label: 'Parqueo', category: 'Hogar' },
  { key: 'pool', label: 'Piscina', category: 'Hogar' },
  { key: 'roof', label: 'Techo', category: 'Hogar' },
  // Seguridad
  { key: 'shield', label: 'Escudo', category: 'Seguridad' },
  { key: 'alarm', label: 'Alarma', category: 'Seguridad' },
  { key: 'camera', label: 'Cámara', category: 'Seguridad' },
  { key: 'fire', label: 'Fuego', category: 'Seguridad' },
  { key: 'smoke', label: 'Humo', category: 'Seguridad' },
  { key: 'extinguisher', label: 'Extintor', category: 'Seguridad' },
  // Limpieza
  { key: 'cleaning', label: 'Limpieza', category: 'Limpieza' },
  { key: 'broom', label: 'Escoba', category: 'Limpieza' },
  { key: 'mop', label: 'Trapeador', category: 'Limpieza' },
  { key: 'laundry', label: 'Lavandería', category: 'Limpieza' },
  { key: 'dishwasher', label: 'Lavaplatos', category: 'Limpieza' },
  { key: 'vacuum', label: 'Aspiradora', category: 'Limpieza' },
  // Mantenimiento
  { key: 'wrench', label: 'Llave inglesa', category: 'Mantenimiento' },
  { key: 'hammer', label: 'Martillo', category: 'Mantenimiento' },
  { key: 'screwdriver', label: 'Destornillador', category: 'Mantenimiento' },
  { key: 'paint', label: 'Pintura', category: 'Mantenimiento' },
  { key: 'plumbing', label: 'Plomería', category: 'Mantenimiento' },
  { key: 'electrical', label: 'Eléctrico', category: 'Mantenimiento' },
  { key: 'pest', label: 'Control plagas', category: 'Mantenimiento' },
  // Salud
  { key: 'health', label: 'Salud', category: 'Salud' },
  { key: 'hospital', label: 'Hospital', category: 'Salud' },
  { key: 'pharmacy', label: 'Farmacia', category: 'Salud' },
  { key: 'ambulance', label: 'Ambulancia', category: 'Salud' },
  { key: 'medical', label: 'Médico', category: 'Salud' },
  { key: 'dental', label: 'Dental', category: 'Salud' },
  { key: 'vision', label: 'Visión', category: 'Salud' },
  // Educación
  { key: 'school', label: 'Escuela', category: 'Educación' },
  { key: 'university', label: 'Universidad', category: 'Educación' },
  { key: 'library', label: 'Biblioteca', category: 'Educación' },
  { key: 'books', label: 'Libros', category: 'Educación' },
  // Transporte
  { key: 'bus', label: 'Bus', category: 'Transporte' },
  { key: 'taxi', label: 'Taxi', category: 'Transporte' },
  { key: 'car', label: 'Carro', category: 'Transporte' },
  { key: 'motorcycle', label: 'Motocicleta', category: 'Transporte' },
  { key: 'bicycle', label: 'Bicicleta', category: 'Transporte' },
  { key: 'fuel', label: 'Combustible', category: 'Transporte' },
  { key: 'toll', label: 'Peaje', category: 'Transporte' },
  { key: 'insurance_car', label: 'Seguro auto', category: 'Transporte' },
  // Finanzas
  { key: 'bank', label: 'Banco', category: 'Finanzas' },
  { key: 'insurance', label: 'Seguro', category: 'Finanzas' },
  { key: 'credit', label: 'Crédito', category: 'Finanzas' },
  { key: 'tax', label: 'Impuestos', category: 'Finanzas' },
  { key: 'pension', label: 'Pensión', category: 'Finanzas' },
  { key: 'investment', label: 'Inversión', category: 'Finanzas' },
  { key: 'savings', label: 'Ahorro', category: 'Finanzas' },
  // Comunicación
  { key: 'mail', label: 'Correo', category: 'Comunicación' },
  { key: 'newspaper', label: 'Periódico', category: 'Comunicación' },
  { key: 'radio', label: 'Radio', category: 'Comunicación' },
  { key: 'podcast', label: 'Podcast', category: 'Comunicación' },
  // Otros
  { key: 'other', label: 'Otro', category: 'Otros' },
  { key: 'star', label: 'Estrella', category: 'Otros' },
  { key: 'heart', label: 'Corazón', category: 'Otros' },
  { key: 'calendar', label: 'Calendario', category: 'Otros' },
  { key: 'clock', label: 'Reloj', category: 'Otros' },
  { key: 'bell', label: 'Campana', category: 'Otros' },
  { key: 'search', label: 'Búsqueda', category: 'Otros' },
  { key: 'user', label: 'Usuario', category: 'Otros' },
  { key: 'group', label: 'Grupo', category: 'Otros' },
  { key: 'gift', label: 'Regalo', category: 'Otros' },
  { key: 'coffee', label: 'Café', category: 'Otros' },
  { key: 'restaurant', label: 'Restaurante', category: 'Otros' },
  { key: 'shopping', label: 'Compras', category: 'Otros' },
  { key: 'pet', label: 'Mascota', category: 'Otros' },
  { key: 'baby', label: 'Bebé', category: 'Otros' },
  { key: 'elderly', label: 'Adulto mayor', category: 'Otros' },
]

export function getIconCategories(): string[] {
  const categories = new Set<string>()
  iconCatalog.forEach(icon => categories.add(icon.category))
  return Array.from(categories)
}

export function filterIcons(query: string, category?: string): IconDefinition[] {
  const q = query.toLowerCase().trim()
  return iconCatalog.filter(icon => {
    const matchesQuery = !q || icon.key.toLowerCase().includes(q) || icon.label.toLowerCase().includes(q)
    const matchesCategory = !category || icon.category === category
    return matchesQuery && matchesCategory
  })
}
