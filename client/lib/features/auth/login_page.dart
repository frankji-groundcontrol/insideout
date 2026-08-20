import 'package:flutter/material.dart';

import '../landing/landing_page.dart';
import 'auth_door.dart';

/// Deep link / auth-redirect target: landing with a login prompt on top.
class LoginPage extends StatelessWidget {
  const LoginPage({super.key});

  @override
  Widget build(BuildContext context) => const LandingPage(initialPrompt: AuthPrompt.login);
}
