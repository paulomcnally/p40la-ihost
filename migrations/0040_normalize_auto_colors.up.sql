-- SPEC-118: normaliza los colores existentes de autos a la paleta canónica.
-- Mapea nombres conocidos en español/inglés (sin distinguir mayúsculas ni
-- espacios) y deja vacío ('') cualquier valor no reconocido para que el
-- usuario elija un color al editar. Los códigos canónicos exactos no se tocan,
-- por lo que la migración es idempotente.

UPDATE autos
SET color = CASE lower(trim(color))
    WHEN 'blanco'       THEN 'white'
    WHEN 'white'        THEN 'white'
    WHEN 'negro'        THEN 'black'
    WHEN 'black'        THEN 'black'
    WHEN 'gris'         THEN 'gray'
    WHEN 'gray'         THEN 'gray'
    WHEN 'grey'         THEN 'gray'
    WHEN 'plata'        THEN 'silver'
    WHEN 'plateado'     THEN 'silver'
    WHEN 'silver'       THEN 'silver'
    WHEN 'rojo'         THEN 'red'
    WHEN 'red'          THEN 'red'
    WHEN 'azul'         THEN 'blue'
    WHEN 'blue'         THEN 'blue'
    WHEN 'verde'        THEN 'green'
    WHEN 'green'        THEN 'green'
    WHEN 'amarillo'     THEN 'yellow'
    WHEN 'yellow'       THEN 'yellow'
    WHEN 'naranja'      THEN 'orange'
    WHEN 'orange'       THEN 'orange'
    WHEN 'marrón'       THEN 'brown'
    WHEN 'marrÓn'       THEN 'brown'
    WHEN 'marron'       THEN 'brown'
    WHEN 'brown'        THEN 'brown'
    WHEN 'beige'        THEN 'beige'
    WHEN 'dorado'       THEN 'gold'
    WHEN 'gold'         THEN 'gold'
    WHEN 'vino'         THEN 'burgundy'
    WHEN 'burgundy'     THEN 'burgundy'
    WHEN 'celeste'      THEN 'light_blue'
    WHEN 'light blue'   THEN 'light_blue'
    WHEN 'light-blue'   THEN 'light_blue'
    WHEN 'light_blue'   THEN 'light_blue'
    WHEN 'rosado'       THEN 'pink'
    WHEN 'pink'         THEN 'pink'
    WHEN 'morado'       THEN 'purple'
    WHEN 'purple'       THEN 'purple'
    ELSE ''
END
WHERE trim(color) <> ''
  AND color NOT IN (
    'white', 'black', 'gray', 'silver',
    'red', 'blue', 'green', 'yellow',
    'orange', 'brown', 'beige', 'gold',
    'burgundy', 'light_blue', 'pink', 'purple'
  );
