import type { LocaleCode } from './messages';
export const shellKeys = ['shell.user', 'shell.search', 'shell.noResults', 'shell.production', 'shell.notifications', 'shell.markAllRead', 'shell.unread', 'shell.read', 'shell.all', 'shell.noNotifications', 'shell.logout', 'shell.toggleMenu'] as const;
export const shellTranslations: Record<LocaleCode, readonly string[]> = {
  'zh-CN': ['用户','搜索功能、页面…','未找到相关内容','生产环境','消息中心','全部标为已读','未读','已读','全部','暂无消息','退出登录','展开或收起菜单'],
  'zh-TW': ['使用者','搜尋功能、頁面…','找不到相關內容','正式環境','訊息中心','全部標示為已讀','未讀','已讀','全部','暫無訊息','登出','展開或收合選單'],
  'en-US': ['User','Search features and pages…','No results found','Production','Notifications','Mark all as read','Unread','Read','All','No notifications','Sign out','Expand or collapse menu'],
  'ja-JP': ['ユーザー','機能やページを検索…','検索結果がありません','本番環境','通知','すべて既読にする','未読','既読','すべて','通知はありません','ログアウト','メニューを開閉'],
  'ko-KR': ['사용자','기능 및 페이지 검색…','검색 결과가 없습니다','운영 환경','알림','모두 읽음으로 표시','읽지 않음','읽음','전체','알림이 없습니다','로그아웃','메뉴 펼치기 또는 접기'],
  'fr-FR': ['Utilisateur','Rechercher des fonctions et des pages…','Aucun résultat','Production','Notifications','Tout marquer comme lu','Non lues','Lues','Toutes','Aucune notification','Déconnexion','Déplier ou replier le menu'],
  'de-DE': ['Benutzer','Funktionen und Seiten suchen…','Keine Ergebnisse gefunden','Produktivbetrieb','Benachrichtigungen','Alle als gelesen markieren','Ungelesen','Gelesen','Alle','Keine Benachrichtigungen','Abmelden','Menü ein- oder ausklappen'],
  'es-ES': ['Usuario','Buscar funciones y páginas…','No se encontraron resultados','Producción','Notificaciones','Marcar todas como leídas','Sin leer','Leídas','Todas','No hay notificaciones','Cerrar sesión','Expandir o contraer el menú'],
  'it-IT': ['Utente','Cerca funzioni e pagine…','Nessun risultato trovato','Produzione','Notifiche','Segna tutte come lette','Non lette','Lette','Tutte','Nessuna notifica','Esci','Espandi o comprimi il menu'],
  'pt-PT': ['Utilizador','Pesquisar funções e páginas…','Nenhum resultado encontrado','Produção','Notificações','Marcar todas como lidas','Não lidas','Lidas','Todas','Sem notificações','Terminar sessão','Expandir ou recolher o menu'],
  'ru-RU': ['Пользователь','Поиск функций и страниц…','Ничего не найдено','Рабочая среда','Уведомления','Отметить все как прочитанные','Непрочитанные','Прочитанные','Все','Нет уведомлений','Выйти','Развернуть или свернуть меню'],
  'ar-SA': ['المستخدم','البحث عن الوظائف والصفحات…','لم يتم العثور على نتائج','بيئة الإنتاج','الإشعارات','تحديد الكل كمقروء','غير مقروءة','مقروءة','الكل','لا توجد إشعارات','تسجيل الخروج','توسيع القائمة أو طيها']
};
export const shellMessages = Object.fromEntries(Object.entries(shellTranslations).map(([code, values]) => [code, Object.fromEntries(shellKeys.map((key, index) => [key, values[index]]))])) as Record<LocaleCode, Record<string, string>>;
