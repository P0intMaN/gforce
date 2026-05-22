# Stage 1: Build the React UI
FROM node:20-alpine AS ui-builder
WORKDIR /ui
COPY ui/package*.json ./
RUN npm ci
COPY ui/ ./
RUN npm run build
# Output: /ui/dist

# Stage 2: Build the Go binaries
FROM golang:1.22-alpine AS go-builder
RUN apk add --no-cache git ca-certificates tzdata
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Embed the built UI into the binary
COPY --from=ui-builder /ui/dist ./ui/dist
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-w -s -X main.version=$(git describe --tags --always 2>/dev/null || echo dev)" \
    -o gforce ./cmd/gforce
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-w -s" \
    -o gforce-operator ./cmd/operator

# Stage 3: Minimal runtime image
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=go-builder /app/gforce /gforce
COPY --from=go-builder /app/gforce-operator /gforce-operator
EXPOSE 8080 2222
USER nonroot:nonroot
ENTRYPOINT ["/gforce"]
