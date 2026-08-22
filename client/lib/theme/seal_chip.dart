import 'package:flutter/material.dart';

import 'ink_seal.dart';

enum SealTone { locked, pending, progress, done, neutral, danger }

/// Vermilion / grey / sage chop used for roadmap lifecycle and landing peeks.
class SealChip extends StatelessWidget {
  const SealChip({super.key, required this.label, required this.tone, this.compact = true});

  final String label;
  final SealTone tone;
  final bool compact;

  static SealTone fromStatus(String status) {
    return switch (status) {
      'done' => SealTone.done,
      'in_progress' => SealTone.progress,
      'locked' => SealTone.locked,
      _ => SealTone.pending,
    };
  }

  @override
  Widget build(BuildContext context) {
    final palette = Theme.of(context).brightness == Brightness.dark ? InkSeal.dark : InkSeal.light;
    final (bg, fg) = switch (tone) {
      SealTone.done => (palette.statusSuccessBg, palette.statusSuccessFg),
      SealTone.progress => (palette.statusInfoBg, palette.statusInfoFg),
      SealTone.locked => (palette.statusNeutralBg, palette.sealLocked),
      SealTone.pending => (palette.statusNeutralBg, palette.statusNeutralFg),
      SealTone.neutral => (palette.statusNeutralBg, palette.statusNeutralFg),
      SealTone.danger => (palette.statusInfoBg, palette.fgDanger),
    };
    return Container(
      padding: EdgeInsets.symmetric(horizontal: compact ? 6 : 10, vertical: compact ? 2 : 4),
      decoration: BoxDecoration(
        color: bg,
        borderRadius: BorderRadius.circular(InkSeal.radiusControl),
      ),
      child: Text(
        label,
        style: Theme.of(context).textTheme.bodySmall?.copyWith(color: fg, fontWeight: FontWeight.w600, fontSize: 10),
      ),
    );
  }
}

/// Square 印-shaped node mark. Grey locked chop, vermilion in-progress, sage done.
class SealMark extends StatelessWidget {
  const SealMark({super.key, required this.status, this.size = 18});

  final String status;
  final double size;

  @override
  Widget build(BuildContext context) {
    final palette = Theme.of(context).brightness == Brightness.dark ? InkSeal.dark : InkSeal.light;
    final color = switch (status) {
      'done' => palette.statusSuccessFg,
      'in_progress' => palette.seal,
      'locked' => palette.sealLocked,
      _ => palette.statusNeutralFg,
    };
    return Container(
      width: size,
      height: size,
      decoration: BoxDecoration(
        color: color,
        borderRadius: BorderRadius.circular(4),
      ),
    );
  }
}
