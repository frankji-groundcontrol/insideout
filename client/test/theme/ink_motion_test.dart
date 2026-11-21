import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/theme/ink_motion.dart';

void main() {
  test('assemble-pop starts tiny and ends at 1', () {
    expect(assemblePopAt(0).opacity, 0);
    expect(assemblePopAt(0).scale, 0.25);
    expect(assemblePopAt(0.6).scale, closeTo(1.12, 0.001));
    expect(assemblePopAt(1).opacity, 1);
    expect(assemblePopAt(1).scale, closeTo(1, 0.001));
  });

  test('seal stamp overshoots then settles', () {
    expect(sealStampAt(0).scale, 1.6);
    expect(sealStampAt(0.55).scale, closeTo(0.88, 0.001));
    expect(sealStampAt(0.78).scale, closeTo(1.07, 0.001));
    expect(sealStampAt(1).scale, closeTo(1, 0.001));
    expect(sealStampAt(1).opacity, 1);
  });
}
