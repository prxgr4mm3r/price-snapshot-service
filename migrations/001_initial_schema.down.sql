-- Crypto Snapshot Service - Rollback Initial Schema
-- Drops all tables and functions created in the up migration

-- Drop triggers first
DROP TRIGGER IF EXISTS update_symbols_updated_at ON symbols;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop tables (snapshots first due to foreign key constraint)
DROP TABLE IF EXISTS snapshots;
DROP TABLE IF EXISTS symbols;
