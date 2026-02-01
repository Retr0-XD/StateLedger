# StateLedger Kustomize Overlays

Kustomize configurations for deploying StateLedger across different environments.

## Directory Structure

```
kustomize/
├── base/                    # Base configuration
│   ├── deployment.yaml
│   ├── service.yaml
│   ├── pvc.yaml
│   ├── serviceaccount.yaml
│   └── kustomization.yaml
└── overlays/
    ├── dev/                # Development environment
    ├── staging/            # Staging environment
    └── prod/               # Production environment
```

## Quick Start

### Development Deployment

```bash
kubectl apply -k deployments/kustomize/overlays/dev
```

Features:
- 1 replica
- Debug logging
- Low resource requests
- Always pull image for latest changes

### Staging Deployment

```bash
kubectl apply -k deployments/kustomize/overlays/staging
```

Features:
- 2 replicas
- Info logging
- Standard resource limits
- 20Gi persistent storage

### Production Deployment

```bash
kubectl apply -k deployments/kustomize/overlays/prod
```

Features:
- 3 replicas with pod anti-affinity
- Warning logging (minimal)
- High resource limits (1Gi RAM, 1CPU)
- 50Gi fast-ssd storage
- Health checks (liveness & readiness)
- PodDisruptionBudget recommended

## Configuration Customization

### Override Images

Create a `kustomization.yaml` overlay:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

bases:
  - ../../base

images:
  - name: retr0xd/stateledger
    newTag: v1.2.3
```

### Add ConfigMap

```yaml
configMapGenerator:
  - name: stateledger-config
    files:
      - config.json
```

### Update Resource Limits

```yaml
patchesStrategicMerge:
  - deployment.yaml
```

In `deployment.yaml`:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: stateledger
spec:
  template:
    spec:
      containers:
      - name: stateledger
        resources:
          requests:
            cpu: 1000m
            memory: 1Gi
```

## Validation

Test configurations without applying:

```bash
# Dev
kubectl kustomize deployments/kustomize/overlays/dev

# Staging
kubectl kustomize deployments/kustomize/overlays/staging

# Prod
kubectl kustomize deployments/kustomize/overlays/prod
```

## Environment-Specific Settings

| Setting | Dev | Staging | Prod |
|---------|-----|---------|------|
| Replicas | 1 | 2 | 3 |
| CPU Req. | 100m | 250m | 500m |
| Memory Req. | 128Mi | 256Mi | 512Mi |
| CPU Limit | 200m | 500m | 1000m |
| Memory Limit | 256Mi | 512Mi | 1Gi |
| Storage | 10Gi | 20Gi | 50Gi |
| Log Level | debug | info | warn |
| Image Pull | Always | IfNotPresent | IfNotPresent |
| Health Checks | None | None | Enabled |

## Namespace Management

Default namespace per environment:

- **dev**: `stateledger-dev`
- **staging**: `stateledger`
- **prod**: `stateledger`

Create namespaces first:

```bash
kubectl create namespace stateledger-dev
kubectl create namespace stateledger
```

## Cleanup

Remove deployment by environment:

```bash
# Dev
kubectl delete -k deployments/kustomize/overlays/dev

# Staging
kubectl delete -k deployments/kustomize/overlays/staging

# Prod
kubectl delete -k deployments/kustomize/overlays/prod
```
