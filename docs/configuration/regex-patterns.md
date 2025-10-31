# Regular Expression Patterns in Navigator

Navigator uses Go's `regexp` package for pattern matching in redirects, rewrites, authentication rules, and fly-replay routing. This guide covers common patterns, limitations, and best practices.

## Table of Contents

1. [Go Regex Basics](#go-regex-basics)
2. [Common Patterns](#common-patterns)
3. [Limitations & Workarounds](#limitations--workarounds)
4. [Real-World Examples](#real-world-examples)
5. [Testing Your Patterns](#testing-your-patterns)

---

## Go Regex Basics

### Syntax

Navigator uses Go's RE2 syntax, which is similar to PCRE but with important differences:

```yaml
# Basic pattern matching
pattern: '^/path/to/resource$'   # Exact match
pattern: '/api/.*'                # Match anything starting with /api/
pattern: '\d+$'                   # Numbers at end of string
```

### Anchors

- `^` - Start of string
- `$` - End of string
- `\b` - Word boundary

**Important:** Always use `^` and `$` to prevent partial matches:

```yaml
# ❌ BAD: Matches anywhere in path
pattern: 'admin'  # Matches /user/admin, /admin-panel, /administrator

# ✅ GOOD: Explicit anchoring
pattern: '^/admin/'  # Only matches paths starting with /admin/
```

---

## Common Patterns

### 1. Optional Trailing Slash

**Pattern:** `/?$`

```yaml
# Matches both /demo and /demo/
redirects:
  - from: '^/showcase/demo/?$'
    to: '/showcase/regions/iad/demo/'
```

**Note:** With Navigator's `normalize_trailing_slashes: true`, this pattern is less necessary for directories.

---

### 2. Path Prefix Matching

**Pattern:** `^/prefix/`

```yaml
# Match all paths under /showcase/studios/
auth:
  public_paths:
    - '/showcase/studios/'  # Trailing slash = prefix match
```

---

### 3. Exact Path Matching

**Pattern:** `^/exact/path$`

```yaml
# Match only exactly /showcase/demo (no trailing slash, no subpaths)
redirects:
  - from: '^/showcase/demo$'
    to: '/showcase/regions/iad/demo/'
```

---

### 4. Optional Path Segment

**Pattern:** `(/segment)?`

```yaml
# Match both / and /showcase
redirects:
  - from: '^/(showcase)?$'
    to: '/showcase/studios/'

# Examples matched:
# - /            → redirects to /showcase/studios/
# - /showcase    → redirects to /showcase/studios/
```

---

### 5. Capture Groups & Substitution

**Pattern:** `(...)` with `$1`, `$2` etc.

```yaml
# Rewrite /assets/* to /showcase/assets/*
rewrites:
  - from: '^/assets/(.*)'
    to: '/showcase/assets/$1'

# /assets/app.js → /showcase/assets/app.js
# /assets/images/logo.png → /showcase/assets/images/logo.png
```

---

### 6. File Extension Matching

**Pattern:** `\.(ext1|ext2|ext3)$`

```yaml
# Match image files at root level
rewrites:
  - from: '^/([^/]+\.(gif|png|jpg|jpeg|ico|svg|webp))$'
    to: '/showcase/$1'

# /logo.png → /showcase/logo.png
# /favicon.ico → /showcase/favicon.ico
```

**Key parts:**
- `[^/]+` - One or more non-slash characters (filename)
- `\.` - Literal dot (escaped)
- `(ext1|ext2)` - Alternation (OR)
- `$` - Must be at end

---

### 7. Alternation (OR)

**Pattern:** `(option1|option2|option3)`

```yaml
# Route multiple studios to iad region
fly:
  replay:
    - path: '^/showcase/2025/(boston|raleigh|alexandria)/.+$'
      region: 'iad'
      status: 307
```

**Performance tip:** Group similar patterns:
```yaml
# ❌ LESS EFFICIENT: Multiple patterns
- path: '^/showcase/2025/boston/'
- path: '^/showcase/2025/raleigh/'
- path: '^/showcase/2025/alexandria/'

# ✅ MORE EFFICIENT: Single pattern with alternation
- path: '^/showcase/2025/(boston|raleigh|alexandria)/'
```

---

### 8. Character Classes

```yaml
# Lowercase letters only
pattern: '^/showcase/studios/([a-z]+)$'

# Alphanumeric with hyphens
pattern: '^/showcase/[a-z0-9-]+$'

# Not slash (match filename)
pattern: '[^/]+'
```

---

### 9. Quantifiers

```yaml
# Zero or more: *
pattern: '/path/.*'           # /path/, /path/anything, /path/a/b/c

# One or more: +
pattern: '/studios/.+'        # /studios/x, /studios/abc (but NOT /studios/)

# Optional (zero or one): ?
pattern: '/demo/?$'           # /demo or /demo/

# Exactly n: {n}
pattern: '/\d{4}/'           # /2025/

# Range: {min,max}
pattern: '/[a-z]{3,10}/'     # 3-10 lowercase letters
```

---

### 10. Non-Capturing Groups

**Pattern:** `(?:...)`

```yaml
# Group without creating capture variable
fly:
  replay:
    - path: '^/showcase/(?:2024|2025)/raleigh/'
      region: 'iad'

# No $1, $2 variables created - more efficient
```

---

## Limitations & Workarounds

### 1. No Negative Lookahead

**Problem:** Go regex doesn't support `(?!pattern)` negative lookahead.

**Example issue:**
```yaml
# ❌ DOESN'T WORK: Negative lookahead
pattern: '^/(?!showcase)'  # "Match / not followed by showcase"
```

**Workaround:** Use character classes to exclude specific characters:

```yaml
# ✅ WORKS: Match root-level files (no slashes in name)
pattern: '^/([^/]+\.png)$'
```

**Real-world example from showcase:**
```ruby
# Want: Root-level images that DON'T start with /showcase
# Go doesn't support: '^/(?!showcase)(.+\.(png|jpg))$'
# Workaround: Match single-level only (no slashes in filename)
pattern: '^/([^/]+\.(gif|png|jpg|jpeg|ico|pdf|svg|webp|txt))$'
```

---

### 2. No Lookbehind

**Problem:** Go regex doesn't support `(?<=pattern)` or `(?<!pattern)`.

**Workaround:** Use capture groups and rewrite:

```yaml
# Want: Everything after /showcase/
# Can't use: '(?<=/showcase/).*'
# Instead: Use capture group
rewrites:
  - from: '^/showcase/(.*)$'
    to: '/$1'  # Extracts everything after /showcase/
```

---

### 3. No Conditional Expressions

**Problem:** Go regex doesn't support `(?(condition)then|else)`.

**Workaround:** Use multiple rules:

```yaml
# Instead of conditional regex, use two rules
redirects:
  - from: '^/$'
    to: '/showcase/studios/'
  - from: '^/showcase$'
    to: '/showcase/studios/'
```

---

### 4. Case Sensitivity

Go regex is **case-sensitive by default**.

```yaml
# ❌ Won't match /Showcase or /SHOWCASE
pattern: '^/showcase'

# ✅ Use character class for case-insensitive
pattern: '^/[Ss][Hh][Oo][Ww][Cc][Aa][Ss][Ee]'

# ✅ Better: Normalize URLs at application level
```

**Recommendation:** Design URLs to be lowercase-only rather than using complex case-insensitive patterns.

---

## Real-World Examples

### Example 1: Studio Routing with Optional Trailing Slash

```yaml
# Before: Required manual handling of both /studio and /studio/
auth:
  auth_patterns:
    - pattern: '^/showcase/2025/(raleigh|boston|miami)/?$'
      action: 'off'

# After: With normalize_trailing_slashes: true
server:
  static:
    normalize_trailing_slashes: true

auth:
  auth_patterns:
    # Pattern without /? - Navigator handles it
    - pattern: '^/showcase/2025/(raleigh|boston|miami)$'
      action: 'off'
```

---

### Example 2: Multi-Region Routing

```yaml
# Group regions efficiently
fly:
  replay:
    # US East Coast regions
    - path: '^/showcase/regions/(iad|ewr|bos)/'
      region: 'iad'
      status: 307

    # US West Coast regions
    - path: '^/showcase/regions/(sjc|lax|sea)/'
      region: 'sjc'
      status: 307

    # International
    - path: '^/showcase/regions/(syd|sin|nrt)/'
      region: 'syd'
      status: 307
```

---

### Example 3: Version-Specific API Routing

```yaml
# Route API versions to different backends
reverse_proxies:
  - path: '^/api/v1/(.*)$'
    target: 'http://api-v1:8080/$1'

  - path: '^/api/v2/(.*)$'
    target: 'http://api-v2:8080/$1'

  - path: '^/api/(.*)$'  # Default to latest
    target: 'http://api-v2:8080/$1'
```

---

### Example 4: File Type Routing

```yaml
# Route documents to PDF generation service
fly:
  replay:
    - path: '^/showcase/.+\.(pdf|xlsx)$'
      app: 'document-generator'
      status: 307

# Route media to CDN
rewrites:
  - from: '^/(.*\.(jpg|png|gif|mp4|webm))$'
    to: 'https://cdn.example.com/$1'
```

---

### Example 5: Authenticated vs Public Paths

```yaml
auth:
  enabled: true
  htpasswd: /data/htpasswd

  # Public paths (no authentication)
  public_paths:
    - '/showcase/docs/'
    - '/showcase/studios/'
    - '/assets/'
    - '/favicon.ico'
    - '/robots.txt'
    - '*.css'
    - '*.js'
    - '*.png'

  # Pattern-based public paths (more complex)
  auth_patterns:
    # Allow studio index pages but not admin paths
    - pattern: '^/showcase/\d{4}/[a-z-]+/?$'
      action: 'off'

    # Allow public subpaths
    - pattern: '^/showcase/\d{4}/[a-z-]+/[a-z-]+/public/'
      action: 'off'
```

---

## Testing Your Patterns

### Online Tools

Test your patterns at [regex101.com](https://regex101.com/):
1. Select "Golang" flavor
2. Enter your pattern
3. Test against sample URLs
4. Check the "Explanation" tab

### Navigator Debug Mode

Enable debug logging to see pattern matches:

```yaml
# Set via environment
LOG_LEVEL=debug ./navigator config.yml
```

Example debug output:
```
DEBUG Checking auth pattern pattern="^/showcase/studios/" path="/showcase/studios/raleigh" matched=true
DEBUG Trying file extensions path="/showcase/studios/raleigh" extensions=[.html .htm]
```

### Common Test Cases

Always test these cases:

```yaml
# For pattern: '^/showcase/studios/?$'
✓ /showcase/studios      # Should match
✓ /showcase/studios/     # Should match
✗ /showcase/studios/raleigh  # Should NOT match (has subpath)
✗ /showcase/studio       # Should NOT match (missing 's')
✗ /other/studios         # Should NOT match (wrong prefix)
```

---

## Best Practices

### 1. Always Anchor Your Patterns

```yaml
# ❌ BAD: Matches anywhere
pattern: 'admin'

# ✅ GOOD: Explicit start/end
pattern: '^/admin$'
pattern: '^/admin/'
```

### 2. Escape Special Characters

```yaml
# Regex special chars: . ^ $ * + ? { } [ ] \ | ( )
# Always escape with backslash:

pattern: '\.'    # Literal dot
pattern: '\$'    # Literal dollar sign
pattern: '\('    # Literal parenthesis
```

### 3. Use Non-Capturing Groups for Alternation

```yaml
# ❌ LESS EFFICIENT: Capturing group not used
pattern: '(jpg|png|gif)$'

# ✅ MORE EFFICIENT: Non-capturing group
pattern: '(?:jpg|png|gif)$'
```

### 4. Group Similar Patterns

```yaml
# ❌ INEFFICIENT: Multiple similar patterns
- pattern: '^/showcase/2025/boston/'
- pattern: '^/showcase/2025/raleigh/'
- pattern: '^/showcase/2025/miami/'

# ✅ EFFICIENT: Single pattern with alternation
- pattern: '^/showcase/2025/(?:boston|raleigh|miami)/'
```

### 5. Document Complex Patterns

```yaml
auth_patterns:
  # Allow public access to event data without authentication
  # Format: /showcase/YYYY/studio-name/event-name/public/anything
  # Example: /showcase/2025/raleigh/shimmer-shine/public/heats
  - pattern: '^/showcase/\d{4}/[a-z-]+/[a-z-]+/public/'
    action: 'off'
```

---

## Pattern Performance

### Fast Patterns ✅

- Literal strings: `/showcase/studios/`
- Simple prefixes: `^/api/`
- Character classes: `[a-z]+`

### Slower Patterns ⚠️

- Long alternations: `(opt1|opt2|...|opt50)`
- Nested groups: `((a|b)(c|d))+`
- Backtracking: `.*.*` (use `[^/]+` instead)

### Optimization Tips

1. **Order matters**: Put most common patterns first
2. **Combine when possible**: Use alternation instead of multiple rules
3. **Be specific**: `^/api/v1/` is faster than `^/api/.*`

---

## Additional Resources

- [Go regexp documentation](https://pkg.go.dev/regexp/syntax)
- [RE2 syntax reference](https://github.com/google/re2/wiki/Syntax)
- [regex101.com](https://regex101.com/) - Online regex tester (select "Golang")
- [Navigator routing documentation](../features/routing.md)

---

## Quick Reference Card

```yaml
# Anchors
^          Start of string
$          End of string
\b         Word boundary

# Quantifiers
*          Zero or more
+          One or more
?          Zero or one (optional)
{n}        Exactly n times
{n,m}      Between n and m times

# Character Classes
.          Any character
[abc]      One of a, b, or c
[^abc]     Not a, b, or c
[a-z]      Lowercase letters
[0-9]      Digits
\d         Digit (same as [0-9])
\w         Word character ([a-zA-Z0-9_])
\s         Whitespace

# Groups
(...)      Capturing group (creates $1, $2, etc.)
(?:...)    Non-capturing group
(a|b)      Alternation (OR)

# Escapes
\.         Literal dot
\\         Literal backslash
\(         Literal parenthesis
```
