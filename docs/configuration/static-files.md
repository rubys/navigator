# Static File Serving

Navigator can serve static files directly from the filesystem, bypassing Rails for better performance. This includes assets, images, documents, and any other static content.

## Basic Configuration

```yaml
static:
  directories:
    - path: /assets/
      root: public/assets/
      cache: 86400
  extensions: [css, js, png, jpg, gif, ico]
```

## Directory Mappings

Map URL paths to filesystem directories:

```yaml
static:
  directories:
    # Rails assets (precompiled)
    - path: /assets/
      root: public/assets/
      cache: 31536000  # 1 year (fingerprinted files)
    
    # Webpack/Vite assets
    - path: /packs/
      root: public/packs/
      cache: 31536000
    
    # User uploads
    - path: /uploads/
      root: storage/uploads/
      cache: 3600  # 1 hour
      
    # Documentation
    - path: /docs/
      root: public/docs/
      cache: 86400  # 1 day
```

### Directory Configuration

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `path` | string | ✓ | URL path (must start and end with /) |
| `root` | string | ✓ | Filesystem directory path |
| `cache` | integer | | Cache-Control max-age in seconds |

## File Extensions

Serve files with specific extensions directly:

```yaml
static:
  extensions: [
    # Web assets
    css, js, map,
    
    # Images
    png, jpg, jpeg, gif, ico, svg, webp,
    
    # Fonts
    woff, woff2, ttf, eot, otf,
    
    # Documents
    pdf, txt, xml, json,
    
    # Audio/Video
    mp3, mp4, webm, ogg
  ]
```

Files matching these extensions are served directly regardless of path.

## Try Files Behavior

Enable nginx-style try_files for better static site support:

```yaml
static:
  try_files:
    enabled: true
    suffixes: [".html", "index.html", ".htm", ".txt", ".xml", ".json"]
    fallback: rails
```

### How Try Files Works

1. Request comes in: `/about`
2. Navigator tries in order:
   - `/about` (exact file)
   - `/about.html`
   - `/about/index.html`
   - `/about.htm`
   - `/about.txt`
   - `/about.xml`
   - `/about.json`
3. If no file found, falls back to Rails

### Try Files Configuration

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | boolean | `false` | Enable try_files behavior |
| `suffixes` | array | `[]` | File suffixes to try |
| `fallback` | string | `"rails"` | What to do when no file found |

## Cache Control

Set appropriate cache headers for different content types:

```yaml
static:
  directories:
    # Long cache for fingerprinted assets
    - path: /assets/
      root: public/assets/
      cache: 31536000  # 1 year
      
    # Medium cache for images
    - path: /images/
      root: public/images/
      cache: 86400     # 1 day
      
    # Short cache for dynamic content
    - path: /api/docs/
      root: public/api-docs/
      cache: 300       # 5 minutes
      
    # No cache for development
    - path: /dev/
      root: public/dev/
      cache: 0         # Always revalidate
```

### Cache Duration Guidelines

| Content Type | Duration | Seconds | Use Case |
|--------------|----------|---------|----------|
| **Immutable assets** | 1 year | 31536000 | Fingerprinted files |
| **Semi-static** | 1 week | 604800 | Images, fonts |
| **Dynamic** | 1 hour | 3600 | Generated content |
| **Real-time** | 5 minutes | 300 | API docs |
| **Development** | No cache | 0 | Local development |

## MIME Types

Navigator automatically detects MIME types based on file extensions:

| Extension | MIME Type | Description |
|-----------|-----------|-------------|
| `.html`, `.htm` | `text/html` | HTML documents |
| `.css` | `text/css` | Stylesheets |
| `.js` | `application/javascript` | JavaScript |
| `.json` | `application/json` | JSON data |
| `.xml` | `application/xml` | XML documents |
| `.png` | `image/png` | PNG images |
| `.jpg`, `.jpeg` | `image/jpeg` | JPEG images |
| `.gif` | `image/gif` | GIF images |
| `.svg` | `image/svg+xml` | SVG images |
| `.ico` | `image/x-icon` | Icons |
| `.pdf` | `application/pdf` | PDF documents |
| `.woff` | `font/woff` | Web fonts |
| `.woff2` | `font/woff2` | Web fonts v2 |

## Performance Optimization

### 1. Serve Assets Directly

```yaml
# Instead of letting Rails serve assets
static:
  directories:
    - path: /assets/
      root: public/assets/
      cache: 31536000
  extensions: [css, js, png, jpg, gif]

applications:
  global_env:
    RAILS_SERVE_STATIC_FILES: "false"  # Let Navigator handle it
```

### 2. Optimize Directory Structure

```bash
# Good: Organized by type
public/
├── assets/          # Compiled assets
├── images/          # Static images
├── fonts/           # Web fonts
└── docs/            # Documentation

# Avoid: Mixed content
public/
├── style.css        # Mixed with HTML
├── app.js
├── logo.png
└── about.html
```

### 3. Use Compression

```bash
# Pre-compress static files
gzip -k -9 public/assets/*.css
gzip -k -9 public/assets/*.js
brotli -k -9 public/assets/*.css
brotli -k -9 public/assets/*.js
```

Navigator will serve pre-compressed files when available.

## Common Patterns

### Rails Application with Assets

```yaml
static:
  directories:
    # Precompiled Rails assets
    - path: /assets/
      root: public/assets/
      cache: 31536000
      
    # Webpack packs (if using Webpacker)
    - path: /packs/
      root: public/packs/
      cache: 31536000
      
  extensions: [css, js, png, jpg, gif, ico, woff, woff2]
  
  # Enable try_files for public pages
  try_files:
    enabled: true
    suffixes: ["index.html", ".html"]
    fallback: rails

applications:
  global_env:
    RAILS_SERVE_STATIC_FILES: "false"
```

### Static Site with Rails API

```yaml
static:
  directories:
    # Static site files
    - path: /
      root: public/dist/
      cache: 3600
      
    # Assets with long cache
    - path: /assets/
      root: public/dist/assets/
      cache: 31536000
      
  try_files:
    enabled: true
    suffixes: ["index.html", "/index.html"]
    fallback: rails

applications:
  tenants:
    # API only handles /api/ paths
    - name: api
      path: /api/
```

### Multi-Tenant with Shared Assets

```yaml
static:
  directories:
    # Shared assets for all tenants
    - path: /shared/
      root: public/shared/
      cache: 86400
      
    # Tenant-specific assets
    - path: /tenant1/assets/
      root: storage/tenants/tenant1/assets/
      cache: 3600
      
    - path: /tenant2/assets/
      root: storage/tenants/tenant2/assets/
      cache: 3600
```

### Development vs Production

=== "Development"

    ```yaml
    static:
      directories:
        - path: /assets/
          root: public/assets/
          cache: 0  # No cache for development
      extensions: [css, js, png, jpg]
    ```

=== "Production"

    ```yaml
    static:
      directories:
        - path: /assets/
          root: public/assets/
          cache: 31536000  # Long cache
        - path: /packs/
          root: public/packs/
          cache: 31536000
      extensions: [css, js, map, png, jpg, gif, ico, svg, woff, woff2, ttf, eot]
    ```

## Security Considerations

### 1. Prevent Directory Traversal

Navigator automatically prevents `..` path traversal attacks, but ensure your directory structure is secure:

```yaml
# Safe
static:
  directories:
    - path: /public/
      root: public/files/  # Contained directory

# Potentially unsafe
static:
  directories:
    - path: /files/
      root: /  # Root filesystem access
```

### 2. Serve Only Intended Files

```yaml
# Use specific extensions
static:
  extensions: [css, js, png, jpg]  # Only these types

# Avoid serving all files
# extensions: ["*"]  # Don't do this
```

### 3. Exclude Sensitive Directories

```yaml
static:
  directories:
    - path: /assets/
      root: public/assets/  # OK - public assets
    
# Avoid
# - path: /config/
#   root: config/         # Contains secrets
# - path: /logs/
#   root: log/           # Contains sensitive data
```

## Troubleshooting

### Files Not Being Served

1. **Check file exists**:
   ```bash
   ls -la public/assets/application.css
   ```

2. **Verify path mapping**:
   ```yaml
   static:
     directories:
       - path: /assets/        # URL path
         root: public/assets/  # File system path
   ```

3. **Test directly**:
   ```bash
   curl -I http://localhost:3000/assets/application.css
   # Look for "X-Served-By: Navigator" header
   ```

### Wrong MIME Type

Navigator uses file extensions for MIME type detection. Ensure files have correct extensions:

```bash
# Correct
app.js        → application/javascript
style.css     → text/css
image.png     → image/png

# Problematic  
app.js.txt    → text/plain (wrong!)
```

### Cache Issues

1. **Check cache headers**:
   ```bash
   curl -I http://localhost:3000/assets/app.css
   # Look for Cache-Control header
   ```

2. **Force refresh**:
   ```bash
   curl -H "Cache-Control: no-cache" http://localhost:3000/assets/app.css
   ```

3. **Clear browser cache** or use incognito mode

### Permission Errors

```bash
# Ensure Navigator can read files
chmod 644 public/assets/*
chmod 755 public/assets/

# Check ownership
ls -la public/assets/
```

## Monitoring

### Check Static File Serving

```bash
# Monitor static file requests
grep "static" /var/log/navigator.log

# Count static vs dynamic requests
grep -c "static" /var/log/navigator.log
grep -c "proxy" /var/log/navigator.log
```

### Performance Testing

```bash
# Test static file performance
ab -n 1000 -c 10 http://localhost:3000/assets/application.css

# Compare with Rails serving
ab -n 1000 -c 10 http://localhost:3000/non-static-path
```

## Migration from nginx

Migrate nginx static file configurations:

```nginx
# nginx
location /assets/ {
    root /var/www/app/public;
    expires 1y;
    add_header Cache-Control "public, immutable";
}

location ~* \.(css|js|png|jpg)$ {
    root /var/www/app/public;
    expires 1w;
}
```

Becomes:

```yaml
# Navigator
static:
  directories:
    - path: /assets/
      root: /var/www/app/public/assets/
      cache: 31536000  # 1 year
  extensions: [css, js, png, jpg]
  # 1 week cache applied to all extensions
```

## See Also

- [Configuration Overview](index.md)
- [Server Settings](server.md)  
- [Try Files Feature](../features/try-files.md)
- [Single Tenant Example](../examples/single-tenant.md)