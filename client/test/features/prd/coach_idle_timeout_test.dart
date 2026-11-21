import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/sse.dart';

void main() {
  testWidgets('a one-second Timer elapses when the tester pumps', (
    tester,
  ) async {
    await tester.pumpWidget(const SizedBox());
    var fired = false;
    Timer(const Duration(seconds: 1), () => fired = true);
    await tester.pump(const Duration(seconds: 1));
    expect(fired, isTrue);
  });

  testWidgets('readCoachSse reports TimeoutException after the idle duration', (
    tester,
  ) async {
    await tester.pumpWidget(const SizedBox());
    final chunks = StreamController<List<int>>();
    Object? err;
    readCoachSse(
      chunks.stream,
      const CoachHandlers(),
      idleTimeout: const Duration(seconds: 1),
    ).then<void>((_) {}, onError: (Object e) => err = e);
    await tester.pump(const Duration(seconds: 1));
    await tester.pump();
    expect(err, isA<TimeoutException>());
    expect(coachSseIdleTimeout, const Duration(seconds: 75));
    await chunks.close();
  });
}
