#!/usr/bin/env perl
use strict;
use warnings;
use FindBin;
use lib "$FindBin::Bin/test-lib";
use Test::More;
use ReleaseTest;
use WorkflowTest;

sub initialize {
    my ($f) = @_;
    return ReleaseTest::execute($f->{repo}, $^X, '-I', "$FindBin::Bin/lib",
        '-e', 'use mch::Release qw(run_cli ensure_release_branches); run_cli(sub { ensure_release_branches() })');
}

for my $mask (0 .. 15) {
    my $f = fixture();
    git($f, 'checkout', '-q', '-b', 'work/in-progress');
    my $original = commit_file($f, 'must not be promoted by initialization');
    my (%expected_local, %expected_remote, %backups);
    for my $index (0, 1) {
        my $branch = (qw(dev stage))[$index];
        my $local_exists = $mask & (1 << ($index * 2));
        my $remote_exists = $mask & (2 << ($index * 2));
        my $local_commit = git($f, 'commit-tree', "$f->{initial}^{tree}", '-p', $f->{initial}, '-m', "local $branch");
        my $remote_commit = git($f, 'commit-tree', "$f->{initial}^{tree}", '-p', $f->{initial}, '-m', "remote $branch");
        git($f, 'push', '-q', 'origin', "$remote_commit:refs/heads/$branch");
        if ($local_exists) {
            git($f, 'update-ref', "refs/heads/$branch", $local_commit);
        } else {
            git($f, 'update-ref', '-d', "refs/heads/$branch");
        }
        git($f->{remote}, 'update-ref', '-d', "refs/heads/$branch") unless $remote_exists;
        # Retain stale tracking refs to verify that remote existence is checked.
        $expected_local{$branch} = $remote_exists ? ($local_exists ? $local_commit : $remote_commit) : $f->{initial};
        $expected_remote{$branch} = $remote_exists ? $remote_commit : $f->{initial};
        $backups{$branch} = $local_commit if $local_exists && !$remote_exists;
    }
    if (!($mask & 2)) {
        $expected_local{dev} = $expected_remote{dev} = $expected_remote{stage};
    }
    my ($status, $out, $error) = initialize($f);
    is($status, 0, "presence combination $mask initializes successfully") or diag $error;
    for my $branch ('dev', 'stage') {
        is(git($f, 'rev-parse', "refs/heads/$branch"), $expected_local{$branch}, "$mask: local $branch has the correct starting commit");
        is(head($f, $branch), $expected_remote{$branch}, "$mask: remote $branch has the correct starting commit");
        is(git($f, 'rev-parse', "refs/remotes/origin/$branch"), $expected_remote{$branch}, "$mask: fetched $branch reflects origin");
        if (exists $backups{$branch}) {
            is(git($f, 'rev-parse', "refs/heads/backup/$branch-before-init-$backups{$branch}"), $backups{$branch},
                "$mask: displaced local $branch work remains in a backup branch");
        }
    }
    is(git($f, 'branch', '--show-current'), 'work/in-progress', "$mask: initialization does not switch branches");
    is(git($f, 'rev-parse', 'HEAD'), $original, "$mask: current commit remains unchanged");
    is(head($f, 'master'), $f->{initial}, "$mask: production is unchanged");
    my $refs = git($f, 'show-ref');
    ($status, $out, $error) = initialize($f);
    is($status, 0, "$mask: repeated initialization succeeds") or diag $error;
    is(git($f, 'show-ref'), $refs, "$mask: repeated initialization does not move branches");
}

for my $checkout ('stage', 'dev') {
    my $f = fixture();
    git($f, 'checkout', '-q', $checkout);
    my $old_tip = commit_file($f, 'unpublished release work');
    for my $branch ('stage', 'dev') {
        git($f->{remote}, 'update-ref', '-d', "refs/heads/$branch");
    }
    my ($status, $out, $error) = initialize($f);
    is($status, 0, "initialization succeeds while on local-only $checkout") or diag $error;
    is(git($f, 'branch', '--show-current'), $checkout, "$checkout remains checked out");
    is(git($f, 'status', '--porcelain'), '', "$checkout index and worktree match its initialized tip");
    is(git($f, 'rev-parse', "backup/$checkout-before-init-$old_tip"), $old_tip,
        "$checkout unpublished work remains accessible in backup");
    for my $branch ('stage', 'dev') {
        is(git($f, 'rev-parse', $branch), $f->{initial}, "$branch local tip starts at production");
        is(head($f, $branch), $f->{initial}, "$branch remote tip starts at production");
    }
}

for my $checkout ('work/in-progress', 'master', 'detached', 'missing-master') {
    my $f = fixture();
    my $latest = git($f, 'commit-tree', "$f->{initial}^{tree}", '-p', $f->{initial}, '-m', 'new production commit');
    # Advance only the server: local master and origin/master are both stale.
    git($f->{remote}, 'fetch', '-q', $f->{repo}, $latest);
    git($f->{remote}, 'update-ref', 'refs/heads/master', $latest);
    if ($checkout eq 'detached') {
        git($f, 'checkout', '-q', '--detach');
    } elsif ($checkout ne 'master') {
        git($f, 'checkout', '-q', '-b', 'work/in-progress');
    }
    git($f, 'branch', '-D', 'master') if $checkout eq 'missing-master';
    my $original = git($f, 'rev-parse', 'HEAD');
    my $original_branch = git($f, 'branch', '--show-current');
    for my $branch ('stage', 'dev') {
        git($f, 'update-ref', '-d', "refs/heads/$branch");
        git($f->{remote}, 'update-ref', '-d', "refs/heads/$branch");
    }
    my ($status, $out, $error) = initialize($f);
    is($status, 0, "$checkout: fresh initialization succeeds") or diag $error;
    for my $branch ('master', 'stage', 'dev') {
        is(git($f, 'rev-parse', $branch), $latest, "$checkout: local $branch starts at latest production commit");
        is(head($f, $branch), $latest, "$checkout: remote $branch shares the production baseline");
    }
    is(git($f, 'branch', '--show-current'), $original_branch, "$checkout: original checkout is restored");
    is(git($f, 'rev-parse', 'HEAD'), $checkout eq 'master' ? $latest : $original,
        "$checkout: only master checkout advances during initialization");
    is(git($f, 'status', '--porcelain'), '', "$checkout: worktree remains clean");
}

for my $failure ('diverged', 'unpublished') {
    my $f = fixture();
    my $local_master = commit_file($f, 'unpublished master commit');
    if ($failure eq 'diverged') {
        my $remote_master = git($f, 'commit-tree', "$f->{initial}^{tree}", '-p', $f->{initial}, '-m', 'remote production commit');
        git($f->{remote}, 'fetch', '-q', $f->{repo}, $remote_master);
        git($f->{remote}, 'update-ref', 'refs/heads/master', $remote_master);
    }
    git($f, 'checkout', '-q', '-b', 'work/in-progress');
    for my $branch ('stage', 'dev') {
        git($f, 'update-ref', '-d', "refs/heads/$branch");
        git($f->{remote}, 'update-ref', '-d', "refs/heads/$branch");
    }
    my ($status, $out, $error) = initialize($f);
    is($status, 1, "$failure: master must be synchronized before initialization");
    like($error, $failure eq 'diverged' ? qr/fast-forward/ : qr/Local master contains unpublished commits/,
        "$failure: master synchronization failure is explained");
    is(git($f, 'branch', '--show-current'), 'work/in-progress', "$failure: original branch is restored after failure");
    is(git($f, 'rev-parse', 'master'), $local_master, "$failure: local master work is preserved");
    for my $branch ('stage', 'dev') {
        is(git($f, 'branch', '--list', $branch), '', "$failure: local $branch is not created from an invalid baseline");
        is(git($f, 'ls-remote', '--heads', 'origin', "refs/heads/$branch"), '', "$failure: remote $branch is not created from an invalid baseline");
    }
}

my $f = fixture();
git($f, 'config', '--replace-all', 'remote.origin.fetch', '+refs/heads/master:refs/remotes/origin/master');
for my $branch ('dev', 'stage') {
    git($f, 'update-ref', '-d', "refs/heads/$branch");
    git($f, 'update-ref', '-d', "refs/remotes/origin/$branch");
}
my ($status, $out, $error) = initialize($f);
is($status, 0, 'single-branch fetch configuration does not prevent initialization') or diag $error;
for my $branch ('dev', 'stage') {
    is(git($f, 'rev-parse', $branch), head($f, $branch), "$branch is fetched explicitly and created locally");
    is(git($f, 'config', "branch.$branch.remote"), 'origin', "$branch is configured for origin");
    is(git($f, 'config', "branch.$branch.merge"), "refs/heads/$branch", "$branch has the correct upstream branch");
}

$f = fixture();
git($f->{remote}, 'update-ref', '-d', 'refs/heads/stage');
reject_push($f);
($status, $out, $error) = initialize($f);
is($status, 1, 'a real push rejection still fails');
like($error, qr/test push rejected/, 'creation failure preserves the server error');
is(git($f, 'ls-remote', '--heads', 'origin', 'refs/heads/stage'), '', 'failed creation does not claim success');
unlink "$f->{remote}/hooks/pre-receive" or die $!;
($status, $out, $error) = initialize($f);
is($status, 0, 'initialization can be retried after a rejected push') or diag $error;

$f = fixture();
for my $branch ('dev', 'stage') {
    git($f, 'update-ref', '-d', "refs/heads/$branch");
    git($f->{remote}, 'update-ref', '-d', "refs/heads/$branch");
}
git($f->{remote}, 'update-ref', '-d', 'refs/heads/master');
($status, $out, $error) = initialize($f);
is($status, 1, 'a missing production baseline is not replaced with an arbitrary commit');
like($error, qr/without origin\/master/, 'missing baseline is explained');

for my $case (
    ['merge-to-dev.pl'], ['promote-to-stage.pl'], ['promote-to-prod.pl'],
    ['merge-to-stage.sh'], ['merge-to-prod.sh'],
    ['branch-create.sh', 'agent/specs/124-new.md'],
) {
    my ($script, @args) = @$case;
    for my $blocked ('dirty', 'unauthenticated') {
        $f = workflow_fixture();
        for my $branch ('dev', 'stage') {
            git($f, 'update-ref', '-d', "refs/heads/$branch");
            git($f->{remote}, 'update-ref', '-d', "refs/heads/$branch");
        }
        if ($blocked eq 'dirty') {
            write_file("$f->{repo}/content", 'uncommitted user work');
        } else {
            $f->{auth_status} = 1;
        }
        ($status, $out, $error) = run_workflow($f, $script, @args);
        is($status, 1, "$script rejects $blocked execution before initialization");
        ok(!-e "$f->{bin}/remote-calls", "$script $blocked: no remote operations");
        for my $branch ('dev', 'stage') {
            is(git($f, 'branch', '--list', $branch), '', "$script $blocked: local $branch stays absent");
            is(git($f, 'ls-remote', '--heads', 'origin', "refs/heads/$branch"), '', "$script $blocked: remote $branch stays absent");
        }
    }
}

for my $wrapper ('merge-to-stage.sh', 'merge-to-prod.sh') {
    $f = workflow_fixture();
    my $master = head($f, 'master');
    for my $branch ('dev', 'stage') {
        git($f, 'update-ref', '-d', "refs/heads/$branch");
        git($f->{remote}, 'update-ref', '-d', "refs/heads/$branch");
    }
    ($status, $out, $error) = run_workflow($f, $wrapper);
    is($status, 0, "$wrapper initializes both release branches and completes") or diag $error;
    is(head($f, 'stage'), head($f, 'dev'), "$wrapper promotes the newly merged commit to stage");
    is(head($f, 'master'), $wrapper eq 'merge-to-prod.sh' ? head($f, 'stage') : $master,
        "$wrapper updates production only when requested");
}

done_testing();
