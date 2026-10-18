part of 'client.dart';

/// Server-sent-event verbs: Coach replies, canvas presence, roadmap progress.
mixin _StreamVerbs on _ApiCore {
  /// The browser's fetch transport, swapped by tests: the real one exists only
  /// on web (on the VM it throws). Setting it also selects the fetch path for
  /// the presence and progress streams, which `kIsWeb` selects in a browser.
  @visibleForTesting
  Future<SseAttempt> Function(
    String url,
    Map<String, String> headers, {
    Future<void>? stop,
  })?
  fetchAttemptForTests;

  /// A GET stream over the browser's fetch.
  Stream<List<int>> _fetchGetStream(
    String url,
    Map<String, String> headers,
    Future<void>? stop,
  ) async* {
    final fetch = fetchAttemptForTests ?? fetchSseAttempt;
    final attempt = await fetch(url, headers, stop: stop);
    if (attempt.status >= 400) {
      throw ApiException('stream failed', status: attempt.status);
    }
    yield* attempt.chunks;
  }

  /// Sends [content] and applies the reply's frames to [handlers]. A turn
  /// holds a model call open for as long as it runs, so completing [stop]
  /// aborts the connection and ends the call.
  Future<void> sendCoach(
    String conversationId,
    String content,
    CoachHandlers handlers, {
    Future<void>? stop,
  }) => _untilStopped(stop, (cancel) async {
    final token = await _readAccess();
    final open = coachSseOpen(
      baseUrl: baseUrl,
      conversationId: conversationId,
      content: content,
      accessToken: token,
    );
    Stream<List<int>> chunks;
    if (kIsWeb) {
      // USER-10: this raw fetch bypasses the dio interceptor that owns the
      // 401 refresh; the opener re-reads the access token, so the
      // post-refresh retry carries the rotated bearer automatically.
      chunks = coachSseWithRefresh(
        () async {
          final o = coachSseOpen(
            baseUrl: baseUrl,
            conversationId: conversationId,
            content: content,
            accessToken: await _readAccess(),
          );
          return fetchSseAttempt(
            o.url,
            o.headers,
            method: o.method,
            body: o.body,
            stop: stop,
          );
        },
        _refreshSingleFlight,
        _clearTokens,
      );
    } else {
      final res = await _dio.post<ResponseBody>(
        '/conversations/$conversationId/messages',
        data: {'content': content},
        options: Options(
          responseType: ResponseType.stream,
          headers: open.headers,
          // The body trickles for the whole model turn, and dio enforces
          // the shared 30s receiveTimeout between chunks — tighter than the
          // server's 90s provider silence and readCoachSse's 75s idle
          // timer. Turn it off for this request alone (zero = no limit) so
          // the idle timer governs silence; every other request keeps 30s.
          receiveTimeout: Duration.zero,
        ),
        cancelToken: cancel,
      );
      if (res.statusCode != null && res.statusCode! >= 400) {
        throw ApiException('coach failed', status: res.statusCode);
      }
      chunks = res.data!.stream;
    }
    await readCoachSse(chunks, handlers, idleTimeout: coachSseIdleTimeout);
  });

  /// Runs [session] with a token that completing [stop] trips. A stop is the
  /// caller leaving, so whatever the abort raises (dio's cancel, a closed
  /// socket, a rejected fetch) ends the call normally instead of failing it.
  Future<void> _untilStopped(
    Future<void>? stop,
    Future<void> Function(CancelToken cancel) session,
  ) async {
    final cancel = CancelToken();
    var stopped = false;
    stop?.then((_) {
      stopped = true;
      cancel.cancel();
    });
    try {
      await session(cancel);
    } catch (_) {
      if (!stopped) rethrow;
    }
  }

  /// Subscribes to the project's live presence (canvas who-is-here).
  /// Every snapshot replace-calls [onChange]; completing [stop] aborts the
  /// connection and ends the call.
  Future<void> presenceStream(
    String projectId,
    String clientId,
    void onChange(List<PresenceEntry> entries), {
    void Function(String sessionId, String name, double x, double y)? onCursor,
    Future<void>? stop,
  }) => _untilStopped(stop, (cancel) async {
    final token = await _readAccess();
    final headers = {
      if (token != null) 'Authorization': 'Bearer $token',
      'Accept': 'text/event-stream',
    };
    Stream<List<int>> chunks;
    if (kIsWeb || fetchAttemptForTests != null) {
      chunks = _fetchGetStream(
        '$baseUrl/projects/$projectId/presence/stream?client=${Uri.encodeQueryComponent(clientId)}',
        headers,
        stop,
      );
    } else {
      final res = await _dio.get<ResponseBody>(
        '/projects/$projectId/presence/stream',
        queryParameters: {'client': clientId},
        options: Options(responseType: ResponseType.stream, headers: headers),
        cancelToken: cancel,
      );
      if (res.statusCode != null && res.statusCode! >= 400) {
        throw ApiException('presence failed', status: res.statusCode);
      }
      chunks = res.data!.stream;
    }
    var buffer = '';
    await for (final text in decodeSseBytes(chunks)) {
      buffer += text;
      final parsed = parseSseBuffer(buffer);
      buffer = parsed.rest;
      for (final frame in parsed.frames) {
        if (frame.event == 'presence') {
          final list = jsonDecode(frame.data);
          onChange(
            (list is List ? list : const [])
                .map((e) => PresenceEntry.fromJson(e as Map<String, dynamic>))
                .toList(),
          );
        } else if (frame.event == 'cursor' && onCursor != null) {
          final m = jsonDecode(frame.data) as Map<String, dynamic>;
          onCursor(
            m['sessionId']?.toString() ?? '',
            m['name']?.toString() ?? '',
            (m['x'] as num?)?.toDouble() ?? 0,
            (m['y'] as num?)?.toDouble() ?? 0,
          );
        }
      }
    }
  });

  /// Subscribes to live progress snapshots. Every `progress` frame
  /// replace-calls [onProgress]; completing [stop] aborts the connection and
  /// ends the call.
  Future<void> progressStream(
    String projectId,
    void Function(RoadmapProgress progress) onProgress, {
    Future<void>? stop,
  }) => _untilStopped(stop, (cancel) async {
    final token = await _readAccess();
    final headers = {
      if (token != null) 'Authorization': 'Bearer $token',
      'Accept': 'text/event-stream',
    };
    Stream<List<int>> chunks;
    if (kIsWeb || fetchAttemptForTests != null) {
      chunks = _fetchGetStream(
        '$baseUrl/projects/$projectId/roadmap/progress/stream',
        headers,
        stop,
      );
    } else {
      final res = await _dio.get<ResponseBody>(
        '/projects/$projectId/roadmap/progress/stream',
        options: Options(responseType: ResponseType.stream, headers: headers),
        cancelToken: cancel,
      );
      if (res.statusCode != null && res.statusCode! >= 400) {
        throw ApiException('progress stream failed', status: res.statusCode);
      }
      chunks = res.data!.stream;
    }
    await readProgressSse(chunks, onProgress);
  });
}
