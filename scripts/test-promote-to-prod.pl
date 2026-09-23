#!/usr/bin/env perl
use strict;
use warnings;
use FindBin;
use lib "$FindBin::Bin/test-lib";
use Test::More;
use ReleaseTest;

promotion_tests('stage', 'master', 'promote-to-prod.pl');
done_testing();
