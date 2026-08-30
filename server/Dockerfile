# syntax=docker/dockerfile:1

# Build stage: compile a static binary so the final image needs no libc.
FROM golang:1.26 AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags='-s -w' -o /out/mu3ic-server ./cmd/server

# Final stage: distroless static, non-root. All deps (pgx, dhowden/tag,
# bcrypt, minio-go) are pure Go, so nothing dynamic is linked.
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/mu3ic-server /usr/local/bin/mu3ic-server
# Migrations are applied on startup (store.RunMigrations) from this path.
COPY --from=build /src/migrations /migrations
ENV MIGRATIONS_DIR=/migrations
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/usr/local/bin/mu3ic-server"]
