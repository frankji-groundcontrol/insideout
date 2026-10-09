import 'package:flutter/material.dart';

import '../../theme/block_button.dart';
import '../../theme/cobalt.dart';

/// The audience switcher shared by the Versions stack and the Audience view —
/// [BlockChoice] blocks on the one selection grammar, never Material
/// `ChoiceChip` ripples.
class AudienceSelector extends StatelessWidget {
  const AudienceSelector({
    super.key,
    required this.audiences,
    required this.selected,
    required this.onSelected,
    this.labels,
    this.alignment = WrapAlignment.center,
  });

  final List<String> audiences;
  final String selected;
  final ValueChanged<String> onSelected;

  /// Optional display label per audience key; falls back to the key itself.
  final String Function(String audience)? labels;
  final WrapAlignment alignment;

  @override
  Widget build(BuildContext context) {
    return Wrap(
      spacing: Cobalt.space2,
      runSpacing: Cobalt.space2,
      alignment: alignment,
      children: [
        for (final a in audiences)
          BlockChoice(
            label: labels?.call(a) ?? a,
            selected: a == selected,
            onTap: () => onSelected(a),
          ),
      ],
    );
  }
}
