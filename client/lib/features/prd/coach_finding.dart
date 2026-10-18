import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../session/appearance.dart';
import 'coach_column.dart';

/// The severities agent/critic.go emits. Anything else prints no severity: a
/// raw token would be English in every locale.
const coachSeverities = {'blocking', 'major', 'minor'};

/// One critic finding as a pull-quote: the issue, a localized
/// "severity / label" line, and the critic's suggestion when it gave one.
class CoachFinding extends StatelessWidget {
  const CoachFinding({super.key, required this.finding});

  /// One entry of a critic_findings event (agent/critic.go CriticFinding).
  final Map<String, dynamic> finding;

  @override
  Widget build(BuildContext context) {
    final issue = finding['issue'];
    if (issue is! String) return const SizedBox.shrink();
    final t = context.watch<Appearance>().t;
    final severity = finding['severity'];
    final suggestion = finding['suggestion'];
    final label = t('coach.finding.label');
    return CoachQuote(
      text: issue,
      role: coachSeverities.contains(severity)
          ? '${t('coach.finding.severity.$severity')} / $label'
          : label,
      note: suggestion is String && suggestion.isNotEmpty
          ? t('coach.finding.suggestion', {'text': suggestion})
          : null,
    );
  }
}
