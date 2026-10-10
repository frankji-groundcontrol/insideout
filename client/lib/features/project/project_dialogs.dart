import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../session/appearance.dart';
import '../../theme/block_button.dart';
import '../../theme/block_field.dart';
import '../../theme/block_modal.dart';
import '../../theme/cobalt.dart';

/// The edit-project modal. Resolves to the fields to save, or null when the
/// user cancelled; the page owns the write.
Future<({String title, String description, String status})?> showProjectEdit(
  BuildContext context,
  Map<String, dynamic> data,
) async {
  final t = context.read<Appearance>().t;
  final navigator = Navigator.of(context);
  final titleCtrl = TextEditingController(text: data['title'] as String? ?? '');
  final descCtrl = TextEditingController(
    text: data['description'] as String? ?? '',
  );
  var status = (data['status'] as String?) ?? 'planning';
  final ok = await showBlockModal<bool>(
    context: context,
    title: t('project.edit'),
    body: StatefulBuilder(
      builder: (ctx, setLocal) => Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          BlockField(
            controller: titleCtrl,
            label: t('project.namePlaceholder'),
          ),
          const SizedBox(height: Cobalt.space4),
          BlockField(
            controller: descCtrl,
            label: t('project.updatePlaceholder'),
          ),
          const SizedBox(height: Cobalt.space4),
          BlockSelect<String>(
            label: t('project.statusLabel'),
            value: status,
            options: [
              for (final s in const [
                'planning',
                'active',
                'paused',
                'done',
                'archived',
              ])
                (value: s, label: t('project.status.$s')),
            ],
            onChanged: (v) => setLocal(() => status = v ?? status),
          ),
        ],
      ),
    ),
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
  // Not disposed: the pop animation keeps the dialog's TextFields mounted
  // past this await, and the guard's SnackBar rebuilds the overlay — a
  // disposed controller would throw. GC'd once the dialog unmounts.
  if (ok != true) return null;
  return (
    title: titleCtrl.text.trim(),
    description: descCtrl.text,
    status: status,
  );
}

/// The delete-project confirm; true only when the user confirmed.
Future<bool> showProjectDelete(BuildContext context) {
  final t = context.read<Appearance>().t;
  return showBlockConfirm(
    context: context,
    title: t('project.delete'),
    message: t('project.deleteConfirm'),
    confirmText: t('common.delete'),
    danger: true,
  );
}

/// The post-an-update modal. Resolves to the kind and trimmed text, or null
/// when the user cancelled or left the text empty.
Future<({String kind, String content})?> showProjectAddUpdate(
  BuildContext context,
) async {
  final t = context.read<Appearance>().t;
  final navigator = Navigator.of(context);
  final content = TextEditingController();
  var kind = 'note';
  final ok = await showBlockModal<bool>(
    context: context,
    title: t('project.addUpdate'),
    body: StatefulBuilder(
      builder: (ctx, setLocal) => Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          BlockSelect<String>(
            value: kind,
            options: [
              for (final k in const ['progress', 'blocker', 'note'])
                (value: k, label: t('project.updateKind.$k')),
            ],
            onChanged: (v) => setLocal(() => kind = v ?? kind),
          ),
          const SizedBox(height: Cobalt.space4),
          BlockField(
            controller: content,
            label: t('project.updatePlaceholder'),
          ),
        ],
      ),
    ),
    footer: [
      BlockButton(
        label: t('common.cancel'),
        variant: BlockButtonVariant.secondary,
        onPressed: () => navigator.pop(false),
      ),
      BlockButton(
        label: t('project.post'),
        onPressed: () => navigator.pop(true),
      ),
    ],
  );
  // Not disposed: the pop animation keeps the dialog's TextField mounted
  // past this await, and the guard's SnackBar rebuilds the overlay — a
  // disposed controller would throw. GC'd once the dialog unmounts.
  if (ok != true || content.text.trim().isEmpty) return null;
  return (kind: kind, content: content.text.trim());
}
