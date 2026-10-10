import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../session/appearance.dart';
import '../../theme/field_card.dart';

/// "Done, with its evidence": the evidence rows of the finished leaves, so
/// the Evidence screen shows what finished work actually produced. The page
/// owns [future] (it reloads with the rest of the page).
class ProjectLeafEvidence extends StatelessWidget {
  const ProjectLeafEvidence({super.key, required this.future});

  final Future<List<({String title, List<Map<String, dynamic>> rows})>> future;

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    final theme = Theme.of(context);
    return FutureBuilder<
      List<({String title, List<Map<String, dynamic>> rows})>
    >(
      future: future,
      builder: (context, esnap) {
        if (!esnap.hasData || esnap.data!.isEmpty) {
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
            for (final leaf in esnap.data!)
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
                          '${row['kind'] ?? ''}: ${row['detail'] ?? ''}',
                          style: theme.textTheme.bodySmall,
                          maxLines: 2,
                          overflow: TextOverflow.ellipsis,
                        ),
                      ),
                  ],
                ),
              ),
            const SizedBox(height: 20),
          ],
        );
      },
    );
  }
}
