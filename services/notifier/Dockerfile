# Build context is the repo root (see docker-compose.yml).
FROM golang:1.23-alpine AS build
WORKDIR /app
ENV GOWORK=off CGO_ENABLED=0
COPY shared ./shared
COPY services/notifier ./services/notifier
WORKDIR /app/services/notifier
RUN go build -o /bin/notifier ./cmd/notifier

FROM gcr.io/distroless/static-debian12
COPY --from=build /bin/notifier /notifier
EXPOSE 8086
ENTRYPOINT ["/notifier"]
