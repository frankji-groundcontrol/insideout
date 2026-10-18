import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../session/appearance.dart';
import '../../theme/cobalt.dart';
import '../../theme/cobalt_theme.dart';
import 'coach_column.dart';

/// The coach's stage proposal, parked until the member ratifies it. Confirm
/// asks the page to write the stage server-side; decline asks it to clear the
/// proposal server-side (rv4-CLOSE-03), keeping the confirmed stage. [notes]
/// are the lines a person should read before answering
/// (coach_trust_notes.dart); [enabled] is false while they may still change.
class CoachStageProposal extends StatefulWidget {
  const CoachStageProposal({
    super.key,
    required this.proposed,
    required this.onConfirm,
    required this.onDecline,
    this.notes = const [],
    this.enabled = true,
  });

  final String proposed;
  final List<String> notes;
  final bool enabled;
  final Future<void> Function() onConfirm;
  final VoidCallback onDecline;

  @override
  State<CoachStageProposal> createState() => _CoachStageProposalState();
}

class _CoachStageProposalState extends State<CoachStageProposal> {
  bool confirming = false;

  Future<void> _confirm() async {
    if (confirming) return;
    setState(() => confirming = true);
    try {
      await widget.onConfirm();
    } finally {
      // On success the page clears the proposal, unmounting this widget.
      if (mounted) setState(() => confirming = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    final palette = Cobalt.of(context);
    final ink = coachInk(context);
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: Cobalt.space6),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            l10n.t('coach.stageProposed', {
              'stage': l10n.t('coach.stage.${widget.proposed}'),
            }),
            style: cobaltSerif(
              palette,
              size: 12,
              color: ink.withValues(alpha: 0.7),
              weight: FontWeight.w600,
            ),
          ),
          for (final note in widget.notes)
            Padding(
              padding: const EdgeInsets.only(top: Cobalt.space2),
              child: Text(
                note,
                style: cobaltSerif(
                  palette,
                  size: 12,
                  color: ink.withValues(alpha: 0.7),
                ),
              ),
            ),
          const SizedBox(height: Cobalt.space2),
          Wrap(
            spacing: Cobalt.space2,
            runSpacing: Cobalt.space2,
            children: [
              CoachPrompt(
                key: const Key('coach-stage-confirm'),
                label: l10n.t('coach.stageConfirm'),
                onTap: confirming || !widget.enabled ? null : _confirm,
              ),
              CoachPrompt(
                key: const Key('coach-stage-decline'),
                label: l10n.t('coach.stageDecline'),
                onTap: widget.enabled ? widget.onDecline : null,
              ),
            ],
          ),
        ],
      ),
    );
  }
}
