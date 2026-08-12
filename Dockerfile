# Build stage
FROM golang:1.23-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o server ./cmd/server

# Runtime stage
FROM gcr.io/distroless/static-debian12
WORKDIR /app

ENV DATA_DIR=/app/data
ENV PORT=8000

COPY --from=builder /app/server /app/server
COPY --from=builder /app/public /app/public
COPY --from=builder /app/migrations /app/migrations

EXPOSE 8000
VOLUME ["/app/data"]

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD ["/app/server", "-healthcheck"]

CMD ["/app/server"]
