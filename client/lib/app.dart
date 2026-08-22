import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import 'session/appearance.dart';
import 'session/session.dart';
import 'theme/ink_seal.dart';
import 'theme/ink_seal_theme.dart';

class InsideOutApp extends StatelessWidget {
  const InsideOutApp({super.key, required this.router});

  final GoRouter router;

  @override
  Widget build(BuildContext context) {
    final appearance = context.watch<Appearance>();
    return MaterialApp.router(
      title: 'InsideOut',
      theme: inkSealTheme(InkSeal.light, brightness: Brightness.light),
      darkTheme: inkSealTheme(InkSeal.dark, brightness: Brightness.dark),
      themeMode: appearance.themeMode,
      routerConfig: router,
    );
  }
}

/// Builds the signed-in chrome: app bar + body.
class AppScaffold extends StatelessWidget {
  const AppScaffold({super.key, required this.title, required this.body, this.fab, this.actions});

  final String title;
  final Widget body;
  final Widget? fab;
  final List<Widget>? actions;

  @override
  Widget build(BuildContext context) {
    final session = context.watch<Session>();
    final appearance = context.watch<Appearance>();
    return Scaffold(
      appBar: AppBar(
        title: Text(title),
        actions: [
          ...?actions,
          IconButton(
            tooltip: appearance.t('lang.switchTo'),
            onPressed: appearance.toggleLocale,
            icon: const Icon(Icons.translate),
          ),
          IconButton(
            tooltip: appearance.t(appearance.dark ? 'theme.toggleLight' : 'theme.toggleDark'),
            onPressed: appearance.toggleTheme,
            icon: Icon(appearance.dark ? Icons.light_mode_outlined : Icons.dark_mode_outlined),
          ),
          if (session.isSignedIn)
            IconButton(
              tooltip: appearance.t('nav.profile'),
              onPressed: () => context.go('/profile'),
              icon: const Icon(Icons.person_outline),
            ),
          if (session.isSignedIn)
            IconButton(
              tooltip: appearance.t('nav.logout'),
              onPressed: () async {
                await session.logout();
                if (context.mounted) context.go('/');
              },
              icon: const Icon(Icons.logout),
            ),
        ],
      ),
      body: body,
      floatingActionButton: fab,
    );
  }
}
