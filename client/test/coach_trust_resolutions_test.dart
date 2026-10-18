import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/features/prd/coach_trust_row.dart';
import 'package:insideout/session/appearance.dart';
import 'package:provider/provider.dart';

// rv4-AIDEV-03: resolve_finding only parks a status; the person's click on
// Confirm is what applies it. The card has to say what that click resolves or
// overrides, because an override is a waiver only a person may grant.
Future<void> _pump(
  WidgetTester tester,
  List<Map<String, dynamic>> resolutions, {
  String locale = 'en-US',
}) async {
  await tester.binding.setSurfaceSize(const Size(800, 1800));
  addTearDown(() => tester.binding.setSurfaceSize(null));
  await tester.pumpWidget(
    ChangeNotifierProvider(
      create: (_) => Appearance(store: MemorySettingsStore(), locale: locale),
      child: MaterialApp(
        home: Scaffold(
          body: CoachProposalRow(
            conversation: Conversation.fromJson({
              'id': 'c1',
              'prdId': 'prd-1',
              'stage': 'critique',
              'status': 'active',
              'pendingStage': 'finalize',
              'pendingResolutions': resolutions,
            }),
            busy: false,
            onConfirm: () async {},
            onDecline: () {},
          ),
        ),
      ),
    ),
  );
  await tester.pumpAndSettle();
}

const _resolved = {
  'id': 'cf1',
  'section': 'background',
  'severity': 'blocking',
  'issue': 'no problem stated',
  'status': 'resolved',
};
const _overridden = {
  'id': 'cf2',
  'section': 'goals',
  'severity': 'major',
  'issue': 'no goal given',
  'status': 'overridden',
};

void main() {
  testWidgets('the card says what confirming will resolve and override', (
    tester,
  ) async {
    await _pump(tester, [_resolved, _overridden]);

    expect(find.textContaining('marks this finding resolved'), findsOneWidget);
    expect(find.textContaining('no problem stated'), findsOneWidget);
    expect(find.textContaining('overrides this finding'), findsOneWidget);
    expect(find.textContaining('no goal given'), findsOneWidget);
    // Severity and the PRD section are words, not wire tokens.
    expect(find.textContaining('Blocking'), findsOneWidget);
    expect(find.textContaining('Goals & Success Metrics'), findsOneWidget);
    expect(find.textContaining('background'), findsNothing);
  });

  testWidgets('a proposal with nothing parked lists no findings', (
    tester,
  ) async {
    await _pump(tester, const []);

    expect(find.byKey(const Key('coach-stage-confirm')), findsOneWidget);
    expect(find.textContaining('this finding'), findsNothing);
  });

  testWidgets('the zh card carries the same lines in Chinese', (tester) async {
    await _pump(tester, [_resolved, _overridden], locale: 'zh-CN');

    expect(find.textContaining('已解决'), findsOneWidget);
    expect(find.textContaining('忽略'), findsOneWidget);
    expect(find.textContaining('no problem stated'), findsOneWidget);
  });
}
