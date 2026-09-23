DATABASE_URL ?= postgres://postgres:postgres@localhost:5432/changes

.PHONY: init run db

init:
	@go install golang.org/x/lint/golint@latest
	@go install honnef.co/go/tools/cmd/staticcheck@latest;

run:
	@kill $$(lsof -ti tcp:8080) >/dev/null 2>&1 || true
	@kill $$(lsof -ti tcp:8000) >/dev/null 2>&1 || true
	@docker-compose up -d
	@cd backend && go build -o aipm-server cmd/server/*.go
	@(cd backend && ./aipm-server) & \
		backend_pid=$$!; \
		(cd frontend && pnpm dev) & \
		frontend_pid=$$!; \
		trap 'kill $$backend_pid $$frontend_pid >/dev/null 2>&1 || true' INT TERM EXIT; \
		echo "Backend:  http://localhost:8080"; \
		echo "Frontend: http://localhost:8000"; \
		wait $$backend_pid $$frontend_pid

db:
	@psql "$(DATABASE_URL)" -v ON_ERROR_STOP=1 -f db/init.sql
	@psql "$(DATABASE_URL)" -v ON_ERROR_STOP=1 -f db/seed.sql
	@psql "$(DATABASE_URL)" -v ON_ERROR_STOP=1 -f db/seed-demo.sql
