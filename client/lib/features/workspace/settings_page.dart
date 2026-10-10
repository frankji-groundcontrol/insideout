import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../api/models.dart';
import '../../app.dart';
import '../../chrome/product_dock.dart';
import '../../session/appearance.dart';
import '../../session/session.dart';
import '../../theme/block_button.dart';
import '../../theme/block_chrome.dart';
import '../../theme/block_field.dart';
import '../../theme/block_modal.dart';
import '../../theme/block_state.dart';
import '../../theme/cobalt.dart';
import '../../theme/cobalt_theme.dart';
import '../../theme/field_card.dart';
import '../../widgets/guarded_action.dart';

/// A settings form reads as a document, not a dashboard — the column stays
/// narrower than the content cap.
const double _formColumnWidth = 640;

class SettingsPage extends StatefulWidget {
  const SettingsPage({super.key, required this.workspaceId});

  final String workspaceId;

  @override
  State<SettingsPage> createState() => _SettingsPageState();
}

class _SettingsPageState extends State<SettingsPage> {
  late Future<({Workspace ws, List<WorkspaceMember> members})> _future;
  final title = TextEditingController();
  final description = TextEditingController();

  @override
  void initState() {
    super.initState();
    _future = _load();
  }

  @override
  void dispose() {
    title.dispose();
    description.dispose();
    super.dispose();
  }

  Future<({Workspace ws, List<WorkspaceMember> members})> _load() async {
    final api = context.read<Session>().api;
    final ws = await api.getWorkspace(widget.workspaceId);
    title.text = ws.title;
    description.text = ws.description;
    return (ws: ws, members: await api.listMembers(widget.workspaceId));
  }

  Future<void> _reload() async {
    setState(() {
      _future = _load();
    });
  }

  Future<void> _save() async {
    final done = await guardWrite(
      context,
      () => context.read<Session>().api.updateWorkspace(
        widget.workspaceId,
        title: title.text.trim(),
        description: description.text,
      ),
    );
    if (done && mounted) await _reload();
  }

  Future<void> _delete() async {
    final t = context.read<Appearance>().t;
    final ok = await showBlockConfirm(
      context: context,
      title: t('workspace.deleteWorkspace'),
      message: t('workspace.deleteWorkspaceConfirm'),
      confirmText: t('common.delete'),
      danger: true,
    );
    if (!ok || !mounted) return;
    final done = await guardWrite(
      context,
      () => context.read<Session>().api.deleteWorkspace(widget.workspaceId),
    );
    if (done && mounted) context.go('/dashboard');
  }

  Future<void> _memberAction(WorkspaceMember m, String value) async {
    final api = context.read<Session>().api;
    final done = await guardWrite(
      context,
      () => value == 'remove'
          ? api.removeMember(widget.workspaceId, m.userId)
          : api.updateMemberRole(widget.workspaceId, m.userId, value),
    );
    if (done && mounted) await _reload();
  }

  @override
  Widget build(BuildContext context) {
    return FutureBuilder<({Workspace ws, List<WorkspaceMember> members})>(
      future: _future,
      builder: (context, snap) {
        final l10n = context.watch<Appearance>();
        return AppScaffold(
          title: l10n.t('workspace.settingsTitle'),
          paintTitle: true,
          body: !snap.hasData
              ? snap.hasError
                    ? BlockError(error: snap.error, onRetry: _reload)
                    : const BlockLoading()
              : Center(
                  child: ConstrainedBox(
                    constraints: const BoxConstraints(
                      maxWidth: _formColumnWidth,
                    ),
                    child: ListView(
                      padding: EdgeInsets.fromLTRB(
                        Cobalt.space4,
                        Cobalt.space4,
                        Cobalt.space4,
                        ProductDock.clearance,
                      ),
                      children: [
                        BlockField(
                          label: l10n.t('workspace.createNamePlaceholder'),
                          controller: title,
                        ),
                        const SizedBox(height: Cobalt.space3),
                        BlockTextarea(
                          label: l10n.t('workspace.createDescPlaceholder'),
                          controller: description,
                        ),
                        const SizedBox(height: Cobalt.space4),
                        BlockButton(
                          label: l10n.t('common.save'),
                          icon: Icons.save_outlined,
                          expand: true,
                          onPressed: _save,
                        ),
                        const SizedBox(height: Cobalt.space8),
                        _inviteCard(context, snap.data!.ws),
                        const SizedBox(height: Cobalt.space8),
                        Text(
                          l10n.t('workspace.membersTitle').toUpperCase(),
                          style: eyebrow(context),
                        ),
                        const SizedBox(height: Cobalt.space3),
                        for (final m in snap.data!.members) ...[
                          _memberCard(context, m),
                          const SizedBox(height: Cobalt.space3),
                        ],
                        const SizedBox(height: Cobalt.space4),
                        BlockButton(
                          label: l10n.t('workspace.deleteWorkspace'),
                          icon: Icons.delete_outline,
                          variant: BlockButtonVariant.danger,
                          expand: true,
                          onPressed: _delete,
                        ),
                      ],
                    ),
                  ),
                ),
        );
      },
    );
  }

  Widget _inviteCard(BuildContext context, Workspace ws) {
    final l10n = context.watch<Appearance>();
    final theme = Theme.of(context);
    return FieldCard(
      padding: const EdgeInsets.all(Cobalt.space4),
      child: Row(
        children: [
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  l10n.t('workspace.inviteHint').toUpperCase(),
                  style: eyebrow(context),
                ),
                const SizedBox(height: Cobalt.space1),
                Text(
                  l10n.t('workspace.inviteCode', {'code': ws.code}),
                  style: theme.textTheme.titleMedium,
                ),
              ],
            ),
          ),
          BlockIconButton(
            icon: Icons.copy,
            tooltip: l10n.t('common.copy'),
            onPressed: () => Clipboard.setData(ClipboardData(text: ws.code)),
          ),
        ],
      ),
    );
  }

  Widget _memberCard(BuildContext context, WorkspaceMember m) {
    final l10n = context.watch<Appearance>();
    final theme = Theme.of(context);
    return FieldCard(
      padding: const EdgeInsets.symmetric(
        horizontal: Cobalt.space4,
        vertical: Cobalt.space3,
      ),
      child: Row(
        children: [
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(m.username, style: theme.textTheme.titleSmall),
                Text(
                  '${m.email} · ${l10n.t('workspace.role.${m.role}')}',
                  style: theme.textTheme.bodySmall,
                ),
              ],
            ),
          ),
          PopupMenuButton<String>(
            tooltip: l10n.t('common.moreActions'),
            onSelected: (value) => _memberAction(m, value),
            itemBuilder: (_) => [
              PopupMenuItem(
                value: 'admin',
                child: Text(l10n.t('workspace.role.admin')),
              ),
              PopupMenuItem(
                value: 'member',
                child: Text(l10n.t('workspace.role.member')),
              ),
              PopupMenuItem(
                value: 'remove',
                child: Text(l10n.t('workspace.removeMember')),
              ),
            ],
          ),
        ],
      ),
    );
  }
}
