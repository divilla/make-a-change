package WorkflowTest;
use strict;
use warnings;
use Exporter 'import';
use Cwd qw(abs_path);
use File::Basename qw(dirname);
use File::Copy qw(copy);
use File::Path qw(make_path);
use ReleaseTest;

our @EXPORT = qw(workflow_fixture workflow_cases run_workflow);
my $scripts = abs_path(dirname(__FILE__) . '/..');
my ($real_git) = map { abs_path($_) } grep { -x $_ } map { "$_/git" } split /:/, $ENV{PATH};
defined $real_git or die "git is not on PATH\n";

sub workflow_fixture {
    my $f = fixture();
    make_path("$f->{repo}/scripts/lib/mch", "$f->{repo}/agent/specs");
    for my $script (qw(git-auth.sh branch-create.sh branch-rename.sh commit-agent.pl
        commit-user.pl merge-to-dev.pl promote-to-stage.pl promote-to-prod.pl
        merge-to-stage.sh merge-to-prod.sh codex-code-spec.pl codex-review-loop.pl)) {
        copy("$scripts/$script", "$f->{repo}/scripts/$script") or die $!;
        chmod 0755, "$f->{repo}/scripts/$script" or die $!;
    }
    for my $module (qw(GitAuth.pm Release.pm Progress.pm)) {
        copy("$scripts/lib/mch/$module", "$f->{repo}/scripts/lib/mch/$module") or die $!;
    }
    for my $name ('123-auth', '124-new') {
        write_file("$f->{repo}/agent/specs/$name.md", '# Test specification');
    }
    git($f, 'add', '.');
    git($f, 'commit', '-q', '-m', 'test tooling');
    for my $branch ('dev', 'stage') { git($f, 'branch', '-f', $branch); }
    git($f, 'push', '-q', 'origin', 'master', 'dev', 'stage');
    git($f, 'checkout', '-q', '-b', 'change/123-auth');
    commit_file($f, 'work to merge');
    git($f, 'push', '-q', 'origin', 'change/123-auth');
    # Use a path supplied only to this fixture; no real Codex process is run.
    write_file("$f->{bin}/codex", <<'CODEX');
#!/bin/sh
touch "$MCH_TEST_CODEX_CALLED"
exit 99
CODEX
    chmod 0755, "$f->{bin}/codex" or die $!;
    write_file("$f->{bin}/git", <<'GIT');
#!/usr/bin/env bash
set -euo pipefail
case ${1-} in
    fetch|pull|push|ls-remote) printf '%s\n' "$*" >>"$MCH_TEST_REMOTE_CALLS" ;;
esac
exec "$MCH_TEST_REAL_GIT" "$@"
GIT
    chmod 0755, "$f->{bin}/git" or die $!;
    return $f;
}

sub workflow_cases {
    return (
        ['merge-to-dev.pl'], ['promote-to-stage.pl'], ['promote-to-prod.pl'],
        ['merge-to-stage.sh'], ['merge-to-prod.sh'],
        ['commit-agent.pl'], ['commit-user.pl'],
        ['branch-create.sh', 'agent/specs/124-new.md'],
        ['branch-rename.sh', 'change/456-renamed'],
        ['codex-code-spec.pl', 'agent/specs/123-auth.md'],
        ['codex-review-loop.pl', 'agent/specs/123-auth.md', '--base', 'dev'],
    );
}

sub run_workflow {
    my ($f, $script, @args) = @_;
    local $ENV{PATH} = "$f->{bin}:$ENV{PATH}";
    local $ENV{MCH_TEST_REAL_GIT} = $real_git;
    local $ENV{MCH_TEST_REMOTE_CALLS} = "$f->{bin}/remote-calls";
    local $ENV{MCH_GIT_SSH_KEY} = "$f->{bin}/test-key";
    local $ENV{MCH_TEST_AUTH_STATUS} = $f->{auth_status} // 0;
    local $ENV{MCH_TEST_AUTH_CALLS} = "$f->{bin}/auth-calls";
    local $ENV{MCH_TEST_CODEX_CALLED} = "$f->{bin}/codex-called";
    my $interpreter = $script =~ /\.pl\z/ ? $^X : 'bash';
    return ReleaseTest::execute($f->{cwd} // $f->{repo}, $interpreter,
        "$f->{repo}/scripts/$script", @args);
}

1;
