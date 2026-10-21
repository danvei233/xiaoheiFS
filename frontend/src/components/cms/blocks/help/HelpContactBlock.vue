<template>
  <section class="contact-section">
    <div class="contact-content">
      <div class="contact-info">
        <h2 class="contact-title">{{ resolved.title }}</h2>
        <p class="contact-description">{{ resolved.description }}</p>

        <div class="contact-channels">
          <div class="channel-item" v-for="(ch, idx) in resolved.channels" :key="idx">
            <component :is="ch.icon" class="channel-icon" />
            <div class="channel-details">
              <h4>{{ ch.title }}</h4>
              <p>{{ ch.subtitle }}</p>
            </div>
          </div>
        </div>
      </div>

      <div class="contact-cta">
        <div class="cta-card">
          <div class="cta-glow"></div>
          <h3>{{ resolved.cta_title }}</h3>
          <p>{{ resolved.cta_desc }}</p>
          <router-link :to="resolved.cta_url" class="cta-button">
            {{ resolved.cta_button_text }}
            <span class="button-arrow">→</span>
          </router-link>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { MessageOutlined, MailOutlined, CustomerServiceOutlined } from "@ant-design/icons-vue";

const props = defineProps<{
  content?: any;
}>();

const resolved = computed(() => {
  const c = props.content || {};
  const channels = Array.isArray(c.channels) ? c.channels : [];

  const fallbackChannels = [
    { key: "chat", title: "在线客服", subtitle: "工作日 9:00 - 18:00", icon: MessageOutlined },
    { key: "mail", title: "邮件支持", subtitle: "24小时内回复", icon: MailOutlined },
    { key: "tickets", title: "工单系统", subtitle: "技术问题优先处理", icon: CustomerServiceOutlined },
  ];

  const iconByKey: Record<string, any> = {
    chat: MessageOutlined,
    mail: MailOutlined,
    tickets: CustomerServiceOutlined,
  };

  const resolvedChannels =
    channels.length > 0
      ? channels.map((x: any) => ({
          key: String(x?.key ?? ""),
          title: String(x?.title ?? ""),
          subtitle: String(x?.subtitle ?? ""),
          icon: iconByKey[String(x?.key ?? "")] || CustomerServiceOutlined,
        }))
      : fallbackChannels;

  return {
    title: String(c.title ?? "还有问题？"),
    description: String(c.description ?? "我们的专业支持团队随时准备为您提供帮助"),
    channels: resolvedChannels,
    cta_title: String(c.cta_title ?? "立即开始使用"),
    cta_desc: String(c.cta_desc ?? "注册账号，享受专业的云服务"),
    cta_button_text: String(c.cta_button_text ?? "免费注册"),
    cta_url: String(c.cta_url ?? "/auth/register"),
  };
});
</script>

<style scoped>
/* Contact Section */
.contact-section {
  position: relative;
  padding: 60px 20px 80px;
  z-index: 1;
}

.contact-content {
  max-width: 1000px;
  margin: 0 auto;
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 40px;
  align-items: center;
}

.contact-title {
  font-family: var(--font-heading);
  font-size: 32px;
  font-weight: 700;
  color: var(--color-text);
  margin: 0 0 12px;
}

.contact-description {
  font-size: 16px;
  color: var(--color-text-muted);
  margin: 0 0 32px;
  line-height: 1.6;
}

.contact-channels {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.channel-item {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 16px;
  background: var(--color-bg-alt);
  border: 1px solid var(--color-border);
  border-radius: 12px;
  transition: all 0.3s;
}

.channel-item:hover {
  border-color: var(--color-primary);
  transform: translateX(8px);
}

.channel-icon {
  font-size: 24px;
  color: var(--color-primary-light);
}

.channel-details h4 {
  font-size: 16px;
  font-weight: 600;
  color: var(--color-text);
  margin: 0 0 4px;
}

.channel-details p {
  font-size: 13px;
  color: var(--color-text-muted);
  margin: 0;
}

/* CTA Card */
.contact-cta {
  display: flex;
  justify-content: center;
}

.cta-card {
  position: relative;
  padding: 40px;
  background: linear-gradient(135deg, var(--color-bg-alt), rgba(14, 165, 233, 0.1));
  border: 1px solid var(--color-border);
  border-radius: 24px;
  text-align: center;
  overflow: hidden;
}

.cta-glow {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  width: 200px;
  height: 200px;
  background: radial-gradient(circle, rgba(14, 165, 233, 0.3) 0%, transparent 70%);
  pointer-events: none;
}

.cta-card h3 {
  font-family: var(--font-heading);
  font-size: 24px;
  font-weight: 700;
  color: var(--color-text);
  margin: 0 0 8px;
  position: relative;
  z-index: 1;
}

.cta-card p {
  font-size: 14px;
  color: var(--color-text-muted);
  margin: 0 0 24px;
  position: relative;
  z-index: 1;
}

.cta-button {
  position: relative;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 14px 32px;
  background: linear-gradient(135deg, var(--color-primary), var(--color-primary-dark));
  border: none;
  border-radius: 12px;
  font-family: var(--font-body);
  font-size: 16px;
  font-weight: 600;
  color: white;
  text-decoration: none;
  cursor: pointer;
  transition: all 0.3s;
  z-index: 1;
}

.cta-button:hover {
  transform: translateY(-2px);
  box-shadow: 0 8px 24px rgba(14, 165, 233, 0.4);
}

.button-arrow {
  transition: transform 0.3s;
}

.cta-button:hover .button-arrow {
  transform: translateX(4px);
}

@media (max-width: 768px) {
  .contact-content {
    grid-template-columns: 1fr;
  }

  .contact-title {
    font-size: 24px;
  }
}
</style>
