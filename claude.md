# SIGEC-VE Enterprise - Documentacao para Claude

> **Proposito deste documento**: Fornecer contexto completo sobre o projeto para que o Claude possa entender rapidamente a arquitetura, tecnologias e funcionalidades ao iniciar uma nova sessao.

---

## 1. VISAO GERAL DO PROJETO

**SIGEC-VE Enterprise** e uma **Plataforma Enterprise de Gestao de Estacoes de Carregamento de Veiculos Eletricos** com recursos avancados de:
- Atendimento por voz (Gemini Live API) em portugues
- Protocolo OCPP 2.0.1 para comunicacao com carregadores fisicos
- Observabilidade completa (OpenTelemetry)
- Suporte a 100k+ conexoes simultaneas

### Objetivo Principal
Gerenciar estacoes de carregamento de VE em larga escala com interface de voz ("EVA, iniciar carregamento"), dashboard em tempo real, pagamentos integrados e observabilidade completa.

### Contexto de Uso
- Multinacionais e operadores de larga escala de estacoes de carregamento
- Mobilidade eletrica - substituicao de combustiveis fosseis
- Ambientes urbanos e rodoviarios

---

## 2. STACK TECNOLOGICO

### Backend
| Tecnologia | Versao | Proposito |
|------------|--------|-----------|
| **Go** | 1.22 | Linguagem principal |
| **Fiber** | v2.52.0 | Framework HTTP (similar Express.js) |
| **gRPC** | - | Comunicacao RPC alta performance |
| **GORM** | v1.25.7 | ORM para PostgreSQL |

### Banco de Dados e Cache
| Tecnologia | Versao | Proposito |
|------------|--------|-----------|
| **PostgreSQL** | 16 | Banco relacional principal |
| **Redis** | 7 | Cache distribuido e sessoes |
| **NATS JetStream** | - | Message broker e event streaming |

### Inteligencia Artificial e Voz
| Tecnologia | Proposito |
|------------|-----------|
| **Google Gemini Live API 2.0** | Processamento de voz bidirecional |
| **OpenAI** | Estrutura preparada (futuro) |
| **Anthropic MCP** | Estrutura preparada (futuro) |

### Autenticacao e Seguranca
- **JWT** (golang-jwt/jwt/v5) - Autenticacao stateless
- **bcrypt** - Hash seguro de senhas
- **TLS 1.3 / mTLS** - Encriptacao de transporte

### Observabilidade
- **OpenTelemetry** + **Jaeger** - Distributed tracing
- **Prometheus** - Metricas
- **Grafana** - Dashboards
- **Zap** - Logger estruturado JSON

### DevOps
- **Docker** (imagem distroless ~20MB)
- **Kubernetes** (Helm Charts, HPA, Kustomize)
- **GitHub Actions** - CI/CD

---

## 3. ARQUITETURA

### Padrao: Hexagonal (Clean Architecture)

```
┌─────────────────────────────────────────────────────────────┐
│           Presentation / External Layer                      │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐    │
│  │   REST   │  │ GraphQL  │  │   gRPC   │  │   OCPP   │    │
│  │   API    │  │  Server  │  │  Server  │  │ WebSocket│    │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘    │
└────────────────────────┬────────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────────┐
│              Application / Service Layer                     │
│  ┌─────────────────────────────────────────────────────┐    │
│  │       Business Logic (Services)                      │    │
│  │  • AuthService      • DeviceService                  │    │
│  │  • TransactionService   • VoiceAssistant             │    │
│  │  • BillingService   • AnalyticsService               │    │
│  └─────────────────────────────────────────────────────┘    │
└────────────────────────┬────────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────────┐
│                  Domain Layer                                │
│  ┌──────────────────────────────────────────────────────┐   │
│  │      Pure Domain Models (Entities)                    │   │
│  │  • User           • ChargePoint                       │   │
│  │  • Transaction    • Connector                         │   │
│  │  • Location       • VoiceResponse                     │   │
│  └──────────────────────────────────────────────────────┘   │
│  ┌──────────────────────────────────────────────────────┐   │
│  │     Ports (Interfaces - Contracts)                    │   │
│  │  • Repositories   • Services  • Cache  • Payment      │   │
│  └──────────────────────────────────────────────────────┘   │
└────────────────────────┬────────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────────┐
│             Infrastructure / Adapter Layer                   │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐    │
│  │PostgreSQL│  │  Redis   │  │   NATS   │  │  Gemini  │    │
│  │  (GORM)  │  │  Cache   │  │  Queue   │  │ Live API │    │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘    │
└─────────────────────────────────────────────────────────────┘
```

### Portas de Servico
| Servico | Porta | Protocolo |
|---------|-------|-----------|
| HTTP/REST API | 8080 | HTTP/HTTPS |
| OCPP WebSocket | 9000 | WebSocket |
| gRPC | 50051 | HTTP/2 |

---

## 4. ESTRUTURA DE PASTAS

```
sigec-ve-enterprise/
├── api/                           # Definicoes de API
│   ├── graphql/                   # Schemas GraphQL
│   ├── openapi/                   # Swagger OpenAPI 3.0
│   └── proto/                     # Protocol Buffers (gRPC)
│       ├── device/v1/
│       ├── transaction/v1/
│       └── voice/v1/
│
├── cmd/                           # Pontos de entrada (binarios)
│   ├── server/main.go             # Servidor principal
│   ├── worker/main.go             # Worker de background
│   └── migrator/main.go           # CLI para migrations
│
├── internal/                      # Codigo privado
│   ├── adapter/                   # Adapters (Infraestrutura)
│   │   ├── ai/                    # IA: anthropic, gemini, openai
│   │   ├── cache/                 # Redis, local
│   │   ├── external/              # notification, payment
│   │   ├── grpc/                  # servidor gRPC
│   │   ├── http/fiber/            # Servidor HTTP Fiber
│   │   │   ├── handlers/          # Auth, Device, Transaction, Voice
│   │   │   └── middleware/        # Auth, CORS, RateLimit, CircuitBreaker
│   │   ├── ocpp/v201/             # OCPP 2.0.1 WebSocket
│   │   ├── queue/                 # NATS, RabbitMQ
│   │   ├── storage/postgres/      # PostgreSQL + GORM
│   │   ├── vault/                 # Secrets
│   │   └── websocket/             # Real-time updates
│   │
│   ├── domain/                    # Domain Layer (Entidades)
│   │   ├── charge_point.go
│   │   ├── transaction.go
│   │   ├── user.go
│   │   └── voice.go
│   │
│   ├── ports/                     # Interfaces (Contratos)
│   │   ├── repositories.go
│   │   ├── services.go
│   │   ├── cache.go
│   │   └── payment.go
│   │
│   ├── service/                   # Business Logic
│   │   ├── auth/                  # JWT, OAuth2, RBAC
│   │   ├── device/                # Dispositivos, Firmware
│   │   ├── transaction/           # Billing, SmartCharging
│   │   ├── analytics/             # Previsao, Energia
│   │   └── voice/                 # Assistente de voz
│   │
│   └── observability/             # Health, Telemetry
│
├── pkg/config/                    # Configuracao publica
├── configs/                       # Arquivos YAML
├── deployments/                   # Docker, Kubernetes, Terraform
├── scripts/                       # Scripts utilitarios
├── tests/                         # unit, integration, e2e, load
├── frontend/                      # Frontend (estrutura)
└── docs/                          # Documentacao
```

---

## 5. ENTIDADES DE DOMINIO

### User
```go
type User struct {
    ID        string    // UUID
    Name      string
    Email     string    // Unique
    Password  string    // Hashed bcrypt
    Role      UserRole  // admin, operator, user
    Status    string    // Active, Inactive, Blocked
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

### ChargePoint (Estacao de Carregamento)
```go
type ChargePoint struct {
    ID              string            // Identificador unico
    Vendor          string            // ABB, Siemens, Tesla
    Model           string            // Terra 184, etc.
    SerialNumber    string
    FirmwareVersion string
    Status          ChargePointStatus // Available, Occupied, Faulted, Unavailable
    Location        *Location         // Lat/Lng
    Connectors      []Connector       // CCS, CHAdeMO, Type2
    LastSeen        time.Time
}

type Connector struct {
    ID            uint
    ChargePointID string
    ConnectorID   int               // 1-based (OCPP)
    Type          string            // CCS, CHAdeMO, Type2
    Status        ChargePointStatus
    MaxPowerKW    float64           // 150kW, 350kW
}
```

### Transaction (Sessao de Carregamento)
```go
type Transaction struct {
    ID            string
    ChargePointID string
    ConnectorID   int
    UserID        string
    IdTag         string            // RFID ou auth token
    StartTime     time.Time
    EndTime       *time.Time        // Null se ativa
    MeterStart    int               // Wh
    MeterStop     int               // Wh
    TotalEnergy   int               // kWh
    Status        TransactionStatus // Started, Stopped, Completed
    Cost          float64
    Currency      string            // BRL
}
```

### VoiceResponse
```go
type VoiceResponse struct {
    Text         string   // Texto transcrito
    Audio        string   // Base64 PCM16
    Intent       string   // check_status, start_charge, stop_charge
    Confidence   float64
    ActionResult string
}
```

---

## 6. SERVICES (LOGICA DE NEGOCIO)

### AuthService
- `Login(email, password)` -> (token, refreshToken)
- `Register(user)` -> error
- `RefreshToken(refreshToken)` -> token
- `ValidateToken(token)` -> *User

### DeviceService
- `GetDevice(id)` -> *ChargePoint
- `ListDevices(filter)` -> []ChargePoint
- `UpdateStatus(id, status)` -> error
- `GetNearby(lat, lon, radius)` -> []ChargePoint

### TransactionService
- `StartTransaction(deviceID, connectorID, userID)` -> *Transaction
- `StopTransaction(txID)` -> *Transaction
- `GetActiveTransaction(userID)` -> *Transaction
- `GetTransactionHistory(userID)` -> []Transaction

### VoiceAssistant (Gemini Live API)
- `ProcessVoiceCommand(ctx, userID, audioChunk)` -> *VoiceResponse
- Intents: check_status, start_charge, stop_charge, check_cost, report_issue

---

## 7. API ENDPOINTS

### REST API v1 (`/api/v1`)

#### Autenticacao
```
POST /auth/login          -> { token, refresh_token }
POST /auth/register       -> { id, email, name, role }
POST /auth/refresh        -> { token }
```

#### Dispositivos (protegido)
```
GET  /devices             -> []ChargePoint
GET  /devices/:id         -> ChargePoint
GET  /devices/nearby      -> []ChargePoint (por localizacao)
PATCH /devices/:id/status -> 200 OK
```

#### Transacoes
```
POST /transactions/start  -> { id, status, start_time }
POST /transactions/:id/stop -> { id, total_energy, cost }
GET  /transactions/:id    -> Transaction
GET  /transactions/active -> Transaction (ativa do usuario)
GET  /transactions/history -> []Transaction
```

#### Voz
```
POST /voice/command       -> { text, audio, intent, action_result }
GET  /voice/history       -> []VoiceCommand
```

#### WebSocket
```
GET /ws/updates           -> Real-time device updates
GET /ws/voice             -> Stream bidirecional voz
```

#### Health e Metricas
```
GET /health/live          -> "OK"
GET /health/ready         -> "Ready"
GET /metrics              -> Prometheus format
```

---

## 8. CONFIGURACAO PRINCIPAL

### Arquivo: `configs/config.yaml`

```yaml
app:
  name: sigec-ve-enterprise
  version: v1.0.0
  environment: production

http:
  port: 8080
  read_timeout: 30s
  write_timeout: 30s

grpc:
  port: 50051
  max_connections: 1000

ocpp:
  port: 9000
  version: 2.0.1
  heartbeat_interval: 300s
  security:
    enabled: true
    tls_cert: /certs/server.crt
    tls_key: /certs/server.key
    client_auth: true  # mTLS

database:
  url: postgres://admin:password@postgres:5432/sigec?sslmode=require
  max_open_conns: 100
  auto_migrate: true

redis:
  url: redis://:password@redis:6379/0
  pool_size: 100

nats:
  url: nats://nats:4222

jwt:
  secret: ${JWT_SECRET}
  access_token_duration: 15m
  refresh_token_duration: 7d

gemini:
  api_key: ${GEMINI_API_KEY}
  model: gemini-2.0-flash-exp
  voice_config:
    voice_name: Aoede
    language: pt-BR

opentelemetry:
  enabled: true
  jaeger:
    endpoint: http://jaeger:14268/api/traces

payment:
  stripe:
    secret_key: ${STRIPE_SECRET_KEY}
    currency: BRL
  pricing:
    per_kwh: 0.75  # R$/kWh
```

### Variaveis de Ambiente Criticas
```
JWT_SECRET
GEMINI_API_KEY
DATABASE_URL
REDIS_URL
STRIPE_SECRET_KEY
STRIPE_WEBHOOK_SECRET
SENDGRID_API_KEY
TWILIO_ACCOUNT_SID
TWILIO_AUTH_TOKEN
```

---

## 9. PADROES DE CODIGO

### 1. Hexagonal Architecture
```go
// Domain Layer - Pure Business Logic
type User struct { ... }

// Port - Interface Contract
type UserRepository interface {
    Save(ctx context.Context, user *User) error
    FindByEmail(ctx context.Context, email string) (*User, error)
}

// Adapter - Implementacao Concreta
type PostgresUserRepository struct { db *gorm.DB }

// Service - Logica de Negocio
type AuthService struct {
    userRepo ports.UserRepository
    cache    ports.Cache
}
```

### 2. Dependency Injection
```go
// main.go
authService := auth.NewService(userRepo, redisCache, cfg.JWT.Secret, logger)
authHandler := handlers.NewAuthHandler(authService, logger)
```

### 3. Repository Pattern
```go
type ChargePointRepository interface {
    Save(ctx context.Context, cp *ChargePoint) error
    FindByID(ctx context.Context, id string) (*ChargePoint, error)
    FindNearby(ctx context.Context, lat, lon, radius float64) ([]ChargePoint, error)
}
```

### 4. Middleware Chain
```go
app.Use(recover.New())
app.Use(logger.New())
app.Use(cors.New(...))
app.Use(middleware.RateLimit())
app.Use(middleware.CircuitBreaker())

protected := v1.Group("", middleware.AuthRequired(authService))
```

---

## 10. PROTOCOLO OCPP 2.0.1

### Mensagens Principais (WebSocket JSON-RPC)

```json
// BootNotification - Carregador registra ao iniciar
[2, "request-id", "BootNotification", {
  "chargingStation": { "model": "Terra 184", "vendorName": "ABB" }
}]

// Heartbeat - Keep-alive
[2, "hb-123", "Heartbeat", {}]

// StatusNotification - Atualiza status
[2, "sn-456", "StatusNotification", {
  "connectorId": 1,
  "connectorStatus": "Available"
}]

// TransactionEvent - Inicio/fim de recarga
[2, "te-789", "TransactionEvent", {
  "eventType": "Started",
  "transactionInfo": { "transactionId": "tx-001" }
}]
```

---

## 11. COMANDOS UTEIS

### Desenvolvimento
```bash
make dev              # Docker Compose: API + DB + Redis + NATS + Jaeger
make run              # Rodar servidor local
make build            # Compilar binario
make test             # Rodar testes
make test-coverage    # Testes com cobertura
make lint             # Linter (golangci-lint)
make fmt              # Formatar codigo
```

### Docker
```bash
docker-compose up -d          # Subir ambiente
docker-compose logs -f api    # Ver logs
docker-compose down           # Parar ambiente
```

### Kubernetes
```bash
kubectl apply -k deployments/kubernetes/overlays/dev
helm install sigec deployments/kubernetes/helm/sigec-ve
```

---

## 12. STATUS DO PROJETO

### Implementado
- [x] Arquitetura Hexagonal completa
- [x] Core models (User, ChargePoint, Transaction)
- [x] PostgreSQL adapter com GORM
- [x] Redis cache
- [x] NATS messaging
- [x] JWT authentication
- [x] REST API com Fiber
- [x] gRPC basics
- [x] OCPP 2.0.1 servidor
- [x] Gemini Live API integration
- [x] OpenTelemetry + Jaeger
- [x] Docker + Kubernetes
- [x] CI/CD GitHub Actions
- [x] Background workers

### Pendente
- [ ] Integracao real Stripe (payment)
- [ ] Notificacoes (Email, SMS, Push)
- [ ] OCPP completo (RemoteStart, FirmwareUpdate)
- [ ] RBAC refinado
- [ ] Testes unitarios/integracao
- [ ] Frontend web app
- [ ] Analytics e previsao
- [ ] V2G (Vehicle-to-Grid)
- [ ] Blockchain audit trail

---

## 13. METRICAS DO PROJETO

- **91 arquivos Go**
- **~3.239 linhas de codigo Go**
- **456 linhas em HTTP handlers**
- **899 linhas em Services**
- **Cobertura de testes**: ~0% (a implementar)

### Performance Targets
- Throughput: 50k req/s
- Latencia p50: 15ms
- Latencia p95: 45ms
- Conexoes OCPP simultaneas: 100k+

---

## 14. FLUXO DE INICIALIZACAO

```
1. Logger (Zap)
2. Config (Viper)
3. OpenTelemetry (Jaeger)
4. PostgreSQL (GORM + migrations)
5. Redis Cache
6. NATS Queue
7. Repositories
8. Services
9. Gemini Live API Client
10. OCPP WebSocket Server (9000)
11. WebSocket Hub
12. Fiber HTTP Server (8080)
13. gRPC Server (50051)
14. Background Workers
15. Graceful Shutdown Handler
```

---

## 15. INFORMACOES PARA DESENVOLVIMENTO

### Adicionar nova funcionalidade:
1. Criar entidade em `internal/domain/`
2. Definir interface (port) em `internal/ports/`
3. Implementar adapter em `internal/adapter/`
4. Criar service em `internal/service/`
5. Criar handler em `internal/adapter/http/fiber/handlers/`
6. Adicionar rotas em `internal/adapter/http/fiber/server.go`

### Adicionar novo endpoint OCPP:
1. Editar `internal/adapter/ocpp/v201/handlers.go`
2. Implementar handler para a mensagem
3. Registrar no `handleMessage()` switch

### Adicionar integracao externa:
1. Criar adapter em `internal/adapter/external/`
2. Definir interface em `internal/ports/`
3. Injetar no service relevante

---

*Documento gerado em: Fevereiro 2025*
*Versao do projeto: v1.0.0*
