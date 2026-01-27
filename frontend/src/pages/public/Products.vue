<template>
  <div class="products-page" ref="productsPage">
    <div class="page-background">
      <div class="grid-overlay"></div>
      <div class="glow-orb glow-1"></div>
      <div class="glow-orb glow-2"></div>
      <div class="glow-orb glow-3"></div>
    </div>

    <template v-for="type in productsBlockOrder" :key="type">
      <ProductsHeroBlock v-if="type === 'hero'" :content="heroContent" />
      <ProductsCalculatorBlock
        v-else-if="type === 'calculator'"
        :content="calculatorContent"
        :scenarios="scenarios"
        :selected-scenario="selectedScenario"
        :on-select="selectScenario"
      />
      <ProductsPricingBlock
        v-else-if="type === 'pricing'"
        :products="products"
        :selected-plan="selectedPlan"
        :on-select="(index) => (selectedPlan = index)"
        :on-hover="handleCardHover"
      />
      <ProductsComparisonBlock
        v-else-if="type === 'comparison'"
        :content="comparisonContent"
        :products="products"
        :rows="comparisonRows"
      />
      <ProductsCtaBlock v-else-if="type === 'cta'" :content="ctaContent" />
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { h, defineComponent } from 'vue'
import { getCmsBlocks } from '@/services/user'
import ProductsHeroBlock from '@/components/cms/blocks/products/ProductsHeroBlock.vue'
import ProductsCalculatorBlock from '@/components/cms/blocks/products/ProductsCalculatorBlock.vue'
import ProductsPricingBlock from '@/components/cms/blocks/products/ProductsPricingBlock.vue'
import ProductsComparisonBlock from '@/components/cms/blocks/products/ProductsComparisonBlock.vue'
import ProductsCtaBlock from '@/components/cms/blocks/products/ProductsCtaBlock.vue'

// Refs
const productsPage = ref<HTMLElement | null>(null)
const pricingSection = ref<HTMLElement | null>(null)
const productsBlockOrder = ref<string[]>(["hero", "calculator", "pricing", "comparison", "cta"])

const setPricingSection = (el: HTMLElement | null) => {
  pricingSection.value = el
}

// State
const selectedScenario = ref<number | null>(null)
const selectedPlan = ref<number>(2) // Default to recommended plan

// Scenarios for calculator
const scenarios = ref([
  {
    icon: '📝',
    name: '个人博客',
    recommended: '基础型 - 1核1G',
    plan: 0
  },
  {
    icon: '🛒',
    name: '小型电商',
    recommended: '标准型 - 2核4G',
    plan: 1
  },
  {
    icon: '🎮',
    name: '游戏服务器',
    recommended: '高性能型 - 4核8G',
    plan: 2
  },
  {
    icon: '🏢',
    name: '企业应用',
    recommended: '企业型 - 8核16G',
    plan: 3
  }
])

const selectScenario = (index: number) => {
  selectedScenario.value = index
  selectedPlan.value = scenarios.value[index]?.plan ?? 0
}

const handleCardHover = (index: number) => {
  // Could add additional hover effects here
}

// Hero features
const heroFeatures = ref([
  "秒级部署",
  "弹性扩容",
  "99.99% SLA",
  "24/7 支持"
])

// SVG Icons
const CloudIcon = defineComponent({
  render: () => h('svg', { viewBox: '0 0 24 24', fill: 'none' }, [
    h('path', {
      d: 'M18 10H17.5C17.1 6 13.5 3 9.5 3C5.5 3 2 6 2 10C2 10.5 2.1 11 2.2 11.5C1 12 0 13.2 0 14.5C0 16.5 1.5 18 3.5 18H18C20.8 18 23 15.8 23 13C23 10.5 21.2 8.5 18.8 8.1C18.6 8 18.3 8 18 8V10Z',
      stroke: 'currentColor',
      'stroke-width': 2,
      transform: 'translate(1 2)'
    })
  ])
})

const RocketIcon = defineComponent({
  render: () => h('svg', { viewBox: '0 0 24 24', fill: 'none' }, [
    h('path', {
      d: 'M4.5 16.5C4.5 16.5 4 14 4 14',
      stroke: 'currentColor',
      'stroke-width': 2,
      'stroke-linecap': 'round'
    }),
    h('path', {
      d: 'M7.5 19.5C7.5 19.5 7 17 7 17',
      stroke: 'currentColor',
      'stroke-width': 2,
      'stroke-linecap': 'round'
    }),
    h('path', {
      d: 'M16.5 10C16.5 10 19 12 19 14C19 17 16.5 19 16.5 19',
      stroke: 'currentColor',
      'stroke-width': 2,
      'stroke-linecap': 'round'
    }),
    h('path', {
      d: 'M15.5 4L13 10L7 10L4.5 4',
      stroke: 'currentColor',
      'stroke-width': 2,
      'stroke-linecap': 'round',
      'stroke-linejoin': 'round'
    }),
    h('path', {
      d: 'M13 10V19C13 20.66 11.66 22 10 22C8.34 22 7 20.66 7 19V10',
      stroke: 'currentColor',
      'stroke-width': 2
    }),
    h('path', {
      d: 'M10 2V4',
      stroke: 'currentColor',
      'stroke-width': 2,
      'stroke-linecap': 'round'
    }),
    h('path', {
      d: 'M16 6L13 10',
      stroke: 'currentColor',
      'stroke-width': 2,
      'stroke-linecap': 'round'
    }),
    h('path', {
      d: 'M4 6L7 10',
      stroke: 'currentColor',
      'stroke-width': 2,
      'stroke-linecap': 'round'
    })
  ])
})

const BoltIcon = defineComponent({
  render: () => h('svg', { viewBox: '0 0 24 24', fill: 'none' }, [
    h('path', {
      d: 'M13 2L3 14H12L11 22L21 10H12L13 2Z',
      stroke: 'currentColor',
      'stroke-width': 2,
      'stroke-linecap': 'round',
      'stroke-linejoin': 'round'
    })
  ])
})

const BuildingIcon = defineComponent({
  render: () => h('svg', { viewBox: '0 0 24 24', fill: 'none' }, [
    h('rect', { x: '4', y: '2', width: '16', height: '20', rx: '2', stroke: 'currentColor', 'stroke-width': 2 }),
    h('path', { d: 'M9 2V22', stroke: 'currentColor', 'stroke-width': 2 }),
    h('path', { d: 'M15 2V22', stroke: 'currentColor', 'stroke-width': 2 }),
    h('path', { d: 'M4 12H20', stroke: 'currentColor', 'stroke-width': 2 }),
    h('path', { d: 'M4 7H9', stroke: 'currentColor', 'stroke-width': 2 }),
    h('path', { d: 'M15 7H20', stroke: 'currentColor', 'stroke-width': 2 })
  ])
})

// Products data
const products = ref([
  {
    icon: CloudIcon,
    name: "基础型",
    description: "适合个人博客、小型网站",
    price: "29",
    recommended: false,
    cta: "立即选购",
    resources: [
      { label: "CPU", value: "1 核", percent: 25 },
      { label: "内存", value: "1 GB", percent: 12.5 },
      { label: "存储", value: "20 GB", percent: 20 },
      { label: "带宽", value: "1 Mbps", percent: 10 }
    ],
    features: [
      "默认 1核1G 配置",
      "20GB SSD 高速云盘",
      "1Mbps 带宽",
      "Linux 操作系统",
      "免费备案服务",
      "99.99% 可用性",
      "7天无理由退款",
      "24/7 工单支持"
    ]
  },
  {
    icon: RocketIcon,
    name: "标准型",
    description: "适合中小企业、Web 应用",
    price: "59",
    recommended: false,
    cta: "立即选购",
    resources: [
      { label: "CPU", value: "2 核", percent: 50 },
      { label: "内存", value: "4 GB", percent: 50 },
      { label: "存储", value: "40 GB", percent: 40 },
      { label: "带宽", value: "3 Mbps", percent: 30 }
    ],
    features: [
      "默认 2核4G 配置",
      "40GB SSD 高速云盘",
      "3Mbps 带宽",
      "Linux/Windows 系统",
      "免费自动备份",
      "负载均衡支持",
      "DDoS 防护",
      "优先技术支持"
    ]
  },
  {
    icon: BoltIcon,
    name: "高性能型",
    description: "适合计算密集型应用",
    price: "129",
    recommended: true,
    cta: "立即选购",
    resources: [
      { label: "CPU", value: "4 核", percent: 100 },
      { label: "内存", value: "8 GB", percent: 100 },
      { label: "存储", value: "80 GB", percent: 80 },
      { label: "带宽", value: "5 Mbps", percent: 50 }
    ],
    features: [
      "默认 4核8G 配置",
      "80GB SSD 高速云盘",
      "5Mbps 带宽",
      "任意操作系统",
      "每日自动备份",
      "弹性伸缩支持",
      "高级 DDoS 防护",
      "专属客服支持",
      "SLA 保障"
    ]
  },
  {
    icon: BuildingIcon,
    name: "企业型",
    description: "适合大型企业、关键业务",
    price: "299",
    recommended: false,
    cta: "联系销售",
    resources: [
      { label: "CPU", value: "8 核", percent: 100 },
      { label: "内存", value: "16 GB", percent: 100 },
      { label: "存储", value: "160 GB", percent: 100 },
      { label: "带宽", value: "10 Mbps", percent: 100 }
    ],
    features: [
      "默认 8核16G 配置",
      "160GB SSD 企业级云盘",
      "10Mbps 独享带宽",
      "任意操作系统",
      "实时异地备份",
      "私有网络部署",
      "企业级安全方案",
      "专属客户经理",
      "定制化服务",
      "99.995% SLA"
    ]
  }
])

// Comparison data
const comparisonRows = ref([
  { feature: "CPU", values: ["1 核", "2 核", "4 核", "8 核"] },
  { feature: "内存", values: ["1 GB", "4 GB", "8 GB", "16 GB"] },
  { feature: "存储", values: ["20 GB SSD", "40 GB SSD", "80 GB SSD", "160 GB SSD"] },
  { feature: "带宽", values: ["1 Mbps", "3 Mbps", "5 Mbps", "10 Mbps"] },
  { feature: "操作系统", values: ["Linux", "Linux/Windows", "任意系统", "任意系统"] },
  { feature: "流量限制", values: ["不限", "不限", "不限", "不限"] },
  { feature: "备份数量", values: ["手动", "每天1次", "每天1次", "实时备份"] },
  { feature: "DDoS防护", values: ["基础", "基础", "高级", "企业级"] },
  { feature: "技术支持", values: ["工单", "工单", "优先", "专属经理"] },
  { feature: "SLA保障", values: ["99.99%", "99.99%", "99.99%", "99.995%"] }
])

const heroContent = ref({
  badge: "",
  title: "",
  subtitle: "",
  features: [] as string[]
})

const calculatorContent = ref({
  title: "",
  desc: ""
})

const comparisonContent = ref({
  title: ""
})

const ctaContent = ref({
  title: "",
  desc: "",
  contact_text: "",
  contact_link: "/console/tickets",
  email: "sales@example.com"
})

const productIconMap: Record<string, any> = {
  cloud: CloudIcon,
  rocket: RocketIcon,
  bolt: BoltIcon,
  building: BuildingIcon
}

const defaultProductBlocks = {
  hero: {
    sort_order: 1,
    visible: true,
    content: {
      badge: "",
      title: "",
      subtitle: "",
      features: [...heroFeatures.value]
    }
  },
  calculator: {
    sort_order: 2,
    visible: true,
    content: {
      title: "",
      desc: "",
      scenarios: [...scenarios.value]
    }
  },
  pricing: {
    sort_order: 3,
    visible: true,
    content: {
      products: products.value.map((item) => ({
        icon: Object.keys(productIconMap).find((key) => productIconMap[key] === item.icon) || "cloud",
        name: item.name,
        description: item.description,
        price: item.price,
        recommended: item.recommended,
        cta: item.cta,
        resources: item.resources,
        features: item.features
      }))
    }
  },
  comparison: {
    sort_order: 4,
    visible: true,
    content: {
      title: "",
      rows: [...comparisonRows.value]
    }
  },
  cta: {
    sort_order: 5,
    visible: true,
    content: {
      title: "",
      desc: "",
      contact_text: "",
      contact_link: "/console/tickets",
      email: "sales@example.com"
    }
  }
}

const parseContentJson = (raw: string) => {
  if (!raw) return {}
  try {
    return JSON.parse(raw)
  } catch (error) {
    return {}
  }
}

const mergeProductContent = (type: string, base: any, incoming: any) => {
  const merged = { ...base, ...incoming }
  if (type === "hero" && Array.isArray(incoming.features)) merged.features = incoming.features
  if (type === "calculator" && Array.isArray(incoming.scenarios)) merged.scenarios = incoming.scenarios
  if (type === "pricing" && Array.isArray(incoming.products)) merged.products = incoming.products
  if (type === "comparison" && Array.isArray(incoming.rows)) merged.rows = incoming.rows
  return merged
}

const resolveProducts = (items: any[]) =>
  items.map((item: any, index: number) => ({
    icon: productIconMap[item.icon] || products.value[index]?.icon || CloudIcon,
    name: item.name || "",
    description: item.description || "",
    price: item.price || "",
    recommended: !!item.recommended,
    cta: item.cta || "",
    resources: Array.isArray(item.resources) ? item.resources : [],
    features: Array.isArray(item.features) ? item.features : []
  }))

const applyProductBlocks = (blocks: any[]) => {
  const merged: Record<string, any> = {}
  Object.entries(defaultProductBlocks).forEach(([type, def]) => {
    merged[type] = {
      sort_order: def.sort_order,
      visible: def.visible,
      content: mergeProductContent(type, def.content, {})
    }
  })

  blocks.forEach((item) => {
    if (!item?.type || !merged[item.type]) return
    const content = parseContentJson(item.content_json || "")
    merged[item.type] = {
      sort_order: item.sort_order ?? merged[item.type].sort_order,
      visible: item.visible ?? merged[item.type].visible,
      content: mergeProductContent(item.type, merged[item.type].content, content)
    }
  })

  const ordered = Object.entries(merged)
    .sort((a, b) => a[1].sort_order - b[1].sort_order)
    .filter(([, value]) => value.visible)
    .map(([type]) => type)
  productsBlockOrder.value = ordered.length > 0 ? ordered : ["hero", "calculator", "pricing", "comparison", "cta"]

  const heroBlock = merged.hero?.content || defaultProductBlocks.hero.content
  const nextHeroFeatures = Array.isArray(heroBlock.features) && heroBlock.features.length > 0
    ? heroBlock.features
    : [...defaultProductBlocks.hero.content.features]
  heroFeatures.value = nextHeroFeatures
  heroContent.value = {
    badge: heroBlock.badge || "",
    title: heroBlock.title || "",
    subtitle: heroBlock.subtitle || "",
    features: nextHeroFeatures
  }

  const calcBlock = merged.calculator?.content || defaultProductBlocks.calculator.content
  calculatorContent.value = {
    title: calcBlock.title || "",
    desc: calcBlock.desc || ""
  }
  scenarios.value = Array.isArray(calcBlock.scenarios) && calcBlock.scenarios.length > 0
    ? calcBlock.scenarios
    : [...defaultProductBlocks.calculator.content.scenarios]

  const pricingBlock = merged.pricing?.content || defaultProductBlocks.pricing.content
  if (Array.isArray(pricingBlock.products) && pricingBlock.products.length > 0) {
    products.value = resolveProducts(pricingBlock.products)
  }

  const comparisonBlock = merged.comparison?.content || defaultProductBlocks.comparison.content
  comparisonContent.value = {
    title: comparisonBlock.title || ""
  }
  comparisonRows.value = Array.isArray(comparisonBlock.rows) && comparisonBlock.rows.length > 0
    ? comparisonBlock.rows
    : [...defaultProductBlocks.comparison.content.rows]

  const ctaBlock = merged.cta?.content || defaultProductBlocks.cta.content
  ctaContent.value = {
    title: ctaBlock.title || "",
    desc: ctaBlock.desc || "",
    contact_text: ctaBlock.contact_text || "",
    contact_link: ctaBlock.contact_link || "/console/tickets",
    email: ctaBlock.email || "sales@example.com"
  }

  if (selectedScenario.value !== null) {
    const idx = Math.min(selectedScenario.value, scenarios.value.length - 1)
    selectedScenario.value = idx >= 0 ? idx : null
    if (selectedScenario.value !== null) {
      selectedPlan.value = scenarios.value[selectedScenario.value]?.plan ?? 0
    }
  }
}

const fetchProductBlocks = async () => {
  try {
    const res = await getCmsBlocks({ page: "products", lang: "zh-CN" })
    const items = res.data?.items || []
    applyProductBlocks(items)
  } catch (error) {
    // fallback to defaults
  }
}

// Scroll animations
const initScrollAnimations = () => {
  const observer = new IntersectionObserver(
    (entries) => {
      entries.forEach(entry => {
        if (entry.isIntersecting) {
          entry.target.classList.add('scroll-animate-active')
        }
      })
    },
    { threshold: 0.1, rootMargin: '0px 0px -50px 0px' }
  )

  document.querySelectorAll('.scroll-animate').forEach(el => observer.observe(el))
}

onMounted(() => {
  fetchProductBlocks()
  setTimeout(initScrollAnimations, 100)
})

onUnmounted(() => {
  // Cleanup if needed
})
</script>

<style>
@import url('https://fonts.googleapis.com/css2?family=Outfit:wght@400;500;600;700;800&family=Work+Sans:wght@300;400;500;600&display=swap');

/* CSS Variables */
:root {
  --color-bg: #0a0e17;
  --color-bg-alt: #111827;
  --color-bg-card: #161f33;
  --color-primary: #0ea5e9;
  --color-primary-light: #38bdf8;
  --color-primary-dark: #0284c7;
  --color-accent: #f97316;
  --color-text: #f1f5f9;
  --color-text-muted: #94a3b8;
  --color-border: #1e293b;
  --color-success: #10b981;
  --font-heading: 'Outfit', sans-serif;
  --font-body: 'Work Sans', sans-serif;
}

/* Base */
.products-page {
  min-height: 100vh;
  font-family: var(--font-body);
  background: var(--color-bg);
  color: var(--color-text);
  position: relative;
  overflow-x: hidden;
}

/* Background */
.products-page .page-background {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  pointer-events: none;
  z-index: 0;
}

.products-page .grid-overlay {
  position: absolute;
  inset: 0;
  background-image:
    linear-gradient(rgba(14, 165, 233, 0.03) 1px, transparent 1px),
    linear-gradient(90deg, rgba(14, 165, 233, 0.03) 1px, transparent 1px);
  background-size: 50px 50px;
}

.products-page .glow-orb {
  position: absolute;
  border-radius: 50%;
  filter: blur(150px);
  opacity: 0.15;
}

.products-page .glow-1 {
  width: 600px;
  height: 600px;
  background: var(--color-primary);
  top: -200px;
  right: -200px;
  animation: orbFloat1 25s ease-in-out infinite;
}

.products-page .glow-2 {
  width: 500px;
  height: 500px;
  background: var(--color-accent);
  top: 50%;
  left: -200px;
  animation: orbFloat2 30s ease-in-out infinite;
}

.products-page .glow-3 {
  width: 400px;
  height: 400px;
  background: var(--color-success);
  bottom: 0;
  right: 20%;
  animation: orbFloat3 20s ease-in-out infinite;
}

@keyframes orbFloat1 {
  0%, 100% { transform: translate(0, 0); }
  50% { transform: translate(-100px, 100px); }
}

@keyframes orbFloat2 {
  0%, 100% { transform: translate(0, 0); }
  50% { transform: translate(100px, -100px); }
}

@keyframes orbFloat3 {
  0%, 100% { transform: translate(0, 0); }
  50% { transform: translate(-50px, 50px); }
}

/* Hero Section */
.products-page .products-hero {
  position: relative;
  padding: 120px 24px 80px;
  text-align: center;
  z-index: 1;
}

.products-page .products-hero .hero-container {
  max-width: 1200px;
  margin: 0 auto;
  /* Prevent Home.vue's global `.hero-container { display: grid; ... }` from leaking into products hero (e.g. in CMS preview). */
  display: block;
  padding: 0;
}

.products-page .hero-badge {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 8px 20px;
  background: rgba(14, 165, 233, 0.1);
  border: 1px solid rgba(14, 165, 233, 0.2);
  border-radius: 100px;
  font-size: 14px;
  font-weight: 500;
  color: var(--color-primary-light);
  margin-bottom: 24px;
}

.products-page .badge-dot {
  width: 6px;
  height: 6px;
  background: var(--color-success);
  border-radius: 50%;
  animation: pulse 2s infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; transform: scale(1); }
  50% { opacity: 0.5; transform: scale(1.2); }
}

.products-page .hero-title {
  font-family: var(--font-heading);
  font-size: 48px;
  font-weight: 700;
  margin: 0 0 16px;
  letter-spacing: -0.02em;
  background: linear-gradient(135deg, var(--color-text) 0%, var(--color-primary-light) 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.products-page .hero-subtitle {
  font-size: 18px;
  line-height: 1.7;
  color: var(--color-text-muted);
  margin: 0 0 40px;
  max-width: 600px;
  margin-left: auto;
  margin-right: auto;
}

.products-page .hero-features {
  display: flex;
  justify-content: center;
  gap: 32px;
  flex-wrap: wrap;
}

.products-page .hero-feature {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  color: var(--color-text-muted);
}

.products-page .hero-feature svg {
  color: var(--color-success);
}

/* Calculator Section */
.products-page .calculator-section {
  padding: 60px 24px;
  z-index: 1;
  position: relative;
}

.products-page .calculator-container {
  max-width: 900px;
  margin: 0 auto;
  background: var(--color-bg-alt);
  border: 1px solid var(--color-border);
  border-radius: 24px;
  padding: 48px;
  text-align: center;
}

.products-page .section-title {
  font-family: var(--font-heading);
  font-size: 32px;
  font-weight: 700;
  margin: 0 0 8px;
}
.products-page .calculator-section .section-title,
.products-page .comparison-section .section-title {
  color: var(--color-text);
}

.products-page .section-desc {
  font-size: 16px;
  color: var(--color-text-muted);
  margin: 0 0 32px;
}

.products-page .scenario-buttons {
  display: flex;
  justify-content: center;
  gap: 16px;
  flex-wrap: wrap;
  margin-bottom: 32px;
}

.products-page .scenario-btn {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px 24px;
  background: var(--color-bg);
  border: 1px solid var(--color-border);
  border-radius: 12px;
  cursor: pointer;
  transition: all 0.3s;
  font-family: var(--font-body);
  font-size: 15px;
  color: var(--color-text-muted);
}

.products-page .scenario-btn:hover {
  border-color: var(--color-primary);
  color: var(--color-text);
}

.products-page .scenario-btn.active {
  background: rgba(14, 165, 233, 0.1);
  border-color: var(--color-primary);
  color: var(--color-primary-light);
}

.products-page .scenario-icon {
  font-size: 20px;
}

.products-page .scenario-name {
  font-weight: 500;
}

.products-page .recommendation {
  display: inline-flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 16px 32px;
  background: rgba(16, 185, 129, 0.1);
  border: 1px solid rgba(16, 185, 129, 0.2);
  border-radius: 12px;
}

.products-page .rec-label {
  font-size: 13px;
  color: var(--color-success);
  font-weight: 600;
  text-transform: uppercase;
}

.products-page .rec-config {
  font-size: 18px;
  font-weight: 600;
  color: var(--color-text);
}

/* Pricing Section */
.products-page .pricing-section {
  padding: 80px 24px;
  position: relative;
  z-index: 1;
}

.products-page .pricing-container {
  max-width: 1400px;
  margin: 0 auto;
}

.products-page .pricing-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 24px;
  align-items: stretch;
}

.products-page .pricing-card {
  position: relative;
  background: var(--color-bg-alt);
  border: 1px solid var(--color-border);
  border-radius: 24px;
  padding: 32px;
  transition: all 0.4s cubic-bezier(0.4, 0, 0.2, 1);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.products-page .pricing-card:hover {
  transform: translateY(-8px);
}

.products-page .pricing-card.recommended {
  border-color: var(--color-primary);
  background: linear-gradient(180deg, rgba(14, 165, 233, 0.05) 0%, var(--color-bg-alt) 100%);
}

.products-page .pricing-card.selected {
  border-color: var(--color-primary);
  box-shadow: 0 0 0 2px var(--color-primary);
}

.products-page .recommended-badge {
  position: absolute;
  top: 16px;
  right: 16px;
  padding: 6px 14px;
  background: linear-gradient(135deg, var(--color-primary) 0%, var(--color-primary-dark) 100%);
  border-radius: 100px;
  font-size: 12px;
  font-weight: 600;
  color: white;
}

.products-page .card-header {
  text-align: center;
  margin-bottom: 24px;
}

.products-page .product-icon {
  width: 64px;
  height: 64px;
  margin: 0 auto 16px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 16px;
  color: white;
}

.products-page .icon-1 { background: linear-gradient(135deg, #6366f1 0%, #4f46e5 100%); }
.products-page .icon-2 { background: linear-gradient(135deg, #0ea5e9 0%, #0284c7 100%); }
.products-page .icon-3 { background: linear-gradient(135deg, #f97316 0%, #ea580c 100%); }
.products-page .icon-4 { background: linear-gradient(135deg, #10b981 0%, #059669 100%); }

.products-page .product-icon svg {
  width: 32px;
  height: 32px;
}

.products-page .product-name {
  font-family: var(--font-heading);
  font-size: 24px;
  font-weight: 700;
  margin: 0 0 8px;
  color: var(--color-text);
}

.products-page .product-desc {
  font-size: 14px;
  color: var(--color-text-muted);
  margin: 0 0 16px;
}

.products-page .product-price {
  display: flex;
  align-items: baseline;
  justify-content: center;
  gap: 4px;
}

.products-page .price-currency {
  font-size: 18px;
  color: var(--color-text-muted);
}

.products-page .price-value {
  font-family: var(--font-heading);
  font-size: 40px;
  font-weight: 700;
  color: var(--color-text);
}

.products-page .price-unit {
  font-size: 16px;
  color: var(--color-text-muted);
}

/* Resource Bars */
.products-page .resource-bars {
  margin-bottom: 24px;
}

.products-page .resource-item {
  margin-bottom: 16px;
}

.products-page .resource-item:last-child {
  margin-bottom: 0;
}

.products-page .resource-header {
  display: flex;
  justify-content: space-between;
  margin-bottom: 6px;
}

.products-page .resource-label {
  font-size: 13px;
  color: var(--color-text-muted);
}

.products-page .resource-value {
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text);
}

.products-page .resource-bar {
  height: 6px;
  background: var(--color-bg);
  border-radius: 3px;
  overflow: hidden;
}

.products-page .resource-fill {
  height: 100%;
  background: linear-gradient(90deg, var(--color-primary) 0%, var(--color-primary-light) 100%);
  border-radius: 3px;
  transition: width 1s ease-out;
}

.products-page .pricing-card:nth-child(2) .resource-fill {
  background: linear-gradient(90deg, #6366f1 0%, #818cf8 100%);
}

.products-page .pricing-card:nth-child(3) .resource-fill {
  background: linear-gradient(90deg, #f97316 0%, #fbbf24 100%);
}

.products-page .pricing-card:nth-child(4) .resource-fill {
  background: linear-gradient(90deg, #10b981 0%, #34d399 100%);
}

/* Features List */
.products-page .features-list {
  flex: 1;
  margin-bottom: 24px;
}

.products-page .feature-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 0;
  font-size: 14px;
  color: var(--color-text-muted);
}

.products-page .feature-check {
  width: 20px;
  height: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(16, 185, 129, 0.1);
  border-radius: 6px;
  color: var(--color-success);
  flex-shrink: 0;
}

/* Card Button */
.products-page .card-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 14px 24px;
  border-radius: 12px;
  font-size: 15px;
  font-weight: 600;
  text-decoration: none;
  transition: all 0.3s;
  cursor: pointer;
}

.products-page .btn-primary {
  background: linear-gradient(135deg, var(--color-primary) 0%, var(--color-primary-dark) 100%);
  color: white;
  box-shadow: 0 4px 20px rgba(14, 165, 233, 0.3);
}

.products-page .btn-primary:hover {
  transform: translateY(-2px);
  box-shadow: 0 6px 30px rgba(14, 165, 233, 0.4);
}

.products-page .btn-secondary {
  background: transparent;
  color: var(--color-text);
  border: 1px solid var(--color-border);
}

.products-page .btn-secondary:hover {
  background: rgba(255, 255, 255, 0.05);
  border-color: var(--color-text-muted);
}

/* Card Glow */
.products-page .card-glow {
  position: absolute;
  inset: 0;
  background: radial-gradient(circle at var(--mouse-x, 50%) var(--mouse-y, 50%), rgba(14, 165, 233, 0.1), transparent 50%);
  opacity: 0;
  transition: opacity 0.3s;
  pointer-events: none;
}

.products-page .pricing-card:hover .card-glow {
  opacity: 1;
}

/* Comparison Section */
.products-page .comparison-section {
  padding: 80px 24px;
  position: relative;
  z-index: 1;
}

.products-page .comparison-container {
  max-width: 1200px;
  margin: 0 auto;
}

.products-page .comparison-table {
  background: var(--color-bg-alt);
  border: 1px solid var(--color-border);
  border-radius: 20px;
  overflow: hidden;
}

.products-page .comparison-row {
  display: grid;
  grid-template-columns: 150px repeat(4, 1fr);
  border-bottom: 1px solid var(--color-border);
}

.products-page .comparison-row:last-child {
  border-bottom: none;
}

.products-page .comparison-row.header {
  background: rgba(14, 165, 233, 0.05);
}

.products-page .comparison-row.header .col-feature {
  font-weight: 600;
  color: var(--color-text-muted);
}

.products-page .comparison-row.header .col-plan {
  font-weight: 600;
  color: var(--color-primary-light);
}

.products-page .col-feature, .products-page .col-plan {
  padding: 16px 20px;
  display: flex;
  align-items: center;
}

.products-page .col-feature {
  color: var(--color-text-muted);
  font-size: 14px;
  border-right: 1px solid var(--color-border);
}

.products-page .col-plan {
  font-size: 14px;
  color: var(--color-text);
  justify-content: center;
}

.products-page .comparison-row:not(.header) .col-plan:nth-child(3) {
  background: rgba(14, 165, 233, 0.03);
}

/* CTA Section */
.products-page .cta-section {
  padding: 80px 24px 120px;
  position: relative;
  z-index: 1;
}

.products-page .cta-section .cta-container {
  max-width: 1200px;
  margin: 0 auto;
  text-align: center;
}

.products-page .cta-section .cta-title {
  font-family: var(--font-heading);
  font-size: 32px;
  font-weight: 700;
  margin: 0 0 12px;
}

.products-page .cta-section .cta-desc {
  font-size: 16px;
  color: var(--color-text-muted);
  margin: 0 0 32px;
}

.products-page .cta-section .cta-actions {
  display: flex;
  justify-content: center;
  gap: 16px;
  flex-wrap: wrap;
}

.products-page .cta-section .btn {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 14px 28px;
  border-radius: 12px;
  font-size: 15px;
  font-weight: 600;
  text-decoration: none;
  transition: all 0.3s;
}

/* Scroll Animations */
.products-page .scroll-animate {
  opacity: 0;
  transform: translateY(30px);
  transition: all 0.6s cubic-bezier(0.4, 0, 0.2, 1);
}

.products-page .scroll-animate-active {
  opacity: 1;
  transform: translateY(0);
}

/* Responsive */
@media (max-width: 1200px) {
  .products-page .pricing-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 768px) {
  .products-page .hero-title {
    font-size: 36px;
  }

  .products-page .pricing-grid {
    grid-template-columns: 1fr;
  }

  .products-page .comparison-table {
    overflow-x: auto;
  }

  .products-page .comparison-row {
    min-width: 600px;
  }

  .products-page .calculator-container {
    padding: 32px 24px;
  }

  .products-page .cta-section .cta-container {
    padding: 32px 24px;
  }

  .products-page .hero-features {
    flex-direction: column;
    gap: 12px;
  }

  .products-page .page-background {
    display: none;
  }
}
</style>


