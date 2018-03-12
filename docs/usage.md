# Usage

## Notification Format

*Docker Flow Swarm Listener*, sends GET notifcations to configured URLs when a service or node is created, updated or removed. Please consult the [configuration](config.md) page on how to configure the URLs.

### Service Notification

When a service is created or updated a notification will be sent to **[DF_NOTIFY_CREATE_SERVICE_URL]** with the following parameters:

| Query       | Description                                                            | Example |
|-------------|------------------------------------------------------------------------|---------|
| serviceName | Name of service. If `com.df.shortName` is true, and the service is part of a stack the stack name will be trimed off. | `go-demo` |
| replicas    | Number of replicas of service. If the service is global, this parameter will be excluded.| `3` |
| nodeInfo    | An array of node with its ip on an overlay network. The network is defined with the label: `com.df.scrapeNetwork`. This parameter is included when environment variable, `DF_INCLUDE_NODE_IP_INFO`, is true. | `[["node-3","10.0.0.23"], ["node-2", "10.0.0.22"]]` |

All service labels prefixed by `com.df.` will be added to the notification. For example, a service with label `com.df.hello=world` will translate to parameter: `hello=world`.

When a service is removed, a notification will be sent to **[DF_NOTIFY_REMOVE_SERVICE_URL]**. Only the `serviceName` parameter is included.

### Node Notification

When a node is created or updated a notification will be sent to **[DF_NOTIFY_CREATE_NODE_UR]** with the following parameters:

| Query | Description | Example |
|-------|-------------|---------|
| id    | The ID of node given by docker | `2pe2xpkrx780xrhujws42a73w` |
| hostname | Hostname of node | `ap1.hostname.com` |
| address  | Address of node | `10.0.0.1` |
| versionIndex | The version index of node | `24` |
| state | State of node. [`unknown`, `down`, `ready`, `disconnected`] | `down` |
| role | Role of node. [`worker`, `manager`] | `worker` |
| availability | Availability of node. [`active`, `pause`, `drain` ]| `active` |

All service labels prefixed by `com.df.` will be added to the notification. For example, a node with label `com.df.hello=world` will translate to parameter: `hello=world`.

When a node is removed, a notification will be sent to **[DF_NOTIFY_REMOVE_NODE_URl]**. Only the `id`, `hostname`, and `address` parameters are included.

## API

*Docker Flow Swarm Listener* exposes a API to query series and to send notifications.

### Get Services

The *Get Services* endpoint is used to query all running services with the `DF_NOTIFY_LABEL` label. A `GET` request to **[SWARM_IP]:[SWARM_PORT]/v1/docker-flow-swarm-listener/get-services** returns a json representation of these services.

### Notify Services

*DFSL* normally sends out notifcations when a service is created, updated, or removed. The *Notify Services* endpoint will force *DFSL* to send out notifications for all running services with the `DF_NOTIFY_LABEL` label. A `GET` request to **[SWARM_IP]:[SWARM_PORT]/v1/docker-flow-swarm-listener/notify-services** sends out the notifications.
