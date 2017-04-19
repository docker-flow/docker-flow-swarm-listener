pipeline {
  agent {
    label "build"
  }
  stages {
    stage("Unit") {
      steps {
        git "https://github.com/vfarcic/docker-flow-swarm-listener.git"
        sh "docker version"
      }
    }
}