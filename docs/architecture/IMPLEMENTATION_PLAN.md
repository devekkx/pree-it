# Realtime Architecture

## WebSocket Flow

Client

↓

Load Balancer

↓

Go WebSocket Gateway

↓

Redis Pub/Sub

↓

Other WebSocket Instances

## Why Redis Pub/Sub?

Because multiple backend instances need to broadcast events consistently.

Example:

- User A connects to Server 1
- User B connects to Server 2
- Redis propagates events between both servers

---

# Recommended Event Types

## Client → Server

{

**"type"**: **"message.send"**,

**"payload"**: {}

}

## Server → Client

{

**"type"**: **"message.created"**,

**"payload"**: {}

}

## Event Categories

### Messaging

- message.send
- message.created
- message.updated
- message.deleted

### Presence

- user.online
- user.offline
- user.typing
- user.stop_typing

### Rooms

- room.join
- room.leave

### Delivery

- message.delivered
- message.read

### System

- error
- reconnect
- heartbeat

---

# Recommended Svelte Stores

## Auth Store

- current user
- access token
- refresh status

## Chat Store

- active conversation
- messages
- unread counts

## Presence Store

- online users
- typing users

## WebSocket Store

- socket state
- reconnect status
- heartbeat timestamps

---

# WebSocket Reconnection Strategy

## Requirements

- Exponential backoff
- Token refresh handling
- Message queueing while offline
- Heartbeat detection

## Reconnect Delays

1s

2s

4s

8s

16s

---

# Security Requirements

## Backend

- Rate limiting
- JWT expiration
- Refresh token rotation
- WebSocket authentication
- CSRF protection
- Input validation
- SQL injection protection
- XSS sanitization

## Realtime Security

- Validate every incoming event
- Conversation membership checks
- Message size limits
- Connection throttling

# Suggested Development Order

## Phase 1 — Foundation

1. Initialize monorepo
2. Configure Docker Compose
3. Setup PostgreSQL
4. Setup Redis
5. Configure SvelteKit
6. Configure Go backend
7. infra (postgres, redis, nats, observability)
8. gateway-service
9. auth-service

## Phase 2 — Authentication

1. Registration
2. Login
3. JWT
4. Refresh tokens
5. Middleware

## Phase 3 — Realtime Infrastructure

1. WebSocket gateway
2. Connection manager
3. Event router
4. Redis Pub/Sub
5. Presence tracking

## Phase 4 — Messaging

1. Conversation creation
2. Message persistence
3. Message delivery
4. Read receipts
5. Typing indicators

## Phase 5 — Advanced Features

1. File uploads
2. Voice notes
3. Push notifications
4. Search
5. Encryption

# IMPORTANT REALITY CHECK

This is **not a beginner architecture anymore** .

To implement it correctly, you must build in this order:

### Phase 1 (foundation)

- infra (postgres, redis, nats, observability)
- gateway-service
- auth-service

### Phase 2 (core domain)

- user-service
- chat-service
- realtime-service

### Phase 3 (scale features)

- media-service
- notification-service
- presence-service

### Phase 4 (advanced)

- search-service
- analytics
- anti-spam / rate intelligence

# Initial MVP Goals

The first usable version should support:

- User authentication
- One-to-one chat
- Realtime delivery
- Online presence
- Persistent messages
- Responsive UI
- Reconnection handling

Avoid adding:

- Video calls
- E2E encryption
- AI features
- Kubernetes
- Complex microservices

until the core messaging pipeline is stable.
