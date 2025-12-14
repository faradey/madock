# Docker Images

This document lists the Docker images used by Madock.

## Core Services

### Nginx
* [nginx](https://hub.docker.com/_/nginx) (version 1.26)

### PHP-FPM
* Created automatically from Dockerfile with customizable PHP version (7.1 - 8.4)

### NodeJS
* [node](https://hub.docker.com/_/node)

## Database

### MySQL / MariaDB
* [mariadb](https://hub.docker.com/_/mariadb)
* [mysql](https://hub.docker.com/_/mysql)

### phpMyAdmin
* [phpmyadmin](https://hub.docker.com/_/phpmyadmin)

## Search Engines

### Elasticsearch
* [elasticsearch](https://hub.docker.com/_/elasticsearch)

### OpenSearch
* [opensearchproject/opensearch](https://hub.docker.com/r/opensearchproject/opensearch)

### OpenSearch Dashboards
* [opensearchproject/opensearch-dashboards](https://hub.docker.com/r/opensearchproject/opensearch-dashboards)

## Caching

### Redis
* [redis](https://hub.docker.com/_/redis)

### Valkey
* [valkey/valkey](https://hub.docker.com/r/valkey/valkey)

### Varnish
* [varnish](https://hub.docker.com/_/varnish)

## Message Queue

### RabbitMQ
* [rabbitmq](https://hub.docker.com/_/rabbitmq)

## Monitoring & Logging

### Grafana Stack

Grafana provides a comprehensive monitoring solution with pre-configured dashboards:

**Images:**
* [grafana/grafana](https://hub.docker.com/r/grafana/grafana) - Visualization platform
* [grafana/loki](https://hub.docker.com/r/grafana/loki) - Log aggregation
* [grafana/promtail](https://hub.docker.com/r/grafana/promtail) - Log collector
* [prom/prometheus](https://hub.docker.com/r/prom/prometheus) - Metrics collection
* [prom/mysqld-exporter](https://hub.docker.com/r/prom/mysqld-exporter) - MySQL metrics exporter
* [kbudde/rabbitmq-exporter](https://hub.docker.com/r/kbudde/rabbitmq-exporter) - RabbitMQ metrics exporter

**Pre-configured Dashboards:**
* **Loki** - Application logs viewer
* **MySQL Overview** - Database performance metrics (connections, queries, buffer pool)
* **Redis** - Cache performance and memory usage
* **RabbitMQ** - Queue metrics, connections, channels, message rates

**Access:** `https://your-domain.test/grafana/`

**Enable:** `madock service:enable grafana`

### Kibana
* [kibana](https://hub.docker.com/_/kibana)

## Email Testing

### Mailpit
* [axllent/mailpit](https://hub.docker.com/r/axllent/mailpit)

## Testing

### Selenium
* [selenium/standalone-chrome](https://hub.docker.com/r/selenium/standalone-chrome) (for MFTF)

