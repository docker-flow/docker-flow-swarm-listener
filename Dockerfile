FROM golang:1.9 AS build
ADD . /src
WORKDIR /src
RUN go get -d -v -t
RUN go build -v -o docker-flow-swarm-listener



FROM alpine:3.6
MAINTAINER 	Viktor Farcic <viktor@farcic.com>

ENV DF_DOCKER_HOST="unix:///var/run/docker.sock" \
    DF_NOTIFICATION_URL="" \
    DF_INTERVAL="5" \
    DF_RETRY="50" \
    DF_RETRY_INTERVAL="5" \
    DF_NOTIFY_LABEL="com.df.notify"

EXPOSE 8080

CMD ["docker-flow-swarm-listener"]

COPY --from=build /src/docker-flow-swarm-listener /usr/local/bin/docker-flow-swarm-listener
RUN chmod +x /usr/local/bin/docker-flow-swarm-listener
