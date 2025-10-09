---
sidebar_label: Architecture
---

# Architecture

This page provides a technical overview of Kargo's architecture, including its control plane components, resource model, and integration points. Understanding this architecture will help you effectively deploy, operate, and troubleshoot Kargo in production environments.

## Overview

Kargo is a Kubernetes-native continuous promotion platform built on controller-runtime. It extends Kubernetes with custom resources and controllers that automate the promotion of software artifacts through various stages of their lifecycle.

The system follows a declarative model where users define their desired promotion pipelines using Kubernetes resources, and Kargo's controllers work continuously to reconcile the actual state with the desired state.

## Control Plane Components

Kargo's control plane consists of several independently deployable components, each serving a specific purpose. In typical deployments, these components run as separate processes within a single Pod, but they can also be deployed separately for advanced deployment scenarios.

### API Server

The API Server provides both gRPC and REST APIs for interacting with Kargo. It serves as the primary interface for:

- **User Interfaces**: The Kargo Dashboard and CLI communicate with this API
- **Authentication & Authorization**: Handles OIDC-based authentication and integrates with Kubernetes RBAC
- **Resource Operations**: Provides CRUD operations on Kargo resources
- **Real-time Updates**: Supports streaming endpoints for watching resource changes

The API Server does not perform reconciliation logic itself; it delegates resource management to the Kubernetes API server and relies on controllers for state reconciliation.

### Core Resource Controllers

The core controllers are responsible for reconciling the primary Kargo resources that define promotion pipelines.

#### Warehouses Controller

Monitors configured artifact sources (Git repositories, container registries, Helm repositories) and:

- Continuously polls subscribed repositories for new artifact revisions
- Packages discovered artifact revisions into `Freight` resources
- Supports various discovery criteria (image tags, Git branches/tags, semantic version ranges)

#### Stages Controllers

Kargo has two types of Stage controllers:

**Regular Stages Controller**: Manages stages that represent actual deployment targets. It:

- Tracks which `Freight` is available for promotion to each stage
- Maintains stage health status by integrating with health checkers
- Coordinates with the Promotions controller to execute promotions
- Supports both verification and control flow stages

**Control Flow Stages Controller**: Handles special stages used for pipeline orchestration:

- Manages stages that don't deploy artifacts but control promotion flow
- Enables complex patterns like fan-in/fan-out and conditional promotions
- Automatically propagates freight through control flow stages based on configured rules

#### Promotions Controller

Executes promotion processes that update stage desired state. It:

- Runs promotion task steps defined in the stage specification
- Supports built-in tasks (Git operations, Helm, Kustomize, Argo CD operations) and custom steps
- Manages promotion lifecycle through status conditions
- Handles rollback and retry logic
- Tracks promotion history and audit information

### Management Controllers

Management controllers handle cluster-level and project-level configuration and lifecycle.

#### Projects Controller

Manages the lifecycle of Kargo projects. It:

- Creates and manages project namespaces
- Sets up RBAC (Roles, RoleBindings) for project resources
- Manages service accounts and their permissions
- Handles project deletion and cleanup

#### Project Configs Controller

Reconciles project-level configuration. It:

- Manages project-specific settings and policies
- Configures external webhook receivers
- Validates and applies project configuration changes

#### Cluster Configs Controller

Handles cluster-wide Kargo configuration. It:

- Manages global settings that apply to all projects
- Configures cluster-level promotion task registrations
- Handles cluster-scoped security policies

### Webhook Servers

Kargo operates two types of webhook servers for different purposes.

#### Kubernetes Webhooks Server

Implements Kubernetes admission webhooks for validation and mutation:

- **Validating Webhooks**: Enforce resource validation rules before resources are persisted
- **Mutating Webhooks**: Apply defaults and transformations to resources
- **Supported Resources**: Projects, Stages, Promotions, Freight, and configuration resources

These webhooks ensure data integrity and enforce policies at resource creation/modification time.

#### External Webhooks Server

Receives notifications from external systems to trigger reactive behavior:

- Accepts webhook payloads from Git providers (GitHub, GitLab, Bitbucket, etc.)
- Receives notifications from container registries (Docker Hub, Harbor, ACR, etc.)
- Triggers immediate warehouse reconciliation upon receiving relevant notifications
- Reduces polling intervals and improves promotion responsiveness

### Garbage Collector

Performs automated cleanup of old resources:

- Removes old `Freight` resources based on retention policies
- Cleans up completed `Promotion` resources
- Honors project-specific retention settings
- Runs on a configurable schedule

## Resource Model

Kargo extends Kubernetes with custom resources that define promotion pipelines.

### Core Resources

```text
┌─────────────┐
│   Project   │
│  (Namespace)│
└──────┬──────┘
       │
       ├──────────────────┬──────────────────┬─────────────
       │                  │                  │
       ▼                  ▼                  ▼
┌────────────┐     ┌────────────┐     ┌────────────┐
│ Warehouse  │────▶│  Freight   │────▶│   Stage    │
└────────────┘     └────────────┘     └──────┬─────┘
                                              │
                                              ▼
                                       ┌────────────┐
                                       │ Promotion  │
                                       └────────────┘
```

#### Project

A unit of tenancy that:

- Maps to a Kubernetes namespace
- Groups related warehouses, freight, stages, and promotions
- Defines RBAC boundaries
- Isolates resources between teams or applications

#### Warehouse

Monitors artifact sources and produces `Freight`. A warehouse:

- Subscribes to one or more artifact repositories
- Defines discovery criteria (branches, tags, version ranges)
- Automatically creates new `Freight` when new artifact revisions are discovered
- Can monitor Git repositories, container images, and Helm charts

#### Freight

An immutable collection of artifact revisions that:

- References specific versions of artifacts (Git commits, image digests, chart versions)
- Travels through stages as a unit
- Carries metadata about origin and verification status
- Is qualified/approved for promotion to stages based on verification and policy

#### Stage

A promotion target representing a deployment environment or logical step. Stages:

- Subscribe to one or more warehouses (or upstream stages)
- Define promotion steps to execute when new freight arrives
- Track current and historical freight deployments
- Support both automatic and manual promotion modes
- Can be connected in a directed acyclic graph (DAG) to form pipelines

#### Promotion

A resource representing a promotion process. It:

- References a piece of `Freight` and target `Stage`
- Executes promotion task steps defined in the stage
- Tracks execution status and phase (Pending, Running, Succeeded, Failed)
- Maintains a record of what was changed during promotion
- Can be created manually or automatically (for auto-promotion stages)

### Configuration Resources

#### ProjectConfig

Project-level configuration for:

- Promotion policies
- External webhook receivers
- Project-specific settings

#### ClusterConfig

Cluster-wide configuration for:

- Default policies
- Global promotion task definitions
- Cluster-level security settings

#### ClusterPromotionTask

Reusable promotion task definitions that can be referenced by stages across all projects.

## Data Flow

The typical flow of artifacts through Kargo follows this pattern:

```text
1. Warehouse Discovery
   └─▶ New artifact revision detected
       └─▶ Freight created

2. Freight Qualification
   └─▶ Freight becomes available to subscribed stages
       └─▶ Verification may run to qualify freight

3. Promotion Trigger
   └─▶ Manual promotion request OR auto-promotion
       └─▶ Promotion resource created

4. Promotion Execution
   └─▶ Promotion steps execute
       ├─▶ Git operations (clone, commit, push)
       ├─▶ Helm/Kustomize rendering
       ├─▶ Argo CD sync operations
       └─▶ Custom steps

5. Stage Update
   └─▶ Stage current freight updated
       └─▶ Downstream stages can now promote this freight
```

## Integration Architecture

### Kubernetes Integration

Kargo is deeply integrated with Kubernetes:

- **Custom Resource Definitions (CRDs)**: Extends Kubernetes API with Kargo resources
- **Controller Runtime**: Built on controller-runtime for reliable reconciliation
- **RBAC**: Uses Kubernetes RBAC for authorization
- **Events**: Emits Kubernetes events for auditing and monitoring
- **Namespaces**: Projects map directly to Kubernetes namespaces

### Argo CD Integration (Optional)

Kargo can optionally integrate with Argo CD:

- **Application Sync**: Promotion steps can trigger Argo CD application syncs
- **Health Checking**: Queries Argo CD application health for stage health
- **Multi-Cluster**: Supports Argo CD's multi-cluster deployments
- **Application Sets**: Works with ApplicationSets for multi-tenant scenarios

The integration is optional; Kargo can work with any GitOps agent or deployment tool.

### Argo Rollouts Integration (Optional)

For advanced deployment strategies:

- **Analysis Runs**: Stages can run Argo Rollouts analysis for freight verification
- **Progressive Delivery**: Coordinates with Rollouts for canary/blue-green deployments

### External System Integration

Kargo integrates with various external systems:

**Git Providers**:

- GitHub, GitLab, Bitbucket, Gitea, Azure DevOps
- For both artifact discovery and webhook notifications

**Container Registries**:

- Docker Hub, Harbor, ACR, ECR, GCR, Quay, Artifactory
- For image discovery and webhook notifications

**Helm Repositories**:

- OCI registries, HTTP chart repositories
- For chart version discovery

**Credentials Management**:

- Kubernetes Secrets
- Flexible credential types (Git, Helm, container registry)
- Project-scoped and cluster-scoped credentials

## Deployment Considerations

### Single Control Plane Deployment

The simplest deployment runs all control plane components in a single Pod:

```text
┌────────────────────────────────────────┐
│         Kargo Control Plane Pod        │
├────────────────────────────────────────┤
│  • API Server                          │
│  • Controllers                         │
│  • Management Controllers              │
│  • Kubernetes Webhooks Server          │
│  • External Webhooks Server            │
│  • Garbage Collector                   │
└────────────────────────────────────────┘
```

This is suitable for:

- Development and testing
- Small to medium deployments
- Simplified operations

### Sharded Controller Deployment

For larger deployments, controllers can be sharded to distribute load:

```text
┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│ Controller   │  │ Controller   │  │ Controller   │
│  Shard A     │  │  Shard B     │  │  Default     │
├──────────────┤  ├──────────────┤  ├──────────────┤
│ Resources    │  │ Resources    │  │ Resources    │
│ labeled:     │  │ labeled:     │  │ without      │
│ shard=A      │  │ shard=B      │  │ shard label  │
└──────────────┘  └──────────────┘  └──────────────┘
```

Sharding is achieved by:

- Labeling resources with `kargo.akuity.io/shard: <shard-name>`
- Deploying multiple controller instances with `--shard` flag
- Each controller processes only resources matching its shard

### High Availability

For production deployments:

- **API Server**: Can run multiple replicas behind a load balancer
- **Controllers**: Leader election ensures only one active controller per resource
- **Webhooks**: Multiple replicas for availability (must use TLS with proper certificates)
- **Database**: Kubernetes API server provides the persistence layer (etcd)

### Resource Requirements

Typical resource requirements vary by scale:

**Small Deployment** (< 50 projects, < 500 stages):

- CPU: 500m-1 core
- Memory: 512Mi-1Gi

**Medium Deployment** (50-200 projects, 500-2000 stages):

- CPU: 1-2 cores
- Memory: 1-2Gi

**Large Deployment** (> 200 projects, > 2000 stages):

- CPU: 2-4 cores (consider sharding)
- Memory: 2-4Gi
- Multiple controller shards

## Security Considerations

### Authentication

- **OIDC Integration**: API server supports OpenID Connect for user authentication
- **Service Accounts**: Controllers use Kubernetes service accounts
- **Token-based**: CLI and API clients use bearer tokens

### Authorization

- **Kubernetes RBAC**: Native integration with Kubernetes RBAC system
- **Project Isolation**: Projects (namespaces) provide tenant isolation
- **Least Privilege**: Controllers run with minimal required permissions
- **Custom Roles**: Support for project-specific role definitions

### Network Security

- **TLS**: HTTPS/gRPC with TLS for API server
- **Webhook TLS**: Kubernetes webhooks require TLS certificates
- **Network Policies**: Standard Kubernetes network policies apply
- **Ingress**: API server typically exposed via Ingress with TLS termination

### Secrets Management

- **Credential Storage**: Credentials stored as Kubernetes Secrets
- **Encryption at Rest**: Leverages Kubernetes secret encryption
- **Project Scoping**: Credentials can be project-scoped or cluster-scoped
- **Minimal Exposure**: Credentials only accessible to necessary components

## Observability

### Metrics

All components expose Prometheus metrics:

- Controller reconciliation rates and durations
- API request rates and latencies
- Queue depths and processing times
- Resource counts and states

### Logging

Structured logging with configurable levels:

- JSON or text format
- Contextual logging (project, stage, promotion)
- Correlation with Kubernetes events

### Events

Kubernetes events for important state changes:

- Freight discovery
- Promotion start/completion
- Errors and warnings
- Resource state transitions

### Tracing

OpenTelemetry support for distributed tracing across components (when configured).

## What's Next?

- **[Basic Installation](./10-basic-installation.md)**: Get Kargo up and running
- **[Advanced Installation](./20-advanced-installation/)**: Customize your deployment
- **[Cluster Configuration](./35-cluster-configuration.md)**: Configure cluster-level settings
- **[Security](./40-security/)**: Secure your Kargo installation
