# portwatch

A lightweight daemon that monitors open ports and alerts on unexpected changes via webhook or email.

## Installation

```bash
go install github.com/yourusername/portwatch@latest
```

Or build from source:

```bash
git clone https://github.com/yourusername/portwatch.git && cd portwatch && go build -o portwatch .
```

## Usage

Create a config file (`config.yaml`):

```yaml
interval: 60s
baseline:
  - 22
  - 80
  - 443
alerts:
  webhook: "https://hooks.example.com/notify"
  email: "ops@example.com"
```

Run the daemon:

```bash
portwatch --config config.yaml
```

portwatch will scan open ports at the specified interval and send an alert whenever a port appears or disappears outside your defined baseline.

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--config` | `config.yaml` | Path to config file |
| `--interval` | `60s` | Scan interval |
| `--once` | `false` | Run a single scan and exit |

### Example Alert Payload

```json
{
  "event": "port_opened",
  "port": 8080,
  "timestamp": "2024-11-01T12:34:56Z"
}
```

## License

MIT © 2024 yourusername