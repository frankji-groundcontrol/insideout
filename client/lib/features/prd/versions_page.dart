import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../api/models.dart';
import '../../app.dart';
import '../../chrome/loop_focus.dart';
import '../../session/session.dart';
import '../../theme/block_state.dart';
import '../../widgets/guarded_action.dart';
import 'versions_commit_dialog.dart';
import 'versions_stack.dart';

/// Versions page: the human Commit as a centered fanned stack.
class VersionsPage extends StatefulWidget {
  const VersionsPage({super.key, required this.prdId});

  final String prdId;

  @override
  State<VersionsPage> createState() => _VersionsPageState();
}

class _VersionsPageState extends State<VersionsPage> {
  late Future<List<PrdCommit>> _commits;
  late Future<PrdReadiness> _readiness;
  late Future<List<Object>> _both;
  String audience = 'decision';
  String? selectedId;

  @override
  void initState() {
    super.initState();
    context.read<LoopFocus>().rememberPrd(widget.prdId);
    _reload();
  }

  void _reload() {
    final api = context.read<Session>().api;
    _commits = api.listPrdCommits(widget.prdId);
    _readiness = api.prdReadiness(widget.prdId);
    // Cached: rebuilding the wait on every setState used to drop the stack
    // back to its loading state on a plain selection change.
    _both = Future.wait<Object>([_readiness, _commits]);
    _commits.then((commits) {
      if (!mounted) return;
      context.read<LoopFocus>().rememberPrd(
        widget.prdId,
        committed: commits.isNotEmpty,
      );
    }, onError: (_) {});
  }

  Future<void> _commitDialog() async {
    List<String> carry;
    try {
      carry =
          (await _readiness).audiences[audience]?.carryIntoCommit ?? const [];
    } catch (_) {
      carry = const [];
    }
    if (!mounted) return;
    final result = await showVersionsCommit(
      context: context,
      initialAudience: audience,
      carryCount: carry.length,
    );
    if (result == null || result.name.isEmpty || !mounted) return;
    final done = await guardWrite(
      context,
      () => context.read<Session>().api.commitPrd(
        widget.prdId,
        name: result.name,
        audience: result.audience,
        summary: result.summary,
        unresolved: carry,
      ),
    );
    if (done && mounted) setState(_reload);
  }

  @override
  Widget build(BuildContext context) {
    return AppScaffold(
      body: FutureBuilder<List<Object>>(
        future: _both,
        builder: (context, snap) {
          if (snap.hasError) {
            return BlockError(
              error: snap.error,
              onRetry: () => setState(_reload),
            );
          }
          if (!snap.hasData) return const BlockLoading();
          final readiness = snap.data![0] as PrdReadiness;
          final commits = snap.data![1] as List<PrdCommit>;
          final selected = commits.cast<PrdCommit?>().firstWhere(
            (c) => c!.id == selectedId,
            orElse: () => commits.isEmpty ? null : commits.first,
          );
          return VersionsStack(
            prdId: widget.prdId,
            commits: commits,
            selected: selected,
            readiness: readiness,
            audience: audience,
            onSelect: (id) => setState(() => selectedId = id),
            onAudience: (a) => setState(() => audience = a),
            onCommit: _commitDialog,
          );
        },
      ),
    );
  }
}
