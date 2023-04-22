# Use a trusted base image
FROM golang:1.20.3-alpine as build

# Set a non-root user
USER nobody:nobody

# Copy the application files
COPY . /app

# Build the application
WORKDIR /app
RUN go build -o app .

# Use a distroless image for security
FROM gcr.io/distroless/base-debian10

# Copy only the necessary files from the build container
COPY --from=build /app/app /app

# Set a non-root user
USER nonroot

# Set the entrypoint to run the application
CMD ["/app/app"]
