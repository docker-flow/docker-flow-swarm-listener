FROM haproxy:1.6-alpine
MAINTAINER 	Viktor Farcic <viktor@farcic.com>

RUN mkdir /lib64 && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2

ENV DF_DOCKER_HOST="unix:///var/run/docker.sock" \
    DF_NOTIFICATION_URL="" \
    DF_INTERVAL="5" \
    DF_RETRY="5" \
    DF_RETRY_INTERVAL="5"

CMD ["docker-flow-swarm-listener"]

COPY docker-flow-swarm-listener /usr/local/bin/docker-flow-swarm-listener
RUN chmod +x /usr/local/bin/docker-flow-swarm-listener
