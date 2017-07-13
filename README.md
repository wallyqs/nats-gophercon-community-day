### NATS Technologies at GopherCon (Community Day Workshop)

Repository with dependencies for the NATS Community Day at GopherCon.

#### Required Dependencies

- [Vagrant](https://www.vagrantup.com/downloads.html) (1.9.7):

- [VirtualBox](https://www.virtualbox.org/wiki/Downloads) (Version 5.1.22)
  
#### Setup

Clone the repo and run:

```
git clone https://github.com/wallyqs/nats-gophercon-community-day
cd nats-gophercon-community-day
vagrant up
vagrant provision
```

This should provide a VM with:

- NATS (v1.0.0)
- Golang (1.8.3)
- Docker (17.06.0-ce)
- nats-top
- Prometheus
- Prometheus NATS Exporter
