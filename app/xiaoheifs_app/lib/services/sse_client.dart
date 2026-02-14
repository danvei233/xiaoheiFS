import 'dart:async';
import 'dart:convert';

import 'package:http/http.dart' as http;

class SseEvent {
  final String? id;
  final String? event;
  final String data;

  const SseEvent({required this.id, required this.event, required this.data});
}

class SseConnection {
  SseConnection._(this._close);

  final void Function() _close;

  void close() => _close();
}

class SseClient {
  static SseConnection connect(
    String url, {
    Map<String, String>? headers,
    required void Function(SseEvent event) onMessage,
    void Function(Object error)? onError,
    void Function()? onDone,
  }) {
    final client = http.Client();
    bool closed = false;
    StreamSubscription<String>? sub;
    final lines = <String>[];

    Future<void> start() async {
      try {
        final req = http.Request('GET', Uri.parse(url));
        req.headers.addAll({
          'Accept': 'text/event-stream',
          if (headers != null) ...headers,
        });
        final streamed = await client.send(req);
        final done = Completer<void>();
        sub = streamed.stream.transform(utf8.decoder).transform(const LineSplitter()).listen(
          (line) {
            if (closed) return;
            if (line.isEmpty) {
              final event = _parseEvent(lines);
              lines.clear();
              if (event != null) onMessage(event);
              return;
            }
            if (line.startsWith(':')) return;
            lines.add(line);
          },
          onError: (e) {
            if (!closed) onError?.call(e);
            if (!done.isCompleted) done.complete();
          },
          onDone: () {
            if (!done.isCompleted) done.complete();
          },
          cancelOnError: true,
        );
        await done.future;
      } catch (e) {
        if (!closed) onError?.call(e);
      } finally {
        if (!closed) onDone?.call();
      }
    }

    unawaited(start());

    return SseConnection._(() {
      closed = true;
      lines.clear();
      sub?.cancel();
      client.close();
    });
  }

  static SseEvent? _parseEvent(List<String> lines) {
    if (lines.isEmpty) return null;
    String? id;
    String? event;
    final dataLines = <String>[];

    for (final line in lines) {
      if (line.startsWith('id:')) {
        id = line.substring(3).trim();
      } else if (line.startsWith('event:')) {
        event = line.substring(6).trim();
      } else if (line.startsWith('data:')) {
        dataLines.add(line.substring(5).trim());
      }
    }
    if (dataLines.isEmpty) return null;
    return SseEvent(id: id, event: event, data: dataLines.join('\n'));
  }
}
