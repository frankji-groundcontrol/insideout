import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../api/export_format.dart';
import '../../app.dart';
import '../../session/appearance.dart';
import '../../session/session.dart';

class ExportPage extends StatefulWidget {
  const ExportPage({super.key, required this.prdId});

  final String prdId;

  @override
  State<ExportPage> createState() => _ExportPageState();
}

class _ExportPageState extends State<ExportPage> {
  String format = 'markdown';
  String? audience;
  String? body;
  String? error;

  Future<void> _run() async {
    setState(() {
      error = null;
      body = null;
    });
    try {
      final text = await context.read<Session>().api.exportPrd(widget.prdId, format, audience: audience);
      setState(() => body = text);
    } catch (e) {
      setState(() => error = e.toString());
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    return AppScaffold(
      title: l10n.t('prd.exportMarkdown'),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          SegmentedButton<String>(
            segments: [
              for (final option in prdExportFormats)
                ButtonSegment(
                  value: option.query,
                  label: Text(l10n.t(option.query == 'print' ? 'prd.exportPrint' : 'prd.exportMarkdown')),
                ),
            ],
            selected: {format},
            onSelectionChanged: (s) => setState(() => format = s.first),
          ),
          const SizedBox(height: 12),
          DropdownButtonFormField<String>(
            initialValue: audience,
            decoration: InputDecoration(labelText: l10n.t('prd.exportAudience')),
            items: [
              DropdownMenuItem(value: null, child: Text(l10n.t('prd.exportFull'))),
              for (final a in const ['decision', 'management', 'delivery', 'validation'])
                DropdownMenuItem(value: a, child: Text(a)),
            ],
            onChanged: (v) => setState(() => audience = v),
          ),
          const SizedBox(height: 12),
          FilledButton(onPressed: _run, child: Text(l10n.t('common.save'))),
          if (error != null) Padding(padding: const EdgeInsets.only(top: 12), child: Text(error!)),
          if (body != null) Padding(padding: const EdgeInsets.only(top: 16), child: SelectableText(body!)),
        ],
      ),
    );
  }
}
