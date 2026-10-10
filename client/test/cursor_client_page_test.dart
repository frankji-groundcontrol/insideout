import 'dart:async';

import 'package:flutter/gestures.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/chrome/loop_focus.dart';
import 'package:insideout/features/project/roadmap_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

// The server keys a viewer's presence entry and cursor by the `client` query
// value, so the page must speak for the same client on both: otherwise the
// cursor lands under the bare user id, apart from the presence entry, and a
// second tab of the same user is indistinguishable.
class _CursorApi extends ApiClient {
  _CursorApi()
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  String? presenceClient;
  final cursorClients = <String>[];
  final _open = Completer<void>(); // a live stream, with no timers

  @override
  Future<List<RoadmapNode>> listRoadmap(String projectId) async => [
    RoadmapNode.fromJson({
      'id': 'n1',
      'projectId': projectId,
      'title': 'Ship it',
      'description': '',
      'status': 'in_progress',
      'position': 0,
    }),
  ];

  @override
  Future<RoadmapProgress> roadmapProgress(String projectId) async =>
      RoadmapProgress(
        now: const [],
        needsDeadline: const [],
        next: const [],
        doneCount: 0,
      );

  @override
  Future<void> progressStream(
    String projectId,
    void Function(RoadmapProgress progress) onProgress, {
    Future<void>? stop,
  }) => _open.future;

  @override
  Future<void> presenceStream(
    String projectId,
    String clientId,
    void Function(List<PresenceEntry> entries) onChange, {
    void Function(String sessionId, String name, double x, double y)? onCursor,
    Future<void>? stop,
  }) {
    presenceClient = clientId;
    return _open.future;
  }

  @override
  Future<void> cursor(
    String projectId,
    double x,
    double y, {
    required String clientId,
  }) async {
    cursorClients.add(clientId);
  }
}

void main() {
  testWidgets('the page sends its pointer under the id its presence uses', (
    tester,
  ) async {
    await tester.binding.setSurfaceSize(const Size(1400, 900));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    final api = _CursorApi();
    final session = Session()
      ..ready = true
      ..api = api;

    await tester.pumpWidget(
      MultiProvider(
        providers: [
          ChangeNotifierProvider.value(value: session),
          ChangeNotifierProvider(
            create: (_) =>
                Appearance(store: MemorySettingsStore(), locale: 'en-US'),
          ),
          ChangeNotifierProvider(create: (_) => LoopFocus()),
        ],
        child: const MaterialApp(home: RoadmapPage(projectId: 'p1')),
      ),
    );
    await tester.pump();
    await tester.tap(find.text('Canvas'));
    await tester.pumpAndSettle();

    final mouse = await tester.createGesture(kind: PointerDeviceKind.mouse);
    await mouse.addPointer(location: Offset.zero);
    addTearDown(mouse.removePointer);
    await mouse.moveTo(tester.getCenter(find.text('Ship it')));
    await tester.pump();

    expect(api.presenceClient, isNotEmpty);
    expect(api.cursorClients, isNotEmpty);
    expect(api.cursorClients.toSet(), {api.presenceClient});
  });
}
