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

# BPM / musical-key analysis prereq. `essentia-examples` provides the
# `streaming_extractor_music` binary that the analysis worker shells out to.
# Kept as a separate optional layer: if the package is unavailable (e.g. on
# a repo without contrib/non-free, or a future Debian release that renames it)
# the image still builds and the analysis worker disables itself with a
# startup warning. Install it manually later to enable the feature.
RUN apt-get update \
    && apt-get install -y --no-install-recommends essentia-examples \
    && rm -rf /var/lib/apt/lists/* \
    || echo "WARNING: essentia-examples not installed; BPM/key analysis will be disabled."

WORKDIR /app
COPY --from=builder /out/server /app/server
COPY soundcloud/downloader.py /app/soundcloud/downloader.py

EXPOSE 8080
VOLUME ["/data"]

ENTRYPOINT ["/app/server"]

