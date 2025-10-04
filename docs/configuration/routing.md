# URL Routing and Rewrites

Navigator provides flexible URL routing with support for rewrites, redirects, and Fly.io's intelligent replay system for regional deployments.

## Basic Routing

Navigator routes requests in this order:

1. **Static files** - Direct filesystem serving
2. **Rewrite rules** - URL transformations
3. **Authentication** - Access control
4. **Applications** - Rails app routing by path prefix

## Rewrite Rules

Transform URLs before they reach your Rails application:

```yaml
routes:
  rewrites:
    # Redirect old URLs
    - pattern: "^/old-blog/(.*)"
      replacement: "/blog/$1"
      redirect: true
      status: 301
    
    # Internal rewrite (no redirect)
    - pattern: "^/api/v1/(.*)"
      replacement: "/api/latest/$1"
      redirect: false
    
    # Simple redirects
    - pattern: "^/home$"
      replacement: "/"
      redirect: true
      status: 302
```

### Rewrite Configuration

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `pattern` | string | - | Regular expression pattern to match |
| `replacement` | string | - | Replacement string (supports `$1`, `$2` capture groups) |
| `redirect` | boolean | `false` | Send HTTP redirect vs internal rewrite |
| `status` | integer | `302` | HTTP status code for redirects |

### Pattern Examples

```yaml
routes:
  rewrites:
    # Capture groups with ()
    - pattern: "^/user/([0-9]+)$"
      replacement: "/users/$1"
    
    # Multiple capture groups
    - pattern: "^/([0-9]{4})/([a-z]+)/(.*)$"
      replacement: "/$1/$2/events/$3"
    
    # Case-insensitive matching
    - pattern: "(?i)^/API/(.*)"
      replacement: "/api/$1"
    
    # Remove file extensions
    - pattern: "^/(.*)\\.html$"
      replacement: "/$1"
```

## Reverse Proxy Routing

Route requests to external services or APIs using reverse proxy:

```yaml
routes:
  reverse_proxies:
    # Simple proxy to external service
    - name: api-backend
      path: "^/api/"
      target: https://api.example.com
      strip_path: true
      headers:
        X-Forwarded-Host: "$host"
        X-Real-IP: "$remote_addr"

    # Proxy with capture group substitution
    - name: studio-requests
      path: "^/showcase/studios/([a-z]+)/request$"
      target: https://backend.example.com/showcase/studios/$1/request
      headers:
        X-Forwarded-Host: "$host"

    # Multiple capture groups
    - name: versioned-api
      path: "^/api/v([0-9]+)/users/([0-9]+)$"
      target: https://backend.example.com/internal/v$1/user/$2

    # WebSocket proxy
    - name: websocket-server
      prefix: /cable
      target: http://localhost:28080
      websocket: true
```

### Reverse Proxy Configuration

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `name` | string | - | Descriptive name for the route |
| `path` | string | - | Regular expression pattern to match URLs |
| `prefix` | string | - | Simple prefix match (alternative to `path`) |
| `target` | string | - | Target URL (supports `$1`, `$2` for capture groups) |
| `strip_path` | boolean | `false` | Remove matched prefix before proxying |
| `headers` | object | - | Custom headers to add (supports variables) |
| `websocket` | boolean | `false` | Enable WebSocket proxying |

### Capture Group Substitution

Use regex capture groups in the `path` and reference them in the `target` URL:

```yaml
routes:
  reverse_proxies:
    # Single capture group
    - name: user-proxy
      path: "^/users/([0-9]+)$"
      target: https://api.example.com/v1/user/$1

    # Multiple capture groups
    - name: date-based-proxy
      path: "^/archive/([0-9]{4})/([0-9]{2})/(.*)$"
      target: https://archive.example.com/$1-$2/$3
```

**How it works:**
- Request: `GET /users/123`
- Pattern matches with `$1 = "123"`
- Proxies to: `https://api.example.com/v1/user/123`

**Multiple captures:**
- Request: `GET /archive/2024/03/report.pdf`
- Pattern matches with `$1 = "2024"`, `$2 = "03"`, `$3 = "report.pdf"`
- Proxies to: `https://archive.example.com/2024-03/report.pdf`

### Header Variables

Custom headers support variable substitution:

| Variable | Description | Example |
|----------|-------------|---------|
| `$host` | Request hostname | `example.com` |
| `$remote_addr` | Client IP address | `203.0.113.45` |
| `$scheme` | Request scheme | `https` |

```yaml
routes:
  reverse_proxies:
    - name: backend-proxy
      path: "^/backend/"
      target: https://internal.example.com
      headers:
        X-Forwarded-Host: "$host"
        X-Forwarded-Proto: "$scheme"
        X-Real-IP: "$remote_addr"
        X-Custom-Header: "static-value"
```

### Path Stripping

Remove the matched prefix before sending to target:

```yaml
routes:
  reverse_proxies:
    # With strip_path
    - name: api-proxy
      prefix: /api/
      target: https://backend.example.com
      strip_path: true
    # Request: /api/users → Proxies to: https://backend.example.com/users

    # Without strip_path (default)
    - name: full-path-proxy
      prefix: /service/
      target: https://backend.example.com
    # Request: /service/users → Proxies to: https://backend.example.com/service/users
```

**Note:** When using capture group substitution in the target, the path is already fully specified, so `strip_path` is ignored.

## Fly-Replay Routing

Route requests to specific Fly.io regions, applications, or machines for optimal performance:

```yaml
routes:
  fly:
    replay:
      # Region-based routing
      - path: "^/sydney/"
        region: syd
        status: 307
        methods: [GET, POST]

      # App-based routing
      - path: "^/.*\\.pdf$"
        app: pdf-generator
        status: 307

      # Machine-specific routing
      - path: "^/priority/.*"
        machine: "48e403dc711e18"
        app: priority-processor
        status: 307
```

### Fly-Replay Configuration

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `path` | string | ✓ | URL regex pattern |
| `region` | string | | Target Fly.io region code |
| `app` | string | | Target Fly.io application name |
| `machine` | string | | Specific machine ID (requires `app`) |
| `status` | integer | `307` | HTTP status code |
| `methods` | array | | HTTP methods to match |

### Fly.io Region Codes

| Region | Code | Location |
|--------|------|----------|
| **North America** |
| Ashburn | `iad` | Virginia, US |
| Chicago | `ord` | Illinois, US |
| Los Angeles | `lax` | California, US |
| Seattle | `sea` | Washington, US |
| Toronto | `yyz` | Ontario, Canada |
| **Europe** |
| Amsterdam | `ams` | Netherlands |
| Frankfurt | `fra` | Germany |
| London | `lhr` | United Kingdom |
| **Asia Pacific** |
| Hong Kong | `hkg` | China |
| Singapore | `sin` | Singapore |
| Sydney | `syd` | Australia |
| Tokyo | `nrt` | Japan |

### Smart Fallback

Navigator automatically falls back to reverse proxy for requests that can't use Fly-Replay:

- **Large requests** (>1MB) - proxied directly to avoid replay limits
- **Non-GET/HEAD methods** - for safety with state-changing operations
- **WebSocket upgrades** - require persistent connections

## Routing Patterns

### 1. Multi-Region Deployment

```yaml
routes:
  fly:
    replay:
      # Route by geographic prefix
      - path: "^/asia/"
        region: sin
        methods: [GET]

      - path: "^/europe/"
        region: fra
        methods: [GET]

      - path: "^/americas/"
        region: ord
        methods: [GET]

      # Heavy processing to dedicated region
      - path: "^/reports/"
        region: fra  # EU region with more CPU
        methods: [POST]
```

### 2. Service-Specific Routing

```yaml
routes:
  fly:
    replay:
      # PDF generation service
      - path: "^/.*\\.pdf$"
        app: pdf-service

      # Image processing
      - path: "^/images/resize/"
        app: image-processor

      # Search service
      - path: "^/search/"
        app: search-engine
```

### 3. Load Balancing by Machine

```yaml
routes:
  fly:
    replay:
      # High-priority requests to specific machine
      - path: "^/priority/"
        machine: "e24a0123456"
        app: main-app

      # Regular traffic uses default routing
      # (no fly replay rule = normal load balancing)
```

### 4. Development vs Production Routing

=== "Development"

    ```yaml
    routes:
      rewrites:
        # Route everything to local Rails
        - pattern: "^/api/(.*)"
          replacement: "/api/$1"
          redirect: false
    
    # No fly_replay in development
    ```

=== "Production"

    ```yaml
    routes:
      fly:
        replay:
          - path: "^/api/heavy/"
            region: fra  # Route heavy API calls to EU

          - path: "^/api/search/"
            app: search-service

      rewrites:
        # Redirect old API versions
        - pattern: "^/api/v[12]/(.*)"
          replacement: "/api/v3/$1"
          redirect: true
          status: 301
    ```

## URL Rewriting Examples

### Legacy URL Support

```yaml
routes:
  rewrites:
    # Old WordPress URLs
    - pattern: "^/\\?p=([0-9]+)$"
      replacement: "/posts/$1"
      redirect: true
      status: 301
    
    # Old category URLs
    - pattern: "^/category/(.+)$"
      replacement: "/categories/$1"
      redirect: true
      status: 301
    
    # Remove trailing slashes
    - pattern: "^(.+)/$"
      replacement: "$1"
      redirect: true
      status: 301
```

### API Versioning

```yaml
routes:
  rewrites:
    # Default to latest API version
    - pattern: "^/api/([^/]+)$"
      replacement: "/api/v3/$1"
      redirect: false
    
    # Redirect old versions
    - pattern: "^/api/v[12]/(.*)"
      replacement: "/api/v3/$1"
      redirect: true
      status: 301
```

### Multi-Language Sites

```yaml
routes:
  rewrites:
    # Default to English
    - pattern: "^/([^/]+)$"
      replacement: "/en/$1"
      redirect: false
    
    # Language-specific routing
    - pattern: "^/(en|es|fr)/(.*)$"
      replacement: "/$2?lang=$1"
      redirect: false
```

## Application Path Routing

Define which Rails applications handle which URL paths:

```yaml
applications:
  tenants:
    # API application
    - name: api
      path: /api/
      working_dir: /var/www/api
    
    # Admin application  
    - name: admin
      path: /admin/
      working_dir: /var/www/admin
    
    # Main application (catch-all)
    - name: main
      path: /
      working_dir: /var/www/main
```

### Path Matching Rules

1. **Exact prefix match**: `/api/` matches `/api/users` but not `/api-v2/users`
2. **Longest match wins**: `/api/v2/` takes precedence over `/api/`
3. **Order matters**: First matching path is used

### Advanced Path Patterns

```yaml
applications:
  tenants:
    # Pattern matching with wildcards
    - name: tenant-sites
      path: /sites/*/
      match_pattern: "/sites/*/admin"
    
    # Multiple paths for same app
    - name: legacy
      path: /old/
    - name: legacy-alt  
      path: /legacy/
      # Both route to same Rails app
```

## HTTP Method Filtering

Control which HTTP methods trigger routing rules:

```yaml
routes:
  fly:
    replay:
      # Only GET requests to region
      - path: "^/catalog/"
        region: syd
        methods: [GET, HEAD]

      # POST requests to processing service
      - path: "^/process/"
        app: processor
        methods: [POST, PUT]

applications:
  tenants:
    # Exclude dangerous methods from API
    - name: readonly-api
      path: /api/read/
      exclude_methods: [POST, PUT, DELETE, PATCH]
```

## Error Handling

### Custom Error Pages

```yaml
routes:
  rewrites:
    # Custom 404 page
    - pattern: "^/404$"
      replacement: "/errors/not_found"
      redirect: false
    
    # Custom 500 page
    - pattern: "^/500$"
      replacement: "/errors/server_error"
      redirect: false
```

### Maintenance Mode

```yaml
maintenance:
  page: "/503.html"  # Custom maintenance page

routes:
  rewrites:
    # Redirect everything to maintenance page
    - pattern: "^/(?!maintenance).*$"
      replacement: "/maintenance"
      redirect: true
      status: 503

server:
  # Assets still served
  static:
    public_dir: public/
```

## Testing Routing Rules

### Manual Testing

```bash
# Test redirects
curl -I http://localhost:3000/old-blog/post1
# Should show 301 redirect

# Test rewrites  
curl -v http://localhost:3000/api/users
# Check final URL in logs

# Test fly-replay
curl -I http://localhost:3000/sydney/products
# Should show 307 with JSON body
```

### Automated Testing

```bash
#!/bin/bash
# Test routing rules

# Test redirect
response=$(curl -s -o /dev/null -w "%{http_code}" -L http://localhost:3000/old-url)
if [ "$response" = "200" ]; then
    echo "✓ Redirect working"
else
    echo "✗ Redirect failed: $response"
fi

# Test rewrite
response=$(curl -s http://localhost:3000/api/test)
if echo "$response" | grep -q "expected content"; then
    echo "✓ Rewrite working"  
else
    echo "✗ Rewrite failed"
fi
```

## Performance Considerations

### 1. Pattern Complexity

```yaml
# Fast - simple prefix
- pattern: "^/api/"
  replacement: "/v3/api/"

# Slower - complex regex
- pattern: "^/([0-9]{4})/([0-9]{2})/([0-9]{2})/(.+)$"
  replacement: "/date/$1-$2-$3/$4"
```

### 2. Rule Order

```yaml
routes:
  rewrites:
    # Put specific rules first
    - pattern: "^/api/v3/special"
      replacement: "/special-handler"
    
    # General rules last  
    - pattern: "^/api/v3/(.*)"
      replacement: "/api/$1"
```

### 3. Fly-Replay Optimization

```yaml
routes:
  fly:
    replay:
      # Use specific patterns to avoid unnecessary checks
      - path: "^/heavy-compute/"  # Specific
        region: fra

      # Avoid overly broad patterns
      # - path: ".*"  # Matches everything!
```

## Troubleshooting

### Rules Not Matching

1. **Test patterns**:
   ```bash
   # Use regex tester or
   echo "/old-blog/post1" | grep -E "^/old-blog/(.*)"
   ```

2. **Check order**: More specific rules should come first

3. **Escape special characters**:
   ```yaml
   # Wrong
   pattern: "/api/v1.0/*"
   
   # Correct  
   pattern: "/api/v1\\.0/.*"
   ```

### Redirects Not Working

1. **Check redirect flag**:
   ```yaml
   routes:
     rewrites:
       - pattern: "^/old"
         replacement: "/new"
         redirect: true  # Must be true for HTTP redirect
         status: 301     # Or 302 for temporary
   ```

2. **Verify status code**: Default is 302 (temporary)

### Fly-Replay Issues

1. **Check app/region exists**: Invalid targets will fail
2. **Large requests**: >1MB automatically use reverse proxy
3. **Method restrictions**: Some methods may not work with replay

## See Also

- [Configuration Overview](index.md)
- [Applications](applications.md)
- [Fly-Replay Feature](../features/fly-replay.md)
- [Multi-Tenant Example](../examples/multi-tenant.md)