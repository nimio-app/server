# Nimio Server

[![License: Apache 2.0](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.22%2B-00ADD8?logo=go)](https://go.dev/)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?logo=docker)](https://www.docker.com/)

> **Respect starts before the first message.**

This repository contains the backend REST API and real-time sync service for **Nimio**—an open-source platform for sharing intentional availability with trusted connections.

---

## 🛠️ Tech Stack

- **Language:** Go (Golang 1.22+)
- **HTTP Router:** [Chi (`go-chi/chi/v5`)](https://github.com/go-chi/chi)
- **Database:** PostgreSQL
- **Database Driver:** `pgx/v5`
- **Authentication:** JWT (Access + Refresh Tokens) + Argon2 Hashing
- **Architecture:** Clean Architecture (Handlers $\rightarrow$ Services $\rightarrow$ Repositories)

---

## 🏗️ Project Architecture

```text
server/
├── cmd/
│   └── api/                  # Application entry point (main.go)
├── internal/
│   ├── config/               # Environment & database configuration
│   ├── domain/               # Core data models (User, Status, Connection)
│   ├── handler/              # HTTP API Handlers / Controllers
│   ├── middleware/           # Auth JWT & CORS middleware
│   ├── repository/           # PostgreSQL data access layer
│   └── service/              # Core business & visibility rules engine
├── migrations/               # Database SQL schema migration files
├── docker-compose.yml        # Local development environment setup
├── Dockerfile                # Lightweight multi-stage build container
├── go.mod
└── README.md
