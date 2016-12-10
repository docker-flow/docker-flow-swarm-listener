## Automated Test

```bash
go test --cover
```

## Build

```bash
docker run --rm -v $PWD:/usr/src/myapp -w /usr/src/myapp -v go:/go golang:1.7-alpine sh -c "go get -d -v -t && go build -v -o docker-flow-swarm-listener"

docker build -t vfarcic/docker-flow-swarm-listener:latest .
```
## Publish

```bash
VERSION=0.6

docker tag vfarcic/docker-flow-swarm-listener:latest vfarcic/docker-flow-swarm-listener:$VERSION

docker push vfarcic/docker-flow-swarm-listener:$VERSION

docker push vfarcic/docker-flow-swarm-listener:latest
```

## Manual Tests

```bash
docker-machine create -d virtualbox test

eval $(docker-machine env test)

docker swarm init --advertise-addr $(docker-machine ip test)

docker run --rm -v $PWD:/usr/src/myapp -w /usr/src/myapp -v go:/go golang:1.7 bash -c "go get -d -v -t && go build -v -o docker-flow-swarm-listener"

docker build -t vfarcic/docker-flow-swarm-listener:beta .

docker network create --driver overlay proxy

docker service create --name swarm-listener \
    --network proxy \
    --mount "type=bind,source=/var/run/docker.sock,target=/var/run/docker.sock" \
    -e DF_NOTIF_CREATE_SERVICE_URL=http://proxy:8080/v1/docker-flow-proxy/reconfigure \
    -e DF_NOTIF_REMOVE_SERVICE_URL=http://proxy:8080/v1/docker-flow-proxy/remove \
    vfarcic/docker-flow-swarm-listener:beta

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

docker service create --name util \
    --network proxy \
    alpine sleep 1000000

docker service create --name proxy \
    -p 80:80 \
    -p 443:443 \
    -p 8080:8080 \
    --network proxy \
    -e MODE=swarm \
    vfarcic/docker-flow-proxy

docker service ls

PROXY_ID=$(docker ps -q --filter "ancestor=vfarcic/docker-flow-proxy")

docker exec -it $PROXY_ID cat /cfg/haproxy.cfg

UTIL_ID=$(docker ps -q --filter "ancestor=alpine")

docker exec -it $UTIL_ID apk add --update drill

docker exec -it $UTIL_ID apk add --update curl

docker exec -it $UTIL_ID drill swarm-listener

docker exec -it $UTIL_ID curl swarm-listener:8080/v1/docker-flow-swarm-listener/notify-services

docker exec -it $PROXY_ID cat /cfg/haproxy.cfg

DFSL_ID=$(docker ps -q -f ancestor=vfarcic/docker-flow-swarm-listener)

docker logs $DFSL_ID

DFP_ID=$(docker ps -q -f ancestor=vfarcic/docker-flow-proxy)

docker exec -it $DFP_ID cat /cfg/haproxy.cfg

docker service rm go-demo

docker logs $DFSL_ID

docker exec -it $DFP_ID cat /cfg/haproxy.cfg

docker service create --name go-demo \
  -e DB=go-demo-db \
  --network proxy \
  -l com.df.notify=true \
  -l com.df.servicePath=/demo \
  -l com.df.port=8080 \
  vfarcic/go-demo

docker logs $DFSL_ID

docker exec -it $DFP_ID cat /cfg/haproxy.cfg

docker service rm proxy swarm-listener go-demo go-demo-db

docker network rm proxy
```
