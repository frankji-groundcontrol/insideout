import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../api/errors_l10n.dart';
import '../../api/models.dart';
import '../../session/appearance.dart';
import '../../theme/block_chrome.dart';
import '../../theme/field_card.dart';

/// The finished leaves the section has evidence for, and how many more
/// finished leaves the page's cap left out.
typedef LeafEvidence = ({
  List<({String title, List<NodeEvidence> rows})> leaves,
  int more,
});

const _kinds = {
  'activity',
  'implementation',
  'review',
  'release',
  'validation',
};

/// An evidence kind in the reader's language (FE-10); one the table does not
/// know stays as the server sent it.
String _kind(TFn t, String kind) =>
    _kinds.contains(kind) ? t('evidence.kind.$kind') : kind;

/// "Done, with its evidence": the evidence rows of the finished leaves, so
/// the Evidence screen shows what finished work actually produced. The page
/// owns [future] (it reloads with the rest of the page).
class ProjectLeafEvidence extends StatelessWidget {
  const ProjectLeafEvidence({super.key, required this.future});

  final Future<LeafEvidence> future;

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    final theme = Theme.of(context);
    return FutureBuilder<LeafEvidence>(
      future: future,
      builder: (context, esnap) {
        final done = esnap.data;
        // Loading and "nothing finished yet" stay quiet; a failure does not.
        if (!esnap.hasError && (done == null || done.leaves.isEmpty)) {
          return const SizedBox.shrink();
        }
        return Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              l10n.t('evidence.doneTitle'),
              style: theme.textTheme.titleLarge,
            ),
            const SizedBox(height: 12),
            if (esnap.hasError)
              BlockBanner(message: localizedError(esnap.error!, l10n.t)),
            for (final leaf in done?.leaves ?? const [])
              FieldCard(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(leaf.title, style: theme.textTheme.titleMedium),
                    if (leaf.rows.isEmpty)
                      Padding(
                        padding: const EdgeInsets.only(top: 4),
                        child: Text(
                          l10n.t('evidence.noRows'),
                          style: theme.textTheme.bodySmall,
                        ),
                      ),
                    for (final row in leaf.rows)
                      Padding(
                        padding: const EdgeInsets.only(top: 4),
                        child: Text(
                          '${_kind(l10n.t, row.kind)}: ${row.detail}',
                          style: theme.textTheme.bodySmall,
                          maxLines: 2,
                          overflow: TextOverflow.ellipsis,
                        ),
                      ),
                  ],
                ),
              ),
            if (done != null && done.more > 0)
              Padding(
                padding: const EdgeInsets.only(top: 8),
                child: Text(
                  l10n.t('evidence.more', {'n': done.more}),
                  style: theme.textTheme.bodySmall,
                ),
              ),
            const SizedBox(height: 20),
          ],
        );
      },
    );
  }
}
