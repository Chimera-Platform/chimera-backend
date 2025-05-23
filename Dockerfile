# Start from the official Go image
FROM golang:1.23.4 as builder

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the code
COPY . .

# Build the Go app (force Linux binary)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o main .

# Use a minimal base image to run the app
FROM alpine:latest

# Set working directory in alpine
WORKDIR /root/

# Copy the built binary from the builder
COPY --from=builder /app/main .

# Expose the port your app runs on
EXPOSE 8080

# Command to run the executable
CMD ["./main"]
