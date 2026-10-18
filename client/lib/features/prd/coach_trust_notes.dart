import '../../api/models.dart';
import '../../session/appearance.dart';
import 'coach_finding.dart' show coachSeverities;

/// The reasons agent/critic.go records for a pass that did not run. Anything
/// else prints a neutral reason: a raw token would be English in every locale.
const _skipReasons = {'contention', 'critic-output-failure'};

/// What the critic did not do, in words: the sentence the finalize card and
/// the live notice share. Null when every critic pass that was due ran
/// (rv4-AIDEV-02).
String? criticSkipNote(Appearance l10n, String? reason, int rounds) {
  if (reason == null || reason.isEmpty) return null;
  final why = l10n.t(
    'coach.criticReason.${_skipReasons.contains(reason) ? reason : 'other'}',
  );
  return l10n.t(rounds == 0 ? 'coach.criticNone' : 'coach.criticPartial', {
    'reason': why,
  });
}

/// What Confirm will do to each finding the coach parked, one line each
/// (rv4-AIDEV-03). The server only lists statuses it will apply; an unknown one
/// prints nothing, and so does a severity or section this build cannot name.
List<String> resolutionNotes(Appearance l10n, List<PendingResolution> parked) =>
    [
      for (final r in parked)
        if (r.status == 'resolved' || r.status == 'overridden')
          l10n.t('coach.resolution.${r.status}', {
            'where': [
              if (coachSeverities.contains(r.severity))
                l10n.t('coach.finding.severity.${r.severity}'),
              if (prdSectionKeys.contains(r.section))
                l10n.t('prd.sections.${r.section}'),
            ].join(' / '),
            'issue': r.issue,
          }),
    ];

/// The lines under the stage proposal: what the person needs to know before
/// confirming it.
List<String> coachTrustNotes(Appearance l10n, Conversation conv) => [
  ?criticSkipNote(l10n, conv.criticSkipped, conv.criticRounds),
  ...resolutionNotes(l10n, conv.pendingResolutions),
];
