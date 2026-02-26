# Admin System Architecture Selection Document

**Document Version:** 1.0  
**Date:** February 26, 2026  
**System:** Admin Backend Management System

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Technology Stack Comparison](#technology-stack-comparison)
3. [Architecture Style Analysis](#architecture-style-analysis)
4. [Recommended Architecture Plan](#recommended-architecture-plan)
5. [Risk Analysis & Mitigation](#risk-analysis--mitigation)
6. [Architecture Diagram](#architecture-diagram)
7. [Module Splitting Strategy](#module-splitting-strategy)
8. [RBAC System Architecture](#rbac-system-architecture)
9. [Infrastructure Components](#infrastructure-components)

---

## 1. Executive Summary

This document provides a comprehensive architecture plan for an enterprise-grade admin backend system supporting RBAC, audit logging, system monitoring, configuration management, and multi-environment deployment capabilities.

**Key Requirements:**
- User & Permission Management (RBAC with 4 role levels)
- Security & Authentication (JWT, IP restrictions, 2FA)
- Audit & Logging System
- Configuration Center
- System Monitoring & Alerting
- Data Analytics Dashboard
- Multi-environment support

---

## 2. Technology Stack Comparison

### Plan A: Node.js Ecosystem (Modern & Fast Development)

| Layer | Technology | Justification |
|-------|-----------|---------------|
| **Backend Framework** | NestJS | TypeScript-based, modular architecture, built-in DI, excellent for scalable systems |
| **API Style** | REST + GraphQL | REST for CRUD, GraphQL for complex queries |
| **Language** | TypeScript | Type safety, better maintainability |
| **ORM** | Prisma / TypeORM | Modern ORM with migrations, type-safe queries |
| **Database** | PostgreSQL 15+ | ACID compliance, JSON support, mature RBAC features |
| **Cache** | Redis 7+ | Session storage, distributed locks, pub/sub |
| **Message Queue** | Bull (Redis-based) | Job scheduling, async tasks, retry mechanisms |
| **Search** | Elasticsearch | Full-text search, log aggregation |
| **Frontend** | React 18 + Ant Design Pro | Rich admin components, proven enterprise UI |
| **State Management** | Zustand / Tanstack Query | Lightweight, server state management |
| **Logging** | Winston + ELK Stack | Structured logging, centralized log management |
| **Monitoring** | Prometheus + Grafana | Metrics collection, visualization |
| **APM** | Sentry / DataDog | Error tracking, performance monitoring |
| **Testing** | Jest + Supertest | Unit & integration testing |
| **Documentation** | Swagger / OpenAPI | Auto-generated API docs |
| **Deployment** | Docker + K8s | Container orchestration, auto-scaling |
| **CI/CD** | GitHub Actions / GitLab CI | Automated testing & deployment |

**Team Size:** 3-5 developers  
**Development Speed:** ⭐⭐⭐⭐⭐  
**Performance:** ⭐⭐⭐⭐  
**Ecosystem Maturity:** ⭐⭐⭐⭐⭐

### Plan B: Java Enterprise Stack (Battle-Tested Reliability)

| Layer | Technology | Justification |
|-------|-----------|---------------|
| **Backend Framework** | Spring Boot 3.x | Industry standard, comprehensive ecosystem |
| **API Style** | REST | RESTful APIs with Spring MVC |
| **Language** | Java 17+ / Kotlin | Enterprise-grade, mature tooling |
| **ORM** | MyBatis-Plus / JPA | Flexible SQL control or entity management |
| **Database** | PostgreSQL / MySQL 8+ | High performance, proven at scale |
| **Cache** | Redis + Caffeine | Distributed + local multi-level caching |
| **Message Queue** | RabbitMQ / Kafka | Enterprise MQ, high throughput |
| **Search** | Elasticsearch | Same as Plan A |
| **Frontend** | Vue 3 + Element Plus | Progressive framework, easy learning curve |
| **State Management** | Pinia | Vue's official state management |
| **Logging** | Logback + ELK Stack | Java standard logging |
| **Monitoring** | Micrometer + Prometheus | Spring Boot integration |
| **APM** | SkyWalking / Pinpoint | Java-specific APM solutions |
| **Testing** | JUnit 5 + Mockito | Comprehensive Java testing |
| **Documentation** | SpringDoc OpenAPI | Integrated Swagger UI |
| **Deployment** | Docker + K8s | Same as Plan A |
| **Security** | Spring Security | Mature security framework |

**Team Size:** 4-8 developers  
**Development Speed:** ⭐⭐⭐⭐  
**Performance:** ⭐⭐⭐⭐⭐  
**Ecosystem Maturity:** ⭐⭐⭐⭐⭐

### Plan C: Go Microservice Stack (High Performance & Cloud-Native)

| Layer | Technology | Justification |
|-------|-----------|---------------|
| **Backend Framework** | Go-Zero / Gin | High performance, built for microservices |
| **API Style** | REST + gRPC | HTTP for external, gRPC for internal |
| **Language** | Go 1.21+ | Fast compilation, excellent concurrency |
| **ORM** | GORM / Ent | Powerful ORM with code generation |
| **Database** | PostgreSQL + CockroachDB | Distributed SQL for scalability |
| **Cache** | Redis Cluster | Distributed caching |
| **Message Queue** | NATS / Kafka | Cloud-native messaging |
| **Search** | Elasticsearch | Same as above |
| **Frontend** | React + Ant Design | Same as Plan A |
| **State Management** | Zustand | Lightweight state management |
| **Logging** | Zap + Loki | High-performance structured logging |
| **Monitoring** | Prometheus + Grafana | Native Go metrics |
| **APM** | Jaeger | Distributed tracing |
| **Testing** | Testify + Gomock | Go testing frameworks |
| **Documentation** | Swaggo | Swagger for Go |
| **Deployment** | Docker + K8s | Cloud-native deployment |
| **Service Mesh** | Istio (optional) | Advanced traffic management |

**Team Size:** 5-10 developers (microservices complexity)  
**Development Speed:** ⭐⭐⭐  
**Performance:** ⭐⭐⭐⭐⭐  
**Ecosystem Maturity:** ⭐⭐⭐⭐

---

## 3. Architecture Style Analysis

### Option 1: Monolithic Architecture

```
┌────────────────────────────────────────┐
│         Single Application             │
│  ┌──────────────────────────────────┐ │
│  │  Controller Layer                │ │
│  ├──────────────────────────────────┤ │
│  │  Service Layer                   │ │
│  │  • RBAC  • Logging  • Config     │ │
│  │  • User  • Monitor  • Message    │ │
│  ├──────────────────────────────────┤ │
│  │  Data Access Layer               │ │
│  └──────────────────────────────────┘ │
└────────────────────────────────────────┘
         │                    │
    ┌─────────┐          ┌─────────┐
    │   DB    │          │  Redis  │
    └─────────┘          └─────────┘
```

**Advantages:**
- ✅ Simple deployment (single process)
- ✅ Easy debugging and testing
- ✅ Fast development in early stages
- ✅ No network latency between modules
- ✅ ACID transactions across all modules
- ✅ Lower infrastructure costs

**Disadvantages:**
- ❌ Tight coupling between modules
- ❌ Difficult to scale specific features
- ❌ Long build and deployment times
- ❌ Technology stack lock-in
- ❌ Single point of failure
- ❌ Resource contention between modules

**Suitable For:**
- Small to medium-sized teams (2-5 developers)
- MVP and early-stage products
- Systems with <100K users
- Budget constraints
- Limited DevOps resources

**Technology Recommendation:** Plan A (NestJS) or Plan B (Spring Boot)

---

### Option 2: Modular Monolithic Architecture (Recommended)

```
┌─────────────────────────────────────────────────────┐
│         Modular Monolithic Application              │
│  ┌────────────┐ ┌────────────┐ ┌────────────┐     │
│  │  RBAC      │ │  Logging   │ │  Config    │     │
│  │  Module    │ │  Module    │ │  Module    │     │
│  └────────────┘ └────────────┘ └────────────┘     │
│  ┌────────────┐ ┌────────────┐ ┌────────────┐     │
│  │  User      │ │  Monitor   │ │  Message   │     │
│  │  Module    │ │  Module    │ │  Module    │     │
│  └────────────┘ └────────────┘ └────────────┘     │
│                                                      │
│  ┌──────────────────────────────────────────────┐  │
│  │         Shared Kernel (Domain Core)          │  │
│  └──────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────┘
         │                    │              │
    ┌─────────┐          ┌─────────┐   ┌─────────┐
    │   DB    │          │  Redis  │   │   MQ    │
    └─────────┘          └─────────┘   └─────────┘
```

**Advantages:**
- ✅ Clear module boundaries with domain separation
- ✅ Independent development by different teams
- ✅ Easier to understand and maintain
- ✅ Can evolve into microservices later
- ✅ Single deployment simplicity
- ✅ Shared infrastructure and utilities
- ✅ Can use database-level isolation (schemas)

**Disadvantages:**
- ❌ Still a single deployment unit
- ❌ Requires discipline to maintain boundaries
- ❌ Can't independently scale modules
- ❌ Shared database can be a bottleneck

**Suitable For:**
- Medium-sized teams (3-8 developers)
- Growing products (100K - 1M users)
- Want clean architecture without microservices complexity
- Plan to scale in the future
- Need fast feature development with good structure

**Technology Recommendation:** Plan A (NestJS with Modules) or Plan B (Spring Boot with Modules)

---

### Option 3: Microservices Architecture

```
                    ┌──────────────────┐
                    │   API Gateway    │
                    │   (Kong/APISIX)  │
                    └──────────────────┘
                             │
        ┌────────────────────┼────────────────────┐
        │                    │                    │
   ┌─────────┐         ┌─────────┐         ┌─────────┐
   │  RBAC   │         │  User   │         │ Logging │
   │ Service │         │ Service │         │ Service │
   └─────────┘         └─────────┘         └─────────┘
        │                    │                    │
   ┌─────────┐         ┌─────────┐         ┌─────────┐
   │ Config  │         │ Monitor │         │ Message │
   │ Service │         │ Service │         │ Service │
   └─────────┘         └─────────┘         └─────────┘
        │                    │                    │
   ┌─────────┐         ┌─────────┐         ┌─────────┐
   │   DB1   │         │   DB2   │         │   DB3   │
   └─────────┘         └─────────┘         └─────────┘
        
             ┌─────────────────────────┐
             │  Service Mesh (Istio)   │
             │  Distributed Tracing    │
             │  Service Discovery      │
             └─────────────────────────┘
```

**Advantages:**
- ✅ Independent scaling of services
- ✅ Technology diversity (polyglot)
- ✅ Independent deployments
- ✅ Better fault isolation
- ✅ Team autonomy
- ✅ Optimized for specific use cases

**Disadvantages:**
- ❌ High operational complexity
- ❌ Distributed system challenges (CAP theorem)
- ❌ Network latency between services
- ❌ Distributed transactions difficulty
- ❌ Testing complexity
- ❌ Higher infrastructure costs
- ❌ Requires mature DevOps practices

**Suitable For:**
- Large teams (8+ developers, multiple teams)
- High-scale systems (1M+ users)
- Need independent scaling
- Different performance requirements per module
- Mature DevOps culture
- Budget for infrastructure

**Technology Recommendation:** Plan C (Go with gRPC) or Plan B (Spring Cloud)

---

## 4. Recommended Architecture Plan

### **🎯 PRIMARY RECOMMENDATION: Modular Monolithic with NestJS (Plan A - Enhanced)**

#### Rationale

Based on the requirements analysis, I recommend a **Modular Monolithic Architecture** using the Node.js ecosystem (NestJS) for the following reasons:

1. **Optimal Balance:** Provides clean architecture without microservices complexity
2. **Fast Development:** TypeScript + NestJS enables rapid feature development
3. **Future-Proof:** Easy migration path to microservices if needed
4. **Team Efficiency:** Suitable for 3-8 developers
5. **Modern Stack:** Rich ecosystem, excellent tooling
6. **Cost-Effective:** Lower infrastructure and operational costs
7. **Performance:** Sufficient for 100K-1M users with proper optimization

#### Complete Technology Stack

```yaml
Backend:
  Framework: NestJS 10.x
  Language: TypeScript 5.x
  Runtime: Node.js 20.x LTS
  API Protocol: REST + GraphQL
  Validation: class-validator + class-transformer
  
Database:
  Primary: PostgreSQL 15+ (with pgvector for future AI features)
  ORM: Prisma 5.x
  Migration: Prisma Migrate
  Connection Pool: Max 50 connections
  Backup: Daily automated backups with 30-day retention
  
Caching:
  Layer 1: Redis 7+ (session, distributed locks, rate limiting)
  Layer 2: Node-cache (in-memory for hot data)
  Strategy: Cache-aside pattern with TTL
  
Message Queue:
  Primary: BullMQ (Redis-based)
  Use Cases: 
    - Email/SMS sending
    - Log processing
    - Report generation
    - Video processing (if needed)
  
Search & Analytics:
  Search: Elasticsearch 8.x
  Time-series: TimescaleDB extension on PostgreSQL
  Analytics: ClickHouse (optional for heavy analytics)
  
File Storage:
  Cloud: AWS S3 / Alibaba Cloud OSS
  CDN: CloudFront / Alibaba CDN
  Local Dev: MinIO
  
Frontend:
  Framework: React 18+ with Vite
  UI Library: Ant Design Pro 5.x
  State: Zustand + TanStack Query
  Routing: React Router 6
  Form: React Hook Form + Zod
  Charts: Apache ECharts
  Build: Vite 5.x
  
Security:
  Authentication: JWT (Access + Refresh tokens)
  Authorization: CASL (Permission-based)
  Encryption: bcrypt (passwords), AES-256 (sensitive data)
  Rate Limiting: express-rate-limit + Redis
  CORS: Configurable whitelist
  Helmet: Security headers
  
Logging:
  Library: Winston
  Format: JSON structured logging
  Levels: error, warn, info, debug
  Transport: Console + File + Elasticsearch
  Log Rotation: Daily with 30-day retention
  
Monitoring & Observability:
  Metrics: Prometheus + Grafana
  APM: Sentry (errors), DataDog (optional)
  Health Checks: @nestjs/terminus
  Uptime: UptimeRobot / Pingdom
  Alerts: PagerDuty / Slack webhooks
  
Testing:
  Unit: Jest
  E2E: Supertest
  Load: k6
  Coverage: >80% target
  
DevOps:
  Containerization: Docker + Docker Compose
  Orchestration: Kubernetes (production)
  CI/CD: GitHub Actions
  IaC: Terraform
  Secrets: Vault / AWS Secrets Manager
  Monitoring: Prometheus + Grafana + Loki
  
Development:
  IDE: VSCode with ESLint, Prettier
  Code Quality: Husky + lint-staged
  Commit: Conventional Commits
  Versioning: Semantic Versioning
  Documentation: Compodoc + Swagger
```

#### Architecture Layers

```
┌─────────────────────────────────────────────────────────┐
│                    Presentation Layer                    │
│  ┌──────────────────────────────────────────────────┐  │
│  │  REST API Controllers + GraphQL Resolvers        │  │
│  │  Guards, Interceptors, Pipes (NestJS)           │  │
│  └──────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────┘
                            │
┌─────────────────────────────────────────────────────────┐
│                   Application Layer                      │
│  ┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐         │
│  │ Auth   │ │ RBAC   │ │ User   │ │ Config │ ...     │
│  │ Service│ │ Service│ │ Service│ │ Service│         │
│  └────────┘ └────────┘ └────────┘ └────────┘         │
│  • Business Logic                                        │
│  • Validation & Orchestration                           │
│  • DTOs (Data Transfer Objects)                         │
└─────────────────────────────────────────────────────────┘
                            │
┌─────────────────────────────────────────────────────────┐
│                     Domain Layer                         │
│  ┌──────────────────────────────────────────────────┐  │
│  │  Domain Models (Entities)                        │  │
│  │  Business Rules & Invariants                     │  │
│  │  Domain Events                                    │  │
│  └──────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────┘
                            │
┌─────────────────────────────────────────────────────────┐
│                  Infrastructure Layer                    │
│  ┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐         │
│  │Prisma  │ │ Redis  │ │ S3     │ │ MQ     │         │
│  │Client  │ │ Client │ │ Client │ │ Client │         │
│  └────────┘ └────────┘ └────────┘ └────────┘         │
│  • Repository Implementations                            │
│  • External Service Integrations                        │
│  • Caching, Logging, Monitoring                         │
└─────────────────────────────────────────────────────────┘
```

#### Module Structure

```
src/
├── modules/
│   ├── auth/                 # Authentication & Authorization
│   │   ├── auth.module.ts
│   │   ├── auth.service.ts
│   │   ├── auth.controller.ts
│   │   ├── strategies/       # JWT, Local, OAuth strategies
│   │   ├── guards/           # Auth guards
│   │   └── decorators/       # Custom decorators
│   │
│   ├── rbac/                 # Role-Based Access Control
│   │   ├── rbac.module.ts
│   │   ├── role.service.ts
│   │   ├── permission.service.ts
│   │   ├── casl/             # CASL ability definitions
│   │   └── guards/           # Permission guards
│   │
│   ├── user/                 # User Management
│   │   ├── user.module.ts
│   │   ├── user.service.ts
│   │   ├── user.controller.ts
│   │   ├── entities/
│   │   └── dto/
│   │
│   ├── audit-log/            # Operation Audit Logging
│   │   ├── audit-log.module.ts
│   │   ├── audit-log.service.ts
│   │   ├── interceptors/     # Auto-logging interceptor
│   │   └── decorators/       # @Auditable decorator
│   │
│   ├── config/               # Configuration Center
│   │   ├── config.module.ts
│   │   ├── config.service.ts
│   │   ├── config.controller.ts
│   │   └── validators/       # Config validation schemas
│   │
│   ├── monitoring/           # System Monitoring
│   │   ├── monitoring.module.ts
│   │   ├── health.controller.ts
│   │   ├── metrics.service.ts
│   │   └── collectors/       # Custom metrics
│   │
│   ├── notification/         # Message Center
│   │   ├── notification.module.ts
│   │   ├── notification.service.ts
│   │   ├── sms/              # SMS service
│   │   ├── email/            # Email service
│   │   └── templates/        # Message templates
│   │
│   ├── analytics/            # Data Center
│   │   ├── analytics.module.ts
│   │   ├── analytics.service.ts
│   │   ├── dashboard.controller.ts
│   │   └── reports/          # Report generation
│   │
│   └── file/                 # File Management
│       ├── file.module.ts
│       ├── file.service.ts
│       ├── upload.controller.ts
│       └── storage/          # Storage adapters
│
├── common/
│   ├── decorators/           # Shared decorators
│   ├── filters/              # Exception filters
│   ├── interceptors/         # Request/Response interceptors
│   ├── pipes/                # Validation pipes
│   ├── guards/               # Shared guards
│   ├── utils/                # Helper functions
│   └── constants/            # App constants
│
├── config/
│   ├── database.config.ts
│   ├── redis.config.ts
│   ├── jwt.config.ts
│   └── app.config.ts
│
├── prisma/
│   ├── schema.prisma
│   └── migrations/
│
├── app.module.ts
└── main.ts
```

---

## 5. Risk Analysis & Mitigation

### Risk 1: Database Performance Bottleneck
**Severity:** HIGH  
**Probability:** MEDIUM

**Risks:**
- High concurrency on user/permission tables
- Slow audit log queries
- Lock contention on configuration updates

**Mitigation:**
```yaml
Short-term:
  - Implement multi-level caching (Redis + in-memory)
  - Database indexing strategy:
      - B-tree indexes on frequently queried columns
      - Partial indexes on filtered queries
      - Covering indexes for specific queries
  - Connection pooling optimization (pgBouncer)
  - READ replicas for analytics queries

Long-term:
  - Partition large tables (audit_logs by date)
  - Implement CQRS for read-heavy operations
  - Consider TimescaleDB for time-series data
  - Archive old data (>6 months) to cold storage
```

---

### Risk 2: Security Vulnerabilities
**Severity:** CRITICAL  
**Probability:** MEDIUM

**Risks:**
- SQL injection attacks
- XSS attacks on admin panel
- JWT token theft
- Privilege escalation
- DDoS attacks

**Mitigation:**
```yaml
Prevention:
  - Use Prisma (parameterized queries)
  - Implement CSP headers
  - Secure token storage (httpOnly cookies + Redis whitelist)
  - Regular security audits with OWASP ZAP
  - Rate limiting (100 req/min per IP)
  - Input validation with class-validator
  - Sensitive data encryption at rest

Detection:
  - Implement WAF (Cloudflare / AWS WAF)
  - Security monitoring with Sentry
  - Anomaly detection on login attempts
  - Regular dependency vulnerability scanning (npm audit, Snyk)

Response:
  - Automated account lockout after 5 failed attempts
  - Security incident playbook
  - Regular backups with 30-day retention
  - Disaster recovery plan (RTO: 4 hours, RPO: 1 hour)
```

---

### Risk 3: System Scalability Limits
**Severity:** MEDIUM  
**Probability:** MEDIUM

**Risks:**
- Monolith cannot handle 1M+ concurrent users
- Message queue bottleneck
- File storage limitations

**Mitigation:**
```yaml
Horizontal Scaling:
  - Kubernetes with HPA (min: 2, max: 10 pods)
  - Stateless application design
  - Redis Cluster for distributed caching
  - Load balancer (Nginx / ALB)

Vertical Scaling:
  - Database: Scale up to 32 cores / 128GB RAM
  - Redis: Scale up memory as needed

Migration Path:
  - Design modules with clear boundaries
  - Use dependency injection for loose coupling
  - Implement API versioning
  - Document service boundaries for future microservices split
  
Targets:
  - <200ms P95 response time
  - Support 10K concurrent users
  - 99.9% uptime SLA
```

---

### Risk 4: Data Loss / Corruption
**Severity:** CRITICAL  
**Probability:** LOW

**Risks:**
- Hardware failure
- Accidental deletions
- Ransomware attacks
- Database corruption

**Mitigation:**
```yaml
Backup Strategy:
  - Automated daily backups (3 AM)
  - Point-in-time recovery capability
  - Multi-region backup replication
  - Backup encryption
  - Monthly restore testing

Data Protection:
  - Soft delete for critical data
  - Audit trail for all changes
  - Database transaction logging
  - WAL archiving for PostgreSQL
  
Recovery:
  - RTO: 4 hours
  - RPO: 1 hour (5-minute WAL archiving)
  - Documented disaster recovery procedures
```

---

### Risk 5: Third-Party Service Failures
**Severity:** MEDIUM  
**Probability:** MEDIUM

**Risks:**
- SMS service downtime
- Cloud storage unavailable
- Payment gateway failures

**Mitigation:**
```yaml
Resilience Patterns:
  - Circuit breaker (nest-circuit-breaker)
  - Retry with exponential backoff
  - Fallback mechanisms
  - Timeout configuration (5s default)

Multi-Provider Strategy:
  - SMS: Alibaba Cloud + Twilio fallback
  - Storage: Primary S3 + backup provider
  - Email: SendGrid + AWS SES

Monitoring:
  - Health checks for all external services
  - Alert on consecutive failures (>3)
  - SLA tracking and reporting
```

---

### Risk 6: Team Knowledge Gaps
**Severity:** MEDIUM  
**Probability:** HIGH

**Risks:**
- TypeScript/NestJS learning curve
- DevOps complexity
- Architecture understanding

**Mitigation:**
```yaml
Training:
  - 2-week onboarding program
  - Code review process (2 approvers required)
  - Pair programming for complex features
  - Weekly tech sharing sessions

Documentation:
  - Architecture decision records (ADR)
  - API documentation (Swagger)
  - Deployment runbooks
  - Troubleshooting guides
  
Knowledge Sharing:
  - Design review meetings
  - Post-mortems for incidents
  - Internal tech blog
  - Code comments and TSDoc
```

---

## 6. Architecture Diagram

### System Architecture Overview

```
┌───────────────────────────────────────────────────────────────────┐
│                         External Layer                             │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐         │
│  │  Admin   │  │  Mobile  │  │ 3rd Party│  │  CDN     │         │
│  │   Web    │  │   App    │  │   API    │  │ (Static) │         │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘         │
└───────────────────────────────────────────────────────────────────┘
                               │
                               ▼
┌───────────────────────────────────────────────────────────────────┐
│                    CDN / WAF (Cloudflare)                          │
│                         DDoS Protection                            │
└───────────────────────────────────────────────────────────────────┘
                               │
                               ▼
┌───────────────────────────────────────────────────────────────────┐
│                    Load Balancer (Nginx/ALB)                       │
│                   SSL Termination / Rate Limiting                  │
└───────────────────────────────────────────────────────────────────┘
                               │
                               ▼
┌───────────────────────────────────────────────────────────────────┐
│                      Application Layer (K8s)                       │
│  ┌───────────────────────────────────────────────────────────┐   │
│  │              NestJS Application (Pods x 3)                 │   │
│  │                                                             │   │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐    │   │
│  │  │  Auth    │ │   RBAC   │ │   User   │ │  Audit   │    │   │
│  │  │  Module  │ │  Module  │ │  Module  │ │  Module  │    │   │
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────┘    │   │
│  │                                                             │   │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐    │   │
│  │  │  Config  │ │  Monitor │ │ Notifica │ │Analytics │    │   │
│  │  │  Module  │ │  Module  │ │ Module   │ │  Module  │    │   │
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────┘    │   │
│  │                                                             │   │
│  │             ┌─────────────────────────────┐               │   │
│  │             │   Cross-Cutting Concerns    │               │   │
│  │             │  • Logging (Winston)        │               │   │
│  │             │  • Metrics (Prometheus)     │               │   │
│  │             │  • Tracing (Jaeger)         │               │   │
│  │             │  • Exception Handling       │               │   │
│  │             └─────────────────────────────┘               │   │
│  └───────────────────────────────────────────────────────────┘   │
└───────────────────────────────────────────────────────────────────┘
                │              │              │              │
                ▼              ▼              ▼              ▼
┌───────────────────────────────────────────────────────────────────┐
│                       Data & Cache Layer                           │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐           │
│  │  PostgreSQL  │  │     Redis    │  │     BullMQ   │           │
│  │   Primary    │  │   Cluster    │  │   (Queue)    │           │
│  │              │  │              │  │              │           │
│  │  • Master    │  │  • Cache     │  │  • SMS Jobs  │           │
│  │  • Replica   │  │  • Session   │  │  • Email     │           │
│  │    (Read)    │  │  • Locks     │  │  • Reports   │           │
│  └──────────────┘  └──────────────┘  └──────────────┘           │
└───────────────────────────────────────────────────────────────────┘
                │              │              │
                ▼              ▼              ▼
┌───────────────────────────────────────────────────────────────────┐
│                       Storage & Search Layer                       │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐           │
│  │      S3      │  │ Elasticsearch│  │   TimescaleDB│           │
│  │   (Files)    │  │  (Logs/Search) │  │  (Metrics) │           │
│  │              │  │              │  │              │           │
│  │  • Images    │  │  • Audit Logs│  │  • Analytics │           │
│  │  • Uploads   │  │  • Full-text │  │  • Time-series│          │
│  │  • Backups   │  │  • APM Data  │  │              │           │
│  └──────────────┘  └──────────────┘  └──────────────┘           │
└───────────────────────────────────────────────────────────────────┘
                               │
                               ▼
┌───────────────────────────────────────────────────────────────────┐
│                   Monitoring & Observability                       │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐           │
│  │  Prometheus  │  │   Grafana    │  │    Sentry    │           │
│  │  (Metrics)   │  │ (Dashboards) │  │   (Errors)   │           │
│  └──────────────┘  └──────────────┘  └──────────────┘           │
│                                                                     │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐           │
│  │     Loki     │  │  AlertManager│  │  PagerDuty   │           │
│  │    (Logs)    │  │   (Alerts)   │  │  (On-call)   │           │
│  └──────────────┘  └──────────────┘  └──────────────┘           │
└───────────────────────────────────────────────────────────────────┘
                               │
                               ▼
┌───────────────────────────────────────────────────────────────────┐
│                     External Services                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐           │
│  │  Alibaba SMS │  │   SendGrid   │  │    OAuth     │           │
│  │   (China)    │  │   (Email)    │  │  Providers   │           │
│  └──────────────┘  └──────────────┘  └──────────────┘           │
└───────────────────────────────────────────────────────────────────┘
```

### Request Flow Diagram

```
User Request
     │
     ▼
┌─────────────────┐
│   CDN / WAF     │ ◄── DDoS protection, SSL
└─────────────────┘
     │
     ▼
┌─────────────────┐
│ Load Balancer   │ ◄── Distribute traffic
└─────────────────┘
     │
     ▼
┌─────────────────────────────────────────┐
│    NestJS Application                   │
│                                         │
│  1. Guards                              │
│     ├── AuthGuard (JWT validation)      │
│     └── PermissionGuard (RBAC check)    │
│           │                             │
│           ▼                             │
│  2. Interceptors                        │
│     ├── LoggingInterceptor              │
│     ├── CacheInterceptor (Redis)        │
│     └── TransformInterceptor            │
│           │                             │
│           ▼                             │
│  3. Pipes                               │
│     └── ValidationPipe (DTO validation) │
│           │                             │
│           ▼                             │
│  4. Controller                          │
│     └── Route Handler                   │
│           │                             │
│           ▼                             │
│  5. Service Layer                       │
│     ├── Business Logic                  │
│     └── Repository Calls                │
│           │                             │
│           ▼                             │
│  6. Data Layer                          │
│     ├── Prisma ORM                      │
│     ├── Redis Cache                     │
│     └── Message Queue                   │
│           │                             │
│           ▼                             │
│  7. Interceptors (Response)             │
│     ├── TransformInterceptor            │
│     └── AuditInterceptor (Log action)   │
└─────────────────────────────────────────┘
     │
     ▼
Response to User
```

### Deployment Architecture (Kubernetes)

```
┌─────────────────────────────────────────────────────────────────┐
│                     Kubernetes Cluster                          │
│                                                                 │
│  ┌───────────────────── Namespace: production ───────────────┐ │
│  │                                                            │ │
│  │  ┌── Deployment: api-service ──────────────────────────┐  │ │
│  │  │                                                       │  │ │
│  │  │  Pod (Replicas: 3)                                   │  │ │
│  │  │  ┌─────────────────────────────────────────────┐    │  │ │
│  │  │  │  Container: nestjs-app                      │    │  │ │
│  │  │  │  Image: api-service:v1.2.3                  │    │  │ │
│  │  │  │  Resources:                                 │    │  │ │
│  │  │  │    CPU: 500m - 2000m                        │    │  │ │
│  │  │  │    Memory: 512Mi - 2Gi                      │    │  │ │
│  │  │  │  Health: /health/liveness, /health/readiness│    │  │ │
│  │  │  └─────────────────────────────────────────────┘    │  │ │
│  │  │                                                       │  │ │
│  │  │  ┌─────────────────────────────────────────────┐    │  │ │
│  │  │  │  Sidecar: fluent-bit (log forwarding)       │    │  │ │
│  │  │  └─────────────────────────────────────────────┘    │  │ │
│  │  └───────────────────────────────────────────────────┘  │ │
│  │                           │                              │ │
│  │  ┌─────────────────────────────────────────────────┐    │ │
│  │  │  Service: api-service (ClusterIP)              │    │ │
│  │  │  Port: 3000                                     │    │ │
│  │  │  Selector: app=api-service                      │    │ │
│  │  └─────────────────────────────────────────────────┘    │ │
│  │                           │                              │ │
│  │  ┌─────────────────────────────────────────────────┐    │ │
│  │  │  Ingress: api-ingress                           │    │ │
│  │  │  Host: api.example.com                          │    │ │
│  │  │  TLS: letsencrypt-prod                          │    │ │
│  │  │  Annotations:                                   │    │ │
│  │  │    - cert-manager.io/cluster-issuer             │    │ │
│  │  │    - nginx.ingress.kubernetes.io/rate-limit     │    │ │
│  │  └─────────────────────────────────────────────────┘    │ │
│  │                                                          │ │
│  │  ┌─────────────────────────────────────────────────┐    │ │
│  │  │  HPA: api-service-hpa                           │    │ │
│  │  │  Min Replicas: 2                                │    │ │
│  │  │  Max Replicas: 10                               │    │ │
│  │  │  Target CPU: 70%                                │    │ │
│  │  │  Target Memory: 80%                             │    │ │
│  │  └─────────────────────────────────────────────────┘    │ │
│  │                                                          │ │
│  │  ┌─────────────────────────────────────────────────┐    │ │
│  │  │  ConfigMap: app-config                          │    │ │
│  │  │  Secret: app-secrets (sealed-secrets)           │    │ │
│  │  └─────────────────────────────────────────────────┘    │ │
│  └────────────────────────────────────────────────────────┘ │
│                                                               │
│  ┌── StatefulSet: postgresql ────┐  ┌── StatefulSet: redis ┐│
│  │  Replicas: 1 (master)         │  │  Replicas: 3 (cluster)││
│  │  PVC: 100Gi (SSD)             │  │  PVC: 20Gi (SSD)      ││
│  └───────────────────────────────┘  └────────────────────────┘│
└─────────────────────────────────────────────────────────────────┘
```

---

## 7. Module Splitting Strategy

### Domain-Driven Design Approach

Based on the requirements, the system is divided into **6 bounded contexts**:

### 1. Identity & Access Management (IAM) Domain

**Responsibility:** Authentication, Authorization, RBAC

```typescript
// Module Structure
iam/
├── auth/
│   ├── dto/
│   │   ├── login.dto.ts
│   │   ├── register.dto.ts
│   │   └── refresh-token.dto.ts
│   ├── strategies/
│   │   ├── jwt.strategy.ts
│   │   ├── jwt-refresh.strategy.ts
│   │   └── local.strategy.ts
│   ├── guards/
│   │   ├── jwt-auth.guard.ts
│   │   └── roles.guard.ts
│   ├── auth.service.ts
│   ├── auth.controller.ts
│   └── auth.module.ts
│
├── rbac/
│   ├── entities/
│   │   ├── role.entity.ts
│   │   ├── permission.entity.ts
│   │   └── user-role.entity.ts
│   ├── casl/
│   │   ├── casl-ability.factory.ts
│   │   └── policies/
│   │       ├── user.policy.ts
│   │       └── admin.policy.ts
│   ├── decorators/
│   │   ├── roles.decorator.ts
│   │   └── check-policies.decorator.ts
│   ├── rbac.service.ts
│   └── rbac.module.ts
│
├── user/
│   ├── entities/
│   │   ├── user.entity.ts
│   │   ├── user-profile.entity.ts
│   │   └── user-session.entity.ts
│   ├── dto/
│   │   ├── create-user.dto.ts
│   │   ├── update-user.dto.ts
│   │   └── user-filter.dto.ts
│   ├── user.service.ts
│   ├── user.controller.ts
│   └── user.module.ts
│
└── security/
    ├── ip-restriction.service.ts
    ├── rate-limiter.service.ts
    ├── two-factor-auth.service.ts
    └── security.module.ts
```

**Database Tables:**
```sql
-- Key tables for IAM domain
users (id, email, password_hash, status, created_at, updated_at)
roles (id, name, description, is_active, created_at)
permissions (id, name, resource, action, description)
role_permissions (role_id, permission_id)
user_roles (user_id, role_id, assigned_at, assigned_by)
user_sessions (id, user_id, token_hash, ip_address, expires_at)
security_events (id, user_id, event_type, ip_address, timestamp)
ip_whitelist (id, user_id, ip_address, created_at)
```

**API Endpoints:**
```
POST   /api/v1/auth/login
POST   /api/v1/auth/refresh
POST   /api/v1/auth/logout
POST   /api/v1/auth/2fa/enable
POST   /api/v1/auth/2fa/verify

GET    /api/v1/users
POST   /api/v1/users
GET    /api/v1/users/:id
PATCH  /api/v1/users/:id
DELETE /api/v1/users/:id

GET    /api/v1/roles
POST   /api/v1/roles
PATCH  /api/v1/roles/:id
POST   /api/v1/roles/:id/permissions

GET    /api/v1/permissions
```

---

### 2. Audit & Compliance Domain

**Responsibility:** Operation logging, audit trails, compliance reporting

```typescript
// Module Structure
audit/
├── audit-log/
│   ├── entities/
│   │   ├── audit-log.entity.ts
│   │   └── audit-log-detail.entity.ts
│   ├── dto/
│   │   ├── audit-log-filter.dto.ts
│   │   └── audit-log-export.dto.ts
│   ├── interceptors/
│   │   └── audit-logging.interceptor.ts
│   ├── decorators/
│   │   └── auditable.decorator.ts
│   ├── audit-log.service.ts
│   ├── audit-log.controller.ts
│   └── audit-log.module.ts
│
├── login-log/
│   ├── entities/
│   │   └── login-log.entity.ts
│   ├── login-log.service.ts
│   └── login-log.module.ts
│
└── data-masking/
    ├── masking.service.ts
    └── masking-rules.ts
```

**Database Tables:**
```sql
audit_logs (
  id, user_id, action, resource, resource_id,
  before_data (JSONB), after_data (JSONB),
  ip_address, user_agent, status,
  created_at
) PARTITION BY RANGE (created_at)

login_logs (
  id, user_id, login_type, status, ip_address,
  location, device_info, created_at
)

export_logs (
  id, user_id, resource_type, filter_criteria,
  record_count, file_path, created_at
)
```

**Features:**
- Automatic audit logging via decorators
- Data comparison (before/after)
- Sensitive field masking (email, phone, ID)
- Export to CSV/Excel
- Compliance reports (SOC2, GDPR)

---

### 3. Configuration Management Domain

**Responsibility:** System configuration, feature flags, environment settings

```typescript
// Module Structure
config/
├── system-config/
│   ├── entities/
│   │   ├── system-config.entity.ts
│   │   └── config-history.entity.ts
│   ├── dto/
│   │   ├── update-config.dto.ts
│   │   └── config-validation.dto.ts
│   ├── validators/
│   │   ├── api-config.validator.ts
│   │   ├── sms-config.validator.ts
│   │   └── jwt-config.validator.ts
│   ├── system-config.service.ts
│   └── system-config.controller.ts
│
├── feature-toggle/
│   ├── entities/
│   │   └── feature-flag.entity.ts
│   ├── feature-toggle.service.ts
│   ├── feature-toggle.decorator.ts
│   └── feature-toggle.module.ts
│
└── config-center.module.ts
```

**Configuration Categories:**
```typescript
interface SystemConfig {
  // API Configuration
  api: {
    baseUrl: string;
    apiKey: string;
    fallbackUrl?: string;
    timeout: number;
  };
  
  // SMS Configuration
  sms: {
    enabled: boolean;
    provider: 'alibaba' | 'tencent' | 'twilio';
    devMode: boolean;
    templates: Record<string, string>;
  };
  
  // JWT Configuration
  jwt: {
    userSecret: string;
    adminSecret: string;
    accessTokenTTL: number;
    refreshTokenTTL: number;
    rotationEnabled: boolean;
  };
  
  // Storage Configuration
  storage: {
    provider: 's3' | 'oss' | 'minio';
    bucket: string;
    region: string;
    cdnUrl: string;
  };
  
  // Feature Flags
  features: {
    twoFactorAuth: boolean;
    multiDeviceLogin: boolean;
    canaryRelease: boolean;
  };
}
```

**Database Tables:**
```sql
system_configs (
  id, category, key, value (JSONB),
  encrypted, version, updated_by, updated_at
)

config_history (
  id, config_id, old_value, new_value,
  changed_by, changed_at
)

feature_flags (
  id, name, enabled, rollout_percentage,
  conditions (JSONB), created_at, updated_at
)
```

---

### 4. Notification & Messaging Domain

**Responsibility:** SMS, email, in-app notifications, templates

```typescript
// Module Structure
notification/
├── sms/
│   ├── providers/
│   │   ├── alibaba-sms.provider.ts
│   │   ├── tencent-sms.provider.ts
│   │   └── twilio-sms.provider.ts
│   ├── sms.service.ts
│   └── sms.module.ts
│
├── email/
│   ├── email.service.ts
│   └── email.module.ts
│
├── in-app/
│   ├── entities/
│   │   ├── notification.entity.ts
│   │   └── announcement.entity.ts
│   ├── notification.service.ts
│   ├── notification.gateway.ts (WebSocket)
│   └── in-app.module.ts
│
├── template/
│   ├── entities/
│   │   └── message-template.entity.ts
│   ├── template.service.ts
│   └── template.module.ts
│
└── notification.module.ts
```

**Database Tables:**
```sql
notifications (
  id, user_id, type, title, content,
  read_at, created_at
)

announcements (
  id, title, content, priority, target_roles,
  start_time, end_time, created_by, created_at
)

message_templates (
  id, name, type, content, variables,
  locale, is_active, created_at
)

sms_logs (
  id, phone, template_id, variables, status,
  response, sent_at
)
```

**Features:**
- Multi-provider SMS with fallback
- Template management with variables
- Rate limiting per user
- WebSocket real-time notifications
- Email queue with retry

---

### 5. Analytics & Reporting Domain

**Responsibility:** Dashboards, statistics, reports, data visualization

```typescript
// Module Structure
analytics/
├── dashboard/
│   ├── dashboard.service.ts
│   ├── dashboard.controller.ts
│   └── dashboard.module.ts
│
├── user-analytics/
│   ├── user-analytics.service.ts
│   └── queries/
│       ├── user-growth.query.ts
│       ├── retention.query.ts
│       └── activity.query.ts
│
├── content-analytics/
│   ├── content-analytics.service.ts
│   └── queries/
│       ├── favorite-stats.query.ts
│       └── playlist-stats.query.ts
│
├── report/
│   ├── report.service.ts
│   ├── generators/
│   │   ├── pdf-generator.ts
│   │   └── excel-generator.ts
│   └── report.module.ts
│
└── analytics.module.ts
```

**Metrics Collection:**
```typescript
interface DashboardMetrics {
  users: {
    total: number;
    active: number;
    newToday: number;
    newThisWeek: number;
    growth: number[]; // time-series
  };
  
  content: {
    favorites: number;
    playlists: number;
    historyRecords: number;
  };
  
  system: {
    apiCallCount: number;
    errorRate: number;
    avgResponseTime: number;
  };
}
```

**Database Tables:**
```sql
user_activities (
  id, user_id, activity_type, resource_id,
  created_at
) PARTITION BY RANGE (created_at)

daily_metrics (
  date, metric_type, metric_value,
  dimensions (JSONB)
)

reports (
  id, name, type, parameters, file_path,
  generated_by, generated_at
)
```

---

### 6. Monitoring & Observability Domain

**Responsibility:** Health checks, metrics, alerts, APM

```typescript
// Module Structure
monitoring/
├── health/
│   ├── health.controller.ts
│   ├── indicators/
│   │   ├── database.indicator.ts
│   │   ├── redis.indicator.ts
│   │   ├── disk.indicator.ts
│   │   └── memory.indicator.ts
│   └── health.module.ts
│
├── metrics/
│   ├── metrics.service.ts
│   ├── collectors/
│   │   ├── api-metrics.collector.ts
│   │   ├── business-metrics.collector.ts
│   │   └── custom-metrics.collector.ts
│   └── metrics.controller.ts
│
├── alerting/
│   ├── alerting.service.ts
│   ├── rules/
│   │   ├── error-rate.rule.ts
│   │   ├── response-time.rule.ts
│   │   └── disk-usage.rule.ts
│   └── alerting.module.ts
│
└── monitoring.module.ts
```

**Health Check Endpoints:**
```
GET /health/liveness   ← Kubernetes liveness probe
GET /health/readiness  ← Kubernetes readiness probe
GET /health/detailed   ← Full system health
GET /metrics           ← Prometheus metrics endpoint
```

**Monitored Metrics:**
```typescript
// Application Metrics
- http_requests_total
- http_request_duration_seconds
- http_errors_total
- business_events_total

// System Metrics
- nodejs_heap_size_used_bytes
- nodejs_external_memory_bytes
- process_cpu_seconds_total

// Database Metrics
- db_connections_active
- db_query_duration_seconds
- db_errors_total

// Custom Business Metrics
- user_registrations_total
- sms_sent_total
- sms_failed_total
- login_attempts_total
- login_failures_total
```

---

## 8. RBAC System Architecture

### Recommended Approach: Hybrid Model (RBAC + ABAC)

**Core Concepts:**
- **Role-Based Access Control (RBAC):** Assign permissions to roles
- **Attribute-Based Access Control (ABAC):** Dynamic permissions based on attributes

### Implementation with CASL (Ability-Based Authorization)

#### 1. Permission Model

```typescript
// Permission Structure
interface Permission {
  action: 'create' | 'read' | 'update' | 'delete' | 'manage';
  subject: string; // Resource type (e.g., 'User', 'Role', 'Config')
  conditions?: Record<string, any>; // Dynamic rules
  fields?: string[]; // Field-level permissions
  reason?: string; // Why permission denied (for logging)
}

// Example Permissions
const permissions = [
  // Super Admin: Can do everything
  { action: 'manage', subject: 'all' },
  
  // Admin: Can manage users but not super admins
  { action: 'manage', subject: 'User', conditions: { role: { $ne: 'SUPER_ADMIN' } } },
  { action: 'read', subject: 'AuditLog' },
  
  // Operator: Read-only + limited edit
  { action: 'read', subject: 'User' },
  { action: 'update', subject: 'User', fields: ['status', 'tags'] },
  
  // Auditor: Only view logs
  { action: 'read', subject: 'AuditLog' },
  { action: 'export', subject: 'AuditLog' },
];
```

#### 2. Role Hierarchy

```
┌─────────────────────┐
│   SUPER_ADMIN       │  ← Full system access, cannot be deleted
│   (System Level)    │     Can rotate JWT secrets, manage admins
└─────────────────────┘
          │
          ├─────────────────────────┐
          │                         │
┌─────────────────────┐   ┌─────────────────────┐
│       ADMIN         │   │     AUDITOR         │
│   (Operations)      │   │   (View Only)       │
│                     │   │                     │
│ • User Management   │   │ • View Logs         │
│ • Content Mgmt      │   │ • Export Reports    │
│ • Config (partial)  │   │ • Read Configs      │
└─────────────────────┘   └─────────────────────┘
          │
          │
┌─────────────────────┐
│     OPERATOR        │
│  (Limited Edit)     │
│                     │
│ • View Users        │
│ • Update Status     │
│ • View Dashboard    │
└─────────────────────┘
```

#### 3. Database Schema

```sql
-- Roles table
CREATE TABLE roles (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name VARCHAR(50) UNIQUE NOT NULL,
  display_name VARCHAR(100) NOT NULL,
  description TEXT,
  is_system BOOLEAN DEFAULT FALSE, -- Cannot be deleted
  is_active BOOLEAN DEFAULT TRUE,
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW()
);

-- Permissions table
CREATE TABLE permissions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name VARCHAR(100) UNIQUE NOT NULL,
  action VARCHAR(20) NOT NULL, -- create, read, update, delete, manage
  subject VARCHAR(50) NOT NULL, -- Resource name
  conditions JSONB, -- Dynamic conditions
  fields TEXT[], -- Field-level permissions
  description TEXT,
  created_at TIMESTAMP DEFAULT NOW()
);

-- Role-Permission mapping (many-to-many)
CREATE TABLE role_permissions (
  role_id UUID REFERENCES roles(id) ON DELETE CASCADE,
  permission_id UUID REFERENCES permissions(id) ON DELETE CASCADE,
  granted_at TIMESTAMP DEFAULT NOW(),
  PRIMARY KEY (role_id, permission_id)
);

-- User-Role mapping (many-to-many)
CREATE TABLE user_roles (
  user_id UUID REFERENCES users(id) ON DELETE CASCADE,
  role_id UUID REFERENCES roles(id) ON DELETE RESTRICT,
  assigned_at TIMESTAMP DEFAULT NOW(),
  assigned_by UUID REFERENCES users(id),
  expires_at TIMESTAMP, -- Optional: Temporary role
  PRIMARY KEY (user_id, role_id)
);

-- Data-level permissions (optional for row-level security)
CREATE TABLE data_permissions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  role_id UUID REFERENCES roles(id),
  resource_type VARCHAR(50) NOT NULL,
  scope VARCHAR(20) NOT NULL, -- 'own', 'department', 'all'
  conditions JSONB, -- Additional filters
  created_at TIMESTAMP DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX idx_role_permissions_role ON role_permissions(role_id);
CREATE INDEX idx_user_roles_user ON user_roles(user_id);
CREATE INDEX idx_permissions_subject ON permissions(subject, action);
```

#### 4. CASL Implementation (NestJS)

```typescript
// casl-ability.factory.ts
import { Injectable } from '@nestjs/common';
import { AbilityBuilder, PureAbility } from '@casl/ability';
import { User } from '../user/entities/user.entity';

export type Action = 'create' | 'read' | 'update' | 'delete' | 'manage';
export type Subject = 'User' | 'Role' | 'Config' | 'AuditLog' | 'all';

export type AppAbility = PureAbility<[Action, Subject]>;

@Injectable()
export class CaslAbilityFactory {
  async createForUser(user: User): Promise<AppAbility> {
    const { can, cannot, build } = new AbilityBuilder(PureAbility);

    // Load user's roles and permissions from database
    const roles = await this.getUserRoles(user.id);
    const permissions = await this.getRolePermissions(roles);

    // Build abilities based on permissions
    for (const permission of permissions) {
      if (permission.conditions) {
        can(permission.action, permission.subject, permission.conditions);
      } else {
        can(permission.action, permission.subject);
      }
    }

    // Special rules
    if (user.role === 'SUPER_ADMIN') {
      can('manage', 'all');
    } else {
      // Users cannot modify themselves to SUPER_ADMIN
      cannot('update', 'User', { role: 'SUPER_ADMIN' });
    }

    return build();
  }

  private async getUserRoles(userId: string): Promise<string[]> {
    // Query database for user roles
    return [];
  }

  private async getRolePermissions(roles: string[]): Promise<any[]> {
    // Query database for role permissions
    return [];
  }
}
```

```typescript
// permissions.guard.ts
import { Injectable, CanActivate, ExecutionContext } from '@nestjs/common';
import { Reflector } from '@nestjs/core';
import { CaslAbilityFactory } from './casl-ability.factory';

@Injectable()
export class PermissionsGuard implements CanActivate {
  constructor(
    private reflector: Reflector,
    private caslAbilityFactory: CaslAbilityFactory,
  ) {}

  async canActivate(context: ExecutionContext): Promise<boolean> {
    const requiredPermissions = this.reflector.getAllAndOverride('permissions', [
      context.getHandler(),
      context.getClass(),
    ]);

    if (!requiredPermissions) {
      return true; // No permissions required
    }

    const request = context.switchToHttp().getRequest();
    const user = request.user;

    const ability = await this.caslAbilityFactory.createForUser(user);

    return requiredPermissions.every((permission) =>
      ability.can(permission.action, permission.subject),
    );
  }
}
```

```typescript
// Usage in controller
@Controller('users')
@UseGuards(JwtAuthGuard, PermissionsGuard)
export class UserController {
  @Get()
  @CheckPermissions({ action: 'read', subject: 'User' })
  findAll() {
    return this.userService.findAll();
  }

  @Post()
  @CheckPermissions({ action: 'create', subject: 'User' })
  create(@Body() createUserDto: CreateUserDto) {
    return this.userService.create(createUserDto);
  }

  @Patch(':id')
  @CheckPermissions({ action: 'update', subject: 'User' })
  update(@Param('id') id: string, @Body() updateUserDto: UpdateUserDto) {
    return this.userService.update(id, updateUserDto);
  }
}
```

#### 5. Permission Caching Strategy

```typescript
// Permission caching service
@Injectable()
export class PermissionCacheService {
  constructor(
    @InjectRedis() private redis: Redis,
    private prisma: PrismaService,
  ) {}

  private getCacheKey(userId: string): string {
    return `permissions:user:${userId}`;
  }

  async getUserPermissions(userId: string): Promise<Permission[]> {
    // Try cache first
    const cached = await this.redis.get(this.getCacheKey(userId));
    if (cached) {
      return JSON.parse(cached);
    }

    // Load from database
    const permissions = await this.loadPermissionsFromDB(userId);

    // Cache for 5 minutes
    await this.redis.setex(
      this.getCacheKey(userId),
      300,
      JSON.stringify(permissions),
    );

    return permissions;
  }

  async invalidateUserPermissions(userId: string): Promise<void> {
    await this.redis.del(this.getCacheKey(userId));
  }

  private async loadPermissionsFromDB(userId: string): Promise<Permission[]> {
    return this.prisma.permission.findMany({
      where: {
        rolePermissions: {
          some: {
            role: {
              userRoles: {
                some: { userId },
              },
            },
          },
        },
      },
    });
  }
}
```

#### 6. Menu & Button-Level Permissions

```typescript
// Frontend permission configuration
export const menuPermissions = {
  dashboard: { action: 'read', subject: 'Dashboard' },
  users: { action: 'read', subject: 'User' },
  'users.create': { action: 'create', subject: 'User' },
  'users.edit': { action: 'update', subject: 'User' },
  'users.delete': { action: 'delete', subject: 'User' },
  roles: { action: 'read', subject: 'Role' },
  'roles.manage': { action: 'manage', subject: 'Role' },
  auditLogs: { action: 'read', subject: 'AuditLog' },
  'auditLogs.export': { action: 'export', subject: 'AuditLog' },
  config: { action: 'read', subject: 'Config' },
  'config.edit': { action: 'update', subject: 'Config' },
};

// React component with permission check
import { useAbility } from '@casl/react';

function UserListPage() {
  const ability = useAbility(AbilityContext);

  return (
    <div>
      <h1>Users</h1>
      {ability.can('create', 'User') && (
        <Button onClick={openCreateModal}>Create User</Button>
      )}
      <Table dataSource={users}>
        <Column title="Name" dataIndex="name" />
        <Column
          title="Actions"
          render={(_, record) => (
            <>
              {ability.can('update', 'User') && (
                <Button onClick={() => edit(record)}>Edit</Button>
              )}
              {ability.can('delete', 'User') && (
                <Button danger onClick={() => delete(record)}>Delete</Button>
              )}
            </>
          )}
        />
      </Table>
    </div>
  );
}
```

---

## 9. Infrastructure Components

### 9.1 Configuration Center Implementation

#### Option A: Database-Backed (Recommended for Modular Monolith)

**Architecture:**
```
┌─────────────────────────────────────────────┐
│          Application Instances              │
│  ┌────────┐  ┌────────┐  ┌────────┐        │
│  │ App 1  │  │ App 2  │  │ App 3  │        │
│  └────────┘  └────────┘  └────────┘        │
│       │           │           │             │
│       └───────────┼───────────┘             │
│                   │                         │
│                   ▼                         │
│  ┌─────────────────────────────────────┐   │
│  │   Config Service (In-App)           │   │
│  │   • Hot Reload via Server-Sent      │   │
│  │     Events (SSE)                    │   │
│  │   • Local Cache (5 min TTL)         │   │
│  │   • Change Detection                │   │
│  └─────────────────────────────────────┘   │
└─────────────────────────────────────────────┘
                    │
                    ▼
        ┌──────────────────────┐
        │   PostgreSQL DB      │
        │   system_configs     │
        │   config_history     │
        └──────────────────────┘
```

**Implementation:**
```typescript
// config-center.service.ts
@Injectable()
export class ConfigCenterService {
  private configCache = new Map<string, any>();
  private lastSync: Date;

  constructor(
    private prisma: PrismaService,
    @InjectRedis() private redis: Redis,
    private eventEmitter: EventEmitter2,
  ) {
    this.initializeWatcher();
  }

  /**
   * Get configuration value with caching
   */
  async get<T>(key: string, defaultValue?: T): Promise<T> {
    // Check cache first
    if (this.configCache.has(key)) {
      return this.configCache.get(key);
    }

    // Load from database
    const config = await this.prisma.systemConfig.findUnique({
      where: { key },
    });

    const value = config?.value ?? defaultValue;
    this.configCache.set(key, value);

    return value;
  }

  /**
   * Update configuration (with version control)
   */
  async set(key: string, value: any, userId: string): Promise<void> {
    const currentConfig = await this.prisma.systemConfig.findUnique({
      where: { key },
    });

    // Save history
    if (currentConfig) {
      await this.prisma.configHistory.create({
        data: {
          configId: currentConfig.id,
          oldValue: currentConfig.value,
          newValue: value,
          changedBy: userId,
        },
      });
    }

    // Update config
    await this.prisma.systemConfig.upsert({
      where: { key },
      create: {
        key,
        value,
        updatedBy: userId,
      },
      update: {
        value,
        version: { increment: 1 },
        updatedBy: userId,
        updatedAt: new Date(),
      },
    });

    // Invalidate cache
    this.configCache.delete(key);

    // Publish change event (for multi-instance sync)
    await this.redis.publish('config:changed', JSON.stringify({ key, value }));

    // Emit local event
    this.eventEmitter.emit('config.changed', { key, value });
  }

  /**
   * Watch for configuration changes (SSE for real-time updates)
   */
  private initializeWatcher(): void {
    // Subscribe to Redis pub/sub for config changes
    this.redis.subscribe('config:changed');
    this.redis.on('message', (channel, message) => {
      if (channel === 'config:changed') {
        const { key } = JSON.parse(message);
        this.configCache.delete(key);
        this.eventEmitter.emit('config.changed', JSON.parse(message));
      }
    });

    // Periodic sync (fallback)
    setInterval(() => this.syncAll(), 60000); // 1 minute
  }

  /**
   * Get all configurations (admin interface)
   */
  async getAllConfigs(): Promise<SystemConfig[]> {
    return this.prisma.systemConfig.findMany({
      orderBy: { category: 'asc' },
    });
  }

  /**
   * Validate configuration before saving
   */
  async validate(key: string, value: any): Promise<boolean> {
    // Load validator based on config category
    const validator = this.getValidator(key);
    return validator.validate(value);
  }
}
```

**Frontend SSE Integration:**
```typescript
// React hook for real-time config updates
export function useConfigSubscription() {
  useEffect(() => {
    const eventSource = new EventSource('/api/v1/config/stream');

    eventSource.addEventListener('config-changed', (event) => {
      const { key, value } = JSON.parse(event.data);
      console.log(`Config updated: ${key} = ${value}`);
      
      // Trigger app refresh or show notification
      queryClient.invalidateQueries(['config', key]);
    });

    return () => eventSource.close();
  }, []);
}
```

#### Option B: Dedicated Config Service (For Microservices)

Consider using **Consul**, **etcd**, or **Apollo Config** for distributed systems.

---

### 9.2 Logging System Implementation

#### Architecture: Centralized Logging with ELK Stack

```
┌─────────────────────────────────────────────────────────────┐
│              Application Instances (NestJS)                  │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  Winston Logger                                        │ │
│  │  • Console Transport (dev)                             │ │
│  │  • File Transport (local logs)                         │ │
│  │  • Elasticsearch Transport (production)                │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
                 ┌─────────────────────┐
                 │   Logstash          │ ← Parse, filter, enrich
                 │   (Optional)        │
                 └─────────────────────┘
                            │
                            ▼
                 ┌─────────────────────┐
                 │   Elasticsearch     │ ← Store & index logs
                 │   Index per day     │
                 └─────────────────────┘
                            │
                            ▼
                 ┌─────────────────────┐
                 │      Kibana         │ ← Visualize & query
                 │   (Dashboard)       │
                 └─────────────────────┘
```

**Implementation:**

```typescript
// logger.service.ts
import * as winston from 'winston';
import { ElasticsearchTransport } from 'winston-elasticsearch';

@Injectable()
export class LoggerService {
  private logger: winston.Logger;

  constructor(private configService: ConfigService) {
    this.logger = winston.createLogger({
      level: this.configService.get('LOG_LEVEL', 'info'),
      format: winston.format.combine(
        winston.format.timestamp(),
        winston.format.errors({ stack: true }),
        winston.format.splat(),
        winston.format.json(),
      ),
      defaultMeta: {
        service: 'admin-api',
        environment: this.configService.get('NODE_ENV'),
      },
      transports: [
        // Console (development)
        new winston.transports.Console({
          format: winston.format.combine(
            winston.format.colorize(),
            winston.format.simple(),
          ),
        }),

        // File (rotating)
        new winston.transports.File({
          filename: 'logs/error.log',
          level: 'error',
          maxsize: 10485760, // 10MB
          maxFiles: 10,
        }),

        // Elasticsearch (production)
        new ElasticsearchTransport({
          level: 'info',
          clientOpts: {
            node: this.configService.get('ELASTICSEARCH_URL'),
          },
          index: 'app-logs',
        }),
      ],
    });
  }

  log(message: string, context?: string, meta?: any) {
    this.logger.info(message, { context, ...meta });
  }

  error(message: string, trace?: string, context?: string, meta?: any) {
    this.logger.error(message, { trace, context, ...meta });
  }

  warn(message: string, context?: string, meta?: any) {
    this.logger.warn(message, { context, ...meta });
  }

  debug(message: string, context?: string, meta?: any) {
    this.logger.debug(message, { context, ...meta });
  }
}
```

**Log Structure (JSON):**
```json
{
  "timestamp": "2026-02-26T10:30:45.123Z",
  "level": "info",
  "message": "User login successful",
  "service": "admin-api",
  "environment": "production",
  "context": "AuthService",
  "userId": "uuid-123",
  "ip": "192.168.1.1",
  "userAgent": "Mozilla/5.0...",
  "requestId": "req-abc123",
  "duration": 245,
  "method": "POST",
  "path": "/api/v1/auth/login",
  "statusCode": 200
}
```

**Audit Logging Interceptor:**
```typescript
@Injectable()
export class AuditLoggingInterceptor implements NestInterceptor {
  constructor(
    private auditLogService: AuditLogService,
    private logger: LoggerService,
  ) {}

  intercept(context: ExecutionContext, next: CallHandler): Observable<any> {
    const request = context.switchToHttp().getRequest();
    const { user, method, url, body, ip } = request;

    const actionName = context.getHandler().name;
    const controllerName = context.getClass().name;

    // Capture before state
    const beforeData = this.captureState(request);

    return next.handle().pipe(
      tap(async (response) => {
        // Capture after state
        const afterData = response;

        // Log to audit table
        await this.auditLogService.create({
          userId: user?.id,
          action: actionName,
          resource: controllerName,
          beforeData,
          afterData,
          ip,
          userAgent: request.headers['user-agent'],
          status: 'success',
        });

        this.logger.log(`Audit: ${method} ${url}`, 'AuditInterceptor', {
          user: user?.id,
          action: actionName,
        });
      }),
      catchError((error) => {
        // Log error
        this.auditLogService.create({
          userId: user?.id,
          action: actionName,
          resource: controllerName,
          ip,
          status: 'failed',
          errorMessage: error.message,
        });

        throw error;
      }),
    );
  }

  private captureState(request: any): any {
    // Capture relevant state (e.g., entity before update)
    return {};
  }
}
```

**Log Retention Policy:**
```yaml
Elasticsearch Index Lifecycle Management (ILM):
  Hot Phase:
    - Keep logs for 7 days
    - Enable full indexing and search
  
  Warm Phase:
    - After 7 days, move to warm nodes
    - Reduce replica count
    - Force merge to reduce segments
  
  Cold Phase:
    - After 30 days, move to cold storage
    - Make read-only
    - Searchable snapshot
  
  Delete Phase:
    - After 90 days, delete indices
```

---

### 9.3 Monitoring System Implementation

#### Architecture: Prometheus + Grafana

```
┌─────────────────────────────────────────────────────────┐
│              Application (NestJS)                        │
│  ┌────────────────────────────────────────────────────┐ │
│  │  Prometheus Client                                 │ │
│  │  • Counter (login_attempts_total)                  │ │
│  │  • Gauge (active_users)                            │ │
│  │  • Histogram (http_request_duration_seconds)       │ │
│  │  • Summary (db_query_duration_seconds)             │ │
│  └────────────────────────────────────────────────────┘ │
│                 Expose /metrics endpoint                 │
└─────────────────────────────────────────────────────────┘
                           │
                           ▼
                ┌──────────────────────┐
                │    Prometheus        │ ← Scrape metrics every 15s
                │    (Time-series DB)  │
                └──────────────────────┘
                           │
                           ▼
                ┌──────────────────────┐
                │   AlertManager       │ ← Send alerts
                │   • Email            │
                │   • Slack            │
                │   • PagerDuty        │
                └──────────────────────┘
                           │
                           ▼
                ┌──────────────────────┐
                │      Grafana         │ ← Visualize metrics
                │    (Dashboards)      │
                └──────────────────────┘
```

**Implementation:**

```typescript
// metrics.service.ts
import { Injectable } from '@nestjs/common';
import { Registry, Counter, Gauge, Histogram } from 'prom-client';

@Injectable()
export class MetricsService {
  private registry: Registry;

  // Counters
  private httpRequestsTotal: Counter;
  private loginAttemptsTotal: Counter;
  private smseSentTotal: Counter;

  // Gauges
  private activeUsers: Gauge;
  private dbConnectionsActive: Gauge;

  // Histograms
  private httpRequestDuration: Histogram;
  private dbQueryDuration: Histogram;

  constructor() {
    this.registry = new Registry();
    this.initializeMetrics();
  }

  private initializeMetrics() {
    // HTTP Request Counter
    this.httpRequestsTotal = new Counter({
      name: 'http_requests_total',
      help: 'Total number of HTTP requests',
      labelNames: ['method', 'route', 'status_code'],
      registers: [this.registry],
    });

    // HTTP Request Duration
    this.httpRequestDuration = new Histogram({
      name: 'http_request_duration_seconds',
      help: 'HTTP request duration in seconds',
      labelNames: ['method', 'route', 'status_code'],
      buckets: [0.1, 0.3, 0.5, 0.7, 1, 3, 5, 7, 10],
      registers: [this.registry],
    });

    // Login Attempts
    this.loginAttemptsTotal = new Counter({
      name: 'login_attempts_total',
      help: 'Total number of login attempts',
      labelNames: ['status'], // success or failed
      registers: [this.registry],
    });

    // SMS Sent
    this.smsSentTotal = new Counter({
      name: 'sms_sent_total',
      help: 'Total number of SMS messages sent',
      labelNames: ['provider', 'status'],
      registers: [this.registry],
    });

    // Active Users
    this.activeUsers = new Gauge({
      name: 'active_users',
      help: 'Number of currently active users',
      registers: [this.registry],
    });

    // Database Connections
    this.dbConnectionsActive = new Gauge({
      name: 'db_connections_active',
      help: 'Number of active database connections',
      registers: [this.registry],
    });

    // DB Query Duration
    this.dbQueryDuration = new Histogram({
      name: 'db_query_duration_seconds',
      help: 'Database query duration in seconds',
      labelNames: ['operation', 'table'],
      buckets: [0.01, 0.05, 0.1, 0.3, 0.5, 1, 3, 5],
      registers: [this.registry],
    });
  }

  // Record HTTP request
  recordHttpRequest(method: string, route: string, statusCode: number, duration: number) {
    this.httpRequestsTotal.inc({ method, route, status_code: statusCode });
    this.httpRequestDuration.observe({ method, route, status_code: statusCode }, duration);
  }

  // Record login attempt
  recordLoginAttempt(status: 'success' | 'failed') {
    this.loginAttemptsTotal.inc({ status });
  }

  // Record SMS sent
  recordSmsSent(provider: string, status: 'success' | 'failed') {
    this.smsSentTotal.inc({ provider, status });
  }

  // Update active users
  setActiveUsers(count: number) {
    this.activeUsers.set(count);
  }

  // Get metrics for Prometheus
  async getMetrics(): Promise<string> {
    return this.registry.metrics();
  }
}
```

**Metrics Controller:**
```typescript
@Controller('metrics')
export class MetricsController {
  constructor(private metricsService: MetricsService) {}

  @Get()
  async getMetrics(): Promise<string> {
    return this.metricsService.getMetrics();
  }
}
```

**Prometheus Configuration (prometheus.yml):**
```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'admin-api'
    static_configs:
      - targets: ['localhost:3000']
    metrics_path: '/metrics'
    scrape_interval: 10s
```

**Alert Rules (alerts.yml):**
```yaml
groups:
  - name: api_alerts
    interval: 30s
    rules:
      # High Error Rate
      - alert: HighErrorRate
        expr: rate(http_requests_total{status_code=~"5.."}[5m]) > 0.05
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "High error rate detected"
          description: "Error rate is {{ $value }} requests/sec"

      # Slow Response Time
      - alert: SlowResponseTime
        expr: histogram_quantile(0.95, http_request_duration_seconds_bucket) > 1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Slow response time (P95 > 1s)"

      # High Failed Login Attempts
      - alert: HighFailedLogins
        expr: rate(login_attempts_total{status="failed"}[5m]) > 10
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Potential brute force attack"

      # Database Connection Pool Exhausted
      - alert: DbConnectionPoolExhausted
        expr: db_connections_active > 45
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "Database connection pool nearly exhausted"

      # SMS Delivery Failure
      - alert: HighSmsFailureRate
        expr: rate(sms_sent_total{status="failed"}[10m]) / rate(sms_sent_total[10m]) > 0.1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High SMS failure rate (>10%)"
```

**Grafana Dashboard Configuration:**
```json
{
  "dashboard": {
    "title": "Admin API Monitoring",
    "panels": [
      {
        "title": "Request Rate",
        "targets": [
          {
            "expr": "rate(http_requests_total[5m])"
          }
        ]
      },
      {
        "title": "P95 Response Time",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))"
          }
        ]
      },
      {
        "title": "Error Rate",
        "targets": [
          {
            "expr": "rate(http_requests_total{status_code=~\"5..\"}[5m])"
          }
        ]
      },
      {
        "title": "Active Users",
        "targets": [
          {
            "expr": "active_users"
          }
        ]
      }
    ]
  }
}
```

---

## 10. Deployment Strategy

### Environment Setup

```yaml
Environments:
  Development:
    - Local Docker Compose
    - Hot reload enabled
    - Debug logging
    - Mock external services
  
  Staging:
    - Kubernetes cluster (single node)
    - Production-like setup
    - Integration with external services
    - Load testing
  
  Production:
    - Kubernetes cluster (multi-node)
    - High availability (3 replicas)
    - Auto-scaling enabled
    - Full monitoring
```

### CI/CD Pipeline (GitHub Actions)

```yaml
# .github/workflows/deploy.yml
name: Deploy

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Setup Node.js
        uses: actions/setup-node@v3
        with:
          node-version: '20'
          cache: 'npm'
      
      - name: Install dependencies
        run: npm ci
      
      - name: Lint
        run: npm run lint
      
      - name: Unit Tests
        run: npm run test:cov
      
      - name: E2E Tests
        run: npm run test:e2e

  build:
    needs: test
    runs-on: ubuntu-latest
    if: github.event_name == 'push'
    steps:
      - uses: actions/checkout@v3
      
      - name: Build Docker Image
        run: |
          docker build -t admin-api:${{ github.sha }} .
          docker tag admin-api:${{ github.sha }} admin-api:latest
      
      - name: Push to Registry
        run: |
          echo ${{ secrets.DOCKER_PASSWORD }} | docker login -u ${{ secrets.DOCKER_USERNAME }} --password-stdin
          docker push admin-api:${{ github.sha }}
          docker push admin-api:latest

  deploy-staging:
    needs: build
    runs-on: ubuntu-latest
    if: github.ref == 'refs/heads/develop'
    steps:
      - name: Deploy to Staging
        run: |
          kubectl set image deployment/admin-api admin-api=admin-api:${{ github.sha }} -n staging
          kubectl rollout status deployment/admin-api -n staging

  deploy-production:
    needs: build
    runs-on: ubuntu-latest
    if: github.ref == 'refs/heads/main'
    environment: production
    steps:
      - name: Deploy to Production
        run: |
          kubectl set image deployment/admin-api admin-api=admin-api:${{ github.sha }} -n production
          kubectl rollout status deployment/admin-api -n production
      
      - name: Run Smoke Tests
        run: |
          curl -f https://api.example.com/health || exit 1
```

---

## 11. Development Roadmap

### Phase 1: Foundation (Weeks 1-4)

- ✅ Project setup & repository structure
- ✅ Database schema design
- ✅ Authentication & JWT implementation
- ✅ Basic RBAC system
- ✅ User management CRUD
- ✅ Audit logging interceptor
- ✅ Health checks
- ✅ CI/CD pipeline
- ✅ Development environment (Docker Compose)

### Phase 2: Core Features (Weeks 5-8)

- ✅ Advanced RBAC (CASL integration)
- ✅ Configuration center
- ✅ Message center (SMS/Email)
- ✅ File upload & management
- ✅ Dashboard & analytics
- ✅ System monitoring setup
- ✅ Frontend admin panel
- ✅ API documentation (Swagger)

### Phase 3: Security & Performance (Weeks 9-10)

- ✅ Security hardening
- ✅ Rate limiting & DDoS protection
- ✅ Performance optimization
- ✅ Caching strategy implementation
- ✅ Load testing & tuning
- ✅ Database optimization (indexes, queries)

### Phase 4: Operations & Deployment (Weeks 11-12)

- ✅ Kubernetes deployment
- ✅ Monitoring & alerting setup
- ✅ Backup & disaster recovery
- ✅ Documentation
- ✅ Staging environment setup
- ✅ Production deployment

### Phase 5: Advanced Features (Post-MVP)

- 🔲 Two-factor authentication
- 🔲 Advanced analytics & reports
- 🔲 Feature toggle system
- 🔲 Canary deployment
- 🔲 Multi-language support (i18n)
- 🔲 Dark mode
- 🔲 Mobile app support

---

## 12. Cost Estimation

### Infrastructure Costs (Monthly, USD)

```yaml
Small Setup (< 10K users):
  Compute: $50 (1x 2 vCPU, 4GB RAM)
  Database: $50 (PostgreSQL, 20GB SSD)
  Redis: $20 (1GB RAM)
  Storage: $10 (S3, 100GB)
  CDN: $20 (1TB traffic)
  Monitoring: $10 (Self-hosted)
  Total: ~$160/month

Medium Setup (10K - 100K users):
  Compute: $200 (3x 4 vCPU, 8GB RAM)
  Database: $150 (PostgreSQL with replica)
  Redis: $50 (Redis Cluster, 4GB)
  Storage: $30 (S3, 500GB)
  CDN: $50 (5TB traffic)
  Monitoring: $50 (DataDog basic)
  Load Balancer: $20
  Total: ~$550/month

Large Setup (100K - 1M users):
  Compute: $800 (10x 8 vCPU, 16GB RAM + HPA)
  Database: $500 (PostgreSQL HA + replicas)
  Redis: $150 (Redis Cluster, 16GB)
  Storage: $100 (S3, 2TB)
  CDN: $200 (20TB traffic)
  Monitoring: $200 (DataDog/New Relic pro)
  Load Balancer: $50
  Elasticsearch: $300 (3-node cluster)
  Total: ~$2,300/month
```

### Team Costs (Monthly, USD)

```yaml
Small Team (3-5 developers):
  1x Senior Full-Stack: $10,000
  2x Mid-level Developers: $12,000
  1x DevOps Engineer (part-time): $5,000
  Total: ~$27,000/month

Medium Team (5-8 developers):
  1x Tech Lead: $12,000
  2x Senior Developers: $18,000
  3x Mid-level Developers: $18,000
  1x DevOps Engineer: $8,000
  1x QA Engineer: $6,000
  Total: ~$62,000/month
```

---

## 13. Conclusion

### Summary

This architecture plan provides a **production-ready, scalable foundation** for an enterprise admin backend system. The recommended **Modular Monolithic architecture with NestJS** offers:

✅ **Optimal balance** between complexity and effectiveness  
✅ **Fast development** with TypeScript ecosystem  
✅ **Clear module boundaries** for future scalability  
✅ **Proven technology stack** with strong community support  
✅ **Cost-effective** infrastructure requirements  
✅ **Future-proof** migration path to microservices

### Key Success Factors

1. **Start Simple:** Begin with modular monolith, scale when needed
2. **Emphasis on Observability:** Comprehensive logging, metrics, and monitoring from day one
3. **Security First:** Implement security best practices early
4. **Automated Testing:** Maintain >80% code coverage
5. **Documentation:** Keep architecture decisions and APIs well-documented
6. **Incremental Delivery:** Deploy features in small, testable increments

### Next Steps

1. **Week 1:** Team kickoff, environment setup, repository creation
2. **Week 2:** Database schema design, authentication module
3. **Week 3:** RBAC implementation, user management
4. **Week 4:** Audit logging, monitoring setup
5. **Ongoing:** Iterative development following roadmap

---

**Document Approval:**

- [ ] Technical Lead
- [ ] Product Manager
- [ ] DevOps Lead
- [ ] Security Team

**Last Updated:** February 26, 2026  
**Version:** 1.0  
**Contact:** architecture-team@example.com
