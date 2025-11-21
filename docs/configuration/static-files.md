# Static File Serving

Navigator can serve static files directly from the filesystem, bypassing Rails for better performance. This includes assets, images, documents, and any other static content.

## Basic Configuration

```yaml
server:
  listen: 3000
  static:
    public_dir: "./public"
    allowed_extensions: [css, js, png, jpg, gif, ico]
    try_files: [index.html, .html, .htm]
    cache_control:
      overrides:
        - path: /assets/
          max_age: 24h
```

## Public Directory

Navigator serves static files from a configured public directory:

```yaml
server:
  static:
    public_dir: "./public"  # Directory containing static files
```

All static files are served from this single directory. Navigator maps URL paths directly to files within this directory.

### Public Directory Configuration

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `public_dir` | string | `"./public"` | Directory path for static files |

**Examples:**
- URL `/assets/app.css` → File `public/assets/app.css`
- URL `/images/logo.png` → File `public/images/logo.png`
- URL `/favicon.ico` → File `public/favicon.ico`

## File Extensions

Optionally restrict which file extensions Navigator will serve:

```yaml
server:
  static:
    allowed_extensions: [
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

When `allowed_extensions` is specified, only files with these extensions will be served. If omitted, all file types are allowed.

## Try Files Behavior

Enable nginx-style try_files for better static site support:

```yaml
server:
  static:
    try_files: [".html", "index.html", ".htm"]
```

### How Try Files Works

When `try_files` is configured, Navigator attempts to find files with different extensions:

1. Request comes in: `/about`
2. Navigator tries in order:
   - `/about` (exact file)
   - `/about.html`
   - `/about/index.html`
   - `/about.htm`
3. If no file found, falls back to the Rails application

### Try Files Configuration

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `try_files` | array | `[]` | File suffixes to try (empty = disabled) |

**Examples:**
- `try_files: [".html"]` - Try `.html` extension
- `try_files: ["index.html", ".html"]` - Try index.html first, then .html
- `try_files: []` - Disabled (exact path match only)

## Cache Control

Set appropriate cache headers for different paths:

```yaml
server:
  static:
    cache_control:
      default: "0"              # Default: always revalidate HTML
      default_immutable: false  # Default: not immutable
      overrides:
        # Long cache for fingerprinted assets with immutable directive
        - path: /assets/
          max_age: 1y           # 1 year (31536000 seconds)
          immutable: true       # Never changes (fingerprinted)

        # Medium cache for images
        - path: /images/
          max_age: 24h          # 1 day

        # Short cache for dynamic content
        - path: /docs/
          max_age: 5m           # 5 minutes

        # No cache for development
        - path: /dev/
          max_age: 0            # Always revalidate
```

### Immutable Directive

The `immutable` directive tells browsers that fingerprinted assets will **never change**:

```yaml
cache_control:
  overrides:
    - path: /assets/
      max_age: 1y         # Cache for 1 year
      immutable: true     # Content never changes
```

**Result:** `Cache-Control: public, max-age=31536000, immutable`

**Benefits:**
- **No revalidation requests** - Browsers never check if file changed
- **Maximum performance** - Zero server requests during cache lifetime
- **Safe for fingerprinted assets** - File hash in name ensures uniqueness

**Best for:**
- Rails asset pipeline files: `application-4a860118.js`
- Webpacker/Shakapacker: `packs/application-abc123.js`
- Fingerprinted images: `logo-def456.png`

**Not for:**
- Non-fingerprinted files (will serve stale content)
- HTML pages (should always revalidate)

### Cache Duration Guidelines

| Content Type | Duration | Immutable | Format | Use Case |
|--------------|----------|-----------|--------|----------|
| **Fingerprinted assets** | 1 year | ✅ Yes | `1y` + `immutable: true` | Rails assets, Webpacker |
| **HTML pages** | 0 | ❌ No | `0` | Always revalidate |
| **Semi-static** | 1 week | ❌ No | `168h` | Images, fonts |
| **Dynamic** | 1 hour | ❌ No | `1h` | Generated content |
| **Real-time** | 5 minutes | ❌ No | `5m` | API docs |

**Duration formats:**
- Years: `1y` = 31536000 seconds (365 days)
- Weeks: `1w` = 604800 seconds (7 days)
- Days: `1d` = 86400 seconds (24 hours)
- Hours: `24h` = 1 day
- Minutes: `5m` = 5 minutes
- Seconds: `30s` = 30 seconds
- Fractional: `1.5y` = 1.5 years
- Raw: `0` = always revalidate

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
server:
  static:
    public_dir: "./public"
    allowed_extensions: [css, js, png, jpg, gif]
    cache_control:
      overrides:
        - path: /assets/
          max_age: 8760h  # 1 year

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
server:
  listen: 3000
  static:
    public_dir: "./public"
    allowed_extensions: [css, js, png, jpg, gif, ico, woff, woff2]
    try_files: ["index.html", ".html"]
    cache_control:
      default: "0"  # HTML: always revalidate
      overrides:
        - path: /assets/
          max_age: 1y         # 1 year
          immutable: true     # Fingerprinted, never changes
        - path: /packs/
          max_age: 1y         # 1 year
          immutable: true     # Fingerprinted, never changes

applications:
  global_env:
    RAILS_SERVE_STATIC_FILES: "false"
```

### Static Site with Rails API

```yaml
server:
  static:
    public_dir: "./public/dist"
    try_files: ["index.html", "/index.html"]
    cache_control:
      default: "0"  # HTML: always revalidate
      overrides:
        - path: /assets/
          max_age: 1y         # 1 year
          immutable: true     # Fingerprinted, never changes

applications:
  tenants:
    # API only handles /api/ paths
    - name: api
      path: /api/
```

### Multi-Tenant with Shared Assets

```yaml
server:
  static:
    public_dir: "./public"
    cache_control:
      overrides:
        # Shared assets for all tenants
        - path: /shared/
          max_age: 24h

        # Tenant-specific assets
        - path: /tenant1/assets/
          max_age: 1h

        - path: /tenant2/assets/
          max_age: 1h
```

### Development vs Production

=== "Development"

    ```yaml
    server:
      static:
        public_dir: "./public"
        allowed_extensions: [css, js, png, jpg]
        cache_control:
          overrides:
            - path: /assets/
              max_age: 0s  # No cache for development
    ```

=== "Production"

    ```yaml
    server:
      static:
        public_dir: "./public"
        allowed_extensions: [css, js, map, png, jpg, gif, ico, svg, woff, woff2, ttf, eot]
        cache_control:
          default: "0"  # HTML: always revalidate
          overrides:
            - path: /assets/
              max_age: 1y         # 1 year
              immutable: true     # Fingerprinted, never changes
            - path: /packs/
              max_age: 1y         # 1 year
              immutable: true     # Fingerprinted, never changes
    ```

## Security Considerations

### 1. Prevent Directory Traversal

Navigator automatically prevents `..` path traversal attacks, but ensure your directory structure is secure:

```yaml
# Safe
server:
  static:
    public_dir: public/  # Contained directory

# Potentially unsafe
server:
  static:
    public_dir: /  # Root filesystem access (avoid!)
```

### 2. Serve Only Intended Files

```yaml
# Use specific extensions
server:
  static:
    allowed_extensions: [css, js, png, jpg]  # Only these types

# Or omit to allow all files (use with caution)
server:
  static:
    # allowed_extensions not specified = all files allowed
```

### 3. Exclude Sensitive Directories

```yaml
# Safe - only serve from public directory
server:
  static:
    public_dir: public/  # Only files in public/ are accessible

# Ensure sensitive files are outside public_dir:
# - config/ (contains secrets)
# - log/ (contains sensitive data)
# - db/ (database files)
```

## Troubleshooting

### Files Not Being Served

1. **Check file exists**:
   ```bash
   ls -la public/assets/application.css
   ```

2. **Verify public directory**:
   ```yaml
   server:
     static:
       public_dir: public/  # Must contain the file
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
server:
  static:
    public_dir: /var/www/app/public
    allowed_extensions: [css, js, png, jpg]
    cache_control:
      overrides:
        - path: /assets/
          max_age: 1y          # 1 year
          immutable: true      # Equivalent to nginx "immutable"
```

## See Also

- [Configuration Overview](index.md)
- [Server Settings](server.md)  
- [Try Files Feature](../features/try-files.md)
- [Single Tenant Example](../examples/single-tenant.md)