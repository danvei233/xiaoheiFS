<template>
  <section class="quick-actions">
    <div class="actions-grid">
      <template v-for="card in resolved.cards" :key="card.key">
        <!-- Tickets has special auth behavior (preserve existing logic) -->
        <router-link
          v-if="card.key === 'tickets' && isAuthenticated"
          :to="card.url"
          class="action-card tickets"
        >
          <div class="card-background">
            <CustomerServiceOutlined class="floating-icon icon-1" />
          </div>
          <div class="card-content">
            <CustomerServiceOutlined class="card-icon" />
            <h3 class="card-title">{{ card.title }}</h3>
            <p class="card-description">{{ card.description }}</p>
            <span class="card-arrow">→</span>
          </div>
        </router-link>
        <router-link
          v-else-if="card.key === 'tickets' && !isAuthenticated && isInternalLink(card.guest_url || '')"
          :to="card.guest_url"
          class="action-card tickets"
        >
          <div class="card-background">
            <CustomerServiceOutlined class="floating-icon icon-1" />
          </div>
          <div class="card-content">
            <CustomerServiceOutlined class="card-icon" />
            <h3 class="card-title">{{ card.title }}</h3>
            <p class="card-description">{{ card.description }}</p>
            <span class="card-arrow">→</span>
          </div>
        </router-link>
        <a v-else-if="card.key === 'tickets' && !isAuthenticated" :href="card.guest_url" class="action-card tickets">
          <div class="card-background">
            <CustomerServiceOutlined class="floating-icon icon-1" />
          </div>
          <div class="card-content">
            <CustomerServiceOutlined class="card-icon" />
            <h3 class="card-title">{{ card.title }}</h3>
            <p class="card-description">{{ card.description }}</p>
            <span class="card-arrow">→</span>
          </div>
        </a>

        <router-link v-else-if="card.router" :to="card.url" :class="['action-card', card.key]">
          <div class="card-background">
            <component :is="card.bgIcons[0]" class="floating-icon icon-1" />
            <component v-if="card.bgIcons[1]" :is="card.bgIcons[1]" class="floating-icon icon-2" :style="card.bgIcon2Style" />
          </div>
          <div class="card-content">
            <component :is="card.icon" class="card-icon" />
            <h3 class="card-title">{{ card.title }}</h3>
            <p class="card-description">{{ card.description }}</p>
            <span class="card-arrow">→</span>
          </div>
        </router-link>

        <router-link v-else-if="isInternalLink(card.url)" :to="card.url" :class="['action-card', card.key]">
          <div class="card-background">
            <component :is="card.bgIcons[0]" class="floating-icon icon-1" />
            <component v-if="card.bgIcons[1]" :is="card.bgIcons[1]" class="floating-icon icon-2" />
          </div>
          <div class="card-content">
            <component :is="card.icon" class="card-icon" />
            <h3 class="card-title">{{ card.title }}</h3>
            <p class="card-description">{{ card.description }}</p>
            <span class="card-arrow">→</span>
          </div>
        </router-link>

        <a v-else :href="card.url" :class="['action-card', card.key]">
          <div class="card-background">
            <component :is="card.bgIcons[0]" class="floating-icon icon-1" />
            <component v-if="card.bgIcons[1]" :is="card.bgIcons[1]" class="floating-icon icon-2" />
          </div>
          <div class="card-content">
            <component :is="card.icon" class="card-icon" />
            <h3 class="card-title">{{ card.title }}</h3>
            <p class="card-description">{{ card.description }}</p>
            <span class="card-arrow">→</span>
          </div>
        </a>
      </template>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from "vue";
import {
  BookOutlined,
  CustomerServiceOutlined,
  NotificationOutlined,
  MailOutlined,
  FileTextOutlined,
} from "@ant-design/icons-vue";

type CardInput = {
  key: string;
  title: string;
  description: string;
  url: string;
  guest_url?: string;
};

const props = defineProps<{
  content?: any;
  isAuthenticated: boolean;
}>();

const isInternalLink = (url: string) => String(url || "").trim().startsWith("/");

const resolved = computed(() => {
  const c = props.content || {};
  const rawCards: CardInput[] = Array.isArray(c.cards) ? c.cards : [];

  const defaults: CardInput[] = [
    { key: "docs", title: "文档中心", description: "详细的产品文档和使用指南", url: "/docs" },
    {
      key: "tickets",
      title: "提交工单",
      description: "获取一对一的技术支持",
      url: "/console/tickets",
      guest_url: "/login",
    },
    { key: "announcements", title: "最新公告", description: "系统更新与重要通知", url: "/announcements" },
    { key: "contact", title: "邮件支持", description: "support@example.com", url: "mailto:support@example.com" },
  ];

  const cards = (rawCards.length > 0 ? rawCards : defaults).map((card) => {
    const key = String(card?.key ?? "");
    const title = String(card?.title ?? "");
    const description = String(card?.description ?? "");
    const url = String(card?.url ?? "#");

    if (key === "docs") {
      return {
        key,
        title: title || "文档中心",
        description: description || "详细的产品文档和使用指南",
        url: url || "/docs",
        router: true,
        icon: BookOutlined,
        bgIcons: [BookOutlined, FileTextOutlined],
        bgIcon2Style: {},
      };
    }
    if (key === "tickets") {
      return {
        key,
        title: title || "提交工单",
        description: description || "获取一对一的技术支持",
        url: url || "/console/tickets",
        guest_url: String(card?.guest_url ?? "/login"),
      };
    }
    if (key === "announcements") {
      return {
        key,
        title: title || "最新公告",
        description: description || "系统更新与重要通知",
        url: url || "/announcements",
        router: true,
        icon: NotificationOutlined,
        bgIcons: [NotificationOutlined, NotificationOutlined],
        bgIcon2Style: { opacity: "0.3" },
      };
    }
    if (key === "contact") {
      return {
        key,
        title: title || "邮件支持",
        description: description || "support@example.com",
        url: url || "mailto:support@example.com",
        router: false,
        icon: MailOutlined,
        bgIcons: [MailOutlined, MailOutlined],
        bgIcon2Style: {},
      };
    }

    // Unknown card: render as external link with a mail icon.
    return {
      key: key || "custom",
      title: title || "链接",
      description: description || "",
      url,
      router: false,
      icon: MailOutlined,
      bgIcons: [MailOutlined, MailOutlined],
      bgIcon2Style: {},
    };
  });

  return { cards };
});
</script>

<style scoped>
/* Quick Actions */
.quick-actions {
  position: relative;
  padding: 60px 20px;
  z-index: 1;
}

.actions-grid {
  max-width: 1200px;
  gap: 24px;
  margin: 0 auto;
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
}

.action-card {
  position: relative;
  display: block;
  padding: 32px;
  background: var(--color-bg-alt);
  border: 1px solid var(--color-border);
  border-radius: 20px;
  overflow: hidden;
  text-decoration: none;
  transition: all 0.4s cubic-bezier(0.4, 0, 0.2, 1);
}

.action-card::before {
  content: "";
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: linear-gradient(135deg, rgba(14, 165, 233, 0.1), transparent);
  opacity: 0;
  transition: opacity 0.4s;
}

.action-card:hover::before {
  opacity: 1;
}

.action-card:hover {
  transform: translateY(-8px);
  border-color: var(--color-primary);
  box-shadow: 0 20px 40px rgba(0, 0, 0, 0.3), 0 0 60px rgba(14, 165, 233, 0.1);
}

.card-background {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  pointer-events: none;
  overflow: hidden;
}

.floating-icon {
  position: absolute;
  font-size: 64px;
  color: var(--color-primary);
  opacity: 0.05;
  transition: transform 0.6s ease;
}

.icon-1 {
  top: -20px;
  right: -20px;
}

.icon-2 {
  bottom: -20px;
  left: -20px;
}

.action-card:hover .floating-icon {
  transform: scale(1.1) rotate(10deg);
}

.card-content {
  position: relative;
  z-index: 1;
}

.card-icon {
  font-size: 32px;
  color: var(--color-primary-light);
  margin-bottom: 16px;
}

.card-title {
  font-family: var(--font-heading);
  font-size: 20px;
  font-weight: 600;
  color: var(--color-text);
  margin: 0 0 8px;
}

.card-description {
  font-size: 14px;
  color: var(--color-text-muted);
  margin: 0 0 16px;
}

.card-arrow {
  display: inline-flex;
  align-items: center;
  font-size: 18px;
  color: var(--color-primary-light);
  transition: transform 0.3s;
}

.action-card:hover .card-arrow {
  transform: translateX(4px);
}

@media (max-width: 768px) {
  .actions-grid {
    grid-template-columns: 1fr;
  }
}
</style>
