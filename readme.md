# Stream Telecom Balance exporter

The Stream Telecom balance exporter for [prometheus](https://prometheus.io) allows exporting balance for [Stream Telecom gateway](https://stream-telecom.ru)

- [How it works](#how-it-works)
- [Configuration](#configuration)
- [Command-line flags](#command-line-flags)
- [Running](#running)
  * [Running with docker](#running-with-docker)
  * [Running with docker-compose](#running-with-docker-compose)
  * [Running with systemctl](#running-with-systemctl)

## How it works
Exporter querying balance every hour (by default) and store it value in memory.
When prometheus make request, exporter retrieve balance value from memory for make response.

## Configuration
You must set environment variables:

* `STREAM_TELECOM_LOGIN` - your login
* `STREAM_TELECOM_PASSWORD` - your password

## Command-line flags

* `listen-address` - the address to listen on for HTTP requests. (Default: `0.0.0.0:9602`)
* `interval` - the interval (in seconds) for querying balance. (Default: `3600`)
* `retry-interval` - the interval (in seconds) for load balance when errors. (Default: `10`)
* `retry-limit` - the count of tries when error. (Default: `10`)

## Running
### Running with docker

```sh
docker run \
    -e STREAM_TELECOM_LOGIN=<your-login> \
    -e STREAM_TELECOM_PASSWORD=<your-password> \
    -p 9601:9601 \
    --restart=unless-stopped \
    --name stream-telecom-balance-exporter \
    -d \
    xxxcoltxxx/stream-telecom-balance-exporter
```

### Running with docker-compose

Create configuration file. For example, file named `docker-compose.yaml`:

```yaml
version: "2.1"

services:
  stream-telecom-balance-exporter:
    image: xxxcoltxxx/stream-telecom-balance-exporter
    restart: unless-stopped
    environment:
      STREAM_TELECOM_LOGIN: <your-login>
      STREAM_TELECOM_PASSWORD: <your-password>
    ports:
      - 9601:9601
```

Run exporter:
```sh
docker-compose up -d
```

Show service logs:
```sh
docker-compose logs -f stream-telecom-balance-exporter
```

### Running with systemctl

Set variables you need:
```sh
STREAM_TELECOM_EXPORTER_VERSION=v0.1.0-beta.1
STREAM_TELECOM_EXPORTER_PLATFORM=linux
STREAM_TELECOM_EXPORTER_ARCH=amd64
STREAM_TELECOM_LOGIN=<your_login>
STREAM_TELECOM_PASSWORD=<your_password>
```

Download release:
```sh
wget https://github.com/xxxcoltxxx/stream-telecom-balance-exporter/releases/download/${STREAM_TELECOM_EXPORTER_VERSION}/stream_telecom_balance_exporter_${STREAM_TELECOM_EXPORTER_VERSION}_${STREAM_TELECOM_EXPORTER_PLATFORM}_${STREAM_TELECOM_EXPORTER_ARCH}.tar.gz
tar xvzf stream_telecom_balance_exporter_${STREAM_TELECOM_EXPORTER_VERSION}_${STREAM_TELECOM_EXPORTER_PLATFORM}_${STREAM_TELECOM_EXPORTER_ARCH}.tar.gz
mv ./stream_telecom_balance_exporter_${STREAM_TELECOM_EXPORTER_VERSION}_${STREAM_TELECOM_EXPORTER_PLATFORM}_${STREAM_TELECOM_EXPORTER_ARCH} /usr/local/bin/stream_telecom_balance_exporter
```

Add service to systemctl. For example, file named `/etc/systemd/system/stream_telecom_balance_exporter.service`:
```sh
[Unit]
Description=Stream Telecom Balance Exporter
Wants=network-online.target
After=network-online.target

[Service]
Environment="STREAM_TELECOM_LOGIN=${STREAM_TELECOM_LOGIN}"
Environment="STREAM_TELECOM_PASSWORD=${STREAM_TELECOM_PASSWORD}"
Type=simple
ExecStart=/usr/local/bin/stream_telecom_balance_exporter

[Install]
WantedBy=multi-user.target
```

Reload systemctl configuration and restart service
```sh
systemctl daemon-reload
systemctl restart stream_telecom_balance_exporter
```

Show service status:
```sh
systemctl status stream_telecom_balance_exporter
```

Show service logs:
```sh
journalctl -fu stream_telecom_balance_exporter
```
