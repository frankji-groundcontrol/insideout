import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../secret_query_scrub.dart';
import '../../session/appearance.dart';
import '../../session/session.dart';
import 'auth_door.dart';

/// `/reset-password?token=...`: the landing page for the emailed reset
/// link. Sets a new password; every existing session was revoked
/// server-side, so the only next step is a fresh login.
class ResetPasswordPage extends StatefulWidget {
  const ResetPasswordPage({super.key, required this.token});

  final String token;

  @override
  State<ResetPasswordPage> createState() => _ResetPasswordPageState();
}

class _ResetPasswordPageState extends State<ResetPasswordPage> {
  final password = TextEditingController();
  final confirm = TextEditingController();
  String? error;
  bool busy = false;
  bool done = false;
  late final String _token;

  @override
  void initState() {
    super.initState();
    _token = widget.token;
    scrubSecretQueryKeys(const ['token']);
  }

  @override
  void dispose() {
    password.dispose();
    confirm.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    final l10n = context.read<Appearance>();
    if (password.text != confirm.text) {
      setState(() => error = l10n.t('reset.mismatch'));
      return;
    }
    setState(() {
      busy = true;
      error = null;
    });
    try {
      await context.read<Session>().api.resetPassword(_token, password.text);
      if (mounted) setState(() => done = true);
    } catch (e) {
      if (mounted) setState(() => error = l10n.t('reset.error'));
    } finally {
      if (mounted) setState(() => busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    return Scaffold(
      body: AuthDoor(
        greeting: l10n.t('reset.title'),
        onClose: () => context.go('/'),
        child: done
            ? Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Text(
                    l10n.t('reset.done'),
                    style: Theme.of(context).textTheme.bodyMedium,
                  ),
                  const SizedBox(height: 20),
                  FilledButton(
                    onPressed: () => context.go('/login'),
                    child: Text(l10n.t('reset.doneLogin')),
                  ),
                ],
              )
            : Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  TextField(
                    controller: password,
                    decoration: InputDecoration(
                      labelText: l10n.t('reset.password'),
                      helperText: l10n.t('reset.passwordHint'),
                    ),
                    obscureText: true,
                  ),
                  const SizedBox(height: 12),
                  TextField(
                    controller: confirm,
                    decoration: InputDecoration(
                      labelText: l10n.t('reset.confirm'),
                    ),
                    obscureText: true,
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
                    child: Text(l10n.t('reset.submit')),
                  ),
                  TextButton(
                    onPressed: () => context.go('/login'),
                    child: Text(l10n.t('reset.back')),
                  ),
                ],
              ),
      ),
    );
  }
}
