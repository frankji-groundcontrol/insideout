import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../session/appearance.dart';
import 'block_button.dart';
import 'block_chrome.dart';
import 'cobalt.dart';
import 'cobalt_theme.dart';

/// The one modal chrome (D2) — every dialog lands here: 512 max width,
/// header with a close block, 2px divider, scrollable body, optional sunken
/// footer bar. Requires the [Appearance] provider (the app root supplies it).
Future<T?> showBlockModal<T>({
  required BuildContext context,
  required String title,
  required Widget body,
  List<Widget>? footer,
  bool barrierDismissible = true,
}) {
  return showDialog<T>(
    context: context,
    barrierColor: const Color(0xB3000000),
    barrierDismissible: barrierDismissible,
    builder: (ctx) {
      final palette = Cobalt.of(ctx);
      final closeLabel = ctx.read<Appearance>().t('common.close');
      return Dialog(
        backgroundColor: Colors.transparent,
        elevation: 0,
        insetPadding: const EdgeInsets.symmetric(
          horizontal: Cobalt.space4,
          vertical: Cobalt.space6,
        ),
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 512),
          child: Container(
            decoration: BoxDecoration(
              boxShadow: cast(Theme.of(ctx).brightness),
            ),
            child: Material(
              color: palette.surfaceRaised,
              shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(Cobalt.radiusCard),
                side: BorderSide(
                  color: palette.strokeSubtle,
                  width: Cobalt.borderContainer,
                ),
              ),
              clipBehavior: Clip.antiAlias,
              child: Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Padding(
                    padding: const EdgeInsets.all(Cobalt.space4),
                    child: Row(
                      children: [
                        Expanded(
                          child: Text(
                            title,
                            style: Theme.of(ctx).textTheme.titleMedium,
                          ),
                        ),
                        BlockIconButton(
                          icon: Icons.close,
                          tooltip: closeLabel,
                          onPressed: () => Navigator.of(ctx).pop(),
                        ),
                      ],
                    ),
                  ),
                  Divider(
                    height: Cobalt.borderContainer,
                    thickness: Cobalt.borderContainer,
                    color: palette.strokeSubtle,
                  ),
                  Flexible(
                    child: SingleChildScrollView(
                      padding: const EdgeInsets.all(Cobalt.space4),
                      child: body,
                    ),
                  ),
                  if (footer != null)
                    Container(
                      padding: const EdgeInsets.all(Cobalt.space4),
                      decoration: BoxDecoration(
                        color: palette.surfaceSunken,
                        border: Border(
                          top: BorderSide(
                            color: palette.strokeSubtle,
                            width: Cobalt.borderContainer,
                          ),
                        ),
                      ),
                      child: Row(
                        mainAxisAlignment: MainAxisAlignment.end,
                        children: [
                          for (var i = 0; i < footer.length; i++) ...[
                            if (i > 0) const SizedBox(width: Cobalt.space3),
                            footer[i],
                          ],
                        ],
                      ),
                    ),
                ],
              ),
            ),
          ),
        ),
      );
    },
  );
}

/// Confirm dialog on the shared chrome — resolves to `true` only on confirm;
/// `danger` weights the confirm button (cobalt-700, never a new hue).
Future<bool> showBlockConfirm({
  required BuildContext context,
  required String title,
  required String message,
  String? confirmText,
  String? cancelText,
  bool danger = false,
  bool barrierDismissible = true,
}) async {
  final t = context.read<Appearance>().t;
  final navigator = Navigator.of(context);
  final ok = await showBlockModal<bool>(
    context: context,
    title: title,
    barrierDismissible: barrierDismissible,
    body: Text(
      message,
      style: TextStyle(
        fontFamily: Cobalt.sans,
        fontFamilyFallback: Cobalt.sansFallback,
        fontSize: 14,
        height: 1.45,
        color: Cobalt.of(context).fgSecondary,
      ),
    ),
    footer: [
      BlockButton(
        label: cancelText ?? t('common.cancel'),
        variant: BlockButtonVariant.secondary,
        onPressed: () => navigator.pop(false),
      ),
      BlockButton(
        label: confirmText ?? t('common.confirm'),
        variant: danger
            ? BlockButtonVariant.danger
            : BlockButtonVariant.primary,
        onPressed: () => navigator.pop(true),
      ),
    ],
  );
  return ok == true;
}
