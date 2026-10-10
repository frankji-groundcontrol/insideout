import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../api/errors_l10n.dart';
import '../../app.dart';
import '../../chrome/product_dock.dart';
import '../../session/appearance.dart';
import '../../session/session.dart';
import '../../theme/block_button.dart';
import '../../theme/block_chrome.dart';
import '../../theme/block_field.dart';
import '../../theme/cobalt.dart';
import 'admin_section.dart';

class ProfilePage extends StatefulWidget {
  const ProfilePage({super.key});

  @override
  State<ProfilePage> createState() => _ProfilePageState();
}

class _ProfilePageState extends State<ProfilePage> {
  late final TextEditingController username;
  late final TextEditingController bio;
  late final TextEditingController keywords;
  Object? error;
  bool busy = false;
  bool saved = false;

  @override
  void initState() {
    super.initState();
    final user = context.read<Session>().user;
    username = TextEditingController(text: user?.username ?? '');
    bio = TextEditingController(text: user?.bio ?? '');
    keywords = TextEditingController(text: user?.keywords.join(', ') ?? '');
  }

  @override
  void dispose() {
    username.dispose();
    bio.dispose();
    keywords.dispose();
    super.dispose();
  }

  void _touched() {
    if (saved) setState(() => saved = false);
  }

  Future<void> _save() async {
    setState(() {
      busy = true;
      error = null;
      saved = false;
    });
    try {
      final session = context.read<Session>();
      final updated = await session.api.updateMe(
        username: username.text.trim(),
        bio: bio.text,
        keywords: keywords.text
            .split(',')
            .map((s) => s.trim())
            .where((s) => s.isNotEmpty)
            .toList(),
      );
      await session.applyUser(updated);
      if (mounted) setState(() => saved = true);
    } catch (e) {
      if (mounted) setState(() => error = e);
    } finally {
      if (mounted) setState(() => busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final user = context.watch<Session>().user;
    final l10n = context.watch<Appearance>();
    return AppScaffold(
      title: l10n.t('profile.title'),
      paintTitle: true,
      body: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: Cobalt.contentMaxWidth),
          child: ListView(
            padding: const EdgeInsets.fromLTRB(
              Cobalt.space6,
              Cobalt.space6,
              Cobalt.space6,
              ProductDock.clearance,
            ),
            children: [
              Text(
                user?.email ?? '',
                style: Theme.of(context).textTheme.bodySmall,
              ),
              const SizedBox(height: Cobalt.space4),
              BlockField(
                controller: username,
                label: l10n.t('profile.username'),
                onChanged: (_) => _touched(),
              ),
              const SizedBox(height: Cobalt.space3),
              BlockTextarea(
                controller: bio,
                label: l10n.t('profile.bio'),
                hint: l10n.t('profile.bioPlaceholder'),
                minLines: 3,
                onChanged: (_) => _touched(),
              ),
              const SizedBox(height: Cobalt.space3),
              BlockField(
                controller: keywords,
                label: l10n.t('profile.keywords'),
                hint: l10n.t('profile.keywordsPlaceholder'),
                onChanged: (_) => _touched(),
              ),
              if (error != null) ...[
                const SizedBox(height: Cobalt.space4),
                BlockBanner(message: localizedError(error!, l10n.t)),
              ],
              if (saved && error == null) ...[
                const SizedBox(height: Cobalt.space4),
                BlockBanner(
                  message: l10n.t('profile.saveSuccess'),
                  tone: BlockTone.success,
                ),
              ],
              const SizedBox(height: Cobalt.space6),
              Align(
                alignment: Alignment.centerLeft,
                child: BlockButton(
                  label: l10n.t('profile.save'),
                  busy: busy,
                  onPressed: _save,
                ),
              ),
              if (user?.isAdmin ?? false) const AdminSection(),
            ],
          ),
        ),
      ),
    );
  }
}
