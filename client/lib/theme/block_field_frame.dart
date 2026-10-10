part of 'block_field.dart';

TextStyle _inputStyle(Cobalt palette) => TextStyle(
  fontFamily: Cobalt.sans,
  fontFamilyFallback: Cobalt.sansFallback,
  fontSize: 17.5,
  color: palette.fgPrimary,
);

BoxDecoration _fieldDecoration(BuildContext context, Color border) {
  final palette = Cobalt.of(context);
  return BoxDecoration(
    color: palette.surfaceRaised,
    border: Border.all(color: border, width: Cobalt.borderContainer),
    boxShadow: cast(Theme.of(context).brightness),
  );
}

/// The raised shell shared by the field trio (D3): label above at 12/w600 in
/// `fgSecondary`, 4px gap, the bordered cast box, 12px helper meta below,
/// 12px error line last. Border resolves rest `strokeSubtle` / focus `brand`
/// / error `fgDanger`.
class _FieldFrame extends StatefulWidget {
  const _FieldFrame({
    required this.box,
    this.label,
    this.helper,
    this.errorText,
    this.enabled = true,
    this.focusNode,
  });

  /// Builds the input box against the resolved border color; [node] carries
  /// focus for inputs that accept text.
  final Widget Function(BuildContext context, Color border, FocusNode node) box;
  final String? label;
  final String? helper;
  final String? errorText;
  final bool enabled;
  final FocusNode? focusNode;

  @override
  State<_FieldFrame> createState() => _FieldFrameState();
}

class _FieldFrameState extends State<_FieldFrame> {
  FocusNode? _ownNode;
  FocusNode get _node => widget.focusNode ?? (_ownNode ??= FocusNode());

  @override
  void initState() {
    super.initState();
    _node.addListener(_repaint);
  }

  @override
  void didUpdateWidget(_FieldFrame oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.focusNode != widget.focusNode) {
      (oldWidget.focusNode ?? _ownNode)?.removeListener(_repaint);
      _node.addListener(_repaint);
    }
  }

  void _repaint() => setState(() {});

  @override
  void dispose() {
    (widget.focusNode ?? _ownNode)?.removeListener(_repaint);
    _ownNode?.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final palette = Cobalt.of(context);
    final hasError = widget.errorText != null && widget.errorText!.isNotEmpty;
    final border = hasError
        ? palette.fgDanger
        : _node.hasFocus
        ? palette.brand
        : palette.strokeSubtle;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      mainAxisSize: MainAxisSize.min,
      children: [
        if (widget.label != null) ...[
          Text(
            widget.label!,
            style: TextStyle(
              fontFamily: Cobalt.sans,
              fontFamilyFallback: Cobalt.sansFallback,
              fontSize: 12,
              fontWeight: FontWeight.w600,
              color: palette.fgSecondary,
            ),
          ),
          const SizedBox(height: Cobalt.space1),
        ],
        Opacity(
          opacity: widget.enabled ? 1 : 0.5,
          child: Semantics(
            label: widget.label,
            child: widget.box(context, border, _node),
          ),
        ),
        if (widget.helper != null) ...[
          const SizedBox(height: Cobalt.space1),
          Text(
            widget.helper!,
            style: TextStyle(
              fontFamily: Cobalt.sans,
              fontFamilyFallback: Cobalt.sansFallback,
              fontSize: 12,
              color: palette.fgMuted,
            ),
          ),
        ],
        if (hasError) ...[
          const SizedBox(height: Cobalt.space1),
          Text(
            widget.errorText!,
            style: TextStyle(
              fontFamily: Cobalt.sans,
              fontFamilyFallback: Cobalt.sansFallback,
              fontSize: 12,
              color: palette.fgDanger,
            ),
          ),
        ],
      ],
    );
  }
}
