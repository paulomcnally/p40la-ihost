// SPEC-134: catálogo estático de países ISO 3166-1 alpha-2.
//
// Se versiona en el repo para que el selector de país funcione sin red, sin
// dependencias externas y con filtrado instantáneo en el iHost. Los nombres se
// mantienen en español e inglés; el orden se resuelve en tiempo de carga.

export interface Country {
  code: string
  name_es: string
  name_en: string
}

export const DEFAULT_COUNTRY_CODE = 'NI'

const COUNTRY_DATA: Country[] = [
  { code: 'AD', name_es: 'Andorra', name_en: 'Andorra' },
  { code: 'AE', name_es: 'Emiratos Árabes Unidos', name_en: 'United Arab Emirates' },
  { code: 'AF', name_es: 'Afganistán', name_en: 'Afghanistan' },
  { code: 'AG', name_es: 'Antigua y Barbuda', name_en: 'Antigua and Barbuda' },
  { code: 'AI', name_es: 'Anguila', name_en: 'Anguilla' },
  { code: 'AL', name_es: 'Albania', name_en: 'Albania' },
  { code: 'AM', name_es: 'Armenia', name_en: 'Armenia' },
  { code: 'AO', name_es: 'Angola', name_en: 'Angola' },
  { code: 'AQ', name_es: 'Antártida', name_en: 'Antarctica' },
  { code: 'AR', name_es: 'Argentina', name_en: 'Argentina' },
  { code: 'AS', name_es: 'Samoa Americana', name_en: 'American Samoa' },
  { code: 'AT', name_es: 'Austria', name_en: 'Austria' },
  { code: 'AU', name_es: 'Australia', name_en: 'Australia' },
  { code: 'AW', name_es: 'Aruba', name_en: 'Aruba' },
  { code: 'AX', name_es: 'Islas Åland', name_en: 'Åland Islands' },
  { code: 'AZ', name_es: 'Azerbaiyán', name_en: 'Azerbaijan' },
  { code: 'BA', name_es: 'Bosnia y Herzegovina', name_en: 'Bosnia and Herzegovina' },
  { code: 'BB', name_es: 'Barbados', name_en: 'Barbados' },
  { code: 'BD', name_es: 'Bangladés', name_en: 'Bangladesh' },
  { code: 'BE', name_es: 'Bélgica', name_en: 'Belgium' },
  { code: 'BF', name_es: 'Burkina Faso', name_en: 'Burkina Faso' },
  { code: 'BG', name_es: 'Bulgaria', name_en: 'Bulgaria' },
  { code: 'BH', name_es: 'Baréin', name_en: 'Bahrain' },
  { code: 'BI', name_es: 'Burundi', name_en: 'Burundi' },
  { code: 'BJ', name_es: 'Benín', name_en: 'Benin' },
  { code: 'BL', name_es: 'San Bartolomé', name_en: 'Saint Barthélemy' },
  { code: 'BM', name_es: 'Bermudas', name_en: 'Bermuda' },
  { code: 'BN', name_es: 'Brunéi', name_en: 'Brunei' },
  { code: 'BO', name_es: 'Bolivia', name_en: 'Bolivia' },
  { code: 'BQ', name_es: 'Caribe Neerlandés', name_en: 'Caribbean Netherlands' },
  { code: 'BR', name_es: 'Brasil', name_en: 'Brazil' },
  { code: 'BS', name_es: 'Bahamas', name_en: 'Bahamas' },
  { code: 'BT', name_es: 'Bután', name_en: 'Bhutan' },
  { code: 'BV', name_es: 'Isla Bouvet', name_en: 'Bouvet Island' },
  { code: 'BW', name_es: 'Botsuana', name_en: 'Botswana' },
  { code: 'BY', name_es: 'Bielorrusia', name_en: 'Belarus' },
  { code: 'BZ', name_es: 'Belice', name_en: 'Belize' },
  { code: 'CA', name_es: 'Canadá', name_en: 'Canada' },
  { code: 'CC', name_es: 'Islas Cocos', name_en: 'Cocos (Keeling) Islands' },
  { code: 'CD', name_es: 'República Democrática del Congo', name_en: 'Democratic Republic of the Congo' },
  { code: 'CF', name_es: 'República Centroafricana', name_en: 'Central African Republic' },
  { code: 'CG', name_es: 'República del Congo', name_en: 'Republic of the Congo' },
  { code: 'CH', name_es: 'Suiza', name_en: 'Switzerland' },
  { code: 'CI', name_es: 'Costa de Marfil', name_en: "Côte d'Ivoire" },
  { code: 'CK', name_es: 'Islas Cook', name_en: 'Cook Islands' },
  { code: 'CL', name_es: 'Chile', name_en: 'Chile' },
  { code: 'CM', name_es: 'Camerún', name_en: 'Cameroon' },
  { code: 'CN', name_es: 'China', name_en: 'China' },
  { code: 'CO', name_es: 'Colombia', name_en: 'Colombia' },
  { code: 'CR', name_es: 'Costa Rica', name_en: 'Costa Rica' },
  { code: 'CU', name_es: 'Cuba', name_en: 'Cuba' },
  { code: 'CV', name_es: 'Cabo Verde', name_en: 'Cape Verde' },
  { code: 'CW', name_es: 'Curazao', name_en: 'Curaçao' },
  { code: 'CX', name_es: 'Isla de Navidad', name_en: 'Christmas Island' },
  { code: 'CY', name_es: 'Chipre', name_en: 'Cyprus' },
  { code: 'CZ', name_es: 'Chequia', name_en: 'Czechia' },
  { code: 'DE', name_es: 'Alemania', name_en: 'Germany' },
  { code: 'DJ', name_es: 'Yibuti', name_en: 'Djibouti' },
  { code: 'DK', name_es: 'Dinamarca', name_en: 'Denmark' },
  { code: 'DM', name_es: 'Dominica', name_en: 'Dominica' },
  { code: 'DO', name_es: 'República Dominicana', name_en: 'Dominican Republic' },
  { code: 'DZ', name_es: 'Argelia', name_en: 'Algeria' },
  { code: 'EC', name_es: 'Ecuador', name_en: 'Ecuador' },
  { code: 'EE', name_es: 'Estonia', name_en: 'Estonia' },
  { code: 'EG', name_es: 'Egipto', name_en: 'Egypt' },
  { code: 'EH', name_es: 'Sáhara Occidental', name_en: 'Western Sahara' },
  { code: 'ER', name_es: 'Eritrea', name_en: 'Eritrea' },
  { code: 'ES', name_es: 'España', name_en: 'Spain' },
  { code: 'ET', name_es: 'Etiopía', name_en: 'Ethiopia' },
  { code: 'FI', name_es: 'Finlandia', name_en: 'Finland' },
  { code: 'FJ', name_es: 'Fiyi', name_en: 'Fiji' },
  { code: 'FK', name_es: 'Islas Malvinas', name_en: 'Falkland Islands' },
  { code: 'FM', name_es: 'Micronesia', name_en: 'Micronesia' },
  { code: 'FO', name_es: 'Islas Feroe', name_en: 'Faroe Islands' },
  { code: 'FR', name_es: 'Francia', name_en: 'France' },
  { code: 'GA', name_es: 'Gabón', name_en: 'Gabon' },
  { code: 'GB', name_es: 'Reino Unido', name_en: 'United Kingdom' },
  { code: 'GD', name_es: 'Granada', name_en: 'Grenada' },
  { code: 'GE', name_es: 'Georgia', name_en: 'Georgia' },
  { code: 'GF', name_es: 'Guayana Francesa', name_en: 'French Guiana' },
  { code: 'GG', name_es: 'Guernsey', name_en: 'Guernsey' },
  { code: 'GH', name_es: 'Ghana', name_en: 'Ghana' },
  { code: 'GI', name_es: 'Gibraltar', name_en: 'Gibraltar' },
  { code: 'GL', name_es: 'Groenlandia', name_en: 'Greenland' },
  { code: 'GM', name_es: 'Gambia', name_en: 'Gambia' },
  { code: 'GN', name_es: 'Guinea', name_en: 'Guinea' },
  { code: 'GP', name_es: 'Guadalupe', name_en: 'Guadeloupe' },
  { code: 'GQ', name_es: 'Guinea Ecuatorial', name_en: 'Equatorial Guinea' },
  { code: 'GR', name_es: 'Grecia', name_en: 'Greece' },
  { code: 'GS', name_es: 'Islas Georgias del Sur y Sandwich del Sur', name_en: 'South Georgia and the South Sandwich Islands' },
  { code: 'GT', name_es: 'Guatemala', name_en: 'Guatemala' },
  { code: 'GU', name_es: 'Guam', name_en: 'Guam' },
  { code: 'GW', name_es: 'Guinea-Bisáu', name_en: 'Guinea-Bissau' },
  { code: 'GY', name_es: 'Guyana', name_en: 'Guyana' },
  { code: 'HK', name_es: 'Hong Kong', name_en: 'Hong Kong' },
  { code: 'HM', name_es: 'Islas Heard y McDonald', name_en: 'Heard Island and McDonald Islands' },
  { code: 'HN', name_es: 'Honduras', name_en: 'Honduras' },
  { code: 'HR', name_es: 'Croacia', name_en: 'Croatia' },
  { code: 'HT', name_es: 'Haití', name_en: 'Haiti' },
  { code: 'HU', name_es: 'Hungría', name_en: 'Hungary' },
  { code: 'ID', name_es: 'Indonesia', name_en: 'Indonesia' },
  { code: 'IE', name_es: 'Irlanda', name_en: 'Ireland' },
  { code: 'IL', name_es: 'Israel', name_en: 'Israel' },
  { code: 'IM', name_es: 'Isla de Man', name_en: 'Isle of Man' },
  { code: 'IN', name_es: 'India', name_en: 'India' },
  { code: 'IO', name_es: 'Territorio Británico del Océano Índico', name_en: 'British Indian Ocean Territory' },
  { code: 'IQ', name_es: 'Irak', name_en: 'Iraq' },
  { code: 'IR', name_es: 'Irán', name_en: 'Iran' },
  { code: 'IS', name_es: 'Islandia', name_en: 'Iceland' },
  { code: 'IT', name_es: 'Italia', name_en: 'Italy' },
  { code: 'JE', name_es: 'Jersey', name_en: 'Jersey' },
  { code: 'JM', name_es: 'Jamaica', name_en: 'Jamaica' },
  { code: 'JO', name_es: 'Jordania', name_en: 'Jordan' },
  { code: 'JP', name_es: 'Japón', name_en: 'Japan' },
  { code: 'KE', name_es: 'Kenia', name_en: 'Kenya' },
  { code: 'KG', name_es: 'Kirguistán', name_en: 'Kyrgyzstan' },
  { code: 'KH', name_es: 'Camboya', name_en: 'Cambodia' },
  { code: 'KI', name_es: 'Kiribati', name_en: 'Kiribati' },
  { code: 'KM', name_es: 'Comoras', name_en: 'Comoros' },
  { code: 'KN', name_es: 'San Cristóbal y Nieves', name_en: 'Saint Kitts and Nevis' },
  { code: 'KP', name_es: 'Corea del Norte', name_en: 'North Korea' },
  { code: 'KR', name_es: 'Corea del Sur', name_en: 'South Korea' },
  { code: 'KW', name_es: 'Kuwait', name_en: 'Kuwait' },
  { code: 'KY', name_es: 'Islas Caimán', name_en: 'Cayman Islands' },
  { code: 'KZ', name_es: 'Kazajistán', name_en: 'Kazakhstan' },
  { code: 'LA', name_es: 'Laos', name_en: 'Laos' },
  { code: 'LB', name_es: 'Líbano', name_en: 'Lebanon' },
  { code: 'LC', name_es: 'Santa Lucía', name_en: 'Saint Lucia' },
  { code: 'LI', name_es: 'Liechtenstein', name_en: 'Liechtenstein' },
  { code: 'LK', name_es: 'Sri Lanka', name_en: 'Sri Lanka' },
  { code: 'LR', name_es: 'Liberia', name_en: 'Liberia' },
  { code: 'LS', name_es: 'Lesoto', name_en: 'Lesotho' },
  { code: 'LT', name_es: 'Lituania', name_en: 'Lithuania' },
  { code: 'LU', name_es: 'Luxemburgo', name_en: 'Luxembourg' },
  { code: 'LV', name_es: 'Letonia', name_en: 'Latvia' },
  { code: 'LY', name_es: 'Libia', name_en: 'Libya' },
  { code: 'MA', name_es: 'Marruecos', name_en: 'Morocco' },
  { code: 'MC', name_es: 'Mónaco', name_en: 'Monaco' },
  { code: 'MD', name_es: 'Moldavia', name_en: 'Moldova' },
  { code: 'ME', name_es: 'Montenegro', name_en: 'Montenegro' },
  { code: 'MF', name_es: 'San Martín', name_en: 'Saint Martin' },
  { code: 'MG', name_es: 'Madagascar', name_en: 'Madagascar' },
  { code: 'MH', name_es: 'Islas Marshall', name_en: 'Marshall Islands' },
  { code: 'MK', name_es: 'Macedonia del Norte', name_en: 'North Macedonia' },
  { code: 'ML', name_es: 'Malí', name_en: 'Mali' },
  { code: 'MM', name_es: 'Birmania', name_en: 'Myanmar' },
  { code: 'MN', name_es: 'Mongolia', name_en: 'Mongolia' },
  { code: 'MO', name_es: 'Macao', name_en: 'Macao' },
  { code: 'MP', name_es: 'Islas Marianas del Norte', name_en: 'Northern Mariana Islands' },
  { code: 'MQ', name_es: 'Martinica', name_en: 'Martinique' },
  { code: 'MR', name_es: 'Mauritania', name_en: 'Mauritania' },
  { code: 'MS', name_es: 'Montserrat', name_en: 'Montserrat' },
  { code: 'MT', name_es: 'Malta', name_en: 'Malta' },
  { code: 'MU', name_es: 'Mauricio', name_en: 'Mauritius' },
  { code: 'MV', name_es: 'Maldivas', name_en: 'Maldives' },
  { code: 'MW', name_es: 'Malaui', name_en: 'Malawi' },
  { code: 'MX', name_es: 'México', name_en: 'Mexico' },
  { code: 'MY', name_es: 'Malasia', name_en: 'Malaysia' },
  { code: 'MZ', name_es: 'Mozambique', name_en: 'Mozambique' },
  { code: 'NA', name_es: 'Namibia', name_en: 'Namibia' },
  { code: 'NC', name_es: 'Nueva Caledonia', name_en: 'New Caledonia' },
  { code: 'NE', name_es: 'Níger', name_en: 'Niger' },
  { code: 'NF', name_es: 'Isla Norfolk', name_en: 'Norfolk Island' },
  { code: 'NG', name_es: 'Nigeria', name_en: 'Nigeria' },
  { code: 'NI', name_es: 'Nicaragua', name_en: 'Nicaragua' },
  { code: 'NL', name_es: 'Países Bajos', name_en: 'Netherlands' },
  { code: 'NO', name_es: 'Noruega', name_en: 'Norway' },
  { code: 'NP', name_es: 'Nepal', name_en: 'Nepal' },
  { code: 'NR', name_es: 'Nauru', name_en: 'Nauru' },
  { code: 'NU', name_es: 'Niue', name_en: 'Niue' },
  { code: 'NZ', name_es: 'Nueva Zelanda', name_en: 'New Zealand' },
  { code: 'OM', name_es: 'Omán', name_en: 'Oman' },
  { code: 'PA', name_es: 'Panamá', name_en: 'Panama' },
  { code: 'PE', name_es: 'Perú', name_en: 'Peru' },
  { code: 'PF', name_es: 'Polinesia Francesa', name_en: 'French Polynesia' },
  { code: 'PG', name_es: 'Papúa Nueva Guinea', name_en: 'Papua New Guinea' },
  { code: 'PH', name_es: 'Filipinas', name_en: 'Philippines' },
  { code: 'PK', name_es: 'Pakistán', name_en: 'Pakistan' },
  { code: 'PL', name_es: 'Polonia', name_en: 'Poland' },
  { code: 'PM', name_es: 'San Pedro y Miquelón', name_en: 'Saint Pierre and Miquelon' },
  { code: 'PN', name_es: 'Islas Pitcairn', name_en: 'Pitcairn Islands' },
  { code: 'PR', name_es: 'Puerto Rico', name_en: 'Puerto Rico' },
  { code: 'PS', name_es: 'Palestina', name_en: 'Palestine' },
  { code: 'PT', name_es: 'Portugal', name_en: 'Portugal' },
  { code: 'PW', name_es: 'Palaos', name_en: 'Palau' },
  { code: 'PY', name_es: 'Paraguay', name_en: 'Paraguay' },
  { code: 'QA', name_es: 'Catar', name_en: 'Qatar' },
  { code: 'RE', name_es: 'Reunión', name_en: 'Réunion' },
  { code: 'RO', name_es: 'Rumania', name_en: 'Romania' },
  { code: 'RS', name_es: 'Serbia', name_en: 'Serbia' },
  { code: 'RU', name_es: 'Rusia', name_en: 'Russia' },
  { code: 'RW', name_es: 'Ruanda', name_en: 'Rwanda' },
  { code: 'SA', name_es: 'Arabia Saudita', name_en: 'Saudi Arabia' },
  { code: 'SB', name_es: 'Islas Salomón', name_en: 'Solomon Islands' },
  { code: 'SC', name_es: 'Seychelles', name_en: 'Seychelles' },
  { code: 'SD', name_es: 'Sudán', name_en: 'Sudan' },
  { code: 'SE', name_es: 'Suecia', name_en: 'Sweden' },
  { code: 'SG', name_es: 'Singapur', name_en: 'Singapore' },
  { code: 'SH', name_es: 'Santa Elena', name_en: 'Saint Helena' },
  { code: 'SI', name_es: 'Eslovenia', name_en: 'Slovenia' },
  { code: 'SJ', name_es: 'Svalbard y Jan Mayen', name_en: 'Svalbard and Jan Mayen' },
  { code: 'SK', name_es: 'Eslovaquia', name_en: 'Slovakia' },
  { code: 'SL', name_es: 'Sierra Leona', name_en: 'Sierra Leone' },
  { code: 'SM', name_es: 'San Marino', name_en: 'San Marino' },
  { code: 'SN', name_es: 'Senegal', name_en: 'Senegal' },
  { code: 'SO', name_es: 'Somalia', name_en: 'Somalia' },
  { code: 'SR', name_es: 'Surinam', name_en: 'Suriname' },
  { code: 'SS', name_es: 'Sudán del Sur', name_en: 'South Sudan' },
  { code: 'ST', name_es: 'Santo Tomé y Príncipe', name_en: 'São Tomé and Príncipe' },
  { code: 'SV', name_es: 'El Salvador', name_en: 'El Salvador' },
  { code: 'SX', name_es: 'Sint Maarten', name_en: 'Sint Maarten' },
  { code: 'SY', name_es: 'Siria', name_en: 'Syria' },
  { code: 'SZ', name_es: 'Esuatini', name_en: 'Eswatini' },
  { code: 'TC', name_es: 'Islas Turcas y Caicos', name_en: 'Turks and Caicos Islands' },
  { code: 'TD', name_es: 'Chad', name_en: 'Chad' },
  { code: 'TF', name_es: 'Tierras Australes Francesas', name_en: 'French Southern Territories' },
  { code: 'TG', name_es: 'Togo', name_en: 'Togo' },
  { code: 'TH', name_es: 'Tailandia', name_en: 'Thailand' },
  { code: 'TJ', name_es: 'Tayikistán', name_en: 'Tajikistan' },
  { code: 'TK', name_es: 'Tokelau', name_en: 'Tokelau' },
  { code: 'TL', name_es: 'Timor-Leste', name_en: 'Timor-Leste' },
  { code: 'TM', name_es: 'Turkmenistán', name_en: 'Turkmenistan' },
  { code: 'TN', name_es: 'Túnez', name_en: 'Tunisia' },
  { code: 'TO', name_es: 'Tonga', name_en: 'Tonga' },
  { code: 'TR', name_es: 'Turquía', name_en: 'Turkey' },
  { code: 'TT', name_es: 'Trinidad y Tobago', name_en: 'Trinidad and Tobago' },
  { code: 'TV', name_es: 'Tuvalu', name_en: 'Tuvalu' },
  { code: 'TW', name_es: 'Taiwán', name_en: 'Taiwan' },
  { code: 'TZ', name_es: 'Tanzania', name_en: 'Tanzania' },
  { code: 'UA', name_es: 'Ucrania', name_en: 'Ukraine' },
  { code: 'UG', name_es: 'Uganda', name_en: 'Uganda' },
  { code: 'UM', name_es: 'Islas Ultramarinas Menores de Estados Unidos', name_en: 'United States Minor Outlying Islands' },
  { code: 'US', name_es: 'Estados Unidos', name_en: 'United States' },
  { code: 'UY', name_es: 'Uruguay', name_en: 'Uruguay' },
  { code: 'UZ', name_es: 'Uzbekistán', name_en: 'Uzbekistan' },
  { code: 'VA', name_es: 'Ciudad del Vaticano', name_en: 'Vatican City' },
  { code: 'VC', name_es: 'San Vicente y las Granadinas', name_en: 'Saint Vincent and the Grenadines' },
  { code: 'VE', name_es: 'Venezuela', name_en: 'Venezuela' },
  { code: 'VG', name_es: 'Islas Vírgenes Británicas', name_en: 'British Virgin Islands' },
  { code: 'VI', name_es: 'Islas Vírgenes de los Estados Unidos', name_en: 'U.S. Virgin Islands' },
  { code: 'VN', name_es: 'Vietnam', name_en: 'Vietnam' },
  { code: 'VU', name_es: 'Vanuatu', name_en: 'Vanuatu' },
  { code: 'WF', name_es: 'Wallis y Futuna', name_en: 'Wallis and Futuna' },
  { code: 'WS', name_es: 'Samoa', name_en: 'Samoa' },
  { code: 'YE', name_es: 'Yemen', name_en: 'Yemen' },
  { code: 'YT', name_es: 'Mayotte', name_en: 'Mayotte' },
  { code: 'ZA', name_es: 'Sudáfrica', name_en: 'South Africa' },
  { code: 'ZM', name_es: 'Zambia', name_en: 'Zambia' },
  { code: 'ZW', name_es: 'Zimbabue', name_en: 'Zimbabwe' },
]

// normalizeText quita acentos y mayúsculas para resolver nombres legacy y
// evitar que "Nicaragua" o "nicaragua " queden sin match.
function normalizeText(value: string): string {
  return value
    .normalize('NFD')
    .replace(/[\u0300-\u036f]/g, '')
    .trim()
    .toLowerCase()
}

export const COUNTRIES: Country[] = [...COUNTRY_DATA].sort((a, b) =>
  a.name_es.localeCompare(b.name_es, 'es'),
)

const byCode = new Map<string, Country>(COUNTRIES.map((c) => [c.code, c]))
const byName = new Map<string, string>()
for (const country of COUNTRIES) {
  byName.set(normalizeText(country.name_es), country.code)
  byName.set(normalizeText(country.name_en), country.code)
}

/** Resuelve un código ISO, un nombre en español o un nombre en inglés. */
export function resolveCountryCode(raw: string): string | null {
  const value = (raw || '').trim()
  if (!value) return null

  const upper = value.toUpperCase()
  if (upper.length === 2 && byCode.has(upper)) return upper

  return byName.get(normalizeText(value)) ?? null
}

/** Devuelve el nombre localizado o el valor original si no se puede resolver. */
export function countryName(value: string, lang: string): string {
  const code = resolveCountryCode(value)
  if (!code) return value
  const country = byCode.get(code)
  if (!country) return value
  return lang === 'en' ? country.name_en : country.name_es
}

/** Opciones para el componente Select, ordenadas por nombre en español. */
export function countryOptions(lang: string): { value: string; label: string }[] {
  return COUNTRIES.map((country) => ({
    value: country.code,
    label: lang === 'en' ? country.name_en : country.name_es,
  }))
}
