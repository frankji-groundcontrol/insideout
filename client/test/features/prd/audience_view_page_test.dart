import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/chrome/loop_focus.dart';
import 'package:insideout/features/prd/audience_view_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

// The payloads below copy the wire shape the server sends (pinned by
// TestPrdViewSendsSectionsAsOrderedPicks in server/internal/api): the
// projection's sections are an ordered LIST of {section, why}, not a map.
class _FakeApi extends ApiClient {
  _FakeApi(this.payload, {this.prd})
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  final Map<String, dynamic> payload;
  final Prd? prd;
  int getPrdCalls = 0;

  @override
  Future<Map<String, dynamic>> prdView(String id, String audience) async =>
      payload;

  @override
  Future<Prd> getPrd(String id) async {
    getPrdCalls++;
    return prd ??
        Prd(
          id: id,
          workspaceId: 'ws1',
          title: '',
          sections: const {},
          status: 'draft',
          currentRevision: 1,
          updatedAt: '2026-08-22T00:00:00Z',
        );
  }
}

void main() {
  testWidgets('audience view loads one audience end to end', (tester) async {
    final session = Session();
    session.ready = true;
    session.api = _FakeApi(
      {
        'prdId': 'p1',
        'title': 'Audience end to end',
        'projection': {
          'audience': 'decision',
          'title': 'Decision view',
          'purpose': 'why, why now, investment and return',
          'sections': [
            {'section': 'background', 'why': 'the why-this argument'},
            {'section': 'goals', 'why': 'the investment being decided on'},
          ],
        },
        'readiness': {
          'ready': false,
          'gaps': [
            {
              'section': 'risks',
              'priority': 'should_clarify_this_version',
              'reason': 'the decisions the answer requires now',
            },
          ],
          'carryIntoCommit': [],
        },
        'latestCommit': {'revision': 3, 'name': 'First decision cut'},
      },
      prd: Prd(
        id: 'p1',
        workspaceId: 'ws1',
        title: 'Audience end to end',
        sections: const {'background': 'The problem is real.', 'goals': ''},
        status: 'draft',
        currentRevision: 3,
        updatedAt: '2026-08-22T00:00:00Z',
      ),
    );
    await tester.binding.setSurfaceSize(const Size(1400, 900));
    addTearDown(() => tester.binding.setSurfaceSize(null));

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
        child: const MaterialApp(
          home: AudienceViewPage(prdId: 'p1', initialAudience: 'decision'),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(tester.takeException(), isNull);
    expect(find.text('Audience end to end'), findsOneWidget);
    expect(find.text('why, why now, investment and return'), findsOneWidget);
    expect(find.text('Background & Problem Statement'), findsOneWidget);
    expect(find.textContaining('the why-this argument'), findsOneWidget);
    expect(find.text('The problem is real.'), findsOneWidget);
    expect(find.text('Goals & Success Metrics'), findsOneWidget);
    expect(find.textContaining('(not yet written'), findsOneWidget);
    expect(find.textContaining('Latest Commit: v3'), findsOneWidget);
    expect(
      find.textContaining('the decisions the answer requires now'),
      findsOneWidget,
    );
    // The view response carries only the whys; the section text is the PRD's,
    // and one fetch of it feeds every card, not one per card.
    expect((session.api as _FakeApi).getPrdCalls, 1);
  });

  testWidgets(
    'audience view shows blocking gaps first and labels the rest as later',
    (tester) async {
      final session = Session();
      session.ready = true;
      session.api = _FakeApi({
        'prdId': 'p1',
        'title': 'Gap order',
        'projection': {
          'audience': 'decision',
          'purpose': 'why now',
          'sections': const <Map<String, String>>[],
        },
        'readiness': {
          'gaps': [
            {
              'section': 'goals',
              'priority': 'should_clarify_this_version',
              'reason': 'the investment and return being decided on',
            },
            {
              'section': 'stories',
              'priority': 'validate_later',
              'reason': 'useful eventually',
            },
            {
              'section': 'background',
              'priority': 'must_clarify_now',
              'reason': 'the why-this, why-now argument a decision maker needs',
            },
          ],
        },
      });
      await tester.binding.setSurfaceSize(const Size(1400, 900));
      addTearDown(() => tester.binding.setSurfaceSize(null));

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
          child: const MaterialApp(home: AudienceViewPage(prdId: 'p1')),
        ),
      );
      await tester.pumpAndSettle();

      expect(tester.takeException(), isNull);
      final blocking = tester.getTopLeft(
        find.textContaining('the why-this, why-now argument'),
      );
      final laterHeading = tester.getTopLeft(find.text('Later'));
      final should = tester.getTopLeft(
        find.textContaining('the investment and return being decided on'),
      );
      final validate = tester.getTopLeft(
        find.textContaining('useful eventually'),
      );
      expect(blocking.dy, lessThan(laterHeading.dy));
      expect(laterHeading.dy, lessThan(should.dy));
      expect(should.dy, lessThan(validate.dy));
      expect(find.textContaining('must_clarify_now'), findsNothing);
      expect(find.textContaining('should_clarify_this_version'), findsNothing);
      expect(find.textContaining('validate_later'), findsNothing);
    },
  );

  testWidgets('the zh audience view speaks the gap reasons, not the server '
      'English', (tester) async {
    // The server sends one fixed English sentence per gap (readiness.go); a
    // zh reader must get the translated reason, never a half-Chinese line.
    final session = Session();
    session.ready = true;
    session.api = _FakeApi({
      'prdId': 'p1',
      'title': 'Gap language',
      'projection': {
        'audience': 'decision',
        'purpose': 'why now',
        'sections': const <Map<String, String>>[],
      },
      'readiness': {
        'gaps': [
          {
            'section': 'background',
            'priority': 'must_clarify_now',
            'reason': 'the why-this, why-now argument a decision maker needs',
          },
          {
            'section': 'stories',
            'priority': 'validate_later',
            'reason': 'useful eventually',
          },
        ],
      },
    });
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
        child: const MaterialApp(home: AudienceViewPage(prdId: 'p1')),
      ),
    );
    await tester.pumpAndSettle();

    expect(tester.takeException(), isNull);
    expect(find.textContaining('背景与问题陈述'), findsOneWidget);
    expect(find.textContaining('决策者需要'), findsOneWidget);
    expect(find.textContaining('迟早有用'), findsOneWidget);
    expect(find.textContaining('decision maker needs'), findsNothing);
    expect(find.textContaining('useful eventually'), findsNothing);
  });
}
