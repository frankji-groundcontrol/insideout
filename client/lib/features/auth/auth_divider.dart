import 'package:flutter/material.dart';

import '../../theme/cobalt.dart';

/// A hairline "— More ways to sign in —" divider separating the primary
/// password path from the alternate sign-in buttons.
class OtherWaysDivider extends StatelessWidget {
  const OtherWaysDivider({super.key, required this.label});

  final String label;

  @override
  Widget build(BuildContext context) {
    final palette = Cobalt.of(context);
    final hairline = Expanded(
      child: Divider(
        thickness: Cobalt.borderDivider,
        height: Cobalt.borderDivider,
        color: palette.strokeSubtle,
      ),
    );
    return Row(
      children: [
        hairline,
        Padding(
          padding: const EdgeInsets.symmetric(horizontal: Cobalt.space3),
          child: Text(
            label,
            style: TextStyle(
              fontFamily: Cobalt.sans,
              fontFamilyFallback: Cobalt.sansFallback,
              fontSize: 12,
              fontWeight: FontWeight.w600,
              color: palette.fgMuted,
            ),
          ),
        ),
        hairline,
      ],
    );
  }
}
