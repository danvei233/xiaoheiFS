import 'package:dio/dio.dart';
import '../../../core/network/api_client.dart';
import '../../../core/constants/api_endpoints.dart';
import '../../models/wallet.dart';

/// 钱包API服务
class WalletApi {
  final Dio _dio = ApiClient.instance.dio;

  /// 获取钱包信息
  Future<Wallet> getWallet() async {
    final response = await _dio.get(ApiEndpoints.wallet);
    return Wallet.fromJson(response.data);
  }

  /// 获取交易记录
  Future<List<Transaction>> getTransactions({
    int? page,
    int? pageSize,
    String? type,
  }) async {
    final response = await _dio.get(
      ApiEndpoints.walletTransactions,
      queryParameters: {
        if (page != null) 'page': page,
        if (pageSize != null) 'page_size': pageSize,
        if (type != null) 'type': type,
      },
    );
    final list = response.data as List;
    return list.map((e) => Transaction.fromJson(e)).toList();
  }

  /// 创建充值订单
  Future<WalletOrder> recharge({
    required double amount,
    required String currency,
    String? note,
  }) async {
    final response = await _dio.post(
      ApiEndpoints.walletRecharge,
      data: {
        'amount': amount,
        'currency': currency,
        if (note != null) 'note': note,
      },
    );
    return WalletOrder.fromJson(response.data);
  }

  /// 创建提现订单
  Future<WalletOrder> withdraw({
    required double amount,
    required String currency,
    String? note,
  }) async {
    final response = await _dio.post(
      ApiEndpoints.walletWithdraw,
      data: {
        'amount': amount,
        'currency': currency,
        if (note != null) 'note': note,
      },
    );
    return WalletOrder.fromJson(response.data);
  }

  /// 获取钱包订单列表
  Future<List<WalletOrder>> getWalletOrders({
    int? page,
    int? pageSize,
    String? type,
  }) async {
    final response = await _dio.get(
      ApiEndpoints.walletOrders,
      queryParameters: {
        if (page != null) 'page': page,
        if (pageSize != null) 'page_size': pageSize,
        if (type != null) 'type': type,
      },
    );
    final list = response.data as List;
    return list.map((e) => WalletOrder.fromJson(e)).toList();
  }

  /// 获取支付方式列表
  Future<List<PaymentProvider>> getPaymentProviders() async {
    final response = await _dio.get(ApiEndpoints.paymentProviders);
    final list = response.data as List;
    return list.map((e) => PaymentProvider.fromJson(e)).toList();
  }
}
