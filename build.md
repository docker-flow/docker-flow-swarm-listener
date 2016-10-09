## Version

```bash
VERSION=0.2
```

## Automated Test

```bash
docker service create --name util-1 \
    -l com.df.notify=true \
    -l com.df.servicePath=/demo \
    alpine sleep 1000000000

docker service create --name util-2 alpine sleep 1000000000

go test --cover

docker service rm util-1 util-2
```

## Build

```bash
docker run --rm -v $PWD:/usr/src/myapp -w /usr/src/myapp -v go:/go golang:1.7 bash -c "go get -d -v -t && go build -v -o docker-flow-swarm-listener"

docker build -t vfarcic/docker-flow-swarm-listener .

docker tag vfarcic/docker-flow-swarm-listener vfarcic/docker-flow-swarm-listener:$VERSION
```

## Manual Tests

```bash
docker network create --driver overlay proxy

docker service create --name proxy \
    -p 80:80 \
    -p 443:443 \
    -p 8080:8080 \
    --network proxy \
    -e MODE=swarm \
    vfarcic/docker-flow-proxy

docker service create --name swarm-listener \
    --network proxy \
    --mount "type=bind,source=/var/run/docker.sock,target=/var/run/docker.sock" \
    -e DF_NOTIF_CREATE_SERVICE_URL=http://proxy:8080/v1/docker-flow-proxy/reconfigure \
    -e DF_NOTIF_REMOVE_SERVICE_URL=http://proxy:8080/v1/docker-flow-proxy/remove \
    vfarcic/docker-flow-swarm-listener:$VERSION

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

## Publish

```bash
docker push vfarcic/docker-flow-swarm-listener

docker push vfarcic/docker-flow-swarm-listener:$VERSION
```
