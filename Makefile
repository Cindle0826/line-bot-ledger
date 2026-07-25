-include .env
export

TUNNEL_NAME := line-bot-ledger-webhook

.PHONY: tunnel run-webhook run-summary run-liff fmt vet test build check \
	docker-build-webhook-local docker-build-summary-local docker-build-liff-local \
	docker-run-webhook-local docker-run-summary-local docker-run-liff-local

tunnel:
	cloudflared tunnel run $(TUNNEL_NAME)

run-webhook:
	go run ./services/webhook

# Different port from webhook (8080, via .env) so both can run locally at
# the same time.
run-summary:
	PORT=8082 go run ./services/summary

run-liff:
	PORT=8081 go run ./services/liff

fmt:
	gofmt -l -w .

vet:
	go vet ./...

test:
	go test ./...

build:
	go build ./...

check: fmt vet test

# Local sanity checks only — native arch (arm64 on Apple Silicon), no push,
# no deploy. Production build+push+deploy is .github/workflows/webhook.yml
# and summary.yml, which build linux/amd64 via deploy/<service>/Dockerfile.
docker-build-webhook-local:
	docker build -f deploy/webhook/Dockerfile.local -t line-bot-ledger-webhook:local .

docker-build-summary-local:
	docker build -f deploy/summary/Dockerfile.local -t line-bot-ledger-summary:local .

docker-build-liff-local:
	docker build -f deploy/liff/Dockerfile.local -t line-bot-ledger-liff:local .

docker-run-webhook-local: docker-build-webhook-local
	docker run --rm --env-file .env -p 8080:8080 line-bot-ledger-webhook:local

docker-run-summary-local: docker-build-summary-local
	docker run --rm --env-file .env -p 8082:8080 line-bot-ledger-summary:local

docker-run-liff-local: docker-build-liff-local
	docker run --rm --env-file .env -p 8081:8080 line-bot-ledger-liff:local
