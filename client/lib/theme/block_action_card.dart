import 'package:flutter/material.dart';

import 'block_chrome.dart';
import 'cobalt.dart';
import 'field_card.dart';

/// A one-line action row: icon well, title, chevron. The home field's
/// create/join rows and the workspace page's nav cards are the same tile —
/// one definition here so a press or focus change lands everywhere at once.
class BlockActionCard extends StatelessWidget {
  const BlockActionCard({
    super.key,
    required this.icon,
    required this.label,
    required this.onTap,
  });

  final IconData icon;
  final String label;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final palette = Cobalt.of(context);
    return FieldCard(
      onTap: onTap,
      padding: const EdgeInsets.all(Cobalt.space4),
      child: Row(
        children: [
          BlockIconWell(icon: icon, size: 40, iconSize: 20),
          const SizedBox(width: Cobalt.space3),
          Expanded(child: Text(label, style: theme.textTheme.titleMedium)),
          Icon(Icons.chevron_right, color: palette.fgMuted),
        ],
      ),
    );
  }
}
