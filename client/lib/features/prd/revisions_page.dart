import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../api/models.dart';
import '../../app.dart';
import '../../session/appearance.dart';
import '../../session/session.dart';
import '../../theme/block_button.dart';
import '../../theme/block_state.dart';
import '../../theme/cobalt.dart';
import 'revisions_table.dart';
import 'snapshot_flow.dart';

class RevisionsPage extends StatefulWidget {
  const RevisionsPage({super.key, required this.prdId});

  final String prdId;

  @override
  State<RevisionsPage> createState() => _RevisionsPageState();
}

class _RevisionsPageState extends State<RevisionsPage> {
  late Future<List<PrdRevision>> _future;
  String? selectedId;

  @override
  void initState() {
    super.initState();
    _reload();
  }

  void _reload() {
    _future = context.read<Session>().api.listRevisions(widget.prdId);
  }

  Future<void> _snapshot() => showSnapshotRevision(
    context,
    prdId: widget.prdId,
    onCreated: () => setState(_reload),
  );

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    return AppScaffold(
      title: l10n.t('prd.revisionsTitle'),
      paintTitle: true,
      body: FutureBuilder<List<PrdRevision>>(
        future: _future,
        builder: (context, snap) {
          if (!snap.hasData) {
            return readGate(snap, () => setState(_reload));
          }
          final revisions = snap.data!;
          if (revisions.isEmpty) {
            return BlockEmpty(
              icon: Icons.history,
              title: l10n.t('prd.snapshotLabel'),
              body: l10n.t('prd.emptyRevisions'),
              action: Wrap(
                spacing: Cobalt.space2,
                runSpacing: Cobalt.space2,
                alignment: WrapAlignment.center,
                children: [
                  BlockButton(
                    key: const Key('revisions-snapshot'),
                    label: l10n.t('prd.snapshotRevision'),
                    icon: Icons.add,
                    onPressed: _snapshot,
                  ),
                  BlockButton(
                    label: l10n.t('prd.backToPrd'),
                    variant: BlockButtonVariant.secondary,
                    onPressed: () => context.go('/prd/${widget.prdId}'),
                  ),
                ],
              ),
            );
          }
          final selected = revisions.cast<PrdRevision?>().firstWhere(
            (r) => r!.id == selectedId,
            orElse: () => revisions.first,
          )!;
          return RevisionsTable(
            revisions: revisions,
            selected: selected,
            onSelect: (id) => setState(() => selectedId = id),
            onSnapshot: _snapshot,
          );
        },
      ),
    );
  }
}
