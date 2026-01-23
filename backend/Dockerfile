FROM golang:1.23-bookworm AS builder

RUN apt-get update && apt-get install -y sqlite3 libsqlite3-dev && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 GOOS=linux GOARCH=arm64 go build -tags="libsqlite3" -o /out/server ./main.go

FROM debian:bookworm-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
       ffmpeg \
       ca-certificates \
       tzdata \
       curl \
       libsqlite3-0 \
       python3 \
       python3-pip \
    && rm -rf /var/lib/apt/lists/* \
    && pip3 install --break-system-packages yt-dlp

WORKDIR /app
COPY --from=builder /out/server /app/server
COPY soundcloud/downloader.py /app/soundcloud/downloader.py

EXPOSE 8080
VOLUME ["/data"]

ENTRYPOINT ["/app/server"]

