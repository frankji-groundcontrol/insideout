import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';

import '../../api/download_stub.dart'
    if (dart.library.ui_web) '../../api/download_web.dart';

import '../../session/appearance.dart';
import '../../session/session.dart';
import '../../theme/block_button.dart';
import '../../theme/block_field.dart';
import '../../theme/cobalt.dart';
import '../../theme/field_card.dart';
import '../../widgets/guarded_action.dart';

/// The GitHub sync card on the Evidence page: link a repo, then pull its
/// commits onto the timeline. The page owns the [repo] controller (it seeds
/// it from the loaded project) and reloads itself through [onChanged].
class ProjectGithubCard extends StatelessWidget {
  const ProjectGithubCard({
    super.key,
    required this.projectId,
    required this.repo,
    required this.onChanged,
  });

  final String projectId;
  final TextEditingController repo;
  final Future<void> Function() onChanged;

  /// Saves the guide as a file on the web, copies it elsewhere (PARB-11).
  Future<void> _guide(BuildContext context, Appearance l10n) async {
    final api = context.read<Session>().api;
    final messenger = ScaffoldMessenger.of(context);
    await guardWrite(context, () async {
      final text = await api.getGuide(projectId);
      if (downloadTextFile('insideout.yaml', text, mime: 'text/yaml')) return;
      await Clipboard.setData(ClipboardData(text: text));
      messenger.showSnackBar(
        SnackBar(content: Text(l10n.t('github.guideCopied'))),
      );
    });
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    final theme = Theme.of(context);
    return FieldCard(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(l10n.t('github.title'), style: theme.textTheme.titleMedium),
          const SizedBox(height: Cobalt.space2),
          Text(l10n.t('github.guideHint'), style: theme.textTheme.bodySmall),
          const SizedBox(height: Cobalt.space3),
          BlockField(controller: repo, label: l10n.t('github.placeholder')),
          const SizedBox(height: Cobalt.space3),
          Wrap(
            spacing: Cobalt.space2,
            runSpacing: Cobalt.space2,
            children: [
              BlockButton(
                label: l10n.t('github.save'),
                variant: BlockButtonVariant.secondary,
                onPressed: () async {
                  final done = await guardWrite(
                    context,
                    () => context.read<Session>().api.setRepo(
                      projectId,
                      repo.text.trim(),
                    ),
                  );
                  if (done && context.mounted) await onChanged();
                },
              ),
              BlockButton(
                label: l10n.t('github.sync'),
                icon: Icons.sync,
                variant: BlockButtonVariant.secondary,
                onPressed: () async {
                  await guardWrite(context, () async {
                    final result = await context.read<Session>().api.syncGithub(
                      projectId,
                    );
                    if (context.mounted) {
                      final count = result['added'] ?? result['count'] ?? 0;
                      ScaffoldMessenger.of(context).showSnackBar(
                        SnackBar(
                          content: Text(
                            l10n.t('github.synced', {'count': count}),
                          ),
                        ),
                      );
                    }
                  });
                },
              ),
              BlockButton(
                label: l10n.t('github.guide'),
                icon: Icons.download_outlined,
                variant: BlockButtonVariant.secondary,
                onPressed: () => _guide(context, l10n),
              ),
            ],
          ),
        ],
      ),
    );
  }
}
