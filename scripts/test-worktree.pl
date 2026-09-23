#!/usr/bin/env perl
use strict;
use warnings;
use FindBin;
use lib "$FindBin::Bin/test-lib";
use Test::More;
use File::Path qw(make_path);
use ReleaseTest;
use WorkflowTest;

my $message = 'error: Uncommitted work detected. Commit or stash your changes, then retry.';

for my $case (grep { $_->[0] !~ /\Acommit-/ } workflow_cases()) {
    my ($script, @args) = @$case;
    for my $dirty ('unstaged', 'staged', 'untracked', 'deleted', 'submodule') {
        my $f = workflow_fixture();
        # Check the entire repository even when invoked from an empty subdirectory.
        make_path("$f->{repo}/inside");
        $f->{cwd} = "$f->{repo}/inside";
        if ($dirty eq 'untracked') {
            git($f, 'config', 'status.showUntrackedFiles', 'no');
            write_file("$f->{repo}/new-file", 'untracked user work');
        } elsif ($dirty eq 'deleted') {
            unlink "$f->{repo}/content" or die $!;
        } elsif ($dirty eq 'submodule') {
            git($f, '-c', 'protocol.file.allow=always', 'submodule', 'add', '-q', $f->{remote}, 'nested');
            git($f, 'commit', '-q', '-m', 'add local test submodule');
            git($f, 'config', 'submodule.nested.ignore', 'all');
            write_file("$f->{repo}/nested/content", 'uncommitted submodule work');
        } else {
            write_file("$f->{repo}/content", 'uncommitted tracked work');
            git($f, 'add', 'content') if $dirty eq 'staged';
        }
        my $before = git($f, 'show-ref');
        my $remote_before = git($f->{remote}, 'show-ref');
        my $status_before = git($f, 'status', '--porcelain', '--untracked-files=all', '--ignore-submodules=none');
        isnt($status_before, '', "$script $dirty fixture contains uncommitted work");

        for my $auth_status (0, 1) {
            $f->{auth_status} = $auth_status;
            my $label = "$script $dirty, auth " . ($auth_status ? 'missing' : 'ready');
            my ($status, $out, $error) = run_workflow($f, $script, @args);
            is($status, 1, "$label: exits one");
            like($error, qr/\Q$message\E/, "$label: explains how to resolve uncommitted work");
            is(git($f, 'show-ref'), $before, "$label: preserves local refs");
            is(git($f->{remote}, 'show-ref'), $remote_before, "$label: preserves remote refs");
            is(git($f, 'status', '--porcelain', '--untracked-files=all', '--ignore-submodules=none'),
                $status_before, "$label: preserves worktree and index");
            ok(!-e "$f->{bin}/remote-calls", "$label: no remote Git commands");
            ok(!-e "$f->{bin}/auth-calls", "$label: fails before authentication");
            ok(!-e "$f->{bin}/codex-called", "$label: does not start Codex");
        }
    }
}

# Commit helpers remain useful with staged, unstaged, and untracked work.
for my $script ('commit-user.pl', 'commit-agent.pl') {
    my $f = workflow_fixture();
    write_file("$f->{repo}/content", 'staged work');
    git($f, 'add', 'content');
    write_file("$f->{repo}/content", 'additional unstaged work');
    write_file("$f->{repo}/new-file", 'untracked work');
    my ($status, $out, $error) = run_workflow($f, $script);
    is($status, 0, "$script accepts uncommitted work when authenticated") or diag $error;
    is(git($f, 'show', 'HEAD:content'), 'additional unstaged work', "$script commits the current tracked content");
    is(git($f, 'show', 'HEAD:new-file'), 'untracked work', "$script includes new files");
    is(git($f, 'status', '--porcelain'), '', "$script leaves a clean worktree");
    is(head($f, 'change/123-auth'), git($f, 'rev-parse', 'HEAD'), "$script publishes the commit");
}

done_testing();
