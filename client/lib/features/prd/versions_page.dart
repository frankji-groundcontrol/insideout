import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../api/models.dart';
import '../../app.dart';
import '../../chrome/loop_focus.dart';
import '../../session/session.dart';
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
    _commits.then((commits) {
      if (!mounted) return;
      context.read<LoopFocus>().rememberPrd(widget.prdId, committed: commits.isNotEmpty);
    }, onError: (_) {});
  }

  Future<void> _commitDialog() async {
    List<String> carry;
    try {
      carry = (await _readiness).audiences[audience]?.carryIntoCommit ?? const [];
    } catch (_) {
      carry = const [];
    }
    if (!mounted) return;
    final result = await showDialog<CommitFormResult>(
      context: context,
      builder: (ctx) => VersionsCommitDialog(initialAudience: audience, carryCount: carry.length),
    );
    if (result == null || result.name.isEmpty) return;
    await context.read<Session>().api.commitPrd(
          widget.prdId,
          name: result.name,
          audience: result.audience,
          summary: result.summary,
          unresolved: carry,
        );
    if (mounted) setState(_reload);
  }

  @override
  Widget build(BuildContext context) {
    return AppScaffold(
      body: FutureBuilder(
        future: Future.wait([_readiness, _commits]),
        builder: (context, snap) {
          if (snap.hasError) return Center(child: Text('${snap.error}'));
          if (!snap.hasData) return const Center(child: CircularProgressIndicator());
          final readiness = snap.data![0] as PrdReadiness;
          final commits = snap.data![1] as List<PrdCommit>;
          final selected = commits.cast<PrdCommit?>().firstWhere((c) => c!.id == selectedId, orElse: () => commits.isEmpty ? null : commits.first);
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
