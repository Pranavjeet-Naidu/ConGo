
## Step 5: Resource Monitoring

- adds resource monitoring capabilities. 
- tracks resource usage within the container including CPU, memory, and process counts, providing insights into container performance and utilization.

### Key Features

- Real-time monitoring of container resources
- Configurable monitoring interval
- Output to specified stats files
- Selective monitoring of specific resources (CPU, memory, processes)
- Integration with existing logging infrastructure

### Architecture

- The monitoring system runs as a background goroutine that periodically collects resource statistics from cgroups and writes them to a configured stats file.
- This approach ensures that monitoring doesn't interfere with container operations.


