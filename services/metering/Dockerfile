# Build context is the repo root (see docker-compose.yml).
FROM golang:1.23-alpine AS build
WORKDIR /app
ENV GOWORK=off CGO_ENABLED=0
COPY shared ./shared
COPY proto ./proto
COPY services/metering ./services/metering
WORKDIR /app/services/metering
RUN go build -o /bin/metering ./cmd/metering

FROM gcr.io/distroless/static-debian12
COPY --from=build /bin/metering /metering
EXPOSE 8081
ENTRYPOINT ["/metering"]
