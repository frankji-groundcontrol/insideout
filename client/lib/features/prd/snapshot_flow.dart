import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../session/appearance.dart';
import '../../session/session.dart';
import '../../theme/block_button.dart';
import '../../theme/block_field.dart';
import '../../theme/block_modal.dart';
import '../../widgets/guarded_action.dart';

/// One snapshot flow for every caller (CQ-03): prd_page and revisions_page
/// carried byte-diverged copies of this modal + write + tail. [onCreated]
/// runs after a successful snapshot while the context is still mounted
/// (revisions reload its table; prd_page has no tail).
Future<void> showSnapshotRevision(
  BuildContext context, {
  required String prdId,
  void Function()? onCreated,
}) async {
  final note = TextEditingController();
  final t = context.read<Appearance>().t;
  final navigator = Navigator.of(context);
  final ok = await showBlockModal<bool>(
    context: context,
    title: t('prd.snapshotRevision'),
    body: BlockField(label: t('prd.revisionNote'), controller: note),
    footer: [
      BlockButton(
        label: t('common.cancel'),
        variant: BlockButtonVariant.secondary,
        onPressed: () => navigator.pop(false),
      ),
      BlockButton(
        label: t('common.save'),
        onPressed: () => navigator.pop(true),
      ),
    ],
  );
  if (ok != true || !context.mounted) return;
  final done = await guardWrite(
    context,
    () => context.read<Session>().api.createRevision(
      prdId,
      note: note.text.trim().isEmpty ? null : note.text.trim(),
    ),
  );
  if (done && context.mounted) onCreated?.call();
  // The controller is not disposed: the pop animation keeps the modal's
  // field mounted past this await, and the guard's SnackBar rebuilds the
  // overlay — a disposed controller would throw. GC'd on unmount.
}
