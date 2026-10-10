import 'package:flutter/material.dart';

import 'cobalt.dart';

export 'block_icon_button.dart';

/// Tone channels stay in-hue. Amber is the warning channel only (the dock
/// warn-dot precedent); on light grounds it darkens to cobalt-700 because
/// pure amber cannot clear 3:1 against a pale surface.
enum BlockTone { neutral, info, success, warning, error }

/// Tone accent — banner strips, badge borders and tints. Non-text: >=3:1
/// against `surfaceSunken` in both themes (asserted in block_primitives_test).
Color blockToneAccent(Cobalt palette, Brightness brightness, BlockTone tone) {
  return switch (tone) {
    BlockTone.neutral => palette.strokeStrong,
    BlockTone.info => palette.brand,
    BlockTone.success => palette.statusSuccessFg,
    BlockTone.warning =>
      brightness == Brightness.dark ? Cobalt.warnAmber : palette.brandHover,
    BlockTone.error => palette.fgDanger,
  };
}

/// Text-safe tone ink — >=4.5:1 over the accent's 15% tint on
/// `surfaceRaised` in both themes (asserted in block_primitives_test).
Color blockToneInk(Cobalt palette, Brightness brightness, BlockTone tone) {
  return switch (tone) {
    BlockTone.neutral => palette.fgSecondary,
    BlockTone.info => palette.statusInfoFg,
    BlockTone.success => palette.statusSuccessFg,
    BlockTone.warning =>
      brightness == Brightness.dark ? Cobalt.warnAmber : palette.brandHover,
    BlockTone.error => palette.fgDanger,
  };
}

/// Inline message strip (D5): full width, sunken ground, 4px tone border.
class BlockBanner extends StatelessWidget {
  const BlockBanner({
    super.key,
    required this.message,
    this.tone = BlockTone.error,
  });

  final String message;
  final BlockTone tone;

  @override
  Widget build(BuildContext context) {
    final palette = Cobalt.of(context);
    final accent = blockToneAccent(palette, Theme.of(context).brightness, tone);
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.symmetric(
        horizontal: Cobalt.space3,
        vertical: 10,
      ),
      decoration: BoxDecoration(
        color: palette.surfaceSunken,
        border: Border(left: BorderSide(color: accent, width: 4)),
      ),
      child: Text(
        message,
        style: TextStyle(
          fontFamily: Cobalt.sans,
          fontFamilyFallback: Cobalt.sansFallback,
          fontSize: 14,
          height: 1.45,
          color: palette.fgPrimary,
        ),
      ),
    );
  }
}

/// Uppercase meta badge (D8) — 12/w600, letterSpacing 0.6, 1px border on a
/// 15% tone tint. The 1px hairline is what separates it from the 2px
/// `ShadeChip` lifecycle chip; they complement, never duplicate.
class BlockBadge extends StatelessWidget {
  const BlockBadge({
    super.key,
    required this.label,
    this.tone = BlockTone.neutral,
  });

  final String label;
  final BlockTone tone;

  @override
  Widget build(BuildContext context) {
    final palette = Cobalt.of(context);
    final brightness = Theme.of(context).brightness;
    final accent = blockToneAccent(palette, brightness, tone);
    return Container(
      padding: const EdgeInsets.symmetric(
        horizontal: Cobalt.space2,
        vertical: 2,
      ),
      decoration: BoxDecoration(
        color: accent.withValues(alpha: 0.15),
        border: Border.all(color: accent, width: Cobalt.borderDivider),
      ),
      child: Text(
        label.toUpperCase(),
        style: TextStyle(
          fontFamily: Cobalt.sans,
          fontFamilyFallback: Cobalt.sansFallback,
          fontSize: 12,
          fontWeight: FontWeight.w600,
          letterSpacing: 0.6,
          color: blockToneInk(palette, brightness, tone),
        ),
      ),
    );
  }
}

/// Uppercase micro tag (D8) — 12/w700, letterSpacing 1.0, 1px border.
class BlockTag extends StatelessWidget {
  const BlockTag({super.key, required this.label, this.color});

  final String label;
  final Color? color;

  @override
  Widget build(BuildContext context) {
    final tone = color ?? Cobalt.of(context).fgSecondary;
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
      decoration: BoxDecoration(
        border: Border.all(color: tone, width: Cobalt.borderDivider),
      ),
      child: Text(
        label.toUpperCase(),
        style: TextStyle(
          fontFamily: Cobalt.sans,
          fontFamilyFallback: Cobalt.sansFallback,
          fontSize: 12,
          fontWeight: FontWeight.w700,
          letterSpacing: 1.0,
          color: tone,
        ),
      ),
    );
  }
}

/// Square accent well (D8) — brand at 10% under a 2px brand border.
class BlockIconWell extends StatelessWidget {
  const BlockIconWell({
    super.key,
    required this.icon,
    this.size = 48,
    this.iconSize = 24,
  });

  final IconData icon;
  final double size;
  final double iconSize;

  @override
  Widget build(BuildContext context) {
    final palette = Cobalt.of(context);
    return Container(
      width: size,
      height: size,
      alignment: Alignment.center,
      decoration: BoxDecoration(
        color: palette.brand.withValues(alpha: 0.1),
        border: Border.all(color: palette.brand, width: Cobalt.borderContainer),
      ),
      child: Icon(icon, size: iconSize, color: palette.brand),
    );
  }
}

/// Square loader (D4) — the decree: loaders are blocks, never circles. A 2px
/// square outline with an inner filled square pulsing 0.25 <-> 1.0 opacity
/// over 900ms; static at full opacity when animations are disabled.
class BlockLoader extends StatefulWidget {
  const BlockLoader({super.key, this.size = 16, this.color});

  final double size;
  final Color? color;

  @override
  State<BlockLoader> createState() => _BlockLoaderState();
}

class _BlockLoaderState extends State<BlockLoader>
    with SingleTickerProviderStateMixin {
  late final AnimationController _controller = AnimationController(
    vsync: this,
    duration: const Duration(milliseconds: 900),
  );
  var _still = false;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    final disabled = MediaQuery.disableAnimationsOf(context);
    if (disabled) {
      _still = true;
      _controller
        ..stop()
        ..value = 1;
    } else if (_still || !_controller.isAnimating) {
      _still = false;
      _controller.repeat(reverse: true);
    }
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final color = widget.color ?? Cobalt.of(context).brand;
    return SizedBox(
      width: widget.size,
      height: widget.size,
      child: Stack(
        fit: StackFit.expand,
        children: [
          DecoratedBox(
            decoration: BoxDecoration(
              border: Border.all(color: color, width: Cobalt.borderContainer),
            ),
          ),
          AnimatedBuilder(
            animation: _controller,
            builder: (context, child) =>
                Opacity(opacity: 0.25 + 0.75 * _controller.value, child: child),
            child: Padding(
              padding: EdgeInsets.all(widget.size / 4),
              child: ColoredBox(color: color),
            ),
          ),
        ],
      ),
    );
  }
}
