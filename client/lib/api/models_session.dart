class UserProfile {
  UserProfile({
    required this.id,
    required this.email,
    required this.username,
    required this.bio,
    required this.keywords,
    this.avatarUrl,
    this.accessToken,
    this.refreshToken,
    this.systemRole,
  });

  final String id;
  final String email;
  final String username;
  final String bio;
  final List<String> keywords;
  final String? avatarUrl;
  final String? accessToken;
  final String? refreshToken;
  final String? systemRole;

  bool get isAdmin => systemRole == 'admin';

  factory UserProfile.fromJson(Map<String, dynamic> j) => UserProfile(
    id: j['id'] as String,
    email: j['email'] as String,
    username: j['username'] as String,
    bio: (j['bio'] as String?) ?? '',
    keywords: ((j['keywords'] as List?) ?? const []).cast<String>(),
    avatarUrl: j['avatarUrl'] as String?,
    accessToken: j['accessToken'] as String?,
    refreshToken: j['refreshToken'] as String?,
    systemRole: j['systemRole'] as String?,
  );
}

class Workspace {
  Workspace({
    required this.id,
    required this.title,
    required this.description,
    required this.code,
    required this.status,
    required this.memberCount,
    required this.myRole,
    required this.createdAt,
  });

  final String id;
  final String title;
  final String description;
  final String code;
  final String status;
  final int memberCount;
  final String myRole;
  final String createdAt;

  factory Workspace.fromJson(Map<String, dynamic> j) => Workspace(
    id: j['id'] as String,
    title: j['title'] as String,
    description: (j['description'] as String?) ?? '',
    code: j['code'] as String,
    status: j['status'] as String,
    memberCount: (j['memberCount'] as num?)?.toInt() ?? 0,
    myRole: j['myRole'] as String,
    createdAt: j['createdAt'] as String,
  );
}

class WorkspaceMember {
  WorkspaceMember({
    required this.userId,
    required this.username,
    required this.email,
    required this.role,
  });

  final String userId;
  final String username;
  final String email;
  final String role;

  factory WorkspaceMember.fromJson(Map<String, dynamic> j) => WorkspaceMember(
    userId: j['userId'] as String,
    username: j['username'] as String,
    email: j['email'] as String,
    role: j['role'] as String,
  );
}
