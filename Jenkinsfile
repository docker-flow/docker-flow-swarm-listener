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
        dfBuild("docker-flow-swarm-listener")
        sh "docker-compose run --rm tests"
      }
    }
    stage("release") {
      when {
        branch "master"
      }
      steps {
        dfRelease("docker-flow-swarm-listener")
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
        dfDeploy("swarm-listener_swarm-listener", "swarm-listener_docs")
        dfDeploy("monitor_swarm-listener", "")
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
