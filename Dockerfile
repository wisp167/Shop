
# Dockerfile
FROM golang:1.22

# Set the working directory
WORKDIR /app
#WORKDIR ${GOPATH}/avito-shop/
#COPY . ${GOPATH}/avito-shop/

# Install Air for live reload
RUN go install github.com/air-verse/air@v1.52.3
#RUN go build -o /build ./internal/cmd \
#    && go clean -cache -modcache

# Copy the Go module files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Expose the application port
EXPOSE 8080

# Use Air as the entrypoint
#CMD ["/build"]
CMD ["air"]
