pipeline {
  agent {
    label "build"
  }
  stages {
    stage("build-proxy") {
      steps {
        sh 'docker run --rm -v $PWD:/usr/src/myapp -w /usr/src/myapp -v go:/go golang:1.6 bash -c "go get -d -v -t && CGO_ENABLED=0 GOOS=linux go build -v -o docker-flow-swarm-listener"'
        sh 'docker build -t vfarcic/docker-flow-swarm-listener .'
        sh 'docker tag vfarcic/docker-flow-swarm-listener vfarcic/docker-flow-swarm-listener:beta'
        sh "docker login -u ${env.DOCKER_USERNAME} -p ${env.DOCKER_PASSWORD}"
        sh 'docker push vfarcic/docker-flow-swarm-listener:beta'
        sh "docker tag vfarcic/docker-flow-swarm-listener vfarcic/docker-flow-swarm-listener:beta.2.${env.BUILD_NUMBER}"
        sh "docker push vfarcic/docker-flow-swarm-listener:beta.2.${env.BUILD_NUMBER}"
        // sh 'docker push vfarcic/docker-flow-swarm-listener'
        stash name: "stack", includes: "stack.yml"
      }
    }
    stage("build-docs") {
      steps {
        sh 'docker run -t -v $PWD:/docs cilerler/mkdocs bash -c "pip install pygments && pip install pymdown-extensions && mkdocs build"'
        sh 'docker build -t vfarcic/docker-flow-swarm-listener-docs -f Dockerfile.docs .'
        sh "docker tag vfarcic/docker-flow-swarm-listener-docs vfarcic/docker-flow-swarm-listener-docs:beta.2.${env.BUILD_NUMBER}"
        sh "docker push vfarcic/docker-flow-swarm-listener-docs:beta.2.${env.BUILD_NUMBER}"
        // sh 'docker push vfarcic/docker-flow-swarm-listener-docs'
      }
    }
    stage("deploy") {
      agent {
        label "prod"
      }
      steps {
        unstash "stack"
        sh "TAG=beta.2.${env.BUILD_NUMBER} docker stack deploy -c stack.yml swarm-listener"
      }
    }
  }
}
