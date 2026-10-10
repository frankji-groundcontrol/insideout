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
    this.required = false,
  });

  final String label;
  final String initial;
  final bool multiline;

  /// Confirm refuses to close while this field is blank after trim: the
  /// write would fail server-side anyway, and a dialog that says so beats
  /// one that closes on a lost tap.
  final bool required;
}

/// Text prompt on the shared modal chrome (D2/D3): label-above raised fields,
/// sunken footer with cancel + confirm. Resolves to the field values, or null
/// on cancel/dismiss. A field marked [BlockFormField.required] keeps the
/// dialog open on confirm, saying why, until it is filled. The form state
/// owns its controllers and disposes them when the dialog route unmounts, so
/// callers receive plain strings and no
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
          final form = formKey.currentState;
          if (form != null && form.validate()) navigator.pop(form.values);
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

  late final List<bool> _missing = List.filled(widget.fields.length, false);

  List<String> get values => [for (final c in _controllers) c.text];

  /// False, with each required field saying why, when one of them is blank
  /// after trim; the confirm button asks before it closes the dialog.
  bool validate() {
    var ok = true;
    for (var i = 0; i < widget.fields.length; i++) {
      _missing[i] =
          widget.fields[i].required && _controllers[i].text.trim().isEmpty;
      if (_missing[i]) ok = false;
    }
    setState(() {});
    return ok;
  }

  void _typed(int i) {
    if (!_missing[i]) return;
    setState(() => _missing[i] = false);
  }

  @override
  void dispose() {
    for (final c in _controllers) {
      c.dispose();
    }
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final t = context.watch<Appearance>().t;
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
              errorText: _missing[i] ? t('common.fieldRequired') : null,
              onChanged: (_) => _typed(i),
            )
          else
            BlockField(
              label: widget.fields[i].label,
              controller: _controllers[i],
              errorText: _missing[i] ? t('common.fieldRequired') : null,
              onChanged: (_) => _typed(i),
            ),
        ],
      ],
    );
  }
}
