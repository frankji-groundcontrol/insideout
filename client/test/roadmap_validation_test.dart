import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/features/project/roadmap_node_card.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

// FEAT-04 (review 3), the client half: a leaf carries a small form that
// records what real users did with the shipped work (a validation
// observation), a done leaf confirms validation with a reason, and the
// chip states the validated milestone with that reason.

class _ValidationApi extends ApiClient {
  _ValidationApi()
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  String? observedNode;
  String? observedDetail;
  String? observedSource;
  DateTime? observedAt;

  String? validatedNode;
  String? validateReason;

  @override
  Future<void> addValidationObservation(
    String nodeId, {
    required String detail,
    String source = '',
    required DateTime observedAt,
  }) async {
    observedNode = nodeId;
    observedDetail = detail;
    observedSource = source;
    this.observedAt = observedAt;
  }

  @override
  Future<RoadmapNode> validateRoadmapNode(String nodeId, String reason) async {
    validatedNode = nodeId;
    validateReason = reason;
    return _doneLeaf;
  }

  static final _doneLeaf = RoadmapNode(
    id: 'n1',
    projectId: 'p1',
    title: 'Tour',
    description: '',
    status: 'done',
    position: 0,
    validationReason: reason68,
  );

  static const reason68 = '7-day retention rose from 40% to 68%';
}

Future<void> _pumpCard(
  WidgetTester tester,
  ApiClient api,
  RoadmapNode node,
) async {
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
      ],
      child: MaterialApp(
        home: Scaffold(
          body: SingleChildScrollView(
            child: RoadmapNodeCard(
              projectId: 'p1',
              node: node,
              onReload: () async {},
            ),
          ),
        ),
      ),
    ),
  );
  await tester.pumpAndSettle();
}

RoadmapNode _leaf(String status) => RoadmapNode(
  id: 'n1',
  projectId: 'p1',
  parentId: 'root',
  title: 'Tour',
  description: '',
  status: status,
  position: 0,
);

void main() {
  testWidgets('a leaf records a typed observation as validation evidence', (
    tester,
  ) async {
    final api = _ValidationApi();
    await _pumpCard(tester, api, _leaf('done'));

    await tester.tap(find.byType(PopupMenuButton<String>));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Record observation'));
    await tester.pumpAndSettle();

    final fields = find.byType(TextField);
    await tester.enterText(
      fields.at(0),
      '4 of 5 pilot users finished the tour',
    );
    await tester.enterText(fields.at(1), 'pilot check-in thread');
    await tester.enterText(fields.at(2), '2026-10-02');
    await tester.tap(find.text('Record'));
    await tester.pumpAndSettle();

    expect(api.observedNode, 'n1');
    expect(api.observedDetail, '4 of 5 pilot users finished the tour');
    expect(api.observedSource, 'pilot check-in thread');
    expect(api.observedAt, DateTime(2026, 10, 2));
  });

  testWidgets('a done leaf confirms validation with a reason', (tester) async {
    final api = _ValidationApi();
    await _pumpCard(tester, api, _leaf('done'));

    await tester.tap(find.byType(PopupMenuButton<String>));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Mark validated'));
    await tester.pumpAndSettle();

    // A blank reason keeps the dialog open — the confirm must say why.
    await tester.tap(find.text('Confirm'));
    await tester.pumpAndSettle();
    expect(find.text('This field is required'), findsOneWidget);
    expect(api.validatedNode, isNull);

    await tester.enterText(find.byType(TextField), _ValidationApi.reason68);
    await tester.tap(find.text('Confirm'));
    await tester.pumpAndSettle();

    expect(api.validatedNode, 'n1');
    expect(api.validateReason, _ValidationApi.reason68);
  });

  testWidgets('a validated leaf shows the validated chip and its reason', (
    tester,
  ) async {
    final api = _ValidationApi();
    await _pumpCard(
      tester,
      api,
      RoadmapNode(
        id: 'n1',
        projectId: 'p1',
        parentId: 'root',
        title: 'Tour',
        description: '',
        status: 'validated',
        position: 0,
        validationReason: _ValidationApi.reason68,
      ),
    );

    expect(find.text('Validated'), findsOneWidget);
    expect(find.textContaining(_ValidationApi.reason68), findsOneWidget);
  });
}
