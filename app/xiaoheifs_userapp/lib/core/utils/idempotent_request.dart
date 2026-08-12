import 'package:dio/dio.dart';

class IdempotentAttempt<T> {
  const IdempotentAttempt({required this.key, required this.payload});

  final String key;
  final T payload;
}

class IdempotentRequest<T> {
  IdempotentRequest(this.prefix);

  static int _sequence = 0;

  final String prefix;
  IdempotentAttempt<T>? _pending;

  IdempotentAttempt<T> prepare(T Function() payloadFactory) {
    return _pending ??= IdempotentAttempt<T>(
      key: '$prefix-${DateTime.now().microsecondsSinceEpoch}-${_sequence++}',
      payload: payloadFactory(),
    );
  }

  void resolve() {
    _pending = null;
  }

  void reject(Object error) {
    final status = error is DioException ? error.response?.statusCode : null;
    final outcomeUnknown =
        error is DioException &&
        (error.response == null || (status != null && status >= 500));
    if (!outcomeUnknown) {
      resolve();
    }
  }
}
