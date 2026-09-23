#!/usr/bin/env perl
use strict;
use warnings;
use FindBin;
use lib "$FindBin::Bin/test-lib";
use Test::More;
use ReleaseTest;
use WorkflowTest;

my $script = 'merge-to-dev.pl';
my $branch = 'change/123-release';
my $message = 'Implement change 123-release';

sub change_fixture {
    my $f = fixture();
    git($f, 'checkout', '-q', '-b', $branch, 'dev');
    commit_file($f, 'work one');
    commit_file($f, 'work two');
    git($f, 'push', '-q', 'origin', $branch);
    return $f;
}

my $f = change_fixture();
my $tree = git($f, 'rev-parse', 'HEAD^{tree}');
my ($status, $out, $error) = run_script($f, $script);
is($status, 0, 'merge succeeds without a PR or working gh command') or diag $error;
ok(!-e "$f->{bin}/calls", 'merge never calls gh');
my $squash = head($f, 'dev');
is(git($f, 'rev-parse', 'HEAD'), $squash, 'local dev equals published commit');
is(git($f, 'branch', '--show-current'), 'dev', 'ends on dev');
is(git($f, 'show', '-s', '--format=%s', 'HEAD'), $message, 'squash message preserved');
is(git($f, 'rev-parse', 'HEAD^{tree}'), $tree, 'squash preserves change contents');
is(git($f, 'rev-parse', 'HEAD^'), $f->{initial}, 'squash has dev as parent');
is(git($f, 'rev-list', '--count', 'HEAD'), 2, 'multiple commits become one');
is(git($f, 'ls-remote', '--heads', 'origin', "refs/heads/$branch"), '', 'remote change branch deleted');
is(git($f, 'rev-parse', $branch), $squash, 'local change branch retained at squash');
is(head($f, 'master'), $f->{initial}, 'master remains unchanged');
is(head($f, 'stage'), $f->{initial}, 'stage remains unchanged');

for my $wrapper ('merge-to-stage.sh', 'merge-to-prod.sh') {
    $f = workflow_fixture();
    my $base = head($f, 'master');
    my @work = (git($f, 'rev-parse', 'HEAD'));
    push @work, commit_file($f, "work $_") for 2 .. 4;
    # Reproduce local release branches mistakenly created from change work,
    # with neither release branch published yet.
    for my $release ('dev', 'stage') {
        git($f, 'branch', '-f', $release, $work[-1]);
        git($f->{remote}, 'update-ref', '-d', "refs/heads/$release");
    }
    commit_file($f, 'final change work');
    git($f, 'push', '-q', 'origin', 'change/123-auth');
    my $tree = git($f, 'rev-parse', 'HEAD^{tree}');
    ($status, $out, $error) = run_workflow($f, $wrapper);
    is($status, 0, "$wrapper succeeds with unpublished release branches pointing into the change") or diag $error;
    my $squash = head($f, 'dev');
    is(git($f, 'rev-parse', "$squash^"), $base, "$wrapper squashes the entire change onto master baseline");
    is(git($f, 'rev-list', '--count', "$base..$squash"), 1, "$wrapper publishes exactly one change commit");
    is(git($f, 'rev-parse', "$squash^{tree}"), $tree, "$wrapper preserves all change contents");
    is(head($f, 'stage'), $squash, "$wrapper promotes that same squash to stage");
    is(head($f, 'master'), $wrapper eq 'merge-to-prod.sh' ? $squash : $base,
        "$wrapper updates master only for production promotion");
    for my $release ('dev', 'stage') {
        is(git($f, 'rev-parse', "refs/heads/backup/$release-before-init-$work[-1]"), $work[-1],
            "$wrapper preserves the previous local $release tip in a backup branch");
    }
}

$f = fixture();
git($f, 'checkout', '-q', 'dev');
my $previous_change = commit_file($f, 'previous published change');
git($f, 'push', '-q', 'origin', 'dev');
git($f, 'checkout', '-q', '-b', $branch);
commit_file($f, 'next change part one');
commit_file($f, 'next change part two');
git($f, 'push', '-q', 'origin', $branch);
($status, $out, $error) = run_script($f, $script);
is($status, 0, 'merge onto established dev succeeds') or diag $error;
is(git($f, 'rev-parse', 'HEAD^'), $previous_change, 'previously published dev change remains the squash parent');
is(git($f, 'rev-list', '--count', "$f->{initial}..HEAD"), 2, 'separate published changes are not combined');

$f = fixture();
git($f, 'checkout', '-q', '-b', $branch, 'dev');
my $existing = commit_file($f, $message);
git($f, 'push', '-q', 'origin', $branch);
($status, $out, $error) = run_script($f, $script);
is($status, 0, 'already squashed merge succeeds') or diag $error;
is(head($f, 'dev'), $existing, 'already squashed commit identity preserved');
ok(!-e "$f->{bin}/calls", 'already squashed merge never calls gh');

for my $stale_tracking_ref (0, 1) {
    $f = change_fixture();
    git($f->{remote}, 'update-ref', '-d', 'refs/heads/dev');
    git($f, 'update-ref', '-d', 'refs/remotes/origin/dev') unless $stale_tracking_ref;
    ($status, $out, $error) = run_script($f, $script);
    is($status, 0, 'unpublished or deleted remote dev is created before merging') or diag $error;
    is(head($f, 'dev'), git($f, 'rev-parse', 'HEAD'), 'merge publishes dev after automatic creation');
    is(git($f, 'rev-parse', 'HEAD^'), $f->{initial}, 'stage supplies the initial dev base');
}

$f = change_fixture();
write_file("$f->{repo}/dirty", 'untracked');
($status, $out, $error) = run_script($f, $script);
is($status, 1, 'dirty worktree rejected');
like($error, qr/error: Uncommitted work detected\./, 'dirty error explained');
unlink "$f->{repo}/dirty" or die $!;
commit_file($f, 'unpublished');
($status, $out, $error) = run_script($f, $script);
is($status, 1, 'unpushed local commits rejected');
like($error, qr/does not match origin/, 'local remote mismatch explained');
git($f, 'push', '-q', 'origin', $branch);
git($f, 'checkout', '-q', 'dev');
my $advanced = commit_file($f, 'new dev commit');
git($f, 'push', '-q', 'origin', 'dev');
git($f, 'checkout', '-q', $branch);
($status, $out, $error) = run_script($f, $script);
is($status, 1, 'outdated change must be rebased');
like($error, qr/rebase needed/, 'ancestry error explained');
is(head($f, 'dev'), $advanced, 'dev never rolls back');

$f = fixture();
for my $args (['all'], ['one', 'two']) {
    ($status, $out, $error) = run_script($f, $script, @$args);
    is($status, 1, 'merge-to-dev takes no arguments');
    like($error, qr/usage: scripts\/merge-to-dev.pl/, 'merge usage supplied');
}
($status, $out, $error) = run_script($f, $script);
is($status, 1, 'non-change branch rejected');
like($error, qr/not a change/, 'branch restriction explained');
git($f, 'checkout', '-q', '--detach');
($status) = run_script($f, $script);
is($status, 1, 'detached HEAD rejected');
git($f, 'checkout', '-q', '-b', $branch);
($status, $out, $error) = run_script($f, $script);
is($status, 1, 'source branch must exist remotely');
like($error, qr/cannot verify/, 'missing remote source explained');

$f = change_fixture();
my $original = head($f, $branch);
reject_push($f);
($status, $out, $error) = run_script($f, $script);
is($status, 1, 'squash push rejection stops merge');
like($error, qr/test push rejected/, 'push failure preserved');
is(head($f, 'dev'), $f->{initial}, 'push rejection leaves dev unchanged');
is(head($f, $branch), $original, 'push rejection preserves remote source');
is(git($f, 'rev-parse', 'HEAD'), $original, 'push rejection preserves local source');

# A dev push may fail after the source squash was already published.
$f = change_fixture();
write_file("$f->{remote}/hooks/pre-receive", <<'HOOK');
#!/bin/sh
while read old new ref; do
    if [ "$ref" = refs/heads/dev ]; then
        echo 'dev push rejected' >&2
        exit 1
    fi
done
HOOK
chmod 0755, "$f->{remote}/hooks/pre-receive" or die $!;
($status, $out, $error) = run_script($f, $script);
is($status, 1, 'dev push failure stops before source deletion');
like($error, qr/dev push rejected/, 'dev push error retained');
is(head($f, 'dev'), $f->{initial}, 'failed dev push does not advance remote dev');
is(head($f, $branch), git($f, 'rev-parse', 'HEAD'), 'published squash remains available to resume');

# A concurrent writer advancing dev after the push must prevent source deletion.
$f = change_fixture();
write_file("$f->{remote}/hooks/post-receive", <<'HOOK');
#!/bin/sh
while read old new ref; do
    if [ "$ref" = refs/heads/dev ]; then
        raced=$(git -c user.name=Race -c user.email=race@example.invalid commit-tree "$new^{tree}" -p "$new" -m race)
        git update-ref "$ref" "$raced" "$new"
    fi
done
HOOK
chmod 0755, "$f->{remote}/hooks/post-receive" or die $!;
($status, $out, $error) = run_script($f, $script);
is($status, 1, 'remote dev is verified after publishing');
like($error, qr/origin\/dev .* does not match squashed change commit/, 'concurrent remote update explained');
isnt(git($f, 'ls-remote', '--heads', 'origin', $branch), '', 'source retained after remote verification failure');

done_testing();
