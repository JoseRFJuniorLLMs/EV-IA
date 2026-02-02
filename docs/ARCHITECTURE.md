# SIGEC-VE Enterprise - Arquitetura do Sistema

## Visão Geral

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                              CLIENTES EXTERNOS                                   │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                  │
│   ┌──────────────┐    ┌──────────────┐    ┌──────────────┐    ┌──────────────┐ │
│   │  Mobile App  │    │   Web App    │    │  Admin Panel │    │  Charge Point│ │
│   │   (Flutter)  │    │   (React)    │    │   (React)    │    │  (OCPP 2.0.1)│ │
│   └──────┬───────┘    └──────┬───────┘    └──────┬───────┘    └──────┬───────┘ │
│          │                   │                   │                   │          │
└──────────┼───────────────────┼───────────────────┼───────────────────┼──────────┘
           │                   │                   │                   │
           ▼                   ▼                   ▼                   ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│                              API GATEWAY / LOAD BALANCER                         │
│                                   (Nginx/Traefik)                                │
└─────────────────────────────────────────────────────────────────────────────────┘
           │                   │                   │                   │
           ▼                   ▼                   ▼                   ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│                           SIGEC-VE ENTERPRISE SERVER                             │
│                                     (Go 1.22)                                    │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                  │
│  ┌─────────────────────────────────────────────────────────────────────────────┐│
│  │                         PRESENTATION LAYER                                   ││
│  ├─────────────────────────────────────────────────────────────────────────────┤│
│  │                                                                              ││
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐ ││
│  │  │  REST API   │  │   GraphQL   │  │    gRPC     │  │   OCPP WebSocket    │ ││
│  │  │  (Fiber)    │  │   Server    │  │   Server    │  │   Server (v2.0.1)   │ ││
│  │  │  :8080      │  │   :8081     │  │   :50051    │  │      :9000          │ ││
│  │  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘  └──────────┬──────────┘ ││
│  │         │                │                │                    │            ││
│  │  ┌──────┴────────────────┴────────────────┴────────────────────┴──────────┐ ││
│  │  │                           HANDLERS                                      │ ││
│  │  │  • AuthHandler      • DeviceHandler      • TransactionHandler           │ ││
│  │  │  • VoiceHandler     • DeviceCommandsHandler  • V2GHandler               │ ││
│  │  │  • AdminHandler     • ReservationHandler     • PaymentHandler           │ ││
│  │  └────────────────────────────────────────────────────────────────────────┘ ││
│  └─────────────────────────────────────────────────────────────────────────────┘│
│                                        │                                         │
│  ┌─────────────────────────────────────┴───────────────────────────────────────┐│
│  │                          SERVICE LAYER (Business Logic)                      ││
│  ├──────────────────────────────────────────────────────────────────────────────┤│
│  │                                                                              ││
│  │  ┌────────────────────────────────────────────────────────────────────────┐ ││
│  │  │                         CORE SERVICES                                   │ ││
│  │  │  • AuthService           • DeviceService         • TransactionService  │ ││
│  │  │  • BillingService        • SmartChargingService  • ReservationService  │ ││
│  │  │  • AdminService          • VoiceAssistant        • FirmwareService     │ ││
│  │  └────────────────────────────────────────────────────────────────────────┘ ││
│  │                                                                              ││
│  │  ┌────────────────────────────────────────────────────────────────────────┐ ││
│  │  │                         V2G SERVICES                                    │ ││
│  │  │  • V2GService            • GridPriceService      • CCEEClient          │ ││
│  │  │  • V2GPaymentService     • ISO15118Service                             │ ││
│  │  └────────────────────────────────────────────────────────────────────────┘ ││
│  │                                                                              ││
│  │  ┌────────────────────────────────────────────────────────────────────────┐ ││
│  │  │                       PAYMENT SERVICES                                  │ ││
│  │  │  • PaymentService        • WalletService         • CardService         │ ││
│  │  └────────────────────────────────────────────────────────────────────────┘ ││
│  └──────────────────────────────────────────────────────────────────────────────┘│
│                                        │                                         │
│  ┌─────────────────────────────────────┴───────────────────────────────────────┐│
│  │                            DOMAIN LAYER (Entities)                           ││
│  ├──────────────────────────────────────────────────────────────────────────────┤│
│  │  • User              • ChargePoint        • Transaction       • Connector   ││
│  │  • V2GSession        • V2GPreferences     • V2GCompensation   • V2GCapability│
│  │  • Payment           • Wallet             • Reservation       • Location    ││
│  │  • ISO15118Certificate • ChargingContract • GridPricePoint                  ││
│  └──────────────────────────────────────────────────────────────────────────────┘│
│                                        │                                         │
│  ┌─────────────────────────────────────┴───────────────────────────────────────┐│
│  │                              PORTS (Interfaces)                              ││
│  ├──────────────────────────────────────────────────────────────────────────────┤│
│  │  • UserRepository     • ChargePointRepository  • TransactionRepository      ││
│  │  • V2GRepository      • ISO15118Repository     • Cache                      ││
│  │  • MessageQueue       • OCPPCommandService     • GridPriceService           ││
│  └──────────────────────────────────────────────────────────────────────────────┘│
│                                        │                                         │
│  ┌─────────────────────────────────────┴───────────────────────────────────────┐│
│  │                         ADAPTER LAYER (Infrastructure)                       ││
│  ├──────────────────────────────────────────────────────────────────────────────┤│
│  │                                                                              ││
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐ ││
│  │  │  PostgreSQL │  │    Redis    │  │    NATS     │  │   External APIs     │ ││
│  │  │   (GORM)    │  │   (Cache)   │  │  (Events)   │  │  Gemini/Stripe/CCEE │ ││
│  │  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────────────┘ ││
│  └──────────────────────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────────────────────┘
```

---

## Fluxo de Comunicação OCPP 2.0.1

```
┌─────────────────┐                                    ┌─────────────────────────┐
│   Charge Point  │                                    │     SIGEC-VE CSMS       │
│   (Carregador)  │                                    │  (Central System)       │
└────────┬────────┘                                    └───────────┬─────────────┘
         │                                                         │
         │  ══════════════ WebSocket Connection ══════════════════ │
         │                    ws://server:9000/ocpp/:cpID          │
         │                                                         │
         │  ─────────────── CP → CSMS (Requisições) ───────────────│
         │                                                         │
         │  [2, msgId, "BootNotification", {...}]  ─────────────►  │
         │  ◄─────────────  [3, msgId, {...}]                      │
         │                                                         │
         │  [2, msgId, "Heartbeat", {}]  ───────────────────────►  │
         │  ◄─────────────  [3, msgId, {currentTime}]              │
         │                                                         │
         │  [2, msgId, "StatusNotification", {...}]  ───────────►  │
         │  ◄─────────────  [3, msgId, {}]                         │
         │                                                         │
         │  [2, msgId, "TransactionEvent", {...}]  ─────────────►  │
         │  ◄─────────────  [3, msgId, {totalCost, ...}]           │
         │                                                         │
         │  [2, msgId, "MeterValues", {...}]  ──────────────────►  │
         │  ◄─────────────  [3, msgId, {}]                         │
         │                                                         │
         │  [2, msgId, "NotifyEVChargingNeeds", {...}]  ────────►  │  (V2G)
         │  ◄─────────────  [3, msgId, {status}]                   │
         │                                                         │
         │  ─────────────── CSMS → CP (Comandos) ──────────────────│
         │                                                         │
         │  ◄─────────────  [2, msgId, "RequestStartTransaction"]  │
         │  [3, msgId, {status, txId}]  ────────────────────────►  │
         │                                                         │
         │  ◄─────────────  [2, msgId, "RequestStopTransaction"]   │
         │  [3, msgId, {status}]  ──────────────────────────────►  │
         │                                                         │
         │  ◄─────────────  [2, msgId, "SetChargingProfile"]       │
         │  [3, msgId, {status}]  ──────────────────────────────►  │
         │                                                         │
         │  ◄─────────────  [2, msgId, "Reset"]                    │
         │  [3, msgId, {status}]  ──────────────────────────────►  │
         │                                                         │
         │  ◄─────────────  [2, msgId, "UpdateFirmware"]           │
         │  [3, msgId, {status}]  ──────────────────────────────►  │
         │                                                         │
```

---

## Arquitetura V2G (Vehicle-to-Grid)

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                            SISTEMA V2G COMPLETO                                  │
└─────────────────────────────────────────────────────────────────────────────────┘

     ┌──────────────┐         ┌──────────────┐         ┌──────────────┐
     │  Veículo EV  │◄───────►│ Charge Point │◄───────►│  SIGEC-VE    │
     │  (Bateria)   │  ISO    │ Bidirecional │  OCPP   │    CSMS      │
     │              │  15118  │              │  2.0.1  │              │
     └──────────────┘         └──────────────┘         └──────┬───────┘
                                                              │
                    ┌─────────────────────────────────────────┼─────────────────┐
                    │                                         │                 │
                    ▼                                         ▼                 ▼
          ┌─────────────────┐                    ┌─────────────────┐   ┌──────────────┐
          │   V2GService    │                    │ GridPriceService│   │ISO15118Service│
          │                 │                    │                 │   │              │
          │ • StartDischarge│                    │ • GetCurrentPrice   │ • AuthVehicle│
          │ • StopDischarge │                    │ • GetForecast   │   │ • ValidateCert│
          │ • OptimizeV2G   │                    │ • IsPeakHour    │   │ • GetContract│
          └────────┬────────┘                    └────────┬────────┘   └──────────────┘
                   │                                      │
                   │         ┌────────────────────────────┘
                   │         │
                   ▼         ▼
          ┌─────────────────────────┐         ┌─────────────────────────┐
          │    V2GPaymentService    │         │       CCEEClient        │
          │                         │         │                         │
          │ • CalculateCompensation │         │ • GetCurrentPLD         │
          │ • ProcessPayout         │◄───────►│ • GetPrices (região)    │
          │ • BatchProcess          │         │ • ConvertPLDToRetail    │
          └───────────┬─────────────┘         └───────────┬─────────────┘
                      │                                   │
                      ▼                                   ▼
          ┌─────────────────────────┐         ┌─────────────────────────┐
          │     WalletService       │         │     API CCEE Brasil     │
          │   (Carteira Usuário)    │         │  (Preços Energia Real)  │
          └─────────────────────────┘         └─────────────────────────┘
```

---

## Fluxo V2G - Descarga para a Rede

```
┌─────────┐     ┌──────────┐     ┌──────────┐     ┌───────────┐     ┌─────────┐
│ Usuário │     │  Mobile  │     │ SIGEC-VE │     │   OCPP    │     │  Charge │
│         │     │   App    │     │   API    │     │  Server   │     │  Point  │
└────┬────┘     └────┬─────┘     └────┬─────┘     └─────┬─────┘     └────┬────┘
     │               │                │                 │                │
     │  Abrir App    │                │                 │                │
     ├──────────────►│                │                 │                │
     │               │                │                 │                │
     │               │ GET /v2g/capability/:cpId        │                │
     │               ├───────────────►│                 │                │
     │               │                │  GetV2GCapability               │
     │               │                ├────────────────►│                │
     │               │                │                 │◄───────────────┤
     │               │◄───────────────┤                 │  V2G Supported │
     │               │  {supported: true, maxPower: 50} │                │
     │               │                │                 │                │
     │  Aceitar V2G  │                │                 │                │
     ├──────────────►│                │                 │                │
     │               │                │                 │                │
     │               │ POST /v2g/discharge/start        │                │
     │               │ {chargePointId, minSOC: 20%}     │                │
     │               ├───────────────►│                 │                │
     │               │                │                 │                │
     │               │                │  SetChargingProfile (negativo)  │
     │               │                ├────────────────►│                │
     │               │                │                 ├───────────────►│
     │               │                │                 │                │
     │               │                │                 │◄───────────────┤
     │               │                │                 │   Accepted     │
     │               │                │◄────────────────┤                │
     │               │◄───────────────┤                 │                │
     │               │  Session Started                 │                │
     │               │                │                 │                │
     │               │                │                 │  TransactionEvent
     │               │                │                 │◄───────────────┤
     │               │                │                 │  (Discharging) │
     │               │                │◄────────────────┤                │
     │               │◄───────────────┤  Update Session │                │
     │  Progresso    │  {power: -30kW, energy: -5kWh}   │                │
     │◄──────────────┤                │                 │                │
     │               │                │                 │                │
     │  SOC = 20%    │                │                 │                │
     ├──────────────►│                │                 │                │
     │  (ou manual)  │ POST /v2g/discharge/stop         │                │
     │               ├───────────────►│                 │                │
     │               │                │  ClearChargingProfile           │
     │               │                ├────────────────►│                │
     │               │                │                 ├───────────────►│
     │               │                │                 │◄───────────────┤
     │               │                │◄────────────────┤                │
     │               │                │                 │                │
     │               │                │  CalculateCompensation          │
     │               │                │  ProcessPayout                  │
     │               │                │                 │                │
     │               │◄───────────────┤                 │                │
     │  Compensação  │  R$ 25,50 creditado              │                │
     │◄──────────────┤                │                 │                │
     │               │                │                 │                │
```

---

## Modelo de Dados PostgreSQL

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                              SCHEMA DO BANCO DE DADOS                            │
└─────────────────────────────────────────────────────────────────────────────────┘

┌──────────────────┐      ┌──────────────────┐      ┌──────────────────┐
│      users       │      │   charge_points  │      │   transactions   │
├──────────────────┤      ├──────────────────┤      ├──────────────────┤
│ id (UUID) PK     │      │ id (VARCHAR) PK  │      │ id (UUID) PK     │
│ name             │      │ vendor           │      │ charge_point_id  │──┐
│ email (UNIQUE)   │      │ model            │      │ connector_id     │  │
│ password         │──┐   │ serial_number    │      │ user_id          │──┼─┐
│ role             │  │   │ firmware_version │      │ id_tag           │  │ │
│ status           │  │   │ status           │      │ start_time       │  │ │
│ created_at       │  │   │ location_id      │      │ end_time         │  │ │
│ updated_at       │  │   │ last_seen        │      │ meter_start      │  │ │
└──────────────────┘  │   │ created_at       │      │ meter_stop       │  │ │
                      │   └──────────────────┘      │ total_energy     │  │ │
                      │            │                │ status           │  │ │
                      │            │                │ cost             │  │ │
                      │            ▼                │ created_at       │  │ │
                      │   ┌──────────────────┐      └──────────────────┘  │ │
                      │   │    connectors    │               │            │ │
                      │   ├──────────────────┤               │            │ │
                      │   │ id (UUID) PK     │               │            │ │
                      │   │ charge_point_id  │◄──────────────┘            │ │
                      │   │ connector_id     │                            │ │
                      │   │ type             │                            │ │
                      │   │ status           │                            │ │
                      │   │ max_power_kw     │                            │ │
                      │   └──────────────────┘                            │ │
                      │                                                   │ │
                      │   ┌──────────────────────────────────────────────┼─┘
                      │   │                                              │
                      ▼   ▼                                              │
┌──────────────────────────────────────────────────────────────────────────────────┐
│                              TABELAS V2G                                         │
└──────────────────────────────────────────────────────────────────────────────────┘

┌──────────────────┐      ┌──────────────────┐      ┌──────────────────┐
│   v2g_sessions   │      │  v2g_preferences │      │   v2g_events     │
├──────────────────┤      ├──────────────────┤      ├──────────────────┤
│ id (UUID) PK     │      │ id (UUID) PK     │      │ id (UUID) PK     │
│ transaction_id   │      │ user_id (UNIQUE) │◄─────│ session_id       │
│ charge_point_id  │      │ auto_discharge   │      │ charge_point_id  │
│ connector_id     │      │ min_grid_price   │      │ event_type       │
│ user_id          │──────│ max_discharge_kwh│      │ direction        │
│ vehicle_id       │      │ preserve_soc     │      │ power_kw         │
│ direction        │      │ notify_on_start  │      │ energy_kwh       │
│ requested_power  │      │ notify_on_end    │      │ grid_price       │
│ actual_power     │      │ created_at       │      │ details (JSONB)  │
│ energy_transfer  │      │ updated_at       │      │ timestamp        │
│ grid_price_start │      └──────────────────┘      └──────────────────┘
│ current_grid_pric│
│ user_compensation│      ┌──────────────────┐      ┌──────────────────┐
│ min_battery_soc  │      │v2g_compensations │      │  v2g_grid_prices │
│ current_soc      │      ├──────────────────┤      ├──────────────────┤
│ start_time       │      │ id (UUID) PK     │      │ id (UUID) PK     │
│ end_time         │      │ session_id       │      │ timestamp        │
│ status           │──────│ user_id          │      │ price            │
│ created_at       │      │ energy_discharged│      │ is_peak          │
│ updated_at       │      │ avg_grid_price   │      │ source           │
└──────────────────┘      │ operator_margin  │      │ region           │
                          │ gross_amount     │      │ created_at       │
                          │ net_amount       │      └──────────────────┘
                          │ currency         │
                          │ status           │      ┌──────────────────┐
                          │ payment_id       │      │ v2g_capabilities │
                          │ paid_at          │      ├──────────────────┤
                          │ created_at       │      │ id (UUID) PK     │
                          │ updated_at       │      │ charge_point_id  │
                          └──────────────────┘      │ connector_id     │
                                                    │ supported        │
┌──────────────────────────────────────────────────────────────────────────────────┐
│                           TABELAS ISO 15118                                      │
└──────────────────────────────────────────────────────────────────────────────────┘
                                                    │ max_discharge_kw │
┌──────────────────┐      ┌──────────────────┐      │ bidirectional    │
│iso15118_certifica│      │ firmware_updates │      │ iso15118_support │
├──────────────────┤      ├──────────────────┤      │ current_soc      │
│ id (UUID) PK     │      │ id (UUID) PK     │      │ battery_capacity │
│ emaid (UNIQUE)   │      │ charge_point_id  │      │ last_updated     │
│ contract_id      │      │ request_id       │      │ created_at       │
│ vehicle_vin      │      │ firmware_url     │      └──────────────────┘
│ certificate_pem  │      │ version          │
│ certificate_chain│      │ retrieve_datetime│
│ private_key_enc  │      │ install_datetime │
│ v2g_capable      │      │ status           │
│ valid_from       │      │ progress         │
│ valid_to         │      │ error_message    │
│ revoked          │      │ retries          │
│ revoked_at       │      │ max_retries      │
│ revocation_reason│      │ retry_interval   │
│ provider_id      │      │ created_at       │
│ max_charge_kw    │      │ updated_at       │
│ max_discharge_kw │      │ completed_at     │
│ created_at       │      └──────────────────┘
│ updated_at       │
└──────────────────┘
```

---

## Infraestrutura de Produção

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                         SERVIDOR eva-ia.org (104.248.219.200)                    │
└─────────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────────┐
│                                  APLICAÇÃO                                       │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                  │
│   ┌─────────────────┐   ┌─────────────────┐   ┌─────────────────┐              │
│   │   sigec-ve      │   │  OCPP Server    │   │   gRPC Server   │              │
│   │   REST API      │   │   WebSocket     │   │                 │              │
│   │   :8080         │   │   :9000         │   │   :50051        │              │
│   └────────┬────────┘   └────────┬────────┘   └────────┬────────┘              │
│            │                     │                     │                        │
│            └─────────────────────┴─────────────────────┘                        │
│                                  │                                              │
└──────────────────────────────────┼──────────────────────────────────────────────┘
                                   │
┌──────────────────────────────────┼──────────────────────────────────────────────┐
│                            BANCO DE DADOS                                        │
├──────────────────────────────────┼──────────────────────────────────────────────┤
│                                  │                                              │
│   ┌─────────────────┐   ┌────────┴────────┐   ┌─────────────────┐              │
│   │   PostgreSQL    │   │     Redis       │   │      NATS       │              │
│   │   :5432         │   │     :6379       │   │     :4222       │              │
│   │                 │   │                 │   │                 │              │
│   │  • users        │   │  • cache        │   │  • eventos      │              │
│   │  • charge_points│   │  • sessions     │   │  • ocpp.*       │              │
│   │  • transactions │   │  • rate_limit   │   │  • v2g.*        │              │
│   │  • v2g_*        │   │                 │   │  • firmware.*   │              │
│   │  • iso15118_*   │   │                 │   │                 │              │
│   └─────────────────┘   └─────────────────┘   └─────────────────┘              │
│                                                                                  │
└──────────────────────────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────────────────────────┐
│                            INTEGRAÇÕES EXTERNAS                                   │
├──────────────────────────────────────────────────────────────────────────────────┤
│                                                                                  │
│   ┌─────────────────┐   ┌─────────────────┐   ┌─────────────────┐              │
│   │   Gemini API    │   │    CCEE API     │   │   Stripe API    │              │
│   │   (Voz)         │   │ (Preços Energia)│   │  (Pagamentos)   │              │
│   └─────────────────┘   └─────────────────┘   └─────────────────┘              │
│                                                                                  │
└──────────────────────────────────────────────────────────────────────────────────┘
```

---

## Tópicos de Eventos NATS

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                              TÓPICOS NATS JETSTREAM                              │
└─────────────────────────────────────────────────────────────────────────────────┘

┌────────────────────────────────┬────────────────────────────────────────────────┐
│           TÓPICO               │                  DESCRIÇÃO                      │
├────────────────────────────────┼────────────────────────────────────────────────┤
│ ocpp.device.connected          │ Charge point conectou ao servidor              │
│ ocpp.device.disconnected       │ Charge point desconectou                       │
│ ocpp.device.boot               │ BootNotification recebido                      │
│ ocpp.device.status             │ StatusNotification (mudança de estado)         │
│ ocpp.command.sent              │ Comando enviado para charge point              │
│ ocpp.command.response          │ Resposta recebida do charge point              │
│ ocpp.command.timeout           │ Timeout de comando (sem resposta)              │
├────────────────────────────────┼────────────────────────────────────────────────┤
│ transaction.started            │ Transação de carregamento iniciada             │
│ transaction.updated            │ Atualização de meter values                    │
│ transaction.completed          │ Transação finalizada                           │
├────────────────────────────────┼────────────────────────────────────────────────┤
│ v2g.session.started            │ Sessão V2G (descarga) iniciada                 │
│ v2g.session.updated            │ Atualização de energia/potência V2G            │
│ v2g.session.completed          │ Sessão V2G finalizada                          │
│ v2g.compensation.calculated    │ Compensação calculada para usuário             │
│ v2g.compensation.paid          │ Compensação paga (creditada na carteira)       │
│ v2g.compensation.failed        │ Falha no pagamento da compensação              │
├────────────────────────────────┼────────────────────────────────────────────────┤
│ firmware.update.started        │ Atualização de firmware iniciada               │
│ firmware.update.progress       │ Progresso da atualização (%)                   │
│ firmware.update.completed      │ Atualização concluída com sucesso              │
│ firmware.update.failed         │ Falha na atualização                           │
├────────────────────────────────┼────────────────────────────────────────────────┤
│ payment.completed              │ Pagamento processado                           │
│ payment.failed                 │ Falha no pagamento                             │
│ wallet.credited                │ Crédito adicionado à carteira                  │
│ wallet.debited                 │ Débito na carteira                             │
└────────────────────────────────┴────────────────────────────────────────────────┘
```

---

## Endpoints REST API

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                              API REST - ENDPOINTS                                │
└─────────────────────────────────────────────────────────────────────────────────┘

┌────────────────────────────────────────────────────────────────────────────────┐
│ AUTENTICAÇÃO                                                                    │
├────────────────────────────────────────────────────────────────────────────────┤
│ POST   /api/v1/auth/login              Login                                   │
│ POST   /api/v1/auth/register           Registrar usuário                       │
│ POST   /api/v1/auth/refresh            Renovar token                           │
│ POST   /api/v1/auth/logout             Logout                                  │
└────────────────────────────────────────────────────────────────────────────────┘

┌────────────────────────────────────────────────────────────────────────────────┐
│ DISPOSITIVOS (Charge Points)                                                    │
├────────────────────────────────────────────────────────────────────────────────┤
│ GET    /api/v1/devices                 Listar dispositivos                     │
│ GET    /api/v1/devices/:id             Obter dispositivo                       │
│ GET    /api/v1/devices/nearby          Dispositivos próximos                   │
│ PATCH  /api/v1/devices/:id/status      Atualizar status                        │
└────────────────────────────────────────────────────────────────────────────────┘

┌────────────────────────────────────────────────────────────────────────────────┐
│ COMANDOS OCPP (CSMS → Charge Point)                                            │
├────────────────────────────────────────────────────────────────────────────────┤
│ POST   /api/v1/devices/:id/remote-start         Iniciar carregamento remoto   │
│ POST   /api/v1/devices/:id/remote-stop          Parar carregamento remoto     │
│ POST   /api/v1/devices/:id/reset                Reiniciar dispositivo         │
│ POST   /api/v1/devices/:id/trigger/:message     Solicitar mensagem            │
│ POST   /api/v1/devices/:id/unlock/:connector    Desbloquear conector          │
│ POST   /api/v1/devices/:id/availability         Mudar disponibilidade         │
│ POST   /api/v1/devices/:id/charging-profile     Definir perfil de carga       │
│ DELETE /api/v1/devices/:id/charging-profile     Limpar perfil de carga        │
│ POST   /api/v1/devices/:id/firmware/update      Atualizar firmware            │
│ GET    /api/v1/devices/:id/firmware/status      Status do firmware            │
└────────────────────────────────────────────────────────────────────────────────┘

┌────────────────────────────────────────────────────────────────────────────────┐
│ TRANSAÇÕES                                                                      │
├────────────────────────────────────────────────────────────────────────────────┤
│ POST   /api/v1/transactions/start      Iniciar transação                       │
│ POST   /api/v1/transactions/:id/stop   Parar transação                         │
│ GET    /api/v1/transactions/:id        Obter transação                         │
│ GET    /api/v1/transactions/active     Transação ativa do usuário              │
│ GET    /api/v1/transactions/history    Histórico de transações                 │
└────────────────────────────────────────────────────────────────────────────────┘

┌────────────────────────────────────────────────────────────────────────────────┐
│ V2G (Vehicle-to-Grid)                                                          │
├────────────────────────────────────────────────────────────────────────────────┤
│ POST   /api/v1/v2g/discharge/start     Iniciar descarga V2G                    │
│ POST   /api/v1/v2g/discharge/stop      Parar descarga V2G                      │
│ GET    /api/v1/v2g/session/:id         Obter sessão V2G                        │
│ GET    /api/v1/v2g/session/active/:id  Sessão V2G ativa                        │
│ GET    /api/v1/v2g/capability/:id      Verificar capacidade V2G                │
│ GET    /api/v1/v2g/grid-price          Preço atual da energia                  │
│ GET    /api/v1/v2g/grid-price/forecast Previsão de preços                      │
│ GET    /api/v1/v2g/preferences         Obter preferências V2G                  │
│ POST   /api/v1/v2g/preferences         Definir preferências V2G                │
│ GET    /api/v1/v2g/stats               Estatísticas V2G do usuário             │
│ POST   /api/v1/v2g/compensation/calc   Calcular compensação                    │
│ POST   /api/v1/v2g/optimize            Otimizar V2G automaticamente            │
└────────────────────────────────────────────────────────────────────────────────┘

┌────────────────────────────────────────────────────────────────────────────────┐
│ VOZ (Gemini Live API)                                                          │
├────────────────────────────────────────────────────────────────────────────────┤
│ POST   /api/v1/voice/command           Processar comando de voz                │
│ GET    /api/v1/voice/history           Histórico de comandos                   │
│ WS     /ws/voice                       Stream bidirecional de voz              │
└────────────────────────────────────────────────────────────────────────────────┘

┌────────────────────────────────────────────────────────────────────────────────┐
│ PAGAMENTOS                                                                      │
├────────────────────────────────────────────────────────────────────────────────┤
│ POST   /api/v1/payments/intent         Criar intent de pagamento               │
│ GET    /api/v1/payments/:id            Obter pagamento                         │
│ GET    /api/v1/payments/history        Histórico de pagamentos                 │
│ POST   /api/v1/payments/pix            Criar pagamento PIX                     │
│ POST   /api/v1/payments/boleto         Criar boleto                            │
│ GET    /api/v1/wallet                  Obter carteira                          │
│ GET    /api/v1/wallet/transactions     Histórico da carteira                   │
└────────────────────────────────────────────────────────────────────────────────┘

┌────────────────────────────────────────────────────────────────────────────────┐
│ ADMIN                                                                          │
├────────────────────────────────────────────────────────────────────────────────┤
│ GET    /api/v1/admin/dashboard         Estatísticas do dashboard               │
│ GET    /api/v1/admin/users             Listar usuários                         │
│ GET    /api/v1/admin/stations          Listar estações                         │
│ GET    /api/v1/admin/transactions      Listar transações                       │
│ GET    /api/v1/admin/alerts            Listar alertas                          │
│ POST   /api/v1/admin/reports           Gerar relatórios                        │
└────────────────────────────────────────────────────────────────────────────────┘

┌────────────────────────────────────────────────────────────────────────────────┐
│ HEALTH & METRICS                                                               │
├────────────────────────────────────────────────────────────────────────────────┤
│ GET    /health/live                    Liveness probe                          │
│ GET    /health/ready                   Readiness probe                         │
│ GET    /metrics                        Métricas Prometheus                     │
└────────────────────────────────────────────────────────────────────────────────┘
```

---

*Documentação gerada em: Fevereiro 2026*
*Versão do Sistema: v2.0.0*
