"""Fresh Go statement coverage with an independently audited source denominator."""
import contextlib
import fcntl
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import sys
import tempfile

BACKEND = Path(__file__).resolve().parents[1]
PATTERNS = ['./cmd/...', './internal/...', './pkg/...']
GO = os.environ.get('GO', 'go')


def command(args, **kwargs):
    return subprocess.run(args, cwd=BACKEND, check=True, text=True, **kwargs)


@contextlib.contextmanager
def fresh_run(name):
    base = BACKEND / '.coverage'
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


def inventory(directory):
    raw = command([GO, 'list', '-json', *PATTERNS], capture_output=True).stdout
    (directory / 'packages.json').write_text(raw)
    packages = list(json_stream(raw))
    if not packages:
        raise ValueError('no production packages')
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
        raw = command([GO, 'list', '-json', *PATTERNS], capture_output=True).stdout
        layout = {p['ImportPath']: sorted(p.get('GoFiles', []) + p.get('CgoFiles', [])) for p in json_stream(raw)}
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
        blocks[key] = (int(size), int(count) + blocks.get(key, (0, 0))[1])
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
    # API metadata can omit entire unlinked packages. Their source blocks count
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
    passed = covered * 100 >= total * 90 if integration else covered * 100 > total * 95
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
    inputs = {name: hashlib.sha256((BACKEND / name).read_bytes()).hexdigest()
              for name in paths if name and (BACKEND / name).is_file()}
    data = {'revision': command(['git', 'rev-parse', 'HEAD'], capture_output=True).stdout.strip(),
            'diff': command(['git', 'diff', 'HEAD', '--', '.'], capture_output=True).stdout,
            'inputs_sha256': inputs,
            'go': command([GO, 'version'], capture_output=True).stdout, **(extra or {})}
    (directory / 'provenance.json').write_text(json.dumps(data, indent=2))


def unit(html=False):
    with fresh_run('unit') as directory:
        metadata = inventory(directory)
        profile = directory / 'coverage.out'
        args = [GO, 'test', '-short', '-count=1', '-race', '-covermode=atomic',
                '-coverpkg=' + ','.join(PATTERNS), '-coverprofile=' + str(profile), *PATTERNS]
        provenance(directory, {'command': args})
        try:
            command(args)
            passed = report(metadata, profile, directory)
        except BaseException:
            profile.unlink(missing_ok=True)
            raise
        if html:
            command([GO, 'tool', 'cover', '-html=' + str(profile), '-o', str(directory / 'coverage.html')])
        return 0 if passed else 1


if __name__ == '__main__':
    try:
        sys.exit(unit('--html' in sys.argv))
    except (OSError, ValueError, subprocess.CalledProcessError) as error:
        print(f'coverage failed: {error}', file=sys.stderr)
        sys.exit(1)
