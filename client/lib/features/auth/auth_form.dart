import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../session/appearance.dart';
import '../../session/session.dart';
import 'auth_door.dart';

class AuthForm extends StatefulWidget {
  const AuthForm({super.key, required this.prompt, this.onSwitch});

  final AuthPrompt prompt;
  final VoidCallback? onSwitch;

  @override
  State<AuthForm> createState() => _AuthFormState();
}

class _AuthFormState extends State<AuthForm> {
  final email = TextEditingController();
  final username = TextEditingController();
  final password = TextEditingController();
  String? error;
  bool busy = false;

  @override
  void dispose() {
    email.dispose();
    username.dispose();
    password.dispose();
    super.dispose();
  }

  bool get _register => widget.prompt == AuthPrompt.register;

  Future<void> _submit() async {
    setState(() {
      busy = true;
      error = null;
    });
    final l10n = context.read<Appearance>();
    try {
      final session = context.read<Session>();
      if (_register) {
        await session.register(email.text.trim(), password.text, username.text.trim());
      } else {
        await session.login(email.text.trim(), password.text);
      }
      if (mounted) context.go('/dashboard');
    } catch (e) {
      setState(() => error = l10n.t(_register ? 'register.error' : 'login.error'));
    } finally {
      if (mounted) setState(() => busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    return Column(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        TextField(
          controller: email,
          decoration: InputDecoration(labelText: l10n.t(_register ? 'register.email' : 'login.username')),
          autofillHints: const [AutofillHints.email],
        ),
        if (_register) ...[
          const SizedBox(height: 12),
          TextField(controller: username, decoration: InputDecoration(labelText: l10n.t('register.username'))),
        ],
        const SizedBox(height: 12),
        TextField(
          controller: password,
          decoration: InputDecoration(
            labelText: l10n.t(_register ? 'register.password' : 'login.password'),
            helperText: _register ? l10n.t('register.passwordHint') : null,
          ),
          obscureText: true,
        ),
        if (error != null) ...[
          const SizedBox(height: 12),
          Text(error!, style: TextStyle(color: Theme.of(context).colorScheme.error)),
        ],
        const SizedBox(height: 20),
        FilledButton(
          onPressed: busy ? null : _submit,
          child: Text(l10n.t(_register ? 'register.submit' : 'login.loginButton')),
        ),
        TextButton(
          onPressed: widget.onSwitch,
          child: Text(l10n.t(_register ? 'register.haveAccount' : 'login.noAccount')),
        ),
      ],
    );
  }
}
