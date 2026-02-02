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

1. [ ] Implementar handlers HTTP/REST para os novos endpoints
2. [ ] Criar repositório V2G (PostgreSQL)
3. [ ] Integrar com sistema de pagamentos para compensação V2G
4. [ ] Adicionar testes unitários
5. [ ] Integrar com API CCEE para preços reais da rede
6. [ ] Adicionar suporte a certificados ISO 15118

---

*Implementado em: Fevereiro 2026*
