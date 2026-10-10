import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/chrome/loop_focus.dart';
import 'package:insideout/features/prd/prd_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

// An empty PRD showed eight headings over blank, borderless fields: nothing
// said the cards were editable or what belonged in them. Each section now
// carries a one-line prompt, in the order prdSectionKeys lists them.
const _hints = {
  'en-US': [
    'What problem is this solving, and why now?',
    'Who is this for? Name a real person or segment.',
    'What should change, and how will you measure it?',
    'What are you deliberately not doing?',
    'Walk through one scenario, start to finish.',
    'What must it do? One behavior per line.',
    'What limits the work: time, budget, tech, dependencies?',
    'What could go wrong, and what is still unanswered?',
  ],
  'zh-CN': [
    '这要解决什么问题？为什么是现在？',
    '这是为谁做的？写出一个真实的人或人群。',
    '要改变什么？如何衡量？',
    '你有意不做什么？',
    '从头到尾走一遍一个具体场景。',
    '它必须做到什么？每行一种行为。',
    '什么在限制这项工作：时间、预算、技术、依赖？',
    '可能出什么问题？还有哪些问题没有答案？',
  ],
};

class _EmptyPrdApi extends ApiClient {
  _EmptyPrdApi()
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  @override
  Future<Prd> getPrd(String id) async => Prd(
    id: id,
    workspaceId: 'ws-1',
    title: 'Ship a thing',
    sections: const {},
    status: 'draft',
    currentRevision: 1,
    updatedAt: '2026-10-05T00:00:00Z',
  );

  @override
  Future<Conversation?> conversationForPrd(String prdId) async => null;

  @override
  Future<Workspace> getWorkspace(String id) async => Workspace.fromJson({
    'id': id,
    'title': 'Board Test WS',
    'description': '',
    'code': 'CODE12',
    'status': 'active',
    'memberCount': 2,
    'myRole': 'admin',
    'createdAt': '2026-01-01T00:00:00Z',
  });
}

Future<void> _mount(WidgetTester tester, String locale) async {
  final session = Session()
    ..ready = true
    ..api = _EmptyPrdApi();
  // Tall enough that the list builds all eight cards, not just the first few.
  await tester.binding.setSurfaceSize(const Size(1400, 3200));
  addTearDown(() => tester.binding.setSurfaceSize(null));
  final router = GoRouter(
    initialLocation: '/prd/prd-1',
    routes: [
      GoRoute(
        path: '/prd/:id',
        builder: (_, s) => PrdPage(id: s.pathParameters['id']!),
      ),
    ],
  );
  addTearDown(router.dispose);
  await tester.pumpWidget(
    MultiProvider(
      providers: [
        ChangeNotifierProvider.value(value: session),
        ChangeNotifierProvider(
          create: (_) =>
              Appearance(store: MemorySettingsStore(), locale: locale),
        ),
        ChangeNotifierProvider(create: (_) => LoopFocus()),
      ],
      child: MaterialApp.router(routerConfig: router),
    ),
  );
  await tester.pumpAndSettle();
}

void main() {
  for (final entry in _hints.entries) {
    testWidgets('every empty section card says what belongs in it '
        '(${entry.key})', (tester) async {
      await _mount(tester, entry.key);

      for (final hint in entry.value) {
        expect(find.text(hint), findsOneWidget, reason: hint);
      }
    });
  }

  testWidgets('the title field carries no hint', (tester) async {
    await _mount(tester, 'en-US');

    final title = tester.widget<TextField>(find.byType(TextField).first);
    expect(title.decoration?.hintText, isNull);
  });
}
