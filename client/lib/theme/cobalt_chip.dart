import 'package:flutter/material.dart';

import 'cobalt.dart';

enum ShadeTone { locked, pending, progress, done, neutral, danger }

/// Cobalt-shade chip. One hue — not a chop.
class ShadeChip extends StatelessWidget {
  const ShadeChip({
    super.key,
    required this.label,
    required this.tone,
    this.compact = true,
  });

  final String label;
  final ShadeTone tone;
  final bool compact;

  static ShadeTone fromStatus(String status) {
    return switch (status) {
      'done' => ShadeTone.done,
      'in_progress' => ShadeTone.progress,
      'locked' => ShadeTone.locked,
      _ => ShadeTone.pending,
    };
  }

  @override
  Widget build(BuildContext context) {
    final palette = Cobalt.of(context);
    final (bg, fg) = switch (tone) {
      ShadeTone.done => (palette.statusSuccessBg, palette.statusSuccessFg),
      ShadeTone.progress => (palette.brand, palette.btnFg),
      ShadeTone.locked => (palette.statusNeutralBg, palette.statusNeutralFg),
      ShadeTone.pending => (palette.statusNeutralBg, palette.statusNeutralFg),
      ShadeTone.neutral => (palette.statusNeutralBg, palette.statusNeutralFg),
      ShadeTone.danger => (palette.statusWarnBg, palette.statusWarnFg),
    };
    return Container(
      padding: EdgeInsets.symmetric(
        horizontal: compact ? Cobalt.space2 : Cobalt.space3,
        vertical: Cobalt.space1,
      ),
      decoration: BoxDecoration(
        color: bg,
        borderRadius: BorderRadius.circular(Cobalt.radiusPill),
        border: Border.all(
          color: palette.strokeSubtle,
          width: Cobalt.borderContainer,
        ),
      ),
      child: Text(
        label,
        style: Theme.of(context).textTheme.labelSmall?.copyWith(
          color: fg,
          fontWeight: FontWeight.w600,
        ),
      ),
    );
  }
}

/// Square status mark — the Cobalt-Neo block language (dots are blocks now).
class ShadeMark extends StatelessWidget {
  const ShadeMark({super.key, required this.status, this.size = 10});

  final String status;
  final double size;

  @override
  Widget build(BuildContext context) {
    final palette = Cobalt.of(context);
    final color = switch (status) {
      'done' => palette.statusSuccessFg,
      'in_progress' => palette.brand,
      'locked' => palette.statusNeutralFg,
      _ => palette.strokeStrong,
    };
    return Container(
      width: size,
      height: size,
      decoration: BoxDecoration(color: color),
    );
  }
}
