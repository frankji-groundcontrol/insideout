import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../app.dart';
import '../../session/appearance.dart';
import '../../session/session.dart';

class ProfilePage extends StatefulWidget {
  const ProfilePage({super.key});

  @override
  State<ProfilePage> createState() => _ProfilePageState();
}

class _ProfilePageState extends State<ProfilePage> {
  late final TextEditingController username;
  late final TextEditingController bio;
  late final TextEditingController keywords;
  String? error;
  bool busy = false;

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

  Future<void> _save() async {
    setState(() {
      busy = true;
      error = null;
    });
    try {
      final session = context.read<Session>();
      final updated = await session.api.updateMe(
        username: username.text.trim(),
        bio: bio.text,
        keywords: keywords.text.split(',').map((s) => s.trim()).where((s) => s.isNotEmpty).toList(),
      );
      await session.applyUser(updated);
    } catch (e) {
      setState(() => error = e.toString());
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
      body: ListView(
        padding: const EdgeInsets.all(24),
        children: [
          Text(user?.email ?? '', style: Theme.of(context).textTheme.bodySmall),
          const SizedBox(height: 16),
          TextField(controller: username, decoration: InputDecoration(labelText: l10n.t('profile.username'))),
          const SizedBox(height: 12),
          TextField(controller: bio, decoration: InputDecoration(labelText: l10n.t('profile.bio'), hintText: l10n.t('profile.bioPlaceholder')), maxLines: 3),
          const SizedBox(height: 12),
          TextField(controller: keywords, decoration: InputDecoration(labelText: l10n.t('profile.keywords'), hintText: l10n.t('profile.keywordsPlaceholder'))),
          if (error != null) Padding(padding: const EdgeInsets.only(top: 12), child: Text(error!)),
          const SizedBox(height: 20),
          FilledButton(onPressed: busy ? null : _save, child: Text(l10n.t('profile.save'))),
        ],
      ),
    );
  }
}
