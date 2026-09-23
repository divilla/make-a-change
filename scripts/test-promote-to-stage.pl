#!/usr/bin/env perl
use strict;
use warnings;
use FindBin;
use lib "$FindBin::Bin/test-lib";
use Test::More;
use ReleaseTest;

promotion_tests('dev', 'stage', 'promote-to-stage.pl');
unit_tests();
done_testing();
