# vk-lite

Minimal social network API demo in Go, Gin, PostgreSQL, Redis, RabbitMQ and Docker Compose.

## Run locally in containers

Start Docker Desktop first, then run:

```powershell
docker compose up -d --build
```

Web UI will be available at:

```text
http://localhost:8080
```

API endpoints are available under:

```text
http://localhost:8080/api/v1
```

RabbitMQ UI:

```text
http://localhost:15672
login: guest
password: guest
```

Prometheus:

```text
http://localhost:9090
```

Grafana:

```text
http://localhost:3000
login: admin
password: admin
```

The Grafana dashboard `vk-lite API` is provisioned automatically from `monitoring/grafana/dashboards`.

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

Upload media for a post:

```powershell
$media = curl.exe -s -X POST http://localhost:8080/api/v1/media `
  -H "Authorization: Bearer $($bob.access_token)" `
  -F "file=@.\photo.jpg" | ConvertFrom-Json

Invoke-RestMethod -Method POST http://localhost:8080/api/v1/posts `
  -Headers @{ Authorization = "Bearer $($bob.access_token)" } `
  -ContentType "application/json" `
  -Body "{`"content`":`"Post with media`",`"media_url`":`"$($media.url)`"}"
```

Open profile posts:

```powershell
Invoke-RestMethod http://localhost:8080/api/v1/users/2/posts?page=1"&"per_page=10 `
  -Headers @{ Authorization = "Bearer $($alice.access_token)" }
```

Alice likes Bob's post and opens her feed:

```powershell
Invoke-RestMethod -Method POST http://localhost:8080/api/v1/posts/1/like `
  -Headers @{ Authorization = "Bearer $($alice.access_token)" }

Invoke-RestMethod http://localhost:8080/api/v1/feed `
  -Headers @{ Authorization = "Bearer $($alice.access_token)" }
```
