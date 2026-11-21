import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../api/errors.dart';
import '../../session/appearance.dart';
import '../../session/session.dart';
import '../../theme/block_modal.dart';
import '../../widgets/guarded_action.dart';
import 'prd_build_preview.dart';

/// Build roadmap from the PRD. The server drafts on /build (writing nothing)
/// and writes on /build/confirm, so the person reviews the drafted tree first
/// (rv5-FEAT-09). A replace answers 409: `replace_conflict` asks for the live
/// count back, `roadmap_replace_has_history` for `force` once the person has
/// read what will be lost (rv5 row 1.5). The project to open is the one
/// confirm names: the draft carries none for an unlinked PRD. Anything else
/// the server rejects (say `prd_not_approved`) surfaces through the guard.
Future<void> buildRoadmapFromPrd(BuildContext context, String prdId) async {
  final api = context.read<Session>().api;
  final t = context.read<Appearance>().t;
  var tree = const <String, dynamic>{};
  final drafted = await guardWrite(context, () async {
    final draft = await api.buildFromPrd(prdId);
    tree = Map<String, dynamic>.from(draft['tree'] as Map);
  });
  if (!drafted || !context.mounted) return;
  if (!await showDraftedRoadmap(context, tree) || !context.mounted) return;
  await guardWrite(context, () async {
    int? expected;
    var force = false;
    while (true) {
      try {
        final result = await api.confirmBuildFromPrd(
          prdId,
          tree,
          expectedCount: expected,
          force: force,
        );
        final projectId = result['projectId']?.toString() ?? '';
        if (projectId.isNotEmpty && context.mounted) {
          context.go('/projects/$projectId/roadmap');
        }
        return;
      } on ApiException catch (e) {
        final live = e.liveCount;
        final history = e.code == 'roadmap_replace_has_history' && !force;
        // A second replace_conflict with a new count means the tree moved.
        final replace = e.code == 'replace_conflict' && expected != live;
        if (live == null || !(history || replace)) rethrow;
        if (!context.mounted) return;
        final ok = await showBlockConfirm(
          context: context,
          title: t(history ? 'prd.buildHistoryTitle' : 'prd.buildMVP'),
          message: history
              ? t('prd.buildHistoryConfirm', {
                  'nodes': live,
                  'evidence': e.evidenceCount ?? 0,
                  'done': e.doneCount ?? 0,
                  'validated': e.validatedCount ?? 0,
                })
              : t(
                  live == 1
                      ? 'prd.buildReplaceConfirmOne'
                      : 'prd.buildReplaceConfirm',
                  {'count': live, 'evidence': e.evidenceCount ?? 0},
                ),
          confirmText: history
              ? t('prd.buildHistoryAction')
              : t('common.confirm'),
          danger: true,
        );
        if (!ok || !context.mounted) return;
        expected = live;
        force = force || history;
      }
    }
  });
}
