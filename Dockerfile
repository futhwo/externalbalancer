# ====== Build stage ======
FROM golang:1.22 AS builder

# Set up env
WORKDIR /workspace

# Pre-cache deps
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o externalbalancer-operator ./main.go

# ====== Runtime stage ======
FROM gcr.io/distroless/static:nonroot

WORKDIR /
COPY --from=builder /workspace/externalbalancer-operator /externalbalancer-operator

USER nonroot:nonroot

ENTRYPOINT ["/externalbalancer-operator"]