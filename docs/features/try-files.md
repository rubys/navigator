# Try Files Feature

Navigator's try files feature allows serving static content directly from the filesystem with flexible file resolution.

## How It Works

When a request comes in, Navigator attempts to serve files in this order:

1. **Exact match**: Look for the exact file path
2. **With suffixes**: Try adding configured suffixes (`.html`, `.htm`, etc.)
3. **Index files**: Look for index files in directories
4. **Fallback**: Fall back to Rails application if no file found

## Configuration

Navigator supports two configuration approaches:

### Modern Server-Based Configuration (Recommended)

The simplified approach configures try files directly in the `server` section:

```yaml
server:
  public_dir: public  # Base directory for static files

  # Try files configuration - if present, feature is enabled
  try_files: [index.html, .html, .htm, .txt, .xml, .json]

  # Optional: restrict allowed file extensions
  allowed_extensions: [html, htm, txt, xml, json, css, js]
```

**Key points:**
- If `try_files` is present, the feature is **enabled**
- If `try_files` is absent, the feature is **disabled**
- Each suffix is tried in order until a file is found
- If no file matches, request falls back to the application

### Legacy Configuration (Backward Compatible)

The original `static` section is still supported:

```yaml
static:
  try_files:
    enabled: true
    suffixes: [index.html, .html, .htm, .txt, .xml, .json]

  directories:
    - path: /assets/
      dir: assets/
    - path: /docs/
      dir: docs/
```

## Use Cases

### Static Site Content

Perfect for serving documentation, marketing pages, or static content:

```yaml
server:
  public_dir: public

  # Enable try files for docs
  try_files: [index.html, .html, .htm]

  # Optional: restrict to documentation file types
  allowed_extensions: [html, htm, json, txt]
```

**Example requests**:
- `GET /docs/installation` → serves `public/docs/installation.html`
- `GET /docs/guides/` → serves `public/docs/guides/index.html`
- `GET /docs/api.json` → serves `public/docs/api.json`

### Multi-Format Content

Serve the same content in multiple formats:

```yaml
server:
  public_dir: public

  # Try multiple format suffixes
  try_files: [.html, .xml, .json, .txt]

  allowed_extensions: [html, xml, json, txt]
```

**Example**:
- Request: `GET /content/users/123`
- Tries: `users/123`, `users/123.html`, `users/123.xml`, `users/123.json`, `users/123.txt`
- Falls back to Rails if none found

### SPA Support

Support Single Page Applications with catch-all routing:

```yaml
server:
  public_dir: public/spa

  # Try index.html for SPA routing
  try_files: [index.html]

  # Allow all SPA assets
  allowed_extensions: [html, js, css, png, jpg, svg, woff, woff2]
```

**Behavior**:
- Static assets served directly
- Unknown routes fall back to `index.html` (SPA router handles routing)
- API routes can still go to Rails via path-specific routing

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
server:
  public_dir: public

  # Try files in order
  try_files: [index.html, .html, .htm]

  # Allow static content types
  allowed_extensions: [html, htm, css, js, png, jpg, svg]

  # Cache static content
  cache_control:
    overrides:
      - path: /studios/
        max_age: 1h
      - path: /regions/
        max_age: 1h
      - path: /docs/
        max_age: 24h
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
- `GET /studios/boston` → `public/studios/boston.html` (via .html suffix)
- `GET /regions/midwest` → `public/regions/midwest.html` (via .html suffix)
- `GET /docs/` → `public/docs/index.html` (via index.html suffix)
- `GET /admin/users` → Rails application (fallback)

## Configuration Options

### Modern Configuration (server.try_files)

**Location**: `server.try_files`

**Type**: Array of strings (or omitted)

**Behavior**:
- **Present**: Try files feature is enabled
- **Absent**: Try files feature is disabled

**Example values**:
```yaml
# Enable with multiple suffixes
try_files: [index.html, .html, .htm, .txt]

# Enable with single suffix
try_files: [.html]

# Disable (omit entirely)
# try_files: ...
```

**Order matters**: Suffixes are tried in the order specified. Put most specific first.

### Legacy Configuration (static.try_files)

For backward compatibility, the legacy format is still supported:

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | boolean | `false` | Enable/disable try files |
| `suffixes` | array | `[]` | Suffixes to try |

## Best Practices

### 1. Order Suffixes by Priority
```yaml
server:
  try_files: [index.html, .html, .htm, .txt]
  # Most specific first, most general last
```

### 2. Combine with Caching
```yaml
server:
  public_dir: public
  try_files: [.html]

  # Add cache headers for static content
  cache_control:
    overrides:
      - path: /docs/
        max_age: 1h
```

### 3. Restrict File Types for Security
```yaml
server:
  try_files: [.html, .htm]

  # Only serve safe file types
  allowed_extensions: [html, htm, css, js, png, jpg]
```

### 4. Use Different Settings for Different Environments
```yaml
# Production: strict and cached
server:
  try_files: [.html]
  allowed_extensions: [html, css, js, png, jpg]
  cache_control:
    overrides:
      - path: /
        max_age: 24h

# Development: permissive and uncached
server:
  try_files: [.html, .htm, .txt]
  # No allowed_extensions = all files allowed
  # No cache_control = no caching
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
1. Verify file exists in `public_dir`
2. Ensure `try_files` is present in configuration
3. Check that file extension is in `allowed_extensions` (if specified)
4. Check file permissions

```bash
# Test file resolution
ls -la public/docs/installation.html
curl -I http://localhost:3000/docs/installation

# Check configuration
grep -A5 "try_files:" navigator.yml
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
1. Ensure files are in `public_dir`
2. Add appropriate caching headers
3. Use proper file extensions for content type detection

```yaml
server:
  public_dir: public
  try_files: [.html]

  # Add caching for performance
  cache_control:
    overrides:
      - path: /docs/
        max_age: 24h
```

## See Also

- [Static File Serving](../configuration/static-files.md)
- [Configuration Reference](../configuration/yaml-reference.md)
- [Performance Features](index.md)