import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../api/models.dart';
import '../../session/appearance.dart';
import '../../session/session.dart';
import '../../theme/block_form.dart';
import '../../theme/cobalt.dart';
import '../../theme/field_card.dart';
import '../../widgets/guarded_action.dart';

/// Adds a step to a Canvas layer: asks for its title under [heading],
/// creates it under [parentId] (null: a top-level goal), makes it wait on
/// [after] and makes [before] wait on it. A step in [before] the caller
/// cannot write keeps its links, and the snack bar names it.
Future<void> addRoadmapStep(
  BuildContext context, {
  required String projectId,
  required String? parentId,
  required String heading,
  List<RoadmapNode> after = const [],
  List<RoadmapNode> before = const [],
  required Future<void> Function() onReload,
}) async {
  final t = context.read<Appearance>().t;
  final api = context.read<Session>().api;
  final values = await showBlockForm(
    context: context,
    title: heading,
    confirmLabel: t('roadmap.add'),
    fields: [
      BlockFormField(label: t('roadmap.stepPlaceholder'), required: true),
    ],
  );
  if (values == null || !context.mounted) return;
  final title = values.first.trim();
  if (title.isEmpty) return;
  final skipped = [
    for (final b in before)
      if (!b.canWrite) b.title,
  ];
  RoadmapNode? made;
  await guardWrite(context, () async {
    made = await api.createRoadmapNode(projectId, title, parentId: parentId);
    for (final a in after) {
      await api.addRoadmapDependency(made!.id, a.id);
    }
    for (final b in before.where((b) => b.canWrite)) {
      await api.addRoadmapDependency(b.id, made!.id);
    }
  });
  // A link that failed after the create leaves the step: show it anyway.
  if (made != null && context.mounted) await onReload();
  if (skipped.isNotEmpty && context.mounted) {
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text(t('roadmap.linkSkipped', {'titles': skipped.join(', ')})),
      ),
    );
  }
}

/// The empty slot after the Canvas's last column: "+ Next step".
class RoadmapNextStepSlot extends StatelessWidget {
  const RoadmapNextStepSlot({super.key, required this.onTap});

  static const width = 248.0;

  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final t = context.watch<Appearance>().t;
    final palette = Cobalt.of(context);
    final label = t('roadmap.nextStep');
    return SizedBox(
      width: width,
      child: Semantics(
        button: true,
        label: label,
        excludeSemantics: true,
        child: FieldCard(
          key: const Key('roadmap-next-step'),
          onTap: onTap,
          padding: const EdgeInsets.all(Cobalt.space6),
          child: Row(
            mainAxisAlignment: MainAxisAlignment.center,
            spacing: Cobalt.space2,
            children: [
              Icon(Icons.add, size: 18, color: palette.fgBrand),
              Text(
                label,
                style: Theme.of(
                  context,
                ).textTheme.labelLarge?.copyWith(color: palette.fgBrand),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
