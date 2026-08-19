import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../session/appearance.dart';
import '../../session/session.dart';

class LoginPage extends StatefulWidget {
  const LoginPage({super.key});

  @override
  State<LoginPage> createState() => _LoginPageState();
}

class _LoginPageState extends State<LoginPage> {
  final email = TextEditingController();
  final password = TextEditingController();
  String? error;
  bool busy = false;

  @override
  void dispose() {
    email.dispose();
    password.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    setState(() {
      busy = true;
      error = null;
    });
    try {
      await context.read<Session>().login(email.text.trim(), password.text);
      if (mounted) context.go('/dashboard');
    } catch (e) {
      setState(() => error = context.read<Appearance>().t('login.error'));
    } finally {
      if (mounted) setState(() => busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    return Scaffold(
      appBar: AppBar(title: Text(l10n.t('login.title'))),
      body: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 400),
          child: Padding(
            padding: const EdgeInsets.all(24),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                TextField(controller: email, decoration: InputDecoration(labelText: l10n.t('login.username')), autofillHints: const [AutofillHints.email]),
                const SizedBox(height: 12),
                TextField(controller: password, decoration: InputDecoration(labelText: l10n.t('login.password')), obscureText: true),
                if (error != null) ...[
                  const SizedBox(height: 12),
                  Text(error!, style: TextStyle(color: Theme.of(context).colorScheme.error)),
                ],
                const SizedBox(height: 20),
                FilledButton(onPressed: busy ? null : _submit, child: Text(l10n.t('login.loginButton'))),
                TextButton(onPressed: () => context.go('/register'), child: Text(l10n.t('login.noAccount'))),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
