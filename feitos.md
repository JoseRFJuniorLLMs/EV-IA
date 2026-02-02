# OCPP 2.0.1 + V2G Implementation Complete

## Resumo

Implementação completa do sistema OCPP 2.0.1 com suporte a V2G (Vehicle-to-Grid).

---

## Arquivos Modificados

### 1. `internal/adapter/ocpp/v201/types.go` (+400 linhas)
- Tipos de mensagens CSMS → CP (RequestStartTransaction, RequestStopTransaction, Reset, etc.)
- Tipos de Charging Profile (ChargingProfile, ChargingSchedule, ChargingSchedulePeriod)
- Tipos V2G (NotifyEVChargingNeeds, DCChargingParameters com suporte a descarga)
- Tipos ISO 15118 (Get15118EVCertificate, Authorize)
- Tipos de Firmware (UpdateFirmware, FirmwareStatusNotification)
- Gerenciamento de variáveis (GetVariables, SetVariables)

### 2. `internal/adapter/ocpp/v201/server.go` (+200 linhas)
- Sistema de tracking PendingRequest para comandos CSMS → CP
- Métodos `SendCommand()` e `SendCommandAsync()`
- `handleCallResult()` e `handleCallError()` para processar respostas
- Tratamento de timeout com limpeza automática

### 3. `internal/adapter/ocpp/v201/handlers.go` (+150 linhas)
- Handler MeterValues
- Handler FirmwareStatusNotification
- Handler NotifyEVChargingNeeds (V2G)
- Handler NotifyEVChargingSchedule
- Handler ReportChargingProfiles
- Handler Authorize

---

## Arquivos Criados

### 4. `internal/adapter/ocpp/v201/commands.go` (~350 linhas)
- RemoteStartTransaction, RemoteStopTransaction
- Reset, TriggerMessage
- SetChargingProfile, ClearChargingProfile, GetChargingProfiles
- UpdateFirmware, UpdateFirmwareSigned
- GetVariables, SetVariables
- UnlockConnector, ChangeAvailability
- V2G específico: SetV2GChargingProfile, CancelV2GDischarge

### 5. `internal/adapter/ocpp/v201/v2g.go` (~250 linhas)
- V2GManager para operações V2G
- Tracking de EVCapability
- ProcessChargingNeeds para detectar EVs com capacidade V2G
- StartV2GDischarge, StopV2GDischarge
- CalculateOptimalDischarge para otimização V2G

### 6. `internal/domain/v2g.go` (~150 linhas)
- V2GSession, V2GDirection, V2GStatus
- V2GCapability, V2GPreferences
- V2GEvent, V2GCompensation
- GridPricePoint, V2GStats
- ISO15118VehicleIdentity, ChargingContract

### 7. `internal/service/v2g/service.go` (~400 linhas)
- StartDischarge, StopDischarge
- CalculateCompensation
- CheckV2GCapability
- SetUserPreferences, GetUserPreferences
- OptimizeV2G (otimização automática baseada em preços da rede)
- UpdateSessionMetrics

### 8. `internal/service/v2g/grid_price.go` (~250 linhas)
- GetCurrentPrice (preço dinâmico)
- GetPriceForecast (previsão 24 horas)
- IsPeakHour, IsSuperPeakHour
- CalculateV2GCompensation
- GetOptimalDischargeHours
- Modelo de preços brasileiro (fora-ponta, ponta, super-ponta)

### 9. `internal/service/device/firmware_service.go` (~300 linhas)
- UpdateFirmware, GetFirmwareStatus
- CancelFirmwareUpdate
- HandleFirmwareStatusNotification
- Tracking de progresso, publicação de eventos

### 10. `internal/ports/services.go` (+200 linhas)
- Interface V2GService
- Interface GridPriceService
- Interface V2GRepository
- Interface OCPPCommandService
- Interface FirmwareService
- Interface MessageQueue

### 11. `cmd/simulator/main.go` + `simulator.go` (~500 linhas)
- Simulador completo de charge point OCPP 2.0.1
- Modos interativo e background
- Suporte V2G (flag --v2g)
- Responde a todos os comandos CSMS
- Simula transações, meter values, atualizações de firmware

---

## Como Usar

### Executar o Simulador

```bash
# Simulador básico
go run cmd/simulator/*.go --id=CP001 --server=ws://localhost:9000/ocpp

# Simulador com capacidade V2G
go run cmd/simulator/*.go --id=CP001 --v2g --soc=80 --battery=75 --discharge-power=50

# Modo interativo
go run cmd/simulator/*.go --id=CP001 --interactive
```

### Comandos do Modo Interativo

```
start <connector>       - Iniciar carregamento no conector
stop                    - Parar carregamento atual
status <connector>      - Definir status do conector (Available/Occupied/Faulted)
meter <value>           - Enviar valor do medidor (Wh)
heartbeat               - Enviar heartbeat
v2g start <power>       - Iniciar descarga V2G (kW)
v2g stop                - Parar descarga V2G
v2g soc <percent>       - Definir SOC da bateria
fault <connector>       - Simular falha no conector
reset                   - Simular reset do dispositivo
firmware accept|reject  - Responder a atualização de firmware
quit                    - Sair do simulador
```

---

## Endpoints REST a Implementar

```
POST /api/v1/devices/:id/remote-start    → RemoteStartTransaction
POST /api/v1/devices/:id/remote-stop     → RemoteStopTransaction
POST /api/v1/devices/:id/reset           → Reset
POST /api/v1/devices/:id/trigger/:msg    → TriggerMessage

POST /api/v1/devices/:id/charging-profile   → SetChargingProfile
DELETE /api/v1/devices/:id/charging-profile → ClearChargingProfile

POST /api/v1/devices/:id/firmware/update → UpdateFirmware
GET /api/v1/devices/:id/firmware/status  → FirmwareStatusNotification

POST /api/v1/v2g/discharge/start         → StartDischarge
POST /api/v1/v2g/discharge/stop          → StopDischarge
GET /api/v1/v2g/session/:id              → GetV2GSession
GET /api/v1/v2g/capability/:deviceId     → CheckV2GCapability
POST /api/v1/v2g/preferences             → SetV2GPreferences
GET /api/v1/v2g/grid-price               → GetCurrentGridPrice
```

---

## Tópicos NATS Implementados

```
ocpp.command.sent              → Comando enviado para carregador
ocpp.command.response          → Resposta recebida do carregador
ocpp.command.timeout           → Timeout de comando

v2g.session.started            → Sessão V2G iniciada
v2g.session.updated            → Atualização de sessão (energia, potência)
v2g.session.completed          → Sessão concluída

firmware.update.started        → Atualização iniciada
firmware.update.progress       → Progresso (%)
firmware.update.completed      → Atualização concluída
firmware.update.failed         → Falha na atualização
```

---

## Estatísticas

| Métrica | Valor |
|---------|-------|
| Linhas de código adicionadas | ~2.500 |
| Arquivos criados | 8 |
| Arquivos modificados | 4 |
| Mensagens OCPP implementadas | 20+ |
| Interfaces de serviço | 5 |

---

## Próximos Passos

1. [x] Implementar handlers HTTP/REST para os novos endpoints
2. [x] Criar repositório V2G (PostgreSQL)
3. [x] Integrar com sistema de pagamentos para compensação V2G
4. [x] Adicionar testes unitários
5. [x] Integrar com API CCEE para preços reais da rede
6. [x] Adicionar suporte a certificados ISO 15118

---

## Fase 2 - Implementação Completa (Fevereiro 2026)

### Novos Arquivos Criados

#### 12. `internal/adapter/http/fiber/handlers/device_commands.go` (~400 linhas)
- Handler RemoteStart, RemoteStop, Reset
- Handler TriggerMessage
- Handler SetChargingProfile, ClearChargingProfile
- Handler UpdateFirmware, GetFirmwareStatus
- Handler UnlockConnector, ChangeAvailability

#### 13. `internal/adapter/http/fiber/handlers/v2g.go` (~400 linhas)
- Handler StartDischarge, StopDischarge
- Handler GetSession, GetActiveSession
- Handler GetCapability
- Handler GetCurrentGridPrice, GetPriceForecast
- Handler GetPreferences, SetPreferences
- Handler GetUserStats, CalculateCompensation
- Handler OptimizeV2G

#### 14. `internal/adapter/storage/postgres/v2g_repository.go` (~350 linhas)
- CreateSession, UpdateSession, GetSession
- GetSessionsByChargePoint, GetSessionsByUser
- GetActiveSessions
- SavePreferences, GetPreferences
- CreateEvent, GetEventsBySession
- GetUserStats, GetChargePointStats, GetGlobalStats
- GetPendingCompensations, MarkCompensationPaid

#### 15. `internal/adapter/storage/migrations/002_v2g_tables.sql` (~330 linhas)
- Tabela `v2g_sessions` - Sessões de V2G
- Tabela `v2g_preferences` - Preferências de usuário
- Tabela `v2g_events` - Eventos para auditoria
- Tabela `v2g_compensations` - Compensações de V2G
- Tabela `v2g_grid_prices` - Cache de preços da rede
- Tabela `v2g_capabilities` - Capacidades V2G detectadas
- Tabela `iso15118_certificates` - Certificados Plug & Charge
- Tabela `firmware_updates` - Atualizações de firmware
- Índices e triggers para updated_at

#### 16. `internal/service/v2g/payment_service.go` (~290 linhas)
- V2GPaymentService para compensação de V2G
- CalculateAndRecordCompensation
- ProcessPayout (pagamento para carteira do usuário)
- ProcessSessionCompensation (fluxo completo)
- BatchProcessPendingPayouts
- GenerateCompensationReport

#### 17. `internal/service/v2g/ccee_client.go` (~350 linhas)
- CCEEClient para API da CCEE (Câmara de Comercialização de Energia Elétrica)
- GetCurrentPLD (Preço de Liquidação das Diferenças)
- GetPrices (preços por região e período)
- Fallback com preços simulados
- ConvertPLDToRetail (conversão MWh para kWh com impostos)
- Suporte a 4 regiões: SE/CO, S, NE, N
- Modelo de tarifação brasileiro (pesada/média/leve)

#### 18. `internal/service/v2g/iso15118_service.go` (~450 linhas)
- ISO15118Service para Plug & Charge
- AuthenticateVehicle - Autenticação via certificado X.509
- ValidateCertificate - Validação de cadeia de certificados
- GetChargingContract - Obter contrato de carregamento
- RevokeCertificate - Revogar certificado
- InstallCertificate - Instalar novo certificado
- GetCertificateStatus - Status do certificado
- Cache de validações
- Extração de identidade do veículo (EMAID, VIN, ContractID)

#### 19. `internal/adapter/storage/postgres/iso15118_repository.go` (~250 linhas)
- ISO15118Repository para persistência de certificados
- StoreCertificate, GetCertificateByEMAID
- GetCertificateByContractID, GetCertificateByVIN
- UpdateCertificate, DeleteCertificate
- GetExpiringCertificates, GetV2GCapableCertificates
- GetCertificateStats

### Testes Unitários Criados

#### 20. `internal/service/v2g/service_test.go` (~350 linhas)
- MockV2GRepository, MockGridPriceService, MockOCPPCommandService
- TestV2GService_CheckV2GCapability
- TestV2GService_SetUserPreferences
- TestV2GService_GetUserPreferences
- TestV2GService_GetUserStats
- TestV2GService_CalculateCompensation

#### 21. `internal/service/v2g/grid_price_test.go` (~250 linhas)
- TestGridPriceService_GetCurrentPrice
- TestGridPriceService_IsPeakHour
- TestGridPriceService_GetPriceForecast
- TestGridPriceService_CalculateV2GCompensation
- TestGridPriceService_PeakHourPricing
- TestGridPriceService_BrazilianTariffStructure

#### 22. `internal/service/v2g/ccee_client_test.go` (~300 linhas)
- TestCCEEClient_DefaultConfig
- TestCCEEClient_GetSimulatedPrices
- TestCCEEClient_RegionalPricing
- TestCCEEClient_GetCurrentPLD
- TestCCEEClient_ConvertPLDToRetail
- TestCCEEClient_GetLoadLevel
- TestCCEEClient_SimulatedPricingModel

#### 23. `internal/service/v2g/iso15118_service_test.go` (~400 linhas)
- MockISO15118Repository
- Helper para gerar certificados de teste
- TestISO15118Service_ParseCertificateChain
- TestISO15118Service_ExtractVehicleIdentity
- TestISO15118Service_ValidateCertificate
- TestISO15118Service_AuthenticateVehicle
- TestISO15118Service_InstallCertificate
- TestISO15118Service_RevokeCertificate
- TestISO15118Service_GetCertificateStatus
- TestISO15118Service_GetChargingContract

#### 24. `internal/service/v2g/payment_service_test.go` (~350 linhas)
- MockWalletService, MockMessageQueue
- TestPaymentConfig_Defaults
- TestV2GPaymentService_CalculateAndRecordCompensation
- TestV2GPaymentService_ProcessPayout
- TestV2GPaymentService_ProcessSessionCompensation
- TestV2GPaymentService_OperatorMarginCalculation
- TestV2GPaymentService_MultiplePayouts

---

## Estatísticas Atualizadas

| Métrica | Valor |
|---------|-------|
| Linhas de código adicionadas | ~6.500 |
| Arquivos criados | 18 |
| Arquivos modificados | 6 |
| Mensagens OCPP implementadas | 25+ |
| Interfaces de serviço | 8 |
| Endpoints REST | 20+ |
| Tabelas PostgreSQL | 8 |
| Testes unitários | 50+ |

---

## Configuração do Ambiente

O servidor de produção (`eva-ia.org`) já possui:
- PostgreSQL 16 configurado
- Redis 7.0 configurado
- NATS JetStream configurado
- Arquivo `.env` com todas as variáveis

### Variáveis de Ambiente Necessárias

```env
# Banco de dados
DATABASE_URL=postgres://...

# CCEE (Preços de energia)
CCEE_API_KEY=...
CCEE_BASE_URL=https://api.ccee.org.br/v1

# Pagamentos
STRIPE_SECRET_KEY=...
WALLET_SERVICE_URL=...

# ISO 15118
ISO15118_ROOT_CA_PATH=/certs/v2g-root-ca.pem
ISO15118_SUB_CA_PATH=/certs/v2g-sub-ca.pem
ISO15118_CPO_CERT_PATH=/certs/cpo-cert.pem
ISO15118_CPO_KEY_PATH=/certs/cpo-key.pem
```

---

*Implementação Fase 1: Fevereiro 2026*
*Implementação Fase 2: Fevereiro 2026*
