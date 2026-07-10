# Build context is the repo root (see docker-compose.yml).
FROM golang:1.23-alpine AS build
WORKDIR /app
ENV GOWORK=off CGO_ENABLED=0
COPY shared ./shared
COPY proto ./proto
COPY services/rating ./services/rating
WORKDIR /app/services/rating
RUN go build -o /bin/rating ./cmd/rating

FROM gcr.io/distroless/static-debian12
COPY --from=build /bin/rating /rating
EXPOSE 8084 9103
ENTRYPOINT ["/rating"]
