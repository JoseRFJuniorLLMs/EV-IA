-- =============================================
-- SEED DATA: EV-Web / EV-IA
-- =============================================

-- Clean existing data (keep admin)
DELETE FROM meter_values;
DELETE FROM transactions;
DELETE FROM wallet_transactions;
DELETE FROM wallets;
DELETE FROM payments;
DELETE FROM reservations;
DELETE FROM connectors;
DELETE FROM charge_points;
DELETE FROM locations;
DELETE FROM authorization_tags;
DELETE FROM alerts;
DELETE FROM users WHERE email != 'admin@sigec-ve.com';

-- =============================================
-- USERS (Clients) - password: 123456 for all
-- bcrypt hash of '123456' (generated with bcrypt.DefaultCost=10)
-- =============================================
INSERT INTO users (id, name, email, password, phone, document, role, status, email_verified) VALUES
('a0000001-0000-0000-0000-000000000001', 'Carlos Silva', 'carlos@ev.com', '$2a$10$t5Ixd/WKmT3/LC9Na6TXK.5m68G87GEfhNiBaAQQmzka3n/cudmba', '11999001001', '12345678901', 'user', 'active', true),
('a0000001-0000-0000-0000-000000000002', 'Ana Oliveira', 'ana@ev.com', '$2a$10$t5Ixd/WKmT3/LC9Na6TXK.5m68G87GEfhNiBaAQQmzka3n/cudmba', '11999002002', '23456789012', 'user', 'active', true),
('a0000001-0000-0000-0000-000000000003', 'Roberto Santos', 'roberto@ev.com', '$2a$10$t5Ixd/WKmT3/LC9Na6TXK.5m68G87GEfhNiBaAQQmzka3n/cudmba', '11999003003', '34567890123', 'user', 'active', true),
('a0000001-0000-0000-0000-000000000004', 'Maria Souza', 'maria@ev.com', '$2a$10$t5Ixd/WKmT3/LC9Na6TXK.5m68G87GEfhNiBaAQQmzka3n/cudmba', '11999004004', '45678901234', 'user', 'active', true),
('a0000001-0000-0000-0000-000000000005', 'Fernando Lima', 'fernando@ev.com', '$2a$10$t5Ixd/WKmT3/LC9Na6TXK.5m68G87GEfhNiBaAQQmzka3n/cudmba', '11999005005', '56789012345', 'user', 'active', true),
('a0000001-0000-0000-0000-000000000006', 'Juliana Costa', 'juliana@ev.com', '$2a$10$t5Ixd/WKmT3/LC9Na6TXK.5m68G87GEfhNiBaAQQmzka3n/cudmba', '11999006006', '67890123456', 'operator', 'active', true),
('a0000001-0000-0000-0000-000000000007', 'Pedro Mendes', 'pedro@ev.com', '$2a$10$t5Ixd/WKmT3/LC9Na6TXK.5m68G87GEfhNiBaAQQmzka3n/cudmba', '11999007007', '78901234567', 'user', 'active', true),
('a0000001-0000-0000-0000-000000000008', 'Lucia Ferreira', 'lucia@ev.com', '$2a$10$t5Ixd/WKmT3/LC9Na6TXK.5m68G87GEfhNiBaAQQmzka3n/cudmba', '11999008008', '89012345678', 'user', 'active', true);

-- =============================================
-- LOCATIONS (Sao Paulo region)
-- =============================================
INSERT INTO locations (id, name, address, city, state, country, postal_code, latitude, longitude, operator_id, status) VALUES
('b0000001-0000-0000-0000-000000000001', 'Shopping Ibirapuera', 'Av. Ibirapuera, 3103 - Moema', 'Sao Paulo', 'SP', 'BR', '04029-902', -23.6115, -46.6660, 'a0000001-0000-0000-0000-000000000006', 'active'),
('b0000001-0000-0000-0000-000000000002', 'Shopping Morumbi', 'Av. Roque Petroni Jr, 1089 - Morumbi', 'Sao Paulo', 'SP', 'BR', '04707-900', -23.6230, -46.6990, 'a0000001-0000-0000-0000-000000000006', 'active'),
('b0000001-0000-0000-0000-000000000003', 'Posto Shell Paulista', 'Av. Paulista, 1578 - Bela Vista', 'Sao Paulo', 'SP', 'BR', '01310-200', -23.5615, -46.6559, 'a0000001-0000-0000-0000-000000000006', 'active'),
('b0000001-0000-0000-0000-000000000004', 'Estacionamento Faria Lima', 'Av. Brig. Faria Lima, 2232 - Pinheiros', 'Sao Paulo', 'SP', 'BR', '01452-000', -23.5745, -46.6845, 'a0000001-0000-0000-0000-000000000006', 'active'),
('b0000001-0000-0000-0000-000000000005', 'Shopping Vila Olimpia', 'R. Olimpiadas, 360 - Vila Olimpia', 'Sao Paulo', 'SP', 'BR', '04551-000', -23.5960, -46.6870, 'a0000001-0000-0000-0000-000000000006', 'active'),
('b0000001-0000-0000-0000-000000000006', 'Aeroporto Congonhas', 'Av. Washington Luis, s/n - Campo Belo', 'Sao Paulo', 'SP', 'BR', '04626-911', -23.6266, -46.6563, 'a0000001-0000-0000-0000-000000000006', 'active'),
('b0000001-0000-0000-0000-000000000007', 'Shopping Eldorado', 'Av. Reboucas, 3970 - Pinheiros', 'Sao Paulo', 'SP', 'BR', '05402-600', -23.5729, -46.6965, 'a0000001-0000-0000-0000-000000000006', 'active'),
('b0000001-0000-0000-0000-000000000008', 'Parque Villa-Lobos', 'Av. Prof. Fonseca Rodrigues, 2001', 'Sao Paulo', 'SP', 'BR', '05461-010', -23.5468, -46.7240, 'a0000001-0000-0000-0000-000000000006', 'active');

-- =============================================
-- CHARGE POINTS (15 stations)
-- =============================================
INSERT INTO charge_points (id, vendor, model, serial_number, firmware_version, location_id, status, last_heartbeat, is_online, max_power_kw) VALUES
('CP-IBIRA-001', 'ABB', 'Terra 360', 'ABB-T360-001', '2.5.1', 'b0000001-0000-0000-0000-000000000001', 'Available', NOW() - interval '2 min', true, 360),
('CP-IBIRA-002', 'ABB', 'Terra 360', 'ABB-T360-002', '2.5.1', 'b0000001-0000-0000-0000-000000000001', 'Available', NOW() - interval '1 min', true, 360),
('CP-MORUM-001', 'Schneider', 'EVLink Pro AC', 'SCH-EVLP-001', '3.1.0', 'b0000001-0000-0000-0000-000000000002', 'Available', NOW() - interval '3 min', true, 22),
('CP-MORUM-002', 'Schneider', 'EVLink Pro AC', 'SCH-EVLP-002', '3.1.0', 'b0000001-0000-0000-0000-000000000002', 'Occupied', NOW() - interval '1 min', true, 22),
('CP-PAUL-001', 'Siemens', 'VersiCharge Ultra', 'SIE-VCU-001', '1.8.4', 'b0000001-0000-0000-0000-000000000003', 'Available', NOW() - interval '5 min', true, 150),
('CP-FARIA-001', 'WEG', 'WEMOB AC22', 'WEG-WAC-001', '1.2.0', 'b0000001-0000-0000-0000-000000000004', 'Available', NOW() - interval '2 min', true, 22),
('CP-FARIA-002', 'WEG', 'WEMOB DC60', 'WEG-WDC-001', '1.3.1', 'b0000001-0000-0000-0000-000000000004', 'Occupied', NOW() - interval '30 sec', true, 60),
('CP-VILAO-001', 'ABB', 'Terra AC W22', 'ABB-TAC-001', '2.2.0', 'b0000001-0000-0000-0000-000000000005', 'Available', NOW() - interval '4 min', true, 22),
('CP-VILAO-002', 'ABB', 'Terra 184', 'ABB-T184-001', '2.4.0', 'b0000001-0000-0000-0000-000000000005', 'Faulted', NOW() - interval '2 hours', false, 180),
('CP-CONGH-001', 'Tritium', 'RTM 75', 'TRI-RTM-001', '4.0.2', 'b0000001-0000-0000-0000-000000000006', 'Available', NOW() - interval '1 min', true, 75),
('CP-CONGH-002', 'Tritium', 'PKM 150', 'TRI-PKM-001', '4.1.0', 'b0000001-0000-0000-0000-000000000006', 'Available', NOW() - interval '2 min', true, 150),
('CP-ELDO-001', 'BYD', 'EV Charger 60kW', 'BYD-EVC-001', '2.0.1', 'b0000001-0000-0000-0000-000000000007', 'Available', NOW() - interval '3 min', true, 60),
('CP-ELDO-002', 'BYD', 'EV Charger 7kW', 'BYD-EVC-002', '2.0.1', 'b0000001-0000-0000-0000-000000000007', 'Unavailable', NOW() - interval '1 day', false, 7),
('CP-VILLA-001', 'Kempower', 'S-Series 240', 'KEM-SS-001', '5.0.0', 'b0000001-0000-0000-0000-000000000008', 'Available', NOW() - interval '1 min', true, 240),
('CP-VILLA-002', 'Kempower', 'S-Series 240', 'KEM-SS-002', '5.0.0', 'b0000001-0000-0000-0000-000000000008', 'Available', NOW() - interval '2 min', true, 240);

-- =============================================
-- CONNECTORS (25 connectors across all stations)
-- =============================================
INSERT INTO connectors (charge_point_id, connector_id, type, status, max_power_kw, max_current_a, max_voltage_v) VALUES
-- Ibirapuera ABB Terra 360
('CP-IBIRA-001', 1, 'CCS2', 'Available', 360, 500, 920),
('CP-IBIRA-001', 2, 'CHAdeMO', 'Available', 200, 350, 600),
('CP-IBIRA-002', 1, 'CCS2', 'Available', 360, 500, 920),
('CP-IBIRA-002', 2, 'CCS2', 'Available', 360, 500, 920),
-- Morumbi Schneider AC
('CP-MORUM-001', 1, 'Type2', 'Available', 22, 32, 400),
('CP-MORUM-001', 2, 'Type2', 'Available', 22, 32, 400),
('CP-MORUM-002', 1, 'Type2', 'Occupied', 22, 32, 400),
('CP-MORUM-002', 2, 'Type2', 'Available', 22, 32, 400),
-- Paulista Siemens DC
('CP-PAUL-001', 1, 'CCS2', 'Available', 150, 250, 920),
('CP-PAUL-001', 2, 'CHAdeMO', 'Available', 100, 200, 600),
-- Faria Lima WEG
('CP-FARIA-001', 1, 'Type2', 'Available', 22, 32, 400),
('CP-FARIA-002', 1, 'CCS2', 'Occupied', 60, 150, 500),
-- Vila Olimpia ABB
('CP-VILAO-001', 1, 'Type2', 'Available', 22, 32, 400),
('CP-VILAO-001', 2, 'Type2', 'Available', 22, 32, 400),
('CP-VILAO-002', 1, 'CCS2', 'Faulted', 180, 300, 920),
-- Congonhas Tritium
('CP-CONGH-001', 1, 'CCS2', 'Available', 75, 150, 500),
('CP-CONGH-002', 1, 'CCS2', 'Available', 150, 250, 920),
-- Eldorado BYD
('CP-ELDO-001', 1, 'CCS2', 'Available', 60, 150, 500),
('CP-ELDO-001', 2, 'Type2', 'Available', 22, 32, 400),
('CP-ELDO-002', 1, 'Type2', 'Unavailable', 7, 32, 230),
-- Villa-Lobos Kempower
('CP-VILLA-001', 1, 'CCS2', 'Available', 240, 400, 920),
('CP-VILLA-001', 2, 'CCS2', 'Available', 240, 400, 920),
('CP-VILLA-002', 1, 'CCS2', 'Available', 240, 400, 920),
('CP-VILLA-002', 2, 'CCS2', 'Available', 240, 400, 920);

-- =============================================
-- WALLETS
-- =============================================
INSERT INTO wallets (user_id, balance, currency) VALUES
('a0000001-0000-0000-0000-000000000001', 250.00, 'BRL'),
('a0000001-0000-0000-0000-000000000002', 180.50, 'BRL'),
('a0000001-0000-0000-0000-000000000003', 75.00, 'BRL'),
('a0000001-0000-0000-0000-000000000004', 420.00, 'BRL'),
('a0000001-0000-0000-0000-000000000005', 50.25, 'BRL'),
('a0000001-0000-0000-0000-000000000006', 1500.00, 'BRL'),
('a0000001-0000-0000-0000-000000000007', 0.00, 'BRL'),
('a0000001-0000-0000-0000-000000000008', 310.75, 'BRL');

-- =============================================
-- AUTHORIZATION TAGS (RFID)
-- =============================================
INSERT INTO authorization_tags (id_tag, user_id, status) VALUES
('TAG-CARLOS-001', 'a0000001-0000-0000-0000-000000000001', 'Accepted'),
('TAG-ANA-001', 'a0000001-0000-0000-0000-000000000002', 'Accepted'),
('TAG-ROBERTO-001', 'a0000001-0000-0000-0000-000000000003', 'Accepted'),
('TAG-MARIA-001', 'a0000001-0000-0000-0000-000000000004', 'Accepted'),
('TAG-FERNANDO-001', 'a0000001-0000-0000-0000-000000000005', 'Accepted'),
('TAG-JULIANA-001', 'a0000001-0000-0000-0000-000000000006', 'Accepted'),
('TAG-PEDRO-001', 'a0000001-0000-0000-0000-000000000007', 'Accepted'),
('TAG-LUCIA-001', 'a0000001-0000-0000-0000-000000000008', 'Accepted');

-- =============================================
-- TRANSACTIONS (2 active + 7 completed)
-- =============================================
INSERT INTO transactions (id, charge_point_id, connector_id, user_id, id_tag, start_time, meter_start, status, cost, tariff_id) VALUES
('c0000001-0000-0000-0000-000000000001', 'CP-MORUM-002', 1, 'a0000001-0000-0000-0000-000000000002', 'TAG-ANA-001', NOW() - interval '45 min', 10000, 'Started', NULL, (SELECT id FROM tariffs WHERE is_default=true LIMIT 1)),
('c0000001-0000-0000-0000-000000000002', 'CP-FARIA-002', 1, 'a0000001-0000-0000-0000-000000000005', 'TAG-FERNANDO-001', NOW() - interval '20 min', 5000, 'Started', NULL, (SELECT id FROM tariffs WHERE is_default=true LIMIT 1));

INSERT INTO transactions (id, charge_point_id, connector_id, user_id, id_tag, start_time, end_time, meter_start, meter_stop, total_energy_wh, status, stop_reason, cost, tariff_id) VALUES
('c0000001-0000-0000-0000-000000000010', 'CP-IBIRA-001', 1, 'a0000001-0000-0000-0000-000000000001', 'TAG-CARLOS-001', NOW() - interval '1 day 3 hours', NOW() - interval '1 day 2 hours', 0, 45000, 45000, 'Completed', 'EVDisconnected', 35.75, (SELECT id FROM tariffs WHERE is_default=true LIMIT 1)),
('c0000001-0000-0000-0000-000000000011', 'CP-PAUL-001', 1, 'a0000001-0000-0000-0000-000000000001', 'TAG-CARLOS-001', NOW() - interval '3 days 5 hours', NOW() - interval '3 days 4 hours', 0, 62000, 62000, 'Completed', 'EVDisconnected', 48.50, (SELECT id FROM tariffs WHERE is_default=true LIMIT 1)),
('c0000001-0000-0000-0000-000000000012', 'CP-CONGH-001', 1, 'a0000001-0000-0000-0000-000000000003', 'TAG-ROBERTO-001', NOW() - interval '2 days 1 hour', NOW() - interval '2 days', 0, 30000, 30000, 'Completed', 'Local', 24.50, (SELECT id FROM tariffs WHERE is_default=true LIMIT 1)),
('c0000001-0000-0000-0000-000000000013', 'CP-VILLA-001', 1, 'a0000001-0000-0000-0000-000000000004', 'TAG-MARIA-001', NOW() - interval '1 day 6 hours', NOW() - interval '1 day 5 hours', 0, 80000, 80000, 'Completed', 'EVDisconnected', 62.00, (SELECT id FROM tariffs WHERE is_default=true LIMIT 1)),
('c0000001-0000-0000-0000-000000000014', 'CP-ELDO-001', 1, 'a0000001-0000-0000-0000-000000000002', 'TAG-ANA-001', NOW() - interval '4 days 2 hours', NOW() - interval '4 days 1 hour', 0, 25000, 25000, 'Completed', 'Remote', 21.25, (SELECT id FROM tariffs WHERE is_default=true LIMIT 1)),
('c0000001-0000-0000-0000-000000000015', 'CP-FARIA-001', 1, 'a0000001-0000-0000-0000-000000000007', 'TAG-PEDRO-001', NOW() - interval '5 days 4 hours', NOW() - interval '5 days 1 hour', 0, 18000, 18000, 'Completed', 'EVDisconnected', 15.50, (SELECT id FROM tariffs WHERE is_default=true LIMIT 1)),
('c0000001-0000-0000-0000-000000000016', 'CP-VILAO-001', 1, 'a0000001-0000-0000-0000-000000000008', 'TAG-LUCIA-001', NOW() - interval '6 days 3 hours', NOW() - interval '6 days 2 hours', 0, 55000, 55000, 'Completed', 'EVDisconnected', 43.25, (SELECT id FROM tariffs WHERE is_default=true LIMIT 1));

-- =============================================
-- PAYMENTS
-- =============================================
INSERT INTO payments (user_id, amount, currency, method, status, transaction_id, description, paid_at) VALUES
('a0000001-0000-0000-0000-000000000001', 35.75, 'BRL', 'wallet', 'completed', 'c0000001-0000-0000-0000-000000000010', 'Carregamento CP-IBIRA-001', NOW() - interval '1 day 2 hours'),
('a0000001-0000-0000-0000-000000000001', 48.50, 'BRL', 'wallet', 'completed', 'c0000001-0000-0000-0000-000000000011', 'Carregamento CP-PAUL-001', NOW() - interval '3 days 4 hours'),
('a0000001-0000-0000-0000-000000000003', 24.50, 'BRL', 'pix', 'completed', 'c0000001-0000-0000-0000-000000000012', 'Carregamento CP-CONGH-001', NOW() - interval '2 days'),
('a0000001-0000-0000-0000-000000000004', 62.00, 'BRL', 'card', 'completed', 'c0000001-0000-0000-0000-000000000013', 'Carregamento CP-VILLA-001', NOW() - interval '1 day 5 hours'),
('a0000001-0000-0000-0000-000000000002', 21.25, 'BRL', 'wallet', 'completed', 'c0000001-0000-0000-0000-000000000014', 'Carregamento CP-ELDO-001', NOW() - interval '4 days 1 hour'),
('a0000001-0000-0000-0000-000000000007', 15.50, 'BRL', 'pix', 'completed', 'c0000001-0000-0000-0000-000000000015', 'Carregamento CP-FARIA-001', NOW() - interval '5 days 1 hour'),
('a0000001-0000-0000-0000-000000000008', 43.25, 'BRL', 'wallet', 'completed', 'c0000001-0000-0000-0000-000000000016', 'Carregamento CP-VILAO-001', NOW() - interval '6 days 2 hours');

-- =============================================
-- ALERTS
-- =============================================
INSERT INTO alerts (type, severity, title, message, source, source_id) VALUES
('device_offline', 'warning', 'Estacao Offline', 'CP-VILAO-002 (Vila Olimpia) esta offline ha 2 horas', 'charge_point', 'CP-VILAO-002'),
('device_offline', 'warning', 'Estacao Offline', 'CP-ELDO-002 (Eldorado) esta offline ha 1 dia', 'charge_point', 'CP-ELDO-002'),
('error', 'error', 'Falha no Conector', 'Conector CCS2 de CP-VILAO-002 reportou falha de comunicacao', 'charge_point', 'CP-VILAO-002'),
('maintenance', 'info', 'Manutencao Programada', 'Estacao CP-ELDO-002 em manutencao preventiva', 'charge_point', 'CP-ELDO-002');

-- =============================================
-- SUMMARY
-- =============================================
SELECT 'Users: ' || count(*) FROM users;
SELECT 'Locations: ' || count(*) FROM locations;
SELECT 'Charge Points: ' || count(*) FROM charge_points;
SELECT 'Connectors: ' || count(*) FROM connectors;
SELECT 'Wallets: ' || count(*) FROM wallets;
SELECT 'Transactions: ' || count(*) FROM transactions;
SELECT 'Payments: ' || count(*) FROM payments;
SELECT 'Alerts: ' || count(*) FROM alerts;
