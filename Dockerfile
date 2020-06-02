# Start from the latest golang base image
FROM golang:latest as builder
# Set the Current Working Directory inside the container
WORKDIR /app
# Copy go mod and sum files
COPY go.mod go.sum ./
# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download
# Copy the source from the current directory to the Working Directory inside the container
COPY . .
# Build the Go app
RUN CGO_ENABLED=0 go build -o woffu-bot .


######## Start a new stage from scratch #######
FROM alpine:latest
ENV TZ Europe/Madrid

# Install deps for HTTPS and time zone
RUN apk add -U tzdata  ca-certificates
# Copy the Pre-built binary file from the previous stage
COPY --from=builder /app/woffu-bot /usr/local/bin/
# Command to run the executable
ENTRYPOINT cp /usr/share/zoneinfo/${TZ} /etc/localtime && woffu-bot
