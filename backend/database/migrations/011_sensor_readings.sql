-- Add sensor_readings JSONB column to metrics table
-- Stores all temperature sensor readings as an array of {key, temperature_celsius} objects
ALTER TABLE metrics ADD COLUMN IF NOT EXISTS sensor_readings JSONB;
