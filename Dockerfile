FROM debian

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates

COPY ./app /app

ENTRYPOINT /app
