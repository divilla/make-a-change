"""Make actions with fail-closed module discovery, including packages without tests."""
import os
import sys
import subprocess

from coverage import GO, command, discover


def action(name):
    production, _ = discover()
    packages = [p['ImportPath'] for p in production]
    lint = os.environ.get('GOLANGCI_LINT', 'golangci-lint')
    commands = {
        'format': [lint, 'fmt', '--no-config', '--enable', 'gofumpt', '--enable', 'goimports', './...'],
        'format-check': [lint, 'fmt', '--no-config', '--enable', 'gofumpt', '--enable', 'goimports', '--diff', './...'],
        'lint': [lint, 'run', '--no-config', '--default', 'standard', '--enable', 'revive', '--timeout', '5m', './...'],
        'vet': [GO, 'vet', './...'],
        'test': [GO, 'test', '-short', '-count=1', *packages],
        'race': [GO, 'test', '-short', '-count=1', '-race', *packages],
        'benchmark': [GO, 'test', '-short', '-run=^$', '-bench=.', '-benchmem', *packages],
        'deps-audit': [os.environ.get('GOVULNCHECK', 'govulncheck'), './...'],
    }
    command(commands[name])


if __name__ == '__main__':
    try:
        action(sys.argv[1])
    except subprocess.CalledProcessError as error:
        sys.exit(error.returncode if error.returncode > 0 else 1)
    except (OSError, ValueError) as error:
        print(error, file=sys.stderr)
        sys.exit(1)
