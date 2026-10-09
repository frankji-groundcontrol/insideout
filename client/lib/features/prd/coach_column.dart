import 'package:flutter/material.dart';

import '../../theme/block_chrome.dart';
import '../../theme/cobalt.dart';
import '../../theme/cobalt_theme.dart';
import '../../widgets/pressable_block.dart';

/// The Coach column is the one place the serif face is allowed, and it sits on
/// a slab that is dark in BOTH themes — `hue950` on the light ground, the
/// neutral `surfaceSunken` in dark. These three helpers keep that slab and its
/// ink resolved in one place so the sans never leaks in.
bool coachIsDark(BuildContext context) =>
    Theme.of(context).brightness == Brightness.dark;

Color coachSlab(BuildContext context) =>
    coachIsDark(context) ? Cobalt.of(context).surfaceSunken : Cobalt.hue950;

Color coachInk(BuildContext context) =>
    coachIsDark(context) ? Cobalt.of(context).fgPrimary : Cobalt.darkType;

/// Large pull-quote — the Coach's signed-off voice. Serif, 26, w600, 1.35.
class CoachQuote extends StatelessWidget {
  const CoachQuote({super.key, required this.text, required this.role});

  final String text;
  final String role;

  @override
  Widget build(BuildContext context) {
    final palette = Cobalt.of(context);
    final ink = coachInk(context);
    return Padding(
      padding: const EdgeInsets.only(bottom: Cobalt.space6),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            '“$text”',
            style: cobaltSerif(
              palette,
              size: 26,
              color: ink,
              weight: FontWeight.w600,
              height: 1.35,
            ),
          ),
          const SizedBox(height: Cobalt.space2),
          Text(
            role,
            style: cobaltSerif(
              palette,
              size: 12,
              color: ink.withValues(alpha: 0.6),
              weight: FontWeight.w600,
            ),
          ),
        ],
      ),
    );
  }
}

/// A suggested prompt on the block grammar — a raised block that casts and
/// presses, replacing the Material `ActionChip` ripple.
class CoachPrompt extends StatelessWidget {
  const CoachPrompt({super.key, required this.label, this.onTap});

  final String label;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    final palette = Cobalt.of(context);
    final chip = Container(
      padding: const EdgeInsets.symmetric(
        horizontal: Cobalt.space3,
        vertical: Cobalt.space2,
      ),
      decoration: BoxDecoration(
        color: palette.surfaceRaised,
        border: Border.all(
          color: palette.strokeSubtle,
          width: Cobalt.borderContainer,
        ),
      ),
      child: Text(
        label,
        style: cobaltSerif(palette, size: 14, weight: FontWeight.w600),
      ),
    );
    return Opacity(
      opacity: onTap == null ? 0.5 : 1,
      child: PressableBlock(onTap: onTap, child: chip),
    );
  }
}

/// Tone strip for the Coach column. Same grammar as `BlockBanner` — full
/// width, 4px tone edge, 14px text — but the ground is the slab itself and the
/// accent is chosen to read on it, which `BlockBanner` cannot do.
class CoachNotice extends StatelessWidget {
  const CoachNotice({super.key, required this.message, required this.accent});

  final String message;
  final Color accent;

  @override
  Widget build(BuildContext context) {
    final palette = Cobalt.of(context);
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.symmetric(
        horizontal: Cobalt.space3,
        vertical: Cobalt.space2,
      ),
      decoration: BoxDecoration(
        border: Border(left: BorderSide(color: accent, width: 4)),
      ),
      child: Text(
        message,
        style: cobaltSerif(palette, size: 14, color: coachInk(context)),
      ),
    );
  }
}

/// Raised composer block: serif input on a cast field, focus edge in cobalt,
/// square send block beside it.
class CoachComposer extends StatefulWidget {
  const CoachComposer({
    super.key,
    required this.controller,
    required this.enabled,
    required this.hint,
    required this.onSend,
    required this.sendTooltip,
  });

  final TextEditingController controller;
  final bool enabled;
  final String hint;
  final VoidCallback? onSend;
  final String sendTooltip;

  @override
  State<CoachComposer> createState() => _CoachComposerState();
}

class _CoachComposerState extends State<CoachComposer> {
  final node = FocusNode();

  @override
  void initState() {
    super.initState();
    node.addListener(_repaint);
  }

  void _repaint() {
    if (mounted) setState(() {});
  }

  @override
  void dispose() {
    node
      ..removeListener(_repaint)
      ..dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final palette = Cobalt.of(context);
    final style = cobaltSerif(palette, size: 17.5, weight: FontWeight.w400);
    return Row(
      crossAxisAlignment: CrossAxisAlignment.end,
      children: [
        Expanded(
          child: Opacity(
            opacity: widget.enabled ? 1 : 0.5,
            child: Container(
              padding: const EdgeInsets.symmetric(
                horizontal: Cobalt.space3,
                vertical: Cobalt.space2,
              ),
              decoration: BoxDecoration(
                color: palette.surfaceRaised,
                border: Border.all(
                  color: node.hasFocus ? palette.brand : palette.strokeSubtle,
                  width: Cobalt.borderContainer,
                ),
                boxShadow: cast(Theme.of(context).brightness),
              ),
              child: TextField(
                controller: widget.controller,
                focusNode: node,
                enabled: widget.enabled,
                minLines: 1,
                maxLines: 4,
                style: style,
                decoration: InputDecoration(
                  hintText: widget.hint,
                  hintStyle: style.copyWith(color: palette.fgMuted),
                  isCollapsed: true,
                  filled: false,
                  border: InputBorder.none,
                  enabledBorder: InputBorder.none,
                  focusedBorder: InputBorder.none,
                  disabledBorder: InputBorder.none,
                  contentPadding: EdgeInsets.zero,
                ),
              ),
            ),
          ),
        ),
        const SizedBox(width: Cobalt.space2),
        BlockIconButton(
          icon: Icons.send,
          tooltip: widget.sendTooltip,
          onPressed: widget.onSend,
        ),
      ],
    );
  }
}
