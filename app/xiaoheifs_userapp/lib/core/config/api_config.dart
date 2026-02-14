/// API闁板秶鐤嗙猾?
/// 缁狅紕鎮夾PI閸╄櫣顢匲RL閸滃瞼娴夐崗鎶藉帳缂?
class ApiConfig {
  /// 姒涙顓籄PI URL
  static const String defaultUrl = 'https://api.example.com';

  /// 瑜版挸澧燗PI URL閿涘牆褰查悽杈╂暏閹村嘲婀惂璇茬秿妞ょ敻鍘ょ純顕嗙礆
  static String _baseUrl = defaultUrl;

  /// 閼惧嘲褰囪ぐ鎾冲API URL
  static String get baseUrl => _baseUrl;

  /// 鐠佸墽鐤咥PI URL閿涘牏鏁ゆ禍搴ｆ暏閹寸柉鍤滅€规矮绠熼張宥呭閸ｃ劌婀撮崸鈧敍?
  static void setBaseUrl(String url) {
    _baseUrl = url;
  }

  /// 闁插秶鐤嗘稉娲帛鐠侇椈RL
  static void reset() {
    _baseUrl = defaultUrl;
  }

  /// 閼惧嘲褰囩€瑰本鏆ｉ惃鍑橮I缁旑垳鍋RL
  static String getEndpoint(String path) {
    return '$baseUrl$path';
  }
}
