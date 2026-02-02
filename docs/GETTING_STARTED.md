# Como Rodar o SIGEC-VE Enterprise

## Pré-requisitos

O servidor já tem configurado:
- PostgreSQL 16 rodando
- Redis 7.0 rodando
- NATS JetStream rodando
- Arquivo `.env` configurado

---

## Passo 1: Instalar Dependências Go

```bash
cd D:\dev\EVA\EV\sigec-ve-enterprise

# Instalar dependências
go mod download
go mod verify
```

---

## Passo 2: Rodar Migrations (Criar Tabelas)

```bash
# Opção 1: Via psql (recomendado para primeiro setup)
psql $DATABASE_URL -f internal/adapter/storage/migrations/001_initial_schema.sql
psql $DATABASE_URL -f internal/adapter/storage/migrations/002_v2g_tables.sql

# Opção 2: Via migrator (se configurado)
go run ./cmd/migrator/main.go up
```

---

## Passo 3: Iniciar o Servidor

### Terminal 1 - Servidor Principal

```bash
# Via make
make run

# Ou diretamente
go run ./cmd/server/main.go
```

O servidor inicia:
- **REST API**: http://localhost:8080
- **OCPP WebSocket**: ws://localhost:9000/ocpp
- **gRPC**: localhost:50051

### Verificar se está rodando

```bash
# Health check
curl http://localhost:8080/health/live

# Resposta esperada:
# OK
```

---

## Passo 4: Rodar o Simulador OCPP

### Terminal 2 - Simulador de Charge Point

```bash
# Simulador básico
go run ./cmd/simulator/main.go --id=CP001 --server=ws://localhost:9000/ocpp

# Simulador com V2G habilitado
go run ./cmd/simulator/main.go --id=CP001 --server=ws://localhost:9000/ocpp --v2g --soc=80 --battery=75

# Modo interativo (recomendado para testar)
go run ./cmd/simulator/main.go --id=CP001 --server=ws://localhost:9000/ocpp --v2g --interactive
```

### Comandos do Modo Interativo

```
help                    - Mostra comandos disponíveis
start <connector>       - Iniciar carregamento (ex: start 1)
stop                    - Parar carregamento atual
status <connector>      - Mudar status (Available/Occupied/Faulted)
meter <value>           - Enviar meter value (Wh)
heartbeat               - Enviar heartbeat
v2g start <power>       - Iniciar descarga V2G (ex: v2g start 30)
v2g stop                - Parar descarga V2G
v2g soc <percent>       - Definir SOC da bateria
fault <connector>       - Simular falha
quit                    - Sair
```

---

## Passo 5: Testar via API REST

### Terminal 3 - Testes com cURL

#### 5.1 Criar usuário e fazer login

```bash
# Registrar usuário
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "João Silva",
    "email": "joao@example.com",
    "password": "senha123"
  }'

# Login
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "joao@example.com",
    "password": "senha123"
  }'

# Resposta (salve o token):
# {"token": "eyJhbGciOiJIUzI1...", "refresh_token": "..."}
```

#### 5.2 Definir token como variável

```bash
# Windows PowerShell
$TOKEN = "eyJhbGciOiJIUzI1..."

# Linux/Mac
export TOKEN="eyJhbGciOiJIUzI1..."
```

#### 5.3 Testar endpoints de dispositivos

```bash
# Listar dispositivos conectados
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/devices

# Ver dispositivo específico
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/devices/CP001
```

#### 5.4 Comandos remotos (CSMS → Charge Point)

```bash
# Iniciar carregamento remoto
curl -X POST http://localhost:8080/api/v1/devices/CP001/remote-start \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"id_token": "USER123", "connector_id": 1}'

# Parar carregamento remoto
curl -X POST http://localhost:8080/api/v1/devices/CP001/remote-stop \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"transaction_id": "tx-001"}'

# Reset do dispositivo
curl -X POST http://localhost:8080/api/v1/devices/CP001/reset \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"type": "Immediate"}'

# Trigger de mensagem
curl -X POST http://localhost:8080/api/v1/devices/CP001/trigger/StatusNotification \
  -H "Authorization: Bearer $TOKEN"
```

#### 5.5 V2G - Vehicle-to-Grid

```bash
# Verificar capacidade V2G
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/v2g/capability/CP001

# Ver preço atual da energia
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/v2g/grid-price

# Previsão de preços (24h)
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/api/v1/v2g/grid-price/forecast?hours=24"

# Iniciar descarga V2G
curl -X POST http://localhost:8080/api/v1/v2g/discharge/start \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "charge_point_id": "CP001",
    "connector_id": 1,
    "max_power_kw": 30,
    "max_energy_kwh": 20,
    "min_battery_soc": 20
  }'

# Parar descarga V2G
curl -X POST http://localhost:8080/api/v1/v2g/discharge/stop \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"session_id": "session-uuid"}'

# Definir preferências V2G
curl -X POST http://localhost:8080/api/v1/v2g/preferences \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "auto_discharge": true,
    "min_grid_price": 0.90,
    "max_discharge_kwh": 30,
    "preserve_soc": 25,
    "notify_on_start": true,
    "notify_on_end": true
  }'

# Ver minhas preferências
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/v2g/preferences

# Estatísticas V2G
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/api/v1/v2g/stats?start_date=2026-01-01&end_date=2026-02-28"
```

---

## Exemplo de Fluxo Completo

### Cenário: Descarga V2G com Compensação

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         DEMONSTRAÇÃO V2G                                     │
└─────────────────────────────────────────────────────────────────────────────┘

Terminal 1 (Servidor):
$ go run ./cmd/server/main.go
INFO    Starting SIGEC-VE Enterprise    {"service": "sigec-ve-enterprise"}
INFO    REST API running on :8080
INFO    OCPP WebSocket running on :9000
INFO    gRPC running on :50051

Terminal 2 (Simulador):
$ go run ./cmd/simulator/main.go --id=CP001 --v2g --soc=85 --interactive

Connected to CSMS
> help
Available commands:
  start <connector>  - Start charging on connector
  stop              - Stop current transaction
  v2g start <kW>    - Start V2G discharge
  v2g stop          - Stop V2G discharge
  v2g soc <percent> - Set battery SOC
  status <conn>     - Set connector status
  quit              - Exit simulator

> v2g start 30
[2026-02-02 15:30:00] V2G Discharge started: 30 kW
[2026-02-02 15:30:01] TransactionEvent sent: Discharging, -30 kW

Terminal 3 (API Calls):
$ curl http://localhost:8080/api/v1/v2g/grid-price -H "Authorization: Bearer $TOKEN"
{
  "price": 0.95,
  "currency": "BRL",
  "unit": "kWh",
  "is_peak": true,
  "timestamp": "2026-02-02T15:30:00-03:00"
}

$ curl http://localhost:8080/api/v1/v2g/session/active/CP001 -H "Authorization: Bearer $TOKEN"
{
  "id": "v2g-session-123",
  "charge_point_id": "CP001",
  "direction": "Discharging",
  "actual_power_kw": -30,
  "energy_transferred": -5.5,
  "current_grid_price": 0.95,
  "current_soc": 78,
  "status": "Active"
}

Terminal 2 (Simulador):
> v2g soc 20
[2026-02-02 15:45:00] SOC updated to 20%
[2026-02-02 15:45:01] Min SOC reached, stopping discharge

>
[2026-02-02 15:45:02] V2G Session completed
[2026-02-02 15:45:02] Energy discharged: 25.5 kWh
[2026-02-02 15:45:02] Compensation: R$ 21.80

Terminal 3 (Verificar compensação):
$ curl http://localhost:8080/api/v1/v2g/stats -H "Authorization: Bearer $TOKEN"
{
  "total_sessions": 1,
  "total_energy_discharged_kwh": 25.5,
  "total_compensation": 21.80,
  "average_session_duration": "45m",
  "peak_hours_participation": 100
}
```

---

## Rodar Testes Automatizados

```bash
# Todos os testes
go test ./... -v

# Testes do V2G
go test ./internal/service/v2g/... -v

# Testes com coverage
go test ./... -cover -coverprofile=coverage.out
go tool cover -html=coverage.out
```

---

## Endpoints de Debug

```bash
# Métricas Prometheus
curl http://localhost:8080/metrics

# Health checks
curl http://localhost:8080/health/live
curl http://localhost:8080/health/ready

# Ver conexões OCPP ativas (se implementado)
curl http://localhost:8080/api/v1/admin/ocpp/connections \
  -H "Authorization: Bearer $TOKEN"
```

---

## Troubleshooting

### Erro: "Failed to connect to database"
```bash
# Verificar se PostgreSQL está rodando
psql -h localhost -U admin -c "SELECT 1"

# Verificar variável DATABASE_URL no .env
echo $DATABASE_URL
```

### Erro: "Failed to connect to Redis"
```bash
# Verificar se Redis está rodando
redis-cli ping
# Resposta: PONG
```

### Erro: "Failed to connect to NATS"
```bash
# Verificar se NATS está rodando
nats server check
```

### Simulador não conecta
```bash
# Verificar se o servidor está rodando na porta 9000
netstat -an | grep 9000

# Verificar logs do servidor
# Procure por "Client connected" ou erros de WebSocket
```

---

## Scripts Úteis

### Windows PowerShell

```powershell
# start-all.ps1
Start-Process -NoNewWindow powershell -ArgumentList "go run ./cmd/server/main.go"
Start-Sleep -Seconds 5
Start-Process -NoNewWindow powershell -ArgumentList "go run ./cmd/simulator/main.go --id=CP001 --v2g --interactive"
```

### Linux/Mac

```bash
#!/bin/bash
# start-all.sh

# Iniciar servidor em background
go run ./cmd/server/main.go &
SERVER_PID=$!
sleep 5

# Iniciar simulador
go run ./cmd/simulator/main.go --id=CP001 --v2g --interactive

# Cleanup
kill $SERVER_PID
```

---

*Atualizado em: Fevereiro 2026*
