# Getting Started

Get Navigator up and running in minutes with your Rails application.

## Prerequisites

- Go 1.21+ (if building from source)
- A Rails application
- Linux, macOS, or Windows (with WSL)

## Installation Options

### Option 1: Download Pre-built Binary (Recommended)

Download the latest release for your platform:

```bash
# Linux (AMD64)
curl -L https://github.com/rubys/navigator/releases/latest/download/navigator-linux-amd64.tar.gz | tar xz

# macOS (Apple Silicon)
curl -L https://github.com/rubys/navigator/releases/latest/download/navigator-darwin-arm64.tar.gz | tar xz

# macOS (Intel)
curl -L https://github.com/rubys/navigator/releases/latest/download/navigator-darwin-amd64.tar.gz | tar xz
```

### Option 2: Build from Source

```bash
git clone https://github.com/rubys/navigator.git
cd navigator
make build
```

### Option 3: Docker

```bash
docker pull rubys/navigator:latest
```

## Quick Start Guide

### 1. Create Configuration

Create a `navigator.yml` file:

```yaml
server:
  listen: 3000
  public_dir: ./public

applications:
  tenants:
    - name: myapp
      path: /
      working_dir: /path/to/your/rails/app
```

### 2. Start Navigator

```bash
./navigator navigator.yml
```

### 3. Access Your Application

Open http://localhost:3000 in your browser.

## What's Next?

- [Configure your first application](first-app.md)
- [Learn about configuration options](basic-config.md)
- [Explore examples](../examples/index.md)