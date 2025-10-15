# Authentication

Navigator provides built-in HTTP Basic Authentication using htpasswd files, supporting multiple hash formats and flexible path-based exclusions.

## Basic Setup

```yaml
server:
  listen: 3000

auth:
  enabled: true
  realm: "My Application"
  htpasswd: /etc/navigator/htpasswd
  public_paths:
    - /assets/
    - /favicon.ico
    - "*.css"
    - "*.js"
```

## Configuration Options

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | boolean | `false` | Enable/disable authentication |
| `realm` | string | `"Protected Area"` | Authentication realm name |
| `htpasswd` | string | `""` | Path to htpasswd file |
| `public_paths` | array | `[]` | Glob/prefix patterns for paths that bypass auth |
| `auth_patterns` | array | `[]` | Regex patterns with actions for auth control |

## Creating htpasswd Files

### Using htpasswd Command

```bash
# Create new file with first user
htpasswd -c /etc/navigator/htpasswd admin

# Add additional users
htpasswd /etc/navigator/htpasswd user1
htpasswd /etc/navigator/htpasswd user2

# Use bcrypt (recommended for new passwords)
htpasswd -B /etc/navigator/htpasswd secure_user

# Use Apache MD5 (default, widely compatible)
htpasswd -m /etc/navigator/htpasswd another_user
```

### Using OpenSSL

```bash
# Generate bcrypt hash
openssl passwd -6
# Enter password when prompted

# Add to htpasswd file
echo "username:$6$salt$hash" >> /etc/navigator/htpasswd
```

### Using Python

```python
import bcrypt
import getpass

username = input("Username: ")
password = getpass.getpass("Password: ")
hashed = bcrypt.hashpw(password.encode('utf-8'), bcrypt.gensalt())

print(f"{username}:{hashed.decode('utf-8')}")
```

## Supported Hash Formats

Navigator supports all major htpasswd hash formats:

| Format | Security | Compatibility | Example |
|--------|----------|---------------|---------|
| **bcrypt** | High | Modern | `$2y$10$...` |
| **APR1 (Apache MD5)** | Medium | Universal | `$apr1$...` |
| **SHA** | Low | Legacy | `{SHA}...` |
| **MD5-crypt** | Low | Old Unix | `$1$...` |
| **Plain text** | None | Testing only | `password` |

!!! warning "Security Recommendation"
    Use bcrypt (`-B` flag) for new passwords. Avoid plain text and SHA formats in production.

## Public Paths

Configure paths that bypass authentication using glob patterns:

```yaml
auth:
  enabled: true
  htpasswd: ./htpasswd
  public_paths:
    # Exact paths
    - /favicon.ico
    - /robots.txt
    - /health
    
    # Directory prefixes  
    - /assets/
    - /public/
    
    # Wildcard patterns
    - "*.css"
    - "*.js"
    - "*.png"
    - "*.jpg"
    - "*.gif"
    
    # Complex patterns
    - "/api/v*/public/*"
    - "/docs/*.html"
```

### Pattern Matching Rules

1. **Exact match**: `/favicon.ico` matches only that path
2. **Prefix match**: `/assets/` matches `/assets/style.css`, `/assets/images/logo.png`
3. **Glob patterns**: `*.css` matches any path ending in `.css`
4. **Wildcards**: `*` matches any characters, `?` matches single character

## Auth Patterns (Advanced)

For complex authentication requirements, use `auth_patterns` with regex patterns:

```yaml
auth:
  enabled: true
  htpasswd: /etc/navigator/htpasswd
  realm: "Protected Area"

  # Simple glob patterns (checked second)
  public_paths:
    - /assets/
    - "*.css"
    - "*.js"

  # Regex patterns (checked first)
  auth_patterns:
    # Allow studio index pages but not tenant apps
    - pattern: "^/showcase/2025/(raleigh|boston|seattle)/?$"
      action: "off"

    # Allow public paths within tenants
    - pattern: "^/showcase/2025/[^/]+/[^/]+/public/"
      action: "off"

    # Require different realm for admin area
    - pattern: "^/admin/"
      action: "Admin Only"
```

### Auth Pattern Options

Each auth pattern has two fields:

| Field | Type | Description |
|-------|------|-------------|
| `pattern` | string | Regular expression to match against request path |
| `action` | string | `"off"` (bypass auth) or realm name (require auth with specific realm) |

### When to Use Auth Patterns

Use **auth_patterns** instead of **public_paths** when you need:

1. **Complex path matching**: Match specific paths with regex precision
2. **Grouped alternations**: One pattern for multiple similar paths (better performance)
3. **Exact matching**: Match paths precisely without wildcards
4. **Per-pattern realms**: Different auth realms for different patterns

**Example: Multi-tenant with public pages**

```yaml
auth_patterns:
  # Group studios by year for better performance
  - pattern: "^/showcase/2025/(boston|seattle|raleigh|portland)/?$"
    action: "off"

  # Group public tenant paths
  - pattern: "^/showcase/2025/(boston|seattle)/public/(heats|entries)$"
    action: "off"
```

This approach is much more efficient than creating individual patterns for each studio or path.

### Pattern Evaluation Order

Navigator checks auth exclusions in this order:

1. **Auth Patterns** (most specific): Regex patterns from `auth_patterns`
2. **Public Paths** (general): Glob/prefix patterns from `public_paths`

If any pattern matches and has `action: "off"`, authentication is bypassed.

### Performance Tips

**DO**: Use grouped alternations
```yaml
# GOOD: One pattern, fast evaluation
auth_patterns:
  - pattern: "^/(boston|seattle|raleigh)/?$"
    action: "off"
```

**DON'T**: Create many individual patterns
```yaml
# BAD: Multiple patterns, slower evaluation
auth_patterns:
  - pattern: "^/boston/?$"
    action: "off"
  - pattern: "^/seattle/?$"
    action: "off"
  - pattern: "^/raleigh/?$"
    action: "off"
```

Grouped alternations reduce:
- Regex compilation overhead at startup
- Number of pattern checks per request
- Memory usage

## Per-Application Authentication

Override authentication settings per application:

```yaml
auth:
  enabled: true
  htpasswd: /etc/navigator/main.htpasswd
  realm: "Main Site"

applications:
  tenants:
    # Uses main auth settings
    - name: main
      path: /
      
    # Custom auth realm
    - name: admin
      path: /admin/
      auth_realm: "Admin Area"
      
    # Different htpasswd file
    - name: api
      path: /api/
      htpasswd: /etc/navigator/api.htpasswd
      
    # No auth required
    - name: public
      path: /public/
      auth_enabled: false
```

## Development vs Production

### Development Configuration

```yaml
# Disable auth for development
auth:
  enabled: false

# Or use simple test credentials
auth:
  enabled: true
  htpasswd: ./dev-htpasswd
  realm: "Dev"
  public_paths:
    - /assets/
    - "*.css"
    - "*.js"
```

Create development htpasswd:
```bash
echo "dev:dev" > dev-htpasswd  # Plain text for development only
```

### Production Configuration

```yaml
auth:
  enabled: true
  htpasswd: /etc/navigator/htpasswd
  realm: "Production System"
  public_paths:
    - /assets/
    - /packs/
    - /favicon.ico
    - /robots.txt
    - /health
    - "*.css"
    - "*.js"
    - "*.png"
    - "*.jpg"
    - "*.gif"
    - "*.woff"
    - "*.woff2"
```

## Security Best Practices

### 1. File Permissions

```bash
# Secure htpasswd file
chmod 600 /etc/navigator/htpasswd
chown navigator:navigator /etc/navigator/htpasswd

# Secure directory
chmod 700 /etc/navigator
```

### 2. Use Strong Passwords

```bash
# Generate secure passwords
openssl rand -base64 32

# Use bcrypt for hashing
htpasswd -B /etc/navigator/htpasswd username
```

### 3. Regular User Audits

```bash
# List all users
cut -d: -f1 /etc/navigator/htpasswd

# Remove unused accounts
htpasswd -D /etc/navigator/htpasswd olduser
```

### 4. Monitor Authentication

```bash
# Monitor auth failures in logs
grep "401\|403" /var/log/navigator.log

# Count auth attempts
grep "Basic auth" /var/log/navigator.log | wc -l
```

## Common Patterns

### Multi-Tier Authentication

```yaml
auth:
  enabled: true
  htpasswd: /etc/navigator/users.htpasswd
  
applications:
  tenants:
    # Public area - no auth
    - name: public
      path: /
      auth_enabled: false
      
    # User area - basic auth
    - name: app
      path: /app/
      # Uses main auth settings
      
    # Admin area - separate auth file
    - name: admin  
      path: /admin/
      htpasswd: /etc/navigator/admin.htpasswd
      auth_realm: "Admin Only"
```

### API with Mixed Auth

```yaml
auth:
  enabled: true
  htpasswd: /etc/navigator/htpasswd
  public_paths:
    # Public API endpoints
    - /api/v1/public/*
    - /api/health
    - /api/status
    
applications:
  tenants:
    # Protected web interface
    - name: web
      path: /
      
    # Mixed API (some endpoints public via public_paths)
    - name: api
      path: /api/
```

### Regional Authentication

```yaml
# Different auth per region
applications:
  tenants:
    - name: us-app
      path: /us/
      htpasswd: /etc/navigator/us.htpasswd
      auth_realm: "US Region"
      
    - name: eu-app
      path: /eu/
      htpasswd: /etc/navigator/eu.htpasswd  
      auth_realm: "EU Region"
```

## Troubleshooting

### Authentication Not Working

1. **Check htpasswd file**:
   ```bash
   # Verify file exists and is readable
   ls -la /etc/navigator/htpasswd
   
   # Test password manually
   htpasswd -v /etc/navigator/htpasswd username
   ```

2. **Check public paths**:
   ```bash
   # Test if path should be public
   curl -I http://localhost:3000/assets/style.css
   ```

3. **Verify configuration**:
   ```yaml
   # Ensure auth is enabled
   auth:
     enabled: true  # Must be true
     htpasswd: /correct/path/to/htpasswd
   ```

### Browser Not Prompting

1. **Check realm configuration**:
   ```yaml
   auth:
     realm: "My Site"  # Must be set for browser prompt
   ```

2. **Clear browser cache**: Browser may cache auth failures

3. **Check for JavaScript interference**: Some JS frameworks interfere with Basic Auth

### Password Not Working

1. **Verify hash format**: Ensure htpasswd file uses supported format
2. **Check for special characters**: Some characters may need escaping
3. **Recreate user**: Delete and recreate user entry

```bash
# Remove user
htpasswd -D /etc/navigator/htpasswd username

# Add user again
htpasswd /etc/navigator/htpasswd username
```

### Performance Issues

1. **Use bcrypt sparingly**: bcrypt is CPU-intensive
2. **Cache htpasswd**: Navigator caches file contents
3. **Optimize public paths**: More specific patterns perform better

## Testing Authentication

### Manual Testing

```bash
# Test without credentials (should get 401)
curl -I http://localhost:3000/

# Test with valid credentials
curl -u username:password http://localhost:3000/

# Test public path (should work without auth)
curl -I http://localhost:3000/assets/style.css
```

### Automated Testing

```bash
#!/bin/bash
# Test authentication endpoints

BASE_URL="http://localhost:3000"
USER="testuser"
PASS="testpass"

# Test auth required
response=$(curl -s -o /dev/null -w "%{http_code}" $BASE_URL/)
if [ "$response" = "401" ]; then
    echo "✓ Auth required correctly"
else
    echo "✗ Auth not working: $response"
fi

# Test valid auth
response=$(curl -s -o /dev/null -w "%{http_code}" -u $USER:$PASS $BASE_URL/)
if [ "$response" = "200" ]; then
    echo "✓ Valid auth works"
else
    echo "✗ Valid auth failed: $response"
fi

# Test public path
response=$(curl -s -o /dev/null -w "%{http_code}" $BASE_URL/assets/test.css)
if [ "$response" = "200" ] || [ "$response" = "404" ]; then
    echo "✓ Public path bypasses auth"
else
    echo "✗ Public path requires auth: $response"
fi
```

## Migration from nginx

When migrating from nginx Basic Auth:

```nginx
# nginx configuration
location / {
    auth_basic "Protected Area";
    auth_basic_user_file /etc/nginx/.htpasswd;
}
```

Becomes:

```yaml
# Navigator configuration
auth:
  enabled: true
  realm: "Protected Area"
  htpasswd: /etc/nginx/.htpasswd
```

The htpasswd files are compatible between nginx and Navigator.

## See Also

- [Configuration Overview](index.md)
- [Applications](applications.md)
- [Static Files](static-files.md)
- [Single Tenant Example](../examples/single-tenant.md)