# dns-sync

A powerful tool that automatically syncs DNS records from active cloud instances to DNS zones. It monitors your cloud infrastructure and keeps your DNS records up-to-date with changing IP addresses of your instances.

## 🚀 Features

- **Automatic DNS Synchronization**: Monitors cloud instances and automatically updates DNS records with their current IP addresses
- **Multi-Cloud Support**: Currently supports DigitalOcean (with extensible architecture for other cloud providers)
- **Weighted Round-Robin DNS**: Distributes traffic across multiple instances using weighted DNS records
- **Health Checks**: Includes readiness probes to ensure only healthy instances are included in DNS
- **Flexible Configuration**: JSON-based configuration with support for multiple sync jobs
- **IP Type Selection**: Choose between public or private IP addresses for DNS records
- **Ignore Lists**: Exclude specific instances from DNS updates using ignore lists
- **Concurrent Processing**: Runs multiple sync jobs in parallel for better performance
- **Tag-based Filtering**: Filter instances by tags for targeted DNS updates
- **Configurable Intervals**: Set custom polling intervals for different sync jobs

## 📋 Prerequisites

Before using dns-sync, ensure you have:

### Cloud Provider Credentials
- **DigitalOcean**: API token with read access to droplets

### DNS Provider Credentials  
- **Google Cloud DNS**: Service account with DNS administrator permissions
- Google Cloud project with DNS zones configured

### System Requirements
- Go 1.16+ (for building from source)
- Network access to cloud provider APIs
- Appropriate IAM permissions for DNS zone management

## 🛠️ Installation

### Option 1: Download Pre-built Binary

Download the latest release from the [releases page](https://github.com/chandanpasunoori/dns-sync/releases):

```bash
# Linux AMD64
wget https://github.com/chandanpasunoori/dns-sync/releases/download/v0.0.3/dns-sync_0.0.3_linux_amd64.tar.gz
tar -xzf dns-sync_0.0.3_linux_amd64.tar.gz
chmod +x dns-sync
sudo mv dns-sync /usr/local/bin/

# Verify installation
dns-sync --version
```

### Option 2: Install using Homebrew (macOS/Linux)

```bash
brew tap chandanpasunoori/tap
brew install dns-sync
```

### Option 3: Build from Source

```bash
git clone https://github.com/chandanpasunoori/dns-sync.git
cd dns-sync
go mod tidy
go build -o dns-sync .
```

### Option 4: Docker

```bash
# Pull the image
docker pull ghcr.io/chandanpasunoori/dns-sync:latest

# Or build locally
docker build -t dns-sync .
```

## ⚙️ Configuration

dns-sync uses JSON configuration files to define sync jobs. Each job specifies a source (cloud provider) and destination (DNS zone).

### Environment Variables

Set the following environment variables for authentication:

```bash
# DigitalOcean API Token
export DIGITALOCEAN_ACCESS_TOKEN="your_digitalocean_token"

# Google Cloud credentials (one of the following)
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/service-account.json"
# OR set individual fields
export GOOGLE_CLOUD_PROJECT="your-project-id"
```

### Configuration File Structure

The configuration file contains an array of jobs, each defining a sync operation:

```json
{
  "jobs": [
    {
      "name": "job-name",
      "suspend": false,
      "source": { /* source configuration */ },
      "tagName": "instance-tag",
      "destination": { /* destination configuration */ }
    }
  ]
}
```

### Sample Configuration File

Here's a complete example configuration (`sample-config.json`):

```json
{
  "jobs": [
    {
      "name": "web-servers-sync",
      "suspend": false,
      "source": {
        "type": "digitalocean",
        "cloud": "digitalocean",
        "interval": "30s"
      },
      "tagName": "web-server",
      "destination": {
        "type": "gcp",
        "project": "your-gcp-project-id",
        "zone": "your-zone-id",
        "zoneName": "example.com.",
        "recordName": "api.example.com.",
        "recordType": "A",
        "ipType": "public",
        "ttl": "300s",
        "readinessProbe": {
          "period": "10s",
          "timeout": "5s",
          "successThreshold": 2,
          "failureThreshold": 3,
          "ipType": "public",
          "httpGet": {
            "path": "/health",
            "port": 8080,
            "scheme": "HTTP"
          },
          "progressDeadline": "300s"
        },
        "ignoreListFilePath": "/path/to/ignore-list.json"
      }
    }
  ]
}
```

### Configuration Parameters

#### Job Level
- **name**: Unique identifier for the job
- **suspend**: Set to `true` to disable the job
- **tagName**: DigitalOcean tag to filter instances
- **source**: Source configuration (cloud provider)
- **destination**: Destination configuration (DNS provider)

#### Source Configuration
- **type**: Cloud provider type (`"digitalocean"`)
- **cloud**: Cloud provider name (`"digitalocean"`)
- **interval**: Polling interval (e.g., `"30s"`, `"1m"`, `"5m"`)

#### Destination Configuration
- **type**: DNS provider type (`"gcp"` for Google Cloud DNS)
- **project**: Google Cloud project ID
- **zone**: DNS zone ID in Google Cloud
- **zoneName**: DNS zone name (must end with `.`)
- **recordName**: DNS record name (must end with `.`)
- **recordType**: DNS record type (`"A"` or `"AAAA"`)
- **ipType**: IP address type (`"public"` or `"private"`)
- **ttl**: DNS record TTL (e.g., `"300s"`, `"1h"`)
- **readinessProbe**: (Optional) Health check configuration
- **ignoreListFilePath**: (Optional) Path to ignore list file

#### Readiness Probe Configuration
- **period**: How often to check health
- **timeout**: Request timeout
- **successThreshold**: Consecutive successful checks needed
- **failureThreshold**: Consecutive failed checks before marking unhealthy
- **ipType**: IP type to use for health checks
- **httpGet**: HTTP health check configuration
  - **path**: Health check endpoint path
  - **port**: Port number
  - **scheme**: `"HTTP"` or `"HTTPS"`
- **progressDeadline**: Maximum time to wait for health checks

### Ignore List File

Create an ignore list file to exclude specific instances from DNS updates:

```json
{
  "nodes": [
    {
      "name": "job-name:instance-name",
      "privateIp": "10.0.0.100", 
      "publicIp": "203.0.113.100"
    }
  ]
}
```

The node name format is `{job-name}:{instance-name}`.

## 🚀 Usage

### Basic Usage

```bash
# Run with default config file (app.json)
dns-sync

# Specify custom config file
dns-sync --config /path/to/config.json

# Enable verbose logging
dns-sync --config config.json --verbose

# Check version
dns-sync --version
```

### Docker Usage

```bash
# Run with config file mounted
docker run -d \
  --name dns-sync \
  -v /path/to/config.json:/app.json \
  -v /path/to/service-account.json:/credentials.json \
  -e GOOGLE_APPLICATION_CREDENTIALS=/credentials.json \
  -e DIGITALOCEAN_ACCESS_TOKEN="your_token" \
  ghcr.io/chandanpasunoori/dns-sync:latest

# Check logs
docker logs dns-sync
```

### Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: dns-sync
spec:
  replicas: 1
  selector:
    matchLabels:
      app: dns-sync
  template:
    metadata:
      labels:
        app: dns-sync
    spec:
      containers:
      - name: dns-sync
        image: ghcr.io/chandanpasunoori/dns-sync:latest
        args: ["--config", "/config/app.json"]
        env:
        - name: DIGITALOCEAN_ACCESS_TOKEN
          valueFrom:
            secretKeyRef:
              name: dns-sync-secrets
              key: digitalocean-token
        - name: GOOGLE_APPLICATION_CREDENTIALS
          value: /credentials/service-account.json
        volumeMounts:
        - name: config
          mountPath: /config
        - name: credentials
          mountPath: /credentials
      volumes:
      - name: config
        configMap:
          name: dns-sync-config
      - name: credentials
        secret:
          secretName: gcp-credentials
```

## 🔧 Setup Guide

### Step 1: Prepare DigitalOcean

1. **Create API Token**:
   - Go to DigitalOcean Control Panel → API → Generate New Token
   - Give it read access to droplets
   - Copy the token and set it as `DIGITALOCEAN_ACCESS_TOKEN`

2. **Tag Your Instances**:
   - Tag your droplets with meaningful tags (e.g., `web-server`, `database`)
   - These tags will be used to filter instances for DNS updates

### Step 2: Setup Google Cloud DNS

1. **Create Service Account**:
   ```bash
   gcloud iam service-accounts create dns-sync-service \
     --display-name="DNS Sync Service Account"
   ```

2. **Grant DNS Admin Role**:
   ```bash
   gcloud projects add-iam-policy-binding YOUR_PROJECT_ID \
     --member="serviceAccount:dns-sync-service@YOUR_PROJECT_ID.iam.gserviceaccount.com" \
     --role="roles/dns.admin"
   ```

3. **Create and Download Key**:
   ```bash
   gcloud iam service-accounts keys create service-account.json \
     --iam-account=dns-sync-service@YOUR_PROJECT_ID.iam.gserviceaccount.com
   ```

4. **Create DNS Zone** (if not exists):
   ```bash
   gcloud dns managed-zones create example-zone \
     --dns-name="example.com." \
     --description="Example zone for dns-sync"
   ```

### Step 3: Configure dns-sync

1. **Create Configuration File**:
   ```bash
   cp sample-config.json app.json
   # Edit app.json with your specific values
   ```

2. **Set Environment Variables**:
   ```bash
   export DIGITALOCEAN_ACCESS_TOKEN="your_digitalocean_token"
   export GOOGLE_APPLICATION_CREDENTIALS="/path/to/service-account.json"
   ```

3. **Test Configuration**:
   ```bash
   dns-sync --config app.json --verbose
   ```

## 🔍 How It Works

1. **Instance Discovery**: Polls DigitalOcean API for droplets with specified tags
2. **Health Checks**: (Optional) Performs HTTP health checks on discovered instances
3. **IP Collection**: Collects public or private IP addresses from healthy instances
4. **DNS Update**: Updates Google Cloud DNS with weighted round-robin records
5. **Continuous Monitoring**: Repeats the process based on configured intervals

### Weighted Round-Robin DNS

dns-sync creates weighted DNS records where traffic is distributed across multiple instances:
- Total weight: 1000 points
- Weight per instance: 1000 ÷ number_of_healthy_instances
- Automatically adjusts as instances are added/removed

## ❗ Troubleshooting

### Common Issues

1. **Authentication Errors**:
   ```
   Error: failed to authenticate with DigitalOcean
   ```
   - Verify `DIGITALOCEAN_ACCESS_TOKEN` is set correctly
   - Check token permissions in DigitalOcean control panel

2. **Google Cloud DNS Errors**:
   ```
   Error: failed to update DNS record
   ```
   - Verify service account has DNS admin permissions
   - Check `GOOGLE_APPLICATION_CREDENTIALS` path is correct
   - Ensure DNS zone exists and zone name ends with `.`

3. **No Instances Found**:
   ```
   Info: total droplet count: 0
   ```
   - Check if droplets have the specified tag
   - Verify droplets are in running state
   - Check DigitalOcean API token permissions

4. **Health Check Failures**:
   ```
   Error: health check failed for instance
   ```
   - Verify health check endpoint is accessible
   - Check firewall rules allow health check traffic
   - Adjust health check thresholds if needed

### Debug Mode

Enable verbose logging for detailed troubleshooting:

```bash
dns-sync --config app.json --verbose
```

### Logs Analysis

Key log messages to monitor:
- `getting node ips` - Instance discovery phase
- `total droplet count: X` - Number of instances found
- `health check passed/failed` - Health check results  
- `updating record` - DNS update operations
- `weightage: X` - Load balancing calculations

## 🔐 Security Considerations

- Store API tokens and service account keys securely
- Use least-privilege access for service accounts
- Regularly rotate API tokens and keys
- Monitor dns-sync logs for unauthorized access attempts
- Use private networks where possible for internal services

## 📈 Monitoring

### Metrics to Monitor

- **Instance Discovery**: Number of instances found per job
- **Health Check Success Rate**: Percentage of successful health checks
- **DNS Update Frequency**: How often DNS records are updated
- **Error Rates**: Authentication and API call failures

### Log Monitoring

Set up log monitoring for:
- Authentication failures
- API rate limits
- DNS update errors
- Health check timeouts

## 🤝 Contributing

Contributions are welcome! Please see the [contributing guidelines](CONTRIBUTING.md) for details.

### Development Setup

1. Clone the repository:
   ```bash
   git clone https://github.com/chandanpasunoori/dns-sync.git
   cd dns-sync
   ```

2. Install dependencies:
   ```bash
   go mod tidy
   ```

3. Build and test:
   ```bash
   go build -o dns-sync .
   go test ./...
   ```

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [DigitalOcean Go Client](https://github.com/digitalocean/godo)
- [Google Cloud DNS API](https://cloud.google.com/dns/docs)
- [Cobra CLI Library](https://github.com/spf13/cobra)
- [Logrus Logging](https://github.com/sirupsen/logrus)
