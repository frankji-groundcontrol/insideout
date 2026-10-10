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
  });

  final bool compact;
  final List<Widget> viewChoices;
  final VoidCallback onAdd;

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    final theme = Theme.of(context);
    final title = Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          l10n.t('roadmap.title').toUpperCase(),
          style: theme.textTheme.displaySmall,
        ),
        const SizedBox(height: 8),
        Text(l10n.t('roadmap.lead'), style: theme.textTheme.bodyLarge),
      ],
    );
    final add = BlockButton(
      label: l10n.t('roadmap.newItem'),
      icon: Icons.add,
      onPressed: onAdd,
    );
    return Padding(
      padding: const EdgeInsets.fromLTRB(28, 64, 28, 8),
      child: compact
          ? Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                title,
                const SizedBox(height: Cobalt.space3),
                Wrap(
                  spacing: Cobalt.space2,
                  runSpacing: Cobalt.space2,
                  crossAxisAlignment: WrapCrossAlignment.center,
                  children: [...viewChoices, add],
                ),
              ],
            )
          : Row(
              children: [
                Expanded(child: title),
                add,
              ],
            ),
    );
  }
}
