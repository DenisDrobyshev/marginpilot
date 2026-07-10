# Build context is the repo root (see docker-compose.yml).
FROM golang:1.23-alpine AS build
WORKDIR /app
ENV GOWORK=off CGO_ENABLED=0
COPY shared ./shared
COPY proto ./proto
COPY services/identity ./services/identity
WORKDIR /app/services/identity
RUN go build -o /bin/identity ./cmd/identity

FROM gcr.io/distroless/static-debian12
COPY --from=build /bin/identity /identity
EXPOSE 8083 9102
ENTRYPOINT ["/identity"]
