# EnvScan

EnvScan is a comprehensive security scanning tool designed to detect and prevent sensitive information leaks in software projects.

## Features

- Scan entire repositories for potential secrets
- Customizable secret detection rules
- Support for multiple file types
- Easy configuration
- Quick integration into CI/CD pipelines

## Installation

### Quick Install
```bash
curl -s https://raw.githubusercontent.com/julianofirme/envscan/main/install.sh | bash
```

### Manual Installation
```bash
# Download the binary
wget https://github.com/julianofirme/envscan/releases/latest/download/envscan

# Make the binary executable
chmod +x envscan

# Move to system path
sudo mv envscan /usr/local/bin/
```

## Usage

### Basic Scanning

1. Download the default configuration:
```bash
curl -o secrets.toml https://raw.githubusercontent.com/julianofirme/envscan/main/secrets.toml
```

2. Run a full repository scan:
```bash
# Scan entire repository
envscan scan /path/to/your/repository

# Scan with custom configuration
envscan scan /path/to/your/repository -c /path/to/secrets.toml
```

### Scanning Options

#### Scan Modes
- `scan`: Full repository scan
- `staged`: Scan only staged files
- `changed`: Scan modified files

#### Flags
- `-c, --config`: Specify custom configuration file
- `--fail-fast`: Stop scanning on first secret detection
- `--no-progress`: Disable progress bar
- `--follow-symlinks`: Include symbolic links in scan

### Configuration

The `secrets.toml` file allows you to define custom secret detection rules:

```toml
[[rules]]
description = "AWS Access Keys"
id = "aws-access-key"
regex = "(?i)(aws_?access_?key(_?id)?)"
secretGroup = 1
keywords = ["aws", "amazon", "cloud"]

[[rules]]
description = "Private SSH Keys"
id = "private-ssh-key"
regex = "-----BEGIN OPENSSH PRIVATE KEY-----"
secretGroup = 0
keywords = ["ssh", "key", "private"]
```

Each rule includes:
- `description`: Human-readable explanation
- `id`: Unique identifier
- `regex`: Regular expression for matching
- `secretGroup`: Capture group containing the secret
- `keywords`: Optional contextual keywords

## Best Practices

1. Use in CI/CD pipelines
2. Regularly update detection rules
3. Integrate with security scanning tools
4. Train team on avoiding hardcoded secrets

## Integrations

- GitHub Actions
- GitLab CI
- Jenkins
- CircleCI

## Troubleshooting

- Ensure you have the latest version
- Check configuration file syntax
- Verify file permissions

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Development Setup
```bash
git clone https://github.com/julianofirme/envscan.git
cd envscan
go mod download
go build
```

## License

Distributed under the MIT License. See `LICENSE` for more information.

## Contact

Project Link: [https://github.com/julianofirme/envscan](https://github.com/julianofirme/envscan)