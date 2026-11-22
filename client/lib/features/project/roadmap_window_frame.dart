import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../api/models.dart';
import '../../session/appearance.dart';
import '../../theme/block_icon_button.dart';
import '../../theme/cobalt.dart';
import '../../theme/cobalt_chip.dart';
import '../../theme/field_card.dart';
import 'roadmap_window_handles.dart';

/// A node window's chrome: a squared card with a title bar (status, title,
/// Open roadmap, close) over a body sized to the window, which scrolls
/// itself. With [onMove]
/// the title bar drags it; with [onResize] its right and bottom edges and
/// corner resize it. Without them it is the phone's full-screen sheet.
class RoadmapWindowFrame extends StatelessWidget {
  const RoadmapWindowFrame({
    super.key,
    required this.node,
    required this.onClose,
    required this.onOpenRoadmap,
    required this.body,
    this.onFocus,
    this.onMove,
    this.onResize,
  });

  final RoadmapNode node;
  final VoidCallback onClose;
  final VoidCallback onOpenRoadmap;
  final Widget body;
  final VoidCallback? onFocus;
  final void Function(Offset delta)? onMove;
  final void Function(Offset delta)? onResize;

  @override
  Widget build(BuildContext context) {
    final palette = Cobalt.of(context);
    final card = FieldCard(
      key: Key('roadmap-window-${node.id}'),
      padding: EdgeInsets.zero,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          _titleBar(context),
          Divider(
            height: Cobalt.borderContainer,
            thickness: Cobalt.borderContainer,
            color: palette.strokeSubtle,
          ),
          // The body lays out to the window, not the screen: a small
          // window gets the phone layout.
          Expanded(
            child: LayoutBuilder(
              builder: (context, box) => MediaQuery(
                data: MediaQuery.of(context).copyWith(
                  size: box.biggest,
                  padding: EdgeInsets.zero,
                  viewPadding: EdgeInsets.zero,
                  viewInsets: EdgeInsets.zero,
                ),
                child: body,
              ),
            ),
          ),
        ],
      ),
    );
    final resize = onResize;
    return Listener(
      onPointerDown: onFocus == null ? null : (_) => onFocus!(),
      child: resize == null
          ? card
          : Stack(
              children: [
                Positioned.fill(child: card),
                ...roadmapWindowHandles(resize),
              ],
            ),
    );
  }

  Widget _titleBar(BuildContext context) {
    final t = context.watch<Appearance>().t;
    final palette = Cobalt.of(context);
    final bar = Container(
      color: palette.surfaceSunken,
      padding: const EdgeInsets.fromLTRB(
        Cobalt.space4,
        Cobalt.space1,
        Cobalt.space2,
        Cobalt.space1,
      ),
      child: Row(
        children: [
          ShadeChip(
            label: t('roadmap.status.${node.status}'),
            tone: ShadeChip.fromStatus(node.status),
          ),
          const SizedBox(width: Cobalt.space2),
          Expanded(
            child: Text(
              node.title,
              key: Key('roadmap-window-title-${node.id}'),
              style: Theme.of(context).textTheme.titleSmall,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
            ),
          ),
          const SizedBox(width: Cobalt.space2),
          BlockIconButton(
            key: Key('roadmap-window-open-${node.id}'),
            icon: Icons.open_in_new,
            tooltip: t('roadmap.window.open'),
            semanticLabel: t('roadmap.window.open'),
            onPressed: onOpenRoadmap,
          ),
          const SizedBox(width: Cobalt.space1),
          BlockIconButton(
            key: Key('roadmap-window-close-${node.id}'),
            icon: Icons.close,
            tooltip: t('roadmap.window.close'),
            semanticLabel: t('roadmap.window.close'),
            onPressed: onClose,
          ),
        ],
      ),
    );
    final move = onMove;
    if (move == null) return bar;
    return MouseRegion(
      cursor: SystemMouseCursors.move,
      child: GestureDetector(
        key: Key('roadmap-window-bar-${node.id}'),
        onPanUpdate: (d) => move(d.delta),
        child: bar,
      ),
    );
  }
}
