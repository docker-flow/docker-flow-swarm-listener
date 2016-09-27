## Test

```bash
docker service create --name util-1 \
    -l com.df.notify=true \
    -l com.df.servicePath=/demo \
    alpine sleep 1000000000

docker service create --name util-2 alpine sleep 1000000000

go test --cover

go build -v -o docker-flow-swarm-listener

docker service rm util-1 util-2
```

## Build

```bash
docker run --rm -v $PWD:/usr/src/myapp -w /usr/src/myapp -v go:/go golang:1.7 bash -c "go get -d -v -t && go build -v -o docker-flow-swarm-listener"

docker build -t vfarcic/docker-flow-swarm-listener .

docker tag vfarcic/docker-flow-swarm-listener vfarcic/docker-flow-swarm-listener:0.1

docker push vfarcic/docker-flow-swarm-listener

docker push vfarcic/docker-flow-swarm-listener:0.1
```
