import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../session/appearance.dart';
import 'block_button.dart';
import 'block_field.dart';
import 'block_modal.dart';
import 'cobalt.dart';

/// One field spec for [showBlockForm].
class BlockFormField {
  const BlockFormField({
    required this.label,
    this.initial = '',
    this.multiline = false,
  });

  final String label;
  final String initial;
  final bool multiline;
}

/// Text prompt on the shared modal chrome (D2/D3): label-above raised fields,
/// sunken footer with cancel + confirm. Resolves to the field values, or null
/// on cancel/dismiss. The form state owns its controllers and disposes them
/// when the dialog route unmounts, so callers receive plain strings and no
/// controller outlives (or is disposed under) the pop animation.
///
/// No autofocus on purpose: a blinking caret never lets the dialog-opening
/// `pumpAndSettle` in widget tests settle.
Future<List<String>?> showBlockForm({
  required BuildContext context,
  required String title,
  required String confirmLabel,
  required List<BlockFormField> fields,
}) {
  final formKey = GlobalKey<_BlockFormState>();
  final t = context.read<Appearance>().t;
  final navigator = Navigator.of(context);
  return showBlockModal<List<String>>(
    context: context,
    title: title,
    body: _BlockForm(key: formKey, fields: fields),
    footer: [
      BlockButton(
        label: t('common.cancel'),
        variant: BlockButtonVariant.secondary,
        onPressed: () => navigator.pop(),
      ),
      BlockButton(
        label: confirmLabel,
        onPressed: () {
          final values = formKey.currentState?.values;
          if (values != null) navigator.pop(values);
        },
      ),
    ],
  );
}

class _BlockForm extends StatefulWidget {
  const _BlockForm({super.key, required this.fields});

  final List<BlockFormField> fields;

  @override
  State<_BlockForm> createState() => _BlockFormState();
}

class _BlockFormState extends State<_BlockForm> {
  late final List<TextEditingController> _controllers = [
    for (final field in widget.fields)
      TextEditingController(text: field.initial),
  ];

  List<String> get values => [for (final c in _controllers) c.text];

  @override
  void dispose() {
    for (final c in _controllers) {
      c.dispose();
    }
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Column(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        for (var i = 0; i < widget.fields.length; i++) ...[
          if (i > 0) const SizedBox(height: Cobalt.space4),
          if (widget.fields[i].multiline)
            BlockTextarea(
              label: widget.fields[i].label,
              controller: _controllers[i],
            )
          else
            BlockField(
              label: widget.fields[i].label,
              controller: _controllers[i],
            ),
        ],
      ],
    );
  }
}
