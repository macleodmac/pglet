VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BINARY := pglet

.PHONY: dev dev-frontend dev-backend build clean generate generate-go generate-ts lint

dev-frontend:
	cd frontend && pnpm run dev

dev-backend:
	PGLET_DEV=1 go run main.go --dev --cors

dev:
	@make -j2 dev-frontend dev-backend

build: build-frontend build-backend

build-frontend:
	cd frontend && pnpm run build

build-backend: build-frontend
	go build -ldflags "-s -w -X main.version=$(VERSION)" -o $(BINARY) .

clean:
	rm -rf frontend/dist $(BINARY)

generate: generate-go generate-ts

generate-go:
	oapi-codegen --config oapi-codegen.yaml openapi.yaml

generate-ts:
	cd frontend && npx @hey-api/openapi-ts -i ../openapi.yaml -o src/api/generated

lint:
	revive -config revive.toml -exclude pkg/api/openapi.gen.go ./...

deps:
	cd frontend && pnpm install
	go mod tidy
