# Docker Flow: Swarm Listener

The project listens to Docker Swarm events and sends requests when a change occurs. At the moment, the only supported option is to send a notification when a new service is created. More extensive feature support is coming soon.

## Example

The example that follows will use the *Swarm Listener* to reconfigure the [Docker Flow: Proxy](https://github.com/vfarcic/docker-flow-proxy) whenever a new service is created.

I will assume that you already have a Swarm cluster set up.

Let's run a Proxy service. We'll use it as a way to demonstrate how *Swarm Listener* works.

```bash
docker network create --driver overlay proxy

docker service create --name proxy \
    -p 80:80 \
    -p 443:443 \
    -p 8080:8080 \
    --network proxy \
    -e MODE=swarm \
    vfarcic/docker-flow-proxy
```

Next, we'll create the `swarm-listener` service.

```bash
docker service create --name swarm-listener \
    --network proxy \
    --mount "type=bind,source=/var/run/docker.sock,target=/var/run/docker.sock" \
    -e DF_NOTIFICATION_URL=http://proxy:8080/v1/docker-flow-proxy/reconfigure \
    vfarcic/docker-flow-swarm-listener
```

The service is attached to the proxy service (just as the `proxy` service), mounts the Docker socket, and declares the environment variable `DF_NOTIFICATION_URL`. We'll see the purpose of the variable soon.

Now we can deploy a service that will trigger the listener.

```bash
docker service create --name go-demo-db \
  --network proxy \
  mongo

docker service create --name go-demo \
  -e DB=go-demo-db \
  --network proxy \
  -l DF_NOTIFY=true \
  -l DF_servicePath=/demo \
  -l DF_port=8080 \
  vfarcic/go-demo
```

Please note that we declared the label `DF_NOTIFY`. Only services with this label (it can hold any value) will be eligible to receive notifications through *Docker Flow: Swarm Listener*. We also declared a couple of other labels (`DF_servicePath` and `DF_port`).

Before proceeding, we should wait until all the services are up and running. Please use the `docker service ls` command to check the status.

Let's see the *Docker Flow: Swarm Listener* logs.

```bash
ID=$(docker ps -q -f ancestor=vfarcic/docker-flow-swarm-listener)

docker logs $ID
```

We found the ID of the container and displayed the logs. The output is as follows.

```
Starting Docker Flow: Swarm Listener
Sending a service notification to http://proxy:8080/v1/docker-flow-proxy/reconfigure?serviceName=go-demo&port=8080&servicePath=/demo
```

As you can see, the listener detected that the `go-demo` service has the label `DF_NOTIFY` and sent the notification request. The address of the notification request is the value of the environment variable `DF_NOTIFICATION_URL` declared in the `swarm-listener` service. The parameters are a combination of the service name and all the labels prefixed with `DF_`.

## Environment Variables

The following environment variables can be used when creating the `swarm listener` service.

|Name               |Description                                               |Default Value|
|-------------------|----------------------------------------------------------|-------------|
|DF_DOCKER_HOST     |Path to the Docker socket                   |unix:///var/run/docker.sock|
|DF_NOTIFICATION_URL|The URL that will be used to send notification requests   |             |
|DF_INTERVAL        |Interval (in seconds) between service discovery requests  |5            |
|DF_RETRY           |Number of notification request retries                    |5            |
|DF_RETRY_INTERVAL  |Interval (in seconds) between notification request retries|5            |
