import java.text.SimpleDateFormat

pipeline {
  options {
    buildDiscarder(logRotator(numToKeepStr: '2'))
    disableConcurrentBuilds()
  }
  agent {
    kubernetes {
      cloud "kubernetes"
      label "kubernetes"
      serviceAccount "jenkins"
      yaml """
apiVersion: v1
kind: Pod
spec:
  containers:
  - name: docker
    image: docker:18.06
    command: ["cat"]
    tty: true
    volumeMounts:
    - mountPath: /var/run/docker.sock
      name: docker-socket
  volumes:
  - name: docker-socket
    hostPath:
      path: /var/run/docker.sock
      type: Socket
"""
    }      
  }
  stages {
    stage("build") {
      steps {
        container("docker") {
          script {
            def dateFormat = new SimpleDateFormat("yy.MM.dd")
            currentBuild.displayName = dateFormat.format(new Date()) + "-" + env.BUILD_NUMBER
          }
          git branch: "k8s", url: "https://github.com/docker-flow/docker-flow-swarm-listener.git" // REMOVE ME!
          k8sBuildGolang("docker-flow-swarm-listener")
          dfBuild2("docker-flow-swarm-listener")
        // sh "docker-compose run --rm tests"
        }
      }
    }
/*    stage("release") {
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
        dfDeploy2("docker-flow-swarm-listener", "swarm-listener_swarm-listener", "swarm-listener_docs")
        dfDeploy2("docker-flow-swarm-listener", "monitor_swarm-listener", "")
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
    }*/
  }
}
