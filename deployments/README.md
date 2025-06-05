# Deployment Configurations

This directory contains deployment configurations and infrastructure-as-code for different environments and platforms.

## Structure

```
deployments/
├── docker/             # Docker configurations
├── k8s/                # Kubernetes manifests
│   ├── base/           # Base Kubernetes resources
│   └── overlays/       # Environment-specific overlays
└── kind/               # Kind (Kubernetes in Docker) configurations
```

## Deployment Options

### Docker (`docker/`)
Containerized deployment using Docker:
- Multi-stage Dockerfile for optimized builds
- Docker Compose configurations for local development
- Production-ready container configurations
- Health checks and monitoring setup

### Kubernetes (`k8s/`)
Cloud-native deployment with Kubernetes:
- **Base**: Common Kubernetes resources (Deployment, Service, ConfigMap)
- **Overlays**: Environment-specific configurations (dev, staging, prod)
- Horizontal Pod Autoscaling configurations
- Ingress and networking setup

### Kind (`kind/`)
Local Kubernetes development using Kind:
- Local cluster configurations
- Development-friendly settings
- Quick iteration and testing setup
- Integration with local Docker registry

## Environment Configurations

### Development
- Single-container setup
- Debug logging enabled
- Hot reload capabilities
- Local database connections

### Staging
- Multi-replica deployment
- Production-like configuration
- Integration testing environment
- Monitoring and observability

### Production
- High availability setup
- Resource limits and requests
- Security policies
- Performance optimization

## Prerequisites

### Docker Deployment
- Docker Engine 20.10+
- Docker Compose 2.0+

### Kubernetes Deployment
- Kubernetes 1.21+
- kubectl configured
- Container registry access

### Kind Deployment
- Kind 0.11+
- Docker Desktop or Docker Engine

## Quick Start

### Docker
```bash
# Build and run locally
cd deployments/docker
docker-compose up --build

# Production deployment
docker build -t assistant:latest .
docker run -p 8080:8080 assistant:latest
```

### Kubernetes
```bash
# Apply base configuration
kubectl apply -k deployments/k8s/base

# Apply environment-specific overlay
kubectl apply -k deployments/k8s/overlays/production
```

### Kind
```bash
# Create local cluster
kind create cluster --config deployments/kind/cluster.yaml

# Deploy to local cluster
kubectl apply -k deployments/k8s/overlays/development
```

## Configuration Management

### Environment Variables
All deployments support configuration via environment variables:
- Database connection strings
- AI provider API keys
- Logging levels
- Feature flags

### Secrets Management
Sensitive data is handled through:
- Kubernetes Secrets
- Docker Secrets
- Environment-specific secret stores

### ConfigMaps
Non-sensitive configuration through:
- Kubernetes ConfigMaps
- Docker environment files
- Configuration file mounting