# Feedback and Contribution

The *Docker Flow Swarm Listener* project welcomes, and depends, on contributions from developers and users in the open source community. Contributions can be made in a number of ways, a few examples are:

* Code patches or new features via pull requests
* Documentation improvements
* Bug reports and patch reviews

## Reporting an Issue

Feel fee to [create a new issue](https://github.com/docker-flow/docker-flow-swarm-listener/issues). Include as much detail as you can.

If an issue is a bug, please provide steps to reproduce it.

If an issue is a request for a new feature, please specify the use-case behind it.

## Discussion

Please join the [DevOps20](http://slack.devops20toolkit.com/) Slack channel if you'd like to discuss the project or have a problem you'd like us to solve.

## Contributing To The Project

I encourage you to contribute to the *Docker Flow Swarm Listener* project.

The project is developed using *Test Driven Development* and *Continuous Deployment* process. Test are divided into unit and integration tests. Every code file has an equivalent with tests (e.g. `reconfigure.go` and `reconfigure_test.go`). Ideally, I expect you to write a test that defines that should be developed, run all the unit tests and confirm that the test fails, write just enough code to make the test pass, repeat. If you are new to testing, feel free to create a pull request indicating that tests are missing and I'll help you out.

Once you are finish implementing a new feature or fixing a bug, run the *Complete Cycle*. You'll find the instructions below.

### Repository

Fork [docker-flow-swarm-listener](https://github.com/docker-flow/docker-flow-swarm-listener).

### Unit Testing

```bash
go get -d -v -t

go test ./... -cover -run UnitTest
```

### Building

```bash
export DOCKER_HUB_USER=[...] # Change to your user in hub.docker.com

docker image build -t $DOCKER_HUB_USER/docker-flow-swarm-listener:beta .

docker image push $DOCKER_HUB_USER/docker-flow-swarm-listener:beta
```

### Pull Request

Once the feature is done, create a pull request.
