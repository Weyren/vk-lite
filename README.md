# vk-lite

Minimal social network API demo in Go, Gin, PostgreSQL, Redis, RabbitMQ and Docker Compose.

## Run locally in containers

Start Docker Desktop first, then run:

```powershell
docker compose up -d --build
```

API will be available at:

```text
http://localhost:8080
```

RabbitMQ UI:

```text
http://localhost:15672
login: guest
password: guest
```

Stop containers:

```powershell
docker compose down
```

## Demo scenario

Health check:

```powershell
Invoke-RestMethod http://localhost:8080/healthz
```

Create two users:

```powershell
Invoke-RestMethod -Method POST http://localhost:8080/api/v1/users `
  -ContentType "application/json" `
  -Body '{"email":"alice@example.com","password":"secret123","name":"Alice"}'

Invoke-RestMethod -Method POST http://localhost:8080/api/v1/users `
  -ContentType "application/json" `
  -Body '{"email":"bob@example.com","password":"secret123","name":"Bob"}'
```

Login:

```powershell
$alice = Invoke-RestMethod -Method POST http://localhost:8080/api/v1/auth/login `
  -ContentType "application/json" `
  -Body '{"email":"alice@example.com","password":"secret123"}'

$bob = Invoke-RestMethod -Method POST http://localhost:8080/api/v1/auth/login `
  -ContentType "application/json" `
  -Body '{"email":"bob@example.com","password":"secret123"}'
```

Alice follows Bob:

```powershell
Invoke-RestMethod -Method POST http://localhost:8080/api/v1/users/2/follow `
  -Headers @{ Authorization = "Bearer $($alice.access_token)" }
```

Bob creates a post:

```powershell
Invoke-RestMethod -Method POST http://localhost:8080/api/v1/posts `
  -Headers @{ Authorization = "Bearer $($bob.access_token)" } `
  -ContentType "application/json" `
  -Body '{"content":"Hello from vk-lite in Docker!"}'
```

Alice likes Bob's post and opens her feed:

```powershell
Invoke-RestMethod -Method POST http://localhost:8080/api/v1/posts/1/like `
  -Headers @{ Authorization = "Bearer $($alice.access_token)" }

Invoke-RestMethod http://localhost:8080/api/v1/feed `
  -Headers @{ Authorization = "Bearer $($alice.access_token)" }
```
