"""Fresh Go statement coverage with an independently audited source denominator."""
import contextlib
import fcntl
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import signal
import subprocess
import sys
import tempfile

CLI = Path(__file__).resolve().parents[1]
PATTERNS = ['./...']
GO = os.environ.get('GO', 'go')
SUPERVISOR = Path(__file__).with_name('owned_command.py')


def command(args, **kwargs):
    return subprocess.run(args, cwd=CLI, check=True, text=True, **kwargs)


@contextlib.contextmanager
def campaign_signals():
    def interrupt(signum, frame):
        # Let cleanup and both journals finish even if cancellation is repeated.
        for sig in (signal.SIGINT, signal.SIGTERM):
            signal.signal(sig, signal.SIG_IGN)
        raise KeyboardInterrupt(signal.Signals(signum).name)

    previous = {sig: signal.signal(sig, interrupt)
                for sig in (signal.SIGINT, signal.SIGTERM)}
    try:
        yield
    finally:
        for sig, handler in previous.items():
            signal.signal(sig, handler)


@contextlib.contextmanager
def fresh_run(name):
    base = CLI / '.coverage'
    base.mkdir(exist_ok=True)
    with (base / (name + '.lock')).open('w') as lock:
        fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
        directory = base / name
        if directory.exists():
            shutil.rmtree(directory)
        directory.mkdir(mode=0o700)
        yield directory.resolve()


def json_stream(text):
    decoder = json.JSONDecoder()
    while text.strip():
        value, end = decoder.raw_decode(text.lstrip())
        yield value
        text = text.lstrip()[end:]


def discover():
    raw = command([GO, 'list', '-json', *PATTERNS], capture_output=True).stdout
    all_packages = list(json_stream(raw))
    packages = []
    for package in all_packages:
        relative = Path(package['Dir']).relative_to(CLI)
        # These are the only explicit non-production trees. Unexpected source
        # elsewhere is included, never silently dropped from the denominator.
        if relative.parts and relative.parts[0] in ('integration', 'scripts'):
            continue
        if package.get('GoFiles') or package.get('CgoFiles'):
            packages.append(package)
    if not packages:
        raise ValueError('no production packages')
    return packages, raw


def inventory(directory):
    packages, raw = discover()
    (directory / 'packages.json').write_text(raw)
    # Generate structural block metadata with the same Go toolchain, without
    # executing tests or importing their counters. This includes unlinked code.
    expected = {}
    sources = {}
    names = []
    with tempfile.TemporaryDirectory(dir=directory) as scratch:
        for package in packages:
            name = package['ImportPath']
            names.append(name)
            for filename in package.get('GoFiles', []) + package.get('CgoFiles', []):
                source = Path(package['Dir']) / filename
                sources[str(source)] = hashlib.sha256(source.read_bytes()).hexdigest()
                output = Path(scratch) / 'instrumented.go'
                command([GO, 'tool', 'cover', '-mode=atomic', '-var=FoundationCover',
                         '-o', str(output), str(source)], capture_output=True)
                text = output.read_text()
                pos = re.search(r'Pos: \[3 \* \d+\]uint32\{(.*?)\n\s*\}', text, re.S)
                nums = re.search(r'NumStmt: \[\d+\]uint16\{(.*?)\n\s*\}', text, re.S)
                if not pos or not nums:
                    raise ValueError('cannot parse Go structural metadata: ' + str(source))
                positions = re.findall(r'(\d+),\s*(\d+),\s*(0x[0-9a-f]+),', pos[1])
                counts = re.findall(r'(\d+),', nums[1])
                if len(positions) != len(counts):
                    raise ValueError('invalid structural metadata')
                for (start, end, columns), count in zip(positions, counts):
                    col = int(columns, 16)
                    key = f'{name}/{filename}:{start}.{col & 65535},{end}.{col >> 16}'
                    expected[key] = int(count)
    metadata = {'packages': names, 'blocks': expected, 'sources': sources,
                'layout': {p['ImportPath']: sorted(p.get('GoFiles', []) + p.get('CgoFiles', [])) for p in packages}}
    (directory / 'denominator.json').write_text(json.dumps(metadata, indent=2))
    return metadata


def unchanged(metadata):
    if 'layout' in metadata:
        packages, _ = discover()
        layout = {p['ImportPath']: sorted(p.get('GoFiles', []) + p.get('CgoFiles', [])) for p in packages}
        if layout != metadata['layout']:
            raise ValueError('production package/file set changed during measurement')
    for name, digest in metadata['sources'].items():
        if hashlib.sha256(Path(name).read_bytes()).hexdigest() != digest:
            raise ValueError('source changed during measurement: ' + name)


def parse_profile(path):
    lines = path.read_text().splitlines()
    if not lines or lines[0] != 'mode: atomic':
        raise ValueError('missing atomic profile header')
    blocks = {}
    for line in lines[1:]:
        match = re.fullmatch(r'(\S+:\d+\.\d+,\d+\.\d+) (\d+) (\d+)', line)
        if not match:
            raise ValueError('malformed coverage block: ' + line)
        key, size, count = match.groups()
        if key in blocks and blocks[key][0] != int(size):
            raise ValueError('conflicting coverage block')
        blocks[key] = (int(size), max(int(count), blocks.get(key, (0, 0))[1]))
    return blocks


def report(metadata, profile, directory, integration=False, *, enforce=True):
    unchanged(metadata)
    blocks = parse_profile(profile)
    if not blocks:
        raise ValueError('empty coverage metadata')
    expected = metadata['blocks']
    for key, (size, _) in blocks.items():
        if key not in expected or size != expected[key]:
            raise ValueError('profile disagrees with structural denominator: ' + key)
    missing = set(expected) - blocks.keys()
    if missing and not integration:
        raise ValueError('incomplete unit profile; missing blocks: ' + ', '.join(sorted(missing)[:5]))
    # Terminal metadata can omit entire unlinked packages. Their source blocks count
    # as zero. A partially represented package is corruption, not unlinked code.
    represented = {key.rsplit('/', 1)[0] for key in blocks}
    if integration and any(key.rsplit('/', 1)[0] in represented for key in missing):
        raise ValueError('partial integration package metadata')
    if integration:
        executable = {key.rsplit('/', 1)[0] for key in expected}
        if set(metadata.get('linked', [])) & executable - represented:
            raise ValueError('missing linked package metadata')
    totals = {name: [0, 0] for name in metadata['packages']}
    for key, size in expected.items():
        totals[key.rsplit('/', 1)[0]][1] += size
        if blocks.get(key, (0, 0))[1] > 0:
            totals[key.rsplit('/', 1)[0]][0] += size
    covered = sum(value[0] for value in totals.values())
    total = sum(value[1] for value in totals.values())
    if not total:
        raise ValueError('empty statement denominator')
    threshold = 70 if integration else 80
    passed = covered * 100 >= total * threshold
    lines = ['package covered/total percent']
    for name, (hit, size) in sorted(totals.items()):
        lines.append(f'{name} {hit}/{size} ' + (f'{100*hit/size:.4f}%' if size else 'no executable statements'))
    outcome = f'gate {"PASS" if passed else "FAIL"}' if enforce else 'diagnostic (no coverage gate)'
    lines.append(f'TOTAL {covered}/{total} {100*covered/total:.4f}% — {outcome}')
    missing_packages = sorted(set(metadata['packages']) - represented)
    lines.append('Absent from runtime profile (structural zero): ' + ', '.join(missing_packages))
    result = '\n'.join(lines) + '\n'
    (directory / 'report.txt').write_text(result)
    (directory / 'result.json').write_text(json.dumps(dict(covered=covered, total=total,
        passed=passed if enforce else None, threshold_enforced=enforce, packages=totals), indent=2))
    print(result, end='')
    return passed if enforce else True


def provenance(directory, extra=None):
    paths = command(['git', 'ls-files', '--cached', '--others', '--exclude-standard',
                     '-z', '--', '.'], capture_output=True).stdout.split('\0')
    inputs = {name: hashlib.sha256((CLI / name).read_bytes()).hexdigest()
              for name in paths if name and (CLI / name).is_file()}
    data = {'revision': command(['git', 'rev-parse', 'HEAD'], capture_output=True).stdout.strip(),
            'diff': command(['git', 'diff', 'HEAD', '--', '.'], capture_output=True).stdout,
            'inputs_sha256': inputs,
            'go': command([GO, 'version'], capture_output=True).stdout,
            'build_environment': command([GO, 'env', 'GOOS', 'GOARCH', 'GOVERSION', 'GOFLAGS', 'CGO_ENABLED'], capture_output=True).stdout,
            **(extra or {})}
    (directory / 'provenance.json').write_text(json.dumps(data, indent=2))
    return data


def recorded(directory, label, args, **kwargs):
    """Keep original exits and output even when callers continue with diagnostics."""
    log = directory / (label + '.log')
    entry = {'label': label, 'args': args, 'exit': None}
    process = None
    try:
        # Let the child write directly to disk, including partial lines. A pipe
        # buffered by the parent loses diagnostics when waiting is interrupted.
        with log.open('wb') as output:
            process = subprocess.Popen([sys.executable, '-B', str(SUPERVISOR), *args],
                                       cwd=CLI, stdout=output, start_new_session=True,
                                       stderr=subprocess.STDOUT, **kwargs)
            try:
                process.wait()
            except BaseException:
                # A detached supervisor owns/reaps the whole tree, including
                # Go test executables and the PTY's separate session. Keep the
                # campaign lock until it finishes, even after repeated signals.
                previous = {sig: signal.signal(sig, signal.SIG_IGN)
                            for sig in (signal.SIGINT, signal.SIGTERM)}
                try:
                    process.terminate()
                    process.wait()
                finally:
                    for sig, handler in previous.items():
                        signal.signal(sig, handler)
                raise
    except BaseException as error:
        entry['error'] = type(error).__name__ + ': ' + str(error)
        raise
    finally:
        if process is not None:
            entry['exit'] = process.returncode
        with (directory / 'commands.jsonl').open('a') as stream:
            stream.write(json.dumps(entry) + '\n')
    output = log.read_text()
    print(output, end='', flush=True)
    if process.returncode:
        raise subprocess.CalledProcessError(process.returncode, args)
    return output


def validate_events(text, required):
    events = [json.loads(line) for line in text.splitlines() if line.startswith('{')]
    if any(event.get('Action') in ('skip', 'fail') for event in events):
        raise ValueError('failed or skipped required execution')
    passed = {event.get('Test') for event in events if event.get('Action') == 'pass'}
    if not required or not set(required) <= passed:
        raise ValueError('missing required scenario results')
    roots = {event['Test'].split('/')[0] for event in events if event.get('Test')}
    if roots != set(required):
        raise ValueError('unselected scenario executed')


def scenarios():
    manifest = json.loads((CLI / 'scripts/terminal-scenarios.json').read_text())
    if set(manifest) != {'program', 'pty'}:
        raise ValueError('both program and PTY scenarios are required')
    for group, suite in manifest.items():
        names = suite['tests']
        package = './integration' if group == 'program' else './integration/terminal'
        if suite['package'] != package or not names or len(set(names)) != len(names):
            raise ValueError('invalid or empty scenario selection')
        # Restrict to known program-boundary test families. Architecture, Flow,
        # adapters and harness unit tests cannot accidentally enter the campaign.
        prefix = ('TestCLIProgram', 'TestCLIStartup') if group == 'program' else ('TestShell',)
        if any(name in ('TestCLIProgramDefReviewUsesDefinitionPromptAndSharedArtifactSession',
                         'TestCLIProgramArtifactChatResumesSharedArtifactSession') or not name.startswith(prefix) or not re.fullmatch(r'Test\w+', name) for name in names):
            raise ValueError('non-program scenario selection')
        available = command([GO, 'test', '-list=^Test', package], capture_output=True).stdout.splitlines()
        if not set(names) <= set(available):
            raise ValueError('unmatched scenario selection')
    return manifest


def union_profiles(paths, destination):
    """Only explicit campaign-local inputs; there is deliberately no profile glob."""
    merged = {}
    if not paths:
        raise ValueError('no campaign profiles')
    for path in paths:
        if path.parent != destination.parent:
            raise ValueError('profile from another campaign')
        blocks = parse_profile(path)
        if not blocks:
            raise ValueError('empty campaign profile')
        for key, (size, count) in blocks.items():
            if key in merged and merged[key][0] != size:
                raise ValueError('mismatched statement inventories')
            merged[key] = (size, max(count, merged.get(key, (0, 0))[1]))
    destination.write_text('mode: atomic\n' + ''.join(
        f'{key} {size} {count}\n' for key, (size, count) in sorted(merged.items())))


def counter_profile(directory, counters, label):
    if not list(counters.glob('covmeta.*')) or not list(counters.glob('covcounters.*')):
        raise ValueError('missing child metadata/counters: ' + str(counters))
    profile = directory / (label + '.out')
    recorded(directory, label + '-convert', [GO, 'tool', 'covdata', 'textfmt',
             '-i=' + str(counters), '-o=' + str(profile)])
    return profile


def diagnostics(metadata, profile, directory, html=False):
    blocks = parse_profile(profile)
    (directory / 'uncovered.txt').write_text(''.join(
        f'{key} {size} statements\n' for key, size in sorted(metadata['blocks'].items())
        if size and not blocks.get(key, (0, 0))[1]))
    # Expand unlinked structural zeros in the display profile too.
    display = directory / 'complete.out'
    display.write_text('mode: atomic\n' + ''.join(
        f'{key} {size} {int(blocks.get(key, (0, 0))[1] > 0)}\n'
        for key, size in sorted(metadata['blocks'].items())))
    recorded(directory, 'functions', [GO, 'tool', 'cover', '-func=' + str(display)])
    if html:
        recorded(directory, 'html', [GO, 'tool', 'cover', '-html=' + str(display),
                                    '-o', str(directory / 'coverage.html')])


def integration(directory, metadata):
    manifest = scenarios()
    (directory / 'scenarios.json').write_text(json.dumps(manifest, indent=2))
    if not shutil.which('socat'):
        raise ValueError('socat is required; PTY campaign incomplete')
    recorded(directory, 'socat-version', ['socat', '-V'])
    binary = directory / 'mch'
    coverpkg = ','.join(metadata['packages'])
    recorded(directory, 'build', [GO, 'build', '-cover', '-covermode=atomic',
             '-coverpkg=' + coverpkg, '-o', str(binary), './cmd/mch'])
    (directory / 'binary.sha256').write_text(hashlib.sha256(binary.read_bytes()).hexdigest() + '\n')
    failures = []
    profiles = []
    for group, suite in manifest.items():
        counters = directory / (group + '-counters')
        counters.mkdir(mode=0o700)
        env = dict(os.environ, MCH_COVER_BINARY=str(binary), MCH_COVER_DIR=str(counters))
        args = [GO, 'test', '-count=1', '-timeout=3m', '-json',
                '-run=^(' + '|'.join(suite['tests']) + ')$']
        if group == 'program':
            profile = directory / 'program.out'
            args += ['-covermode=atomic', '-coverpkg=' + coverpkg, '-coverprofile=' + str(profile)]
            profiles.append(profile)
        args.append(suite['package'])
        try:
            output = recorded(directory, group, args, env=env)
            validate_events(output, suite['tests'])
            profiles.append(counter_profile(directory, counters, group + '-child'))
        except (subprocess.CalledProcessError, ValueError) as error:
            failures.append(error)
    if failures:
        raise failures[0]
    # The program test binary cannot link cmd/mch; the actual executable must.
    # go list -deps establishes every package the executable links independently.
    linked = command([GO, 'list', '-deps', './cmd/mch'], capture_output=True).stdout.splitlines()
    metadata['linked'] = sorted(set(linked) & set(metadata['packages']))
    (directory / 'denominator.json').write_text(json.dumps(metadata, indent=2))
    profile = directory / 'coverage.out'
    union_profiles(profiles, profile)
    return profile


def campaign(terminal=False, html=False):
    with campaign_signals(), fresh_run('integration' if terminal else 'unit') as directory:
        print('Campaign artifacts: ' + str(directory), flush=True)
        status = {'complete': False, 'exit': None}
        try:
            inputs = provenance(directory)
            for label, args in [('lint-version', [os.environ.get('GOLANGCI_LINT', 'golangci-lint'), 'version']),
                                ('audit-version', [os.environ.get('GOVULNCHECK', 'govulncheck'), '-version'])]:
                recorded(directory, label, args)
            metadata = inventory(directory)
            if terminal:
                profile = integration(directory, metadata)
            else:
                profile = directory / 'coverage.out'
                args = [GO, 'test', '-short', '-count=1', '-race', '-timeout=3m', '-json',
                        '-covermode=atomic', '-coverpkg=' + ','.join(metadata['packages']),
                        '-coverprofile=' + str(profile), *metadata['packages']]
                output = recorded(directory, 'unit', args)
                if any(e.get('Action') == 'fail' or (e.get('Action') == 'skip' and e.get('Test')) for e in json_stream(output)):
                    raise ValueError('failed or skipped unit execution')
            if inputs:
                for name, digest in inputs['inputs_sha256'].items():
                    if hashlib.sha256((CLI / name).read_bytes()).hexdigest() != digest:
                        raise ValueError('campaign input changed: ' + name)
            diagnostics(metadata, profile, directory, html)
            passed = report(metadata, profile, directory, terminal)
            status.update(complete=True, exit=0 if passed else 1)
            return status['exit']
        except BaseException as error:
            status.update(error=str(error), exit=getattr(error, 'returncode', 1))
            # Raw output is diagnostic only; never leave a success result after
            # failed tests, interrupted commands, or failed postprocessing.
            for name in ('result.json', 'report.txt', 'coverage.html', 'complete.out'):
                (directory / name).unlink(missing_ok=True)
            raise
        finally:
            (directory / 'status.json').write_text(json.dumps(status, indent=2))


if __name__ == '__main__':
    try:
        sys.exit(campaign('--integration' in sys.argv, '--html' in sys.argv))
    except subprocess.CalledProcessError as error:
        print(f'coverage incomplete: {error}', file=sys.stderr)
        sys.exit(error.returncode if error.returncode > 0 else 1)
    except (OSError, ValueError) as error:
        print(f'coverage incomplete: {error}', file=sys.stderr)
        sys.exit(1)
