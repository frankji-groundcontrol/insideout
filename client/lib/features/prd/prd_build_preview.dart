import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../session/appearance.dart';
import '../../theme/block_button.dart';
import '../../theme/block_modal.dart';
import '../../theme/cobalt.dart';

/// Shows the AI-drafted roadmap [tree] (`{title, children}`) read-only and
/// resolves to true only when the person applies it (rv5-FEAT-09): the draft
/// writes nothing, so this is the one look they get before confirm creates it.
Future<bool> showDraftedRoadmap(
  BuildContext context,
  Map<String, dynamic> tree,
) async {
  final t = context.read<Appearance>().t;
  final navigator = Navigator.of(context);
  final ok = await showBlockModal<bool>(
    context: context,
    title: t('prd.buildReviewTitle'),
    body: Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(t('prd.buildReviewHint')),
        const SizedBox(height: Cobalt.space3),
        DraftedRoadmapTree(node: tree),
      ],
    ),
    footer: [
      BlockButton(
        label: t('common.cancel'),
        variant: BlockButtonVariant.secondary,
        onPressed: () => navigator.pop(false),
      ),
      BlockButton(
        label: t('prd.buildApply'),
        onPressed: () => navigator.pop(true),
      ),
    ],
  );
  return ok == true;
}

/// One drafted node and, indented under it, its children.
class DraftedRoadmapTree extends StatelessWidget {
  const DraftedRoadmapTree({super.key, required this.node, this.depth = 0});

  final Map<String, dynamic> node;
  final int depth;

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    final children = (node['children'] as List?) ?? const [];
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Padding(
          padding: EdgeInsets.only(
            left: depth * Cobalt.space4,
            bottom: Cobalt.space2,
          ),
          child: Text(
            node['title']?.toString() ?? '',
            style: depth == 0 ? text.titleSmall : text.bodyMedium,
          ),
        ),
        for (final c in children)
          if (c is Map)
            DraftedRoadmapTree(
              node: Map<String, dynamic>.from(c),
              depth: depth + 1,
            ),
      ],
    );
  }
}
