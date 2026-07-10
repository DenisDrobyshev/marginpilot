# Build context is the repo root (see docker-compose.yml).
FROM golang:1.23-alpine AS build
WORKDIR /app
ENV GOWORK=off CGO_ENABLED=0
COPY shared ./shared
COPY proto ./proto
COPY services/budget ./services/budget
WORKDIR /app/services/budget
RUN go build -o /bin/budget ./cmd/budget

FROM gcr.io/distroless/static-debian12
COPY --from=build /bin/budget /budget
EXPOSE 8082 9101
ENTRYPOINT ["/budget"]
