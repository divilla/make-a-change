#!/usr/bin/env perl
use strict;
use warnings;
use FindBin;
use lib "$FindBin::Bin/lib";
use mch::Release qw(run_cli merge_to_dev);

run_cli(sub { merge_to_dev(@ARGV) });
