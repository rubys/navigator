# Installation

Navigator can be installed in several ways depending on your needs and environment.

## System Requirements

- **Operating System**: Linux, macOS, or Windows
- **Architecture**: AMD64 or ARM64
- **Memory**: Minimum 128MB (plus Rails app requirements)
- **Go**: Version 1.21+ (only if building from source)

!!! success "Cross-Platform Support"
    Navigator offers **native support** for all major platforms:

    - **Linux**: AMD64, ARM64
    - **macOS**: Intel (AMD64), Apple Silicon (ARM64)
    - **Windows**: AMD64, ARM64 (native Windows support)

    Platform-specific signal handling ensures proper behavior on all operating systems.

## Installation Methods

### Pre-built Binaries

The easiest way to install Navigator is using pre-built binaries from GitHub Releases.

=== "Linux"

    ```bash
    # AMD64
    curl -L https://github.com/rubys/navigator/releases/latest/download/navigator-linux-amd64.tar.gz -o navigator.tar.gz
    tar xzf navigator.tar.gz
    sudo mv navigator /usr/local/bin/
    
    # ARM64
    curl -L https://github.com/rubys/navigator/releases/latest/download/navigator-linux-arm64.tar.gz -o navigator.tar.gz
    tar xzf navigator.tar.gz
    sudo mv navigator /usr/local/bin/
    ```

=== "macOS"

    ```bash
    # Apple Silicon (M1/M2/M3)
    curl -L https://github.com/rubys/navigator/releases/latest/download/navigator-darwin-arm64.tar.gz -o navigator.tar.gz
    tar xzf navigator.tar.gz
    sudo mv navigator /usr/local/bin/
    
    # Intel
    curl -L https://github.com/rubys/navigator/releases/latest/download/navigator-darwin-amd64.tar.gz -o navigator.tar.gz
    tar xzf navigator.tar.gz
    sudo mv navigator /usr/local/bin/
    ```

=== "Windows"

    ```powershell
    # Download the Windows binary
    Invoke-WebRequest -Uri "https://github.com/rubys/navigator/releases/latest/download/navigator-windows-amd64.zip" -OutFile "navigator.zip"
    
    # Extract
    Expand-Archive -Path "navigator.zip" -DestinationPath "."
    
    # Move to a directory in PATH
    Move-Item -Path "navigator.exe" -Destination "C:\Program Files\Navigator\"
    ```

### Building from Source

If you need the latest development version or want to contribute:

```bash
# Prerequisites
go version  # Should be 1.21 or higher

# Clone repository
git clone https://github.com/rubys/navigator.git
cd navigator

# Build (creates both legacy and refactored versions)
make build

# Or build the refactored version directly (recommended)
go build -mod=readonly -o bin/navigator cmd/navigator/main.go

# Install globally (optional)
sudo cp bin/navigator /usr/local/bin/navigator
```

### Docker

Navigator is available as a Docker Hub image containing the Linux AMD64 binary:

#### Using in Your Dockerfile

The recommended way to use Navigator is to copy the binary from the Docker Hub image into your own Docker image:

```dockerfile
# Copy Navigator binary from Docker Hub
COPY --from=samruby/navigator:latest /navigator /usr/local/bin/navigator
RUN chmod +x /usr/local/bin/navigator
```

#### Available Tags

- `samruby/navigator:latest` - Latest stable release
- `samruby/navigator:v1.0.0` - Specific version (replace with actual version)

#### Example Dockerfile

```dockerfile
FROM ruby:3.2-slim

# Copy Navigator from Docker Hub
COPY --from=samruby/navigator:latest /navigator /usr/local/bin/navigator
RUN chmod +x /usr/local/bin/navigator

# Your application setup
WORKDIR /app
COPY . .
RUN bundle install

CMD ["navigator", "config/navigator.yml"]
```

### Package Managers

!!! info "Coming Soon"
    We're working on adding Navigator to popular package managers:
    
    - Homebrew (macOS/Linux)
    - APT (Debian/Ubuntu)
    - YUM (RHEL/CentOS)
    - Snap

## Verifying Installation

After installation, verify Navigator is working:

```bash
# Check version
navigator --version

# Display help
navigator --help
```

## Directory Structure

A typical Navigator installation includes:

```
/usr/local/bin/
└── navigator              # The Navigator binary

/etc/navigator/            # Configuration (optional)
├── navigator.yml         # Main configuration
└── htpasswd             # Authentication file

/var/log/navigator/       # Logs (optional)
└── navigator.log        # Application logs
```

## Permissions

Navigator needs appropriate permissions to:

- Read configuration files
- Write PID files to `/tmp`
- Bind to network ports (may require root for ports < 1024)
- Start Rails processes
- Access Rails application directories

### Running as Non-root

For production, it's recommended to run Navigator as a non-root user:

```bash
# Create a dedicated user
sudo useradd -r -s /bin/false navigator

# Set ownership of application directories
sudo chown -R navigator:navigator /path/to/rails/app

# Use a port > 1024 or set capabilities
sudo setcap 'cap_net_bind_service=+ep' /usr/local/bin/navigator
```

## Next Steps

- [Configure your first application](first-app.md)
- [Explore configuration options](basic-config.md)
- [Set up systemd service](../examples/systemd.md)