#!/usr/bin/env perl
use strict;
use warnings;

sub foreground {
    my ($text, $color) = @_;
    my ($red, $green, $blue) = map { hex($_) } $color =~ /^#(..)(..)(..)$/;
    return sprintf "\e[38;2;%d;%d;%dm%s\e[0m", $red, $green, $blue, $text;
}

sub swatch {
    my ($color) = @_;
    my ($red, $green, $blue) = map { hex($_) } $color =~ /^#(..)(..)(..)$/;
    return sprintf "\e[48;2;%d;%d;%dm      \e[0m", $red, $green, $blue;
}

sub paint_color {
    my ($color) = @_;
    return foreground($color, $color) . '   ' . swatch($color);
}

sub paint_reference {
    my ($text, $theme, $gradient) = @_;
    return paint_color($text) if $text =~ /^#/;

    my ($name) = $text =~ /\.([^.]+)$/;
    if ($name eq 'GradientColors' && @$gradient) {
        return foreground($text, $gradient->[0]) . '   '
            . join '   ', map { swatch($_) } @$gradient;
    }
    return exists $theme->{$name}
        ? foreground($text, $theme->{$name}) . '   ' . swatch($theme->{$name})
        : $text;
}

my @lines = <DATA>;
my $source = join '', @lines;
my %theme = $source =~ /^\s*(\w+):\s*'(#[0-9A-Fa-f]{6})'/mg;
my ($gradient_values) = $source =~ /^\s*GradientColors:\s*\[([^]]+)\]/m;
my @gradient = defined $gradient_values
    ? $gradient_values =~ /#[0-9A-Fa-f]{6}/g
    : ();

$source =~ s{^([ \t]*primary: )this\.colors\.Background(,)$}
    {$1 . swatch($theme{Background}) . $2}gme;
for my $entry (
    ['message', 'MessageBackground'],
    ['input', 'InputBackground'],
    ['focus', 'FocusBackground'],
) {
    my ($field, $color_name) = @$entry;
    $source =~ s{^([ \t]*\Q$field\E:)\n[ \t]*this\.colors\.\Q$color_name\E \?\?\n[ \t]*'#[0-9A-Fa-f]{6}',}
        {$1 . ' ' . swatch($theme{$color_name}) . ','}gme;
}
$source =~ s{^([ \t]*(?:added|removed): )this\.colors\.(DiffAdded|DiffRemoved)(,)$}
    {$1 . swatch($theme{$2}) . $3}gme;

for my $line (split /(?<=\n)/, $source) {
    $line =~ s/(this\.colors\.\w+|#[0-9A-Fa-f]{6})(?![0-9A-Fa-f])/paint_reference($1, \%theme, \@gradient)/ge;
    print $line;
}

__DATA__
export const darkTheme: ColorsTheme = {
  type: 'dark',
  Background: '#000000',
  Foreground: '#FFFFFF',
  LightBlue: '#AFD7D7',
  AccentBlue: '#87AFFF',
  AccentPurple: '#D7AFFF',
  AccentCyan: '#87D7D7',
  AccentGreen: '#D7FFD7',
  AccentYellow: '#FFFFAF',
  AccentRed: '#FF87AF',
  DiffAdded: '#005F00',
  DiffRemoved: '#5F0000',
  Comment: '#AFAFAF',
  Gray: '#AFAFAF',
  DarkGray: '#878787',
  InputBackground: '#5F5F5F',
  MessageBackground: '#5F5F5F',
  FocusBackground: '#005F00',
  GradientColors: ['#4796E4', '#847ACE', '#C3677F'],
};

    this.semanticColors = semanticColors ?? {
      text: {
        primary: this.colors.Foreground,
        secondary: this.colors.Gray,
        link: this.colors.AccentBlue,
        accent: this.colors.AccentPurple,
        response: this.colors.Foreground,
      },
      background: {
        primary: this.colors.Background,
        message:
          this.colors.MessageBackground ??
          '#2A2A2A',
        input:
          this.colors.InputBackground ??
          '#2A2A2A',
        focus:
          this.colors.FocusBackground ??
          '#2B332B',
        diff: {
          added: this.colors.DiffAdded,
          removed: this.colors.DiffRemoved,
        },
      },
      border: {
        default: this.colors.DarkGray,
      },
      ui: {
        comment: this.colors.Gray,
        symbol: this.colors.AccentCyan,
        active: this.colors.AccentBlue,
        dark: this.colors.DarkGray,
        focus: this.colors.FocusColor ?? this.colors.AccentGreen,
        gradient: this.colors.GradientColors,
      },
      status: {
        error: this.colors.AccentRed,
        success: this.colors.AccentGreen,
        warning: this.colors.AccentYellow,
      },
    };

export const DEFAULT_BACKGROUND_OPACITY = 0.16;
export const DEFAULT_INPUT_BACKGROUND_OPACITY = 0.24;
export const DEFAULT_SELECTION_OPACITY = 0.2;
export const DEFAULT_BORDER_OPACITY = 0.4;
