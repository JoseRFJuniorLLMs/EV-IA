-- Migration: Initial Schema
-- Created: 2026-02-02
-- Description: Creates base tables for SIGEC-VE Enterprise

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ================================================
-- Users Table
-- ================================================
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    password VARCHAR(255) NOT NULL,
    phone VARCHAR(50),
    document VARCHAR(20), -- CPF/CNPJ
    role VARCHAR(20) NOT NULL DEFAULT 'user', -- admin, operator, user
    status VARCHAR(20) NOT NULL DEFAULT 'active', -- active, inactive, blocked
    email_verified BOOLEAN NOT NULL DEFAULT FALSE,
    email_verified_at TIMESTAMP WITH TIME ZONE,
    last_login TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes for users
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);
CREATE INDEX IF NOT EXISTS idx_users_status ON users(status);

-- ================================================
-- Locations Table
-- ================================================
CREATE TABLE IF NOT EXISTS locations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    address VARCHAR(500),
    city VARCHAR(100),
    state VARCHAR(50),
    country VARCHAR(50) DEFAULT 'BR',
    postal_code VARCHAR(20),
    latitude DECIMAL(10,7),
    longitude DECIMAL(10,7),
    timezone VARCHAR(50) DEFAULT 'America/Sao_Paulo',
    operator_id UUID,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_location_operator FOREIGN KEY (operator_id) REFERENCES users(id) ON DELETE SET NULL
);

-- Indexes for locations
CREATE INDEX IF NOT EXISTS idx_locations_operator ON locations(operator_id);
CREATE INDEX IF NOT EXISTS idx_locations_coords ON locations(latitude, longitude);

-- ================================================
-- Charge Points Table
-- ================================================
CREATE TABLE IF NOT EXISTS charge_points (
    id VARCHAR(100) PRIMARY KEY, -- OCPP ChargePointId
    vendor VARCHAR(100),
    model VARCHAR(100),
    serial_number VARCHAR(100),
    firmware_version VARCHAR(50),
    iccid VARCHAR(50),
    imsi VARCHAR(50),

    -- Location
    location_id UUID,

    -- Status
    status VARCHAR(20) NOT NULL DEFAULT 'Unavailable', -- Available, Occupied, Faulted, Unavailable

    -- Connection
    last_heartbeat TIMESTAMP WITH TIME ZONE,
    last_boot TIMESTAMP WITH TIME ZONE,
    is_online BOOLEAN NOT NULL DEFAULT FALSE,

    -- Configuration
    max_power_kw DECIMAL(10,2),

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_charge_point_location FOREIGN KEY (location_id) REFERENCES locations(id) ON DELETE SET NULL
);

-- Indexes for charge_points
CREATE INDEX IF NOT EXISTS idx_charge_points_status ON charge_points(status);
CREATE INDEX IF NOT EXISTS idx_charge_points_location ON charge_points(location_id);
CREATE INDEX IF NOT EXISTS idx_charge_points_online ON charge_points(is_online) WHERE is_online = TRUE;

-- ================================================
-- Connectors Table
-- ================================================
CREATE TABLE IF NOT EXISTS connectors (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    charge_point_id VARCHAR(100) NOT NULL,
    connector_id INTEGER NOT NULL, -- 1-based (OCPP standard)
    type VARCHAR(50) NOT NULL DEFAULT 'Type2', -- CCS, CHAdeMO, Type2, etc.
    status VARCHAR(20) NOT NULL DEFAULT 'Available',
    max_power_kw DECIMAL(10,2),
    max_current_a DECIMAL(10,2),
    max_voltage_v DECIMAL(10,2),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_connector_charge_point FOREIGN KEY (charge_point_id) REFERENCES charge_points(id) ON DELETE CASCADE,
    CONSTRAINT uk_connector_cp_id UNIQUE (charge_point_id, connector_id)
);

-- Indexes for connectors
CREATE INDEX IF NOT EXISTS idx_connectors_charge_point ON connectors(charge_point_id);
CREATE INDEX IF NOT EXISTS idx_connectors_status ON connectors(status);

-- ================================================
-- Transactions Table
-- ================================================
CREATE TABLE IF NOT EXISTS transactions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    transaction_id VARCHAR(100), -- OCPP transaction ID
    charge_point_id VARCHAR(100) NOT NULL,
    connector_id INTEGER NOT NULL DEFAULT 1,
    user_id UUID,
    id_tag VARCHAR(100), -- RFID tag or auth token

    -- Timing
    start_time TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    end_time TIMESTAMP WITH TIME ZONE,

    -- Metering
    meter_start INTEGER NOT NULL DEFAULT 0, -- Wh
    meter_stop INTEGER, -- Wh
    total_energy_wh INTEGER, -- Wh

    -- Billing
    cost DECIMAL(12,4),
    currency VARCHAR(3) DEFAULT 'BRL',
    tariff_id UUID,

    -- Status
    status VARCHAR(20) NOT NULL DEFAULT 'Started', -- Started, Stopped, Completed, Failed
    stop_reason VARCHAR(50),

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_transaction_charge_point FOREIGN KEY (charge_point_id) REFERENCES charge_points(id) ON DELETE CASCADE,
    CONSTRAINT fk_transaction_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
);

-- Indexes for transactions
CREATE INDEX IF NOT EXISTS idx_transactions_charge_point ON transactions(charge_point_id);
CREATE INDEX IF NOT EXISTS idx_transactions_user ON transactions(user_id);
CREATE INDEX IF NOT EXISTS idx_transactions_status ON transactions(status);
CREATE INDEX IF NOT EXISTS idx_transactions_start_time ON transactions(start_time DESC);
CREATE INDEX IF NOT EXISTS idx_transactions_id_tag ON transactions(id_tag);

-- ================================================
-- Meter Values Table
-- ================================================
CREATE TABLE IF NOT EXISTS meter_values (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    transaction_id UUID NOT NULL,
    charge_point_id VARCHAR(100) NOT NULL,
    connector_id INTEGER NOT NULL DEFAULT 1,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    measurand VARCHAR(50) NOT NULL, -- Energy.Active.Import.Register, Power.Active.Import, etc.
    value DECIMAL(15,4) NOT NULL,
    unit VARCHAR(20), -- Wh, kWh, W, kW, A, V, etc.
    phase VARCHAR(10), -- L1, L2, L3, N, L1-N, L2-N, L3-N
    context VARCHAR(20), -- Sample.Periodic, Transaction.Begin, Transaction.End

    CONSTRAINT fk_meter_value_transaction FOREIGN KEY (transaction_id) REFERENCES transactions(id) ON DELETE CASCADE
);

-- Indexes for meter_values
CREATE INDEX IF NOT EXISTS idx_meter_values_transaction ON meter_values(transaction_id);
CREATE INDEX IF NOT EXISTS idx_meter_values_timestamp ON meter_values(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_meter_values_measurand ON meter_values(measurand);

-- ================================================
-- Wallets Table
-- ================================================
CREATE TABLE IF NOT EXISTS wallets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL UNIQUE,
    balance DECIMAL(12,4) NOT NULL DEFAULT 0,
    currency VARCHAR(3) NOT NULL DEFAULT 'BRL',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_wallet_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- ================================================
-- Wallet Transactions Table
-- ================================================
CREATE TABLE IF NOT EXISTS wallet_transactions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    wallet_id UUID NOT NULL,
    user_id UUID NOT NULL,
    type VARCHAR(20) NOT NULL, -- credit, debit
    amount DECIMAL(12,4) NOT NULL,
    balance_after DECIMAL(12,4) NOT NULL,
    description VARCHAR(255),
    reference_id VARCHAR(100), -- payment_id, transaction_id, etc.
    reference_type VARCHAR(50), -- payment, charging, v2g_compensation, refund
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_wallet_tx_wallet FOREIGN KEY (wallet_id) REFERENCES wallets(id) ON DELETE CASCADE,
    CONSTRAINT fk_wallet_tx_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Indexes for wallet_transactions
CREATE INDEX IF NOT EXISTS idx_wallet_tx_wallet ON wallet_transactions(wallet_id);
CREATE INDEX IF NOT EXISTS idx_wallet_tx_user ON wallet_transactions(user_id);
CREATE INDEX IF NOT EXISTS idx_wallet_tx_created ON wallet_transactions(created_at DESC);

-- ================================================
-- Payments Table
-- ================================================
CREATE TABLE IF NOT EXISTS payments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,
    amount DECIMAL(12,4) NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'BRL',
    method VARCHAR(20) NOT NULL, -- card, pix, boleto, wallet
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending, processing, completed, failed, refunded
    provider VARCHAR(50), -- stripe, pagseguro, etc.
    provider_id VARCHAR(100), -- external payment ID
    transaction_id UUID, -- related charging transaction
    description VARCHAR(255),
    metadata JSONB,
    paid_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_payment_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_payment_transaction FOREIGN KEY (transaction_id) REFERENCES transactions(id) ON DELETE SET NULL
);

-- Indexes for payments
CREATE INDEX IF NOT EXISTS idx_payments_user ON payments(user_id);
CREATE INDEX IF NOT EXISTS idx_payments_status ON payments(status);
CREATE INDEX IF NOT EXISTS idx_payments_provider_id ON payments(provider_id);

-- ================================================
-- Reservations Table
-- ================================================
CREATE TABLE IF NOT EXISTS reservations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,
    charge_point_id VARCHAR(100) NOT NULL,
    connector_id INTEGER NOT NULL DEFAULT 1,
    start_time TIMESTAMP WITH TIME ZONE NOT NULL,
    end_time TIMESTAMP WITH TIME ZONE NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending, confirmed, active, completed, cancelled, expired
    reservation_id INTEGER, -- OCPP reservation ID
    notes VARCHAR(500),
    cancelled_at TIMESTAMP WITH TIME ZONE,
    cancellation_reason VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_reservation_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_reservation_charge_point FOREIGN KEY (charge_point_id) REFERENCES charge_points(id) ON DELETE CASCADE
);

-- Indexes for reservations
CREATE INDEX IF NOT EXISTS idx_reservations_user ON reservations(user_id);
CREATE INDEX IF NOT EXISTS idx_reservations_charge_point ON reservations(charge_point_id);
CREATE INDEX IF NOT EXISTS idx_reservations_status ON reservations(status);
CREATE INDEX IF NOT EXISTS idx_reservations_start_time ON reservations(start_time);

-- ================================================
-- Tariffs Table
-- ================================================
CREATE TABLE IF NOT EXISTS tariffs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL,
    description VARCHAR(500),
    price_per_kwh DECIMAL(10,4) NOT NULL, -- R$/kWh
    price_per_minute DECIMAL(10,4), -- R$/min (idle fee)
    start_fee DECIMAL(10,4), -- R$ (connection fee)
    currency VARCHAR(3) NOT NULL DEFAULT 'BRL',
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    valid_from TIMESTAMP WITH TIME ZONE,
    valid_to TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- ================================================
-- OCPP Authorization Tags Table
-- ================================================
CREATE TABLE IF NOT EXISTS authorization_tags (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    id_tag VARCHAR(100) NOT NULL UNIQUE,
    user_id UUID,
    status VARCHAR(20) NOT NULL DEFAULT 'Accepted', -- Accepted, Blocked, Expired, Invalid
    expiry_date TIMESTAMP WITH TIME ZONE,
    parent_id_tag VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_auth_tag_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
);

-- Indexes for authorization_tags
CREATE INDEX IF NOT EXISTS idx_auth_tags_id_tag ON authorization_tags(id_tag);
CREATE INDEX IF NOT EXISTS idx_auth_tags_user ON authorization_tags(user_id);

-- ================================================
-- Alerts Table
-- ================================================
CREATE TABLE IF NOT EXISTS alerts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    type VARCHAR(50) NOT NULL, -- device_offline, low_power, error, maintenance, etc.
    severity VARCHAR(20) NOT NULL DEFAULT 'info', -- info, warning, error, critical
    title VARCHAR(255) NOT NULL,
    message TEXT,
    source VARCHAR(50), -- charge_point, system, payment, etc.
    source_id VARCHAR(100), -- charge_point_id, user_id, etc.
    acknowledged BOOLEAN NOT NULL DEFAULT FALSE,
    acknowledged_by UUID,
    acknowledged_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_alert_acknowledged_by FOREIGN KEY (acknowledged_by) REFERENCES users(id) ON DELETE SET NULL
);

-- Indexes for alerts
CREATE INDEX IF NOT EXISTS idx_alerts_type ON alerts(type);
CREATE INDEX IF NOT EXISTS idx_alerts_severity ON alerts(severity);
CREATE INDEX IF NOT EXISTS idx_alerts_acknowledged ON alerts(acknowledged) WHERE acknowledged = FALSE;
CREATE INDEX IF NOT EXISTS idx_alerts_created ON alerts(created_at DESC);

-- ================================================
-- Update trigger function
-- ================================================
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply triggers to all tables with updated_at
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_locations_updated_at BEFORE UPDATE ON locations FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_charge_points_updated_at BEFORE UPDATE ON charge_points FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_connectors_updated_at BEFORE UPDATE ON connectors FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_transactions_updated_at BEFORE UPDATE ON transactions FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_wallets_updated_at BEFORE UPDATE ON wallets FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_payments_updated_at BEFORE UPDATE ON payments FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_reservations_updated_at BEFORE UPDATE ON reservations FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_tariffs_updated_at BEFORE UPDATE ON tariffs FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_authorization_tags_updated_at BEFORE UPDATE ON authorization_tags FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ================================================
-- Comments
-- ================================================
COMMENT ON TABLE users IS 'System users (drivers, operators, admins)';
COMMENT ON TABLE locations IS 'Physical locations of charging stations';
COMMENT ON TABLE charge_points IS 'OCPP charge points (charging stations)';
COMMENT ON TABLE connectors IS 'Individual connectors on charge points';
COMMENT ON TABLE transactions IS 'Charging transactions (sessions)';
COMMENT ON TABLE meter_values IS 'Energy meter readings during transactions';
COMMENT ON TABLE wallets IS 'User wallets for prepaid balance';
COMMENT ON TABLE wallet_transactions IS 'Wallet credit/debit history';
COMMENT ON TABLE payments IS 'Payment records';
COMMENT ON TABLE reservations IS 'Charging station reservations';
COMMENT ON TABLE tariffs IS 'Pricing tariffs';
COMMENT ON TABLE authorization_tags IS 'RFID/Auth tags for OCPP authorization';
COMMENT ON TABLE alerts IS 'System alerts and notifications';

-- ================================================
-- Insert default data
-- ================================================

-- Default tariff
INSERT INTO tariffs (id, name, description, price_per_kwh, price_per_minute, start_fee, is_default)
VALUES (
    uuid_generate_v4(),
    'Tarifa Padrão',
    'Tarifa padrão para carregamento',
    0.75, -- R$ 0.75/kWh
    0.05, -- R$ 0.05/min idle fee
    2.00, -- R$ 2.00 start fee
    TRUE
) ON CONFLICT DO NOTHING;

-- Admin user (password: admin123 - bcrypt hash)
INSERT INTO users (id, name, email, password, role, status, email_verified)
VALUES (
    uuid_generate_v4(),
    'Administrador',
    'admin@sigec-ve.com',
    '$2a$10$N9qo8uLOickgx2ZMRZoMy.MqrqMXlDqK5EgZR1lSPgPNf7UQ.m1.6', -- admin123
    'admin',
    'active',
    TRUE
) ON CONFLICT (email) DO NOTHING;
