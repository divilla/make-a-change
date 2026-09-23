package mch::GitAuth;
use strict;
use warnings;
use Exporter 'import';
use File::Basename qw(dirname);
use File::Spec;

our @EXPORT_OK = qw(require_git_auth);
my $auth_script = File::Spec->rel2abs(dirname(__FILE__) . '/../../git-auth.sh');

sub require_git_auth {
    system('bash', $auth_script, '--check') == 0 or exit 1;
}

1;
