# Opensourcer CLI

A command-line tool to deploy self-hosted open source software with a single command.

## Installation

```bash
go install github.com/opengittr/opensourcer-cli@latest
```

Or build from source:

```bash
git clone https://github.com/opengittr/opensourcer-cli.git
cd opensourcer-cli
go build -o opensourcer .
```

## Quick Start

```bash
# Update the software catalog
opensourcer update

# List available software
opensourcer catalog

# Get info about a software
opensourcer info plausible

# Deploy locally (requires Docker)
opensourcer deploy plausible
```

## Commands

| Command | Description |
|---------|-------------|
| `catalog` | List available software in the catalog |
| `update` | Update the local catalog from repository |
| `info <software>` | Show details about a software |
| `deploy <software>` | Deploy software locally using Docker |
| `list` | List your deployments |
| `logs <software>` | View logs for a deployment |
| `stop <software>` | Stop a running deployment |
| `start <software>` | Start a stopped deployment |
| `destroy <software>` | Remove a deployment completely |

## Requirements

- Docker and Docker Compose
- Git (for catalog updates)

## How It Works

1. The CLI downloads the software catalog from [opensourcer-catalog](https://github.com/opengittr/opensourcer-catalog)
2. Each software has a `docker-compose.yaml` and configuration
3. Running `deploy` creates a local deployment with auto-generated credentials
4. Deployments are tracked in `~/.opensourcer/deployments.json`

## Configuration

All data is stored in `~/.opensourcer/`:

```
~/.opensourcer/
├── catalog/           # Downloaded software catalog
├── deployments/       # Active deployment directories
└── deployments.json   # Deployment tracking
```

## Contributing

Contributions are welcome! To contribute:

1. Fork this repository
2. Create a feature branch (`git checkout -b feature/my-feature`)
3. Make your changes
4. Run tests and ensure the build passes
5. Submit a pull request

For adding new software to the catalog, please contribute to the [opensourcer-catalog](https://github.com/opengittr/opensourcer-catalog) repository instead.

## License

MIT
