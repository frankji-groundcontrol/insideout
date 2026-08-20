import 'package:flutter/material.dart';

import '../landing/landing_page.dart';
import 'auth_door.dart';

/// Deep link target: landing with a register prompt on top.
class RegisterPage extends StatelessWidget {
  const RegisterPage({super.key});

  @override
  Widget build(BuildContext context) => const LandingPage(initialPrompt: AuthPrompt.register);
}
