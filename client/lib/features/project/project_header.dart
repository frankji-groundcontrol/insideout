import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../api/models.dart';
import '../../session/appearance.dart';
import '../../theme/cobalt.dart';
import '../../theme/ink_motion.dart';

/// The Evidence page header: the project title, its owner and the lead line.
/// The dock's Evidence pulse flashes it ([flash]) — a brandSubtle fill behind
/// a strokeStrong border that fades back out (Cobalt-Neo tokens, 0 radius).
class ProjectHeader extends StatelessWidget {
  const ProjectHeader({
    super.key,
    required this.title,
    required this.ownerName,
    required this.flash,
  });

  final String title;

  /// Null when the project has no owner; '' while the members are unknown.
  final String? ownerName;
  final bool flash;

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    final theme = Theme.of(context);
    final palette = Cobalt.of(context);
    return AnimatedContainer(
      key: const ValueKey('evidence-pulse-flash'),
      duration: inkReduceMotion(context)
          ? Duration.zero
          : const Duration(milliseconds: 180),
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: flash
            ? palette.brandSubtle
            : palette.brandSubtle.withValues(alpha: 0),
        border: Border.all(
          color: flash
              ? palette.strokeStrong
              : palette.strokeStrong.withValues(alpha: 0),
          width: Cobalt.borderContainer,
        ),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        mainAxisSize: MainAxisSize.min,
        children: [
          Text(
            l10n.t('evidence.title').toUpperCase(),
            style: theme.textTheme.labelSmall,
          ),
          const SizedBox(height: 8),
          Text(title, style: theme.textTheme.displaySmall),
          const SizedBox(height: 8),
          if (ownerName != '')
            Text(
              ownerName == null
                  ? l10n.t('project.noOwner')
                  : l10n.t('project.owner', {'name': ownerName}),
              style: theme.textTheme.bodySmall,
            ),
          Text(l10n.t('evidence.lead'), style: theme.textTheme.bodyLarge),
        ],
      ),
    );
  }
}

/// The owner's name for [ProjectHeader] and the board card: null when the
/// project has no owner, '' when the owner is not among [members].
String? projectOwnerName(String? ownerId, List<WorkspaceMember> members) {
  if (ownerId == null) return null;
  for (final m in members) {
    if (m.userId == ownerId) return m.username;
  }
  return '';
}
