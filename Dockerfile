
# Dockerfile
#FROM golang:1.22

# Set the working directory
#WORKDIR /app
#WORKDIR ${GOPATH}/avito-shop/
#COPY . ${GOPATH}/avito-shop/

# Install Air for live reload
#RUN go install github.com/air-verse/air@v1.52.3
#RUN go build -o /build ./internal/cmd \
#    && go clean -cache -modcache

# Copy the Go module files
#COPY go.mod go.sum ./

# Download dependencies
#RUN go mod download

# Copy the source code
#COPY . .

# Expose the application port
#EXPOSE 8080

# Use Air as the entrypoint
#CMD ["/build"]
#CMD ["air"]
# Stage 1: Build the application (using a builder image)
FROM golang:1.22 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . /app

RUN CGO_ENABLED=0 go build -o /app/avito-shop-service -ldflags="-s -w" ./internal/cmd
#RUN go build -o /app/avito-shop-service -ldflags="-s -w" ./internal/cmd # Build in builder stage

RUN ls -l /app
# Stage 2: Create the final image (using a smaller image)
FROM alpine:latest AS runner

WORKDIR /app

COPY --from=builder /app/avito-shop-service /app/
COPY --from=builder /app/.env /app/

EXPOSE 8080

CMD ["/app/avito-shop-service"]
