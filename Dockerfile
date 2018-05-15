FROM golang:1.10.0-alpine3.7 AS build

RUN apk add --update git
ADD . /src
WORKDIR /src
RUN go get -d -v -t
RUN go build -v -o docker-flow-swarm-listener



FROM alpine:3.7
LABEL maintainer="Viktor Farcic <viktor@farcic.com>"

ENV DF_DOCKER_HOST="unix:///var/run/docker.sock" \
    DF_NOTIFICATION_URL="" \
    DF_RETRY="50" \
    DF_RETRY_INTERVAL="5" \
    DF_NOTIFY_LABEL="com.df.notify" \
    DF_INCLUDE_NODE_IP_INFO="false"

EXPOSE 8080

CMD ["docker-flow-swarm-listener"]

HEALTHCHECK --interval=10s --start-period=10s --timeout=5s CMD wget -qO- "http://localhost:8080/v1/docker-flow-swarm-listener/ping"

COPY --from=build /src/docker-flow-swarm-listener /usr/local/bin/docker-flow-swarm-listener
RUN chmod +x /usr/local/bin/docker-flow-swarm-listener
