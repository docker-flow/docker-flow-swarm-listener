FROM alpine:3.1
MAINTAINER 	Viktor Farcic <viktor@farcic.com>

ENV DF_DOCKER_HOST="unix:///var/run/docker.sock" \
    DF_NOTIFICATION_URL="" \
    DF_INTERVAL="5" \
    DF_RETRY="50" \
    DF_RETRY_INTERVAL="5"

EXPOSE 8080

CMD ["docker-flow-swarm-listener"]

COPY docker-flow-swarm-listener /usr/local/bin/docker-flow-swarm-listener
RUN chmod +x /usr/local/bin/docker-flow-swarm-listener
