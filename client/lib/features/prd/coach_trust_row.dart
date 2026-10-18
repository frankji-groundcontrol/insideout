import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../api/models.dart';
import '../../session/appearance.dart';
import '../../theme/cobalt.dart';
import 'coach_stage_proposal.dart';
import 'coach_trust_notes.dart';

/// The stage proposal under the masthead, with the trust notes the stored
/// conversation carries. [busy] holds the confirm button while a turn streams:
/// the notes come from the conversation view, which the page refreshes when the
/// turn ends, so a confirm in the middle of a turn would be an uninformed one.
class CoachProposalRow extends StatelessWidget {
  const CoachProposalRow({
    super.key,
    required this.conversation,
    required this.busy,
    required this.onConfirm,
    required this.onDecline,
  });

  final Conversation conversation;
  final bool busy;
  final Future<void> Function() onConfirm;
  final VoidCallback onDecline;

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    return Padding(
      padding: const EdgeInsets.only(bottom: Cobalt.space3),
      child: CoachStageProposal(
        proposed: conversation.pendingStage!,
        notes: coachTrustNotes(l10n, conversation),
        canConfirm: !busy,
        onConfirm: onConfirm,
        onDecline: onDecline,
      ),
    );
  }
}
