import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../api/models.dart';
import '../../session/appearance.dart';
import '../../theme/block_chrome.dart';
import '../../theme/cobalt.dart';
import 'versions_gap_text.dart';

/// What a committed version records beyond its name: the change summary and the
/// decision note (what and why), the diff against the previous commit (counts,
/// then one line per changed section), and the open items it carried
/// (PRODUCT.md, "Commit" and "Decision Log"). The card is a fixed height and
/// scrolls, so the short human text comes first and the long lists last.
class CommitDetails extends StatelessWidget {
  const CommitDetails({super.key, required this.commit});

  final PrdCommit commit;

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    final theme = Theme.of(context);
    Widget line(String text) => Padding(
      padding: const EdgeInsets.only(top: Cobalt.space1),
      child: Text(text, style: theme.textTheme.bodySmall),
    );
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
        if (commit.decisionNote.isNotEmpty) ...[
          const SizedBox(height: Cobalt.space3),
          Text(l10n.t('versions.noteTitle'), style: theme.textTheme.titleSmall),
          line(commit.decisionNote),
        ],
        const SizedBox(height: Cobalt.space3),
        Text(
          l10n.t('versions.diffTitle', {
            'a': commit.diffCounts['added'] ?? 0,
            'c': commit.diffCounts['changed'] ?? 0,
            'r': commit.diffCounts['removed'] ?? 0,
          }),
          style: theme.textTheme.bodySmall,
        ),
        for (final e in commit.diffSections.entries)
          line(
            '· ${l10n.t('prd.sections.${e.key}')} — '
            '${l10n.t('versions.diff.${e.value}')}',
          ),
        if (commit.unresolved.isNotEmpty) ...[
          const SizedBox(height: Cobalt.space3),
          Text(
            l10n.t('versions.unresolvedTitle'),
            style: theme.textTheme.titleSmall,
          ),
          for (final item in commit.unresolved)
            line('· ${carriedItem(l10n, commit.primaryAudience, item)}'),
        ],
      ],
    );
  }
}
