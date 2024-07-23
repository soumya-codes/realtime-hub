# hubclient/docker/Dockerfile

# Use the chainguard base image for Go
FROM cgr.dev/chainguard/go:latest AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy the go.mod and go.sum files
COPY go.mod go.sum ./

# Download the Go module dependencies
RUN go mod download -x

# Copy the rest of the application code
COPY . .

# Build the application
RUN CGO_ENABLED=0 go build -o hubclient ./cmd/hubclient

# Set the executable permission for the binary
RUN chmod +x ./hubclient

# Use a smaller base image to run the application
#FROM cgr.dev/chainguard/go:latest
#FROM scratch
FROM cgr.dev/chainguard/wolfi-base

# Set the working directory inside the container
WORKDIR /app

# Copy the built binary from the builder stage
COPY --from=builder /app/hubclient .

# Copy the HTML template file
COPY internal/templates/index.html ./internal/templates/index.html

# Have a non-root user
USER 65532:65532

# Expose the port that the application listens on
EXPOSE 9081
EXPOSE 9082

# Set the entrypoint for the container
ENTRYPOINT ["./hubclient"]