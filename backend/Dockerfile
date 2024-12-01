# This is a standard Dockerfile for building a Go app.
# It is a multi-stage build: the first stage compiles the Go source into a binary, and
#   the second stage copies only the binary into an alpine base.

# -- Stage 1 -- #
# Compile the app.
FROM golang:1.12-alpine as builder
WORKDIR /app

# Copy the necessary files from the project structure
COPY . .

# Build the Go app
RUN go build -mod=vendor -o bin/veedeo

# -- Stage 2 -- #
# Create the final environment with the compiled binary.
FROM alpine

# Install any required dependencies, including ffmpeg
RUN apk --no-cache add ca-certificates ffmpeg

WORKDIR /root/

# Copy the binary from the builder stage and set it as the default command
COPY --from=builder /app/bin/veedeo /usr/local/bin/

CMD ["veedeo"]
