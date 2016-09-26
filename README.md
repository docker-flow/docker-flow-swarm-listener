# Docker Flow: Swarm Listener

The project listens to Docker Swarm events and sends requests when a change occurs. At the moment, the only supported option is to send a notification when a new service is created. Wide feature support is coming soon.

## Example

The example that follows will use the *Swarm Listener* to reconfigure the [Docker Flow: Proxy](https://github.com/vfarcic/docker-flow-proxy) whenever a new service is created.

```bash
git clone
```

## Environment Variables

* DF_DOCKER_HOST
* DF_NOTIFICATION_URL
* DF_INTERVAL
* DF_RETRY
* DF_RETRY_INTERVAL

## Test

```bash
docker swarm init

docker service create --name util-1 \
    -l DF_NOTIFY=true \
    -l DF_servicePath=/demo \
    alpine sleep 1000000000

docker service create --name util-2 alpine sleep 1000000000

go test --cover

go build -v -o docker-flow-swarm-listener

DF_INTERVAL=1 DF_NOTIFICATION_URL=http://localhost ./docker-flow-swarm-listener

docker service rm util
```

## Build

```bash
docker run --rm -v $PWD:/usr/src/myapp -w /usr/src/myapp -v go:/go golang:1.7 bash -c "go get -d -v -t && go build -v -o docker-flow-swarm-listener"

docker build -t vfarcic/docker-flow-swarm-listener .
```

## TODO

- [ ] Write README
