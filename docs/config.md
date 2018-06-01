# Configuring Docker Flow Swarm Listener

The following environment variables can be used when creating the `swarm-listener` service.

|Name               |Description                                                                    |
|-------------------|-------------------------------------------------------------------------------|
|DF_DOCKER_HOST     |Path to the Docker socket<br>**Default**: `unix:///var/run/docker.sock`            |
|DF_NOTIFY_LABEL    |Label that is used to distinguish whether a service should trigger a notification<br>**Default**: `com.df.notify`<br>**Example**: `com.df.notifyDev`|
|DF_NOTIFY_CREATE_SERVICE_URL|Comma separated list of URLs that will be used to send notification requests when a service is created. If `com.df.notifyService` service labels is present, only URLs related to that service will be used. The `com.df.notifyService` label can have multiple values separated with comma (`,`).<br>**Example**: `url1,url2`|
|DF_NOTIFY_REMOVE_SERVICE_URL|Comma separated list of URLs that will be used to send notification requests when a service is removed.<br>**Example**: `url1,url2`|
|DF_INCLUDE_NODE_IP_INFO|Include node and ip information for service in notification.<br>**Default**:`false`|
|DF_NOTIFY_CREATE_NODE_URL |Comma separated list of URLs that will be used to send notification requests when a node is created or updated.<br>**Example**: `url1,url2`|
|DF_NOTIFY_REMOVE_NODE_URL |Comma separated list of URLs that will be used to send notification requests when a node is remove.<br>**Example**: `url1,url2`|
|DF_RETRY           |Number of notification request retries<br>**Default**: `50`<br>**Example**: `100`|
|DF_RETRY_INTERVAL  |Time between each notificationo request retry, in seconds.<br>**Default**: `5`<br>**Example**:`10`|
|DF_SERVICE_POLLING_INTERVAL |Time between each service polling request, in seconds. When this value is set less than or equal to zero, service polling is disabled.<br>**Default**: `-1`<br>**Example**:`20`|
|DF_USE_DOCKER_SERVICE_EVENTS|Use docker events api to get service updates.<br>**Default**:`true`|
|DF_NODE_POLLING_INTERVAL |Time between each node polling request, in seconds. When this value is set less than or equal to zero, node polling is disabled.<br>**Default**: `-1`<br>**Example**:`20`|
|DF_USE_DOCKER_NODE_EVENTS|Use docker events api to get node updates.<br>**Default**:`true`|
