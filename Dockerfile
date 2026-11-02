# ── Build stage ──────────────────────────────────────────────────────────────
FROM golang:1.27-alpine AS builder

WORKDIR /app

COPY go.mod ./
# go.sum will be copied once dependencies are added
COPY go.sum* ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o terminus ./cmd/server

# ── Run stage ─────────────────────────────────────────────────────────────────
FROM alpine:latest

RUN addgroup -S terminus && adduser -S terminus -G terminus

WORKDIR /app

COPY --from=builder /app/terminus .

USER terminus

EXPOSE 8081

ENTRYPOINT ["./terminus"]
