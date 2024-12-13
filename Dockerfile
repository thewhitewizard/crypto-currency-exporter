# Use a minimal base image (Alpine) to keep the container size small
FROM alpine:3.18

# Copy the binary from the builder stage
ADD  bin/crypto-currency-exporter.amd64  /usr/local/bin/exporter

# Command to run the application
CMD ["exporter"]