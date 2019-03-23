FROM alpine:latest as certificates
RUN apk update && apk add --no-cache ca-certificates && update-ca-certificates


FROM golang:1.12 as builder

WORKDIR /build/github.com/benclapp/updog
COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-X main.version=$(cat VERSION)" -o /app/updog


FROM scratch

# Copy certs from alpine as they don't exist from scratch
COPY --from=certificates /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/updog /app/updog

# Import a sample config
COPY updog.yaml /app/updog.yaml

WORKDIR /app
ENTRYPOINT [ "/app/updog" ]
