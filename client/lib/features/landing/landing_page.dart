import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../session/appearance.dart';
import '../../session/session.dart';
import '../../theme/ink_motion.dart';
import '../auth/auth_door.dart';
import '../auth/auth_form.dart';
import 'assembly_diagram.dart';
import 'assembly_step.dart';
import 'step_peek.dart';

const _steps = [
  AssemblyStepData(
    n: 1,
    sealAsset: 'assets/seals/luomo.webp',
    titleKey: 'landing.step1Title',
    bodyKey: 'landing.step1Body',
    kind: PeekKind.idea,
  ),
  AssemblyStepData(
    n: 2,
    sealAsset: 'assets/seals/chengwen.webp',
    titleKey: 'landing.step2Title',
    bodyKey: 'landing.step2Body',
    kind: PeekKind.prd,
  ),
  AssemblyStepData(
    n: 3,
    sealAsset: 'assets/seals/fenzhi.webp',
    titleKey: 'landing.step3Title',
    bodyKey: 'landing.step3Body',
    kind: PeekKind.roadmap,
  ),
  AssemblyStepData(
    n: 4,
    sealAsset: 'assets/seals/gaiyin.webp',
    titleKey: 'landing.step4Title',
    bodyKey: 'landing.step4Body',
    kind: PeekKind.shipped,
  ),
];

class LandingPage extends StatefulWidget {
  const LandingPage({super.key, this.initialPrompt});

  final AuthPrompt? initialPrompt;

  @override
  State<LandingPage> createState() => _LandingPageState();
}

class _LandingPageState extends State<LandingPage> {
  late AuthPrompt? _prompt = widget.initialPrompt;

  void _open(AuthPrompt prompt) => setState(() => _prompt = prompt);

  void _close() {
    setState(() => _prompt = null);
    final router = GoRouter.maybeOf(context);
    final path = router?.routerDelegate.currentConfiguration.uri.path;
    if (path == '/login' || path == '/register') router?.go('/');
  }

  @override
  Widget build(BuildContext context) {
    final signedIn = context.watch<Session>().isSignedIn;
    final l10n = context.watch<Appearance>();
    final theme = Theme.of(context);
    return Stack(
      children: [
        Scaffold(
          appBar: AppBar(
            title: Row(
              children: [
                Image.asset('assets/seals/yin.webp', width: 28, height: 28),
                const SizedBox(width: 10),
                Text(l10n.t('nav.brand')),
              ],
            ),
            actions: [
              IconButton(
                tooltip: l10n.t('lang.switchTo'),
                onPressed: l10n.toggleLocale,
                icon: const Icon(Icons.translate),
              ),
              IconButton(
                tooltip: l10n.t(l10n.dark ? 'theme.toggleLight' : 'theme.toggleDark'),
                onPressed: l10n.toggleTheme,
                icon: Icon(l10n.dark ? Icons.light_mode_outlined : Icons.dark_mode_outlined),
              ),
            ],
          ),
          body: ScrollNotificationObserver(
            child: ListView(
              padding: const EdgeInsets.only(bottom: 80),
              children: [
                Padding(
                  padding: const EdgeInsets.fromLTRB(24, 48, 24, 16),
                  child: Center(
                    child: ConstrainedBox(
                      constraints: const BoxConstraints(maxWidth: 720),
                      child: Column(
                        children: [
                          InkReveal(
                            duration: const Duration(milliseconds: 700),
                            child: Text(l10n.t('landing.heroTitle'), style: theme.textTheme.displaySmall, textAlign: TextAlign.center),
                          ),
                          const SizedBox(height: 16),
                          InkReveal(
                            delay: const Duration(milliseconds: 150),
                            duration: const Duration(milliseconds: 600),
                            dy: 20,
                            child: Text(l10n.t('landing.heroSubtitle'), style: theme.textTheme.bodyLarge, textAlign: TextAlign.center),
                          ),
                          const SizedBox(height: 32),
                          InkReveal(
                            delay: const Duration(milliseconds: 300),
                            duration: const Duration(milliseconds: 600),
                            dy: 20,
                            child: Wrap(
                              alignment: WrapAlignment.center,
                              spacing: 12,
                              runSpacing: 12,
                              children: [
                                FilledButton(
                                  onPressed: () {
                                    if (signedIn) {
                                      context.go('/dashboard');
                                    } else {
                                      _open(AuthPrompt.register);
                                    }
                                  },
                                  child: Text(signedIn ? l10n.t('nav.dashboard') : l10n.t('landing.ctaPrimary')),
                                ),
                                if (!signedIn)
                                  OutlinedButton(
                                    onPressed: () => _open(AuthPrompt.login),
                                    child: Text(l10n.t('landing.ctaSecondary')),
                                  ),
                              ],
                            ),
                          ),
                          const SizedBox(height: 48),
                          AssemblyDiagram(
                            mode: AssemblyDiagramMode.hero,
                            semanticsLabel: l10n.t('landing.diagramAlt'),
                          ),
                        ],
                      ),
                    ),
                  ),
                ),
                for (var i = 0; i < _steps.length; i++)
                  Center(child: AssemblyStep(data: _steps[i], flip: i.isOdd)),
                Padding(
                  padding: const EdgeInsets.fromLTRB(24, 32, 24, 16),
                  child: Center(
                    child: ConstrainedBox(
                      constraints: const BoxConstraints(maxWidth: 560),
                      child: InkReveal(
                        whenVisible: true,
                        duration: const Duration(milliseconds: 700),
                        child: Column(
                          children: [
                            InkStamp(child: Image.asset('assets/seals/yin.webp', width: 56, height: 56)),
                            const SizedBox(height: 20),
                            Text(l10n.t('landing.ctaCloseTitle'), style: theme.textTheme.headlineMedium, textAlign: TextAlign.center),
                            const SizedBox(height: 12),
                            Text(l10n.t('landing.ctaCloseBody'), style: theme.textTheme.bodyLarge, textAlign: TextAlign.center),
                            const SizedBox(height: 24),
                            FilledButton(
                              onPressed: () {
                                if (signedIn) {
                                  context.go('/dashboard');
                                } else {
                                  _open(AuthPrompt.register);
                                }
                              },
                              child: Text(signedIn ? l10n.t('nav.dashboard') : l10n.t('landing.ctaCloseButton')),
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
        if (_prompt != null)
          AuthDoor(
            greeting: l10n.t(_prompt == AuthPrompt.register ? 'register.title' : 'login.title'),
            onClose: _close,
            child: AuthForm(
              prompt: _prompt!,
              onSwitch: () => _open(_prompt == AuthPrompt.login ? AuthPrompt.register : AuthPrompt.login),
            ),
          ),
      ],
    );
  }
}
