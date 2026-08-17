# EdgeStream Scheduler

EdgeStream Scheduler is a lightweight daemon application that periodically executes user-defined shell commands at specified intervals. It uses a YAML-based configuration file for customization and supports log rotation, logging levels, and daemonization.

---

## Features

- **Configurable Command Scheduling:** Define multiple commands with custom intervals using a YAML configuration file.
- **Flexible Logging:** Supports log rotation with configurable size, retention period, and compression.
- **Daemonized Operation:** Runs as a background service with a specified PID file.
- **Simple CLI:** Easy to start and configure with command-line arguments.

---

## Prerequisites

- Go 1.18+ (if compiling from source)
- A Linux environment with systemd for service management.

---

## Installation

1. **Download the Binary:**  
   Place the compiled binary (`edgestream-scheduler`) in `/opt/edgestream-scheduler/bin`.

2. **Configuration File:**  
   Create the configuration file at `/etc/edgestream-scheduler/config.yml`. A sample configuration is provided below.

3. **Systemd Service File:**  
   Save the following systemd service file as `/etc/systemd/system/edgestream-scheduler.service`:

   ```ini
   [Unit]
   Description=EdgeStream Scheduler
   After=network.target

   [Service]
   Type=forking
   User=root
   WorkingDirectory=/etc/edgestream-scheduler
   LimitNOFILE=4096
   UMask=0027
   ExecStart=/opt/edgestream-scheduler/bin/edgestream-scheduler --config=/etc/edgestream-scheduler/config.yml
   SuccessExitStatus=0
   TimeoutStopSec=10
   Restart=on-failure
   RestartSec=5

   [Install]
   WantedBy=multi-user.target
    ```

4. **Enable and Start the Service:**  
   Run the following commands:

   ```bash
   sudo systemctl daemon-reload
   sudo systemctl enable edgestream-scheduler
   sudo systemctl start edgestream-scheduler
   ```

---

## Configuration File

The configuration file is written in YAML and includes settings for logging, PID file management, and scheduled commands.

Example: `/etc/edgestream-scheduler/config.yml`

```yaml
logging:
log_file: /var/log/edgestream-scheduler.log
max_size: 100     # Maximum log file size in MB before rotation
max_age:  728     # Maximum age (in days) to retain old log files
max_backups: 5    # Maximum number of old log files to retain
local_time: false # Use local time for log timestamps (default: UTC)
compress: true    # Compress old log files
log_level: debug  # Log level: debug, info, etc.
pid_file: /var/run/edgestream-scheduler.pid
commands:
- interval: 60s
  command: "echo 'Running every 60 seconds'"
- interval: 1h5m
  command: "echo 'Running every 1 hour and 5 minutes'"
- interval: 3d2h10m
  command: "echo 'Running every 3 days, 2 hours, and 10 minutes'"
  ```

---

## Command-Line Usage

```bash
edgestream-scheduler --config=<config_file>
```

### Example:
```bash
edgestream-scheduler --config=/etc/edgestream-scheduler/config.yml
```

---

## Log Settings

- **Log File:** Path to the log file where output is stored.
- **Rotation:** Configure `max_size`, `max_age`, and `max_backups` to control log rotation.
- **Compression:** Enable or disable gzip compression for old logs.
- **Log Level:** Use `debug` for detailed output, or `info` for general operational messages.

---

## Managing the Service

### Start the Service:
```bash
sudo systemctl start edgestream-scheduler
```

### Stop the Service:
```bash
sudo systemctl stop edgestream-scheduler
```

### Restart the Service:
```bash
sudo systemctl restart edgestream-scheduler
```

### Check Status:
```bash
sudo systemctl status edgestream-scheduler
```

### View Logs:
```bash
journalctl -u edgestream-scheduler -f
```

---

## Customization

### Command Scheduling

Commands are specified in the `commands` section of the YAML file. Each command has:

- **`interval`**: The interval between command executions (e.g., `60s`, `1h5m`, `3d2h10m`).
- **`command`**: The shell command to execute.

### Logging Configuration

Logging options include:

- **`log_file`**: Log file path.
- **`max_size`**: Maximum log file size (MB).
- **`max_age`**: Maximum retention period for old logs (days).
- **`max_backups`**: Maximum number of rotated log files.
- **`compress`**: Enable or disable log compression.
- **`log_level`**: Adjust log verbosity (`debug`, `info`, etc.).

---

## Troubleshooting

- **Check Configuration:** Verify the YAML file for syntax errors using `yamllint`.
- **Inspect Logs:** View detailed logs using `journalctl`.
- **Daemon Issues:** Ensure the `pid_file` path is writable and does not conflict with other services.

---

## 📄 License

EdgeStream WebUI is licensed under the **Functional Source License 1.1,
ALv2 Future License (FSL-1.1-ALv2)**.

See:

- `LICENSE` for full terms  
- `COMMERCIAL.md` for commercial licensing requirements  
