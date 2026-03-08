# ============================================
# SIGEC-VE Enterprise - Makefile
# ============================================
# Automação de tarefas de desenvolvimento e deployment

.PHONY: help
.DEFAULT_GOAL := help

# ============================================
# Variables
# ============================================
APP_NAME := sigec-ve
VERSION := $(shell git describe --tags --always --dirty)
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT := $(shell git rev-parse --short HEAD)
REGISTRY := gcr.io
PROJECT_ID := your-gcp-project
IMAGE := $(REGISTRY)/$(PROJECT_ID)/$(APP_NAME)

GO := go
GOFLAGS := -v
LDFLAGS := -ldflags "-w -s -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)"

# ============================================
# Help
# ============================================
help: ## Mostra este help
	@echo "SIGEC-VE Enterprise - Makefile Commands"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

# ============================================
# Development
# ============================================
install: ## Instala dependências
	@echo "📦 Instalando dependências..."
	$(GO) mod download
	$(GO) mod verify
	@echo "✅ Dependências instaladas"

install-tools: ## Instala ferramentas de desenvolvimento
	@echo "🔧 Instalando ferramentas..."
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	$(GO) install github.com/swaggo/swag/cmd/swag@latest
	$(GO) install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	$(GO) install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	$(GO) install github.com/vektra/mockery/v2@latest
	@echo "✅ Ferramentas instaladas"

run: ## Roda o servidor localmente
	@echo "🚀 Iniciando servidor..."
	$(GO) run $(GOFLAGS) ./cmd/server/main.go

run-worker: ## Roda o worker localmente
	@echo "🚀 Iniciando worker..."
	$(GO) run $(GOFLAGS) ./cmd/worker/main.go

dev: ## Inicia ambiente de desenvolvimento completo
	@echo "🚀 Iniciando ambiente de desenvolvimento..."
	docker-compose up -d
	@echo "✅ Ambiente iniciado!"
	@echo "📊 Grafana: http://localhost:3000 (admin/admin)"
	@echo "📈 Prometheus: http://localhost:9090"
	@echo "🔍 Jaeger: http://localhost:16686"
	@echo "🗄️  PgAdmin: http://localhost:5050"

dev-down: ## Para ambiente de desenvolvimento
	@echo "⏹️  Parando ambiente..."
	docker-compose down
	@echo "✅ Ambiente parado"

dev-logs: ## Mostra logs do ambiente de desenvolvimento
	docker-compose logs -f api

dev-reset: ## Reset completo (deleta volumes)
	@echo "⚠️  ATENÇÃO: Isso vai deletar todos os dados!"
	@read -p "Tem certeza? [y/N] " -n 1 -r; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		docker-compose down -v; \
		echo "✅ Reset concluído"; \
	fi

# ============================================
# Building
# ============================================
build: ## Build do binário
	@echo "🔨 Building $(APP_NAME)..."
	CGO_ENABLED=0 $(GO) build $(GOFLAGS) $(LDFLAGS) -o bin/$(APP_NAME) ./cmd/server/main.go
	@echo "✅ Build concluído: bin/$(APP_NAME)"

build-worker: ## Build do worker
	@echo "🔨 Building worker..."
	CGO_ENABLED=0 $(GO) build $(GOFLAGS) $(LDFLAGS) -o bin/$(APP_NAME)-worker ./cmd/worker/main.go
	@echo "✅ Build concluído: bin/$(APP_NAME)-worker"

build-all: build build-worker ## Build de todos os binários

docker-build: ## Build da imagem Docker
	@echo "🐳 Building Docker image..."
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		-t $(IMAGE):$(VERSION) \
		-t $(IMAGE):latest \
		-f deployments/docker/Dockerfile \
		.
	@echo "✅ Docker image criada: $(IMAGE):$(VERSION)"

docker-push: ## Push da imagem para registry
	@echo "📤 Pushing Docker image..."
	docker push $(IMAGE):$(VERSION)
	docker push $(IMAGE):latest
	@echo "✅ Push concluído"

docker-run: ## Roda a imagem Docker localmente
	@echo "🐳 Running Docker container..."
	docker run -p 8080:8080 -p 9000:9000 \
		-e DATABASE_URL=postgres://admin:password@host.docker.internal:5432/sigec \
		-e NietzscheDB_URL=NietzscheDB://host.docker.internal:6379/0 \
		$(IMAGE):latest

# ============================================
# Code Generation
# ============================================
proto-gen: ## Gera código Go dos Protocol Buffers
	@echo "⚙️  Gerando código dos .proto files..."
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		api/proto/**/*.proto
	@echo "✅ Código gerado"

swagger-gen: ## Gera documentação Swagger
	@echo "📝 Gerando Swagger docs..."
	swag init -g cmd/server/main.go -o api/swagger
	@echo "✅ Swagger docs gerados"

mocks-gen: ## Gera mocks para testes
	@echo "🎭 Gerando mocks..."
	mockery --all --keeptree --output=./mocks
	@echo "✅ Mocks gerados"

generate: proto-gen swagger-gen mocks-gen ## Gera todo o código

# ============================================
# Testing
# ============================================
test: ## Roda testes unitários
	@echo "🧪 Running tests..."
	$(GO) test -v -race -coverprofile=coverage.out ./...
	@echo "✅ Testes concluídos"

test-coverage: test ## Mostra coverage dos testes
	@echo "📊 Test coverage:"
	$(GO) tool cover -func=coverage.out

test-coverage-html: test ## Gera relatório HTML de coverage
	@echo "📊 Gerando relatório HTML..."
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "✅ Abra coverage.html no navegador"

test-integration: ## Roda testes de integração
	@echo "🧪 Running integration tests..."
	$(GO) test -v -tags=integration ./tests/integration/...
	@echo "✅ Testes de integração concluídos"

test-e2e: ## Roda testes end-to-end
	@echo "🧪 Running e2e tests..."
	$(GO) test -v -tags=e2e ./tests/e2e/...
	@echo "✅ Testes e2e concluídos"

test-load: ## Roda testes de carga com k6
	@echo "🔥 Running load tests..."
	k6 run tests/load/k6/load_test.js
	@echo "✅ Load tests concluídos"

bench: ## Roda benchmarks
	@echo "⚡ Running benchmarks..."
	$(GO) test -bench=. -benchmem ./...
	@echo "✅ Benchmarks concluídos"

# ============================================
# Code Quality
# ============================================
lint: ## Roda linter
	@echo "🔍 Running linter..."
	golangci-lint run --timeout=5m
	@echo "✅ Linting concluído"

lint-fix: ## Roda linter e corrige automaticamente
	@echo "🔧 Running linter with auto-fix..."
	golangci-lint run --fix --timeout=5m
	@echo "✅ Auto-fix concluído"

fmt: ## Formata código
	@echo "✨ Formatting code..."
	$(GO) fmt ./...
	@echo "✅ Código formatado"

vet: ## Roda go vet
	@echo "🔍 Running go vet..."
	$(GO) vet ./...
	@echo "✅ Vet concluído"

security-scan: ## Roda scan de segurança
	@echo "🔒 Running security scan..."
	gosec -fmt=json -out=gosec-report.json ./...
	@echo "✅ Security scan concluído"

code-quality: fmt lint vet security-scan ## Roda todas as verificações de qualidade

# ============================================
# Database
# ============================================
db-create: ## Cria banco de dados
	@echo "🗄️  Criando banco de dados..."
	docker-compose exec postgres psql -U admin -c "CREATE DATABASE sigec_dev;"
	@echo "✅ Banco criado"

db-drop: ## Deleta banco de dados
	@echo "⚠️  Deletando banco de dados..."
	docker-compose exec postgres psql -U admin -c "DROP DATABASE IF EXISTS sigec_dev;"
	@echo "✅ Banco deletado"

db-migrate: ## Roda migrations
	@echo "📊 Running migrations..."
	$(GO) run ./cmd/migrator/main.go up
	@echo "✅ Migrations executadas"

db-migrate-down: ## Reverte última migration
	@echo "⏪ Reverting migration..."
	$(GO) run ./cmd/migrator/main.go down
	@echo "✅ Migration revertida"

db-seed: ## Popula banco com dados de exemplo
	@echo "🌱 Seeding database..."
	$(GO) run ./scripts/seed/main.go
	@echo "✅ Seeding concluído"

db-console: ## Abre console do NietzscheDB
	docker-compose exec postgres psql -U admin -d sigec_dev

# ============================================
# Kubernetes
# ============================================
k8s-deploy: ## Deploy no Kubernetes
	@echo "☸️  Deploying to Kubernetes..."
	kubectl apply -f deployments/kubernetes/base/
	@echo "✅ Deploy concluído"

k8s-status: ## Mostra status do deployment
	@echo "📊 Kubernetes status:"
	kubectl get pods -n sigec-ve-prod
	kubectl get svc -n sigec-ve-prod
	kubectl get ingress -n sigec-ve-prod

k8s-logs: ## Mostra logs do pod
	kubectl logs -f deployment/sigec-ve-api -n sigec-ve-prod

k8s-shell: ## Abre shell no container
	kubectl exec -it deployment/sigec-ve-api -n sigec-ve-prod -- /bin/sh

k8s-scale: ## Escala deployment (uso: make k8s-scale REPLICAS=5)
	kubectl scale deployment/sigec-ve-api --replicas=$(REPLICAS) -n sigec-ve-prod

k8s-rollback: ## Faz rollback do deployment
	kubectl rollout undo deployment/sigec-ve-api -n sigec-ve-prod

k8s-restart: ## Restart do deployment
	kubectl rollout restart deployment/sigec-ve-api -n sigec-ve-prod

# ============================================
# Monitoring
# ============================================
metrics: ## Abre Prometheus
	@open http://localhost:9090 || xdg-open http://localhost:9090

dashboard: ## Abre Grafana
	@open http://localhost:3000 || xdg-open http://localhost:3000

traces: ## Abre Jaeger
	@open http://localhost:16686 || xdg-open http://localhost:16686

# ============================================
# Cleanup
# ============================================
clean: ## Remove binários e arquivos temporários
	@echo "🧹 Cleaning up..."
	rm -rf bin/
	rm -rf coverage.*
	rm -rf *.log
	rm -rf tmp/
	@echo "✅ Cleanup concluído"

clean-docker: ## Remove imagens Docker locais
	@echo "🧹 Removing Docker images..."
	docker rmi $(IMAGE):latest $(IMAGE):$(VERSION) 2>/dev/null || true
	@echo "✅ Docker images removidas"

clean-all: clean clean-docker ## Remove tudo

# ============================================
# Git
# ============================================
tag: ## Cria nova tag de versão (uso: make tag VERSION=v1.0.0)
	@if [ -z "$(VERSION)" ]; then \
		echo "❌ Erro: VERSION não especificada"; \
		echo "Uso: make tag VERSION=v1.0.0"; \
		exit 1; \
	fi
	@echo "🏷️  Criando tag $(VERSION)..."
	git tag -a $(VERSION) -m "Release $(VERSION)"
	git push origin $(VERSION)
	@echo "✅ Tag criada e enviada"

# ============================================
# Release
# ============================================
release: code-quality test docker-build docker-push ## Build e push completo para release
	@echo "🎉 Release $(VERSION) concluído!"
	@echo "📦 Docker image: $(IMAGE):$(VERSION)"

# ============================================
# Info
# ============================================
info: ## Mostra informações do projeto
	@echo "📋 Project Information"
	@echo "  App Name:    $(APP_NAME)"
	@echo "  Version:     $(VERSION)"
	@echo "  Git Commit:  $(GIT_COMMIT)"
	@echo "  Build Time:  $(BUILD_TIME)"
	@echo "  Go Version:  $(shell go version)"
	@echo "  Docker:      $(IMAGE):$(VERSION)"
