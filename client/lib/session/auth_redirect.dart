const publicAuthLocations = {'/', '/login', '/register'};

/// Shipped auth gate used by [buildRouter]. Returns a new location or null.
String? authRedirect({
  required bool ready,
  required bool signedIn,
  required String location,
}) {
  if (!ready) return null;
  if (!signedIn && !publicAuthLocations.contains(location)) return '/login';
  if (signedIn && (location == '/login' || location == '/register')) {
    return '/dashboard';
  }
  return null;
}
