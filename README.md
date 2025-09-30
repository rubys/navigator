# Navigator

A lightweight web server for multi-tenant applications with on-demand process management. Deploy multiple customers or projects from a single configuration file.

**[📚 Full Documentation](https://rubys.github.io/navigator/)**

## Quick Start

### Installation

**Docker** (recommended):
```dockerfile
COPY --from=samruby/navigator:latest /navigator /usr/local/bin/navigator
CMD ["navigator", "config/navigator.yml"]
```

**Download Binary**:
```bash
# Linux (AMD64)
curl -L https://github.com/rubys/navigator/releases/latest/download/navigator-linux-amd64.tar.gz | tar xz

# macOS (ARM64)
curl -L https://github.com/rubys/navigator/releases/latest/download/navigator-darwin-arm64.tar.gz | tar xz
```

**Build from Source**:
```bash
git clone https://github.com/rubys/navigator.git
cd navigator
make build
```

### Minimal Configuration

Create `config/navigator.yml`:

```yaml
server:
  listen: 3000
  public_dir: ./public

applications:
  tenants:
    - name: myapp
      path: /
      working_dir: /path/to/app
```

### Run Navigator

```bash
# With config file
./bin/navigator-refactored config/navigator.yml

# Reload configuration (no restart)
./bin/navigator-refactored -s reload
```

## What Can Navigator Do?

- **Multi-tenant hosting** - Isolated databases and processes per customer
- **Cost optimization** - Auto-suspend idle machines on Fly.io
- **WebSocket support** - Built-in Action Cable and WebSocket proxying
- **Smart routing** - Regional routing with Fly-Replay
- **Static file serving** - High-performance direct filesystem access
- **Authentication** - htpasswd support with flexible exclusions
- **Process management** - Automatic startup, shutdown, and recovery

## Documentation

- **[Getting Started](https://rubys.github.io/navigator/getting-started/)** - Installation and first app
- **[Configuration Reference](https://rubys.github.io/navigator/configuration/yaml-reference/)** - Complete YAML options
- **[Examples](https://rubys.github.io/navigator/examples/)** - Copy-paste configurations
- **[Use Cases](https://rubys.github.io/navigator/use-cases/)** - Real-world patterns
- **[Features](https://rubys.github.io/navigator/features/)** - Detailed feature guides
- **[Architecture](https://rubys.github.io/navigator/architecture/)** - Technical details

## Production Ready

Trusted in production serving 75+ customers across 8 countries with:
- 81.2% test coverage
- Cross-platform support (Linux, macOS, Windows)
- Comprehensive documentation
- Active development

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Links

- [Documentation](https://rubys.github.io/navigator/)
- [Releases](https://github.com/rubys/navigator/releases)
- [Issues](https://github.com/rubys/navigator/issues)
- [Discussions](https://github.com/rubys/navigator/discussions)