# Configuring Docker Flow Swarm Listener

The following environment variables can be used when creating the `swarm-listener` service.

|Name               |Description                                               |Default Value|Example|
|-------------------|----------------------------------------------------------|-------------|-------|
|DF_DOCKER_HOST     |Path to the Docker socket                   |unix:///var/run/docker.sock|       |
|DF_NOTIFY_CREATE_SERVICE_URL|Comma separated list of URLs that will be used to send notification requests when a service is created.|url1,url2|
|DF_NOTIFY_REMOVE_SERVICE_URL|Comma separated list of URLs that will be used to send notification requests when a service is removed.|url1,url2|
|DF_INTERVAL        |Interval (in seconds) between service discovery requests  |5            |10     |
|DF_RETRY           |Number of notification request retries                    |50           |100    |
|DF_RETRY_INTERVAL  |Interval (in seconds) between notification request retries|5            |10     |
