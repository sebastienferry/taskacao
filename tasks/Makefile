.PHONY: dev build run test clean

dev:
	@echo "Starting Go backend & Vite frontend in development mode..."
	@(go run ./cmd/server & cd web && npm run dev)

dev-server:
	go run ./cmd/server

dev-web:
	cd web && npm run dev

build:
	@echo "Building React frontend..."
	cd web && npm run build
	@echo "Building Go server binary..."
	go build -o bin/server ./cmd/server
	@echo "Build complete! Binary located at bin/server"

run: build
	./bin/server

clean:
	rm -rf bin/ tasks.db tasks.db-wal tasks.db-shm web/dist
