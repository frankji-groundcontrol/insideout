import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/features/project/project_dialogs.dart';
import 'package:insideout/session/appearance.dart';
import 'package:provider/provider.dart';

// rv4-USER-11: the Edit project dialog labelled its description field with the
// status-post prompt ("What's new?"), reused from the Post-an-update dialog.
// The description field must read as a description in both locales, and the
// update dialog must keep its own prompt. Rendered output, not the l10n table.

const _prompt = {'en-US': "What's new?", 'zh-CN': '有什么新进展？'};
const _description = {'en-US': 'Project description', 'zh-CN': '项目描述'};

Future<void> _open(
  WidgetTester tester,
  String locale,
  Future<Object?> Function(BuildContext context) show,
) async {
  await tester.pumpWidget(
    ChangeNotifierProvider(
      create: (_) => Appearance(store: MemorySettingsStore(), locale: locale),
      child: MaterialApp(
        home: Builder(
          builder: (context) => Scaffold(
            body: TextButton(
              onPressed: () => show(context),
              child: const Text('open'),
            ),
          ),
        ),
      ),
    ),
  );
  await tester.tap(find.text('open'));
  await tester.pumpAndSettle();
}

void main() {
  for (final locale in const ['en-US', 'zh-CN']) {
    testWidgets('Edit project labels its description as a description '
        '($locale)', (tester) async {
      await _open(
        tester,
        locale,
        (context) => showProjectEdit(context, const {
          'title': 'Flows',
          'description': 'old desc',
          'status': 'active',
        }),
      );

      expect(
        find.text(_prompt[locale]!),
        findsNothing,
        reason: 'the status-post prompt must not label the description field',
      );
      expect(find.text(_description[locale]!), findsOneWidget);
    });

    testWidgets('Post an update keeps its own prompt ($locale)', (
      tester,
    ) async {
      await _open(tester, locale, showProjectAddUpdate);

      expect(find.text(_prompt[locale]!), findsOneWidget);
      expect(find.text(_description[locale]!), findsNothing);
    });
  }
}
