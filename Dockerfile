FROM golang:1.22-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/api ./cmd/api

FROM alpine:3.20
# ca-certificates is needed for TLS to a managed Postgres/Redis (Tencent
# Cloud managed services likely require TLS) - scratch would be smaller but
# can't verify TLS certs at all.
RUN apk add --no-cache ca-certificates && \
    adduser -D -u 10001 appuser
WORKDIR /app
COPY --from=build /out/api .
USER appuser
EXPOSE 8080
ENTRYPOINT ["./api"]