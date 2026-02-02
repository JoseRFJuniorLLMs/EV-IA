-- Migration: V2G (Vehicle-to-Grid) Tables
-- Created: 2026-02-02
-- Description: Creates tables for V2G sessions, preferences, events, and compensation

-- Enable UUID extension if not exists
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ================================================
-- V2G Sessions Table
-- ================================================
CREATE TABLE IF NOT EXISTS v2g_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    transaction_id VARCHAR(100),
    charge_point_id VARCHAR(100) NOT NULL,
    connector_id INTEGER NOT NULL DEFAULT 1,
    user_id UUID NOT NULL,
    vehicle_id VARCHAR(100),

    -- Direction and power
    direction VARCHAR(20) NOT NULL DEFAULT 'Idle', -- Charging, Discharging, Idle
    requested_power_kw DECIMAL(10,2) NOT NULL DEFAULT 0,
    actual_power_kw DECIMAL(10,2) NOT NULL DEFAULT 0,
    energy_transferred DECIMAL(12,4) NOT NULL DEFAULT 0, -- kWh (negative = discharge)

    -- Pricing
    grid_price_at_start DECIMAL(10,4) NOT NULL DEFAULT 0, -- R$/kWh
    current_grid_price DECIMAL(10,4) NOT NULL DEFAULT 0,
    user_compensation DECIMAL(12,4) NOT NULL DEFAULT 0, -- R$ to pay user

    -- Battery constraints
    min_battery_soc INTEGER NOT NULL DEFAULT 20, -- Minimum SOC to maintain (%)
    current_soc INTEGER NOT NULL DEFAULT 0, -- Current battery SOC (%)

    -- Timing
    start_time TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    end_time TIMESTAMP WITH TIME ZONE,

    -- Status
    status VARCHAR(20) NOT NULL DEFAULT 'Pending', -- Pending, Active, Completed, Failed, Cancelled

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Foreign keys
    CONSTRAINT fk_v2g_session_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Indexes for v2g_sessions
CREATE INDEX IF NOT EXISTS idx_v2g_sessions_user_id ON v2g_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_v2g_sessions_charge_point_id ON v2g_sessions(charge_point_id);
CREATE INDEX IF NOT EXISTS idx_v2g_sessions_status ON v2g_sessions(status);
CREATE INDEX IF NOT EXISTS idx_v2g_sessions_created_at ON v2g_sessions(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_v2g_sessions_transaction_id ON v2g_sessions(transaction_id) WHERE transaction_id IS NOT NULL;

-- ================================================
-- V2G User Preferences Table
-- ================================================
CREATE TABLE IF NOT EXISTS v2g_preferences (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL UNIQUE,

    -- Auto discharge settings
    auto_discharge BOOLEAN NOT NULL DEFAULT FALSE,
    min_grid_price DECIMAL(10,4) NOT NULL DEFAULT 0.80, -- R$/kWh minimum to accept V2G
    max_discharge_kwh DECIMAL(10,2) NOT NULL DEFAULT 50.0, -- Max kWh per day
    preserve_soc INTEGER NOT NULL DEFAULT 20, -- Minimum SOC to preserve (%)

    -- Notification settings
    notify_on_start BOOLEAN NOT NULL DEFAULT TRUE,
    notify_on_end BOOLEAN NOT NULL DEFAULT TRUE,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Foreign keys
    CONSTRAINT fk_v2g_preferences_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- ================================================
-- V2G Events Table (Audit/Analytics)
-- ================================================
CREATE TABLE IF NOT EXISTS v2g_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    session_id UUID NOT NULL,
    charge_point_id VARCHAR(100) NOT NULL,

    -- Event details
    event_type VARCHAR(50) NOT NULL, -- started, updated, completed, failed, compensated, etc.
    direction VARCHAR(20) NOT NULL,
    power_kw DECIMAL(10,2) NOT NULL DEFAULT 0,
    energy_kwh DECIMAL(12,4) NOT NULL DEFAULT 0,
    grid_price DECIMAL(10,4) NOT NULL DEFAULT 0,

    -- Additional data
    details JSONB, -- Additional event-specific data

    -- Timestamp
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Foreign keys
    CONSTRAINT fk_v2g_event_session FOREIGN KEY (session_id) REFERENCES v2g_sessions(id) ON DELETE CASCADE
);

-- Indexes for v2g_events
CREATE INDEX IF NOT EXISTS idx_v2g_events_session_id ON v2g_events(session_id);
CREATE INDEX IF NOT EXISTS idx_v2g_events_timestamp ON v2g_events(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_v2g_events_event_type ON v2g_events(event_type);

-- ================================================
-- V2G Compensations Table
-- ================================================
CREATE TABLE IF NOT EXISTS v2g_compensations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    session_id UUID NOT NULL,
    user_id UUID NOT NULL,

    -- Compensation details
    energy_discharged_kwh DECIMAL(12,4) NOT NULL DEFAULT 0,
    average_grid_price DECIMAL(10,4) NOT NULL DEFAULT 0,
    operator_margin DECIMAL(5,4) NOT NULL DEFAULT 0.10, -- 10%
    gross_amount DECIMAL(12,4) NOT NULL DEFAULT 0,
    net_amount DECIMAL(12,4) NOT NULL DEFAULT 0, -- Amount to pay user
    currency VARCHAR(3) NOT NULL DEFAULT 'BRL',

    -- Payment status
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending, processed, paid, failed
    payment_id VARCHAR(100), -- Reference to payment system
    paid_at TIMESTAMP WITH TIME ZONE,

    -- Timestamps
    calculated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Foreign keys
    CONSTRAINT fk_v2g_compensation_session FOREIGN KEY (session_id) REFERENCES v2g_sessions(id) ON DELETE CASCADE,
    CONSTRAINT fk_v2g_compensation_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Indexes for v2g_compensations
CREATE INDEX IF NOT EXISTS idx_v2g_compensations_user_id ON v2g_compensations(user_id);
CREATE INDEX IF NOT EXISTS idx_v2g_compensations_session_id ON v2g_compensations(session_id);
CREATE INDEX IF NOT EXISTS idx_v2g_compensations_status ON v2g_compensations(status);

-- ================================================
-- V2G Grid Prices Cache Table
-- ================================================
CREATE TABLE IF NOT EXISTS v2g_grid_prices (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- Price data
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    price DECIMAL(10,4) NOT NULL, -- R$/kWh
    is_peak BOOLEAN NOT NULL DEFAULT FALSE,
    source VARCHAR(50) NOT NULL DEFAULT 'simulated', -- ccee, simulated, custom
    region VARCHAR(10) DEFAULT 'SE/CO', -- Brazilian regions

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Unique constraint on timestamp + region
    CONSTRAINT uk_v2g_grid_prices_timestamp_region UNIQUE (timestamp, region)
);

-- Indexes for v2g_grid_prices
CREATE INDEX IF NOT EXISTS idx_v2g_grid_prices_timestamp ON v2g_grid_prices(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_v2g_grid_prices_region ON v2g_grid_prices(region);

-- ================================================
-- V2G Capabilities Cache Table
-- ================================================
CREATE TABLE IF NOT EXISTS v2g_capabilities (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    charge_point_id VARCHAR(100) NOT NULL,
    connector_id INTEGER NOT NULL DEFAULT 1,

    -- Capability details
    supported BOOLEAN NOT NULL DEFAULT FALSE,
    max_discharge_power_kw DECIMAL(10,2) NOT NULL DEFAULT 0,
    max_discharge_current DECIMAL(10,2) NOT NULL DEFAULT 0,
    bidirectional_charging BOOLEAN NOT NULL DEFAULT FALSE,
    iso15118_support BOOLEAN NOT NULL DEFAULT FALSE,

    -- Current state
    current_soc INTEGER NOT NULL DEFAULT 0,
    battery_capacity_kwh DECIMAL(10,2) NOT NULL DEFAULT 0,

    -- Timestamps
    last_updated TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Unique constraint
    CONSTRAINT uk_v2g_capabilities_cp_connector UNIQUE (charge_point_id, connector_id)
);

-- Indexes for v2g_capabilities
CREATE INDEX IF NOT EXISTS idx_v2g_capabilities_charge_point_id ON v2g_capabilities(charge_point_id);
CREATE INDEX IF NOT EXISTS idx_v2g_capabilities_supported ON v2g_capabilities(supported) WHERE supported = TRUE;

-- ================================================
-- ISO 15118 Certificates Table
-- ================================================
CREATE TABLE IF NOT EXISTS iso15118_certificates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- Certificate identity
    emaid VARCHAR(100) NOT NULL, -- E-Mobility Account Identifier
    contract_id VARCHAR(100) NOT NULL,
    vehicle_vin VARCHAR(50),

    -- Certificate data
    certificate_pem TEXT NOT NULL,
    certificate_chain TEXT,
    private_key_encrypted TEXT, -- Encrypted with system key

    -- V2G capability
    v2g_capable BOOLEAN NOT NULL DEFAULT FALSE,

    -- Validity
    valid_from TIMESTAMP WITH TIME ZONE NOT NULL,
    valid_to TIMESTAMP WITH TIME ZONE NOT NULL,
    revoked BOOLEAN NOT NULL DEFAULT FALSE,
    revoked_at TIMESTAMP WITH TIME ZONE,
    revocation_reason VARCHAR(200),

    -- Contract details
    provider_id VARCHAR(50),
    max_charge_power_kw DECIMAL(10,2),
    max_discharge_power_kw DECIMAL(10,2),

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Unique constraints
    CONSTRAINT uk_iso15118_emaid UNIQUE (emaid),
    CONSTRAINT uk_iso15118_contract_id UNIQUE (contract_id)
);

-- Indexes for iso15118_certificates
CREATE INDEX IF NOT EXISTS idx_iso15118_certificates_emaid ON iso15118_certificates(emaid);
CREATE INDEX IF NOT EXISTS idx_iso15118_certificates_vehicle_vin ON iso15118_certificates(vehicle_vin) WHERE vehicle_vin IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_iso15118_certificates_v2g ON iso15118_certificates(v2g_capable) WHERE v2g_capable = TRUE;
CREATE INDEX IF NOT EXISTS idx_iso15118_certificates_valid ON iso15118_certificates(valid_to) WHERE revoked = FALSE;

-- ================================================
-- Firmware Updates Table
-- ================================================
CREATE TABLE IF NOT EXISTS firmware_updates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    charge_point_id VARCHAR(100) NOT NULL,
    request_id INTEGER NOT NULL,

    -- Firmware details
    firmware_url TEXT NOT NULL,
    version VARCHAR(50),
    retrieve_datetime TIMESTAMP WITH TIME ZONE NOT NULL,
    install_datetime TIMESTAMP WITH TIME ZONE,

    -- Status
    status VARCHAR(50) NOT NULL DEFAULT 'Idle', -- Idle, Downloading, Downloaded, Installing, Installed, etc.
    progress INTEGER NOT NULL DEFAULT 0, -- 0-100%
    error_message TEXT,

    -- Retry settings
    retries INTEGER NOT NULL DEFAULT 0,
    max_retries INTEGER NOT NULL DEFAULT 3,
    retry_interval INTEGER NOT NULL DEFAULT 60, -- seconds

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE
);

-- Indexes for firmware_updates
CREATE INDEX IF NOT EXISTS idx_firmware_updates_charge_point_id ON firmware_updates(charge_point_id);
CREATE INDEX IF NOT EXISTS idx_firmware_updates_status ON firmware_updates(status);
CREATE INDEX IF NOT EXISTS idx_firmware_updates_created_at ON firmware_updates(created_at DESC);

-- ================================================
-- Update trigger for updated_at columns
-- ================================================
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply triggers
DROP TRIGGER IF EXISTS update_v2g_sessions_updated_at ON v2g_sessions;
CREATE TRIGGER update_v2g_sessions_updated_at
    BEFORE UPDATE ON v2g_sessions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_v2g_preferences_updated_at ON v2g_preferences;
CREATE TRIGGER update_v2g_preferences_updated_at
    BEFORE UPDATE ON v2g_preferences
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_v2g_compensations_updated_at ON v2g_compensations;
CREATE TRIGGER update_v2g_compensations_updated_at
    BEFORE UPDATE ON v2g_compensations
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_iso15118_certificates_updated_at ON iso15118_certificates;
CREATE TRIGGER update_iso15118_certificates_updated_at
    BEFORE UPDATE ON iso15118_certificates
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_firmware_updates_updated_at ON firmware_updates;
CREATE TRIGGER update_firmware_updates_updated_at
    BEFORE UPDATE ON firmware_updates
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ================================================
-- Comments
-- ================================================
COMMENT ON TABLE v2g_sessions IS 'V2G discharge/charge sessions between vehicles and grid';
COMMENT ON TABLE v2g_preferences IS 'User preferences for automatic V2G operations';
COMMENT ON TABLE v2g_events IS 'Audit trail of V2G events for analytics';
COMMENT ON TABLE v2g_compensations IS 'Compensation calculations and payments for V2G';
COMMENT ON TABLE v2g_grid_prices IS 'Cache of grid electricity prices';
COMMENT ON TABLE v2g_capabilities IS 'Cache of EV V2G capabilities detected via OCPP';
COMMENT ON TABLE iso15118_certificates IS 'ISO 15118 Plug & Charge certificates';
COMMENT ON TABLE firmware_updates IS 'Firmware update tracking for charge points';
