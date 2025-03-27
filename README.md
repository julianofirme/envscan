# envscan

EnvScan is a tool for scanning projects for secrets and sensitive variables, helping you prevent leaks and maintain security.

## Installation

To install Envscan, use the provided installation script:

```bash
curl -s https://raw.githubusercontent.com/julianofirme/envscan/main/install.sh | bash
```

## Usage

To scan a repository, follow these steps:

1. Download the configuration file:
```bash
curl -o secrets.toml https://raw.githubusercontent.com/julianofirme/envscan/main/secrets.toml
```

2. Run the scan:
  ```bash
envscan run /path/to/your/repository -c /path/to/secrets.toml
```

## Flags
- -c: Specifies the path to the configuration file containing the secret patterns (e.g., -c /path/to/secret-patterns.toml).

## Adding Custom Rules
You can add custom rules by editing the secrets.toml file. Each rule must have a description, id, regex, secretGroup, and keywords.

## Contributing

Pull requests are welcome. For major changes, please open an issue first
to discuss what you would like to change.

Please make sure to update tests as appropriate.

## License

[MIT](https://choosealicense.com/licenses/mit/)