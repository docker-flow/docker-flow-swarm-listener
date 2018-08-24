import java.text.SimpleDateFormat

pipeline {
  agent {
    label "test"
  }
  options {
    buildDiscarder(logRotator(numToKeepStr: '2'))
    disableConcurrentBuilds()
  }
  stages {
    stage("build") {
      steps {
        script {
          def dateFormat = new SimpleDateFormat("yy.MM.dd")
          currentBuild.displayName = dateFormat.format(new Date()) + "-" + env.BUILD_NUMBER
        }
        dfBuild2("docker-flow-swarm-listener")
        sh "docker-compose run --rm tests"
      }
    }
    stage("release") {
      when {
        branch "master"
      }
      steps {
        dfRelease2("docker-flow-swarm-listener")
        dfReleaseGithub2("docker-flow-swarm-listener")
      }
    }
    stage("deploy") {
      when {
        branch "master"
      }
      agent {
        label "prod"
      }
      steps {
        sh "helm upgrade -i docker-flow-swarm-listener helm/docker-flow-swarm-listener --namespace df --set image.tag=${currentBuild.displayName}"
      }
    }
  }
  post {
    always {
      sh "docker system prune -f"
    }
    failure {
      slackSend(
        color: "danger",
        message: "${env.JOB_NAME} failed: ${env.RUN_DISPLAY_URL}"
      )
    }
  }
}
