import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../session/appearance.dart';
import '../../session/session.dart';

class RegisterPage extends StatefulWidget {
  const RegisterPage({super.key});

  @override
  State<RegisterPage> createState() => _RegisterPageState();
}

class _RegisterPageState extends State<RegisterPage> {
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

  Future<void> _submit() async {
    setState(() {
      busy = true;
      error = null;
    });
    try {
      await context.read<Session>().register(email.text.trim(), password.text, username.text.trim());
      if (mounted) context.go('/dashboard');
    } catch (e) {
      setState(() => error = context.read<Appearance>().t('register.error'));
    } finally {
      if (mounted) setState(() => busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    return Scaffold(
      appBar: AppBar(title: Text(l10n.t('register.title'))),
      body: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 400),
          child: Padding(
            padding: const EdgeInsets.all(24),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                TextField(controller: email, decoration: InputDecoration(labelText: l10n.t('register.email'))),
                const SizedBox(height: 12),
                TextField(controller: username, decoration: InputDecoration(labelText: l10n.t('register.username'))),
                const SizedBox(height: 12),
                TextField(
                  controller: password,
                  decoration: InputDecoration(labelText: l10n.t('register.password'), helperText: l10n.t('register.passwordHint')),
                  obscureText: true,
                ),
                if (error != null) ...[
                  const SizedBox(height: 12),
                  Text(error!, style: TextStyle(color: Theme.of(context).colorScheme.error)),
                ],
                const SizedBox(height: 20),
                FilledButton(onPressed: busy ? null : _submit, child: Text(l10n.t('register.submit'))),
                TextButton(onPressed: () => context.go('/login'), child: Text(l10n.t('register.haveAccount'))),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
