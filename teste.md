# SIGEC-VE Enterprise - Endpoints

## Servidor: eva-ia.org (104.248.219.200)

| Servico          | Endereco                             |
|------------------|--------------------------------------|
| REST API         | http://eva-ia.org:8080/api/v1/       |
| Health Check     | http://eva-ia.org:8080/health/live   |
| OCPP WebSocket   | ws://eva-ia.org:9000/ocpp            |
| gRPC             | eva-ia.org:50051                     |

---

## Credenciais Admin

- **Email**: admin@sigec-ve.com
- **Senha**: admin123

---

## Comandos Uteis

### Iniciar servidor
```bash
cd ~/EV-IA && nohup ./sigec-ve > sigec-ve.log 2>&1 &
```

### Parar servidor
```bash
pkill sigec-ve
```

### Ver logs
```bash
tail -f ~/EV-IA/sigec-ve.log
```

### Testar login
```bash
curl -X POST http://eva-ia.org:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@sigec-ve.com","password":"admin123"}'
```

### Rodar simulador OCPP V2G
```bash
cd ~/EV-IA && go run ./cmd/simulator/main.go --id=CP001 --server=ws://localhost:9000/ocpp --v2g --interactive
```

---

## Portas

- **8080** - HTTP REST API
- **9000** - OCPP 2.0.1 WebSocket
- **50051** - gRPC

---

*Atualizado: Fevereiro 2026*
