import 'package:go_router/go_router.dart';

import 'features/auth/login_page.dart';
import 'features/auth/register_page.dart';
import 'features/dashboard/dashboard_page.dart';
import 'features/landing/landing_page.dart';
import 'features/prd/export_page.dart';
import 'features/prd/prd_page.dart';
import 'features/prd/revisions_page.dart';
import 'features/prd/audience_view_page.dart';
import 'features/prd/versions_page.dart';
import 'features/profile/profile_page.dart';
import 'features/project/project_page.dart';
import 'features/project/roadmap_page.dart';
import 'features/workspace/ideas_page.dart';
import 'features/workspace/settings_page.dart';
import 'features/workspace/workspace_page.dart';
import 'session/auth_redirect.dart';
import 'session/session.dart';

/// Current product surfaces. [buildRouter] registers each path from this map.
final productRouteBuilders = <String, GoRouterWidgetBuilder>{
  '/': (c, s) => const LandingPage(),
  '/login': (c, s) => const LoginPage(),
  '/register': (c, s) => const RegisterPage(),
  '/dashboard': (c, s) => const DashboardPage(),
  '/profile': (c, s) => const ProfilePage(),
  '/workspace/:id': (c, s) => WorkspacePage(id: s.pathParameters['id']!),
  '/workspace/:id/ideas': (c, s) => IdeasPage(workspaceId: s.pathParameters['id']!),
  '/workspace/:id/settings': (c, s) => SettingsPage(workspaceId: s.pathParameters['id']!),
  '/projects/:id': (c, s) => ProjectPage(id: s.pathParameters['id']!),
  '/projects/:id/roadmap': (c, s) => RoadmapPage(projectId: s.pathParameters['id']!),
  '/prd/:id': (c, s) => PrdPage(id: s.pathParameters['id']!),
  '/prd/:id/revisions': (c, s) => RevisionsPage(prdId: s.pathParameters['id']!),
  '/prd/:id/versions': (c, s) => VersionsPage(prdId: s.pathParameters['id']!),
  '/prd/:id/view': (c, s) => AudienceViewPage(prdId: s.pathParameters['id']!),
  '/prd/:id/export': (c, s) => ExportPage(prdId: s.pathParameters['id']!),
};

Iterable<String> get productRoutePaths => productRouteBuilders.keys;

GoRouter buildRouter(Session session) {
  return GoRouter(
    initialLocation: '/',
    refreshListenable: session,
    redirect: (context, state) => authRedirect(
      ready: session.ready,
      signedIn: session.isSignedIn,
      location: state.matchedLocation,
    ),
    routes: [
      for (final e in productRouteBuilders.entries) GoRoute(path: e.key, builder: e.value),
    ],
  );
}
