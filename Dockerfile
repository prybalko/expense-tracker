FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Copy source
COPY . .

# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -o main ./cmd/server

######## Start a new stage from scratch #######
FROM alpine:latest  

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /app/main .
COPY --from=builder /app/web/templates ./web/templates
COPY --from=builder /app/web/static ./web/static

# Expose port 8080 to the outside world
EXPOSE 8080

# Command to run the executable
CMD ["./main"]
