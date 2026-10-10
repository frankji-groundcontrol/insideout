import 'dart:convert';

import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';

import 'errors.dart';
import 'export_format.dart';
import 'models.dart';
import 'requests.dart' as req;
import 'sse.dart';
import 'sse_transport_stub.dart'
    if (dart.library.ui_web) 'sse_transport_web.dart';
import '../session/token_store.dart';

part 'client_core.dart';
part 'client_auth.dart';
part 'client_workspace.dart';
part 'client_project.dart';
part 'client_prd.dart';
part 'client_stream.dart';

/// The one HTTP client the app talks through. Its verbs live in per-resource
/// mixins (one `part` file each), not extensions, so they stay ordinary
/// instance members that test subclasses can override.
// ignore: library_private_types_in_public_api
class ApiClient = _ApiCore
    with _AuthVerbs, _WorkspaceVerbs, _ProjectVerbs, _PrdVerbs, _StreamVerbs;
