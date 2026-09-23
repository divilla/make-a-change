#!/usr/bin/env perl
use strict;
use warnings;
use FindBin;
use lib "$FindBin::Bin/lib";
use mch::Release qw(run_cli promote);

run_cli(sub { promote('dev', 'stage', 'promote-to-stage.pl', @ARGV) });
