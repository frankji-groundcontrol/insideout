import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../secret_query_scrub.dart';
import '../../session/appearance.dart';
import '../../session/session.dart';
import '../../theme/block_button.dart';
import '../../theme/block_chrome.dart';
import '../../theme/cobalt.dart';
import 'auth_door.dart';

/// `/magic-link?token=...`: the landing page for the emailed sign-in link.
/// Consumes the one-time token on load — success lands on /dashboard,
/// anything else shows the dead-link state.
class MagicLinkPage extends StatefulWidget {
  const MagicLinkPage({super.key, required this.token});

  final String token;

  @override
  State<MagicLinkPage> createState() => _MagicLinkPageState();
}

class _MagicLinkPageState extends State<MagicLinkPage> {
  bool _working = true;
  late final String _token;

  @override
  void initState() {
    super.initState();
    _token = widget.token;
    scrubSecretQueryKeys(const ['token']);
    WidgetsBinding.instance.addPostFrameCallback((_) => _consume());
  }

  Future<void> _consume() async {
    try {
      await context.read<Session>().consumeMagicLink(_token);
      if (mounted) context.go('/dashboard');
    } catch (e) {
      if (mounted) setState(() => _working = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    return Scaffold(
      body: AuthDoor(
        greeting: l10n.t('magic.title'),
        onClose: () => context.go('/'),
        child: _working
            ? Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.center,
                children: [
                  const BlockLoader(size: Cobalt.space6),
                  const SizedBox(height: Cobalt.space4),
                  Text(
                    l10n.t('magic.working'),
                    style: Theme.of(context).textTheme.bodyMedium,
                  ),
                ],
              )
            : Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  BlockBanner(message: l10n.t('magic.error')),
                  const SizedBox(height: Cobalt.space6),
                  BlockButton(
                    label: l10n.t('magic.back'),
                    expand: true,
                    onPressed: () => context.go('/login'),
                  ),
                ],
              ),
      ),
    );
  }
}
