import 'dart:io';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/errors.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/chrome/loop_focus.dart';
import 'package:insideout/features/dashboard/dashboard_page.dart';
import 'package:insideout/features/prd/versions_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

// r3/cq-async-branch-copypaste: the raw snap.error leak recurred through
// USER-07, USER-11 and review, always in a page's FutureBuilder branch.
// These tests pin the two former raw sites (dashboard, versions) to the
// zh-CN banner — the UI is zh-first, so an English leak reads as a leak —
// and the scan keeps pages from reading the snapshot's error at all: reads
// go through the one shared gate in theme/block_state.dart.
class _FailingReadsApi extends ApiClient {
  _FailingReadsApi()
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  @override
  Future<List<Workspace>> listWorkspaces() async =>
      throw ApiException('boom', status: 500);

  @override
  Future<PrdReadiness> prdReadiness(String id) async =>
      throw ApiException('boom', status: 500);

  @override
  Future<List<PrdCommit>> listPrdCommits(String id) async =>
      throw ApiException('boom', status: 500);
}

Future<void> _pumpZh(WidgetTester tester, Widget page) async {
  final session = Session();
  session.ready = true;
  session.api = _FailingReadsApi();
  await tester.binding.setSurfaceSize(const Size(1400, 900));
  addTearDown(() => tester.binding.setSurfaceSize(null));
  await tester.pumpWidget(
    MultiProvider(
      providers: [
        ChangeNotifierProvider.value(value: session),
        ChangeNotifierProvider(
          create: (_) =>
              Appearance(store: MemorySettingsStore(), locale: 'zh-CN'),
        ),
        ChangeNotifierProvider(create: (_) => LoopFocus()),
      ],
      child: MaterialApp(home: page),
    ),
  );
  await tester.pumpAndSettle();
}

void main() {
  testWidgets('a failed dashboard read shows the zh banner and Retry', (
    tester,
  ) async {
    await _pumpZh(tester, const DashboardPage());

    expect(find.text('出错了'), findsOneWidget);
    expect(find.text('服务器开小差了，请稍后重试'), findsOneWidget);
    expect(find.text('重试'), findsOneWidget);
    expect(find.textContaining('boom'), findsNothing);
  });

  testWidgets('a failed versions read shows the zh banner and Retry', (
    tester,
  ) async {
    await _pumpZh(tester, const VersionsPage(prdId: 'p1'));

    expect(find.text('出错了'), findsOneWidget);
    expect(find.text('服务器开小差了，请稍后重试'), findsOneWidget);
    expect(find.text('重试'), findsOneWidget);
    expect(find.textContaining('boom'), findsNothing);
  });

  test(
    'no page reads the snapshot error directly — reads go through the gate',
    () {
      // The shared readGate in theme/block_state.dart is the one place allowed
      // to read the snapshot's error; every page routes through it, so a new
      // page cannot hand-roll the branch or drop its error half.
      final files = Directory('lib')
          .listSync(recursive: true)
          .whereType<File>()
          .where((f) => f.path.endsWith('.dart'))
          .where((f) => f.path != 'lib/theme/block_state.dart')
          .toList();
      expect(files, isNotEmpty);

      // \b: a neighboring snapshot local like esnap.error is its own (already
      // localized) read and must not be flagged.
      final snapshotError = RegExp(r'\bsnap\.error');
      final hits = <String>[
        for (final file in files)
          for (final (i, line) in file.readAsLinesSync().indexed)
            if (snapshotError.hasMatch(line))
              '${file.path}:${i + 1}: ${line.trim()}',
      ];
      expect(
        hits,
        isEmpty,
        reason:
            'Pages reading snap.error instead of the shared gate:\n'
            '${hits.join('\n')}',
      );
    },
  );
}
