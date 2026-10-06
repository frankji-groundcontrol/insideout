import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../../theme/cobalt.dart';
import '../../theme/cobalt_theme.dart';
import '../../theme/ink_motion.dart';

enum AuthPrompt { login, register }

/// Prompt modal: cobalt mark on a raised card over a solid dim barrier.
class AuthDoor extends StatelessWidget {
  const AuthDoor({
    super.key,
    required this.greeting,
    required this.child,
    required this.onClose,
  });

  final String greeting;
  final Widget child;
  final VoidCallback onClose;

  // Geometry anchors: the wordmark must stay inside the card, not on the barrier.
  static const cardKey = Key('auth-door-card');
  static const wordmarkKey = Key('auth-door-wordmark');

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final palette = Cobalt.of(context);
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
                  child: const ColoredBox(color: Colors.black54),
                ),
              ),
              SafeArea(
                child: Center(
                  child: SingleChildScrollView(
                    padding: const EdgeInsets.all(16),
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
                                    32,
                                    40,
                                    32,
                                    32,
                                  ),
                                  child: Column(
                                    mainAxisSize: MainAxisSize.min,
                                    children: [
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
                                      const SizedBox(height: 20),
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
                                      const SizedBox(height: 8),
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
                                          style: theme.textTheme.bodySmall,
                                        ),
                                      ),
                                      const SizedBox(height: 32),
                                      child,
                                    ],
                                  ),
                                ),
                                Positioned(
                                  top: 8,
                                  right: 8,
                                  child: IconButton(
                                    tooltip: MaterialLocalizations.of(
                                      context,
                                    ).closeButtonTooltip,
                                    onPressed: onClose,
                                    icon: const Icon(Icons.close, size: 20),
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
