-- SPEC-118: revierte los códigos canónicos a etiquetas en español.
-- Best-effort: los valores que la migración up limpió ('') no son
-- recuperables; el rollback real es el backup previo de la DB.

UPDATE autos
SET color = CASE color
    WHEN 'white'      THEN 'Blanco'
    WHEN 'black'      THEN 'Negro'
    WHEN 'gray'       THEN 'Gris'
    WHEN 'silver'     THEN 'Plata'
    WHEN 'red'        THEN 'Rojo'
    WHEN 'blue'       THEN 'Azul'
    WHEN 'green'      THEN 'Verde'
    WHEN 'yellow'     THEN 'Amarillo'
    WHEN 'orange'     THEN 'Naranja'
    WHEN 'brown'      THEN 'Marrón'
    WHEN 'beige'      THEN 'Beige'
    WHEN 'gold'       THEN 'Dorado'
    WHEN 'burgundy'   THEN 'Vino'
    WHEN 'light_blue' THEN 'Celeste'
    WHEN 'pink'       THEN 'Rosado'
    WHEN 'purple'     THEN 'Morado'
    ELSE color
END
WHERE color IN (
    'white', 'black', 'gray', 'silver',
    'red', 'blue', 'green', 'yellow',
    'orange', 'brown', 'beige', 'gold',
    'burgundy', 'light_blue', 'pink', 'purple'
);
