import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../api/models.dart';
import '../../l10n/l10n.dart';
import '../../session/appearance.dart';
import '../../theme/cobalt.dart';
import '../../widgets/pressable_block.dart';
import 'roadmap_painters.dart';

/// Minimap: one tiny column per root band, one shape-coded mark per
/// node (three levels deep; deeper nodes fold into their parent's
/// column). Tapping a column scrolls the canvas to that band.
///
/// Shape — not color — carries the state so the marks read in
/// grayscale and for color-blind users: done = filled block,
/// in_progress = half-filled, locked = dashed outline, pending =
/// hollow outline. Every mark also exposes its state to screen
/// readers and hover tooltips via [_MinimapMark].
class RoadmapMinimap extends StatelessWidget {
  const RoadmapMinimap({
    super.key,
    required this.nodes,
    required this.onTapBand,
  });

  final List<RoadmapNode> nodes;
  final void Function(int bandIndex) onTapBand;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final roots = nodes.where((n) => n.parentId == null).toList()
      ..sort((a, b) => a.position.compareTo(b.position));

    // The titles sit on the 12px floor, so a few bands can outgrow a phone:
    // scroll instead of overflowing, still centered while they fit.
    return SizedBox(
      height: 84,
      child: Center(
        child: SingleChildScrollView(
          scrollDirection: Axis.horizontal,
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              for (var i = 0; i < roots.length; i++)
                Padding(
                  padding: const EdgeInsets.symmetric(horizontal: 6),
                  // The whole column is the pressable block (D7): a decorated
                  // container, so every pixel of the band is tappable — no
                  // Material ripple in the block language.
                  child: PressableBlock(
                    onTap: () => onTapBand(i),
                    child: Container(
                      decoration: BoxDecoration(
                        color: theme.colorScheme.surfaceContainerHighest,
                        border: Border.all(
                          color: theme.colorScheme.outlineVariant,
                        ),
                      ),
                      padding: const EdgeInsets.fromLTRB(6, 2, 6, 0),
                      child: Column(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          Expanded(
                            child: SizedBox(
                              width: 18,
                              child: Column(
                                mainAxisAlignment:
                                    MainAxisAlignment.spaceEvenly,
                                children: [
                                  for (final n in _bandMarks(roots[i]))
                                    _MinimapMark(status: n.status),
                                ],
                              ),
                            ),
                          ),
                          const SizedBox(height: 2),
                          Text(
                            _clip(roots[i].title),
                            style: theme.textTheme.bodySmall,
                            overflow: TextOverflow.ellipsis,
                          ),
                        ],
                      ),
                    ),
                  ),
                ),
            ],
          ),
        ),
      ),
    );
  }

  /// Up to seven marks per band (root first, then children in order) —
  /// enough shape to recognize a branch at a glance.
  List<RoadmapNode> _bandMarks(RoadmapNode root) {
    final kids = nodes.where((n) => n.parentId == root.id).toList()
      ..sort((a, b) => a.position.compareTo(b.position));
    return [root, ...kids].take(7).toList();
  }

  String _clip(String s) => s.length <= 6 ? s : s.substring(0, 6);
}

/// One minimap mark. The shape encodes the node state (see
/// [RoadmapMinimap]); [Semantics]/[Tooltip] carry the same state as
/// text, from the `roadmap.minimap.status.*` l10n keys.
/// Stroke/fill colors come from Cobalt tokens so every state stays
/// ≥3:1 against the sunken column background in both themes.
class _MinimapMark extends StatelessWidget {
  const _MinimapMark({required this.status});

  final String status;

  @override
  Widget build(BuildContext context) {
    final cobalt = Cobalt.of(context);
    final Widget mark;
    switch (status) {
      case 'done':
        mark = Container(
          width: double.infinity,
          height: 4,
          decoration: BoxDecoration(
            color: cobalt.statusSuccessFg,
            borderRadius: BorderRadius.zero,
          ),
        );
      case 'in_progress':
        mark = Container(
          width: double.infinity,
          height: 4,
          decoration: BoxDecoration(
            border: Border.all(
              color: cobalt.strokeStrong,
              width: Cobalt.borderDivider,
            ),
            borderRadius: BorderRadius.zero,
          ),
          child: FractionallySizedBox(
            alignment: Alignment.centerLeft,
            widthFactor: 0.5,
            child: Container(color: cobalt.fgBrand),
          ),
        );
      case 'locked':
        mark = CustomPaint(
          painter: MinimapDashMarkPainter(cobalt.strokeStrong),
          child: const SizedBox(width: double.infinity, height: 4),
        );
      default:
        mark = Container(
          width: double.infinity,
          height: 4,
          decoration: BoxDecoration(
            border: Border.all(
              color: cobalt.fgMuted,
              width: Cobalt.borderDivider,
            ),
            borderRadius: BorderRadius.zero,
          ),
        );
    }
    final label = _label(context);
    return Semantics(
      container: true,
      label: label,
      child: Tooltip(message: label, excludeFromSemantics: true, child: mark),
    );
  }

  /// The state in the active locale; the English table when the minimap is
  /// shown without an [Appearance] above it.
  String _label(BuildContext context) {
    final state = switch (status) {
      'done' || 'in_progress' || 'locked' => status,
      _ => 'pending',
    };
    final key = 'roadmap.minimap.status.$state';
    return context.watch<Appearance?>()?.t(key) ?? lookup('en-US', key);
  }
}
