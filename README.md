# pree-it

A real time chat application

## Stack Overview

### Frontend (Web)

- Framework: SvelteKit
- Language: TypeScript
- State Management: Svelte runes
- Styling: TailwindCSS
- Realtime Transport: WebSocket
- Authentication: JWT + refresh tokens
- Deployment Target: Vercel / Cloudflare / Docker

### Mobile Frontend

- Framework: Jetpack Compose
- Language: Kotlin
- Architecture: MVVM + Clean Architecture
- Dependency Injection: Hilt
- Networking: Ktor or Retrofit
- Local Storage: Room
- Realtime Transport: WebSocket
- Push Notifications: Pusher Beam
- Media Handling: Coil
- Navigation: Navigation Compose

### Backend

- Language: Go
- Framework: Gin
- Realtime Layer: Gorilla WebSocket or nhooyr/websocket
- Authentication: JWT
- Database: PostgreSQL
- Cache / PubSub: Redis
- Message Queue (optional later): NATS or Kafka
- Object Storage (optional): S3-compatible storage

### Infrastructure

- Docker + Docker Compose
- Caddy reverse proxy
- CI/CD: GitHub Actions
- Observability:
  - OpenTelemetry
  - Prometheus
  - Grafana
  - Loki

## Core Features

### MVP Features

#### Authentication

- Register
- Login
- Logout
- Refresh tokens
- Email verification
- Password reset

#### User Features

- User profiles
- Online/offline presence
- Last seen
- Typing indicators
- User search

#### Chat Features

- Direct messaging
- Group chats
- Realtime message delivery
- Read receipts
- Message reactions
- Message editing
- Message deletion
- Infinite scroll history

#### Media Features

- Image uploads
- File uploads
- Voice notes
- Link previews

#### Realtime Features

- Presence tracking
- Typing indicators
- Delivery acknowledgements
- Reconnect handling
- Heartbeats/ping-pong
