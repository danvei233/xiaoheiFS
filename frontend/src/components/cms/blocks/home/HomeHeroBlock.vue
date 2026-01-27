<template>
  <section class="hero-section">
    <div class="hero-grid">
      <div class="grid-line" v-for="i in 12" :key="i" :style="{ left: `${i * 8.33}%` }"></div>
      <div class="grid-line horizontal" v-for="i in 8" :key="`h-${i}`" :style="{ top: `${i * 12.5}%` }"></div>

      <div class="hero-container">
        <div class="hero-content">
          <!-- Badge - 可编辑 -->
          <InlineEdit
            v-if="isVisualMode"
            field-path="hero.badge"
            v-model="localBadge"
            edit-type="text"
            label="徽标文案"
          />
          <div v-show="!isVisualMode" class="hero-badge">
            <span class="badge-dot"></span>
            <span>{{ heroContent.badge || $t('home.hero.badge') || '下一代云计算平台' }}</span>
          </div>

          <h1 class="hero-title">
            <!-- Title1 - 可编辑 -->
            <InlineEdit
              v-if="isVisualMode"
              field-path="hero.title1"
              v-model="localTitle1"
              edit-type="text"
              label="??"
            />
            <span v-show="!isVisualMode" class="title-line">{{ heroContent.title1 || $t('home.hero.title1') || '构建未来' }}</span>

            <!-- Typewriter text - keep line height stable even when empty -->
            <span class="title-line typewriter-row">
              <span class="title-gradient typewriter">{{ typewriterText || '\u00A0' }}</span>
              <span class="cursor" aria-hidden="true">|</span>
            </span>
          </h1>

          <!-- Subtitle - 可编辑 -->
          <InlineEdit
            v-if="isVisualMode"
            field-path="hero.subtitle"
            v-model="localSubtitle"
            edit-type="textarea"
            label="??"
            :rows="2"
          />
          <p v-show="!isVisualMode" class="hero-subtitle">
            {{ heroContent.subtitle || $t('home.hero.subtitle') || '企业级云服务器，全球节点覆盖，99.99% SLA 保障。秒级部署，弹性扩展，为您的业务保驾护航。' }}
          </p>

          <!-- Buttons - 可编辑 -->
          <div class="hero-actions">
            <template v-if="isVisualMode">
              <InlineEdit
                field-path="hero.primary_button_text"
                v-model="localPrimaryButtonText"
                edit-type="text"
                label="?????"
              />
              <InlineEdit
                field-path="hero.primary_button_link"
                v-model="localPrimaryButtonLink"
                edit-type="url"
                label="?????"
              />
            </template>
            <router-link v-show="!isVisualMode" :to="heroContent.primary_button_link || '/register'" class="btn btn-primary">
              <span>{{ heroContent.primary_button_text || $t('home.hero.getStarted') || '立即开始' }}</span>
              <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
                <path d="M3 8H13M13 8L9 4M13 8L9 12" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
              </svg>
            </router-link>

            <template v-if="isVisualMode">
              <InlineEdit
                field-path="hero.secondary_button_text"
                v-model="localSecondaryButtonText"
                edit-type="text"
                label="?????"
              />
              <InlineEdit
                field-path="hero.secondary_button_link"
                v-model="localSecondaryButtonLink"
                edit-type="url"
                label="?????"
              />
            </template>
            <router-link v-show="!isVisualMode" :to="heroContent.secondary_button_link || '/products'" class="btn btn-secondary">
              <span>{{ heroContent.secondary_button_text || $t('home.hero.viewProducts') || '浏览产品' }}</span>
            </router-link>
          </div>

          <!-- Stats - 可编辑数组 -->
          <div class="hero-stats">
            <template v-if="isVisualMode">
              <InlineEdit
                v-for="(stat, index) in stats"
                :key="`stat-value-${index}`"
                :field-path="`hero.stats.${index}.value`"
                :model-value="stat.value"
                @update:model-value="(val) => updateStatField(index, 'value', val)"
                edit-type="number"
                label="统计数值"
                :is-array-item="true"
                :can-add="index === stats.length - 1"
                :can-remove="stats.length > 1"
                @add-item="addStatItem"
                @remove-item="() => removeStatItem(index)"
              />
              <InlineEdit
                v-for="(stat, index) in stats"
                :key="`stat-suffix-${index}`"
                :field-path="`hero.stats.${index}.suffix`"
                :model-value="stat.suffix"
                @update:model-value="(val) => updateStatField(index, 'suffix', val)"
                edit-type="text"
                label="后缀"
              />
              <InlineEdit
                v-for="(stat, index) in stats"
                :key="`stat-label-${index}`"
                :field-path="`hero.stats.${index}.label`"
                :model-value="stat.label"
                @update:model-value="(val) => updateStatField(index, 'label', val)"
                edit-type="text"
                label="标签"
              />
            </template>
            <template v-if="!isVisualMode">
              <div class="stat-item" v-for="(stat, index) in stats" :key="`stat-display-${index}`">
                <div class="stat-value">
                  <span>{{ animatedStats[index] }}</span>{{ stat.suffix }}
                </div>
                <div class="stat-label">{{ stat.label }}</div>
              </div>
            </template>
          </div>
        </div>

        <div class="hero-visual">
          <div class="server-rack">
            <div class="rack-unit" v-for="i in 4" :key="i">
              <div class="unit-lights">
                <span class="light green" :style="{ animationDelay: `${i * 0.2}s` }"></span>
                <span class="light blue" :style="{ animationDelay: `${i * 0.2 + 0.1}s` }"></span>
                <span class="light orange" :style="{ animationDelay: `${i * 0.2 + 0.2}s` }"></span>
              </div>
              <div class="unit-label">SERVER {{ i }}</div>
            </div>
          </div>

          <!-- Floating Cards - 可编辑数组 -->
          <template v-if="isVisualMode">
            <div
              v-for="(card, index) in heroCards"
              :key="`card-edit-${index}`"
              class="floating-card tilt-card"
              :class="`card-${index + 1}`"
            >
              <div class="card-icon">
                <svg width="24" height="24" viewBox="0 0 24 24" fill="none">
                  <path v-if="index === 0" d="M13 2L3 14H12L11 22L21 10H12L13 2Z" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
                  <path v-else-if="index === 1" d="M12 22C17.5228 22 22 17.5228 22 12C22 6.47715 17.5228 2 12 2C6.47715 2 2 6.47715 2 12C2 17.5228 6.47715 22 12 22Z" stroke="currentColor" stroke-width="2"/>
                  <path v-else d="M12 2L2 7L12 12L22 7L12 2Z" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
                </svg>
              </div>
              <div class="card-text">
                <InlineEdit
                  :field-path="`hero.cards.${index}.title`"
                  :model-value="card.title"
                  @update:model-value="(val) => updateCardField(index, 'title', val)"
                  edit-type="text"
                  label="卡片标题"
                  :is-array-item="true"
                  :can-add="index === heroCards.length - 1 && heroCards.length < 3"
                  :can-remove="heroCards.length > 1"
                  @add-item="addCardItem"
                  @remove-item="() => removeCardItem(index)"
                />
                <InlineEdit
                  :field-path="`hero.cards.${index}.desc`"
                  :model-value="card.desc"
                  @update:model-value="(val) => updateCardField(index, 'desc', val)"
                  edit-type="text"
                  label="卡片描述"
                />
              </div>
            </div>
            <!-- 空数组时显示添加按钮 -->
            <div v-if="heroCards.length === 0" class="empty-cards">
              <a-button type="dashed" @click="addCardItem">+ 添加卡片</a-button>
            </div>
          </template>
          <template v-if="!isVisualMode">
            <div
              class="floating-card card-1 tilt-card"
              @mousemove="(event) => handleTilt?.(event, 'card1')"
              @mouseleave="() => resetTilt?.('card1')"
              :ref="card1Ref"
            >
              <div class="card-icon">
                <svg width="24" height="24" viewBox="0 0 24 24" fill="none">
                  <path d="M13 2L3 14H12L11 22L21 10H12L13 2Z" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
                </svg>
              </div>
              <div class="card-text">
                <div class="card-title">{{ heroCards[0]?.title || '极速部署' }}</div>
                <div class="card-desc">{{ heroCards[0]?.desc || '60秒开机' }}</div>
              </div>
            </div>
            <div
              class="floating-card card-2 tilt-card"
              @mousemove="(event) => handleTilt?.(event, 'card2')"
              @mouseleave="() => resetTilt?.('card2')"
              :ref="card2Ref"
            >
              <div class="card-icon">
                <svg width="24" height="24" viewBox="0 0 24 24" fill="none">
                  <path d="M12 22C17.5228 22 22 17.5228 22 12C22 6.47715 17.5228 2 12 2C6.47715 2 2 6.47715 2 12C2 17.5228 6.47715 22 12 22Z" stroke="currentColor" stroke-width="2"/>
                  <path d="M2 12H22" stroke="currentColor" stroke-width="2"/>
                  <path d="M12 2C14.5013 4.73835 15.9228 8.29203 16 12C15.9228 15.708 14.5013 19.2616 12 22C9.49872 19.2616 8.07725 15.708 8 12C8.07725 8.29203 9.49872 4.73835 12 2Z" stroke="currentColor" stroke-width="2"/>
                </svg>
              </div>
              <div class="card-text">
                <div class="card-title">{{ heroCards[1]?.title || '全球网络' }}</div>
                <div class="card-desc">{{ heroCards[1]?.desc || '覆盖150+国家' }}</div>
              </div>
            </div>
            <div
              class="floating-card card-3 tilt-card"
              @mousemove="(event) => handleTilt?.(event, 'card3')"
              @mouseleave="() => resetTilt?.('card3')"
              :ref="card3Ref"
            >
              <div class="card-icon">
                <svg width="24" height="24" viewBox="0 0 24 24" fill="none">
                  <path d="M12 2L2 7L12 12L22 7L12 2Z" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
                  <path d="M2 17L12 22L22 17" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
                  <path d="M2 12L12 17L22 12" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
                </svg>
              </div>
              <div class="card-text">
                <div class="card-title">{{ heroCards[2]?.title || '多层防护' }}</div>
                <div class="card-desc">{{ heroCards[2]?.desc || 'DDoS防御' }}</div>
              </div>
            </div>
          </template>
        </div>
      </div>
    </div>
    <div class="hero-glow glow-1"></div>
    <div class="hero-glow glow-2"></div>
  </section>
</template>

<script setup lang="ts">
import { RouterLink } from 'vue-router'
import { inject, computed } from 'vue'
import { Button as AButton } from 'ant-design-vue'
import InlineEdit from '@/components/InlineEdit.vue'

interface HeroContent {
  badge?: string
  title1?: string
  subtitle?: string
  primary_button_text?: string
  primary_button_link?: string
  secondary_button_text?: string
  secondary_button_link?: string
  stats?: Array<{ value: number | string; suffix?: string; label?: string }>
  cards?: Array<{ title?: string; desc?: string }>
}

const props = defineProps<{
  heroContent: HeroContent
  typewriterText: string
  stats: Array<{ value: number | string; suffix?: string; label?: string }>
  animatedStats: string[]
  heroCards: Array<{ title?: string; desc?: string }>
  handleTilt?: (event: MouseEvent, cardName: string) => void
  resetTilt?: (cardName: string) => void
  card1Ref?: any
  card2Ref?: any
  card3Ref?: any
}>()

// 获取 CMS 编辑上下文
const cmsEditContext: any = inject('cmsEditContext', null)
const isVisualMode = cmsEditContext?.editMode === 'visual'

// 本地绑定 computed 属性
const localBadge = computed({
  get: () => props.heroContent.badge || '',
  set: (val: string) => cmsEditContext?.updateField('hero.badge', val)
})

const localTitle1 = computed({
  get: () => props.heroContent.title1 || '',
  set: (val: string) => cmsEditContext?.updateField('hero.title1', val)
})

const localSubtitle = computed({
  get: () => props.heroContent.subtitle || '',
  set: (val: string) => cmsEditContext?.updateField('hero.subtitle', val)
})

const localPrimaryButtonText = computed({
  get: () => props.heroContent.primary_button_text || '',
  set: (val: string) => cmsEditContext?.updateField('hero.primary_button_text', val)
})

const localPrimaryButtonLink = computed({
  get: () => props.heroContent.primary_button_link || '',
  set: (val: string) => cmsEditContext?.updateField('hero.primary_button_link', val)
})

const localSecondaryButtonText = computed({
  get: () => props.heroContent.secondary_button_text || '',
  set: (val: string) => cmsEditContext?.updateField('hero.secondary_button_text', val)
})

const localSecondaryButtonLink = computed({
  get: () => props.heroContent.secondary_button_link || '',
  set: (val: string) => cmsEditContext?.updateField('hero.secondary_button_link', val)
})

// Stats 数组操作
const addStatItem = () => {
  cmsEditContext?.addArrayItem('hero.stats', { value: 0, suffix: '', label: '' })
}

const removeStatItem = (index: number) => {
  cmsEditContext?.removeArrayItem('hero.stats', index)
}

const updateStatField = (
  index: number,
  key: 'value' | 'suffix' | 'label',
  value: any,
) => {
  cmsEditContext?.updateField(`hero.stats.${index}.${key}`, value)
}

// Cards 数组操作
const addCardItem = () => {
  cmsEditContext?.addArrayItem('hero.cards', { title: '', desc: '' })
}

const removeCardItem = (index: number) => {
  cmsEditContext?.removeArrayItem('hero.cards', index)
}

const updateCardField = (
  index: number,
  key: 'title' | 'desc',
  value: any,
) => {
  cmsEditContext?.updateField(`hero.cards.${index}.${key}`, value)
}
</script>

<style scoped>
.empty-cards {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
}

/* 覆盖 InlineEdit 样式以适配 hero-section */
.hero-title :deep(.editable-region),
.hero-subtitle :deep(.editable-region),
.hero-actions :deep(.editable-region) {
  display: inline;
}
</style>
