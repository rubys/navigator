# Try Files Feature

Navigator's try files feature allows serving static content directly from the filesystem with flexible file resolution, similar to nginx's `try_files` directive.

## How It Works

When a request comes in, Navigator attempts to serve files in this order:

1. **Exact match**: Look for the exact file path
2. **With suffixes**: Try adding configured suffixes (`.html`, `.htm`, etc.)
3. **Index files**: Look for index files in directories
4. **Fallback**: Fall back to Rails application if no file found

## Configuration

```yaml
static:
  try_files:
    enabled: true
    suffixes: ["index.html", ".html", ".htm", ".txt", ".xml", ".json"]
    fallback: rails  # or "404" for 404 error
  
  directories:
    - path: /assets/
      root: public/assets/
    - path: /docs/
      root: public/docs/
```

## Use Cases

### Static Site Content

Perfect for serving documentation, marketing pages, or static content:

```yaml
# Serve docs directly without Rails
static:
  try_files:
    enabled: true
    suffixes: [".html", ".htm", "index.html"]
  directories:
    - path: /docs/
      root: public/docs/
```

**Example requests**:
- `GET /docs/installation` → serves `public/docs/installation.html`
- `GET /docs/guides/` → serves `public/docs/guides/index.html`
- `GET /docs/api.json` → serves `public/docs/api.json`

### Multi-Format Content

Serve the same content in multiple formats:

```yaml
static:
  try_files:
    suffixes: [".html", ".xml", ".json", ".txt"]
  directories:
    - path: /content/
      root: public/content/
```

**Example**:
- Request: `GET /content/users/123`
- Tries: `users/123`, `users/123.html`, `users/123.xml`, `users/123.json`, `users/123.txt`
- Falls back to Rails if none found

### SPA Support

Support Single Page Applications with catch-all routing:

```yaml
static:
  try_files:
    enabled: true
    suffixes: ["index.html"]
    fallback: rails
  directories:
    - path: /app/
      root: public/spa/
```

**Behavior**:
- Static assets served directly
- Unknown routes fall back to `index.html` (SPA router handles routing)
- API routes still go to Rails

## Performance Benefits

Try files provides significant performance improvements:

| Content Type | Without Try Files | With Try Files | Improvement |
|--------------|-------------------|----------------|-------------|
| **Static HTML** | ~50ms (Rails) | ~1ms (direct) | 50x faster |
| **Documentation** | ~30ms (Rails) | ~1ms (direct) | 30x faster |
| **JSON APIs** | ~25ms (Rails) | ~1ms (direct) | 25x faster |

## Real-World Example

Navigator serves the showcase application with try files enabled:

```yaml
static:
  try_files:
    enabled: true
    suffixes: ["index.html", ".html", ".htm"]
    fallback: rails
  
  directories:
    - path: /showcase/studios/
      root: public/studios/
    - path: /showcase/regions/
      root: public/regions/
    - path: /showcase/docs/
      root: public/docs/
```

**File structure**:
```
public/
├── studios/
│   ├── boston.html
│   ├── chicago.html
│   └── raleigh.html
├── regions/
│   ├── northeast.html
│   ├── midwest.html
│   └── southeast.html
└── docs/
    ├── index.html
    ├── installation.html
    └── configuration.html
```

**Request examples**:
- `GET /showcase/studios/boston` → `public/studios/boston.html` (direct)
- `GET /showcase/regions/midwest` → `public/regions/midwest.html` (direct)
- `GET /showcase/docs/` → `public/docs/index.html` (with suffix)
- `GET /showcase/admin/users` → Rails application (fallback)

## Configuration Options

### enabled
Enable or disable try files feature.
- **Type**: Boolean
- **Default**: `false`

### suffixes
List of file suffixes to try when exact match fails.
- **Type**: Array of strings
- **Default**: `[]`
- **Example**: `["index.html", ".html", ".htm", ".txt"]`

### fallback
What to do when no file is found.
- **Type**: String
- **Options**: `rails`, `404`
- **Default**: `rails`

## Best Practices

### 1. Order Suffixes by Priority
```yaml
suffixes: ["index.html", ".html", ".htm", ".txt"]
# Most specific first, most general last
```

### 2. Use Appropriate Fallback
```yaml
# For mixed static/dynamic sites
fallback: rails

# For pure static sites
fallback: 404
```

### 3. Combine with Caching
```yaml
static:
  try_files:
    enabled: true
    suffixes: [".html"]
  directories:
    - path: /docs/
      root: public/docs/
      cache: 3600  # Cache for 1 hour
```

### 4. Separate Static from Dynamic
```yaml
# Serve docs statically
- path: /docs/
  root: public/docs/
  
# Serve app routes via Rails  
- path: /app/
  # No root specified = goes to Rails
```

## Debugging Try Files

Enable debug logging to see file resolution:

```bash
LOG_LEVEL=debug navigator config.yml
```

**Debug output**:
```
DEBUG Try files enabled for /docs/installation
DEBUG Trying exact match: public/docs/installation
DEBUG Trying with suffix: public/docs/installation.html
DEBUG File found: public/docs/installation.html
DEBUG Serving static file: public/docs/installation.html
```

## Common Issues

### Files Not Found
**Problem**: Try files not working for expected paths

**Solution**:
1. Verify file exists in specified root directory
2. Check path mapping in configuration
3. Ensure try_files is enabled
4. Check file permissions

```bash
# Test file resolution
ls -la public/docs/installation.html
curl -I http://localhost:3000/docs/installation
```

### Wrong Content Type
**Problem**: Files served with incorrect MIME type

**Solution**: Navigator automatically detects content type based on file extension. Ensure files have proper extensions:

```bash
# Good
installation.html  # → text/html
api.json          # → application/json
styles.css        # → text/css

# Problematic
installation      # → text/plain (generic)
```

### Performance Issues
**Problem**: Static files still slow despite try files

**Solution**:
1. Ensure files are in correct directory structure
2. Add appropriate caching headers
3. Use extensions for better content type detection

```yaml
static:
  directories:
    - path: /docs/
      root: public/docs/
      cache: 86400  # Cache for 24 hours
```

## See Also

- [Static File Serving](../configuration/static-files.md)
- [Configuration Reference](../configuration/yaml-reference.md)
- [Performance Features](index.md)