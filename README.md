# Tengine Proxy Manager [![Beta](https://img.shields.io/badge/status-beta-orange)]()

Web-based management panel for [Tengine](https://github.com/alibaba/tengine), a high-performance web server forked from Nginx by Alibaba. Tengine adds dynamic module loading, advanced load balancing, and active health checks while maintaining full Nginx compatibility. Built with Go, React, and PostgreSQL.

> **Beta Notice:** This project is in active development. Features may change and bugs may exist. Feedback and contributions are welcome.

## Features

- **Proxy Host Management** — Create and manage reverse proxy entries with domain, forwarding rules, SSL, health checks, and load balancing
- **SSL Certificates** — Let's Encrypt automation, self-signed certs for local dev, custom certificate upload
- **Access Control** — IP/CIDR-based allow/deny lists with expression editor
- **Two-Factor Auth** — TOTP-based 2FA for admin accounts
- **Audit Logging** — Full audit trail with IP tracking, filters, and retention policy
- **Configurable Default Server** — Customize Tengine's fallback response on unmatched domains
- **Automatic Config Generation** — Tengine configs generated from templates and auto-reloaded

## Tech Stack

| Layer    | Technology                      |
|----------|---------------------------------|
| Frontend | React 18, TypeScript, Tailwind, Zustand |
| Backend  | Go, Echo v4, GORM              |
| Database | PostgreSQL 16                   |
| Proxy    | Tengine 3.1                     |

## Quick Start

```bash
# Clone
git clone <repo-url>
cd TengineProxyManager

# Create .env from example
cp .env.example .env

# Start all services
docker compose up -d

# Development (with hot reload)
docker compose -f docker-compose.dev.yml up -d
```

### Access

| Service    | URL                          |
|------------|------------------------------|
| Frontend   | http://localhost:5173        |
| Backend    | http://localhost:4000        |
| Tengine    | http://localhost:80          |

On first run, create an admin account via the setup page.

## Project Structure

```
├── frontend/          # React + Vite + Tailwind
│   └── src/
│       ├── pages/     # Page components
│       ├── components/# Reusable UI components
│       ├── store/     # Zustand state stores
│       └── api/       # Axios API client
├── backend/           # Go + Echo + GORM
│   ├── cmd/server/    # Entry point
│   └── internal/
│       ├── handler/   # HTTP handlers
│       ├── service/   # Business logic (Tengine, Certbot, Audit)
│       ├── model/     # GORM models
│       ├── middleware/ # Auth, rate limiting, security
│       └── database/  # Connection + migrations
├── docker/            # Docker configs
│   └── tengine/       # Tengine Dockerfile + nginx.conf
└── scripts/           # Utility scripts
```

## Environment Variables

| Variable              | Description                     | Default  |
|-----------------------|---------------------------------|----------|
| `DB_HOST`             | PostgreSQL host                 | postgres |
| `DB_PORT`             | PostgreSQL port                 | 5432     |
| `DB_NAME`             | Database name                   | tpm      |
| `DB_USER`             | Database user                   | tpm      |
| `DB_PASSWORD`         | Database password               | —        |
| `JWT_SECRET`          | JWT signing secret              | —        |
| `AUDIT_RETENTION_DAYS`| Auto-cleanup age (0=disabled)  | 30       |
| `TENGINE_CONF_DIR`    | Tengine config directory        | /etc/tengine/conf.d |
| `CORS_ORIGINS`        | Extra CORS origins (comma-separated) | —   |

## Contributing

Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.

1. Fork the repo
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m "Add amazing feature"`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

[MIT](LICENSE)
