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

class Conversation {
  Conversation({
    required this.id,
    required this.prdId,
    required this.stage,
    required this.status,
    this.pendingStage,
    this.criticSkipped,
    this.criticRounds = 0,
    this.pendingResolutions = const [],
  });

  final String id;
  final String prdId;
  final String stage;
  final String status;

  /// Why a critic pass did not run (contention, critic-output-failure), when
  /// one was skipped; [criticRounds] is how many did. A skip opens the finalize
  /// gate, so the confirm card says so (rv4-AIDEV-02).
  final String? criticSkipped;
  final int criticRounds;

  /// The finding resolutions the coach parked: confirming the stage applies
  /// them, so the confirm card lists them (rv4-AIDEV-03).
  final List<PendingResolution> pendingResolutions;

  /// The coach's stage proposal, parked by the server until the member
  /// confirms it (POST /conversations/{id}/stage). It only reaches the
  /// client while the conversation view carries it.
  final String? pendingStage;

  /// A proposal clear is not a proposal keep: [pendingStage] takes a
  /// sentinel so `copyWith(pendingStage: null)` really clears it.
  static const _unset = Object();

  Conversation copyWith({String? stage, Object? pendingStage = _unset}) =>
      Conversation(
        id: id,
        prdId: prdId,
        stage: stage ?? this.stage,
        status: status,
        pendingStage: pendingStage == _unset
            ? this.pendingStage
            : pendingStage as String?,
        criticSkipped: criticSkipped,
        criticRounds: criticRounds,
        pendingResolutions: pendingResolutions,
      );

  factory Conversation.fromJson(Map<String, dynamic> j) => Conversation(
    id: j['id'] as String,
    prdId: j['prdId'] as String,
    stage: j['stage'] as String,
    status: j['status'] as String,
    pendingStage: j['pendingStage'] as String?,
    criticSkipped: j['criticSkipped'] as String?,
    criticRounds: (j['criticRounds'] as num?)?.toInt() ?? 0,
    pendingResolutions: [
      for (final r in (j['pendingResolutions'] as List?) ?? const [])
        PendingResolution.fromJson(r as Map<String, dynamic>),
    ],
  );
}

/// One resolve_finding request the coach parked, joined to its finding.
/// [status] (resolved | overridden) is what Confirm will set.
class PendingResolution {
  PendingResolution({
    required this.section,
    required this.severity,
    required this.issue,
    required this.status,
  });

  final String section;
  final String severity;
  final String issue;
  final String status;

  factory PendingResolution.fromJson(Map<String, dynamic> j) =>
      PendingResolution(
        section: j['section'] as String? ?? '',
        severity: j['severity'] as String? ?? '',
        issue: j['issue'] as String? ?? '',
        status: j['status'] as String? ?? '',
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
      unresolved: ((j['unresolved'] as List?) ?? const [])
          .map((e) => e.toString())
          .toList(),
      diffCounts: counts.map(
        (k, v) => MapEntry(k.toString(), (v as num?)?.toInt() ?? 0),
      ),
      diffSections: sections.map((k, v) {
        final change = (v as Map?)?['change']?.toString() ?? '';
        return MapEntry(k.toString(), change);
      }),
      committedByName: j['committedByName'] as String?,
    );
  }
}

class ReadinessGap {
  ReadinessGap({
    required this.section,
    required this.priority,
    required this.reason,
  });

  final String section;
  final String priority;
  final String reason;
}

class AudienceReadiness {
  AudienceReadiness({
    required this.audience,
    required this.ready,
    required this.gaps,
    required this.carryIntoCommit,
  });

  final String audience;
  final bool ready;
  final List<ReadinessGap> gaps;
  final List<String> carryIntoCommit;

  factory AudienceReadiness.fromJson(String audience, Map<String, dynamic> j) =>
      AudienceReadiness(
        audience: audience,
        ready: j['ready'] as bool? ?? false,
        gaps: ((j['gaps'] as List?) ?? const [])
            .map(
              (e) => ReadinessGap(
                section: (e as Map)['section']?.toString() ?? '',
                priority: e['priority']?.toString() ?? '',
                reason: e['reason']?.toString() ?? '',
              ),
            )
            .toList(),
        carryIntoCommit: ((j['carryIntoCommit'] as List?) ?? const [])
            .map((e) => e.toString())
            .toList(),
      );
}

class PrdReadiness {
  PrdReadiness({required this.audiences});

  final Map<String, AudienceReadiness> audiences;

  factory PrdReadiness.fromJson(Map<String, dynamic> j) => PrdReadiness(
    audiences: ((j['audiences'] as Map?) ?? const {}).map(
      (k, v) => MapEntry(
        k.toString(),
        AudienceReadiness.fromJson(k.toString(), v as Map<String, dynamic>),
      ),
    ),
  );
}
