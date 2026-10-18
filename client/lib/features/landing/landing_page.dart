import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../session/appearance.dart';
import '../../session/session.dart';
import '../../theme/block_button.dart';
import '../../theme/block_chrome.dart';
import '../../theme/cobalt.dart';
import '../../theme/ink_motion.dart';
import '../auth/auth_door.dart';
import '../auth/auth_form.dart';
import 'assembly_diagram.dart';
import 'assembly_step.dart';
import 'step_peek.dart';

const _steps = [
  AssemblyStepData(
    n: 1,
    titleKey: 'landing.step1Title',
    bodyKey: 'landing.step1Body',
    kind: PeekKind.idea,
  ),
  AssemblyStepData(
    n: 2,
    titleKey: 'landing.step2Title',
    bodyKey: 'landing.step2Body',
    kind: PeekKind.prd,
  ),
  AssemblyStepData(
    n: 3,
    titleKey: 'landing.step3Title',
    bodyKey: 'landing.step3Body',
    kind: PeekKind.roadmap,
  ),
  AssemblyStepData(
    n: 4,
    titleKey: 'landing.step4Title',
    bodyKey: 'landing.step4Body',
    kind: PeekKind.shipped,
  ),
];

class LandingPage extends StatefulWidget {
  const LandingPage({super.key, this.initialPrompt, this.invite = ''});

  final AuthPrompt? initialPrompt;

  /// Invitation code handed to the register prompt (deep link carry-over).
  final String invite;

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
                // Identity exception: the mark is a disc.
                Container(
                  width: 22,
                  height: 22,
                  decoration: const BoxDecoration(
                    color: Cobalt.hue500,
                    shape: BoxShape.circle,
                  ),
                ),
                const SizedBox(width: Cobalt.space2),
                Text(l10n.t('nav.brand')),
              ],
            ),
            actions: [
              Padding(
                padding: const EdgeInsets.symmetric(horizontal: Cobalt.space1),
                child: BlockIconButton(
                  icon: Icons.translate,
                  tooltip: l10n.t('lang.switchTo'),
                  semanticLabel: l10n.t('lang.switchTo'),
                  onPressed: l10n.toggleLocale,
                ),
              ),
              Padding(
                padding: const EdgeInsets.symmetric(horizontal: Cobalt.space1),
                child: BlockIconButton(
                  icon: l10n.dark
                      ? Icons.light_mode_outlined
                      : Icons.dark_mode_outlined,
                  tooltip: l10n.t(
                    l10n.dark ? 'theme.toggleLight' : 'theme.toggleDark',
                  ),
                  semanticLabel: l10n.t(
                    l10n.dark ? 'theme.toggleLight' : 'theme.toggleDark',
                  ),
                  onPressed: l10n.toggleTheme,
                ),
              ),
              const SizedBox(width: Cobalt.space2),
            ],
          ),
          body: ScrollNotificationObserver(
            child: ListView(
              padding: const EdgeInsets.only(bottom: 80),
              children: [
                Padding(
                  padding: const EdgeInsets.fromLTRB(
                    Cobalt.space6,
                    48,
                    Cobalt.space6,
                    Cobalt.space4,
                  ),
                  child: Center(
                    child: ConstrainedBox(
                      constraints: const BoxConstraints(maxWidth: 720),
                      child: Column(
                        children: [
                          InkReveal(
                            duration: const Duration(milliseconds: 700),
                            child: Text(
                              l10n.t('landing.heroTitle'),
                              style: theme.textTheme.displaySmall,
                              textAlign: TextAlign.center,
                            ),
                          ),
                          const SizedBox(height: Cobalt.space4),
                          InkReveal(
                            delay: const Duration(milliseconds: 150),
                            duration: const Duration(milliseconds: 600),
                            dy: 20,
                            child: Text(
                              l10n.t('landing.heroSubtitle'),
                              style: theme.textTheme.bodyLarge,
                              textAlign: TextAlign.center,
                            ),
                          ),
                          const SizedBox(height: Cobalt.space8),
                          InkReveal(
                            delay: const Duration(milliseconds: 300),
                            duration: const Duration(milliseconds: 600),
                            dy: 20,
                            child: Wrap(
                              alignment: WrapAlignment.center,
                              spacing: Cobalt.space3,
                              runSpacing: Cobalt.space3,
                              children: [
                                BlockButton(
                                  size: BlockButtonSize.lg,
                                  label: signedIn
                                      ? l10n.t('nav.dashboard')
                                      : l10n.t('landing.ctaPrimary'),
                                  onPressed: () {
                                    if (signedIn) {
                                      context.go('/dashboard');
                                    } else {
                                      _open(AuthPrompt.register);
                                    }
                                  },
                                ),
                                if (!signedIn)
                                  BlockButton(
                                    size: BlockButtonSize.lg,
                                    variant: BlockButtonVariant.secondary,
                                    label: l10n.t('landing.ctaSecondary'),
                                    onPressed: () => _open(AuthPrompt.login),
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
                  Center(
                    child: AssemblyStep(data: _steps[i], flip: i.isOdd),
                  ),
                Padding(
                  padding: const EdgeInsets.fromLTRB(
                    Cobalt.space6,
                    Cobalt.space8,
                    Cobalt.space6,
                    Cobalt.space4,
                  ),
                  child: Center(
                    child: ConstrainedBox(
                      constraints: const BoxConstraints(maxWidth: 560),
                      child: InkReveal(
                        whenVisible: true,
                        duration: const Duration(milliseconds: 700),
                        child: Column(
                          children: [
                            // Identity exception: the mark is a disc.
                            Container(
                              width: 56,
                              height: 56,
                              decoration: const BoxDecoration(
                                color: Cobalt.hue500,
                                shape: BoxShape.circle,
                              ),
                            ),
                            const SizedBox(height: 20),
                            Text(
                              l10n.t('landing.ctaCloseTitle'),
                              style: theme.textTheme.headlineMedium,
                              textAlign: TextAlign.center,
                            ),
                            const SizedBox(height: Cobalt.space3),
                            Text(
                              l10n.t('landing.ctaCloseBody'),
                              style: theme.textTheme.bodyLarge,
                              textAlign: TextAlign.center,
                            ),
                            const SizedBox(height: Cobalt.space6),
                            BlockButton(
                              size: BlockButtonSize.lg,
                              label: signedIn
                                  ? l10n.t('nav.dashboard')
                                  : l10n.t('landing.ctaCloseButton'),
                              onPressed: () {
                                if (signedIn) {
                                  context.go('/dashboard');
                                } else {
                                  _open(AuthPrompt.register);
                                }
                              },
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
            greeting: l10n.t(
              _prompt == AuthPrompt.register ? 'register.title' : 'login.title',
            ),
            onClose: _close,
            footer: PrivacyNoticeLink(
              label: l10n.t('privacy.link'),
              copiedMessage: l10n.t('privacy.linkCopied'),
            ),
            child: AuthForm(
              prompt: _prompt!,
              inviteCode: widget.invite,
              onSwitch: () => _open(
                _prompt == AuthPrompt.login
                    ? AuthPrompt.register
                    : AuthPrompt.login,
              ),
            ),
          ),
      ],
    );
  }
}
