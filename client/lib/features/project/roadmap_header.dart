import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../session/appearance.dart';
import '../../theme/block_button.dart';
import '../../theme/cobalt.dart';

/// The roadmap's title block and its "New item" button.
///
/// [compact] (a phone-width screen) stacks them: beside the button the title
/// was squeezed until "ROADMAP" broke mid-word at 390px. The view toggles,
/// which no longer fit the top-right utility cluster there, are passed in and
/// sit with the button.
class RoadmapHeader extends StatelessWidget {
  const RoadmapHeader({
    super.key,
    required this.compact,
    required this.viewChoices,
    required this.onAdd,
    this.breadcrumb,
    this.title,
    this.top = 64,
    this.windowed = false,
  });

  /// In a node's window: the title bar already names the node, so the big
  /// title and the lead line go, and the toggles share one row with the
  /// button.
  final bool windowed;

  /// The room above the title: the page clears the utility block; a node's
  /// window has none.
  final double top;

  final bool compact;
  final List<Widget> viewChoices;

  /// Null hides "New item": the caller may not write where it would add.
  final VoidCallback? onAdd;

  /// The path back to the whole tree, when a node is open as its own tree.
  final Widget? breadcrumb;

  /// The opened node's title, which names the page in place of "ROADMAP".
  final String? title;

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    final theme = Theme.of(context);
    final heading = Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        ?breadcrumb,
        Text(
          title ?? l10n.t('roadmap.title').toUpperCase(),
          key: const Key('roadmap-title'),
          style: theme.textTheme.displaySmall,
        ),
        const SizedBox(height: 8),
        Text(l10n.t('roadmap.lead'), style: theme.textTheme.bodyLarge),
      ],
    );
    final add = onAdd == null
        ? null
        : BlockButton(
            label: l10n.t('roadmap.newItem'),
            icon: Icons.add,
            onPressed: onAdd,
          );
    if (windowed) {
      return Padding(
        padding: EdgeInsets.fromLTRB(20, top, 20, 4),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            ?breadcrumb,
            const SizedBox(height: Cobalt.space2),
            Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Expanded(
                  child: Wrap(
                    spacing: Cobalt.space2,
                    runSpacing: Cobalt.space2,
                    children: viewChoices,
                  ),
                ),
                if (add != null) ...[const SizedBox(width: Cobalt.space2), add],
              ],
            ),
          ],
        ),
      );
    }
    return Padding(
      padding: EdgeInsets.fromLTRB(28, top, 28, 8),
      child: compact
          ? Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                heading,
                const SizedBox(height: Cobalt.space3),
                Wrap(
                  spacing: Cobalt.space2,
                  runSpacing: Cobalt.space2,
                  crossAxisAlignment: WrapCrossAlignment.center,
                  children: [...viewChoices, ?add],
                ),
              ],
            )
          : Row(
              children: [
                Expanded(child: heading),
                ?add,
              ],
            ),
    );
  }
}
