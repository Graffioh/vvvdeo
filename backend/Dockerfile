FROM golang:1.22.3-alpine AS builder
WORKDIR /app

COPY . .

# Generate vendor directory and build
RUN go mod vendor && \
    go build -mod=vendor -o bin/vvvdeo

FROM alpine:latest

# Install the latest FFmpeg and other dependencies
RUN apk --no-cache add ca-certificates yt-dlp \
    && apk --no-cache --repository http://dl-cdn.alpinelinux.org/alpine/edge/community add ffmpeg

WORKDIR /root/

# Copy the binary from the builder stage
COPY --from=builder /app/bin/vvvdeo /usr/local/bin/

# Ensure the binary is executable
RUN chmod +x /usr/local/bin/vvvdeo

CMD ["vvvdeo"]
