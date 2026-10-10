import 'package:flutter/material.dart';

import '../../theme/cobalt.dart';
import '../../widgets/pressable_block.dart';

/// Side of the quick-capture block. The workspace board derives its reserved
/// bottom scroll inset from this, so the two cannot drift apart.
const double quickCaptureSize = 56;

/// The workspace quick-capture control: a primary block — cobalt, edged in
/// ink — on [PressableBlock], so it carries the (4,4) cast at rest and the
/// 100ms press travel every other block has. Material's FloatingActionButton
/// can do neither. It is icon-only, so [label] is both the hover/long-press
/// tooltip and the screen-reader name.
class QuickCaptureFab extends StatelessWidget {
  const QuickCaptureFab({
    super.key,
    required this.label,
    required this.onPressed,
  });

  final String label;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    final palette = Cobalt.of(context);
    return Tooltip(
      message: label,
      // The Semantics below already names the control; do not announce twice.
      excludeFromSemantics: true,
      child: Semantics(
        label: label,
        child: PressableBlock(
          onTap: onPressed,
          child: Container(
            width: quickCaptureSize,
            height: quickCaptureSize,
            alignment: Alignment.center,
            decoration: BoxDecoration(
              color: palette.btn,
              border: Border.all(
                color: Cobalt.lightInk,
                width: Cobalt.borderContainer,
              ),
            ),
            child: Icon(Icons.add, color: palette.btnFg),
          ),
        ),
      ),
    );
  }
}
