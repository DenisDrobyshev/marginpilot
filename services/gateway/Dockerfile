# Build context is the repo root (see docker-compose.yml).
FROM golang:1.23-alpine AS build
WORKDIR /app
ENV GOWORK=off CGO_ENABLED=0
COPY shared ./shared
COPY proto ./proto
COPY services/gateway ./services/gateway
WORKDIR /app/services/gateway
RUN go build -o /bin/gateway ./cmd/gateway

FROM gcr.io/distroless/static-debian12
COPY --from=build /bin/gateway /gateway
EXPOSE 8080
ENTRYPOINT ["/gateway"]
