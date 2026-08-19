import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../session/appearance.dart';
import '../../session/session.dart';

class LandingPage extends StatelessWidget {
  const LandingPage({super.key});

  @override
  Widget build(BuildContext context) {
    final signedIn = context.watch<Session>().isSignedIn;
    final l10n = context.watch<Appearance>();
    return Scaffold(
      appBar: AppBar(
        title: Text(l10n.t('nav.brand')),
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
      body: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 560),
          child: Padding(
            padding: const EdgeInsets.all(24),
            child: Column(
              mainAxisAlignment: MainAxisAlignment.center,
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(l10n.t('landing.heroTitle'), style: Theme.of(context).textTheme.displaySmall),
                const SizedBox(height: 12),
                Text(l10n.t('landing.heroSubtitle'), style: Theme.of(context).textTheme.bodyLarge),
                const SizedBox(height: 32),
                Wrap(
                  spacing: 12,
                  runSpacing: 12,
                  children: [
                    FilledButton(
                      onPressed: () => context.go(signedIn ? '/dashboard' : '/register'),
                      child: Text(signedIn ? l10n.t('nav.dashboard') : l10n.t('landing.ctaPrimary')),
                    ),
                    if (!signedIn)
                      OutlinedButton(
                        onPressed: () => context.go('/login'),
                        child: Text(l10n.t('landing.ctaSecondary')),
                      ),
                  ],
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
