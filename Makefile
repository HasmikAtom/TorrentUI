.PHONY: dev dev-build dev-down dev-logs dev-restart prod prod-build prod-down prod-logs prod-restart clean-dev clean-prod

# Development
dev:
	docker-compose -f docker-compose.dev.yml up

dev-build:
	docker compose -f docker-compose.dev.yml up --build

dev-down:
	docker-compose -f docker-compose.dev.yml down

dev-logs:
	docker-compose -f docker-compose.dev.yml logs -f

dev-restart:
	docker-compose -f docker-compose.dev.yml restart

# Production
prod:
	docker-compose -f docker-compose.yml up -d

prod-build:
	docker-compose -f docker-compose.yml up -d --build

prod-down:
	docker-compose -f docker-compose.yml down

prod-logs:
	docker-compose -f docker-compose.yml logs -f

prod-restart:
	docker-compose -f docker-compose.yml restart

# Cleanup
clean-dev:
	docker-compose -f docker-compose.dev.yml down -v --rmi all --remove-orphans
	-docker rmi torrent-project-frontend-dev torrent-project-backend-dev 2>/dev/null

clean-prod:
	docker-compose -f docker-compose.yml down -v --rmi all --remove-orphans
	-docker rmi torrent-project-frontend torrent-project-backend 2>/dev/null
