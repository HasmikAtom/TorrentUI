# Torrent Media Manager

A self-hosted web application for managing torrent downloads, designed to work seamlessly with [Plex](https://www.plex.tv/) and [Transmission](https://transmissionbt.com/). Search for torrents, download them to organized media folders, and let Plex automatically pick them up.

## Features

- **Magnet Link Downloads** - Paste magnet links directly or upload .torrent files
- **Torrent Search** - Search ThePirateBay and RuTracker directly from the UI
- **Batch Downloads** - Select multiple search results and download them at once
- **Media Organization** - Automatically sorts downloads into Movies, Series, or Music folders
- **Download Monitoring** - Track download progress in real-time
- **Plex Integration** - Downloads go directly to Plex-monitored directories

## Architecture

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│    Frontend     │────▶│     Backend     │────▶│  Transmission   │
│   React + Vite  │     │    Go + Gin     │     │   BitTorrent    │
│    Port 3000    │     │    Port 8080    │     │    Port 9091    │
└─────────────────┘     └─────────────────┘     └─────────────────┘
                                                        │
                                                        ▼
                                                ┌─────────────────┐
                                                │  /mediastorage  │
                                                │  ├── Movies     │
                                                │  ├── Series     │
                                                │  └── Music      │
                                                └─────────────────┘
                                                        │
                                                        ▼
                                                ┌─────────────────┐
                                                │      Plex       │
                                                └─────────────────┘
```

## Tech Stack

**Frontend:**
- React 18 with TypeScript
- Vite for bundling
- Tailwind CSS for styling
- Radix UI components
- Lucide icons

**Backend:**
- Go 1.23
- Gin web framework
- Chromedp for web scraping (headless Chrome)
- Transmission RPC client

## Prerequisites

- Docker and Docker Compose
- Transmission daemon running and accessible
- (Optional) Plex Media Server with libraries pointing to `/mediastorage/*`

## Quick Start

1. **Clone the repository**
   ```bash
   git clone <your-repo-url>
   cd torrent-project
   ```

2. **Configure environment**
   ```bash
   cp backend/.env.example backend/.env
   # Edit backend/.env with your Transmission credentials
   ```

3. **Start the application**
   ```bash
   # Production (detached)
   make prod-build-deploy

   # Development (with hot reload)
   make dev-build
   ```

4. **Access the UI**
   - Frontend: http://localhost:3000
   - Backend API: http://localhost:8080

## Configuration

Create `backend/.env` with the following variables:

```env
# Development
DEV_APP_PORT=8085
DEV_TRANSMISSION_HOST=localhost
DEV_TRANSMISSION_PORT=9091
DEV_TRANSMISSION_USERNAME=your_username
DEV_TRANSMISSION_PASSWORD=your_password
DEV_RUTRACKER_USERNAME=your_rutracker_user
DEV_RUTRACKER_PASSWORD=your_rutracker_pass

# Production (Docker)
PROD_APP_PORT=8080
PROD_TRANSMISSION_HOST=host.docker.internal
PROD_TRANSMISSION_PORT=9091
PROD_TRANSMISSION_USERNAME=your_username
PROD_TRANSMISSION_PASSWORD=your_password
PROD_RUTRACKER_USERNAME=your_rutracker_user
PROD_RUTRACKER_PASSWORD=your_rutracker_pass
```

## Makefile Commands

| Command | Description |
|---------|-------------|
| `make dev` | Start development containers |
| `make dev-build` | Build and start development containers |
| `make dev-down` | Stop development containers |
| `make dev-logs` | View development logs |
| `make prod-deploy` | Start production containers |
| `make prod-build-deploy` | Build and start production containers |
| `make prod-down` | Stop production containers |
| `make prod-logs` | View production logs |
| `make clean-dev` | Remove dev containers and images |
| `make clean-prod` | Remove prod containers and images |

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/health` | Health check |
| `POST` | `/download` | Add magnet link or torrent file |
| `POST` | `/download/batch` | Add multiple magnet links |
| `POST` | `/download/file` | Download torrent from URL (RuTracker) |
| `POST` | `/download/file/batch` | Download multiple torrents from URLs |
| `GET` | `/status/:id` | Get torrent download status |
| `GET` | `/torrents` | List all torrents |
| `POST` | `/scrape/piratebay/:name` | Search ThePirateBay |
| `POST` | `/scrape/rutracker/:name` | Search RuTracker |

## Media Folder Structure

Downloads are organized into:

```
/mediastorage/
├── Movies/      # contentType: "Movies"
├── Series/      # contentType: "Series"
└── Music/       # contentType: "Music"
```

Configure your Plex libraries to monitor these directories.

## Transmission Setup

Ensure Transmission is configured to:
1. Allow RPC connections from the backend container
2. Have write access to `/mediastorage/` directories
3. Have RPC authentication enabled (recommended)

Example Transmission settings (`settings.json`):
```json
{
  "rpc-enabled": true,
  "rpc-port": 9091,
  "rpc-authentication-required": true,
  "rpc-username": "your_username",
  "rpc-password": "your_password",
  "rpc-whitelist-enabled": false
}
```

## Security Notes

- This application is designed for personal/home server use
- RuTracker credentials are stored in environment variables
- CORS is configured permissively - restrict in production if exposed externally
- Do not expose this application to the public internet without additional security measures

## License

Personal use project. Use at your own risk.
