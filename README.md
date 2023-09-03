# Asus Router Prometheus Exporter

# About

Asus-prometheus-exporter uses a asus router's local endpoints to export the router’s metric data (CPU usage, Memory usage, incoming and outgoing traffic) into Prometheus metric format. Setting up a local instance of Prometheus and Grafana, useful insights are provided into the health of your home network.

## Example Grafana Dashboard

![Example Grafana Dashboard](https://github.com/CipherDoc34/asus-prometheus-exporter/blob/412bc2f5fb492b3dc9bbc109b12ff97b2cc75181/Example%20Grafana%20Dashboard.png)

Asus-prometheus-exporter allows for passive collection of metrics without touching your routers configuration or using SNMP. Asus-prometheus-exporter utilizes the same endpoints that the routers homepage uses to display the status of the router and network.

# Capability

Can export:

- CPU usage on each core
- Memory usage
- Incoming and Outgoing Traffic
    - Bridge Traffic
    - Internet Traffic
    - Wired Traffic
    - Wireless Traffic

# Requirements

- Golang
- Asus Router
    - Works with RT-AX82U

# Usage

Raspberry Pi (Raspberry Pi OS)

1. Clone the GitHub Repository: 

```bash
git clone https://github.com/CipherDoc34/asus-prometheus-exporter.git
```

1. Run the `main.go` file in headless mode:  

```bash
nohup go run asus-prometheus-expoter/main.go -uname=<username default:admin> -passwd=<password>&
```

1. Navigate to [localhost:8000/metrics](http://localhost:8000/metrics) to confirm everything is working

## Configuring Prometheus

1. Follow Prometheus installation steps: [https://prometheus.io/docs/prometheus/latest/installation/](https://prometheus.io/docs/prometheus/latest/installation/)
2. Edit `prometheus.yml`

```yaml
global:
  scrape_interval: 15s # Set the scrape interval to every 15 seconds. Default is every 1 minute.
  evaluation_interval: 15s # Evaluate rules every 15 seconds. The default is every 1 minute.
  # scrape_timeout is set to the global default (10s).

# A scrape configuration containing exactly one endpoint to scrape:
scrape_configs:
  - job_name: "asus"

    # metrics_path defaults to '/metrics'
    # scheme defaults to 'http'.
    static_configs:
      - targets: ["localhost:8000"] #Asus-prometheus-exporter endpoint
```

1. Run Prometheus using the prometheus.yml config file:

```yaml
./prometheus --config.file=prometheus.yml &
```

1. Visit [localhost:9090](http://localhost:9090) to confirm that Prometheus is running

## Add Grafana Visualization

1. Follow Grafana’s installation steps: [https://grafana.com/docs/grafana/latest/setup-grafana/installation/](https://grafana.com/docs/grafana/latest/setup-grafana/installation/)
2. Run Grafana Server. 
    
    Linux:
    

```bash
sudo systemctl start grafana-server
```

1. Open [localhost:3000](http://localhost:3000). Login to Grafana (default username and password is admin)
2. Add a new Prometheus Data Source. Configure with Prometheus server URL [http://localhost:9090](http://localhost:9090)
3. Press Build a Dashboard to start configuring
