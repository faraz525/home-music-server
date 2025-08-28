FROM golang:1.22-bookworm AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o /out/server ./

FROM alpine:3.20

RUN apk add --no-cache ffmpeg ca-certificates tzdata curl

WORKDIR /app
COPY --from=builder /out/server /app/server

EXPOSE 8080
VOLUME ["/data"]

ENTRYPOINT ["/app/server"]

