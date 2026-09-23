package ReleaseTest;
use strict;
use warnings;
use Exporter 'import';
use Cwd qw(abs_path getcwd);
use File::Basename qw(dirname);
use File::Path qw(make_path);
use File::Copy qw(copy);
use File::Temp qw(tempdir);
use Test::More;

our @EXPORT = qw(fixture git run_script head commit_file write_file reject_push promotion_tests unit_tests);
my $scripts = abs_path(dirname(__FILE__) . '/..');
my $root = tempdir(CLEANUP => 1);
my $counter = 0;

# Keep all Git state and subprocess output outside the user's checkout.
$ENV{GIT_CONFIG_NOSYSTEM} = 1;
$ENV{GIT_CONFIG_GLOBAL} = '/dev/null';
$ENV{GIT_TERMINAL_PROMPT} = 0;
$ENV{LC_ALL} = 'C';
delete @ENV{qw(GIT_DIR GIT_WORK_TREE GIT_INDEX_FILE GIT_COMMON_DIR)};

sub write_file {
    my ($path, $contents) = @_;
    open my $fh, '>', $path or die "$path: $!";
    print {$fh} $contents;
    close $fh or die "$path: $!";
}

sub execute {
    my ($cwd, @command) = @_;
    my $id = ++$counter;
    my $pid = fork();
    defined $pid or die "fork: $!";
    if (!$pid) {
        chdir $cwd or die "chdir: $!";
        open STDOUT, '>', "$root/$id.out" or die $!;
        open STDERR, '>', "$root/$id.err" or die $!;
        exec @command;
        die "exec @command: $!";
    }
    waitpid($pid, 0);
    my $status = $?;
    my @output;
    for my $suffix ('out', 'err') {
        open my $fh, '<', "$root/$id.$suffix" or die $!;
        local $/;
        push @output, <$fh> // '';
    }
    return ($status >> 8 || ($status & 127 ? 1 : 0), @output);
}

sub git {
    my ($f, @args) = @_;
    my $cwd = ref $f ? $f->{repo} : $f;
    my ($status, $output, $error) = execute($cwd, 'git', @args);
    die "git @args: $error" if $status;
    $output =~ s/\s+\z//;
    return $output;
}

sub fixture {
    my $dir = "$root/fixture " . ++$counter;
    make_path("$dir/repo", "$dir/bin");
    my $f = { repo => "$dir/repo", remote => "$dir/origin.git", bin => "$dir/bin" };
    git($dir, 'init', '--bare', '-q', $f->{remote});
    git($f, 'init', '-q', '-b', 'master');
    git($f, 'config', 'user.name', 'Release Test');
    git($f, 'config', 'user.email', 'release@example.invalid');
    git($f, 'config', 'commit.gpgsign', 'false');
    git($f, 'remote', 'add', 'origin', $f->{remote});
    $f->{initial} = commit_file($f, 'initial');
    git($f, 'branch', 'dev');
    git($f, 'branch', 'stage');
    git($f, 'push', '-q', 'origin', 'master', 'stage', 'dev');
    # Fail and record any accidental GitHub dependency in the Git-only workflow.
    write_file("$f->{bin}/gh", <<'GH');
#!/usr/bin/env perl
use strict;
use warnings;
open my $out, '>', "$ENV{RELEASE_TEST_BIN}/calls" or die $!;
print {$out} "gh @ARGV\n";
close $out;
die "gh must not be called\n";
GH
    chmod 0755, "$f->{bin}/gh" or die $!;
    copy("$scripts/test-lib/ssh-add", "$f->{bin}/ssh-add") or die $!;
    chmod 0755, "$f->{bin}/ssh-add" or die $!;
    return $f;
}

sub head {
    my ($f, $branch) = @_;
    return git($f->{remote}, 'rev-parse', "refs/heads/$branch");
}

sub commit_file {
    my ($f, $message) = @_;
    write_file("$f->{repo}/content", "$message\n");
    git($f, 'add', 'content');
    git($f, 'commit', '-q', '-m', $message);
    return git($f, 'rev-parse', 'HEAD');
}

sub run_script {
    my ($f, $script, @args) = @_;
    local $ENV{PATH} = "$f->{bin}:$ENV{PATH}";
    local $ENV{RELEASE_TEST_BIN} = $f->{bin};
    local $ENV{MCH_GIT_SSH_KEY} = "$f->{bin}/test-key";
    return execute($f->{repo}, $^X, "$scripts/$script", @args);
}

sub reject_push {
    my ($f) = @_;
    write_file("$f->{remote}/hooks/pre-receive", "#!/bin/sh\necho 'test push rejected' >&2\nexit 1\n");
    chmod 0755, "$f->{remote}/hooks/pre-receive" or die $!;
}

sub promotion_tests {
    my ($source, $destination, $script) = @_;
    my $f = fixture();
    git($f, 'checkout', '-q', $source);
    my @commits = map { commit_file($f, "pending $_") } 1 .. 4;
    git($f, 'push', '-q', 'origin', $source);
    # Neither a stale local source branch nor a deleted tracking ref may choose the target.
    git($f, 'reset', '--hard', $f->{initial});
    git($f, 'update-ref', '-d', "refs/remotes/origin/$source");
    my ($status, $out, $error) = run_script($f, $script);
    is($status, 0, 'default promotion succeeds') or diag $error;
    is(head($f, $destination), $commits[0], 'default promotes exactly the oldest missing commit');
    is(head($f, $source), $commits[3], 'source remains unchanged');
    is(git($f, 'branch', '--show-current'), $destination, 'successful promotion checks out destination');

    ($status, $out, $error) = run_script($f, $script, substr($commits[2], 0, 12));
    is($status, 0, 'abbreviated SHA promotion succeeds') or diag $error;
    is(head($f, $destination), $commits[2], 'SHA mode includes intervening commits but not later ones');
    my $before = git($f, 'rev-parse', 'HEAD');
    ($status) = run_script($f, $script, $commits[0]);
    is($status, 0, 'SHA already in destination history is a no-op');
    is(head($f, $destination), $before, 'already-promoted SHA never rolls destination back');
    ($status, $out, $error) = run_script($f, $script, 'all');
    is($status, 0, 'all promotion succeeds') or diag $error;
    is(head($f, $destination), $commits[3], 'all promotes to source tip');
    for my $args ([], ['all'], [$commits[3]]) {
        ($status, $out) = run_script($f, $script, @$args);
        is($status, 0, 'nothing pending exits zero');
        like($out, qr/Nothing to promote/, 'no-op is reported');
    }

    for my $invalid ('0' x 40, 'not-a-sha', '--help', '', $source) {
        ($status, $out, $error) = run_script($f, $script, $invalid);
        is($status, 1, 'invalid SHA fails even when no commits are pending');
        is($error =~ /\Q$invalid does not exist in $source\E\n\z/ ? 1 : 0, 1, 'invalid SHA reports source name');
    }
    ($status, $out, $error) = run_script($f, $script, 'all', 'extra');
    is($status, 1, 'extra arguments rejected');
    like($error, qr/usage:/, 'usage supplied');
    is(head($f, $destination), $commits[3], 'invalid invocations leave remote unchanged');

    my $outside = commit_file($f, 'destination only');
    git($f, 'push', '-q', 'origin', $destination);
    ($status) = run_script($f, $script);
    is($status, 0, 'destination ahead of source has nothing to promote');
    ($status, $out, $error) = run_script($f, $script, $outside);
    is($status, 1, 'SHA in destination but absent from source fails validation first');
    like($error, qr/\Q$outside does not exist in $source\E/, 'existing commit outside source rejected');
    git($f, 'checkout', '-q', $source);
    git($f, 'reset', '--hard', $commits[3]);
    my $divergent = commit_file($f, 'source only');
    git($f, 'push', '-q', 'origin', $source);
    for my $args ([], ['all'], [$divergent]) {
        ($status, $out, $error) = run_script($f, $script, @$args);
        is($status, 1, 'divergent promotion rejected');
        like($error, qr/cannot fast-forward/, 'divergence explained');
    }
    is(head($f, $destination), $outside, 'divergence never forces destination');

    $f = fixture();
    git($f, 'checkout', '-q', $source);
    my $pending = commit_file($f, 'pending');
    git($f, 'push', '-q', 'origin', $source);
    write_file("$f->{repo}/dirty", 'untracked');
    ($status, $out, $error) = run_script($f, $script);
    is($status, 1, 'dirty worktree blocks mutation');
    like($error, qr/error: Uncommitted work detected\./, 'dirty worktree explained');
    is(head($f, $destination), $f->{initial}, 'dirty failure leaves destination unchanged');
    unlink "$f->{repo}/dirty" or die $!;
    reject_push($f);
    ($status, $out, $error) = run_script($f, $script, 'all');
    is($status, 1, 'rejected push exits one');
    like($error, qr/test push rejected/, 'underlying push error preserved');
    is(head($f, $destination), $f->{initial}, 'rejected push leaves remote unchanged');
    unlink "$f->{remote}/hooks/pre-receive" or die $!;
    ($status, $out, $error) = run_script($f, $script, $pending);
    is($status, 0, 'promotion can resume after rejected push') or diag $error;
    is(head($f, $destination), $pending, 'resumed promotion publishes exact SHA');

    $f = fixture();
    git($f, 'checkout', '-q', $source);
    $pending = commit_file($f, 'remote pending');
    git($f, 'push', '-q', 'origin', $source);
    git($f, 'branch', '-f', $destination, $pending);
    git($f, 'checkout', '-q', $destination);
    my $local = commit_file($f, 'unpublished destination');
    ($status, $out, $error) = run_script($f, $script, $pending);
    is($status, 1, 'local destination ahead of target is rejected');
    like($error, qr/does not match promotion commit/, 'local extra commits explained');
    is(head($f, $destination), $f->{initial}, 'local extra commits never published');
    is(git($f, 'rev-parse', 'HEAD'), $local, 'local work preserved');

    # A missing local destination is created from its fetched remote branch.
    git($f, 'checkout', '-q', $source);
    git($f, 'branch', '-D', $destination);
    ($status, $out, $error) = run_script($f, $script);
    is($status, 0, 'destination need not exist locally') or diag $error;
    is(head($f, $destination), $pending, 'remote-tracking destination is advanced correctly');

    # Missing remote dev/stage start from their release baseline, even if the
    # local branch points into unpublished change work.
    for my $missing ($source, $destination) {
        $f = fixture();
        git($f, 'checkout', '-q', $source);
        commit_file($f, 'pending');
        git($f, 'push', '-q', 'origin', $source);
        git($f->{remote}, 'update-ref', '-d', "refs/heads/$missing");
        ($status, $out, $error) = run_script($f, $script, 'all');
        if ($missing eq 'master') {
            is($status, 1, 'missing remote master still fails');
            like($error, qr/origin\/master is unavailable/, 'production branch remains explicitly required');
        } else {
            is($status, 0, 'missing remote release branch is created automatically') or diag $error;
            is(head($f, $destination), head($f, $source), 'promotion completes after branch creation');
        }
    }

    $f = fixture();
    git($f, 'remote', 'set-url', 'origin', "$f->{repo}/missing-remote");
    ($status, $out, $error) = run_script($f, $script);
    is($status, 1, 'failed fetch stops promotion');
    like($error, qr/git fetch.*failed/, 'fetch failure explained');

    # Nonlinear source history follows the first-parent release path.
    $f = fixture();
    git($f, 'checkout', '-q', $source);
    my $first = commit_file($f, 'first release commit');
    git($f, 'checkout', '-q', '-b', 'side', $f->{initial});
    write_file("$f->{repo}/side", 'side change');
    git($f, 'add', 'side');
    git($f, 'commit', '-q', '-m', 'side commit');
    git($f, 'checkout', '-q', $source);
    git($f, 'merge', '--no-ff', '-m', 'merge side', 'side');
    my $merge = git($f, 'rev-parse', 'HEAD');
    git($f, 'push', '-q', 'origin', $source);
    ($status, $out, $error) = run_script($f, $script);
    is($status, 0, 'default handles source merge history') or diag $error;
    is(head($f, $destination), $first, 'default selects release-line commit, not side commit');
    ($status, $out, $error) = run_script($f, $script);
    is($status, 0, 'next default promotes merge commit') or diag $error;
    is(head($f, $destination), $merge, 'merge commit includes side history without rewriting');

    $f = fixture();
    git($f, 'checkout', '-q', $source);
    commit_file($f, 'pending');
    git($f, 'push', '-q', 'origin', $source);
    write_file("$f->{remote}/hooks/post-receive", <<'HOOK');
#!/bin/sh
while read old new ref; do
    raced=$(git -c user.name=Race -c user.email=race@example.invalid commit-tree "$new^{tree}" -p "$new" -m race)
    git update-ref "$ref" "$raced" "$new"
done
HOOK
    chmod 0755, "$f->{remote}/hooks/post-receive" or die $!;
    ($status, $out, $error) = run_script($f, $script);
    is($status, 1, 'published destination SHA is checked after pushing');
    like($error, qr/origin\/$destination .* does not match promotion commit/, 'concurrent destination update explained');
}

sub unit_tests {
    require lib;
    lib->import("$scripts/lib");
    require mch::Release;
    my ($status, $output) = mch::Release::capture_status($^X, '-e', 'print "result\n"; exit 3');
    is($status, 3 << 8, 'capture preserves failed command status');
    is($output, 'result', 'capture preserves output without trailing whitespace');
    is(mch::Release::capture($^X, '-e', 'exit 0'), '', 'empty output supported');
    eval { mch::Release::capture($^X, '-e', 'exit 4') };
    like($@, qr/failed/, 'capture rejects command failure');
    eval { mch::Release::run($^X, '-e', 'exit 4') };
    like($@, qr/failed/, 'run rejects command failure');
    {
        no warnings 'redefine';
        local *mch::Release::capture_status = sub { return (128 << 8, '') };
        eval { mch::Release::ancestor('a', 'b') };
        like($@, qr/cannot check ancestry/, 'ancestry command failure is not treated as divergence');
    }
    {
        no warnings 'redefine';
        local *mch::Release::capture = sub { return '' };
        eval { mch::Release::remote_head('dev') };
        like($@, qr/cannot verify origin\/dev/, 'missing remote head is rejected');
    }
}

1;
