FROM alpine:3.7
LABEL maintainer="Viktor Farcic <viktor@farcic.com>"

ENV DF_DOCKER_HOST="unix:///var/run/docker.sock" \
    DF_NOTIFICATION_URL="" \
    DF_RETRY="50" \
    DF_RETRY_INTERVAL="5" \
    DF_NOTIFY_LABEL="com.df.notify" \
    DF_INCLUDE_NODE_IP_INFO="false" \
    DF_SERVICE_POLLING_INTERVAL="-1" \
    DF_USE_DOCKER_SERVICE_EVENTS="true" \
    DF_NODE_POLLING_INTERVAL="-1" \
    DF_USE_DOCKER_NODE_EVENTS="true" \
    DF_SERVICE_NAME_PREFIX="" \
    DF_NOTIFY_CREATE_SERVICE_METHOD="GET" \
    DF_NOTIFY_REMOVE_SERVICE_METHOD="GET"

EXPOSE 8080

CMD ["docker-flow-swarm-listener"]

HEALTHCHECK --interval=10s --start-period=5s --timeout=5s CMD wget -qO- "http://localhost:8080/v1/docker-flow-swarm-listener/ping"

COPY --from=build /src/docker-flow-swarm-listener /usr/local/bin/docker-flow-swarm-listener
RUN chmod +x /usr/local/bin/docker-flow-swarm-listener
