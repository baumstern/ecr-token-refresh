FROM golang:1.16-alpine AS builder
WORKDIR /usr/src/app

# Copy project and build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -ldflags '-s' -v


# Multi stage build which reduces image size
FROM alpine:3.13

RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*
COPY --from=builder /usr/src/app/ecr-token-refresh .

CMD ["./ecr-token-refresh"]

