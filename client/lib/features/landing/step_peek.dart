import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../session/appearance.dart';
import '../../theme/block_chrome.dart';
import '../../theme/cobalt.dart';
import '../../theme/cobalt_chip.dart';
import '../../theme/field_card.dart';

enum PeekKind { idea, prd, roadmap, shipped }

/// 1:1 call-out of the real artifact each Assembly step produces.
class StepPeek extends StatelessWidget {
  const StepPeek({super.key, required this.kind});

  final PeekKind kind;

  @override
  Widget build(BuildContext context) {
    final palette = Cobalt.of(context);
    return FieldCard(
      padding: const EdgeInsets.fromLTRB(20, 20, 20, Cobalt.space4),
      child: Stack(
        children: [
          Padding(
            padding: const EdgeInsets.only(top: Cobalt.space2),
            child: switch (kind) {
              PeekKind.idea => _IdeaPeek(palette: palette),
              PeekKind.prd => _PrdPeek(palette: palette),
              PeekKind.roadmap => _RoadmapPeek(palette: palette),
              PeekKind.shipped => _ShippedPeek(palette: palette),
            },
          ),
          const Positioned(top: 0, right: 0, child: BlockTag(label: '1:1')),
        ],
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
      child: Container(height: height, color: color),
    );
  }
}

class _IdeaPeek extends StatelessWidget {
  const _IdeaPeek({required this.palette});

  final Cobalt palette;

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            // Identity exception: the mark is a disc.
            Container(
              width: 8,
              height: 8,
              decoration: BoxDecoration(
                color: palette.brand,
                shape: BoxShape.circle,
              ),
            ),
            const SizedBox(width: Cobalt.space2),
            Expanded(
              child: _Bar(
                widthFactor: 0.55,
                color: palette.fgPrimary.withValues(alpha: 0.75),
              ),
            ),
          ],
        ),
        const SizedBox(height: Cobalt.space3),
        _Bar(widthFactor: 0.92, color: palette.surfaceSunken),
        const SizedBox(height: Cobalt.space2),
        _Bar(widthFactor: 0.58, color: palette.surfaceSunken),
        const SizedBox(height: Cobalt.space3),
        ShadeChip(label: l10n.t('idea.status.inbox'), tone: ShadeTone.neutral),
      ],
    );
  }
}

class _PrdPeek extends StatelessWidget {
  const _PrdPeek({required this.palette});

  final Cobalt palette;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Align(
          alignment: Alignment.centerLeft,
          child: Container(
            padding: const EdgeInsets.symmetric(
              horizontal: Cobalt.space3,
              vertical: Cobalt.space2,
            ),
            color: palette.surfaceSunken,
            child: SizedBox(
              width: 160,
              child: _Bar(
                widthFactor: 1,
                color: palette.strokeSubtle,
                height: 8,
              ),
            ),
          ),
        ),
        const SizedBox(height: Cobalt.space2),
        Align(
          alignment: Alignment.centerRight,
          child: Container(
            padding: const EdgeInsets.symmetric(
              horizontal: Cobalt.space3,
              vertical: Cobalt.space2,
            ),
            color: palette.btn,
            child: SizedBox(
              width: 96,
              child: _Bar(
                widthFactor: 1,
                color: palette.btnFg.withValues(alpha: 0.7),
                height: 8,
              ),
            ),
          ),
        ),
        const SizedBox(height: Cobalt.space3),
        Container(
          padding: const EdgeInsets.all(Cobalt.space3),
          decoration: BoxDecoration(
            border: Border.all(
              color: palette.strokeSubtle,
              width: Cobalt.borderContainer,
            ),
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              SizedBox(
                width: 96,
                child: _Bar(
                  widthFactor: 1,
                  color: palette.fgPrimary.withValues(alpha: 0.8),
                  height: 10,
                ),
              ),
              const SizedBox(height: Cobalt.space2),
              _Bar(widthFactor: 1, color: palette.surfaceSunken, height: 6),
              const SizedBox(height: Cobalt.space1),
              _Bar(widthFactor: 0.83, color: palette.surfaceSunken, height: 6),
              const SizedBox(height: Cobalt.space1),
              _Bar(
                widthFactor: 0.5,
                color: palette.brand.withValues(alpha: 0.7),
                height: 6,
              ),
            ],
          ),
        ),
      ],
    );
  }
}

class _RoadmapPeek extends StatelessWidget {
  const _RoadmapPeek({required this.palette});

  final Cobalt palette;

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    return Column(
      children: [
        _RoadRow(
          dot: palette.statusSuccessFg,
          bar: 0.7,
          chip: ShadeChip(
            label: l10n.t('roadmap.status.done'),
            tone: ShadeTone.done,
          ),
        ),
        _RoadRow(
          dot: palette.brand,
          bar: 0.55,
          indent: true,
          chip: ShadeChip(
            label: l10n.t('roadmap.status.in_progress'),
            tone: ShadeTone.progress,
          ),
        ),
        _RoadRow(
          dot: palette.statusNeutralFg,
          bar: 0.45,
          indent: true,
          chip: ShadeChip(
            label: l10n.t('roadmap.status.locked'),
            tone: ShadeTone.locked,
          ),
        ),
      ],
    );
  }
}

class _RoadRow extends StatelessWidget {
  const _RoadRow({
    required this.dot,
    required this.bar,
    required this.chip,
    this.indent = false,
  });

  final Color dot;
  final double bar;
  final Widget chip;
  final bool indent;

  @override
  Widget build(BuildContext context) {
    final palette = Cobalt.of(context);
    return Padding(
      padding: EdgeInsets.only(
        left: indent ? Cobalt.space4 : 0,
        bottom: Cobalt.space2,
      ),
      child: Row(
        children: [
          Container(width: 10, height: 10, color: dot),
          const SizedBox(width: Cobalt.space2),
          Expanded(
            child: _Bar(widthFactor: bar, color: palette.surfaceSunken),
          ),
          const SizedBox(width: Cobalt.space2),
          chip,
        ],
      ),
    );
  }
}

class _ShippedPeek extends StatelessWidget {
  const _ShippedPeek({required this.palette});

  final Cobalt palette;

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    const hashes = ['a1b2c3d', 'e4f5a6b', 'c7d8e9f'];
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        for (final h in hashes)
          Padding(
            padding: const EdgeInsets.only(bottom: Cobalt.space3),
            child: Row(
              children: [
                Container(width: 14, height: 14, color: palette.brand),
                const SizedBox(width: Cobalt.space3),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      _Bar(widthFactor: 0.7, color: palette.surfaceSunken),
                      const SizedBox(height: Cobalt.space1),
                      _Bar(
                        widthFactor: 0.28,
                        color: palette.surfaceSunken.withValues(alpha: 0.7),
                        height: 6,
                      ),
                    ],
                  ),
                ),
                Text(
                  h,
                  style: Theme.of(
                    context,
                  ).textTheme.bodySmall?.copyWith(fontFamily: 'monospace'),
                ),
              ],
            ),
          ),
        Text(
          l10n.t('github.synced', {'count': 12}),
          style: Theme.of(context).textTheme.bodySmall,
        ),
      ],
    );
  }
}
