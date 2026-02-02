# SIGEC-VE Enterprise - Tarefas

> Documento de acompanhamento de tarefas baseado na auditoria de 02/02/2026

---

## TAREFAS CONCLUIDAS

### Bugs Corrigidos
- [x] **CircuitBreaker logger global** - Corrigido em `internal/adapter/http/fiber/middleware/circuit_breaker.go`
  - Removida variavel global `var logger *zap.Logger`
  - Criado `CircuitBreakerConfig` struct com configuracao
  - Adicionado `CircuitBreakerWithLogger()` e `CircuitBreakerWithConfig()`
  - Logger usa `zap.NewNop()` como fallback seguro

- [x] **FindNearby sem geolocalizacao** - Corrigido em `internal/adapter/storage/postgres/charge_point_repository.go`
  - Implementada formula Haversine em SQL puro
  - Calcula distancia real em km entre coordenadas
  - Ordena por proximidade (mais perto primeiro)
  - Preload de Connectors e Location mantido
  - Limite de 50 resultados

---

## TAREFAS PENDENTES - PRIORIDADE CRITICA

### Pagamentos (Stripe Mock)
- [ ] **Implementar Stripe real** - `internal/adapter/external/payment/stripe.go`
  - Atual: Retorna `pi_mock_123456789` hardcoded
  - Necessario: Integrar SDK Stripe real
  - Implementar CreatePaymentIntent com Stripe API
  - Implementar ConfirmPayment
  - Implementar RefundPayment
  - Configurar webhooks

### Injecao de Dependencias
- [ ] **Injetar BillingService em TransactionService** - `internal/service/transaction/service.go`
  - Atual: TransactionService usa preco hardcoded (0.75 BRL/kWh)
  - Necessario: Injetar BillingService no construtor
  - Usar `billingService.CalculateCost()` em `StopTransaction()`
  - Beneficio: Peak pricing e idle fee funcionarao

---

## TAREFAS PENDENTES - PRIORIDADE ALTA

### Notificacoes (Adapters Vazios)
- [ ] **Implementar EmailService adapter** - `internal/adapter/external/notification/email.go`
  - Atual: Arquivo com 2 linhas (vazio)
  - Necessario: Integrar SendGrid ou SMTP
  - Conectar com `internal/service/email/service.go` que ja existe

- [ ] **Implementar SMSService adapter** - `internal/adapter/external/notification/sms.go`
  - Atual: Arquivo com 2 linhas (vazio)
  - Necessario: Integrar Twilio

- [ ] **Implementar PushService adapter** - `internal/adapter/external/notification/push.go`
  - Atual: Arquivo com 2 linhas (vazio)
  - Necessario: Integrar Firebase Cloud Messaging

### Middleware
- [ ] **Implementar CORS middleware** - `internal/adapter/http/fiber/middleware/cors.go`
  - Atual: Arquivo com 2 linhas (vazio)
  - Necessario: Configurar origens permitidas

### Limpeza de Codigo
- [ ] **Remover arquivos duplicados de repositorio**
  - `internal/adapter/storage/postgres/user_repo.go` (2 linhas - VAZIO)
  - `internal/adapter/storage/postgres/charge_point_repo.go` (2 linhas - VAZIO)
  - `internal/adapter/storage/postgres/transaction_repo.go` (2 linhas - VAZIO)
  - Manter apenas os `*_repository.go`

---

## TAREFAS PENDENTES - PRIORIDADE MEDIA

### AdminService (14 metodos vazios)
- [ ] **Implementar GetDashboardStats** - `internal/service/admin/service.go`
- [ ] **Implementar GetRevenueStats**
- [ ] **Implementar GetUsageStats**
- [ ] **Implementar user management (CRUD)**
- [ ] **Implementar station management**
- [ ] **Implementar transaction management**
- [ ] **Implementar alerts system**
- [ ] **Implementar reports generation**

### Stubs em Services
- [ ] **SmartChargingService.GetChargingProfile** - `internal/service/transaction/smart_charging.go:303-314`
  - Atual: Retorna nil sempre
  - Necessario: Buscar profile do database/cache

- [ ] **ReservationService.GetReservationSummary** - `internal/service/reservation/service.go:432-445`
  - Atual: Retorna zeros
  - Necessario: Implementar agregacoes do banco

- [ ] **EnergyAnalytics.PredictDemand** - `internal/service/analytics/energy_analytics.go:39-42`
  - Atual: Retorna 0, nil (stub)
  - Necessario: Integrar modelo ML ou heuristica

### Cache Local
- [ ] **Implementar cache local** - `internal/adapter/cache/local.go`
  - Atual: Arquivo com 2 linhas (vazio)
  - Necessario: Implementar cache in-memory com TTL
  - Usar como fallback do Redis

### RBAC/OAuth2
- [ ] **Implementar jwt_service.go** - `internal/service/auth/jwt_service.go`
  - Atual: Vazio
  - Necessario: Extrair logica JWT do service.go

- [ ] **Implementar rbac_service.go** - `internal/service/auth/rbac_service.go`
  - Atual: Vazio
  - Necessario: Role-based access control refinado

- [ ] **Implementar oauth2_service.go** - `internal/service/auth/oauth2_service.go`
  - Atual: Vazio (presumido)
  - Necessario: OAuth2 providers (Google, GitHub)

---

## TAREFAS PENDENTES - PRIORIDADE BAIXA

### AI Adapters (Futuros)
- [ ] **Implementar OpenAI adapter** - `internal/adapter/ai/openai/embeddings.go`
  - Atual: Arquivo com 2 linhas (vazio)
  - Para: Fallback de embeddings/completions

- [ ] **Implementar Anthropic MCP** - `internal/adapter/ai/anthropic/mcp_client.go`
  - Atual: Arquivo com 2 linhas (vazio)
  - Para: Model Context Protocol

### Queue Alternativa
- [ ] **Implementar RabbitMQ adapter** - `internal/adapter/queue/rabbitmq.go`
  - Atual: Arquivo com 2 linhas (vazio)
  - Para: Alternativa ao NATS

### gRPC
- [ ] **Implementar gRPC interceptors** - `internal/adapter/grpc/interceptors/`
  - auth.go, logging.go, metrics.go (vazios)

- [ ] **Completar gRPC handlers** - `internal/adapter/grpc/server/server.go`
  - ListDevices() - retorna vazio
  - UpdateDeviceStatus() - retorna vazio
  - StreamDeviceEvents() - retorna vazio

### OCPP 1.6
- [ ] **Implementar OCPP v1.6** - `internal/adapter/ocpp/v16/`
  - handlers.go e server.go vazios
  - Para: Compatibilidade com carregadores antigos

---

## TESTES PENDENTES

### Cobertura de Testes
- [ ] **Adicionar testes E2E** - `tests/e2e/`
  - Pasta existe mas vazia
  - Usar Selenium/Playwright

- [ ] **Aumentar cobertura unitaria para 70%**
  - Services SEM testes: Billing, SmartCharging, Voice, Payment, Reservation, Analytics, Admin

---

## DOCUMENTACAO

- [ ] **Atualizar CLAUDE.md**
  - Adicionar servicos nao documentados: Admin, Email, Payment, Reservation, Wallet, Card
  - Corrigir contagem de arquivos (121, nao 91)
  - Corrigir contagem de linhas
  - Documentar novos metodos do CircuitBreaker

---

## METRICAS DE PROGRESSO

| Categoria | Antes | Depois |
|-----------|-------|--------|
| Bugs criticos | 2 | 0 |
| Codigo implementado | ~40% | ~42% |
| Adapters vazios | 15 | 15 |
| Services com stubs | 5 | 5 |

---

*Ultima atualizacao: 02/02/2026*
