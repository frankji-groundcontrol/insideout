import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../../theme/ink_motion.dart';
import '../../theme/ink_seal.dart';

enum AuthPrompt { login, register }

/// Prompt overlay: 印 stamp + paper panel over a translucent celadon scrim.
/// Lands on top of the current page instead of replacing it.
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

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
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
                  child: DecoratedBox(
                    decoration: BoxDecoration(
                      gradient: RadialGradient(
                        center: const Alignment(0, -0.35),
                        radius: 1.15,
                        colors: [
                          scheme.secondary.withValues(alpha: 0.08),
                          theme.scaffoldBackgroundColor.withValues(alpha: 0.88),
                        ],
                      ),
                    ),
                  ),
                ),
              ),
              SafeArea(
                child: Center(
                  child: SingleChildScrollView(
                    padding: const EdgeInsets.all(16),
                    child: ConstrainedBox(
                      constraints: const BoxConstraints(maxWidth: 384),
                      child: Column(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          InkStamp(
                            child: Image.asset(
                              'assets/seals/yin.webp',
                              width: 80,
                              height: 80,
                              filterQuality: FilterQuality.medium,
                            ),
                          ),
                          const SizedBox(height: 20),
                          InkReveal(
                            delay: const Duration(milliseconds: 300),
                            duration: const Duration(milliseconds: 550),
                            dy: 16,
                            child: Text('InsideOut', style: theme.textTheme.headlineSmall),
                          ),
                          const SizedBox(height: 8),
                          InkReveal(
                            delay: const Duration(milliseconds: 400),
                            duration: const Duration(milliseconds: 500),
                            dy: 12,
                            child: Text(greeting, style: theme.textTheme.bodySmall),
                          ),
                          const SizedBox(height: 32),
                          InkReveal(
                            delay: const Duration(milliseconds: 350),
                            duration: const Duration(milliseconds: 500),
                            dy: 24,
                            scaleFrom: 0.98,
                            child: Material(
                              color: scheme.surface,
                              shape: RoundedRectangleBorder(
                                borderRadius: BorderRadius.circular(InkSeal.radiusHero),
                                side: BorderSide(color: scheme.outline),
                              ),
                              child: Stack(
                                children: [
                                  Padding(
                                    padding: const EdgeInsets.fromLTRB(32, 40, 32, 32),
                                    child: child,
                                  ),
                                  Positioned(
                                    top: 8,
                                    right: 8,
                                    child: IconButton(
                                      tooltip: MaterialLocalizations.of(context).closeButtonTooltip,
                                      onPressed: onClose,
                                      icon: const Icon(Icons.close, size: 20),
                                    ),
                                  ),
                                ],
                              ),
                            ),
                          ),
                        ],
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
