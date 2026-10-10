part of 'client.dart';

typedef TokenReader = Future<String?> Function();
typedef TokenWriter = Future<void> Function(String access, String refresh);
typedef TokenClearer = Future<void> Function();

/// What every verb mixin builds on: construction, the Dio interceptors
/// (bearer attach, single-flight 401 refresh), `refresh()` itself, and the
/// `_exec` / `_parse` helpers.
abstract class _ApiCore {
  _ApiCore({
    required this.baseUrl,
    required TokenReader readAccess,
    required TokenReader readRefresh,
    required TokenWriter writeTokens,
    required TokenClearer clearTokens,
  }) : _readAccess = readAccess,
       _readRefresh = readRefresh,
       _writeTokens = writeTokens,
       _clearTokens = clearTokens {
    _dio = Dio(
      BaseOptions(
        baseUrl: baseUrl,
        connectTimeout: Duration(seconds: 10),
        receiveTimeout: Duration(seconds: 30),
        headers: {'Content-Type': 'application/json'},
        validateStatus: (s) => s != null && s < 500,
      ),
    );
    _dio.interceptors.add(
      InterceptorsWrapper(
        onRequest: (options, handler) async {
          final token = await _readAccess();
          if (token != null && token.isNotEmpty) {
            options.headers['Authorization'] = 'Bearer $token';
          }
          handler.next(options);
        },
        // validateStatus (<500) delivers 401s here as ordinary responses, so
        // the expired-token recovery lives in onResponse rather than onError:
        // refresh once, retry the request, and let a terminal 401 fall through
        // to _parse, which surfaces it as the same ApiException as before.
        onResponse: (response, handler) async {
          final options = response.requestOptions;
          // /auth/* endpoints establish sessions; a 401 there is terminal
          // (bad credentials, dead link) and must surface untouched.
          if (response.statusCode != 401 ||
              options.path.startsWith('/auth/') ||
              options.extra[_retriedKey] == true) {
            handler.next(response);
            return;
          }
          options.extra[_retriedKey] = true;
          try {
            await _refreshSingleFlight();
          } catch (_) {
            await _clearTokens();
            handler.next(response);
            return;
          }
          // The onRequest interceptor re-attaches the rotated bearer token.
          try {
            handler.resolve(await _dio.fetch<dynamic>(options));
          } on DioException catch (e) {
            handler.reject(e);
          }
        },
      ),
    );
  }

  static const _retriedKey = 'io_retried';

  Future<void>? _refreshing;

  /// Exactly one in-flight refresh no matter how many requests 401 at the
  /// same moment. A failed refresh clears the session so the app routes to
  /// the auth door instead of retrying with a dead token.
  Future<void> _refreshSingleFlight() =>
      _refreshing ??= refresh().whenComplete(() => _refreshing = null);

  final String baseUrl;
  final TokenReader _readAccess;
  final TokenReader _readRefresh;
  final TokenWriter _writeTokens;
  final TokenClearer _clearTokens;
  late final Dio _dio;

  /// Tests swap in a recording adapter to assert the merged per-request
  /// options (the coach stream's receiveTimeout override) without live
  /// sockets.
  @visibleForTesting
  Dio get dioForTests => _dio;

  Future<T> _parse<T>(Response res, T Function(dynamic) map) async {
    if (res.statusCode != null && res.statusCode! >= 400) {
      final body = res.data is Map<String, dynamic>
          ? res.data as Map<String, dynamic>
          : <String, dynamic>{'error': res.statusMessage};
      throw toApiError(res.statusCode, body);
    }
    return map(res.data);
  }

  Future<Response<dynamic>> _exec(req.ApiCall call) {
    return _dio.request<dynamic>(
      call.path,
      data: call.body,
      options: Options(method: call.method),
    );
  }

  Future<void> refresh() async {
    final refresh = await _readRefresh();
    if (refresh == null || refresh.isEmpty) {
      throw ApiException('no refresh token', status: 401);
    }
    final res = await _dio.post(
      '/auth/refresh',
      data: {'refreshToken': refresh},
    );
    final data = await _parse(res, (d) => d as Map<String, dynamic>);
    final pair = tokensFromRefreshBody(data);
    await _writeTokens(pair.access, pair.refresh);
  }
}
