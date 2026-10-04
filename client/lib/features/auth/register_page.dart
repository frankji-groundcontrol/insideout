import 'package:flutter/material.dart';

import '../landing/landing_page.dart';
import 'auth_door.dart';

/// Deep link target: landing with a register prompt on top. [invite] is the
/// code from an emailed/admin-copied `/register?invite=` link.
class RegisterPage extends StatelessWidget {
  const RegisterPage({super.key, this.invite = ''});

  final String invite;

  @override
  Widget build(BuildContext context) => LandingPage(initialPrompt: AuthPrompt.register, invite: invite);
}
