# Navigator Documentation Plan

## Overview
Create a comprehensive MkDocs-based documentation site hosted on GitHub Pages with focused, practical examples for Navigator users.

## Documentation Structure

```
docs/
├── index.md                    # Landing page with quick start
├── installation/
│   ├── index.md                # Installation overview
│   ├── binary.md               # Using pre-built binaries
│   ├── source.md               # Building from source
│   └── docker.md               # Docker deployment
├── getting-started/
│   ├── index.md                # Getting started overview
│   ├── first-app.md            # Your first Rails app
│   ├── basic-config.md         # Basic configuration
│   └── testing.md              # Testing your setup
├── configuration/
│   ├── index.md                # Configuration overview
│   ├── yaml-reference.md       # Complete YAML reference
│   ├── server.md               # Server settings
│   ├── applications.md         # Application configuration
│   ├── authentication.md       # Auth and htpasswd setup
│   ├── static-files.md         # Static file serving
│   ├── routing.md              # URL routing and rewrites
│   ├── processes.md            # Managed processes
│   └── templates.md            # Variable substitution
├── examples/
│   ├── index.md                # Examples overview
│   ├── single-tenant.md        # Single Rails app
│   ├── multi-tenant.md         # Multi-tenant setup
│   ├── with-redis.md           # Rails with Redis
│   ├── with-sidekiq.md         # Background jobs
│   ├── action-cable.md         # WebSocket support
│   ├── static-site.md          # Serving static sites
│   ├── mixed-apps.md           # Multiple app types
│   ├── fly-deployment.md       # Fly.io deployment
│   ├── systemd.md              # Systemd service
│   └── docker-compose.md       # Docker Compose setup
├── features/
│   ├── index.md                # Features overview
│   ├── process-management.md   # Process lifecycle
│   ├── port-allocation.md      # Dynamic ports
│   ├── try-files.md            # Try files behavior
│   ├── authentication.md       # Auth patterns
│   ├── fly-replay.md           # Fly-Replay routing
│   ├── machine-suspend.md      # Auto-suspension
│   ├── hot-reload.md           # Configuration reload
│   └── logging.md              # Logging and debugging
├── deployment/
│   ├── index.md                # Deployment overview
│   ├── production.md           # Production best practices
│   ├── fly-io.md               # Fly.io deployment
│   ├── aws.md                  # AWS deployment
│   ├── digitalocean.md         # DigitalOcean setup
│   └── monitoring.md           # Monitoring and alerts
├── migration/
│   ├── index.md                # Migration overview
│   ├── from-nginx.md           # From nginx/Passenger
│   ├── from-puma.md            # From standalone Puma
│   └── from-apache.md          # From Apache
├── troubleshooting/
│   ├── index.md                # Troubleshooting overview
│   ├── common-issues.md        # Common problems
│   ├── port-conflicts.md       # Port issues
│   ├── process-errors.md       # Process failures
│   └── debugging.md            # Debug techniques
├── reference/
│   ├── index.md                # Reference overview
│   ├── cli.md                  # Command-line options
│   ├── environment.md          # Environment variables
│   ├── signals.md              # Signal handling
│   └── api.md                  # Internal APIs
└── cookbook/
    ├── index.md                # Cookbook overview
    ├── ssl-termination.md      # SSL with reverse proxy
    ├── load-balancing.md       # Load balancer setup
    ├── blue-green.md           # Blue-green deployment
    ├── rate-limiting.md        # Rate limiting setup
    └── custom-errors.md        # Custom error pages
```

## Example Content Structure

### Each example should include:
1. **Scenario Description** - What problem it solves
2. **Configuration File** - Complete, working YAML
3. **Setup Steps** - Clear, numbered instructions
4. **Testing Commands** - How to verify it works
5. **Common Variations** - Alternative approaches
6. **Troubleshooting** - What might go wrong

## Small, Focused Examples

### 1. Single Rails Application
```yaml
# Simplest possible setup
server:
  listen: 3000
  public_dir: ./public

applications:
  tenants:
    - name: myapp
      path: /
      working_dir: /path/to/rails/app
```

### 2. Rails with Static Assets
```yaml
# Serve assets directly, bypass Rails
server:
  listen: 3000
  public_dir: ./public

static:
  directories:
    - path: /assets/
      root: public/assets/
      cache: 86400
  extensions: [css, js, png, jpg]

applications:
  tenants:
    - name: myapp
      path: /
      working_dir: /path/to/rails/app
```

### 3. Multi-Tenant by Subdomain
```yaml
# Different apps on different paths
applications:
  tenants:
    - name: boston
      path: /boston/
      database: boston_db
      
    - name: chicago
      path: /chicago/
      database: chicago_db
```

### 4. Rails with Redis
```yaml
# Start Redis alongside Rails
managed_processes:
  - name: redis
    command: redis-server
    auto_restart: true

applications:
  global_env:
    REDIS_URL: redis://localhost:6379
  tenants:
    - name: myapp
      path: /
```

### 5. WebSocket Support
```yaml
# Action Cable on separate path
applications:
  tenants:
    - name: main
      path: /
      
    - name: cable
      path: /cable
      force_max_concurrent_requests: 0
```

### 6. Protected Admin Area
```yaml
# Different auth for admin
auth:
  enabled: true
  htpasswd: ./htpasswd
  public_paths:
    - /assets/
    - /api/

applications:
  tenants:
    - name: app
      path: /
    - name: admin
      path: /admin/
      auth_realm: "Admin Area"
```

### 7. Fly.io Region Routing
```yaml
# Route by region
routes:
  fly_replay:
    - path: "^/sydney/"
      region: syd
      status: 307
```

### 8. Development with Hot Reload
```yaml
# Auto-reload on file changes
server:
  listen: 3000

applications:
  global_env:
    RAILS_ENV: development
  tenants:
    - name: dev
      path: /
```

## MkDocs Configuration

```yaml
# mkdocs.yml
site_name: Navigator Documentation
site_url: https://rubys.github.io/navigator/
repo_url: https://github.com/rubys/navigator
repo_name: rubys/navigator

theme:
  name: material
  features:
    - navigation.instant
    - navigation.tracking
    - navigation.tabs
    - navigation.sections
    - navigation.expand
    - navigation.indexes
    - toc.follow
    - search.suggest
    - search.highlight
    - content.code.copy
  palette:
    - scheme: default
      primary: indigo
      accent: indigo
      toggle:
        icon: material/brightness-7
        name: Switch to dark mode
    - scheme: slate
      primary: indigo
      accent: indigo
      toggle:
        icon: material/brightness-4
        name: Switch to light mode

plugins:
  - search
  - minify:
      minify_html: true

markdown_extensions:
  - pymdownx.highlight:
      anchor_linenums: true
  - pymdownx.superfences
  - pymdownx.tabbed:
      alternate_style: true
  - admonition
  - pymdownx.details
  - pymdownx.snippets
  - attr_list
  - md_in_html
  - toc:
      permalink: true

nav:
  - Home: index.md
  - Getting Started:
    - Installation: installation/index.md
    - First Application: getting-started/first-app.md
    - Basic Configuration: getting-started/basic-config.md
  - Configuration:
    - Overview: configuration/index.md
    - YAML Reference: configuration/yaml-reference.md
    - Applications: configuration/applications.md
    - Authentication: configuration/authentication.md
    - Static Files: configuration/static-files.md
    - Routing: configuration/routing.md
  - Examples:
    - Overview: examples/index.md
    - Single Tenant: examples/single-tenant.md
    - Multi-Tenant: examples/multi-tenant.md
    - With Redis: examples/with-redis.md
    - With Sidekiq: examples/with-sidekiq.md
    - WebSockets: examples/action-cable.md
  - Features:
    - Process Management: features/process-management.md
    - Port Allocation: features/port-allocation.md
    - Try Files: features/try-files.md
    - Fly-Replay: features/fly-replay.md
  - Deployment:
    - Production: deployment/production.md
    - Fly.io: deployment/fly-io.md
    - Monitoring: deployment/monitoring.md
  - Reference:
    - CLI Options: reference/cli.md
    - Environment: reference/environment.md
    - Signals: reference/signals.md

extra:
  social:
    - icon: fontawesome/brands/github
      link: https://github.com/rubys/navigator
  analytics:
    provider: google
    property: G-XXXXXXXXXX
```

## GitHub Actions Workflow

```yaml
# .github/workflows/docs.yml
name: Deploy Documentation

on:
  push:
    branches: [main]
    paths:
      - 'docs/**'
      - 'mkdocs.yml'
      - '.github/workflows/docs.yml'
  workflow_dispatch:

permissions:
  contents: read
  pages: write
  id-token: write

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - uses: actions/setup-python@v4
        with:
          python-version: '3.x'
      
      - name: Install MkDocs
        run: |
          pip install mkdocs-material
          pip install mkdocs-minify-plugin
      
      - name: Build documentation
        run: mkdocs build
      
      - name: Upload artifact
        uses: actions/upload-pages-artifact@v2
        with:
          path: ./site

  deploy:
    environment:
      name: github-pages
      url: ${{ steps.deployment.outputs.page_url }}
    runs-on: ubuntu-latest
    needs: build
    steps:
      - name: Deploy to GitHub Pages
        id: deployment
        uses: actions/deploy-pages@v2
```

## Example Page Template

```markdown
# Example: Rails with Sidekiq Background Jobs

## Scenario
You have a Rails application that needs to process background jobs using Sidekiq. Navigator will manage both the Rails app and the Sidekiq worker process.

## Configuration

```yaml
# config/navigator.yml
server:
  listen: 3000
  public_dir: ./public

managed_processes:
  - name: redis
    command: redis-server
    auto_restart: true
    start_delay: 0
    
  - name: sidekiq
    command: bundle
    args: [exec, sidekiq]
    working_dir: /path/to/rails/app
    env:
      RAILS_ENV: production
      REDIS_URL: redis://localhost:6379
    auto_restart: true
    start_delay: 2  # Wait for Redis to start

applications:
  global_env:
    RAILS_ENV: production
    REDIS_URL: redis://localhost:6379
    
  tenants:
    - name: myapp
      path: /
      working_dir: /path/to/rails/app
```

## Setup Steps

1. **Install dependencies**
   ```bash
   bundle add sidekiq
   bundle add redis
   ```

2. **Configure Sidekiq in Rails**
   ```ruby
   # config/initializers/sidekiq.rb
   Sidekiq.configure_server do |config|
     config.redis = { url: ENV['REDIS_URL'] }
   end
   
   Sidekiq.configure_client do |config|
     config.redis = { url: ENV['REDIS_URL'] }
   end
   ```

3. **Start Navigator**
   ```bash
   ./bin/navigator config/navigator.yml
   ```

## Testing

1. **Verify processes are running**
   ```bash
   ps aux | grep -E '(redis|sidekiq|rails)'
   ```

2. **Check Sidekiq web interface**
   ```ruby
   # config/routes.rb
   require 'sidekiq/web'
   mount Sidekiq::Web => '/sidekiq'
   ```

3. **Create a test job**
   ```ruby
   class TestJob < ApplicationJob
     def perform(message)
       Rails.logger.info "Processing: #{message}"
     end
   end
   
   TestJob.perform_later("Hello from Sidekiq!")
   ```

## Common Variations

### Using Resque instead of Sidekiq
```yaml
managed_processes:
  - name: resque
    command: bundle
    args: [exec, rake, resque:work]
    env:
      QUEUE: '*'
```

### Multiple queues
```yaml
managed_processes:
  - name: sidekiq-critical
    command: bundle
    args: [exec, sidekiq, -q, critical]
    
  - name: sidekiq-default
    command: bundle
    args: [exec, sidekiq, -q, default, -q, low]
```

## Troubleshooting

**Issue**: Sidekiq not connecting to Redis
- Check Redis is running: `redis-cli ping`
- Verify REDIS_URL is set correctly
- Check firewall rules if Redis is remote

**Issue**: Jobs not processing
- Check Sidekiq logs: `tail -f log/sidekiq.log`
- Verify queues are configured correctly
- Ensure Rails app can enqueue jobs

## See Also
- [Managed Processes](/features/process-management/)
- [Environment Variables](/configuration/templates/)
- [Production Deployment](/deployment/production/)
```

## Implementation Phases

### Phase 1: Foundation (Week 1)
- Set up MkDocs project structure
- Create GitHub Actions workflow
- Write landing page and installation guide
- Deploy initial site to GitHub Pages

### Phase 2: Core Documentation (Week 2)
- Configuration reference
- Basic examples (5-6 scenarios)
- Getting started guide
- CLI reference

### Phase 3: Advanced Topics (Week 3)
- Deployment guides
- Migration guides
- Troubleshooting section
- Advanced examples

### Phase 4: Polish (Week 4)
- Search optimization
- Cross-references
- Version selector
- Feedback mechanism

## Success Metrics
- All examples are copy-paste ready
- Each example takes <5 minutes to implement
- Documentation covers 90% of use cases
- Site loads in <2 seconds
- Search returns relevant results

## Maintenance Plan
- Update examples with each release
- Add new examples based on user questions
- Regular review of troubleshooting section
- Automated link checking
- Version-specific documentation branches