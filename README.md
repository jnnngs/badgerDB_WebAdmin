
# 🛠️ BadgerAdmin

A lightweight web-based admin panel for inspecting and managing your BadgerDB data.

## 🚀 Features

- View, edit, delete keys
- Backup & restore database
- Web UI with auth
- Docker support
- Easily embeddable or standalone

## 🐳 Run with Docker

```bash
docker-compose up --build
```

Visit: [http://localhost:8080](http://localhost:8080)  
Login: `admin / botfluence` (configurable via env)

## 🔐 Auth Environment Variables

- `BADGER_ADMIN_USER=admin`
- `BADGER_ADMIN_PASS=botfluence`

## 📦 Backup & Restore

- 🔽 `/backup` – Download full snapshot
- 🔼 `/restore` – Upload backup `.bak` file

## 📁 Data Volume

Your BadgerDB lives in `./data` and is mounted into the container for persistence.
