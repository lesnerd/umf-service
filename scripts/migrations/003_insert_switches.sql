-- Migration: 003_insert_switches.sql
-- Description: Insert switch records for the telemetry generator
-- Date: 2024-01-01

-- Insert switch records for switches 001-010
INSERT INTO switches (id, name, location) VALUES
    ('switch-001', 'Switch 001', 'Data Center A'),
    ('switch-002', 'Switch 002', 'Data Center A'),
    ('switch-003', 'Switch 003', 'Data Center A'),
    ('switch-004', 'Switch 004', 'Data Center A'),
    ('switch-005', 'Switch 005', 'Data Center A'),
    ('switch-006', 'Switch 006', 'Data Center B'),
    ('switch-007', 'Switch 007', 'Data Center B'),
    ('switch-008', 'Switch 008', 'Data Center B'),
    ('switch-009', 'Switch 009', 'Data Center B'),
    ('switch-010', 'Switch 010', 'Data Center B')
ON CONFLICT (id) DO NOTHING; 