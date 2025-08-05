
## Configuration Functions

### `parseConfig`

Parses command-line arguments to extract monitoring configuration options:

- `--enable-monitor`: Enables resource monitoring
- `--monitor-interval`: Sets the monitoring interval in seconds
- `--monitor-stats-file`: Specifies the file path for storing monitoring statistics
- `--monitor-cpu`: Enables CPU usage monitoring
- `--monitor-memory`: Enables memory usage monitoring
- `--monitor-processes`: Enables process count monitoring

## Monitoring Setup and Operation

### `startResourceMonitoring`

```go
func startResourceMonitoring(config *Config) error
```

Initializes and starts the resource monitoring system:
- Creates a stats file for writing monitoring data
- Starts a background goroutine that runs at the specified interval
- Configures monitoring based on the specified options

### `collectResourceStats`

```go
func collectResourceStats(config *Config) (string, error)
```

Collects resource statistics from the container:
- Reads CPU statistics from cgroups (compatible with both cgroup v1 and v2)
- Gathers memory usage information
- Tracks the number of running processes
- Formats the collected data as a string for logging

## Integration Functions

### `setupLogging`

```go
func setupLogging(config *Config) error
```

Sets up logging for the container, which is utilized by the monitoring system:
- Creates log files for stdout and stderr
- Redirects container output to these files
- Provides necessary infrastructure for monitoring stats

### `setupContainer`

```go
func setupContainer(config *Config) error
```

Main container setup function that now includes monitoring initialization:
- Calls `startResourceMonitoring` when monitoring is enabled in the configuration
