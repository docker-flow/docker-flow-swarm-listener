# Docker Flow Swarm Listener Walkthrough

This tutorial will walk you through some of the most common use cases.

!!! info
	If you are a Windows user, please run all the examples from *Git Bash* (installed through *Docker Toolbox* or *Git*).

## Sending Notification Requests On Service Creation and Removal

The example that follows will use the *Swarm Listener* to reconfigure the [Docker Flow: Proxy](https://github.com/vfarcic/docker-flow-proxy) whenever a new service is created.

I will assume that you already have a Swarm cluster set up with Docker Machines. If that's not the case, feel free to use the [scripts/dm-swarm.sh](https://github.com/vfarcic/docker-flow-swarm-listener/blob/master/scripts/dm-swarm.sh) script to create a three nodes cluster.

Let's run the Proxy service. We'll use it as a way to demonstrate how *Swarm Listener* works.

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
    -e DF_NOTIFY_CREATE_SERVICE_URL=http://proxy:8080/v1/docker-flow-proxy/reconfigure \
    -e DF_NOTIFY_REMOVE_SERVICE_URL=http://proxy:8080/v1/docker-flow-proxy/remove \
    --constraint 'node.role==manager' \
    vfarcic/docker-flow-swarm-listener
```

The service is attached to the proxy network (just as the `proxy` service), mounts the Docker socket, and declares the environment variables `DF_NOTIFY_CREATE_SERVICE_URL` and `DF_NOTIFY_REMOVE_SERVICE_URL`. We'll see the purpose of the variables soon.

Now we can deploy a service that will trigger the listener.

```bash
docker service create --name go-demo-db \
  --network proxy \
  mongo

docker service create --name go-demo \
  -e DB=go-demo-db \
  --network proxy \
  -l com.df.notify=true \
  -l com.df.servicePath=/demo \
  -l com.df.port=8080 \
  vfarcic/go-demo
```

Please note that we declared the label `com.df.notify`. Only services with this label (it can hold any value) will be eligible to receive notifications through *Docker Flow: Swarm Listener*. We also declared a couple of other labels (`DF_servicePath` and `DF_port`).

Before proceeding, we should wait until all the services are up and running. Please use the `docker service ls` command to check the status.

Please output the *Docker Flow: Swarm Listener* logs. If you're using Docker Machine, the command are as follows.

```bash
docker service logs swarm-listener
```

''' warning
    At the time of this writing, `docker service logs` command is still in experimental stage. It might not work if you used your own cluster without experimental features enabled.

The output is as follows (timestamps are removed for brevity).

```
Starting Docker Flow: Swarm Listener
Starting iterations
Sending service created notification to http://proxy:8080/v1/docker-flow-proxy/reconfigure?port=8080&serviceName=go-demo&servicePath=%2Fdemo
```

As you can see, the listener detected that the `go-demo` service has the label `com.df.notify` and sent the notification request. The address of the notification request is the value of the environment variable `DF_NOTIFY_CREATE_SERVICE_URL` declared in the `swarm-listener` service. The parameters are a combination of the service name and all the labels prefixed with `DF_`.

You might have seen few entries stating that the notification request failed and will be retried. *Docker Flow: Swarm Listener* has a built-in retry mechanism. As long as the output message does not start with `ERROR:`, the notification will reach the destination. Please see the [Environment Variables](#environment-variables) for more info.

Let's see what happens if a service is removed.

```bash
docker service rm go-demo

docker service logs swarm-listener
```

The output of the `docker logs` commands is as follows (timestamps are removed for brevity).

```bash
Starting Docker Flow: Swarm Listener
Starting iterations
Sending service created notification to http://proxy:8080/v1/docker-flow-proxy/reconfigure?port=8080&serviceName=go-demo&servicePath=%2Fdemo
Sending service removed notification to http://proxy:8080/v1/docker-flow-proxy/remove?distribute=true&serviceName=go-demo
```

As you can see, the last output entry was the acknowledgment that the listener detected that the service was removed and that the notification was sent.

## Sending Notification Requests To Multiple Destinations

*Docker Flow Swarm Listener* accepts multiple notification URLs as well. That can come in handy when you want to send notification requests to multiple services at the same time.

We'll start by recreating the `go-demo` service we removed a few moments ago.

```bash
docker service create --name go-demo \
  -e DB=go-demo-db \
  --network proxy \
  -l com.df.notify=true \
  -l com.df.servicePath=/demo \
  -l com.df.port=8080 \
  vfarcic/go-demo
```

The environment variables `DF_NOTIFY_CREATE_SERVICE_URL` and `DF_NOTIFY_REMOVE_SERVICE_URL` allow multiple values separated with comma (*,*). We can, for example, configure the `swarm-listener` service to send notifications both to the *proxy* and the *go-demo* services. Since the `swarm-listener` service is already running, we'll update it with the new values.

```bash
docker service update \
    --env-add DF_NOTIFY_CREATE_SERVICE_URL=http://proxy:8080/v1/docker-flow-proxy/reconfigure,http://go-demo:8080/demo/hello \
    --env-add DF_NOTIFY_REMOVE_SERVICE_URL=http://proxy:8080/v1/docker-flow-proxy/remove,http://go-demo:8080/demo/hello \
    swarm-listener
```

Now we can consult the logs and confirm that the request was sent to both addresses.

```bash
docker service logs swarm-listener
```

The output is as follows (timestamps are removed for brevity).

```bash
Starting Docker Flow: Swarm Listener
Starting iterations
Sending service created notification to http://proxy:8080/v1/docker-flow-proxy/reconfigure?port=8080&serviceName=go-demo&servicePath=%2Fdemo
Sending service created notification to http://go-demo:8080/demo/hello?port=8080&serviceName=go-demo&servicePath=%2Fdemo
```

As you can see, the notification requests were sent both to the `proxy` and `go-demo` addresses.
