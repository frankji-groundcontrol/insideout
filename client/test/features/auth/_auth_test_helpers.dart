import 'package:insideout/api/client.dart';

/// An ApiClient for tests that mount the signed-out forms. They ask the public
/// config when they open, and a real client would leave a request (and dio's
/// timer) pending when the test ends. Mail counts as delivered here, so no
/// notice appears.
class QuietAuthApi extends ApiClient {
  QuietAuthApi()
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  @override
  Future<String> authConfig() async => 'invitation_only';

  @override
  Future<bool> authMailEnabled() async => true;
}
