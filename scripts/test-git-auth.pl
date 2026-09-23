#!/usr/bin/env perl
use strict;
use warnings;
use FindBin;
use lib "$FindBin::Bin/test-lib";
use Test::More;
use File::Copy qw(copy);
use File::Path qw(make_path);
use File::Temp qw(tempdir);
use ReleaseTest;
use WorkflowTest;

my $scripts = $FindBin::Bin;
my $root = tempdir(CLEANUP => 1);
my $parent_pid = $$;
my $agent_pid;
END { kill 'TERM', $agent_pid if $$ == $parent_pid && $agent_pid; }

sub command {
    return ReleaseTest::execute($root, @_);
}

my ($status, $out, $error) = command('ssh-keygen', '-q', '-t', 'ed25519', '-N', '', '-f', "$root/key");
is($status, 0, 'create a disposable SSH key') or BAIL_OUT($error);
($status, $out, $error) = command('ssh-agent', '-a', "$root/agent.sock", '-s');
is($status, 0, 'start an isolated SSH agent') or BAIL_OUT($error);
my ($socket) = $out =~ /SSH_AUTH_SOCK=([^;]+);/;
($agent_pid) = $out =~ /SSH_AGENT_PID=(\d+);/;
defined $socket && defined $agent_pid or BAIL_OUT('cannot parse test agent environment');
local $ENV{SSH_AUTH_SOCK} = $socket;
local $ENV{SSH_AGENT_PID} = $agent_pid;
local $ENV{MCH_GIT_SSH_KEY} = "$root/key";

sub auth_check {
    return command('bash', "$scripts/git-auth.sh", '--check');
}

($status, $out, $error) = auth_check();
is($status, 1, 'empty agent is rejected');
like($error, qr/Git authentication is not ready.*Run 'bash .*git-auth\.sh'.*retry/s, 'error explains how to authenticate and retry');
{
    local $ENV{SSH_AUTH_SOCK} = "$root/missing-agent";
    ($status, $out, $error) = auth_check();
    is($status, 1, 'unavailable agent is rejected');
    ($status, $out, $error) = command('bash', "$scripts/git-auth.sh");
    is($status, 1, 'setup fails when no agent is available');
    like($error, qr/Make sure an SSH agent is running/, 'setup failure is actionable');
}

($status, $out, $error) = command('ssh-keygen', '-q', '-t', 'ed25519', '-N', '', '-f', "$root/other-key");
is($status, 0, 'create another disposable key') or BAIL_OUT($error);
($status) = command('ssh-add', "$root/other-key");
is($status, 0, 'load an unrelated identity');
($status) = auth_check();
is($status, 1, 'unrelated identity does not satisfy authentication');

($status, $out, $error) = command('bash', "$scripts/git-auth.sh");
is($status, 0, 'git-auth loads the configured key') or diag $error;
($status, $out, $error) = auth_check();
is($status, 0, 'loaded configured key passes the check');
is($out . $error, '', 'successful check is silent');
($status) = command('ssh-add', '-d', "$root/key");
is($status, 0, 'remove the configured identity');
($status) = auth_check();
is($status, 1, 'previous setup does not bypass an expired or removed identity');

for my $args (['invalid'], ['--check', 'extra']) {
    ($status, $out, $error) = command('bash', "$scripts/git-auth.sh", @$args);
    is($status, 1, 'invalid arguments rejected');
    like($error, qr/usage: scripts\/git-auth.sh/, 'usage shown');
}
{
    local $ENV{MCH_GIT_SSH_KEY} = "$root/missing-key";
    ($status) = auth_check();
    is($status, 1, 'missing public key is rejected');
    ($status, $out, $error) = command('bash', "$scripts/git-auth.sh");
    is($status, 1, 'missing private key fails setup');
    like($error, qr/Unable to load the Git SSH key/, 'setup failure explained');
}
{
    local $ENV{HOME} = "$root/home";
    local $ENV{MCH_GIT_SSH_KEY};
    delete $ENV{MCH_GIT_SSH_KEY};
    my $key_dir = "$ENV{HOME}/.ssh/divilla-github";
    make_path($key_dir);
    for my $suffix ('', '.pub') {
        copy("$root/key$suffix", "$key_dir/id_ed25519$suffix") or die $!;
    }
    chmod 0600, "$key_dir/id_ed25519" or die $!;
    ($status, $out, $error) = command('bash', "$scripts/git-auth.sh");
    is($status, 0, 'default key location is relative to the user home') or diag $error;
    ($status) = auth_check();
    is($status, 0, 'default key passes authentication after setup');
}


for my $case (workflow_cases()) {
    my ($script, @args) = @$case;
    my $f = workflow_fixture();
    $f->{auth_status} = 1;
    if ($script =~ /\Acommit-/) {
        write_file("$f->{repo}/content", 'uncommitted user work');
    }
    my $before = git($f, 'show-ref');
    my $remote_before = git($f->{remote}, 'show-ref');
    my $status_before = git($f, 'status', '--porcelain');
    ($status, $out, $error) = run_workflow($f, $script, @args);
    is($status, 1, "$script refuses unauthenticated execution");
    like($error, qr/Git authentication is not ready.*git-auth\.sh.*retry/s, "$script explains how to proceed");
    is(git($f, 'show-ref'), $before, "$script leaves local refs unchanged");
    is(git($f->{remote}, 'show-ref'), $remote_before, "$script leaves remote refs unchanged");
    is(git($f, 'status', '--porcelain'), $status_before, "$script leaves worktree and index unchanged");
    ok(!-e "$f->{bin}/remote-calls", "$script fails before any remote Git command");
    ok(!-e "$f->{bin}/codex-called", "$script fails before starting Codex");
}

# Authenticated commit and rename paths complement the existing release/Codex suites.
for my $script ('commit-agent.pl', 'commit-user.pl', 'branch-rename.sh') {
    my $f = workflow_fixture();
    my @args;
    if ($script eq 'branch-rename.sh') {
        @args = ('change/456-renamed');
    } else {
        write_file("$f->{repo}/content", 'new content to commit');
    }
    ($status, $out, $error) = run_workflow($f, $script, @args);
    is($status, 0, "$script succeeds with authentication ready") or diag $error;
    my $branch = @args ? $args[0] : 'change/123-auth';
    is(head($f, $branch), git($f, 'rev-parse', 'HEAD'), "$script publishes the expected branch");
}

done_testing();
