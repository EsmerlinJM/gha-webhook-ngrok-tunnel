# Use a trusted base image
FROM golang:1.20.3-alpine as build

# Set a non-root user
USER nonroot:nonroot

# Copy the application files
COPY . /app

# Build the application
WORKDIR /app
RUN CGO_ENABLED=0 go build -ldflags="-w -s" -v -o app .

# Use a distroless image for security
FROM gcr.io/distroless/static

COPY --from=build /app/app /app

# Set a non-root user
USER nonroot

# Set the entrypoint to run the application
ENTRYPOINT ["/app"]
