# Congo: Resource Monitoring Usage Guide

This guide explains how to use the resource monitoring features in the Congo container implementation.

## Monitoring Command-Line Options

The following options are available for configuring resource monitoring:

| Option | Description |
|--------|-------------|
| `--enable-monitor` | Enables resource monitoring |
| `--monitor-interval <seconds>` | Sets the interval between monitoring checks (default: 30 seconds) |
| `--monitor-stats-file <path>` | Specifies the file where monitoring stats will be written |
| `--monitor-cpu` | Enables CPU usage monitoring |
| `--monitor-memory` | Enables memory usage monitoring |
| `--monitor-processes` | Enables process count monitoring |

## Basic Usage

To run a container with resource monitoring enabled:

```bash
sudo ./main [standard_args] --enable-monitor -- /bin/sh
```

This will enable monitoring with default settings (30-second interval, monitoring CPU, memory, and processes).

## Advanced Usage Examples

### Custom Monitoring Interval

Set a custom monitoring interval of 10 seconds:

```bash
sudo ./main [standard_args] --enable-monitor --monitor-interval 10 -- /bin/sh
```

### Custom Stats File Location

Specify a custom location for the stats file:

```bash
sudo ./main [standard_args] --enable-monitor --monitor-stats-file /tmp/container-stats.log -- /bin/sh
```

### Selective Resource Monitoring

Monitor only CPU and memory usage:

```bash
sudo ./main [standard_args] --enable-monitor --monitor-cpu --monitor-memory -- /bin/sh
```

### Combined with Logging

Use monitoring alongside logging features:

```bash
sudo ./main [standard_args] --log-dir /var/log/congo --enable-monitor --monitor-interval 5 -- /bin/sh
```

## Interpreting Monitoring Output

The monitoring data is written to the specified stats file in the following format:

```
[2023-07-28T12:34:56Z] CPU: usage_usec 123456 user_usec 98765 system_usec 24691 | Memory: 45.67 MB | Processes: 12
```

Each line contains:
- Timestamp in ISO 8601 format
- CPU usage metrics (if enabled)
- Memory usage in MB (if enabled)
- Process count (if enabled)

## Troubleshooting

If monitoring isn't working as expected:

1. Ensure the container has the necessary permissions to read from cgroups
2. Check that the specified stats file location is writable
3. Verify that cgroups are properly set up for the container
4. Check the container logs for any monitoring-related errors
