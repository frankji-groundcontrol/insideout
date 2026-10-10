import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/chrome/loop_focus.dart';
import 'package:insideout/features/prd/prd_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

// The page's _load used to write its title/section controllers (and seed
// LoopFocus) after its awaits resolved, guarded only by a mounted check far
// below the writes: a getPrd that answered after the member left the PRD ran
// the whole block against a disposed page. The empty title steps past the
// disposed-controller assert (same text short-circuits the notifier) so the
// run reaches the LoopFocus seed — the witness that the unguarded block ran
// for a page that is already gone.
class _SlowApi extends ApiClient {
  _SlowApi(this.prd)
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  final Prd prd;
  final Completer<Prd> pending = Completer<Prd>();

  @override
  Future<Prd> getPrd(String id) => pending.future;

  @override
  Future<Conversation?> conversationForPrd(String prdId) async => null;

  // Only reached with the page alive; kept off the network like the rest.
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

Prd _prd() => Prd(
  id: 'prd-1',
  workspaceId: 'ws-1',
  title: '',
  sections: const {},
  status: 'draft',
  currentRevision: 1,
  updatedAt: '2026-10-05T00:00:00Z',
);

void main() {
  testWidgets('a load that lands after the page is gone writes nothing', (
    tester,
  ) async {
    final focus = LoopFocus();
    final api = _SlowApi(_prd());
    final session = Session()
      ..ready = true
      ..api = api;
    tester.view.physicalSize = const Size(1400, 900);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
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
        child: const MaterialApp(home: PrdPage(id: 'prd-1')),
      ),
    );
    await tester.pump();
    // initState only seeds the PRD id; the workspace arrives with the load.
    expect(focus.workspaceId, isNull);

    await tester.pumpWidget(const SizedBox.shrink());
    api.pending.complete(api.prd);
    await tester.pump();
    await tester.pump();

    expect(focus.workspaceId, isNull);
    expect(tester.takeException(), isNull);
  });
}
