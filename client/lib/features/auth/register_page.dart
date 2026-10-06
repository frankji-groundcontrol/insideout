import 'package:flutter/material.dart';

import '../../secret_query_scrub.dart';
import '../landing/landing_page.dart';
import 'auth_door.dart';

/// Deep link target: landing with a register prompt on top. [invite] is the
/// code from an emailed/admin-copied `/register?invite=` link.
class RegisterPage extends StatefulWidget {
  const RegisterPage({super.key, this.invite = ''});

  final String invite;

  @override
  State<RegisterPage> createState() => _RegisterPageState();
}

class _RegisterPageState extends State<RegisterPage> {
  late final String _invite;

  @override
  void initState() {
    super.initState();
    _invite = widget.invite;
    scrubSecretQueryKeys(const ['invite']);
  }

  @override
  Widget build(BuildContext context) =>
      LandingPage(initialPrompt: AuthPrompt.register, invite: _invite);
}
