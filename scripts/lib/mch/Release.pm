package mch::Release;
use strict;
use warnings;
use Exporter 'import';
use mch::GitAuth qw(require_git_auth);

our @EXPORT_OK = qw(run_cli merge_to_dev promote ensure_release_branches);

sub run_cli {
    my ($action) = @_;
    eval { $action->(); 1 } or do {
        my $error = $@ || "unknown failure\n";
        print STDERR $error;
        exit 1;
    };
}

sub fail { die "$_[0]\n" }

sub capture_status {
    my (@command) = @_;
    open my $fh, '-|', @command
        or fail(join(' ', @command) . " failed to start: $!");
    local $/;
    my $output = <$fh> // '';
    close $fh;
    my $status = $?;
    $output =~ s/\s+\z//;
    return ($status, $output);
}

sub capture {
    my (@command) = @_;
    my ($status, $output) = capture_status(@command);
    $status == 0 or fail(join(' ', @command) . ' failed');
    return $output;
}

sub run {
    my (@command) = @_;
    system @command;
    $? == 0 or fail(join(' ', @command) . ' failed');
}

sub ancestor {
    my ($older, $newer) = @_;
    my ($status) = capture_status('git', 'merge-base', '--is-ancestor', $older, $newer);
    return 1 if $status == 0;
    return 0 if $status == 256;
    fail("cannot check ancestry of $older and $newer");
}

sub ensure_clean {
    capture(qw(git status --porcelain=v1 --untracked-files=all --ignore-submodules=none)) eq ''
        or fail('error: Uncommitted work detected. Commit or stash your changes, then retry.');
}

sub remote_head {
    my ($branch) = @_;
    my $output = capture('git', 'ls-remote', '--heads', 'origin', "refs/heads/$branch");
    $output =~ /\A([0-9a-f]+)\s+refs\/heads\/\Q$branch\E\z/
        or fail("cannot verify origin/$branch");
    return $1;
}

sub fetched_head {
    my ($branch) = @_;
    my ($status, $commit) = capture_status('git', 'rev-parse', '--verify', '--quiet',
        "refs/remotes/origin/${branch}^{commit}");
    $status == 0 or fail("error: origin/$branch is unavailable after fetching. "
        . "Ensure $branch exists on origin and remote.origin.fetch includes it. "
        . "To publish an existing local branch, run: git push -u origin $branch");
    return $commit;
}

sub fetch_branch {
    my ($branch, $remote_commit) = @_;
    my ($status, $commit) = capture_status('git', 'rev-parse', '--verify', '--quiet',
        "refs/remotes/origin/${branch}^{commit}");
    if ($status != 0 || $commit ne $remote_commit) {
        # Explicit refspecs also work with a single-branch clone's fetch config.
        run('git', 'fetch', 'origin', "+refs/heads/$branch:refs/remotes/origin/$branch");
    }
    return fetched_head($branch);
}

sub pull_master {
    my ($remote_commit) = @_;
    defined $remote_commit
        or fail('error: Cannot initialize release branches without origin/master.');
    my $baseline = fetch_branch('master', $remote_commit);
    my ($status) = capture_status(qw(git rev-parse --verify --quiet refs/heads/master^{commit}));
    run(qw(git branch --no-track master), $baseline) if $status != 0;
    my $original_branch = capture(qw(git branch --show-current));
    my $original_head = capture(qw(git rev-parse HEAD));
    my $ok = eval {
        run(qw(git checkout --quiet master));
        run(qw(git pull --ff-only origin master));
        $baseline = fetched_head('master');
        capture(qw(git rev-parse HEAD)) eq $baseline
            or fail('error: Local master contains unpublished commits. Publish or resolve them before initializing release branches.');
        1;
    };
    my $error = $@;
    if ($original_branch ne 'master') {
        if ($original_branch eq '') {
            run(qw(git checkout --quiet --detach), $original_head);
        } else {
            run(qw(git checkout --quiet), $original_branch);
        }
    }
    die $error unless $ok;
    return $baseline;
}

sub ensure_release_branches {
    # Callers check the worktree and authentication before any branch creation.
    my $heads = capture(qw(git ls-remote --heads origin refs/heads/dev refs/heads/stage refs/heads/master));
    my %remote;
    for my $line (split /\n/, $heads) {
        my ($commit, $ref) = split /\s+/, $line;
        $remote{$ref} = $commit;
    }
    my %local;
    for my $branch ('stage', 'dev') {
        my ($status, $commit) = capture_status('git', 'rev-parse', '--verify', '--quiet',
            "refs/heads/${branch}^{commit}");
        $local{$branch} = $commit if $status == 0;
    }
    my $needs_setup = grep { !exists $local{$_} || !exists $remote{"refs/heads/$_"} } qw(stage dev);
    my $parent = $needs_setup ? pull_master($remote{'refs/heads/master'}) : undef;
    for my $branch ('stage', 'dev') {
        my $ref = "refs/heads/$branch";
        my $start;
        if (exists $remote{$ref}) {
            $start = fetch_branch($branch, $remote{$ref});
        } else {
            # The initial release chain is master -> stage -> dev.
            # A local-only release branch may point into unsquashed change
            # work, so it must not become the published release baseline.
            $start = $parent;
        }
        if (!exists $remote{$ref} && exists $local{$branch} && $local{$branch} ne $start) {
            my $backup = "backup/$branch-before-init-$local{$branch}";
            run('git', 'update-ref', "refs/heads/$backup", $local{$branch});
            print "Preserved local $branch as $backup before initializing from the release baseline\n";
            if (capture(qw(git branch --show-current)) eq $branch) {
                # Keep the index and worktree consistent when initializing the
                # checked-out branch; its previous contents remain in backup.
                run('git', 'checkout', '--quiet', '--detach', $local{$branch});
                run('git', 'branch', '-f', $branch, $start);
                run('git', 'checkout', '--quiet', $branch);
            } else {
                run('git', 'branch', '-f', $branch, $start);
            }
        }
        if (!exists $local{$branch}) {
            run('git', 'branch', '--no-track', $branch, $start);
        }
        if (!exists $remote{$ref}) {
            # An empty expected value permits creation only, never overwriting
            # a branch concurrently created by another writer.
            run('git', 'push', "--force-with-lease=$ref:", 'origin', "$start:$ref");
            fetch_branch($branch, $start);
        }
        if (!exists $local{$branch} || !exists $remote{$ref}) {
            run('git', 'config', "branch.$branch.remote", 'origin');
            run('git', 'config', "branch.$branch.merge", $ref);
        }
        $parent = $start;
    }
}

sub merge_to_dev {
    @_ == 0 or fail('usage: scripts/merge-to-dev.pl');
    my $branch = capture(qw(git branch --show-current));
    my ($name) = $branch =~ m{^change/([0-9]+-[0-9A-Za-z_-]+)$}
        or fail("current branch is not a change/<change-slug> branch: $branch");
    ensure_clean();
    require_git_auth();
    run(qw(git fetch --prune origin));
    my $original = capture(qw(git rev-parse HEAD));
    my $remote = remote_head($branch);
    $original eq $remote
        or fail("local $branch $original does not match origin/$branch $remote");
    ensure_release_branches();
    my $base = fetched_head('dev');
    ancestor($base, $original) or fail('rebase needed: origin/dev is not an ancestor of HEAD');

    my $message = "Implement change $name";
    my $parents = capture('git', 'rev-list', '--parents', '-n', '1', $original);
    my $subject = capture('git', 'log', '-1', '--format=%s', $original);
    my $squashed = $original;
    if ($parents ne "$original $base" || $subject ne $message) {
        my $tree = capture(qw(git rev-parse HEAD^{tree}));
        $squashed = capture('git', 'commit-tree', $tree, '-p', $base, '-m', $message);
        run('git', 'push', "--force-with-lease=refs/heads/$branch:$remote", 'origin',
            "$squashed:refs/heads/$branch");
        run('git', 'update-ref', "refs/heads/$branch", $squashed, $original);
    }

    run(qw(git checkout dev));
    run(qw(git pull --ff-only origin dev));
    run('git', 'merge', '--ff-only', $branch);
    my $head = capture(qw(git rev-parse HEAD));
    $head eq $squashed or fail("dev HEAD $head does not match squashed change commit $squashed");
    run(qw(git push -u origin dev));
    my $published = remote_head('dev');
    $published eq $squashed
        or fail("origin/dev $published does not match squashed change commit $squashed");
    run('git', 'push', "--force-with-lease=refs/heads/$branch:$squashed", 'origin', '--delete', $branch);
    print "Merged $squashed into dev\n";
}

sub promote {
    my ($source, $destination, $script, @args) = @_;
    @args <= 1 or fail("usage: scripts/$script [all|<commit-sha>]");
    ensure_clean();
    require_git_auth();
    run(qw(git fetch --prune origin));
    ensure_release_branches();
    my $source_tip = fetched_head($source);
    my $destination_tip = fetched_head($destination);
    my $target;
    if (@args && $args[0] ne 'all') {
        my $sha = $args[0];
        $sha =~ /\A[0-9a-fA-F]{4,64}\z/ or fail("$sha does not exist in $source");
        my ($status, $commit) = capture_status('git', 'rev-parse', '--verify', '--quiet',
            '--end-of-options', "${sha}^{commit}");
        $status == 0 && ancestor($commit, $source_tip)
            or fail("$sha does not exist in $source");
        $target = $commit;
    } else {
        $target = $source_tip;
    }

    if (ancestor($target, $destination_tip)) {
        print "Nothing to promote from $source to $destination\n";
        return;
    }
    ancestor($destination_tip, $target)
        or fail("cannot fast-forward $destination to $target: histories have diverged");
    if (!@args) {
        # Follow the release line; a merge brings its side-branch commits with it.
        my $pending = capture('git', 'rev-list', '--first-parent', '--ancestry-path', '--reverse',
            "$destination_tip..$source_tip");
        ($target) = split /\n/, $pending;
        defined $target or fail("cannot find next commit from $source to $destination");
    }

    ensure_clean();
    run('git', 'checkout', $destination);
    run('git', 'merge', '--ff-only', $target);
    my $head = capture(qw(git rev-parse HEAD));
    $head eq $target or fail("$destination HEAD $head does not match promotion commit $target");
    run('git', 'push', '-u', 'origin', "$target:refs/heads/$destination");
    my $published = remote_head($destination);
    $published eq $target
        or fail("origin/$destination $published does not match promotion commit $target");
    print "Promoted $target from $source to $destination\n";
}

1;
