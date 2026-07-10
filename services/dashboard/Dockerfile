# Build context is the repo root (see docker-compose.yml).
FROM golang:1.23-alpine AS build
WORKDIR /app
ENV GOWORK=off CGO_ENABLED=0
COPY shared ./shared
COPY services/dashboard ./services/dashboard
WORKDIR /app/services/dashboard
RUN go build -o /bin/dashboard ./cmd/dashboard

FROM gcr.io/distroless/static-debian12
COPY --from=build /bin/dashboard /dashboard
EXPOSE 8087
ENTRYPOINT ["/dashboard"]
