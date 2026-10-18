import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../../api/open_url_stub.dart'
    if (dart.library.ui_web) '../../api/open_url_web.dart';
import '../../theme/block_button.dart';
import '../../theme/block_chrome.dart';
import '../../theme/cobalt.dart';
import '../../theme/cobalt_theme.dart';
import '../../theme/ink_motion.dart';

enum AuthPrompt { login, register }

/// The privacy notice the auth door links to (rv4-PUB-02). The web build does
/// not bundle PRIVACY.md, so the link goes to the copy on the project's public
/// repository; a deployment that hosts its own notice changes this one line.
const privacyNoticeUrl =
    'https://github.com/frankji-groundcontrol/insideout/blob/main/PRIVACY.md';

/// Prompt modal: cobalt mark on a raised card over a solid dim barrier.
class AuthDoor extends StatelessWidget {
  const AuthDoor({
    super.key,
    required this.greeting,
    required this.child,
    required this.onClose,
    this.footer,
  });

  final String greeting;
  final Widget child;
  final VoidCallback onClose;

  /// Sits under [child] inside the card (the sign-in and sign-up screens put
  /// the privacy link here); the other doors leave it out.
  final Widget? footer;

  // Geometry anchors: the wordmark must stay inside the card, not on the barrier.
  static const cardKey = Key('auth-door-card');
  static const wordmarkKey = Key('auth-door-wordmark');

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final palette = Cobalt.of(context);
    final closeLabel = MaterialLocalizations.of(context).closeButtonLabel;
    return Material(
      color: Colors.transparent,
      child: CallbackShortcuts(
        bindings: {const SingleActivator(LogicalKeyboardKey.escape): onClose},
        child: Focus(
          autofocus: true,
          child: Stack(
            fit: StackFit.expand,
            children: [
              Positioned.fill(
                child: GestureDetector(
                  onTap: onClose,
                  // modalBarrier from cobalt_theme — the same dim every
                  // showBlockModal dialog sits on.
                  child: const ColoredBox(color: modalBarrier),
                ),
              ),
              SafeArea(
                child: Center(
                  child: SingleChildScrollView(
                    padding: const EdgeInsets.all(Cobalt.space4),
                    child: ConstrainedBox(
                      constraints: const BoxConstraints(maxWidth: 384),
                      child: InkReveal(
                        delay: const Duration(milliseconds: 350),
                        duration: const Duration(milliseconds: 500),
                        dy: 24,
                        scaleFrom: 0.98,
                        child: Container(
                          decoration: BoxDecoration(
                            boxShadow: cast(theme.brightness),
                          ),
                          child: Material(
                            key: cardKey,
                            color: palette.surfaceRaised,
                            shape: RoundedRectangleBorder(
                              borderRadius: BorderRadius.circular(
                                Cobalt.radiusHero,
                              ),
                              side: BorderSide(
                                color: palette.strokeSubtle,
                                width: Cobalt.borderContainer,
                              ),
                            ),
                            child: Stack(
                              children: [
                                Padding(
                                  padding: const EdgeInsets.fromLTRB(
                                    Cobalt.space8,
                                    Cobalt.space10,
                                    Cobalt.space8,
                                    Cobalt.space8,
                                  ),
                                  child: Column(
                                    mainAxisSize: MainAxisSize.min,
                                    children: [
                                      // Identity exception: the mark is a disc.
                                      InkReveal(
                                        child: Container(
                                          width: 56,
                                          height: 56,
                                          decoration: const BoxDecoration(
                                            color: Cobalt.hue500,
                                            shape: BoxShape.circle,
                                          ),
                                        ),
                                      ),
                                      const SizedBox(height: Cobalt.space4),
                                      InkReveal(
                                        delay: const Duration(
                                          milliseconds: 300,
                                        ),
                                        duration: const Duration(
                                          milliseconds: 550,
                                        ),
                                        dy: 16,
                                        child: Text(
                                          'InsideOut',
                                          key: wordmarkKey,
                                          style: theme.textTheme.headlineSmall,
                                        ),
                                      ),
                                      const SizedBox(height: Cobalt.space2),
                                      InkReveal(
                                        delay: const Duration(
                                          milliseconds: 400,
                                        ),
                                        duration: const Duration(
                                          milliseconds: 500,
                                        ),
                                        dy: 12,
                                        child: Text(
                                          greeting,
                                          textAlign: TextAlign.center,
                                          style: theme.textTheme.bodySmall,
                                        ),
                                      ),
                                      const SizedBox(height: Cobalt.space8),
                                      child,
                                      if (footer != null) ...[
                                        const SizedBox(height: Cobalt.space4),
                                        footer!,
                                      ],
                                    ],
                                  ),
                                ),
                                Positioned(
                                  top: Cobalt.space2,
                                  right: Cobalt.space2,
                                  child: BlockIconButton(
                                    icon: Icons.close,
                                    tooltip: closeLabel,
                                    semanticLabel: closeLabel,
                                    onPressed: onClose,
                                  ),
                                ),
                              ],
                            ),
                          ),
                        ),
                      ),
                    ),
                  ),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

/// The link to [privacyNoticeUrl]. In a browser it opens the notice in a new
/// tab. With no browser to hand it to (a test, a desktop build) it copies the
/// address and says so in place of the label: a snackbar would land on the
/// page under this door's dim barrier, where nobody reads it.
class PrivacyNoticeLink extends StatefulWidget {
  const PrivacyNoticeLink({
    super.key,
    required this.label,
    required this.copiedMessage,
  });

  final String label;
  final String copiedMessage;

  @override
  State<PrivacyNoticeLink> createState() => _PrivacyNoticeLinkState();
}

class _PrivacyNoticeLinkState extends State<PrivacyNoticeLink> {
  var _copied = false;

  Future<void> _open() async {
    if (openExternalUrl(privacyNoticeUrl)) return;
    await Clipboard.setData(const ClipboardData(text: privacyNoticeUrl));
    if (mounted) setState(() => _copied = true);
  }

  @override
  Widget build(BuildContext context) => BlockButton(
    label: _copied ? widget.copiedMessage : widget.label,
    variant: BlockButtonVariant.ghost,
    size: BlockButtonSize.sm,
    expand: true,
    trailingIcon: _copied ? Icons.check : Icons.open_in_new,
    onPressed: _open,
  );
}
