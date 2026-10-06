import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../session/appearance.dart';
import '../../session/session.dart';
import 'auth_door.dart';

/// `/forgot-password`: requests the emailed password-reset link.
class ForgotPasswordPage extends StatefulWidget {
  const ForgotPasswordPage({super.key});

  @override
  State<ForgotPasswordPage> createState() => _ForgotPasswordPageState();
}

class _ForgotPasswordPageState extends State<ForgotPasswordPage> {
  final email = TextEditingController();
  String? error;
  bool busy = false;
  bool sent = false;

  @override
  void dispose() {
    email.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    setState(() {
      busy = true;
      error = null;
    });
    final l10n = context.read<Appearance>();
    try {
      await context.read<Session>().api.forgotPassword(email.text.trim());
      if (mounted) setState(() => sent = true);
    } catch (e) {
      if (mounted) setState(() => error = l10n.t('forgot.error'));
    } finally {
      if (mounted) setState(() => busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    return Scaffold(
      body: AuthDoor(
        greeting: l10n.t('forgot.title'),
        onClose: () => context.go('/'),
        child: sent
            ? Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Text(
                    l10n.t('forgot.sent'),
                    style: Theme.of(context).textTheme.bodyMedium,
                  ),
                  const SizedBox(height: 20),
                  TextButton(
                    onPressed: () => context.go('/login'),
                    child: Text(l10n.t('forgot.back')),
                  ),
                ],
              )
            : Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  TextField(
                    controller: email,
                    decoration: InputDecoration(
                      labelText: l10n.t('forgot.email'),
                    ),
                    autofillHints: const [AutofillHints.email],
                  ),
                  if (error != null) ...[
                    const SizedBox(height: 12),
                    Text(
                      error!,
                      style: TextStyle(
                        color: Theme.of(context).colorScheme.error,
                      ),
                    ),
                  ],
                  const SizedBox(height: 20),
                  FilledButton(
                    onPressed: busy ? null : _submit,
                    child: Text(l10n.t('forgot.submit')),
                  ),
                  TextButton(
                    onPressed: () => context.go('/login'),
                    child: Text(l10n.t('forgot.back')),
                  ),
                ],
              ),
      ),
    );
  }
}
