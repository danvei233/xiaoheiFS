<template>
  <footer class="marketing-footer">
    <div class="marketing-footer__inner">
      <div class="marketing-footer__top">
        <div class="marketing-footer__brand">
          <RouterLink to="/" class="marketing-footer__logo">
            <SiteLogoMedia :size="30" :src="logoUrl" :alt="siteName" />
            <strong>{{ siteName }}</strong>
          </RouterLink>
          <p>{{ content.description || fallbackDescription }}</p>
          <div v-if="socialLinks.length" class="marketing-footer__social">
            <a
              v-for="social in socialLinks"
              :key="`${social.key}-${social.url}`"
              :href="safeHref(social.url)"
              target="_blank"
              rel="noopener noreferrer"
            >
              {{ social.key || 'Link' }}
            </a>
          </div>
        </div>

        <nav class="marketing-footer__sections" aria-label="页脚导航">
          <section v-for="section in sections" :key="section.title">
            <h2>{{ section.title }}</h2>
            <template v-for="link in section.links" :key="`${link.label}-${link.url}`">
              <RouterLink v-if="isInternalLink(link.url)" :to="link.url || '/'">{{
                link.label
              }}</RouterLink>
              <a v-else :href="safeHref(link.url)" rel="noopener noreferrer">{{ link.label }}</a>
            </template>
          </section>
        </nav>
      </div>

      <div class="marketing-footer__bottom">
        <p>{{ copyrightText || `© ${currentYear} ${siteName}. All rights reserved.` }}</p>
        <div v-if="beianInfoList.length" class="marketing-footer__beian">
          <a
            v-for="beian in beianInfoList"
            :key="beian.number"
            :href="safeHref(beian.link_url)"
            :target="beian.link_url ? '_blank' : '_self'"
            rel="noopener noreferrer"
          >
            <img v-if="beian.icon_url" :src="beian.icon_url" :alt="beian.number" />
            <span>{{ beian.number }}</span>
          </a>
        </div>
        <div v-if="badges.length" class="marketing-footer__badges">
          <span v-for="badge in badges" :key="badge"
            ><ArtSvgIcon icon="ri:shield-check-line" />{{ badge }}</span
          >
        </div>
      </div>
    </div>
  </footer>
</template>

<script setup lang="ts">
  import { RouterLink } from 'vue-router'
  import SiteLogoMedia from '@/components/brand/SiteLogoMedia.vue'
  import { safeHref } from '@/utils/urlPolicy'

  const props = defineProps<{
    siteName: string
    logoUrl?: string
    content: { description?: string; social_links?: Array<{ key?: string; url?: string }> }
    sections: Array<{ title?: string; links: Array<{ label?: string; url?: string }> }>
    badges: string[]
    copyrightText?: string
    beianInfoList?: Array<{ number: string; icon_url?: string; link_url?: string }>
  }>()

  const currentYear = new Date().getFullYear()
  const fallbackDescription = '专业的云服务提供商，为企业提供可靠、安全、高性能的云计算解决方案'
  const socialLinks = computed(() => props.content.social_links || [])
  const isInternalLink = (url?: string) =>
    String(url || '')
      .trim()
      .startsWith('/')
</script>

<style lang="scss" scoped>
  .marketing-footer {
    color: var(--art-gray-600);
    background: var(--default-box-color);
    border-top: 1px solid var(--default-border);

    &__inner {
      max-width: 1240px;
      padding: 64px 24px 24px;
      margin: 0 auto;
    }

    &__top {
      display: grid;
      grid-template-columns: minmax(240px, 1.1fr) minmax(0, 2fr);
      gap: 72px;
      padding-bottom: 42px;
    }

    &__brand > p {
      max-width: 360px;
      margin: 18px 0;
      line-height: 1.75;
    }

    &__logo {
      display: inline-flex;
      gap: 10px;
      align-items: center;
      font-size: 17px;
      color: var(--art-gray-900);
      text-decoration: none;
    }

    &__social {
      display: flex;
      flex-wrap: wrap;
      gap: 8px 16px;

      a {
        color: var(--art-gray-600);
        text-decoration: none;

        &:hover {
          color: var(--theme-color);
        }
      }
    }

    &__sections {
      display: grid;
      grid-template-columns: repeat(4, minmax(0, 1fr));
      gap: 28px;

      section {
        display: flex;
        flex-direction: column;
        gap: 11px;
      }

      h2 {
        margin: 0 0 4px;
        font-size: 14px;
        color: var(--art-gray-900);
      }

      a {
        font-size: 13px;
        color: var(--art-gray-600);
        text-decoration: none;
        overflow-wrap: anywhere;

        &:hover {
          color: var(--theme-color);
        }
      }
    }

    &__bottom {
      display: grid;
      grid-template-columns: minmax(0, 1fr) auto minmax(0, 1fr);
      gap: 20px;
      align-items: center;
      padding-top: 22px;
      border-top: 1px solid var(--default-border);

      > p {
        margin: 0;
        font-size: 12px;
      }
    }

    &__beian,
    &__badges {
      display: flex;
      flex-wrap: wrap;
      gap: 8px 14px;
      justify-content: center;
    }

    &__beian a,
    &__badges span {
      display: inline-flex;
      gap: 5px;
      align-items: center;
      font-size: 12px;
      color: var(--art-gray-600);
      text-decoration: none;
    }

    &__beian img {
      width: 16px;
      height: 16px;
      object-fit: contain;
    }

    &__badges {
      justify-content: flex-end;

      .art-svg-icon {
        color: var(--art-success);
      }
    }
  }

  @media (width <= 900px) {
    .marketing-footer {
      &__top {
        grid-template-columns: 1fr;
        gap: 38px;
      }

      &__bottom {
        grid-template-columns: 1fr;
        text-align: center;
      }

      &__badges {
        justify-content: center;
      }
    }
  }

  @media (width <= 600px) {
    .marketing-footer {
      &__inner {
        padding: 48px 18px 22px;
      }

      &__sections {
        grid-template-columns: repeat(2, minmax(0, 1fr));
      }
    }
  }
</style>
