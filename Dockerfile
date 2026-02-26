FROM golang:1.25-bookworm AS build

WORKDIR /src
COPY . .
RUN cd cmd/xmppd && go mod download
RUN cd cmd/xmppd && CGO_ENABLED=1 GOOS=linux go build -o /out/xmppd .

FROM debian:bookworm-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates libsqlite3-0 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=build /out/xmppd /usr/local/bin/xmppd

ENV XMPP_ADDR=":5222" \
    XMPP_DOMAIN="example.com" \
    XMPP_STORAGE="file" \
    XMPP_STORAGE_PATH="/var/lib/xmpp/data"

EXPOSE 5222
VOLUME ["/var/lib/xmpp"]

ENTRYPOINT ["/usr/local/bin/xmppd"]
