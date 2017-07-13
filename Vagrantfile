Vagrant.configure("2") do |config|
  config.vm.box = "ubuntu/zesty64"

  # Expose same ports in case may want to access
  # from the host.
  config.vm.network "forwarded_port", guest: 4222, host: 4222
  config.vm.network "forwarded_port", guest: 8222, host: 8222
  config.vm.network "forwarded_port", guest: 9411, host: 9411

  GO_VERSION = "1.8.3"

  plan = <<EXEC
source /home/ubuntu/.bashrc
echo "Starting..."

# Install latest Go...
[[ ! -d /usr/local/go ]] && {
    echo "Installing Go #{GO_VERSION}..."
    wget https://storage.googleapis.com/golang/go#{GO_VERSION}.linux-amd64.tar.gz -O /usr/local/go.tar.gz 2> /dev/null
    cd /usr/local && tar xvf go.tar.gz
    echo "export PATH=\\$PATH:/usr/local/go/bin" >> /home/ubuntu/.bashrc
}

# Installing Docker...
[[ ! -f /usr/bin/docker ]] && {
    echo "Installing Docker CE..."
    apt-get update
    apt-get -y install \
        apt-transport-https \
        ca-certificates \
        curl \
        software-properties-common
     curl -fsSL https://download.docker.com/linux/ubuntu/gpg | apt-key add -

    add-apt-repository \
       "deb [arch=amd64] https://download.docker.com/linux/ubuntu \
       $(lsb_release -cs) \
       stable"
    apt-get update
    apt-get -y install docker-ce
}

# Pulling required images..
docker pull nats
docker pull prom/prometheus
docker pull grafana/grafana
docker pull openzipkin/zipkin

# Install nats-top
[[ ! -f /usr/local/bin/nats-top ]] && {
    echo "Installing nats-top..."
    apt-get install unzip
    mkdir -p /tmp/example
    curl -O -L -f https://github.com/nats-io/nats-top/releases/download/v0.3.2/nats-top-linux_amd64.zip 2> /dev/null
    unzip nats-top-linux_amd64.zip
    cp nats-top-linux_amd64/nats-top  /usr/local/bin/nats-top
}
EXEC

  user_plan = <<PLAN
    /usr/local/go/bin/go get github.com/nats-io/go-nats
    /usr/local/go/bin/go get github.com/nats-io/gnatsd
    /usr/local/go/bin/go get github.com/nats-io/prometheus-nats-exporter
    /usr/local/go/bin/go get github.com/minio/minio
PLAN

  config.vm.provision "shell", inline: plan
  config.vm.provision "shell", inline: user_plan, privileged: false
end
