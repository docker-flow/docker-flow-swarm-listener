# Configuring Docker Flow Swarm Listener

The following environment variables can be used when creating the `swarm-listener` service.

|Name               |Description                                                                    |
|-------------------|-------------------------------------------------------------------------------|
|DF_DOCKER_HOST     |Path to the Docker socket<br>**Default**: `unix:///var/run/docker.sock`            |
|DF_NOTIFY_CREATE_SERVICE_URL|Comma separated list of URLs that will be used to send notification requests when a service is created. If `com.df.notifyService` service labels is present, only URLs related to that service will be used. The `com.df.notifyService` label can have multiple values separated with comma (`,`).<br>**Example**: `url1,url2`|
|DF_NOTIFY_LABEL    |Label that is used to distinguish whether a service should trigger a notification<br>**Default**: `com.df.notify`<br>**Example**: `com.df.notifyDev`|
|DF_NOTIFY_REMOVE_SERVICE_URL|Comma separated list of URLs that will be used to send notification requests when a service is removed.<br>**Example**: `url1,url2`|
|DF_INTERVAL        |Interval (in seconds) between service discovery requests<br>**Default**: `5`<br>**Example**: `10`|
|DF_RETRY           |Number of notification request retries<br>**Default**: `50`<br>**Example**: `100`|
|DF_RETRY_INTERVAL  |Interval (in seconds) between notification request retries<br>**Default**: `5`<br>**Example**: `10`|
