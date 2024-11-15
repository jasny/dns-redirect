# Build stage
FROM golang:alpine AS builder

# Set environment variables for Go build
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64

# Install necessary packages
RUN apk --no-cache add git

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum to download dependencies first (better caching)
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the rest of the application code
COPY . .

# Build the Go application
RUN go build -o server

# Final stage
FROM alpine:latest

# Install necessary packages
RUN apk --no-cache add ca-certificates

# Set the working directory inside the container
RUN mkdir /app
WORKDIR /app

# Copy the compiled binary from the build stage
COPY --from=builder /app/server .

# Copy HTML pages
COPY static ./static

# Expose HTTP and HTTPS ports
EXPOSE 80 443

# Command to run the application
CMD ["./server"]
