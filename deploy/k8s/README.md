# Kubernetes Deployment Manifests

This directory contains Kubernetes manifests for the Subcults platform.

## Manifests

| File                          | Description                                          |
| ----------------------------- | ---------------------------------------------------- |
| `namespace.yaml`              | Subcults namespace                                   |
| `configmap.yaml`              | Non-secret configuration (env, ports, feature flags) |
| `secrets.yaml.template`       | Secret template — fill in before applying            |
| `api-deployment.yaml`         | API Deployment (3 replicas) + Service + HPA          |
| `indexer-statefulset.yaml`    | Indexer StatefulSet (1 replica) + Service            |
| `frontend-deployment.yaml`    | Frontend Deployment (2 replicas) + Service           |
| `network-policies.yaml`       | Inter-service traffic restrictions                   |
| `pod-disruption-budgets.yaml` | PDBs for API and Frontend                            |
| `migration-job.yaml`          | Pre-deployment database migration Job                |

## Quick Start

```bash
# 1. Create namespace
kubectl apply -f deploy/k8s/namespace.yaml

# 2. Create secrets (fill in real values first)
cp deploy/k8s/secrets.yaml.template deploy/k8s/secrets.yaml
# Edit secrets.yaml with real values, then:
kubectl apply -f deploy/k8s/secrets.yaml

# 3. Apply config and workloads
kubectl apply -f deploy/k8s/configmap.yaml
kubectl apply -f deploy/k8s/api-deployment.yaml
kubectl apply -f deploy/k8s/indexer-statefulset.yaml
kubectl apply -f deploy/k8s/frontend-deployment.yaml

# 4. Apply policies
kubectl apply -f deploy/k8s/network-policies.yaml
kubectl apply -f deploy/k8s/pod-disruption-budgets.yaml
```

## Pre-Deployment Migration

Run migrations before deploying a new version:

```bash
kubectl apply -f deploy/k8s/migration-job.yaml
kubectl wait --for=condition=complete job/subcults-migrate -n subcults --timeout=300s
```
