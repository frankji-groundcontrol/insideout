import 'package:flutter/material.dart';

import 'cobalt.dart';
import 'cobalt_theme.dart';

part 'block_field_frame.dart';

/// Single-line raised field — fixed 44 (dense 40), label above, no Material
/// floating label.
class BlockField extends StatelessWidget {
  const BlockField({
    super.key,
    this.label,
    this.controller,
    this.focusNode,
    this.hint,
    this.helper,
    this.errorText,
    this.enabled = true,
    this.obscure = false,
    this.keyboardType,
    this.textInputAction,
    this.autofillHints,
    this.autofocus = false,
    this.maxLength,
    this.onChanged,
    this.onSubmitted,
    this.dense = false,
  });

  final String? label;
  final TextEditingController? controller;
  final FocusNode? focusNode;
  final String? hint;
  final String? helper;
  final String? errorText;
  final bool enabled;
  final bool obscure;
  final TextInputType? keyboardType;
  final TextInputAction? textInputAction;
  final Iterable<String>? autofillHints;
  final bool autofocus;
  final int? maxLength;
  final ValueChanged<String>? onChanged;
  final ValueChanged<String>? onSubmitted;
  final bool dense;

  @override
  Widget build(BuildContext context) {
    final palette = Cobalt.of(context);
    return _FieldFrame(
      label: label,
      helper: helper,
      errorText: errorText,
      enabled: enabled,
      focusNode: focusNode,
      box: (context, border, node) => Container(
        height: dense ? Cobalt.controlHeightDense : Cobalt.controlHeight,
        alignment: Alignment.center,
        decoration: _fieldDecoration(context, border),
        child: TextField(
          controller: controller,
          focusNode: node,
          enabled: enabled,
          obscureText: obscure,
          keyboardType: keyboardType,
          textInputAction: textInputAction,
          autofillHints: autofillHints,
          autofocus: autofocus,
          maxLength: maxLength,
          onChanged: onChanged,
          onSubmitted: onSubmitted,
          style: _inputStyle(palette),
          decoration: InputDecoration(
            hintText: hint,
            hintStyle: _inputStyle(palette).copyWith(color: palette.fgMuted),
            counterText: '',
            filled: false,
            isDense: true,
            border: InputBorder.none,
            enabledBorder: InputBorder.none,
            focusedBorder: InputBorder.none,
            disabledBorder: InputBorder.none,
            contentPadding: const EdgeInsets.symmetric(
              horizontal: Cobalt.space3,
            ),
          ),
        ),
      ),
    );
  }
}

/// Multi-line raised field — grows by lines instead of a fixed height.
class BlockTextarea extends StatelessWidget {
  const BlockTextarea({
    super.key,
    this.label,
    this.controller,
    this.focusNode,
    this.hint,
    this.errorText,
    this.enabled = true,
    this.minLines = 2,
    this.maxLines = 6,
    this.onChanged,
  });

  final String? label;
  final TextEditingController? controller;
  final FocusNode? focusNode;
  final String? hint;
  final String? errorText;
  final bool enabled;
  final int minLines;
  final int maxLines;
  final ValueChanged<String>? onChanged;

  @override
  Widget build(BuildContext context) {
    final palette = Cobalt.of(context);
    return _FieldFrame(
      label: label,
      errorText: errorText,
      enabled: enabled,
      focusNode: focusNode,
      box: (context, border, node) => Container(
        decoration: _fieldDecoration(context, border),
        child: TextField(
          controller: controller,
          focusNode: node,
          enabled: enabled,
          minLines: minLines,
          maxLines: maxLines,
          onChanged: onChanged,
          style: _inputStyle(palette),
          decoration: InputDecoration(
            hintText: hint,
            hintStyle: _inputStyle(palette).copyWith(color: palette.fgMuted),
            counterText: '',
            filled: false,
            isDense: true,
            border: InputBorder.none,
            enabledBorder: InputBorder.none,
            focusedBorder: InputBorder.none,
            disabledBorder: InputBorder.none,
            contentPadding: const EdgeInsets.symmetric(
              horizontal: Cobalt.space3,
              vertical: Cobalt.space2,
            ),
          ),
        ),
      ),
    );
  }
}

/// Raised dropdown — fixed 44 (dense 40), same shell grammar as [BlockField].
class BlockSelect<T> extends StatelessWidget {
  const BlockSelect({
    super.key,
    required this.value,
    required this.options,
    this.label,
    this.placeholder,
    this.errorText,
    this.onChanged,
    this.enabled = true,
    this.dense = false,
  });

  final T? value;
  final List<({T value, String label})> options;
  final String? label;
  final String? placeholder;
  final String? errorText;
  final ValueChanged<T?>? onChanged;
  final bool enabled;
  final bool dense;

  @override
  Widget build(BuildContext context) {
    final palette = Cobalt.of(context);
    return _FieldFrame(
      label: label,
      errorText: errorText,
      enabled: enabled,
      box: (context, border, _) => MouseRegion(
        cursor: enabled ? SystemMouseCursors.click : SystemMouseCursors.basic,
        child: Container(
          height: dense ? Cobalt.controlHeightDense : Cobalt.controlHeight,
          padding: const EdgeInsets.symmetric(horizontal: Cobalt.space3),
          alignment: Alignment.center,
          decoration: _fieldDecoration(context, border),
          child: DropdownButtonHideUnderline(
            child: DropdownButton<T>(
              value: options.any((o) => o.value == value) ? value : null,
              isExpanded: true,
              isDense: true,
              borderRadius: BorderRadius.circular(Cobalt.radiusControl),
              hint: placeholder == null
                  ? null
                  : Text(
                      placeholder!,
                      style: _inputStyle(
                        palette,
                      ).copyWith(color: palette.fgMuted),
                    ),
              dropdownColor: palette.surfaceRaised,
              iconEnabledColor: palette.fgSecondary,
              style: _inputStyle(palette),
              items: [
                for (final option in options)
                  DropdownMenuItem<T>(
                    value: option.value,
                    child: Text(option.label, overflow: TextOverflow.ellipsis),
                  ),
              ],
              onChanged: enabled ? onChanged : null,
            ),
          ),
        ),
      ),
    );
  }
}
