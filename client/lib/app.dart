import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import 'chrome/loop_focus.dart';
import 'chrome/product_dock.dart';
import 'session/appearance.dart';
import 'session/boot.dart';
import 'session/session.dart';
import 'theme/cobalt.dart';
import 'theme/cobalt_theme.dart';

class InsideOutApp extends StatelessWidget {
  const InsideOutApp({super.key, required this.router});

  final GoRouter router;

  @override
  Widget build(BuildContext context) {
    final appearance = context.watch<Appearance>();
    final boot = context.watch<BootController>();
    final light = cobaltTheme(Cobalt.light, brightness: Brightness.light);
    final dark = cobaltTheme(Cobalt.dark, brightness: Brightness.dark);
    // Until boot settles, only the loading page mounts — no product
    // surface builds before the web fonts are ready, preserving the
    // first-frame tofu fix.
    if (!boot.done) {
      return MaterialApp(
        title: 'InsideOut',
        theme: light,
        darkTheme: dark,
        themeMode: appearance.themeMode,
        home: const BootLoadingPage(),
      );
    }
    return MaterialApp.router(
      title: 'InsideOut',
      theme: light,
      darkTheme: dark,
      themeMode: appearance.themeMode,
      routerConfig: router,
    );
  }
}

/// Signed-in product chrome: page-owned titles, top-right utilities, bottom dock.
class AppScaffold extends StatelessWidget {
  const AppScaffold({
    super.key,
    required this.body,
    this.title,
    this.fab,
    this.actions,
    this.paintTitle = false,
  });

  final String? title;
  final Widget body;
  final Widget? fab;
  final List<Widget>? actions;
  final bool paintTitle;

  @override
  Widget build(BuildContext context) {
    final session = context.watch<Session>();
    final appearance = context.watch<Appearance>();
    final signedIn = session.isSignedIn;
    final palette = Cobalt.of(context);
    return Scaffold(
      backgroundColor: palette.surfaceBase,
      extendBody: true,
      body: Stack(
        children: [
          Column(
            children: [
              if (paintTitle && title != null)
                SafeArea(
                  bottom: false,
                  child: Padding(
                    padding: const EdgeInsets.fromLTRB(24, 16, 88, 0),
                    child: Align(
                      alignment: Alignment.centerLeft,
                      child: Text(title!, style: Theme.of(context).textTheme.headlineSmall),
                    ),
                  ),
                ),
              Expanded(child: body),
            ],
          ),
          Positioned(
            top: 0,
            right: 0,
            child: SafeArea(
              child: Padding(
                padding: const EdgeInsets.fromLTRB(8, 8, 12, 8),
                child: Row(
                  mainAxisSize: MainAxisSize.min,
                  children: [
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
                    if (signedIn)
                      IconButton(
                        tooltip: appearance.t('nav.profile'),
                        onPressed: () => context.go('/profile'),
                        icon: const Icon(Icons.person_outline),
                      ),
                    if (signedIn)
                      IconButton(
                        tooltip: appearance.t('nav.logout'),
                        onPressed: () async {
                          context.read<LoopFocus>().clear();
                          await session.logout();
                          if (context.mounted) context.go('/');
                        },
                        icon: const Icon(Icons.logout),
                      ),
                  ],
                ),
              ),
            ),
          ),
        ],
      ),
      floatingActionButton: fab,
      floatingActionButtonLocation: FloatingActionButtonLocation.endFloat,
      bottomNavigationBar: signedIn ? const ProductDock() : null,
    );
  }
}
