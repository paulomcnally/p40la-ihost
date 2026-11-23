-- Solo elimina la categoría si ninguna institución la usa.
DELETE FROM institution_categories
WHERE key = 'real_estate'
  AND NOT EXISTS (
      SELECT 1 FROM institutions WHERE category_id = institution_categories.id
  );
