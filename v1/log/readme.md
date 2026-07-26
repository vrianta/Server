# Logging

A simple and lightweight logging package for Go with support for colored output, JSON logs, and file logging.

## Installation

```bash
go get github.com/vrianta/gulog
```

## Basic Usage

```go
log.Debug("Debug message")
log.Info("Application started")
log.Success("Operation completed")
log.Warn("Something looks wrong")
log.Error("Something failed")
```

## Formatted Messages

```go
log.Info("Listening on port %d", 8080)
log.Error("Failed to connect: %v", err)
```

## Custom Logger

```go
logger := log.New(&log.Config{
    Format: "json",
    File:   "app.log",
})

logger.Info("Server started")
```

## Configuration

| Field | Description |
|-------|-------------|
| `NoLog` | Disable console output. |
| `Format` | `text` or `json`. |
| `File` | Save logs to a file. |

## Environment Variables

| Variable | Default |
|----------|---------|
| `NO_GULOG` | `false` |
| `GULOG_FORMAT` | `text` |
| `GULOG_LOGFIE` | *(empty)* |

## Output

### Text

```text
[2026-07-22 12:30:45] INFO: Server started
```

### JSON

```json
{
    "timestamp": "2026-07-22 12:30:45",
    "level": "INFO",
    "message": "Server started"
}
```