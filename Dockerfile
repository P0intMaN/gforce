# Stage 1: build
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /out/gforce ./cmd/gforce

# Stage 2: minimal runtime image
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /out/gforce /gforce
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

EXPOSE 8080

USER nonroot:nonroot

ENTRYPOINT ["/gforce", "serve"]
