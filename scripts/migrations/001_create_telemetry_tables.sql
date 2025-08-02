-- Migration: 001_create_telemetry_tables.sql
-- Description: Create tables for telemetry data storage
-- Date: 2024-01-01

-- Create switches table
CREATE TABLE IF NOT EXISTS switches (
    id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    location VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create telemetry_metrics table
CREATE TABLE IF NOT EXISTS telemetry_metrics (
    id BIGSERIAL PRIMARY KEY,
    switch_id VARCHAR(50) NOT NULL REFERENCES switches(id) ON DELETE CASCADE,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    bandwidth_mbps DECIMAL(10,2) NOT NULL DEFAULT 0,
    latency_ms DECIMAL(8,3) NOT NULL DEFAULT 0,
    packet_errors BIGINT NOT NULL DEFAULT 0,
    utilization_pct DECIMAL(5,2) NOT NULL DEFAULT 0,
    temperature_c DECIMAL(5,2) NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_telemetry_switch_id ON telemetry_metrics(switch_id);
CREATE INDEX IF NOT EXISTS idx_telemetry_timestamp ON telemetry_metrics(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_telemetry_switch_time ON telemetry_metrics(switch_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_telemetry_created_at ON telemetry_metrics(created_at DESC);

-- Add constraints
ALTER TABLE telemetry_metrics 
ADD CONSTRAINT chk_bandwidth_positive CHECK (bandwidth_mbps >= 0);

ALTER TABLE telemetry_metrics 
ADD CONSTRAINT chk_latency_positive CHECK (latency_ms >= 0);

ALTER TABLE telemetry_metrics 
ADD CONSTRAINT chk_packet_errors_positive CHECK (packet_errors >= 0);

ALTER TABLE telemetry_metrics 
ADD CONSTRAINT chk_utilization_range CHECK (utilization_pct >= 0 AND utilization_pct <= 100);

ALTER TABLE telemetry_metrics 
ADD CONSTRAINT chk_temperature_range CHECK (temperature_c >= -50 AND temperature_c <= 150);



