.PHONY: dev dev-build dev-down dev-logs dev-restart prod prod-build prod-down prod-logs prod-restart clean-dev clean-prod

# Development
dev:
	docker compose -p torrent-dev -f docker-compose.dev.yml up

dev-build:
	docker compose -p torrent-dev -f docker-compose.dev.yml up --build

dev-build-deploy:
	docker compose -p torrent-dev -f docker-compose.dev.yml up -d --build

dev-down:
	docker compose -p torrent-dev -f docker-compose.dev.yml down

dev-logs:
	docker compose -p torrent-dev -f docker-compose.dev.yml logs -f

dev-restart:
	docker compose -p torrent-dev -f docker-compose.dev.yml restart

# Production
prod-deploy:
	docker compose -p torrent-prod -f docker-compose.yml up -d

prod-build-deploy:
	docker compose -p torrent-prod -f docker-compose.yml up -d --build

prod-down:
	docker compose -p torrent-prod -f docker-compose.yml down

prod-logs:
	docker compose -p torrent-prod -f docker-compose.yml logs -f

prod-restart:
	docker compose -p torrent-prod -f docker-compose.yml restart

# Cleanup
clean-dev:
	docker compose -p torrent-dev -f docker-compose.dev.yml down -v --rmi all --remove-orphans
	-docker rmi torrent-project-frontend-dev torrent-project-backend-dev 2>/dev/null

clean-prod:
	docker compose -p torrent-prod -f docker-compose.yml down -v --rmi all --remove-orphans
	-docker rmi torrent-project-frontend torrent-project-backend 2>/dev/null
