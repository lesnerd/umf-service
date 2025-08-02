# UMF Service Helm Chart

A Helm chart for deploying the Unified Fabric Manager (UMF) Service on Kubernetes.

## Prerequisites

- Kubernetes 1.16+
- Helm 3.0+

## Installing the Chart

To install the chart with the release name `my-ufm-service`:

```bash
helm install my-ufm-service ./helm/ufm-service
```

## Uninstalling the Chart

To uninstall/delete the `my-ufm-service` deployment:

```bash
helm delete my-ufm-service
```

## Configuration

The following table lists the configurable parameters of the UMF Service chart and their default values.

### Common Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `replicaCount` | Number of replicas | `1` |
| `image.repository` | Image repository | `ufm-service` |
| `image.pullPolicy` | Image pull policy | `IfNotPresent` |
| `image.tag` | Image tag | `""` (uses appVersion) |
| `nameOverride` | Override name of the chart | `""` |
| `fullnameOverride` | Override fullname of the chart | `""` |

### Service Account

| Parameter | Description | Default |
|-----------|-------------|---------|
| `serviceAccount.create` | Create service account | `true` |
| `serviceAccount.annotations` | Service account annotations | `{}` |
| `serviceAccount.name` | Service account name | `""` |

### Service

| Parameter | Description | Default |
|-----------|-------------|---------|
| `service.type` | Service type | `ClusterIP` |
| `service.port` | Service port | `8080` |
| `service.targetPort` | Target port | `8080` |

### Ingress

| Parameter | Description | Default |
|-----------|-------------|---------|
| `ingress.enabled` | Enable ingress | `false` |
| `ingress.className` | Ingress class name | `""` |
| `ingress.annotations` | Ingress annotations | `{}` |
| `ingress.hosts` | Ingress hosts configuration | `[{host: "ufm-service.local", paths: [{path: "/", pathType: "Prefix"}]}]` |
| `ingress.tls` | Ingress TLS configuration | `[]` |

### UMF Service Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `config.database.enabled` | Enable database | `true` |
| `config.database.url` | Database URL | `postgres://umf_user:umf_password@postgres:5432/umf_db?sslmode=disable` |
| `config.server.port` | Server port | `8080` |
| `config.telemetry.enabled` | Enable telemetry | `true` |
| `config.monitoring.metrics.enabled` | Enable metrics | `true` |
| `config.monitoring.metrics.port` | Metrics port | `8080` |
| `config.monitoring.metrics.path` | Metrics path | `/metrics` |
| `config.monitoring.tracing.enabled` | Enable tracing | `false` |
| `config.monitoring.tracing.jaeger.endpoint` | Jaeger endpoint | `""` |

### Health Checks

| Parameter | Description | Default |
|-----------|-------------|---------|
| `healthcheck.enabled` | Enable health checks | `true` |
| `healthcheck.path` | Health check path | `/api/v1/system/ping` |
| `healthcheck.initialDelaySeconds` | Initial delay | `30` |
| `healthcheck.periodSeconds` | Check period | `30` |
| `healthcheck.timeoutSeconds` | Check timeout | `10` |
| `healthcheck.failureThreshold` | Failure threshold | `3` |

### Autoscaling

| Parameter | Description | Default |
|-----------|-------------|---------|
| `autoscaling.enabled` | Enable HPA | `false` |
| `autoscaling.minReplicas` | Minimum replicas | `1` |
| `autoscaling.maxReplicas` | Maximum replicas | `100` |
| `autoscaling.targetCPUUtilizationPercentage` | Target CPU utilization | `80` |

### Resources

| Parameter | Description | Default |
|-----------|-------------|---------|
| `resources` | Resource limits and requests | `{}` |
| `nodeSelector` | Node selector | `{}` |
| `tolerations` | Tolerations | `[]` |
| `affinity` | Affinity rules | `{}` |

## Examples

### Basic Installation

```bash
helm install ufm-service ./helm/ufm-service
```

### With Custom Values

```bash
helm install ufm-service ./helm/ufm-service \
  --set replicaCount=3 \
  --set image.tag=v1.2.3 \
  --set config.database.url="postgres://user:pass@db:5432/ufm"
```

### With Ingress

```bash
helm install ufm-service ./helm/ufm-service \
  --set ingress.enabled=true \
  --set ingress.hosts[0].host=ufm.example.com \
  --set ingress.hosts[0].paths[0].path=/
```

### With Resource Limits

```bash
helm install ufm-service ./helm/ufm-service \
  --set resources.limits.cpu=500m \
  --set resources.limits.memory=512Mi \
  --set resources.requests.cpu=250m \
  --set resources.requests.memory=256Mi
```

### Using Values File

Create a `custom-values.yaml` file:

```yaml
replicaCount: 2

image:
  tag: "v1.2.3"

config:
  database:
    url: "postgres://production-user:secure-pass@prod-db:5432/ufm_production"
  
resources:
  limits:
    cpu: 1000m
    memory: 1Gi
  requests:
    cpu: 500m
    memory: 512Mi

ingress:
  enabled: true
  hosts:
    - host: ufm.production.com
      paths:
        - path: /
          pathType: Prefix

autoscaling:
  enabled: true
  minReplicas: 2
  maxReplicas: 10
  targetCPUUtilizationPercentage: 70
```

Then install:

```bash
helm install ufm-service ./helm/ufm-service -f custom-values.yaml
```

## Monitoring and Observability

The chart includes built-in support for:

- **Metrics**: Prometheus metrics exposed on `/metrics` endpoint
- **Health Checks**: Kubernetes liveness and readiness probes
- **Tracing**: Optional Jaeger integration
- **Logging**: Structured JSON logging

## Development

To test the chart locally:

```bash
# Lint the chart
helm lint ./helm/ufm-service

# Generate templates
helm template my-ufm-service ./helm/ufm-service

# Dry run installation
helm install my-ufm-service ./helm/ufm-service --dry-run --debug
```

## Contributing

1. Make changes to the chart
2. Update the version in `Chart.yaml`
3. Test the changes
4. Submit a pull request