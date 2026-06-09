# syntax=docker/dockerfile:1

######## Stage 1: Build the React + Vite PWA bundle ########
FROM node:20-alpine AS web-builder

WORKDIR /app/web/app

COPY web/app/package*.json ./
RUN npm ci

COPY web/app ./
RUN npm run build

######## Stage 2: Build the Go binary with the embedded bundle ########
FROM golang:1.25-alpine AS go-builder

WORKDIR /app

COPY go.mod go.sum ./
COPY vendor ./vendor
COPY cmd ./cmd
COPY internal ./internal
COPY web ./web

# Bring in the freshly built bundle so //go:embed all:dist picks it up.
COPY --from=web-builder /app/web/dist ./web/dist

RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -o main ./cmd/server

######## Stage 3: Minimal runtime image ########
FROM alpine:latest

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=go-builder /app/main .

EXPOSE 8080

CMD ["./main"]
