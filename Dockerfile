FROM golang:1.26.5-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/url-shortener ./cmd/shortener

FROM alpine:3.22
RUN apk add --no-cache ca-certificates && addgroup -S app && adduser -S -G app app
WORKDIR /app
COPY --from=builder /out/url-shortener /app/url-shortener
USER app
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=2s --retries=3 CMD wget -qO- http://127.0.0.1:8080/health/live >/dev/null || exit 1
ENTRYPOINT ["/app/url-shortener"]
