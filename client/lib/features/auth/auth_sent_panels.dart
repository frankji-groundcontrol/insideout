import 'package:flutter/material.dart';

import '../../theme/block_button.dart';
import '../../theme/block_chrome.dart';
import '../../theme/block_field.dart';
import '../../theme/cobalt.dart';

/// Confirmation panel after the emailed sign-in link was sent.
class MagicSentPanel extends StatelessWidget {
  const MagicSentPanel({
    super.key,
    required this.message,
    required this.backLabel,
    required this.onBack,
  });

  final String message;
  final String backLabel;
  final VoidCallback onBack;

  @override
  Widget build(BuildContext context) {
    return Column(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Text(message, style: Theme.of(context).textTheme.bodyMedium),
        const SizedBox(height: Cobalt.space6),
        BlockButton(
          label: backLabel,
          variant: BlockButtonVariant.ghost,
          expand: true,
          onPressed: onBack,
        ),
      ],
    );
  }
}

/// Code-entry panel after the emailed 6-digit sign-in code was sent.
class CodeEntryPanel extends StatelessWidget {
  const CodeEntryPanel({
    super.key,
    required this.message,
    required this.controller,
    required this.codeLabel,
    required this.error,
    required this.busy,
    required this.verifyLabel,
    required this.resendLabel,
    required this.backLabel,
    required this.onVerify,
    required this.onResend,
    required this.onBack,
  });

  final String message;
  final TextEditingController controller;
  final String codeLabel;
  final String? error;
  final bool busy;
  final String verifyLabel;
  final String resendLabel;
  final String backLabel;
  final VoidCallback onVerify;
  final VoidCallback onResend;
  final VoidCallback onBack;

  @override
  Widget build(BuildContext context) {
    return Column(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Text(message, style: Theme.of(context).textTheme.bodyMedium),
        const SizedBox(height: Cobalt.space4),
        BlockField(
          controller: controller,
          label: codeLabel,
          keyboardType: TextInputType.number,
          maxLength: 6,
          autofocus: true,
        ),
        if (error != null) ...[
          const SizedBox(height: Cobalt.space3),
          BlockBanner(message: error!),
        ],
        const SizedBox(height: Cobalt.space6),
        BlockButton(
          label: verifyLabel,
          expand: true,
          busy: busy,
          onPressed: onVerify,
        ),
        const SizedBox(height: Cobalt.space1),
        BlockButton(
          label: resendLabel,
          variant: BlockButtonVariant.ghost,
          expand: true,
          busy: busy,
          onPressed: onResend,
        ),
        BlockButton(
          label: backLabel,
          variant: BlockButtonVariant.ghost,
          expand: true,
          onPressed: onBack,
        ),
      ],
    );
  }
}
