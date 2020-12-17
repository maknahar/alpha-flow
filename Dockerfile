FROM golang:1.15-alpine as builder

WORKDIR /build

# Fetch dependencies.
COPY go.mod go.sum ./
RUN go mod download

# Copy code.
COPY . .

# Build the command inside the container.
RUN CGO_ENABLED=0 GOOS=linux go build -v -o main main.go

FROM scratch

# Copy the binary to the production image from the builder stage.
COPY --from=builder /build/main /main
COPY internal/db/migrations/postgres/ internal/db/migrations/postgres/

EXPOSE 9001

# Entrypoint.
ENTRYPOINT ["/main"]
