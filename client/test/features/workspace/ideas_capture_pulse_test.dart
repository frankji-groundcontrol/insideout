import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/chrome/loop_focus.dart';
import 'package:insideout/features/workspace/ideas_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

// rv5-FE-08: the dock's Capture and Ideas items, tapped on the ideas wall,
// bump ideasPulse; the wall answers by scrolling to the top and focusing its
// capture field, so the tap is not a dead one.
class _Api extends ApiClient {
  _Api()
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  @override
  Future<List<Idea>> listIdeas(String workspaceId) async => [
    for (var i = 0; i < 30; i++)
      Idea(
        id: 'i$i',
        workspaceId: workspaceId,
        title: 'Idea $i',
        content: '',
        status: 'inbox',
        createdAt: '2026-11-01T00:00:00Z',
      ),
  ];

  @override
  Future<List<Project>> listProjects(String workspaceId) async => [];
}

void main() {
  testWidgets('an ideas pulse scrolls to the capture field and focuses it', (
    tester,
  ) async {
    final focus = LoopFocus();
    final session = Session()
      ..ready = true
      ..api = _Api();
    await tester.binding.setSurfaceSize(const Size(1200, 700));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    await tester.pumpWidget(
      MultiProvider(
        providers: [
          ChangeNotifierProvider.value(value: session),
          ChangeNotifierProvider(
            create: (_) =>
                Appearance(store: MemorySettingsStore(), locale: 'en-US'),
          ),
          ChangeNotifierProvider.value(value: focus),
        ],
        child: const MaterialApp(home: IdeasPage(workspaceId: 'ws1')),
      ),
    );
    await tester.pumpAndSettle();

    final scrollable = find.byType(Scrollable).first;
    await tester.drag(scrollable, const Offset(0, -2000));
    await tester.pumpAndSettle();
    expect(find.byType(EditableText), findsNothing);

    focus.pulseIdeas();
    await tester.pumpAndSettle();
    final editable = tester.widget<EditableText>(
      find.byType(EditableText).first,
    );
    expect(editable.focusNode.hasFocus, isTrue);
  });
}
