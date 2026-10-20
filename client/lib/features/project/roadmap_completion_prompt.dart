import 'package:flutter/widgets.dart';

import '../../api/models.dart';
import '../../theme/block_form.dart';

/// Applies a status pick. When the pick marks an in-progress node done
/// and the node has no evidence, asks for the one-sentence override
/// first. Cancel leaves the control unchanged. [reason] is null when the
/// move does not need one (including when evidence already covers it).
Future<void> pickRoadmapDoneStatus({
  required BuildContext context,
  required RoadmapNode node,
  required String current,
  required String? picked,
  required Future<List<Map<String, dynamic>>> Function() loadEvidence,
  required void Function(String status, String? reason) apply,
  required String Function(String key) t,
}) async {
  final next = picked ?? current;
  final markingDone =
      node.status == 'in_progress' && current != 'done' && next == 'done';
  if (!markingDone) {
    apply(next, null);
    return;
  }
  List<Map<String, dynamic>> rows;
  try {
    rows = await loadEvidence();
  } on Exception {
    return;
  }
  if (!context.mounted) return;
  if (rows.isNotEmpty) {
    apply(next, null);
    return;
  }
  final values = await showBlockForm(
    context: context,
    title: t('roadmap.completionReason'),
    confirmLabel: t('common.confirm'),
    fields: [
      BlockFormField(
        label: t('roadmap.completionReasonHint'),
        multiline: true,
        required: true,
      ),
    ],
  );
  if (values == null || !context.mounted) return;
  final reason = values.first.trim();
  if (reason.isEmpty) return;
  apply(next, reason);
}
