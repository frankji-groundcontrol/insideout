import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../session/appearance.dart';
import '../../theme/cobalt.dart';
import '../../theme/cobalt_theme.dart';
import 'coach_column.dart';

/// What the coach column shows before the viewer has a conversation: [message]
/// and, when they may coach this PRD, a block that starts one. A null
/// [onStart] withholds the block.
class CoachStart extends StatefulWidget {
  const CoachStart({super.key, required this.message, this.onStart});

  final String message;
  final Future<void> Function()? onStart;

  @override
  State<CoachStart> createState() => _CoachStartState();
}

class _CoachStartState extends State<CoachStart> {
  bool starting = false;

  Future<void> _start() async {
    final onStart = widget.onStart;
    if (onStart == null || starting) return;
    setState(() => starting = true);
    try {
      await onStart();
    } finally {
      // On success the page swaps this widget for the conversation.
      if (mounted) setState(() => starting = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    return Padding(
      padding: const EdgeInsets.all(Cobalt.space6),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            widget.message,
            style: cobaltSerif(
              Cobalt.of(context),
              size: 20,
              color: coachInk(context),
            ),
          ),
          if (widget.onStart != null) ...[
            const SizedBox(height: Cobalt.space4),
            CoachPrompt(
              label: l10n.t('coach.start'),
              onTap: starting ? null : _start,
            ),
          ],
        ],
      ),
    );
  }
}
