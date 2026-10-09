import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../api/errors_l10n.dart';
import '../session/appearance.dart';
import 'block_button.dart';
import 'block_chrome.dart';
import 'cobalt.dart';

/// The read-path trio (D4/D5): centered on the field, content constrained to
/// 480. Strings localize through the [Appearance] provider; pass [t] to use
/// the widgets outside a provider harness.
class BlockEmpty extends StatelessWidget {
  const BlockEmpty({
    super.key,
    this.title,
    this.body,
    this.icon,
    this.action,
    this.t,
  });

  final String? title;
  final String? body;
  final IconData? icon;
  final Widget? action;
  final TFn? t;

  @override
  Widget build(BuildContext context) {
    final tr = t ?? context.watch<Appearance>().t;
    final theme = Theme.of(context);
    return Center(
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: 480),
        child: Padding(
          padding: const EdgeInsets.all(Cobalt.space6),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              if (icon != null) ...[
                BlockIconWell(icon: icon!),
                const SizedBox(height: Cobalt.space4),
              ],
              Text(
                title ?? tr('state.emptyTitle'),
                textAlign: TextAlign.center,
                style: theme.textTheme.titleMedium,
              ),
              const SizedBox(height: Cobalt.space2),
              Text(
                body ?? tr('state.emptyBody'),
                textAlign: TextAlign.center,
                style: theme.textTheme.bodyMedium,
              ),
              if (action != null) ...[
                const SizedBox(height: Cobalt.space4),
                action!,
              ],
            ],
          ),
        ),
      ),
    );
  }
}

/// Square-block busy state — never a [CircularProgressIndicator].
class BlockLoading extends StatelessWidget {
  const BlockLoading({super.key, this.label, this.t});

  final String? label;
  final TFn? t;

  @override
  Widget build(BuildContext context) {
    final tr = t ?? context.watch<Appearance>().t;
    final palette = Cobalt.of(context);
    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          const BlockLoader(size: Cobalt.space6),
          const SizedBox(height: Cobalt.space3),
          Text(
            label ?? tr('common.loading'),
            style: TextStyle(
              fontFamily: Cobalt.sans,
              fontFamilyFallback: Cobalt.sansFallback,
              fontSize: 12,
              fontWeight: FontWeight.w600,
              color: palette.fgSecondary,
            ),
          ),
        ],
      ),
    );
  }
}

/// Localized failure state — [error] runs through `localizedError` into a
/// tone banner; the raw server string never reaches the page again.
class BlockError extends StatelessWidget {
  const BlockError({super.key, this.error, this.title, this.onRetry, this.t});

  final Object? error;
  final String? title;
  final VoidCallback? onRetry;
  final TFn? t;

  @override
  Widget build(BuildContext context) {
    final tr = t ?? context.watch<Appearance>().t;
    final theme = Theme.of(context);
    return Center(
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: 480),
        child: Padding(
          padding: const EdgeInsets.all(Cobalt.space6),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Text(
                title ?? tr('state.errorTitle'),
                textAlign: TextAlign.center,
                style: theme.textTheme.titleMedium,
              ),
              const SizedBox(height: Cobalt.space3),
              BlockBanner(
                message: error == null
                    ? tr('error.generic')
                    : localizedError(error!, tr),
              ),
              if (onRetry != null) ...[
                const SizedBox(height: Cobalt.space4),
                BlockButton(
                  label: tr('common.retry'),
                  icon: Icons.refresh,
                  variant: BlockButtonVariant.secondary,
                  size: BlockButtonSize.sm,
                  onPressed: onRetry,
                ),
              ],
            ],
          ),
        ),
      ),
    );
  }
}
