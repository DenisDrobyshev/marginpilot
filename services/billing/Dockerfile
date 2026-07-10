# Build context is the repo root (see docker-compose.yml).
FROM golang:1.23-alpine AS build
WORKDIR /app
ENV GOWORK=off CGO_ENABLED=0
COPY shared ./shared
COPY services/billing ./services/billing
WORKDIR /app/services/billing
RUN go build -o /bin/billing ./cmd/billing

FROM gcr.io/distroless/static-debian12
COPY --from=build /bin/billing /billing
EXPOSE 8085
ENTRYPOINT ["/billing"]
