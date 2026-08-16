CREATE TABLE IF NOT EXISTS institution_categories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    key TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT DEFAULT '',
    icon_key TEXT DEFAULT 'other',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO institution_categories (key, name, description, icon_key) VALUES
    ('insurance', 'Seguros', 'Pólizas de seguro (auto, hogar, vida, salud)', 'shield'),
    ('telecommunications', 'Telecomunicaciones', 'Telefonía celular, fija, internet', 'signal'),
    ('pay_tv', 'TV por Cable', 'Cable, satelital, IPTV', 'monitor'),
    ('electricity', 'Electricidad', 'Empresas de energía eléctrica', 'bolt'),
    ('natural_gas', 'Gas Natural', 'Distribuidoras de gas', 'flame'),
    ('water', 'Agua', 'Servicios de agua potable', 'water'),
    ('waste', 'Basura', 'Recolección de desechos', 'trash'),
    ('banking', 'Banca', 'Cuentas bancarias, tarjetas de crédito', 'bank'),
    ('loans', 'Préstamos', 'Hipotecas, préstamos personales, auto, estudiantiles', 'credit'),
    ('subscriptions', 'Suscripciones', 'Servicios recurrentes (apps, membresías)', 'refresh'),
    ('entertainment', 'Entretenimiento', 'Streaming, cine, eventos, juegos', 'film'),
    ('education', 'Educación', 'Colegios, universidades, cursos, capacitación', 'education'),
    ('health', 'Salud', 'Clínicas, farmacias, consultas médicas', 'health'),
    ('transport', 'Transporte', 'Gasolina, peajes, estacionamiento, mantenimiento vehicular', 'vehicle'),
    ('housing', 'Vivienda', 'Alquiler, condominio, administración de propiedades', 'home'),
    ('security', 'Seguridad', 'Alarmas, vigilancia, sistemas de seguridad', 'lock'),
    ('internet', 'Internet y Hosting', 'Hosting, dominios, servicios en la nube', 'globe'),
    ('digital_services', 'Servicios Digitales', 'Software, almacenamiento, licencias', 'cloud'),
    ('maintenance', 'Mantenimiento', 'Mantenimiento del hogar, jardinería, limpieza', 'wrench'),
    ('professional', 'Servicios Profesionales', 'Legal, contabilidad, consultoría', 'briefcase');

ALTER TABLE institutions ADD COLUMN category_id INTEGER REFERENCES institution_categories(id);
