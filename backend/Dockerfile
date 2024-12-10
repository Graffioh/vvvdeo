# -- Stage 1 -- #
# Compile the Go app.
FROM golang:1.22.3-alpine as builder
WORKDIR /app

# Copy the necessary files from the project structure
COPY . .

# Build the Go app
RUN go build -mod=vendor -o bin/vvvdeo

# -- Stage 2 -- #
# Create the final environment with the latest FFmpeg.
FROM alpine:latest

# Install the latest FFmpeg and other dependencies
RUN apk --no-cache add ca-certificates yt-dlp \
    && apk --no-cache --repository http://dl-cdn.alpinelinux.org/alpine/edge/community add ffmpeg

WORKDIR /root/

# Copy the binary from the builder stage
COPY --from=builder /app/bin/vvvdeo /usr/local/bin/

# Ensure the binary is executable
RUN chmod +x /usr/local/bin/vvvdeo

# Set the command
CMD ["vvvdeo"]
