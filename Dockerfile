FROM debian

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -ldflags '-s' -v

COPY ./ecr-token-refresh /ecr-token-refresh

ENTRYPOINT /ecr-token-refresh
