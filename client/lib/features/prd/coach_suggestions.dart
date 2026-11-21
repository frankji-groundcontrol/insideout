import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../session/appearance.dart';
import '../../theme/cobalt.dart';
import '../../theme/cobalt_theme.dart';
import 'coach_column.dart';

/// The stage's suggested prompts above the transcript. A null [onPick]
/// (a turn in flight or a back-off) disables every chip.
class CoachSuggestions extends StatelessWidget {
  const CoachSuggestions({
    super.key,
    required this.suggestions,
    required this.onPick,
  });

  final List<String> suggestions;
  final void Function(String prompt)? onPick;

  @override
  Widget build(BuildContext context) {
    final palette = Cobalt.of(context);
    final ink = coachInk(context);
    final pick = onPick;
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: Cobalt.space6),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            context.watch<Appearance>().t('coach.suggestTitle'),
            style: cobaltSerif(
              palette,
              size: 12,
              weight: FontWeight.w600,
              color: ink.withValues(alpha: 0.7),
            ),
          ),
          const SizedBox(height: Cobalt.space2),
          Wrap(
            spacing: Cobalt.space2,
            runSpacing: Cobalt.space2,
            children: [
              for (final prompt in suggestions)
                CoachPrompt(
                  label: prompt,
                  onTap: pick == null ? null : () => pick(prompt),
                ),
            ],
          ),
        ],
      ),
    );
  }
}
