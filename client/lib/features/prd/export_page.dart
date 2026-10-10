import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';

import '../../api/errors_l10n.dart';
import '../../api/export_format.dart';
import '../../app.dart';
import '../../chrome/product_dock.dart';
import '../../session/appearance.dart';
import '../../session/session.dart';
import '../../theme/block_button.dart';
import '../../theme/block_chrome.dart';
import '../../theme/block_field.dart';
import '../../theme/cobalt.dart';
import '../../theme/field_card.dart';
import '../../api/download_stub.dart'
    if (dart.library.ui_web) '../../api/download_web.dart';

const _audiences = ['decision', 'management', 'delivery', 'validation'];

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
  Object? error;
  bool running = false;

  Future<void> _run() async {
    setState(() {
      error = null;
      body = null;
      running = true;
    });
    try {
      final text = await context.read<Session>().api.exportPrd(
        widget.prdId,
        format,
        audience: audience,
      );
      if (mounted) setState(() => body = text);
    } catch (e) {
      // Keep the object: localizedError maps ApiException by status and a
      // stringified error always falls through to the generic apology.
      if (mounted) setState(() => error = e);
    } finally {
      if (mounted) setState(() => running = false);
    }
  }

  /// downloadTextFile is false wherever a file cannot be saved (every build
  /// but the web one); copying keeps the button from doing nothing.
  Future<void> _saveOrCopy(String filename, String mime) async {
    final text = body!;
    if (downloadTextFile(filename, text, mime: mime)) return;
    final messenger = ScaffoldMessenger.of(context);
    final copied = context.read<Appearance>().t('prd.exportCopied');
    await Clipboard.setData(ClipboardData(text: text));
    messenger.showSnackBar(SnackBar(content: Text(copied)));
  }

  String _formatLabel(TFn t, String value) =>
      t(value == 'print' ? 'prd.exportPrint' : 'prd.exportMarkdown');

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    final theme = Theme.of(context);
    final palette = Cobalt.of(context);
    final markdown = format == 'markdown';
    return AppScaffold(
      title: l10n.t('prd.exportTitle'),
      paintTitle: true,
      body: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: Cobalt.contentMaxWidth),
          child: ListView(
            padding: const EdgeInsets.fromLTRB(
              Cobalt.space6,
              Cobalt.space4,
              Cobalt.space6,
              ProductDock.clearance,
            ),
            children: [
              Wrap(
                spacing: Cobalt.space2,
                runSpacing: Cobalt.space2,
                children: [
                  for (final option in prdExportFormats)
                    BlockButton(
                      label: _formatLabel(l10n.t, option.query),
                      variant: option.query == format
                          ? BlockButtonVariant.primary
                          : BlockButtonVariant.secondary,
                      onPressed: () => setState(() {
                        format = option.query;
                        // The server 400s an audience projection in any
                        // format but markdown, so a pick made earlier must
                        // not ride along into Print.
                        if (format != 'markdown') audience = null;
                        // A preview (or error) belongs to the option that
                        // produced it; Print must not show markdown's.
                        body = null;
                        error = null;
                      }),
                    ),
                ],
              ),
              const SizedBox(height: Cobalt.space4),
              BlockSelect<String>(
                label: l10n.t('prd.exportAudience'),
                enabled: markdown,
                value: audience ?? '',
                options: [
                  (value: '', label: l10n.t('prd.exportFull')),
                  for (final a in _audiences)
                    (value: a, label: l10n.t('versions.audience.$a')),
                ],
                onChanged: (v) => setState(() {
                  audience = v == null || v.isEmpty ? null : v;
                  // Same as the format switch: the preview on screen belongs
                  // to the previous audience, not the one just picked.
                  body = null;
                  error = null;
                }),
              ),
              const SizedBox(height: Cobalt.space4),
              Align(
                alignment: Alignment.centerLeft,
                child: BlockButton(
                  label: l10n.t('prd.exportRun'),
                  icon: Icons.bolt,
                  busy: running,
                  onPressed: running ? null : _run,
                ),
              ),
              if (error != null)
                Padding(
                  padding: const EdgeInsets.only(top: Cobalt.space4),
                  child: BlockBanner(message: localizedError(error!, l10n.t)),
                ),
              if (body != null)
                Padding(
                  padding: const EdgeInsets.only(top: Cobalt.space6),
                  child: FieldCard(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Row(
                          children: [
                            Expanded(
                              child: Text(
                                _formatLabel(l10n.t, format),
                                style: theme.textTheme.titleSmall,
                              ),
                            ),
                            BlockButton(
                              label: markdown
                                  ? l10n.t('common.download')
                                  : l10n.t('prd.exportPrint'),
                              icon: markdown
                                  ? Icons.download
                                  : Icons.print_outlined,
                              variant: BlockButtonVariant.secondary,
                              size: BlockButtonSize.sm,
                              // The print format is HTML sized for paper;
                              // opening it in a tab puts the browser's
                              // print-to-PDF one keystroke away, with a
                              // download fallback when tabs are blocked.
                              onPressed: () {
                                if (markdown) {
                                  _saveOrCopy(
                                    'prd-${widget.prdId}.md',
                                    'text/markdown',
                                  );
                                } else if (!openTextInNewTab(
                                  body!,
                                  mime: 'text/html',
                                )) {
                                  _saveOrCopy(
                                    'prd-${widget.prdId}.html',
                                    'text/html',
                                  );
                                }
                              },
                            ),
                          ],
                        ),
                        const SizedBox(height: Cobalt.space3),
                        Divider(
                          height: Cobalt.borderDivider,
                          thickness: Cobalt.borderDivider,
                          color: palette.strokeSubtle,
                        ),
                        const SizedBox(height: Cobalt.space3),
                        SelectableText(
                          body!,
                          maxLines: 8,
                          style: theme.textTheme.bodySmall,
                        ),
                      ],
                    ),
                  ),
                ),
            ],
          ),
        ),
      ),
    );
  }
}
