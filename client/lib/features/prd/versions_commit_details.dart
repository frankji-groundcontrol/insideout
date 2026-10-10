import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../api/models.dart';
import '../../session/appearance.dart';
import '../../theme/block_chrome.dart';
import '../../theme/cobalt.dart';
import 'versions_gap_text.dart';

/// What a committed version records beyond its name: the change summary, the
/// diff against the previous commit, and the open items it carried
/// (PRODUCT.md, "Commit").
class CommitDetails extends StatelessWidget {
  const CommitDetails({super.key, required this.commit});

  final PrdCommit commit;

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    final theme = Theme.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      mainAxisSize: MainAxisSize.min,
      children: [
        const SizedBox(height: Cobalt.space3),
        BlockTag(label: l10n.t('versions.details')),
        const SizedBox(height: Cobalt.space2),
        if (commit.changeSummary.isNotEmpty)
          Text(
            commit.changeSummary,
            maxLines: 4,
            overflow: TextOverflow.ellipsis,
            style: theme.textTheme.bodyLarge,
          ),
        Text(
          l10n.t('versions.diffTitle', {
            'a': commit.diffCounts['added'] ?? 0,
            'c': commit.diffCounts['changed'] ?? 0,
            'r': commit.diffCounts['removed'] ?? 0,
          }),
          style: theme.textTheme.bodySmall,
        ),
        if (commit.unresolved.isNotEmpty) ...[
          const SizedBox(height: Cobalt.space3),
          Text(
            l10n.t('versions.unresolvedTitle'),
            style: theme.textTheme.titleSmall,
          ),
          for (final item in commit.unresolved)
            Padding(
              padding: const EdgeInsets.only(top: Cobalt.space1),
              child: Text(
                '· ${carriedItem(l10n, commit.primaryAudience, item)}',
                style: theme.textTheme.bodySmall,
              ),
            ),
        ],
      ],
    );
  }
}
