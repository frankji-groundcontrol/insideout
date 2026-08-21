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
  });

  final String id;
  final String email;
  final String username;
  final String bio;
  final List<String> keywords;
  final String? avatarUrl;
  final String? accessToken;
  final String? refreshToken;

  factory UserProfile.fromJson(Map<String, dynamic> j) => UserProfile(
        id: j['id'] as String,
        email: j['email'] as String,
        username: j['username'] as String,
        bio: (j['bio'] as String?) ?? '',
        keywords: ((j['keywords'] as List?) ?? const []).cast<String>(),
        avatarUrl: j['avatarUrl'] as String?,
        accessToken: j['accessToken'] as String?,
        refreshToken: j['refreshToken'] as String?,
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

class Project {
  Project({
    required this.id,
    required this.workspaceId,
    required this.title,
    required this.description,
    required this.status,
    required this.repoUrl,
    required this.createdAt,
    this.latestUpdateKind,
    this.latestUpdateContent,
  });

  final String id;
  final String workspaceId;
  final String title;
  final String description;
  final String status;
  final String repoUrl;
  final String createdAt;
  final String? latestUpdateKind;
  final String? latestUpdateContent;

  factory Project.fromJson(Map<String, dynamic> j) => Project(
        id: j['id'] as String,
        workspaceId: j['workspaceId'] as String,
        title: j['title'] as String,
        description: (j['description'] as String?) ?? '',
        status: j['status'] as String,
        repoUrl: (j['repoUrl'] as String?) ?? '',
        createdAt: j['createdAt'] as String,
        latestUpdateKind: j['latestUpdateKind'] as String?,
        latestUpdateContent: j['latestUpdateContent'] as String?,
      );
}

class ProjectUpdate {
  ProjectUpdate({
    required this.id,
    required this.kind,
    required this.content,
    required this.createdAt,
    required this.authorId,
  });

  final String id;
  final String kind;
  final String content;
  final String createdAt;
  final String authorId;

  factory ProjectUpdate.fromJson(Map<String, dynamic> j) => ProjectUpdate(
        id: j['id'] as String,
        kind: j['kind'] as String,
        content: j['content'] as String,
        createdAt: j['createdAt'] as String,
        authorId: j['authorId'] as String,
      );
}

class Idea {
  Idea({
    required this.id,
    required this.workspaceId,
    required this.title,
    required this.content,
    required this.status,
    required this.createdAt,
    this.prdId,
  });

  final String id;
  final String workspaceId;
  final String title;
  final String content;
  final String status;
  final String createdAt;
  final String? prdId;

  factory Idea.fromJson(Map<String, dynamic> j) => Idea(
        id: j['id'] as String,
        workspaceId: j['workspaceId'] as String,
        title: j['title'] as String,
        content: (j['content'] as String?) ?? '',
        status: j['status'] as String,
        createdAt: j['createdAt'] as String,
        prdId: j['prdId'] as String?,
      );
}

class Prd {
  Prd({
    required this.id,
    required this.workspaceId,
    required this.title,
    required this.sections,
    required this.status,
    required this.currentRevision,
    required this.updatedAt,
    this.authorId,
    this.projectId,
  });

  final String id;
  final String workspaceId;
  final String title;
  final Map<String, String> sections;
  final String status;
  final int currentRevision;
  final String updatedAt;
  final String? authorId;
  final String? projectId;

  factory Prd.fromJson(Map<String, dynamic> j) => Prd(
        id: j['id'] as String,
        workspaceId: j['workspaceId'] as String,
        title: j['title'] as String,
        sections: ((j['sections'] as Map?) ?? const {}).map(
          (k, v) => MapEntry(k.toString(), v?.toString() ?? ''),
        ),
        status: j['status'] as String,
        currentRevision: (j['currentRevision'] as num?)?.toInt() ?? 0,
        updatedAt: j['updatedAt'] as String,
        authorId: j['authorId'] as String?,
        projectId: j['projectId'] as String?,
      );
}

const prdSectionKeys = [
  'background',
  'users',
  'goals',
  'nonGoals',
  'stories',
  'requirements',
  'constraints',
  'risks',
];

class PrdRevision {
  PrdRevision({
    required this.id,
    required this.revision,
    required this.createdAt,
    this.note,
    this.sections = const {},
  });

  final String id;
  final int revision;
  final String createdAt;
  final String? note;
  final Map<String, String> sections;

  factory PrdRevision.fromJson(Map<String, dynamic> j) => PrdRevision(
        id: j['id'] as String,
        revision: (j['revision'] as num?)?.toInt() ?? 0,
        createdAt: j['createdAt'] as String,
        note: j['note'] as String?,
        sections: ((j['sections'] as Map?) ?? const {}).map(
          (k, v) => MapEntry(k.toString(), v?.toString() ?? ''),
        ),
      );
}

class RoadmapNode {
  RoadmapNode({
    required this.id,
    required this.projectId,
    required this.title,
    required this.description,
    required this.status,
    required this.position,
    this.parentId,
    this.creatorName,
    this.editorName,
  });

  final String id;
  final String projectId;
  final String? parentId;
  final String title;
  final String description;
  final String status;
  final int position;
  final String? creatorName;
  final String? editorName;

  factory RoadmapNode.fromJson(Map<String, dynamic> j) => RoadmapNode(
        id: j['id'] as String,
        projectId: j['projectId'] as String,
        parentId: j['parentId'] as String?,
        title: j['title'] as String,
        description: (j['description'] as String?) ?? '',
        status: j['status'] as String,
        position: (j['position'] as num?)?.toInt() ?? 0,
        creatorName: j['creatorName'] as String?,
        editorName: j['editorName'] as String?,
      );
}

class Conversation {
  Conversation({
    required this.id,
    required this.prdId,
    required this.stage,
    required this.status,
  });

  final String id;
  final String prdId;
  final String stage;
  final String status;

  factory Conversation.fromJson(Map<String, dynamic> j) => Conversation(
        id: j['id'] as String,
        prdId: j['prdId'] as String,
        stage: j['stage'] as String,
        status: j['status'] as String,
      );
}

class ChatMessage {
  ChatMessage({required this.id, required this.role, required this.content});

  final String id;
  final String role;
  final String content;

  factory ChatMessage.fromJson(Map<String, dynamic> j) => ChatMessage(
        id: j['id'] as String,
        role: j['role'] as String,
        content: j['content'] as String,
      );
}

class PrdCommit {
  PrdCommit({
    required this.id,
    required this.revision,
    required this.name,
    required this.primaryAudience,
    required this.createdAt,
    this.changeSummary = '',
    this.decisionNote = '',
    this.unresolved = const [],
    this.diffCounts = const {},
    this.diffSections = const {},
    this.committedByName,
  });

  final String id;
  final int revision;
  final String name;
  final String primaryAudience;
  final String createdAt;
  final String changeSummary;
  final String decisionNote;
  final List<String> unresolved;
  final Map<String, int> diffCounts;
  final Map<String, String> diffSections;
  final String? committedByName;

  factory PrdCommit.fromJson(Map<String, dynamic> j) {
    final diff = (j['diff'] as Map?) ?? const {};
    final counts = (diff['counts'] as Map?) ?? const {};
    final sections = (diff['sections'] as Map?) ?? const {};
    return PrdCommit(
      id: j['id'] as String,
      revision: (j['revision'] as num?)?.toInt() ?? 0,
      name: j['name'] as String,
      primaryAudience: j['primaryAudience'] as String,
      createdAt: j['createdAt'] as String,
      changeSummary: j['changeSummary']?.toString() ?? '',
      decisionNote: j['decisionNote']?.toString() ?? '',
      unresolved: ((j['unresolved'] as List?) ?? const []).map((e) => e.toString()).toList(),
      diffCounts: counts.map((k, v) => MapEntry(k.toString(), (v as num?)?.toInt() ?? 0)),
      diffSections: sections.map((k, v) {
        final change = (v as Map?)?['change']?.toString() ?? '';
        return MapEntry(k.toString(), change);
      }),
      committedByName: j['committedByName'] as String?,
    );
  }
}

class ReadinessGap {
  ReadinessGap({required this.section, required this.priority, required this.reason});

  final String section;
  final String priority;
  final String reason;
}

class AudienceReadiness {
  AudienceReadiness({required this.audience, required this.ready, required this.gaps, required this.carryIntoCommit});

  final String audience;
  final bool ready;
  final List<ReadinessGap> gaps;
  final List<String> carryIntoCommit;

  factory AudienceReadiness.fromJson(String audience, Map<String, dynamic> j) => AudienceReadiness(
        audience: audience,
        ready: j['ready'] as bool? ?? false,
        gaps: ((j['gaps'] as List?) ?? const [])
            .map((e) => ReadinessGap(
                  section: (e as Map)['section']?.toString() ?? '',
                  priority: e['priority']?.toString() ?? '',
                  reason: e['reason']?.toString() ?? '',
                ))
            .toList(),
        carryIntoCommit: ((j['carryIntoCommit'] as List?) ?? const []).map((e) => e.toString()).toList(),
      );
}

class PrdReadiness {
  PrdReadiness({required this.audiences});

  final Map<String, AudienceReadiness> audiences;

  factory PrdReadiness.fromJson(Map<String, dynamic> j) => PrdReadiness(
        audiences: ((j['audiences'] as Map?) ?? const {})
            .map((k, v) => MapEntry(k.toString(), AudienceReadiness.fromJson(k.toString(), v as Map<String, dynamic>))),
      );
}
