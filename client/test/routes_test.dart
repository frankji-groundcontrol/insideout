import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/router.dart';

void main() {
  test('productRouteBuilders registers every current surface', () {
    expect(
      productRouteBuilders.keys.toSet(),
      {
        '/',
        '/login',
        '/register',
        '/dashboard',
        '/profile',
        '/workspace/:id',
        '/workspace/:id/ideas',
        '/workspace/:id/settings',
        '/projects/:id',
        '/projects/:id/roadmap',
        '/prd/:id',
        '/prd/:id/revisions',
        '/prd/:id/export',
      },
    );
  });
}
