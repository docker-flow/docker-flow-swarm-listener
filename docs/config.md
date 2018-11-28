# Configuring Docker Flow Swarm Listener

The following environment variables can be used when creating the `swarm-listener` service.

|Name               |Description                                                                    |
|-------------------|-------------------------------------------------------------------------------|
|DF_DOCKER_HOST     |Path to the Docker socket<br>**Default**: `unix:///var/run/docker.sock`            |
|DF_NOTIFY_LABEL    |Label that is used to distinguish whether a service should trigger a notification<br>**Default**: `com.df.notify`<br>**Example**: `com.df.notifyDev`|
|DF_NOTIFY_CREATE_SERVICE_URL|Comma separated list of URLs that will be used to send notification requests when a service is created. If `com.df.notifyService` service labels is present, only URLs related to that service will be used. The `com.df.notifyService` label can have multiple values separated with comma (`,`).<br>**Example**: `url1,url2`|
|DF_NOTIFY_REMOVE_SERVICE_URL|Comma separated list of URLs that will be used to send notification requests when a service is removed.<br>**Example**: `url1,url2`|
|DF_NOTIFY_CREATE_SERVICE_METHOD|Comma separated list of HTTP methods used to send requests to its corresponding `DF_NOTIFY_CREATE_SERVICE_URL`. If the number of comma separated list of HTTP methods is less than the number of create service URLs, then the last HTTP method in the list will be used for the rest of the services.<br>**Default**: `GET` <br>**Example**: `GET,POST`|
|DF_NOTIFY_REMOVE_SERVICE_METHOD|Comma separated list of HTTP methods used to send requests to its corresponding `DF_NOTIFY_REMOVE_SERVICE_URL`. If the number of comma separated list of HTTP methods is less than the number of remove service URLs, then the last HTTP method in the list will be used for the rest of the services<br>**Default**: `GET` <br>**Example**: `GET,POST`|
|DF_INCLUDE_NODE_IP_INFO|Include node and ip information for service in notification.<br>**Default**:`false`|
|DF_NODE_IP_INFO_INCLUDES_TASK_ADDRESS|Include task ip address when `DF_INCLUDE_NODE_IP_INFO` is true.<br>**Default**: `true`|
|DF_NOTIFY_CREATE_NODE_URL |Comma separated list of URLs that will be used to send notification requests when a node is created or updated.<br>**Example**: `url1,url2`|
|DF_NOTIFY_REMOVE_NODE_URL |Comma separated list of URLs that will be used to send notification requests when a node is remove.<br>**Example**: `url1,url2`|
|DF_RETRY           |Number of notification request retries<br>**Default**: `50`<br>**Example**: `100`|
|DF_RETRY_INTERVAL  |Time between each notificationo request retry, in seconds.<br>**Default**: `5`<br>**Example**:`10`|
|DF_SERVICE_POLLING_INTERVAL |Time between each service polling request, in seconds. When this value is set less than or equal to zero, service polling is disabled.<br>**Default**: `-1`<br>**Example**:`20`|
|DF_USE_DOCKER_SERVICE_EVENTS|Use docker events api to get service updates.<br>**Default**:`true`|
|DF_NODE_POLLING_INTERVAL |Time between each node polling request, in seconds. When this value is set less than or equal to zero, node polling is disabled.<br>**Default**: `-1`<br>**Example**:`20`|
|DF_USE_DOCKER_NODE_EVENTS|Use docker events api to get node updates.<br>**Default**:`true`|
|DF_SERVICE_NAME_PREFIX|Value to prefix service names with.<br>**Example**:`dev1`|
|DF_NOTIFY_CREATE_SERVICE_IMMEDIATELY|Sends create service without waiting for service to converge. After the service converges, another create notifcation will be sent out.<br>**Default**: `false`|

## Configuring Notification URLS with Docker Secrets

*Docker Flow Swarm Listener*'s notification URLs can be set with Docker Secrets. Secrets with names `df_notify_create_service_url`,
`df_notify_remove_service_url`, `df_notify_create_node_url`, and `df_notify_remove_node_url` are used, in addition to their
corresponding environment variables, to configure notification urls. The secrets must be a comma separated list of URLs.
