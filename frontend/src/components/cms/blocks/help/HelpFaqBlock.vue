<template>
  <section class="faq-section">
    <div class="section-header">
      <h2 class="section-title">{{ resolved.title }}</h2>
      <p class="section-subtitle">{{ resolved.subtitle }}</p>
    </div>

    <!-- Category Tabs -->
    <div class="category-tabs">
      <button
        v-for="category in resolved.categories"
        :key="category.key"
        :class="['tab-button', { active: activeCategory === category.key }]"
        @click="activeCategory = category.key"
      >
        <component :is="category.icon" class="tab-icon" />
        <span class="tab-label">{{ category.label }}</span>
      </button>
    </div>

    <!-- FAQ Accordion -->
    <div class="faq-accordion">
      <div
        v-for="(faq, index) in filteredFAQs"
        :key="index"
        :class="['faq-item', { active: openFAQ === index }]"
        @click="toggleFAQ(index)"
      >
        <div class="faq-question">
          <div class="question-content">
            <span class="question-icon">Q</span>
            <span class="question-text">{{ faq.question }}</span>
          </div>
          <span class="faq-toggle">{{ openFAQ === index ? '−' : '+' }}</span>
        </div>
        <div class="faq-answer" :class="{ expanded: openFAQ === index }">
          <div class="answer-content">
            <span class="answer-icon">A</span>
            <p class="answer-text">{{ faq.answer }}</p>
          </div>
        </div>
      </div>

      <!-- No Results -->
      <div v-if="filteredFAQs.length === 0" class="no-results">
        <SearchOutlined class="no-results-icon" />
        <p>未找到相关问题</p>
        <button class="clear-search-btn" @click="$emit('clear-search')">清除搜索</button>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import {
  SearchOutlined,
  FileTextOutlined,
  UserOutlined,
  CreditCardOutlined,
  DesktopOutlined,
  AccountBookOutlined,
} from "@ant-design/icons-vue";

type FAQ = { category: string; question: string; answer: string };
type Category = { key: string; label: string; icon: any };

const props = defineProps<{
  content?: any;
  searchQuery: string;
}>();

defineEmits<{
  (e: "clear-search"): void;
}>();

const activeCategory = ref<string>("all");
const openFAQ = ref<number | null>(null);

const fallbackCategories: Category[] = [
  { key: "all", label: "全部", icon: FileTextOutlined },
  { key: "account", label: "账号相关", icon: UserOutlined },
  { key: "payment", label: "支付问题", icon: CreditCardOutlined },
  { key: "vps", label: "VPS使用", icon: DesktopOutlined },
  { key: "billing", label: "账单退款", icon: AccountBookOutlined },
];

const fallbackFAQs: FAQ[] = [
  { category: "account", question: "如何注册账号？", answer: '点击页面右上角的"注册"按钮，填写用户名、邮箱和密码即可完成注册。注册后需要验证邮箱才能使用全部功能。' },
  { category: "account", question: "忘记密码怎么办？", answer: '点击登录页面的"忘记密码"链接，输入您的注册邮箱，我们会发送密码重置链接到您的邮箱。' },
  { category: "account", question: "如何修改个人资料？", answer: '登录后进入控制台，点击右上角的用户头像，选择"个人资料"，即可修改您的基本信息、联系方式等。' },
  { category: "account", question: "如何开启二次验证？", answer: "在控制台的\"安全设置\"中，可以开启两步验证功能，支持验证器应用（如Google Authenticator）提高账户安全性。" },

  { category: "payment", question: "支持哪些支付方式？", answer: "我们支持支付宝、微信支付、银行卡等多种支付方式。企业用户还可以申请对公转账和发票服务。" },
  { category: "payment", question: "支付失败怎么办？", answer: "如果支付失败，请先检查账户余额是否充足。如问题仍未解决，请联系客服并提供订单号，我们会协助您处理。" },
  { category: "payment", question: "可以申请发票吗？", answer: "可以。企业用户可以在控制台的\"发票管理\"中申请开具增值税专用发票或普通发票。个人用户可申请电子发票。" },
  { category: "payment", question: "充值有优惠吗？", answer: "我们不定期会推出充值优惠活动，请关注我们的公告页面或订阅邮件通知获取最新优惠信息。" },

  { category: "vps", question: "VPS多久可以开通？", answer: "订单支付成功后，系统会自动开通VPS，通常在1-5分钟内完成。开通成功后您会收到邮件通知。" },
  { category: "vps", question: "如何远程连接VPS？", answer: "Windows系统使用远程桌面连接，Linux系统使用SSH工具。控制台会显示您的IP地址和初始密码，请在首次登录后及时修改密码。" },
  { category: "vps", question: "VPS可以升级配置吗？", answer: "可以。在控制台的VPS管理页面，选择\"升级配置\"，选择更高配置的套餐并支付差价即可。升级过程不会影响您的数据。" },
  { category: "vps", question: "如何重装系统？", answer: "在控制台选择您要重装的VPS，点击\"重装系统\"，选择所需的系统镜像并确认。重装会清空系统盘数据，请提前备份。" },
  { category: "vps", question: "VPS可以做什么？", answer: "您可以使用VPS搭建网站、运行应用程序、部署游戏服务器、搭建开发测试环境等。但请遵守我们的服务条款，禁止用于非法用途。" },
  { category: "vps", question: "带宽是如何计算的？", answer: "我们提供的带宽是指峰值带宽，您可以随时使用达到该峰值。流量方面，不同套餐有不同配额，超出后可购买额外流量包。" },

  { category: "billing", question: "如何查看我的账单？", answer: "登录控制台后，进入\"账单管理\"可以查看所有历史订单、消费记录和账单详情。支持按时间范围筛选和导出账单。" },
  { category: "billing", question: "支持自动续费吗？", answer: "支持。您可以在VPS管理页面开启\"自动续费\"功能，系统会在到期前自动从余额扣款续费。请确保账户余额充足。" },
  { category: "billing", question: "退款政策是什么？", answer: "我们提供7天无理由退款服务。新用户在首次购买后的7天内，如对服务不满意，可以申请全额退款（已使用流量按标准扣除费用）。" },
  { category: "billing", question: "VPS到期会怎样？", answer: "VPS到期后会被停用，数据保留15天。期间您可以续费恢复服务。超过15天未续费，服务器将被回收，数据将无法恢复。" },
];

const iconByKey: Record<string, any> = {
  all: FileTextOutlined,
  account: UserOutlined,
  payment: CreditCardOutlined,
  vps: DesktopOutlined,
  billing: AccountBookOutlined,
};

const resolved = computed(() => {
  const c = props.content || {};
  const title = String(c.title ?? "常见问题");
  const subtitle = String(c.subtitle ?? "快速找到您关心的问题答案");

  const rawCats = Array.isArray(c.categories) ? c.categories : [];
  const cats: Category[] =
    rawCats.length > 0
      ? rawCats.map((x: any) => ({
          key: String(x?.key ?? ""),
          label: String(x?.label ?? ""),
          icon: iconByKey[String(x?.key ?? "")] || FileTextOutlined,
        }))
      : fallbackCategories;

  const rawFaqs = Array.isArray(c.faqs) ? c.faqs : [];
  const faqs: FAQ[] =
    rawFaqs.length > 0
      ? rawFaqs.map((x: any) => ({
          category: String(x?.category ?? "all"),
          question: String(x?.question ?? ""),
          answer: String(x?.answer ?? ""),
        }))
      : fallbackFAQs;

  return { title, subtitle, categories: cats, faqs };
});

const filteredFAQs = computed(() => {
  let result = resolved.value.faqs;

  if (activeCategory.value !== "all") {
    result = result.filter((faq) => faq.category === activeCategory.value);
  }

  const q = props.searchQuery.trim().toLowerCase();
  if (q) {
    result = result.filter((faq) => faq.question.toLowerCase().includes(q) || faq.answer.toLowerCase().includes(q));
  }

  return result;
});

const toggleFAQ = (index: number) => {
  openFAQ.value = openFAQ.value === index ? null : index;
};
</script>

<style scoped>
/* FAQ Section */
.faq-section {
  position: relative;
  padding: 60px 20px;
  z-index: 1;
}

.section-header {
  text-align: center;
  margin-bottom: 40px;
}

.section-title {
  font-family: var(--font-heading);
  font-size: 36px;
  font-weight: 700;
  color: var(--color-text);
  margin: 0 0 12px;
}

.section-subtitle {
  font-size: 16px;
  color: var(--color-text-muted);
  margin: 0;
}

.category-tabs {
  display: flex;
  justify-content: center;
  gap: 12px;
  margin-bottom: 32px;
  flex-wrap: wrap;
}

.tab-button {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 12px 20px;
  background: var(--color-bg-alt);
  border: 1px solid var(--color-border);
  border-radius: 12px;
  font-family: var(--font-body);
  font-size: 14px;
  color: var(--color-text-muted);
  cursor: pointer;
  transition: all 0.3s;
}

.tab-button:hover {
  background: rgba(14, 165, 233, 0.1);
  border-color: var(--color-primary);
  color: var(--color-primary-light);
}

.tab-button.active {
  background: var(--color-primary);
  border-color: var(--color-primary);
  color: white;
}

.tab-icon {
  font-size: 16px;
}

.faq-accordion {
  max-width: 900px;
  margin: 0 auto;
}

.faq-item {
  background: var(--color-bg-alt);
  border: 1px solid var(--color-border);
  border-radius: 16px;
  margin-bottom: 12px;
  overflow: hidden;
  cursor: pointer;
  transition: all 0.3s;
}

.faq-item:hover {
  border-color: rgba(14, 165, 233, 0.5);
}

.faq-item.active {
  border-color: var(--color-primary);
  background: rgba(14, 165, 233, 0.05);
}

.faq-question {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 20px 24px;
  gap: 16px;
}

.question-content {
  display: flex;
  align-items: center;
  gap: 16px;
  flex: 1;
}

.question-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  background: linear-gradient(135deg, var(--color-primary), var(--color-accent));
  border-radius: 10px;
  font-size: 14px;
  font-weight: 700;
  color: white;
  flex-shrink: 0;
}

.question-text {
  font-size: 16px;
  font-weight: 500;
  color: var(--color-text);
}

.faq-toggle {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  background: rgba(255, 255, 255, 0.05);
  border-radius: 8px;
  font-size: 20px;
  color: var(--color-text-muted);
  transition: all 0.3s;
  flex-shrink: 0;
}

.faq-item.active .faq-toggle {
  background: var(--color-primary);
  color: white;
}

.faq-answer {
  max-height: 0;
  overflow: hidden;
  transition: max-height 0.3s ease, padding 0.3s ease;
}

.faq-answer.expanded {
  max-height: 500px;
  padding: 0 24px 20px;
}

.answer-content {
  display: flex;
  gap: 16px;
  padding-left: 48px;
}

.answer-icon {
  display: flex;
  align-items: flex-start;
  justify-content: center;
  width: 32px;
  height: 32px;
  background: rgba(16, 185, 129, 0.2);
  border-radius: 10px;
  font-size: 14px;
  font-weight: 700;
  color: var(--color-success);
  flex-shrink: 0;
  margin-top: -4px;
}

.answer-text {
  flex: 1;
  font-size: 15px;
  line-height: 1.7;
  color: var(--color-text-muted);
  margin: 0;
}

.no-results {
  text-align: center;
  padding: 60px 20px;
}

.no-results-icon {
  font-size: 40px;
  color: var(--color-text-muted);
  margin-bottom: 16px;
}

.no-results p {
  font-size: 16px;
  color: var(--color-text-muted);
  margin: 0 0 16px;
}

.clear-search-btn {
  padding: 10px 20px;
  background: var(--color-primary);
  border: none;
  border-radius: 8px;
  font-size: 14px;
  color: white;
  cursor: pointer;
  transition: all 0.3s;
}

.clear-search-btn:hover {
  background: var(--color-primary-dark);
}

@media (max-width: 768px) {
  .category-tabs {
    gap: 8px;
  }

  .tab-button {
    padding: 10px 16px;
    font-size: 13px;
  }

  .section-title {
    font-size: 28px;
  }

  .answer-content {
    padding-left: 0;
    flex-direction: column;
    gap: 8px;
  }

  .answer-icon {
    align-self: flex-start;
  }
}
</style>
