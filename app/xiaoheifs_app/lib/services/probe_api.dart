import 'api_client.dart';

class ProbeApi {
  ProbeApi(this._client);

  final ApiClient _client;

  Future<ProbeListResponse> listProbes({
    String? keyword,
    String? status,
    int limit = 20,
    int offset = 0,
    int? timestamp,
  }) async {
    final query = <String, String>{
      'limit': '$limit',
      'offset': '$offset',
      if (keyword != null && keyword.isNotEmpty) 'keyword': keyword,
      if (status != null && status.isNotEmpty) 'status': status,
      if (timestamp != null) '_t': '$timestamp',
    };
    final resp = await _client.getJson('/admin/api/v1/probes', query: query);
    final items = (resp['items'] as List<dynamic>? ?? [])
        .map((e) => ProbeNode.fromJson(_asMap(e)))
        .toList();
    final total = _asInt(resp['total'], fallback: items.length);
    return ProbeListResponse(items: items, total: total);
  }

  Future<CreateProbeResult> createProbe({
    required String name,
    required String agentId,
    required String osType,
    required List<String> tags,
  }) async {
    final resp = await _client.postJson('/admin/api/v1/probes', body: {
      'name': name,
      'agent_id': agentId,
      'os_type': osType,
      'tags': tags,
    });
    return CreateProbeResult(
      probe: resp['probe'] == null ? null : ProbeNode.fromJson(_asMap(resp['probe'])),
      enrollToken: _asString(resp['enroll_token']),
    );
  }

  Future<ProbeDetailResult> getProbeDetail(
    int id, {
    bool refreshSnapshot = false,
    int? timestamp,
  }) async {
    final query = <String, String>{
      if (refreshSnapshot) 'refresh': '1',
      if (timestamp != null) '_t': '$timestamp',
    };
    final resp = await _client.getJson('/admin/api/v1/probes/$id', query: query);
    return ProbeDetailResult(
      probe: resp['probe'] == null ? null : ProbeNode.fromJson(_asMap(resp['probe'])),
      online: _asBool(resp['online']),
    );
  }

  Future<String?> resetEnrollToken(int id) async {
    final resp = await _client.postJson('/admin/api/v1/probes/$id/enroll-token/reset');
    return _asString(resp['enroll_token']);
  }

  Future<ProbeSla?> getProbeSla(
    int id, {
    int days = 7,
    int? timestamp,
  }) async {
    final query = <String, String>{
      'days': '$days',
      if (timestamp != null) '_t': '$timestamp',
    };
    final resp = await _client.getJson('/admin/api/v1/probes/$id/sla', query: query);
    final raw = resp['sla'];
    if (raw == null) return null;
    return ProbeSla.fromJson(_asMap(raw));
  }

  Future<ProbeLogSessionResult> createLogSession(
    int id, {
    required String source,
    required String keyword,
    required int lines,
    required bool follow,
  }) async {
    final resp = await _client.postJson('/admin/api/v1/probes/$id/log-sessions', body: {
      'source': source,
      'keyword': keyword,
      'lines': lines,
      'follow': follow,
    });
    return ProbeLogSessionResult(
      sessionId: _asString(resp['session_id']),
      streamPath: _asString(resp['stream_path']) ?? '',
      logSession: resp['log_session'] == null
          ? null
          : ProbeLogSession.fromJson(_asMap(resp['log_session'])),
    );
  }
}

class ProbeListResponse {
  final List<ProbeNode> items;
  final int total;

  const ProbeListResponse({required this.items, required this.total});
}

class CreateProbeResult {
  final ProbeNode? probe;
  final String? enrollToken;

  const CreateProbeResult({required this.probe, required this.enrollToken});
}

class ProbeDetailResult {
  final ProbeNode? probe;
  final bool? online;

  const ProbeDetailResult({required this.probe, required this.online});
}

class ProbeLogSessionResult {
  final String? sessionId;
  final String streamPath;
  final ProbeLogSession? logSession;

  const ProbeLogSessionResult({
    required this.sessionId,
    required this.streamPath,
    required this.logSession,
  });
}

class ProbeNode {
  final int id;
  final String name;
  final String agentId;
  final String status;
  final String osType;
  final List<String> tags;
  final String lastHeartbeatAt;
  final String lastSnapshotAt;
  final ProbeSnapshot snapshot;
  final String createdAt;
  final String updatedAt;

  const ProbeNode({
    required this.id,
    required this.name,
    required this.agentId,
    required this.status,
    required this.osType,
    required this.tags,
    required this.lastHeartbeatAt,
    required this.lastSnapshotAt,
    required this.snapshot,
    required this.createdAt,
    required this.updatedAt,
  });

  factory ProbeNode.fromJson(Map<String, dynamic> json) {
    final tagsRaw = json['tags'] ?? json['Tags'];
    return ProbeNode(
      id: _asInt(json['id'] ?? json['ID']),
      name: _asString(json['name'] ?? json['Name']) ?? '',
      agentId: _asString(json['agent_id'] ?? json['AgentID']) ?? '',
      status: (_asString(json['status'] ?? json['Status']) ?? '').toLowerCase(),
      osType: (_asString(json['os_type'] ?? json['OsType']) ?? '').toLowerCase(),
      tags: tagsRaw is List ? tagsRaw.map((e) => '$e').toList() : const [],
      lastHeartbeatAt: _asString(json['last_heartbeat_at'] ?? json['LastHeartbeatAt']) ?? '',
      lastSnapshotAt: _asString(json['last_snapshot_at'] ?? json['LastSnapshotAt']) ?? '',
      snapshot: ProbeSnapshot.fromJson(_asMap(json['snapshot'] ?? json['Snapshot'])),
      createdAt: _asString(json['created_at'] ?? json['CreatedAt']) ?? '',
      updatedAt: _asString(json['updated_at'] ?? json['UpdatedAt']) ?? '',
    );
  }
}

class ProbeSnapshot {
  final Map<String, dynamic> system;
  final Map<String, dynamic> cpu;
  final Map<String, dynamic> memory;
  final List<Map<String, dynamic>> disks;
  final List<Map<String, dynamic>> ports;
  final Map<String, dynamic> raw;

  const ProbeSnapshot({
    required this.system,
    required this.cpu,
    required this.memory,
    required this.disks,
    required this.ports,
    required this.raw,
  });

  factory ProbeSnapshot.fromJson(Map<String, dynamic> json) {
    final disks = (json['disks'] as List<dynamic>? ?? [])
        .map((e) => _asMap(e))
        .toList();
    final ports = (json['ports'] as List<dynamic>? ?? [])
        .map((e) => _asMap(e))
        .toList();
    return ProbeSnapshot(
      system: _asMap(json['system']),
      cpu: _asMap(json['cpu']),
      memory: _asMap(json['memory']),
      disks: disks,
      ports: ports,
      raw: _asMap(json['raw']),
    );
  }
}

class ProbeSla {
  final String windowFrom;
  final String windowTo;
  final int totalSeconds;
  final int onlineSeconds;
  final double uptimePercent;
  final List<ProbeStatusEvent> events;

  const ProbeSla({
    required this.windowFrom,
    required this.windowTo,
    required this.totalSeconds,
    required this.onlineSeconds,
    required this.uptimePercent,
    required this.events,
  });

  factory ProbeSla.fromJson(Map<String, dynamic> json) {
    final events = (json['events'] as List<dynamic>? ?? [])
        .map((e) => ProbeStatusEvent.fromJson(_asMap(e)))
        .toList();
    return ProbeSla(
      windowFrom: _asString(json['window_from'] ?? json['WindowFrom']) ?? '',
      windowTo: _asString(json['window_to'] ?? json['WindowTo']) ?? '',
      totalSeconds: _asInt(json['total_seconds'] ?? json['TotalSeconds']),
      onlineSeconds: _asInt(json['online_seconds'] ?? json['OnlineSeconds']),
      uptimePercent: _asDouble(json['uptime_percent'] ?? json['UptimePercent']),
      events: events,
    );
  }
}

class ProbeStatusEvent {
  final int id;
  final int probeId;
  final String status;
  final String at;
  final String reason;
  final String createdAt;

  const ProbeStatusEvent({
    required this.id,
    required this.probeId,
    required this.status,
    required this.at,
    required this.reason,
    required this.createdAt,
  });

  factory ProbeStatusEvent.fromJson(Map<String, dynamic> json) {
    return ProbeStatusEvent(
      id: _asInt(json['id'] ?? json['ID']),
      probeId: _asInt(json['probe_id'] ?? json['ProbeID']),
      status: _asString(json['status'] ?? json['Status']) ?? '',
      at: _asString(json['at'] ?? json['At']) ?? '',
      reason: _asString(json['reason'] ?? json['Reason']) ?? '',
      createdAt: _asString(json['created_at'] ?? json['CreatedAt']) ?? '',
    );
  }
}

class ProbeLogSession {
  final int id;
  final int probeId;
  final int operatorId;
  final String source;
  final String status;
  final String startedAt;
  final String endedAt;
  final String createdAt;

  const ProbeLogSession({
    required this.id,
    required this.probeId,
    required this.operatorId,
    required this.source,
    required this.status,
    required this.startedAt,
    required this.endedAt,
    required this.createdAt,
  });

  factory ProbeLogSession.fromJson(Map<String, dynamic> json) {
    return ProbeLogSession(
      id: _asInt(json['id'] ?? json['ID']),
      probeId: _asInt(json['probe_id'] ?? json['ProbeID']),
      operatorId: _asInt(json['operator_id'] ?? json['OperatorID']),
      source: _asString(json['source'] ?? json['Source']) ?? '',
      status: _asString(json['status'] ?? json['Status']) ?? '',
      startedAt: _asString(json['started_at'] ?? json['StartedAt']) ?? '',
      endedAt: _asString(json['ended_at'] ?? json['EndedAt']) ?? '',
      createdAt: _asString(json['created_at'] ?? json['CreatedAt']) ?? '',
    );
  }
}

Map<String, dynamic> _asMap(dynamic value) {
  if (value is Map<String, dynamic>) return value;
  if (value is Map) return value.cast<String, dynamic>();
  return <String, dynamic>{};
}

String? _asString(dynamic value) {
  if (value == null) return null;
  final text = value.toString();
  return text;
}

int _asInt(dynamic value, {int fallback = 0}) {
  if (value is int) return value;
  if (value is num) return value.toInt();
  if (value is String) return int.tryParse(value) ?? fallback;
  return fallback;
}

double _asDouble(dynamic value) {
  if (value is double) return value;
  if (value is num) return value.toDouble();
  if (value is String) return double.tryParse(value) ?? 0;
  return 0;
}

bool? _asBool(dynamic value) {
  if (value is bool) return value;
  if (value is String) {
    final v = value.toLowerCase();
    if (v == 'true' || v == '1') return true;
    if (v == 'false' || v == '0') return false;
  }
  return null;
}
