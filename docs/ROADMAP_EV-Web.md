# EV-Web: Roadmap - Voice-First EV Charging Frontend

## Conceito

O cliente chega na estacao de carregamento, abre o **EV-Web** no celular e **fala com a EVA** para abastecer o carro. Todo o fluxo e por voz: identificar estacao, plugar, carregar, pagar, receber recibo. Zero toque em botoes.

```
"EVA, quero carregar meu carro"
    -> "Oi! Vi que voce esta na estacao 5. Qual conector? CCS ou Type 2?"
"CCS"
    -> "Perfeito. Plugue o cabo CCS no seu carro. Aviso quando detectar a conexao."
    -> [OCPP detecta plugin]
    -> "Carro conectado! Quer carregar ate 80% ou por tempo?"
"Ate 80%"
    -> "Iniciando carregamento. Custo estimado: R$45. Confirma?"
"Sim"
    -> [StartTransaction via OCPP 2.0.1]
    -> "Carregamento iniciado! Vou te avisar quando chegar em 80%."
    ...
    -> "Seu carro chegou em 80%! Total: R$42,30. Debito na carteira ou cartao?"
"Carteira"
    -> [Wallet debit + StopTransaction]
    -> "Pronto! Recibo enviado pro seu email. Boa viagem!"
```

---

## Stack Tecnica

Baseado na **EVA-Web** (stack comprovada em producao):

| Camada | Tecnologia | Motivo |
|--------|-----------|--------|
| **Framework** | React 18 + TypeScript | Mesmo da EVA-Web |
| **Build** | Vite 5 | Ultra-rapido, HMR |
| **Styling** | Tailwind CSS 3.4 | Mobile-first |
| **Audio** | Web Audio API + AudioWorklet | Captura mic 16kHz, playback 24kHz |
| **Comunicacao** | WebSocket (browser <-> EV-IA backend) | Real-time bidirecional |
| **Voice AI** | Gemini `gemini-2.5-flash-native-audio-preview-12-2025` | Via backend (INTOCAVEL) |
| **Maps** | Leaflet / Google Maps | Localizar estacoes |
| **Pagamento** | Stripe Elements | Cartao + Wallet |
| **PWA** | Workbox | Offline-first, install no celular |
| **State** | TanStack Query + Zustand | Server state + local state |
| **Notificacoes** | Web Push API + Sonner | Alertas de carregamento |

---

## Arquitetura

```
┌─────────────────────────────────────────────────────┐
│                   EV-Web (PWA)                       │
│               React + TypeScript + Vite              │
│                                                       │
│  ┌──────────┐  ┌──────────┐  ┌──────────────────┐  │
│  │ Voice UI │  │ Map View │  │ Charging Monitor │  │
│  │ (EVA)    │  │ (nearby) │  │ (real-time)      │  │
│  └────┬─────┘  └────┬─────┘  └────────┬─────────┘  │
│       │              │                  │             │
│  ┌────┴──────────────┴──────────────────┴──────────┐│
│  │          WebSocket + REST Client                 ││
│  │    ws://api:8082/ws/ev    GET /api/v1/...       ││
│  └──────────────────┬──────────────────────────────┘│
└─────────────────────┼───────────────────────────────┘
                      │
         ┌────────────┼────────────┐
         │     EV-IA Backend       │
         │     (Go + Fiber)        │
         │                         │
         │  /ws/ev     → Voice     │
         │  /api/v1    → REST      │
         │  :50051     → gRPC      │
         │  :9000      → OCPP WS   │
         │                         │
         │  ┌─────────────────┐   │
         │  │ Gemini Live API │   │
         │  │ (Audio Native)  │   │
         │  └─────────────────┘   │
         │                         │
         │  ┌─────┐ ┌─────┐      │
         │  │ PG  │ │Redis│      │
         │  └─────┘ └─────┘      │
         │                         │
         │  ┌─────────────────┐   │
         │  │ OCPP 2.0.1      │   │
         │  │ Charge Stations │   │
         │  └─────────────────┘   │
         └─────────────────────────┘
```

---

## Fases de Desenvolvimento

### FASE 1: Foundation (2 semanas)
> Projeto base + autenticacao + mapa de estacoes

**1.1 Setup do Projeto**
- [ ] `npx create-vite ev-web --template react-ts`
- [ ] Tailwind CSS + PostCSS config
- [ ] PWA manifest + service worker (Workbox)
- [ ] ESLint + Prettier + Husky
- [ ] CI/CD (GitHub Actions -> GHCR -> VM)

**1.2 Autenticacao**
- [ ] Tela de login (CPF + senha / Google OAuth)
- [ ] JWT token management (refresh token flow)
- [ ] Contexto de auth (`useAuth` hook)
- [ ] Rota protegida + redirect

**1.3 Mapa de Estacoes**
- [ ] Componente `<StationMap>` com Leaflet
- [ ] Geolocation API (posicao do usuario)
- [ ] Fetch estacoes proximas: `GET /api/v1/devices/nearby?lat=X&lon=Y&radius=5`
- [ ] Marcadores com status (disponivel/ocupado/fora)
- [ ] Card de estacao (connectors, preco, distancia)

**1.4 REST Client**
- [ ] Axios instance com interceptor JWT
- [ ] TanStack Query hooks: `useNearbyStations`, `useStationDetails`
- [ ] Error boundary + retry logic

**Entregavel F1:** App PWA com login, mapa funcional, lista de estacoes

---

### FASE 2: Voice Engine (2 semanas)
> Integracao da EVA com voz — copiando arquitetura da EVA-Web

**2.1 Audio Engine** (copiar de EVA-Web)
- [ ] `useAudioEngine.ts` — mic capture (16kHz), speaker playback (24kHz)
- [ ] `audio-processor.js` — AudioWorklet para captura em tempo real
- [ ] `audioUtils.ts` — Float32 -> Int16, base64 encode/decode
- [ ] Waveform canvas visualization

**2.2 WebSocket Voice Session**
- [ ] Novo endpoint no EV-IA: `ws://api:8082/ws/ev`
- [ ] Handler Go: `ev_voice_handler.go` (similar ao `browser_voice_handler.go` do EVA-Mind)
- [ ] Protocolo de mensagens:
  ```
  Browser -> Backend:  { type: "audio", data: base64_pcm_16khz }
  Backend -> Browser:  { type: "audio", data: base64_pcm_24khz }
  Backend -> Browser:  { type: "text", text: "subtitle" }
  Backend -> Browser:  { type: "status", text: "charging_started" }
  Backend -> Browser:  { type: "ui_action", action: "show_payment", data: {...} }
  ```
- [ ] Gemini client no backend (reusar `gemini/client.go` do EVA-Mind)
- [ ] System instruction com contexto EV:
  ```
  Voce e EVA, assistente de carregamento de veiculos eletricos.
  Ajude o usuario a: encontrar estacoes, iniciar carregamento,
  monitorar progresso, pagar e receber recibo.
  Fale em portugues brasileiro, seja objetiva e amigavel.
  ```

**2.3 Session UI**
- [ ] `<EvaVoiceButton>` — botao grande "Falar com EVA" (push-to-talk ou hands-free)
- [ ] `<EvaSessionOverlay>` — overlay fullscreen com waveform + subtitles
- [ ] `<EvaChatHistory>` — historico de mensagens da sessao
- [ ] Feedback visual: pulsing mic, waveform, status text

**2.4 Tool Calling (Gemini -> OCPP)**
- [ ] Tools declaration para Gemini:
  ```json
  [
    { "name": "find_nearby_stations", "params": { "lat": "float", "lon": "float" } },
    { "name": "get_station_status", "params": { "station_id": "string" } },
    { "name": "start_charging", "params": { "station_id": "string", "connector_id": "int" } },
    { "name": "stop_charging", "params": { "transaction_id": "string" } },
    { "name": "get_charging_status", "params": { "transaction_id": "string" } },
    { "name": "check_balance", "params": {} },
    { "name": "process_payment", "params": { "amount": "float", "method": "string" } },
    { "name": "send_receipt", "params": { "transaction_id": "string", "email": "string" } }
  ]
  ```
- [ ] Tool executor no backend: mapeia tool calls -> service layer do EV-IA
- [ ] Resultado injetado de volta no Gemini para resposta natural

**Entregavel F2:** Falar com EVA, ela responde por voz, executa comandos OCPP

---

### FASE 3: Charging Flow (2 semanas)
> Fluxo completo de carregamento por voz + UI companion

**3.1 Fluxo de Carregamento (Voice-Driven)**
```
1. DISCOVER  -> "EVA, onde tem estacao perto?"
                -> Tool: find_nearby_stations
                -> UI: mapa com estacoes highlighted

2. SELECT    -> "Quero a estacao 5, conector CCS"
                -> Tool: get_station_status (verifica disponibilidade)
                -> UI: card da estacao com detalhes

3. CONNECT   -> "Ja pluguei o cabo"
                -> OCPP: StatusNotification listener
                -> EVA confirma conexao detectada

4. START     -> "Carrega ate 80%"
                -> Tool: start_charging
                -> OCPP: RequestStartTransaction
                -> UI: monitor de carregamento abre

5. MONITOR   -> "Como ta o carregamento?"
                -> Tool: get_charging_status
                -> EVA: "Esta em 45%, faltam 20 minutos"
                -> UI: barra de progresso + metricas

6. COMPLETE  -> [Auto-detect 80% ou usuario pede pra parar]
                -> Tool: stop_charging
                -> OCPP: RequestStopTransaction

7. PAY       -> "Paga com carteira"
                -> Tool: process_payment
                -> Stripe/Wallet debit

8. RECEIPT   -> Tool: send_receipt (auto)
                -> EVA: "Recibo enviado! Boa viagem!"
```

**3.2 Charging Monitor Component**
- [ ] `<ChargingMonitor>` — tela de acompanhamento
  - Barra circular de progresso (% bateria)
  - kWh consumido (real-time via MeterValues OCPP)
  - Custo atual (preco x kWh)
  - Tempo decorrido / estimativa
  - Potencia atual (kW)
- [ ] WebSocket para MeterValues em tempo real
- [ ] Push notification quando carregamento completar

**3.3 Payment Flow**
- [ ] `<PaymentSheet>` — Stripe Elements ou Wallet balance
- [ ] Carteira digital (saldo pre-pago)
- [ ] Historico de pagamentos
- [ ] Recibo PDF (jsPDF) + envio por email

**3.4 Voice Alerts**
- [ ] EVA avisa por voz quando:
  - Cabo conectado / desconectado
  - Carregamento atingiu meta (80%, 100%)
  - Erro na estacao (OCPP fault)
  - Pagamento confirmado

**Entregavel F3:** Fluxo completo voice-driven de carregamento funcional

---

### FASE 4: Polish & UX (1 semana)
> PWA install, offline, notificacoes, multi-idioma

**4.1 PWA Completa**
- [ ] Install banner ("Adicionar a tela inicial")
- [ ] Offline fallback (cached station data)
- [ ] Background sync (queue commands offline)
- [ ] App icon + splash screen

**4.2 Push Notifications**
- [ ] Firebase Cloud Messaging integration
- [ ] Notificacao: "Carregamento concluido!" (mesmo com app fechado)
- [ ] Notificacao: "Estacao favorita disponivel"

**4.3 UX Mobile**
- [ ] Gesture controls (swipe up = falar, swipe down = fechar)
- [ ] Haptic feedback (vibrar no inicio/fim carregamento)
- [ ] Dark mode (para uso noturno na estacao)
- [ ] Acessibilidade (VoiceOver/TalkBack compativel)

**4.4 Multi-idioma**
- [ ] i18next setup (pt-BR, en-US, es-ES)
- [ ] EVA fala no idioma do usuario
- [ ] `speech_config.language_code` dinamico

**Entregavel F4:** PWA instalavel, offline-ready, notificacoes push

---

### FASE 5: Advanced Features (2 semanas)
> Reservas, smart charging, gamificacao

**5.1 Reservas por Voz**
- [ ] "EVA, reserva a estacao 5 pra amanha as 8h"
- [ ] Tool: `create_reservation`
- [ ] Calendar picker (fallback visual)
- [ ] Lembrete 15min antes

**5.2 Smart Charging por Voz**
- [ ] "Carrega no modo economico" -> horario com tarifa baixa
- [ ] "Carrega rapido, tenho pressa" -> potencia maxima
- [ ] Tool: `set_charging_profile` (OCPP SetChargingProfile)

**5.3 Historico & Analytics**
- [ ] "EVA, quanto gastei esse mes?"
- [ ] Tool: `get_usage_summary`
- [ ] Graficos: Recharts (consumo mensal, gastos, CO2 evitado)

**5.4 Gamificacao**
- [ ] Pontos por carregamento
- [ ] Badges: "Primeiro carregamento", "100 kWh", "Eco Warrior"
- [ ] Ranking mensal de usuarios
- [ ] Desconto por pontos acumulados

**5.5 QR Code**
- [ ] Scan QR na estacao para identificar automaticamente
- [ ] `navigator.mediaDevices` + jsQR
- [ ] EVA: "Detectei estacao 5, conector 2. Quer comecar?"

**Entregavel F5:** Reservas, smart charging, historico, gamificacao

---

## Estrutura do Projeto

```
ev-web/
├── public/
│   ├── audio-processor.js          # AudioWorklet (copiado de EVA-Web)
│   ├── manifest.json               # PWA manifest
│   ├── sw.js                       # Service Worker
│   └── icons/                      # App icons (192, 512)
│
├── src/
│   ├── main.tsx                    # Entry point
│   ├── App.tsx                     # Router + providers
│   │
│   ├── pages/
│   │   ├── LoginPage.tsx           # Auth (CPF/Google)
│   │   ├── HomePage.tsx            # Mapa + estacoes
│   │   ├── StationPage.tsx         # Detalhes da estacao
│   │   ├── ChargingPage.tsx        # Monitor de carregamento
│   │   ├── WalletPage.tsx          # Carteira + pagamentos
│   │   ├── HistoryPage.tsx         # Historico
│   │   └── ProfilePage.tsx         # Perfil + veiculos
│   │
│   ├── components/
│   │   ├── eva/
│   │   │   ├── EvaVoiceButton.tsx  # Botao "Falar com EVA"
│   │   │   ├── EvaSessionOverlay.tsx # Overlay de voz fullscreen
│   │   │   ├── EvaChatBubble.tsx   # Bolha de mensagem
│   │   │   └── EvaWaveform.tsx     # Visualizacao de audio
│   │   │
│   │   ├── charging/
│   │   │   ├── ChargingMonitor.tsx # Progresso circular
│   │   │   ├── ChargingStats.tsx   # kWh, custo, tempo
│   │   │   └── ConnectorCard.tsx   # Tipo conector + status
│   │   │
│   │   ├── station/
│   │   │   ├── StationMap.tsx      # Mapa Leaflet
│   │   │   ├── StationCard.tsx     # Card resumo
│   │   │   └── StationList.tsx     # Lista proximas
│   │   │
│   │   ├── payment/
│   │   │   ├── PaymentSheet.tsx    # Stripe Elements
│   │   │   ├── WalletBalance.tsx   # Saldo
│   │   │   └── ReceiptCard.tsx     # Recibo
│   │   │
│   │   └── ui/
│   │       ├── Button.tsx
│   │       ├── Modal.tsx
│   │       └── LoadingSpinner.tsx
│   │
│   ├── hooks/
│   │   ├── useAudioEngine.ts       # Mic + speaker (de EVA-Web)
│   │   ├── useEvaSession.ts        # WebSocket voice session
│   │   ├── useGeolocation.ts       # GPS do usuario
│   │   ├── useStations.ts          # TanStack Query: estacoes
│   │   ├── useCharging.ts          # TanStack Query: carregamento
│   │   ├── useWallet.ts            # TanStack Query: carteira
│   │   └── useAuth.ts              # JWT auth
│   │
│   ├── services/
│   │   ├── api.ts                  # Axios instance
│   │   ├── websocket.ts            # WebSocket manager
│   │   └── notifications.ts        # Push notifications
│   │
│   ├── utils/
│   │   ├── audioUtils.ts           # PCM encode/decode (de EVA-Web)
│   │   └── formatters.ts           # Moeda, kWh, tempo
│   │
│   ├── types/
│   │   ├── station.ts              # ChargePoint, Connector
│   │   ├── transaction.ts          # Transaction, MeterValues
│   │   ├── voice.ts                # SessionMessage, ToolCall
│   │   └── user.ts                 # User, Vehicle, Wallet
│   │
│   └── styles/
│       └── globals.css             # Tailwind base
│
├── index.html
├── vite.config.ts
├── tailwind.config.ts
├── tsconfig.json
├── package.json
└── Dockerfile
```

---

## Backend: Novo Endpoint WebSocket

Adicionar ao EV-IA (`internal/adapter/websocket/ev_voice_handler.go`):

```go
// Novo handler WebSocket para voz do EV-Web
// Rota: ws://api:8082/ws/ev

func (h *EVVoiceHandler) HandleConnection(conn *websocket.Conn, userID string) {
    // 1. Conectar ao Gemini Live API
    // 2. Injetar system instruction com contexto EV
    // 3. Registrar tools (find_stations, start_charging, etc.)
    // 4. Loop: browser audio -> Gemini -> response audio -> browser
    // 5. Tool calls -> execute via service layer -> inject result
}
```

**Tools a registrar no Gemini:**

| Tool | Descricao | Service Layer |
|------|-----------|---------------|
| `find_nearby_stations` | Busca estacoes proximas | `DeviceService.FindNearby()` |
| `get_station_status` | Status da estacao | `DeviceService.GetStatus()` |
| `start_charging` | Inicia carregamento | `TransactionService.StartTransaction()` |
| `stop_charging` | Para carregamento | `TransactionService.StopTransaction()` |
| `get_charging_status` | Status em tempo real | `TransactionService.GetActiveByUser()` |
| `check_balance` | Saldo da carteira | `WalletService.GetBalance()` |
| `process_payment` | Processa pagamento | `PaymentService.ProcessPayment()` |
| `send_receipt` | Envia recibo | `NotificationService.SendEmail()` |
| `create_reservation` | Cria reserva | `ReservationService.Create()` |
| `get_history` | Historico do usuario | `TransactionService.GetHistory()` |

---

## Timeline Resumida

| Fase | Duracao | Entregavel |
|------|---------|-----------|
| **F1: Foundation** | 2 sem | PWA com login + mapa de estacoes |
| **F2: Voice Engine** | 2 sem | EVA responde por voz + executa comandos |
| **F3: Charging Flow** | 2 sem | Carregamento completo voice-driven |
| **F4: Polish** | 1 sem | PWA install, push notifications, dark mode |
| **F5: Advanced** | 2 sem | Reservas, smart charging, gamificacao |

---

## Diagrama de Fluxo do Usuario

```
┌─────────────┐     ┌──────────────┐     ┌──────────────┐
│  Abrir App  │────>│  Login/Auth   │────>│  Mapa Home   │
│  (PWA)      │     │  (CPF/Google) │     │  (estacoes)  │
└─────────────┘     └──────────────┘     └──────┬───────┘
                                                  │
                              ┌────────────────────┤
                              │                    │
                    ┌─────────▼──────┐   ┌────────▼────────┐
                    │  Tocar "EVA"   │   │  Selecionar     │
                    │  (voz ativa)   │   │  Estacao (toque) │
                    └─────────┬──────┘   └────────┬────────┘
                              │                    │
                              ▼                    ▼
                    ┌──────────────────────────────────────┐
                    │         EVA Voice Session             │
                    │                                        │
                    │  "Quero carregar" ──> find_station    │
                    │  "CCS"            ──> select_connector│
                    │  "Ate 80%"        ──> start_charging  │
                    │  "Como ta?"       ──> get_status      │
                    │  "Para"           ──> stop_charging   │
                    │  "Paga carteira"  ──> process_payment │
                    └──────────────────┬───────────────────┘
                                       │
                              ┌────────▼────────┐
                              │  Charging        │
                              │  Monitor         │
                              │  (real-time)     │
                              │  ⚡ 45% ████░░  │
                              │  12.5 kWh        │
                              │  R$ 9,37         │
                              └────────┬────────┘
                                       │
                              ┌────────▼────────┐
                              │  Pagamento +     │
                              │  Recibo          │
                              │  "Boa viagem!"   │
                              └─────────────────┘
```

---

## Requisitos de Infra

| Recurso | Detalhe |
|---------|---------|
| **Gemini API Key** | Mesma do EVA-Mind (billing ativo) |
| **Stripe Account** | Para pagamentos reais |
| **Firebase** | Push notifications |
| **VM malaria** | Backend ja rodando (porta 8082) |
| **Dominio** | `ev.eva-ia.org` (HTTPS obrigatorio para mic) |
| **SSL** | Let's Encrypt (mic exige HTTPS) |
