import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:opensource_userapp/core/utils/idempotent_request.dart';

void main() {
  group('IdempotentRequest', () {
    test('reuses the key and payload while the outcome is unknown', () {
      final request = IdempotentRequest<Map<String, Object?>>('order');
      final first = request.prepare(() => {'qty': 1});

      request.reject(
        DioException(
          requestOptions: RequestOptions(path: '/orders/items'),
          type: DioExceptionType.connectionError,
        ),
      );
      final retry = request.prepare(() => {'qty': 2});

      expect(retry.key, first.key);
      expect(retry.payload, first.payload);
    });

    test('creates a new identity after a definitive response', () {
      final request = IdempotentRequest<Map<String, Object?>>('pay');
      final first = request.prepare(() => {'amount': 1});
      final options = RequestOptions(path: '/orders/1/payments');

      request.reject(
        DioException(
          requestOptions: options,
          response: Response<void>(requestOptions: options, statusCode: 400),
          type: DioExceptionType.badResponse,
        ),
      );
      final next = request.prepare(() => {'amount': 2});

      expect(next.key, isNot(first.key));
      expect(next.payload, {'amount': 2});
    });

    test('reuses the identity after an uncertain server error', () {
      final request = IdempotentRequest<Map<String, Object?>>('order');
      final first = request.prepare(() => {'qty': 1});
      final options = RequestOptions(path: '/orders/items');

      request.reject(
        DioException(
          requestOptions: options,
          response: Response<void>(requestOptions: options, statusCode: 503),
          type: DioExceptionType.badResponse,
        ),
      );
      final retry = request.prepare(() => {'qty': 2});

      expect(retry.key, first.key);
      expect(retry.payload, first.payload);
    });

    test('creates a new identity after success', () {
      final request = IdempotentRequest<Object?>('order');
      final first = request.prepare(() => null);

      request.resolve();
      final next = request.prepare(() => null);

      expect(next.key, isNot(first.key));
    });
  });
}
