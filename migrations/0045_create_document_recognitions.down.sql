-- SPEC-127: revierte la tabla de reconocimientos de comprobantes.

DROP INDEX IF EXISTS idx_document_recognitions_entity;
DROP INDEX IF EXISTS idx_document_recognitions_reference;
DROP INDEX IF EXISTS idx_document_recognitions_status;
DROP TABLE IF EXISTS document_recognitions;
