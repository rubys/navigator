# Redirect Configuration Review

Analysis of redirects in `app/controllers/concerns/configurator.rb` following the directory trailing-slash fix.

## Current Redirects

### 1. Root Path Redirects (Lines 344-351)

```ruby
# With region (Fly.io):
routes['redirects'] << { 'from' => '^/$', 'to' => "#{root}/studios/" }

# Without region, with root prefix:
routes['redirects'] << { 'from' => '^/(showcase)?$', 'to' => "#{root}/studios/" }

# Without region, no root prefix:
routes['redirects'] << { 'from' => '^/$', 'to' => "#{root}/studios/" }
```

**Purpose:** Redirect root URLs to the studios index page.

**Can be retired?** ❌ No - These perform **path transformation** (adding `/studios/`), not just trailing slash management.

**Pattern notes:**
- `^/(showcase)?$` - Clever use of optional group to match both `/` and `/showcase`
- Ensures consistent landing page regardless of entry point

---

### 2. Demo Path Redirect (Line 346)

```ruby
routes['redirects'] << { 'from' => "^#{root}/demo/?$", 'to' => "#{root}/regions/#{region}/demo/" }
```

**Purpose:** Redirect `/showcase/demo` or `/showcase/demo/` to region-specific demo path.

**Can be retired?** ❌ No - Performs **path transformation** (adding `/regions/{region}`), not just trailing slash.

**Pattern notes:**
- `demo/?$` - Handles **optional trailing slash** explicitly with `/?$`
- This shows configurator.rb is aware of trailing slash variations
- **Suggestion:** Navigator could have a "normalize trailing slash" feature to make this simpler

---

## Current Rewrites

### 1. Assets Prefix (Line 355)

```ruby
routes['rewrites'] << { 'from' => '^/assets/(.*)', 'to' => "#{root}/assets/$1" }
```

**Purpose:** Add `/showcase` prefix to asset requests.

**Why needed:** Allows Rails to serve assets at `/showcase/assets/` while users request `/assets/`

---

### 2. Static Files Prefix (Line 360)

```ruby
routes['rewrites'] << {
  'from' => '^/([^/]+\.(gif|png|jpg|jpeg|ico|pdf|svg|webp|txt))$',
  'to' => "#{root}/$1"
}
```

**Purpose:** Add `/showcase` prefix to root-level static files.

**Pattern notes:**
- Matches single-level files only (`[^/]+` ensures no slashes in filename)
- Specific extensions list prevents unintended matches
- **Limitation:** Comment notes Go regex doesn't support negative lookahead

---

## Fly-Replay Routing Patterns

### 1. PDF/XLSX Generation (Lines 388-398)

```ruby
routes['fly']['replay'] << {
  'path' => "^#{root}/.+\\.pdf$",
  'app' => 'smooth-pdf',
  'status' => 307
}
```

**Purpose:** Route document generation to dedicated app.

**Pattern:** `.+\\.pdf$` requires at least one character before `.pdf`

---

### 2. Cross-Region Routing (Lines 416-443)

```ruby
# Region paths
routes['fly']['replay'] << {
  'path' => "^#{root}/regions/#{target_region}/.+$",
  'region' => target_region,
  'status' => 307
}

# Multi-event studios (exclude index, include subpaths)
routes['fly']['replay'] << {
  'path' => "^#{root}/(?:#{year})/(?:#{sites})/.+$",
  'region' => target_region,
  'status' => 307
}

# Single-tenant studios (include index AND subpaths)
routes['fly']['replay'] << {
  'path' => "^#{root}/(?:#{year})/(?:#{sites})(?:/.*)?$",
  'region' => target_region,
  'status' => 307
}
```

**Purpose:** Route requests to appropriate geographic regions.

**Pattern notes:**
- `.+$` - Requires at least one character (excludes bare directory)
- `(?:/.*)?$` - Optional trailing path (includes bare directory)
- Distinction between multi-event (prerendered index) and single-tenant (dynamic index)

---

## Questions & Recommendations

### Can Any Redirects Be Retired?

**No.** All current redirects perform **path transformation**, not just trailing slash normalization:
- Root redirects add `/studios/`
- Demo redirects add `/regions/{region}`

The new directory redirect feature is **complementary**, not a replacement.

---

### Patterns That Suggest Navigator Improvements

#### 1. Optional Trailing Slash Pattern (`/?$`)

**Current:** Line 346 uses `demo/?$` to match with or without trailing slash.

**Suggestion:** Navigator could add a `normalize_trailing_slashes` config option:
```yaml
server:
  static:
    normalize_trailing_slashes: true  # Automatically handle both /path and /path/
```

This would reduce the need for explicit `/?$` patterns in redirects.

---

#### 2. Prefix/Path Rewriting

**Current:** Multiple rewrites add `/showcase` prefix to various paths.

**Observation:** This pattern is common in multi-tenant apps. Navigator already handles this well via `root_path` config, but the rewrites handle edge cases (root-level assets).

**Status:** ✅ Working as designed.

---

#### 3. Go Regex Limitations

**Current:** Line 358 comment notes Go regex doesn't support negative lookahead.

**Note:** This is a Go standard library limitation. The workaround using `[^/]+` (no slashes in match) is appropriate.

**Suggestion:** Document common regex patterns and workarounds for Navigator users.

---

#### 4. Distinction Between Directory Index Patterns

**Observation:**
- Multi-event: `/.+$` (requires subpath, excludes bare `/studio/`)
- Single-tenant: `(?:/.*)?$` (allows bare `/studio/` or subpaths)

**Reason:** Multi-event studios have prerendered static index pages (served by Navigator), so only subpaths need fly-replay. Single-tenant studios need fly-replay for everything including index.

**New capability:** The directory redirect fix now makes prerendered indexes work better! Requests like `/showcase/studios/laval` will:
1. Redirect to `/showcase/studios/laval/` (301)
2. Serve `laval/index.html` statically (no fly-replay needed)

This **improves performance** for the prerendered multi-event studio indexes.

---

## Impact Analysis

### Before Directory Redirect Fix

```
GET /showcase/studios/laval (bot request without trailing slash)
  ↓ No static match
  ↓ Proxy to index server
  ↓ Wake up Rails app (slow, 3.5s)
```

### After Directory Redirect Fix

```
GET /showcase/studios/laval
  ↓ Navigator detects directory with index.html
  ↓ 301 redirect to /showcase/studios/laval/
  ↓ Serve static laval/index.html
  ↓ Fast (0.002s), no Rails wake-up
```

### Combined with Existing Redirects

All redirects continue to work as expected:
- Root redirects still transform path to `/studios/`
- Demo redirects still add region prefix
- Fly-replay still routes to correct regions
- **New:** Directory indexes now work efficiently

---

## Summary

### Can Retire? ❌

**No redirects can be retired** because they perform path transformations, not just trailing slash normalization.

### Suggested Navigator Enhancements

1. **Optional trailing slash normalization** - Config option to automatically handle `/?$` patterns
2. **Regex pattern documentation** - Common patterns and Go regex workarounds
3. **Performance monitoring** - Track redirect → static serve efficiency

### Impact of Directory Fix

✅ **Positive:** Significantly improves performance for prerendered studio index pages
✅ **Compatible:** Works seamlessly with existing redirect configuration
✅ **No conflicts:** Complementary to path transformation redirects

The directory redirect fix is a **performance optimization** that works alongside the existing routing configuration, not a replacement for any current redirects.
