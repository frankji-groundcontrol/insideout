import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../session/appearance.dart';
import '../../theme/ink_seal.dart';
import '../../theme/seal_chip.dart';

enum PeekKind { idea, prd, roadmap, shipped }

/// 1:1 call-out of the real artifact each Assembly step produces.
class StepPeek extends StatelessWidget {
  const StepPeek({super.key, required this.kind});

  final PeekKind kind;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final palette = theme.brightness == Brightness.dark ? InkSeal.dark : InkSeal.light;
    return Material(
      color: theme.colorScheme.surface,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(InkSeal.radiusCard),
        side: BorderSide(color: theme.colorScheme.outline),
      ),
      child: Padding(
        padding: const EdgeInsets.fromLTRB(20, 20, 20, 16),
        child: Stack(
          children: [
            Padding(
              padding: const EdgeInsets.only(top: 8),
              child: switch (kind) {
                PeekKind.idea => _IdeaPeek(palette: palette),
                PeekKind.prd => _PrdPeek(palette: palette),
                PeekKind.roadmap => _RoadmapPeek(palette: palette),
                PeekKind.shipped => _ShippedPeek(palette: palette),
              },
            ),
            Positioned(
              top: 0,
              right: 0,
              child: Text('1:1', style: theme.textTheme.bodySmall?.copyWith(letterSpacing: 0.6)),
            ),
          ],
        ),
      ),
    );
  }
}

class _Bar extends StatelessWidget {
  const _Bar({required this.widthFactor, required this.color, this.height = 8});

  final double widthFactor;
  final Color color;
  final double height;

  @override
  Widget build(BuildContext context) {
    return FractionallySizedBox(
      widthFactor: widthFactor,
      alignment: Alignment.centerLeft,
      child: Container(
        height: height,
        decoration: BoxDecoration(color: color, borderRadius: BorderRadius.circular(99)),
      ),
    );
  }
}

class _IdeaPeek extends StatelessWidget {
  const _IdeaPeek({required this.palette});

  final InkSeal palette;

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            Container(width: 8, height: 8, decoration: BoxDecoration(color: palette.seal, shape: BoxShape.circle)),
            const SizedBox(width: 8),
            Expanded(child: _Bar(widthFactor: 0.55, color: palette.fgPrimary.withValues(alpha: 0.75))),
          ],
        ),
        const SizedBox(height: 12),
        _Bar(widthFactor: 0.92, color: palette.surfaceSunken),
        const SizedBox(height: 8),
        _Bar(widthFactor: 0.58, color: palette.surfaceSunken),
        const SizedBox(height: 12),
        SealChip(label: l10n.t('idea.status.inbox'), tone: SealTone.neutral),
      ],
    );
  }
}

class _PrdPeek extends StatelessWidget {
  const _PrdPeek({required this.palette});

  final InkSeal palette;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Align(
          alignment: Alignment.centerLeft,
          child: Container(
            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
            decoration: BoxDecoration(
              color: palette.surfaceSunken,
              borderRadius: const BorderRadius.only(
                topLeft: Radius.circular(4),
                topRight: Radius.circular(12),
                bottomLeft: Radius.circular(12),
                bottomRight: Radius.circular(12),
              ),
            ),
            child: SizedBox(width: 160, child: _Bar(widthFactor: 1, color: palette.strokeSubtle, height: 8)),
          ),
        ),
        const SizedBox(height: 10),
        Align(
          alignment: Alignment.centerRight,
          child: Container(
            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
            decoration: BoxDecoration(
              color: palette.btn,
              borderRadius: const BorderRadius.only(
                topLeft: Radius.circular(12),
                topRight: Radius.circular(4),
                bottomLeft: Radius.circular(12),
                bottomRight: Radius.circular(12),
              ),
            ),
            child: SizedBox(width: 96, child: _Bar(widthFactor: 1, color: palette.btnFg.withValues(alpha: 0.7), height: 8)),
          ),
        ),
        const SizedBox(height: 12),
        Container(
          padding: const EdgeInsets.all(12),
          decoration: BoxDecoration(
            border: Border.all(color: palette.strokeSubtle),
            borderRadius: BorderRadius.circular(InkSeal.radiusControl),
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              SizedBox(width: 96, child: _Bar(widthFactor: 1, color: palette.fgPrimary.withValues(alpha: 0.8), height: 10)),
              const SizedBox(height: 8),
              _Bar(widthFactor: 1, color: palette.surfaceSunken, height: 6),
              const SizedBox(height: 6),
              _Bar(widthFactor: 0.83, color: palette.surfaceSunken, height: 6),
              const SizedBox(height: 6),
              _Bar(widthFactor: 0.5, color: palette.seal.withValues(alpha: 0.7), height: 6),
            ],
          ),
        ),
      ],
    );
  }
}

class _RoadmapPeek extends StatelessWidget {
  const _RoadmapPeek({required this.palette});

  final InkSeal palette;

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    return Column(
      children: [
        _RoadRow(dot: palette.statusSuccessFg, bar: 0.7, chip: SealChip(label: l10n.t('roadmap.status.done'), tone: SealTone.done)),
        _RoadRow(dot: palette.seal, bar: 0.55, indent: true, chip: SealChip(label: l10n.t('roadmap.status.in_progress'), tone: SealTone.progress)),
        _RoadRow(dot: palette.statusNeutralFg, bar: 0.45, indent: true, chip: SealChip(label: l10n.t('roadmap.status.locked'), tone: SealTone.locked)),
      ],
    );
  }
}

class _RoadRow extends StatelessWidget {
  const _RoadRow({required this.dot, required this.bar, required this.chip, this.indent = false});

  final Color dot;
  final double bar;
  final Widget chip;
  final bool indent;

  @override
  Widget build(BuildContext context) {
    final palette = Theme.of(context).brightness == Brightness.dark ? InkSeal.dark : InkSeal.light;
    return Padding(
      padding: EdgeInsets.only(left: indent ? 16 : 0, bottom: 10),
      child: Row(
        children: [
          Container(width: 10, height: 10, decoration: BoxDecoration(color: dot, shape: BoxShape.circle)),
          const SizedBox(width: 10),
          Expanded(child: _Bar(widthFactor: bar, color: palette.surfaceSunken)),
          const SizedBox(width: 8),
          chip,
        ],
      ),
    );
  }
}

class _ShippedPeek extends StatelessWidget {
  const _ShippedPeek({required this.palette});

  final InkSeal palette;

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    const hashes = ['a1b2c3d', 'e4f5g6h', 'i7j8k9l'];
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        for (final h in hashes)
          Padding(
            padding: const EdgeInsets.only(bottom: 12),
            child: Row(
              children: [
                Container(
                  width: 14,
                  height: 14,
                  decoration: BoxDecoration(color: palette.seal, borderRadius: BorderRadius.circular(4)),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      _Bar(widthFactor: 0.7, color: palette.surfaceSunken),
                      const SizedBox(height: 6),
                      _Bar(widthFactor: 0.28, color: palette.surfaceSunken.withValues(alpha: 0.7), height: 6),
                    ],
                  ),
                ),
                Text(h, style: Theme.of(context).textTheme.bodySmall?.copyWith(fontFamily: 'monospace')),
              ],
            ),
          ),
        Text(l10n.t('github.synced', {'count': 12}), style: Theme.of(context).textTheme.bodySmall),
      ],
    );
  }
}
