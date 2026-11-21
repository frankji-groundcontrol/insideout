import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/requests.dart';
import 'package:insideout/features/project/project_github_card.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

// rv5-PARB-11: the GitHub card hands over the insideout.yaml guide that makes
// commits attach to roadmap leaves; where no file can be saved, it is copied.
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
  Future<String> getGuide(String projectId) async => 'version: 1\nnodes: {}\n';
}

void main() {
  test('the guide request reads /projects/{id}/guide', () {
    final c = getProjectGuide('p1');
    expect(c.method, 'GET');
    expect(c.path, '/projects/p1/guide');
  });

  testWidgets('the guide button copies insideout.yaml off the web', (
    tester,
  ) async {
    final copied = <String>[];
    final messenger = tester.binding.defaultBinaryMessenger;
    messenger.setMockMethodCallHandler(SystemChannels.platform, (call) async {
      if (call.method == 'Clipboard.setData') {
        copied.add((call.arguments as Map)['text'] as String);
      }
      return null;
    });
    addTearDown(
      () => messenger.setMockMethodCallHandler(SystemChannels.platform, null),
    );
    final session = Session()
      ..ready = true
      ..api = _Api();
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
            body: ProjectGithubCard(
              projectId: 'p1',
              repo: TextEditingController(),
              onChanged: () async {},
            ),
          ),
        ),
      ),
    );
    await tester.tap(find.text('Get insideout.yaml'));
    await tester.pumpAndSettle();
    expect(copied, ['version: 1\nnodes: {}\n']);
    expect(find.textContaining('insideout.yaml copied'), findsOneWidget);
  });
}
