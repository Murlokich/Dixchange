FROM golang:1.26 AS build

WORKDIR /src

# Dependencies first, so code changes don't invalidate the module download layer.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/app ./cmd

# Small, but with a shell and apk for debugging (`docker compose exec app sh`).
FROM alpine:3.24

RUN addgroup -S app && adduser -S -G app app
USER app

# The app loads migrations from file://../migrations, relative to its working
# directory, so the layout mirrors the repository: binary in cmd/, migrations next to it.
WORKDIR /app/cmd
COPY --from=build /out/app /app/cmd/app
COPY migrations /app/migrations

EXPOSE 8080

# Exec form: the app is PID 1 and receives SIGTERM from `docker stop` directly.
ENTRYPOINT ["/app/cmd/app"]
