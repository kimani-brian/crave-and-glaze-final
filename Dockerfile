# STAGE 1: Build the Go Binary
FROM golang:1.25-alpine AS builder

# Install git (needed for dependencies)
RUN apk add --no-cache git

# Set working directory inside the container
WORKDIR /app

# Copy dependency files and download
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build the application
# -o main names the binary "main"
# ./cmd/server/main.go is the path to your entry point
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/server/main.go

# STAGE 2: Run the Binary (Tiny Image)
FROM alpine:latest

# Install CA Certs (Required for MPESA & Gmail HTTPS)
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Copy the binary from the builder
COPY --from=builder /app/main .

# CRITICAL: Copy the "web" folder (HTML templates & CSS)
# If you don't do this, the app will panic because it can't find templates
COPY --from=builder /app/web ./web

COPY --from=builder /app/schema.sql ./schema.sql

# Expose the port your app runs on
EXPOSE 8080

# Run the binary
CMD ["./main"]