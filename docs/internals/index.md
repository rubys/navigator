# Internals

This section provides in-depth technical documentation about Navigator's internal architecture and request handling.

## Available Documentation

### [Request Flow](request-flow.md)

Comprehensive explanation of how Navigator processes HTTP requests from start to finish.

**Topics covered:**

- 10-step request processing pipeline
- Authentication flow with auth_patterns
- Static file serving and try_files
- Web application proxying
- Sticky sessions and Fly-Replay routing
- WebSocket handling
- Error recovery and retry logic
- Performance optimizations

**Useful for:**

- Understanding how requests are routed
- Debugging authentication issues
- Optimizing application configuration
- Learning about Navigator's architecture

## Why This Matters

Understanding Navigator's request flow helps you:

1. **Configure authentication correctly** - Know when auth_patterns vs public_paths are checked
2. **Optimize performance** - Use the right patterns for your use case
3. **Debug issues** - Understand where requests might be blocked or delayed
4. **Design better apps** - Leverage Navigator's features effectively

## Related Documentation

- [Authentication Configuration](../configuration/authentication.md) - How to configure auth
- [Static Files](../configuration/static-files.md) - Static file serving configuration
- [Routing](../configuration/routing.md) - Rewrite rules and redirects
- [Features Overview](../features/index.md) - High-level feature documentation
