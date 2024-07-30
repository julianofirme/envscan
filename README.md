# envscan

EnvScan is a tool for scanning Git repositories for secrets and sensitive variables, helping you prevent leaks and maintain security.

## Installation

To install EnvScan, use the provided installation script:

```bash
curl -s https://raw.githubusercontent.com/julianofirme/envscan/main/install.sh | bash
```

## Usage

To scan a repository and optionally send results to a Discord webhook, follow these steps:

1. Download the configuration file:
```bash
curl -o secrets.toml https://raw.githubusercontent.com/julianofirme/envscan/main/secrets.toml
```

2. Run the scan:
  ```bash
envscan run /path/to/your/repository -c /path/to/secrets.toml
```

## Notifications via Discord
To receive scan results via Discord, follow these steps:

Create a Discord Webhook URL. Refer to Discord's documentation for instructions.

Use the webhook URL with the -d flag when running envscan:

```bash
envscan scan /path/to/your/repository -c /path/to/secret-patterns.toml -d https://discord.com/api/webhooks/your_webhook_id
```

## Flags
- -c: Specifies the path to the configuration file containing the secret patterns (e.g., -c /path/to/secret-patterns.toml).
- -d: (Optional) Specifies the Discord webhook URL for sending notifications of found secrets (e.g., -d https://discord.com/api/webhooks/your_webhook_id).

## Adding Custom Rules
You can add custom rules by editing the secret-patterns.toml file. Each rule must have a description, id, regex, secretGroup, and keywords.

## Contributing

Pull requests are welcome. For major changes, please open an issue first
to discuss what you would like to change.

Please make sure to update tests as appropriate.

## License

[MIT](https://choosealicense.com/licenses/mit/)