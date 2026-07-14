# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /src

RUN apk add --no-cache ca-certificates tzdata

COPY go.mod ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/ais140-server ./cmd/server

# Runtime stage
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata \
	&& adduser -D -H -u 10001 appuser

WORKDIR /app

COPY --from=builder /out/ais140-server /app/ais140-server

USER appuser

EXPOSE 5001

ENV AIS140_HOST=0.0.0.0 \
	AIS140_PORT=5001 \
	AIS140_READ_TIMEOUT=5m \
	AIS140_WRITE_TIMEOUT=30s \
	AIS140_READ_BUFFER=4096

ENTRYPOINT ["/app/ais140-server"]
