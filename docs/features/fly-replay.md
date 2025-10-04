# Fly-Replay Routing

Navigator provides intelligent Fly-Replay routing for optimal request distribution across regions, applications, and specific machine instances in Fly.io deployments.

## What is Fly-Replay?

Fly-Replay is Fly.io's request routing system that allows applications to redirect requests to different regions, applications, or specific machines based on various criteria. Navigator enhances this with smart fallback behavior and automatic optimization.

## How It Works

When a request matches a fly-replay rule, Navigator:

1. **Evaluates the pattern** against the request URL
2. **Checks request size** - Large requests (>1MB) automatically use reverse proxy
3. **Generates Fly-Replay response** with appropriate target information
4. **Falls back to reverse proxy** when replay isn't suitable

## Configuration

### Basic Fly-Replay Setup

```yaml
routes:
  fly:
    replay:
      - path: "^/api/heavy/"
        region: fra          # Route to Frankfurt region
        status: 307
        methods: [GET, POST]
```

### Multi-Target Routing Types

Navigator supports three types of Fly-Replay routing:

#### 1. Region-Based Routing

Route requests to specific Fly.io regions:

```yaml
routes:
  fly:
    replay:
      # Route Australian users to Sydney
      - path: "^/au/"
        region: syd
        status: 307

      # Route European users to Frankfurt
      - path: "^/eu/"
        region: fra
        status: 307

      # Route Asian users to Singapore
      - path: "^/asia/"
        region: sin
        status: 307
```

#### 2. Application-Based Routing

Route requests to specific Fly.io applications:

```yaml
routes:
  fly:
    replay:
      # Route PDF generation to specialized app
      - path: "^/.*\\.pdf$"
        app: pdf-generator
        status: 307

      # Route image processing to image service
      - path: "^/images/resize/"
        app: image-processor
        status: 307

      # Route search requests to search service
      - path: "^/search/"
        app: search-engine
        status: 307
```

#### 3. Machine-Specific Routing

Route requests to specific machine instances:

```yaml
routes:
  fly:
    replay:
      # Route high-priority requests to specific machine
      - path: "^/priority/"
        machine: "48e403dc711e18"
        app: priority-handler
        status: 307

      # Route admin requests to admin machine
      - path: "^/admin/"
        machine: "e24a0123456789"
        app: admin-console
        status: 307
```

## Fly.io Regions

### Available Regions

| Region | Code | Location | Use Case |
|--------|------|----------|----------|
| **North America** |
| Ashburn | `iad` | Virginia, US | East Coast users |
| Chicago | `ord` | Illinois, US | Central US users |
| Los Angeles | `lax` | California, US | West Coast users |
| Seattle | `sea` | Washington, US | Pacific Northwest |
| Toronto | `yyz` | Ontario, Canada | Canadian users |
| **Europe** |
| Amsterdam | `ams` | Netherlands | Northern Europe |
| Frankfurt | `fra` | Germany | Central Europe |
| London | `lhr` | United Kingdom | UK users |
| **Asia Pacific** |
| Hong Kong | `hkg` | China | Asian users |
| Singapore | `sin` | Singapore | Southeast Asia |
| Sydney | `syd` | Australia | Australian users |
| Tokyo | `nrt` | Japan | Japanese users |

### Regional Routing Strategy

```yaml
routes:
  fly:
    replay:
      # Americas
      - path: "^/(us|ca|mx)/"
        region: ord
        methods: [GET]

      # Europe
      - path: "^/(uk|de|fr|it|es)/"
        region: fra
        methods: [GET]

      # Asia Pacific
      - path: "^/(au|nz|sg|jp)/"
        region: syd
        methods: [GET]
```

## Smart Fallback Behavior

Navigator automatically uses reverse proxy instead of Fly-Replay when:

### 1. Large Requests (>1MB)

```yaml
routes:
  fly:
    replay:
      - path: "^/upload/"
        region: fra
        # Large file uploads automatically use reverse proxy
```

Navigator detects:
- Large POST/PUT request bodies
- File uploads
- Bulk data submissions

### 2. Method Restrictions

```yaml
routes:
  fly:
    replay:
      - path: "^/api/"
        region: fra
        methods: [GET, HEAD]  # Only safe methods use replay
        # POST/PUT/DELETE automatically use reverse proxy
```

### 3. WebSocket Upgrades

WebSocket connections require persistent connections, so Navigator automatically uses reverse proxy for:
- WebSocket upgrade requests
- Server-sent events
- Long-polling connections

## Advanced Patterns

### Geographic Routing

```yaml
routes:
  fly:
    replay:
      # Route based on URL patterns suggesting geography
      - path: "^/stores/sydney/"
        region: syd

      - path: "^/stores/london/"
        region: lhr

      - path: "^/stores/newyork/"
        region: iad
```

### Service-Specific Applications

```yaml
routes:
  fly:
    replay:
      # PDF processing service
      - path: "^/documents/.+\\.pdf$"
        app: pdf-service

      # Image optimization service
      - path: "^/media/optimize/"
        app: image-optimizer

      # Heavy computation service
      - path: "^/compute/"
        app: compute-cluster
        region: fra  # Use high-CPU region
```

### Load Balancing Scenarios

```yaml
routes:
  fly:
    replay:
      # Route specific customers to dedicated instances
      - path: "^/enterprise/customer1/"
        machine: "dedicated-1"
        app: enterprise-app

      - path: "^/enterprise/customer2/"
        machine: "dedicated-2"
        app: enterprise-app

      # Standard customers use default routing
      # (no routes.fly.replay rule = normal load balancing)
```

## Fallback Proxy Configuration

When Navigator falls back to reverse proxy, it constructs internal URLs based on target type:

### Internal URL Patterns

| Target Type | Internal URL Pattern |
|-------------|---------------------|
| **Machine** | `http://{machine}.vm.{app}.internal:{port}{path}` |
| **App** | `http://{app}.internal:{port}{path}` |
| **Region** | `http://{region}.{FLY_APP_NAME}.internal:{port}{path}` |

### Environment Requirements

For fallback proxy to work, ensure:

```bash
# Required environment variable for region-based fallback
export FLY_APP_NAME=your-app-name

# Navigator constructs URLs like:
# http://fra.your-app-name.internal:3000/path
# http://pdf-service.internal:3000/path  
# http://machine123.vm.pdf-service.internal:3000/path
```

## Performance Optimization

### Pattern Optimization

```yaml
routes:
  fly:
    replay:
      # Specific patterns first (faster matching)
      - path: "^/api/v2/heavy-compute"
        region: fra

      # General patterns last
      - path: "^/api/"
        region: ord
```

### Method Filtering

```yaml
routes:
  fly:
    replay:
      # Only apply expensive routing to read operations
      - path: "^/reports/"
        region: fra
        methods: [GET, HEAD]  # Skip POST/PUT/DELETE
```

### Conditional Routing

```yaml
routes:
  fly:
    replay:
      # Route large computations to powerful region
      - path: "^/compute/heavy/"
        region: fra

      # Route quick operations to closest region
      - path: "^/compute/quick/"
        region: ord
```

## Monitoring and Debugging

### Request Flow Logging

Navigator logs Fly-Replay decisions:

```bash
# Enable debug logging
LOG_LEVEL=debug navigator config.yml

# Monitor replay decisions
tail -f /var/log/navigator.log | grep -E "(fly-replay|fallback)"
```

### Testing Fly-Replay Rules

```bash
# Test region routing
curl -I http://localhost:3000/eu/dashboard
# Should return 307 with Fly-Replay headers

# Test large request fallback  
curl -X POST -H "Content-Length: 2000000" \
     -d @large-file.json \
     http://localhost:3000/api/upload
# Should use reverse proxy (no 307)

# Test method exclusion
curl -X DELETE http://localhost:3000/api/data
# Should use reverse proxy if DELETE excluded
```

### Fly-Replay Response Format

When Navigator sends a Fly-Replay response:

```json
{
  "region": "fra",
  "app": "pdf-service", 
  "prefer_instance": "machine123",
  "transform": {
    "set_headers": [
      {"name": "X-Navigator-Retry", "value": "true"}
    ]
  }
}
```

## Common Use Cases

### 1. Multi-Region Deployment

```yaml
# Deploy same app to multiple regions
routes:
  fly:
    replay:
      # Route users to closest region based on path
      - path: "^/us-east/"
        region: iad
      - path: "^/us-west/"
        region: lax
      - path: "^/europe/"
        region: fra
      - path: "^/asia/"
        region: sin
```

### 2. Microservices Architecture

```yaml
routes:
  fly:
    replay:
      # Route to specialized services
      - path: "^/auth/"
        app: auth-service
      - path: "^/payments/"
        app: payment-service
      - path: "^/notifications/"
        app: notification-service
```

### 3. Resource-Heavy Operations

```yaml
routes:
  fly:
    replay:
      # Route CPU-intensive operations to powerful machines
      - path: "^/process/video/"
        machine: "high-cpu-1"
        app: video-processor

      # Route memory-intensive operations
      - path: "^/process/data/"
        machine: "high-memory-1"
        app: data-processor
```

### 4. Customer Isolation

```yaml
routes:
  fly:
    replay:
      # Enterprise customers on dedicated machines
      - path: "^/enterprise/acme-corp/"
        machine: "acme-dedicated"
        app: enterprise-platform

      - path: "^/enterprise/big-co/"
        machine: "bigco-dedicated"
        app: enterprise-platform
```

## Troubleshooting

### Fly-Replay Not Working

1. **Check pattern matching**:
   ```bash
   # Test regex patterns
   echo "/eu/dashboard" | grep -E "^/eu/"
   ```

2. **Verify Fly.io environment**:
   ```bash
   # Required for region fallback
   echo $FLY_APP_NAME

   # Check internal DNS resolution
   nslookup fra.your-app.internal
   ```

3. **Check request method**:
   ```yaml
   routes:
     fly:
       replay:
         - path: "^/api/"
           region: fra
           methods: [GET, HEAD]  # Ensure method is included
   ```

### Fallback Proxy Issues

1. **Check internal networking**:
   ```bash
   # Test internal connectivity
   curl -I http://fra.your-app.internal:3000/
   ```

2. **Verify app/machine names**:
   ```bash
   # Check app exists
   fly apps list | grep your-app-name
   
   # Check machine exists
   fly machines list -a your-app-name
   ```

### Performance Issues

1. **Optimize patterns**:
   ```yaml
   # Put specific patterns first
   routes:
     fly:
       replay:
         - path: "^/api/v2/specific-endpoint"  # Specific first
         - path: "^/api/"                      # General last
   ```

2. **Limit method scope**:
   ```yaml
   routes:
     fly:
       replay:
         - path: "^/heavy-compute/"
           region: fra
           methods: [GET]  # Don't route all methods
   ```

## Security Considerations

### Request Validation

Navigator validates Fly-Replay requests:
- Checks pattern matches
- Validates target exists
- Ensures method is allowed

### Internal Network Security

Fallback proxy uses Fly.io internal networking:
- Traffic stays within Fly.io network
- Automatic TLS encryption
- No external network exposure

### Retry Headers

Navigator adds retry tracking headers:
```
X-Navigator-Retry: true
```

This prevents infinite retry loops when requests bounce between regions.

## See Also

- [Configuration Overview](../configuration/index.md)
- [Routing Configuration](../configuration/routing.md)
- [Multi-Tenant Examples](../examples/multi-tenant.md)
- [Deployment on Fly.io](../deployment/fly-io.md)