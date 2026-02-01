# StateLedger Helm Chart

Production-ready Helm chart for deploying StateLedger on Kubernetes.

## Quick Start

```bash
helm install stateledger ./deployments/helm/stateledger \
  --namespace stateledger \
  --create-namespace
```

## Features

- ✅ Persistent volume support for ledger database
- ✅ Security best practices (non-root user, read-only filesystem)
- ✅ Horizontal Pod Autoscaling (HPA) support
- ✅ Health checks (liveness & readiness probes)
- ✅ Service account and RBAC ready
- ✅ Ingress support
- ✅ Resource limits and requests

## Configuration

### Basic Settings

```yaml
# Default image
image:
  repository: retr0xd/stateledger
  tag: latest

# Replica count
replicaCount: 1

# Resource limits
resources:
  limits:
    cpu: 500m
    memory: 512Mi
  requests:
    cpu: 250m
    memory: 256Mi
```

### Persistence

Enable persistent storage for ledger database:

```yaml
persistence:
  enabled: true
  storageClass: standard
  size: 10Gi
  mountPath: /app/ledger
```

### Autoscaling

Enable HPA for automatic scaling:

```yaml
autoscaling:
  enabled: true
  minReplicas: 1
  maxReplicas: 3
  targetCPUUtilizationPercentage: 80
```

### Ingress

Configure ingress for external access:

```yaml
ingress:
  enabled: true
  className: nginx
  hosts:
    - host: stateledger.example.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: stateledger-tls
      hosts:
        - stateledger.example.com
```

### Health Checks

Enable liveness and readiness probes:

```yaml
livenessProbe:
  enabled: true
  initialDelaySeconds: 30
  periodSeconds: 10

readinessProbe:
  enabled: true
  initialDelaySeconds: 10
  periodSeconds: 5
```

## Deployment Examples

### Simple Deployment

```bash
helm install stateledger ./deployments/helm/stateledger
```

### Production Deployment with Persistence

```bash
helm install stateledger ./deployments/helm/stateledger \
  -f values-prod.yaml \
  --set persistence.enabled=true \
  --set persistence.size=50Gi
```

### With Autoscaling

```bash
helm install stateledger ./deployments/helm/stateledger \
  --set autoscaling.enabled=true \
  --set autoscaling.maxReplicas=5
```

## Uninstall

```bash
helm uninstall stateledger
```

## Testing

Validate the chart:

```bash
helm lint ./deployments/helm/stateledger
helm template stateledger ./deployments/helm/stateledger
```

Dry-run installation:

```bash
helm install stateledger ./deployments/helm/stateledger --dry-run --debug
```
