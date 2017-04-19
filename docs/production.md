# Production

## Deploy

```bash
docker network create \
    --driver overlay \
    proxy

curl -o swarm-listener.yml \
    https://raw.githubusercontent.com/vfarcic/docker-flow-swarm-listener/master/stack.yml

docker stack deploy \
    -c swarm-listener.yml \
    swarm-listener
```