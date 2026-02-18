# Build stage
FROM golang:1.23-alpine AS builder
RUN apk add --no-cache ca-certificates
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /brain ./cmd/brain

# Runtime stage
FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app

COPY --from=builder /brain /app/brain
COPY migrations /app/migrations
COPY internal/correlator/rules/chains.yml /app/internal/correlator/rules/chains.yml
COPY internal/correlator/taxonomy/behaviors.yml /app/internal/correlator/taxonomy/behaviors.yml

EXPOSE 8080

ENTRYPOINT ["/app/brain"]
