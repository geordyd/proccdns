# DNS Proxy

A configurable DNS proxy that supports multiple DNS servers in priority order and custom domain mappings.

## Features

- Multiple DNS servers with priority order
- Custom domain mappings (e.g., `.docker` domains)
- Automatic fallback to next DNS server if one fails
- Detailed logging of DNS resolution
- Works on both macOS and Linux

## Installation

### Prerequisites

- Go 1.21 or later
- sudo privileges

### Building

```bash
go build -o proccdns
```

## Usage

### Basic Usage

```bash
sudo ./proccdns
```

This will use the default configuration:
- Listen on port 53
- Primary DNS: 8.8.8.8 (Google DNS)
- Secondary DNS: 1.1.1.1 (Cloudflare DNS)

### Advanced Usage

```bash
sudo ./proccdns -servers="8.8.8.8,1.1.1.1,9.9.9.9" -domains=".docker=172.168.1.1,.test=192.168.1.1"
```

Parameters:
- `-listen`: Address to listen on (default: ":53")
- `-servers`: Comma-separated list of DNS servers in priority order
- `-domains`: Comma-separated list of domain=ip mappings

## Platform-Specific Setup

### Linux (Fedora/Ubuntu)

1. Stop the system DNS resolver:
```bash
sudo systemctl stop systemd-resolved
sudo systemctl disable systemd-resolved
```

2. Configure the system to use localhost as DNS:
```bash
# For NetworkManager
sudo nmcli connection modify "Wired connection 1" ipv4.dns "127.0.0.1"
sudo nmcli connection up "Wired connection 1"

# Or edit resolv.conf directly
sudo bash -c 'echo "nameserver 127.0.0.1" > /etc/resolv.conf'
```

3. Run the DNS proxy:
```bash
sudo ./proccdns -servers="8.8.8.8,1.1.1.1" -domains=".docker=172.168.1.1"
```

### macOS

1. Configure DNS settings (choose one method):

Method 1 - Using networksetup (Recommended):
```bash
# Get your current network service name
networksetup -listallnetworkservices

# Set DNS server to localhost (replace "Wi-Fi" with your service name)
sudo networksetup -setdnsservers "Wi-Fi" 127.0.0.1
```

Method 2 - Using resolv.conf (Alternative):
```bash
# Check if resolv.conf is a symlink
ls -l /etc/resolv.conf

# If it's a symlink, remove it
sudo rm /etc/resolv.conf

# Create a new resolv.conf file
sudo bash -c 'echo "nameserver 127.0.0.1" > /etc/resolv.conf'
```

Note: The resolv.conf method may be overwritten by the system. Using `networksetup` is recommended for more persistent configuration.

2. Run the DNS proxy:
```bash
sudo ./proccdns -servers="8.8.8.8,1.1.1.1" -domains=".docker=172.168.1.1"
```

## Running as a Service

### Linux (systemd)

1. Create a systemd service file:
```bash
sudo bash -c 'cat > /etc/systemd/system/proccdns.service << EOL
[Unit]
Description=Custom DNS Proxy
After=network.target

[Service]
Type=simple
User=root
ExecStart=/path/to/proccdns -servers="8.8.8.8,1.1.1.1" -domains=".docker=172.168.1.1"
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOL'
```

2. Enable and start the service:
```bash
sudo systemctl daemon-reload
sudo systemctl enable proccdns
sudo systemctl start proccdns
```

### macOS (LaunchAgent)

1. Create a LaunchAgent plist:
```bash
cat > ~/Library/LaunchAgents/com.proccdns.plist << EOL
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.proccdns</string>
    <key>ProgramArguments</key>
    <array>
        <string>/path/to/proccdns</string>
        <string>-servers=8.8.8.8,1.1.1.1</string>
        <string>-domains=.docker=172.168.1.1</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardErrorPath</key>
    <string>/tmp/proccdns.err</string>
    <key>StandardOutPath</key>
    <string>/tmp/proccdns.out</string>
</dict>
</plist>
EOL
```

2. Load the LaunchAgent:
```bash
launchctl load ~/Library/LaunchAgents/com.proccdns.plist
```

## Testing

Test the DNS proxy with:
```bash
# Test a regular domain
dig google.com

# Test a mapped domain
dig test.docker
```

## Reverting Changes

### Linux

1. Stop the DNS proxy:
```bash
sudo systemctl stop proccdns
sudo systemctl disable proccdns
```

2. Restore system DNS:
```bash
sudo systemctl start systemd-resolved
sudo systemctl enable systemd-resolved
```

### macOS

1. Unload the LaunchAgent:
```bash
launchctl unload ~/Library/LaunchAgents/com.proccdns.plist
```

2. Reset DNS settings:
```bash
# If using networksetup method:
sudo networksetup -setdnsservers "Wi-Fi" empty

# If using resolv.conf method:
sudo rm /etc/resolv.conf
sudo ln -s /var/run/resolv.conf /etc/resolv.conf
```

## Logs

- Linux: Check systemd logs with `journalctl -u proccdns`
- macOS: Check logs in `/tmp/proccdns.out` and `/tmp/proccdns.err` 