<template>
  <div class="home-page" ref="homePage">
    <canvas ref="particleCanvas" class="particle-canvas"></canvas>
    <div class="mouse-glow" ref="mouseGlow"></div>

    <template v-for="type in homeBlockOrder" :key="type">
      <HomeHeroBlock
        v-if="type === 'hero'"
        :hero-content="heroContent"
        :typewriter-text="typewriterText"
        :stats="stats"
        :animated-stats="animatedStats"
        :hero-cards="heroCards"
        :handle-tilt="handleTilt"
        :reset-tilt="resetTilt"
        :card1-ref="card1"
        :card2-ref="card2"
        :card3-ref="card3"
      />
      <HomeFeaturesBlock
        v-else-if="type === 'features'"
        :content="featuresContent"
        :features="features"
        :handle-feature-glow="handleFeatureGlow"
        :reset-feature-glow="resetFeatureGlow"
        :register-feature-bg="registerFeatureBg"
      />
      <HomeProductsBlock
        v-else-if="type === 'products'"
        :content="productsContent"
        :products="products"
      />
      <HomeCtaBlock
        v-else-if="type === 'cta'"
        :content="ctaContent"
        :features="ctaFeatures"
      />
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, nextTick, reactive } from "vue";
import { getCmsBlocks } from "@/services/user";
import HomeHeroBlock from "@/components/cms/blocks/home/HomeHeroBlock.vue";
import HomeFeaturesBlock from "@/components/cms/blocks/home/HomeFeaturesBlock.vue";
import HomeProductsBlock from "@/components/cms/blocks/home/HomeProductsBlock.vue";
import HomeCtaBlock from "@/components/cms/blocks/home/HomeCtaBlock.vue";
import {
  ThunderIcon,
  ShieldIcon,
  GlobeIcon,
  ServerIcon,
  DatabaseIcon,
  SettingsIcon,
  CloudIcon,
  CubeIcon,
  CodeIcon,
} from "@/components/cms/blocks/home/icons";

// Refs
const homePage = ref<HTMLElement | null>(null);
const particleCanvas = ref<HTMLCanvasElement | null>(null);
const mouseGlow = ref<HTMLElement | null>(null);
const statNumbers = ref<HTMLElement[]>([]);
const card1 = ref<HTMLElement | null>(null);
const card2 = ref<HTMLElement | null>(null);
const card3 = ref<HTMLElement | null>(null);
const featuresSection = ref<HTMLElement | null>(null);
const productsSection = ref<HTMLElement | null>(null);
const ctaSection = ref<HTMLElement | null>(null);
const featureBgs = ref<HTMLElement[]>([]);

const setFeaturesSection = (el: HTMLElement | null) => {
  featuresSection.value = el;
};

const setProductsSection = (el: HTMLElement | null) => {
  productsSection.value = el;
};

const setCtaSection = (el: HTMLElement | null) => {
  ctaSection.value = el;
};

const registerFeatureBg = (el: HTMLElement | null, index: number) => {
  if (!el) return;
  featureBgs.value[index] = el;
};

const heroContent = reactive({
  badge: "",
  title1: "",
  subtitle: "",
  primary_button_text: "",
  primary_button_link: "/register",
  secondary_button_text: "",
  secondary_button_link: "/products",
});

const heroCards = ref([
  { title: "极速部署", desc: "60秒开机" },
  { title: "全球网络", desc: "覆盖150+国家" },
  { title: "多层防护", desc: "DDoS防御" },
]);

const featuresContent = reactive({
  badge: "",
  title: "",
  desc: "",
});

const productsContent = reactive({
  badge: "",
  title: "",
});

const ctaContent = reactive({
  title: "",
  desc: "",
  button_text: "",
  button_link: "/register",
});

// Typewriter effect
const typewriterText = ref("");
const typewriterWords = ref(["云端智能", "无限可能", "卓越性能", "安全可靠"]);
let typewriterIndex = 0;
let typewriterCharIndex = 0;
let typewriterDeleting = false;
let typewriterTimeout: number | null = null;

const startTypewriter = () => {
  if (!typewriterWords.value.length) {
    typewriterText.value = "";
    return;
  }
  const currentWord = typewriterWords.value[typewriterIndex] || "";

  if (typewriterDeleting) {
    typewriterText.value = currentWord.substring(0, typewriterCharIndex - 1);
    typewriterCharIndex--;
  } else {
    typewriterText.value = currentWord.substring(0, typewriterCharIndex + 1);
    typewriterCharIndex++;
  }

  let typeSpeed = typewriterDeleting ? 50 : 100;

  if (!typewriterDeleting && typewriterCharIndex === currentWord.length) {
    typeSpeed = 2000;
    typewriterDeleting = true;
  } else if (typewriterDeleting && typewriterCharIndex === 0) {
    typewriterDeleting = false;
    const total = typewriterWords.value.length || 1;
    typewriterIndex = (typewriterIndex + 1) % total;
    typeSpeed = 500;
  }

  typewriterTimeout = window.setTimeout(startTypewriter, typeSpeed);
};

// Animated stats
const stats = ref([
  { value: 99.99, suffix: "%", label: "可用性" },
  { value: 50, suffix: "+", label: "全球节点" },
  { value: 100, suffix: "K+", label: "企业用户" },
]);
const animatedStats = ref(stats.value.map(() => "0"));

const animateStats = () => {
  stats.value.forEach((stat, index) => {
    const targetValue = stat.value;
    const duration = 2000;
    const startTime = performance.now();

    const animate = (currentTime: number) => {
      const elapsed = currentTime - startTime;
      const progress = Math.min(elapsed / duration, 1);
      const easeOutQuart = 1 - Math.pow(1 - progress, 4);
      const currentValue = targetValue * easeOutQuart;

      if (stat.value % 1 !== 0) {
        animatedStats.value[index] = currentValue.toFixed(2);
      } else {
        animatedStats.value[index] = Math.floor(currentValue).toString();
      }

      if (progress < 1) {
        requestAnimationFrame(animate);
      }
    };

    requestAnimationFrame(animate);
  });
};

// Particle animation
class Particle {
  x: number;
  y: number;
  vx: number;
  vy: number;
  radius: number;
  opacity: number;

  constructor(canvas: HTMLCanvasElement) {
    this.x = Math.random() * canvas.width;
    this.y = Math.random() * canvas.height;
    this.vx = (Math.random() - 0.5) * 0.5;
    this.vy = (Math.random() - 0.5) * 0.5;
    this.radius = Math.random() * 2 + 1;
    this.opacity = Math.random() * 0.5 + 0.2;
  }

  update(canvas: HTMLCanvasElement) {
    this.x += this.vx;
    this.y += this.vy;

    if (this.x < 0 || this.x > canvas.width) this.vx *= -1;
    if (this.y < 0 || this.y > canvas.height) this.vy *= -1;
  }

  draw(ctx: CanvasRenderingContext2D) {
    ctx.beginPath();
    ctx.arc(this.x, this.y, this.radius, 0, Math.PI * 2);
    ctx.fillStyle = `rgba(14, 165, 233, ${this.opacity})`;
    ctx.fill();
  }
}

let particles: Particle[] = [];
let animationId: number | null = null;

const initParticles = () => {
  const canvas = particleCanvas.value;
  if (!canvas) return;

  canvas.width = window.innerWidth;
  canvas.height = window.innerHeight;

  particles = [];
  const particleCount = Math.floor((canvas.width * canvas.height) / 15000);

  for (let i = 0; i < particleCount; i++) {
    particles.push(new Particle(canvas));
  }
};

const animateParticles = () => {
  const canvas = particleCanvas.value;
  const ctx = canvas?.getContext("2d");
  if (!canvas || !ctx) return;

  ctx.clearRect(0, 0, canvas.width, canvas.height);

  // Draw connections
  particles.forEach((particle, i) => {
    particle.update(canvas);
    particle.draw(ctx);

    // Connect nearby particles
    particles.slice(i + 1).forEach((otherParticle) => {
      const dx = particle.x - otherParticle.x;
      const dy = particle.y - otherParticle.y;
      const distance = Math.sqrt(dx * dx + dy * dy);

      if (distance < 150) {
        ctx.beginPath();
        ctx.strokeStyle = `rgba(14, 165, 233, ${0.1 * (1 - distance / 150)})`;
        ctx.lineWidth = 1;
        ctx.moveTo(particle.x, particle.y);
        ctx.lineTo(otherParticle.x, otherParticle.y);
        ctx.stroke();
      }
    });
  });

  animationId = requestAnimationFrame(animateParticles);
};

// Mouse glow effect
const handleMouseMove = (e: MouseEvent) => {
  if (!mouseGlow.value) return;

  const glow = mouseGlow.value;
  glow.style.left = `${e.clientX - 200}px`;
  glow.style.top = `${e.clientY - 200}px`;
};

// 3D Card tilt effect
const handleTilt = (e: MouseEvent, cardName: string) => {
  const card =
    cardName === "card1"
      ? card1.value
      : cardName === "card2"
        ? card2.value
        : card3.value;
  if (!card) return;

  const rect = card.getBoundingClientRect();
  const x = e.clientX - rect.left;
  const y = e.clientY - rect.top;
  const centerX = rect.width / 2;
  const centerY = rect.height / 2;
  const rotateX = (y - centerY) / 10;
  const rotateY = (centerX - x) / 10;

  card.style.transform = `perspective(1000px) rotateX(${rotateX}deg) rotateY(${rotateY}deg) translateZ(10px)`;
};

const resetTilt = (cardName: string) => {
  const card =
    cardName === "card1"
      ? card1.value
      : cardName === "card2"
        ? card2.value
        : card3.value;
  if (!card) return;

  card.style.transform =
    "perspective(1000px) rotateX(0) rotateY(0) translateZ(0)";
};

// Feature card glow effect
const handleFeatureGlow = (e: MouseEvent, index: number) => {
  const card = e.currentTarget as HTMLElement;
  const rect = card.getBoundingClientRect();
  const x = e.clientX - rect.left;
  const y = e.clientY - rect.top;

  card.style.setProperty("--mouse-x", `${x}px`);
  card.style.setProperty("--mouse-y", `${y}px`);

  const glow = card.querySelector(`.feature-glow-${index}`) as HTMLElement;
  if (glow) {
    glow.style.left = `${x}px`;
    glow.style.top = `${y}px`;
  }
};

const resetFeatureGlow = (index: number) => {
  // Reset handled by CSS
};

// Scroll animations
const initScrollAnimations = () => {
  const observer = new IntersectionObserver(
    (entries) => {
      entries.forEach((entry) => {
        if (entry.isIntersecting) {
          entry.target.classList.add("scroll-animate-active");

          // Trigger stat animation when stats section is visible
          if (entry.target.classList.contains("stat-item")) {
            animateStats();
          }
        }
      });
    },
    { threshold: 0.1, rootMargin: "0px 0px -50px 0px" },
  );

  document
    .querySelectorAll(".scroll-animate")
    .forEach((el) => observer.observe(el));
};

// CTA features
const ctaFeatures = ref(["无需绑定信用卡", "随时取消", "24/7 技术支持"]);

const features = ref([
  {
    icon: ThunderIcon,
    title: "极致性能",
    description: "采用最新一代CPU和NVMe SSD，提供卓越计算性能和I/O吞吐量",
  },
  {
    icon: ShieldIcon,
    title: "安全可靠",
    description: "多层安全防护体系，DDoS防护、WAF、SSL证书全方位保障",
  },
  {
    icon: GlobeIcon,
    title: "全球覆盖",
    description: "50+ 数据中心遍布全球，BGP多线接入，智能调度最优线路",
  },
  {
    icon: ServerIcon,
    title: "弹性伸缩",
    description: "秒级扩容缩容，按需付费，资源利用率最大化",
  },
  {
    icon: DatabaseIcon,
    title: "数据保护",
    description: "多重备份机制，快照回滚，异地容灾，数据安全无忧",
  },
  {
    icon: SettingsIcon,
    title: "简单易用",
    description: "可视化控制台，一键部署应用，API 丰富的自动化运维",
  },
]);

const products = ref([
  {
    icon: CloudIcon,
    tag: "入门首选",
    title: "云服务器",
    description: "适合个人开发者、小型项目",
    price: "29",
  },
  {
    icon: CubeIcon,
    tag: "企业推荐",
    title: "弹性计算",
    description: "适合中型企业、Web 应用",
    price: "99",
  },
  {
    icon: CodeIcon,
    tag: "性能旗舰",
    title: "GPU 实例",
    description: "适合 AI 训练、渲染任务",
    price: "399",
  },
]);

const featureIconMap: Record<string, any> = {
  thunder: ThunderIcon,
  shield: ShieldIcon,
  globe: GlobeIcon,
  server: ServerIcon,
  database: DatabaseIcon,
  settings: SettingsIcon,
};

const productIconMap: Record<string, any> = {
  cloud: CloudIcon,
  cube: CubeIcon,
  code: CodeIcon,
};

const homeBlockOrder = ref<string[]>(["hero", "features", "products", "cta"]);

const defaultHomeBlocks = {
  hero: {
    sort_order: 1,
    visible: true,
    content: {
      badge: "",
      title1: "",
      subtitle: "",
      primary_button_text: "",
      primary_button_link: "/register",
      secondary_button_text: "",
      secondary_button_link: "/products",
      typewriter_words: [...typewriterWords.value],
      cards: [...heroCards.value],
      stats: [...stats.value],
    },
  },
  features: {
    sort_order: 2,
    visible: true,
    content: {
      badge: "",
      title: "",
      desc: "",
      items: features.value.map((item) => ({
        icon:
          Object.keys(featureIconMap).find(
            (key) => featureIconMap[key] === item.icon,
          ) || "thunder",
        title: item.title,
        description: item.description,
      })),
    },
  },
  products: {
    sort_order: 3,
    visible: true,
    content: {
      badge: "",
      title: "",
      items: products.value.map((item) => ({
        icon:
          Object.keys(productIconMap).find(
            (key) => productIconMap[key] === item.icon,
          ) || "cloud",
        tag: item.tag,
        title: item.title,
        description: item.description,
        price: item.price,
      })),
    },
  },
  cta: {
    sort_order: 4,
    visible: true,
    content: {
      title: "",
      desc: "",
      button_text: "",
      button_link: "/register",
      features: [...ctaFeatures.value],
    },
  },
};

const parseContentJson = (raw: string) => {
  if (!raw) return {};
  try {
    return JSON.parse(raw);
  } catch (error) {
    return {};
  }
};

const resolveFeatureItems = (items: any[]) =>
  items.map((item: any, index: number) => ({
    icon:
      featureIconMap[item.icon] || features.value[index]?.icon || ThunderIcon,
    title: item.title || "",
    description: item.description || "",
  }));

const resolveProductItems = (items: any[]) =>
  items.map((item: any, index: number) => ({
    icon: productIconMap[item.icon] || products.value[index]?.icon || CloudIcon,
    tag: item.tag || "",
    title: item.title || "",
    description: item.description || "",
    price: item.price || "",
  }));

const mergeBlockContent = (type: string, base: any, incoming: any) => {
  const merged = { ...base, ...incoming };
  if (type === "hero") {
    if (Array.isArray(incoming.typewriter_words))
      merged.typewriter_words = incoming.typewriter_words;
    if (Array.isArray(incoming.cards)) merged.cards = incoming.cards;
    if (Array.isArray(incoming.stats)) merged.stats = incoming.stats;
  }
  if (type === "features" && Array.isArray(incoming.items))
    merged.items = incoming.items;
  if (type === "products" && Array.isArray(incoming.items))
    merged.items = incoming.items;
  if (type === "cta" && Array.isArray(incoming.features))
    merged.features = incoming.features;
  return merged;
};

const applyHomeBlocks = (blocks: any[]) => {
  const mergedBlocks: Record<string, any> = {};
  Object.entries(defaultHomeBlocks).forEach(([type, def]) => {
    mergedBlocks[type] = {
      sort_order: def.sort_order,
      visible: def.visible,
      content: mergeBlockContent(type, def.content, {}),
    };
  });

  blocks.forEach((item) => {
    if (!item?.type || !mergedBlocks[item.type]) return;
    const content = parseContentJson(item.content_json || "");
    mergedBlocks[item.type] = {
      sort_order: item.sort_order ?? mergedBlocks[item.type].sort_order,
      visible: item.visible ?? mergedBlocks[item.type].visible,
      content: mergeBlockContent(
        item.type,
        mergedBlocks[item.type].content,
        content,
      ),
    };
  });

  const statsBlock = blocks.find((item) => item?.type === "stats");
  if (statsBlock) {
    const statsContent = parseContentJson(statsBlock.content_json || "");
    const statsItems = Array.isArray(statsContent.items)
      ? statsContent.items
      : Array.isArray(statsContent.stats)
        ? statsContent.stats
        : [];
    const heroStats = mergedBlocks.hero.content?.stats;
    const shouldApplyStats =
      Array.isArray(statsItems) &&
      statsItems.length > 0 &&
      (!Array.isArray(heroStats) || heroStats.length === 0);
    if (shouldApplyStats) {
      mergedBlocks.hero.content = mergeBlockContent(
        "hero",
        mergedBlocks.hero.content,
        { stats: statsItems },
      );
      mergedBlocks.hero.visible = true;
    }
  }

  const ordered = Object.entries(mergedBlocks)
    .sort((a, b) => a[1].sort_order - b[1].sort_order)
    .filter(([, value]) => value.visible)
    .map(([type]) => type);
  homeBlockOrder.value =
    ordered.length > 0 ? ordered : ["hero", "features", "products", "cta"];

  const heroBlock =
    mergedBlocks.hero?.content || defaultHomeBlocks.hero.content;
  heroContent.badge = heroBlock.badge || "";
  heroContent.title1 = heroBlock.title1 || "";
  heroContent.subtitle = heroBlock.subtitle || "";
  heroContent.primary_button_text = heroBlock.primary_button_text || "";
  heroContent.primary_button_link =
    heroBlock.primary_button_link || "/register";
  heroContent.secondary_button_text = heroBlock.secondary_button_text || "";
  heroContent.secondary_button_link =
    heroBlock.secondary_button_link || "/products";

  if (
    Array.isArray(heroBlock.typewriter_words) &&
    heroBlock.typewriter_words.length > 0
  ) {
    typewriterWords.value = heroBlock.typewriter_words;
    typewriterIndex = 0;
    typewriterCharIndex = 0;
    typewriterDeleting = false;
    typewriterText.value = "";
  }
  if (Array.isArray(heroBlock.cards) && heroBlock.cards.length > 0) {
    heroCards.value = heroBlock.cards;
  }
  if (Array.isArray(heroBlock.stats) && heroBlock.stats.length > 0) {
    stats.value = heroBlock.stats;
    animatedStats.value = stats.value.map(() => "0");
  }

  const featuresBlock =
    mergedBlocks.features?.content || defaultHomeBlocks.features.content;
  featuresContent.badge = featuresBlock.badge || "";
  featuresContent.title = featuresBlock.title || "";
  featuresContent.desc = featuresBlock.desc || "";
  if (Array.isArray(featuresBlock.items) && featuresBlock.items.length > 0) {
    features.value = resolveFeatureItems(featuresBlock.items);
  }

  const productsBlock =
    mergedBlocks.products?.content || defaultHomeBlocks.products.content;
  productsContent.badge = productsBlock.badge || "";
  productsContent.title = productsBlock.title || "";
  if (Array.isArray(productsBlock.items) && productsBlock.items.length > 0) {
    products.value = resolveProductItems(productsBlock.items);
  }

  const ctaBlock = mergedBlocks.cta?.content || defaultHomeBlocks.cta.content;
  ctaContent.title = ctaBlock.title || "";
  ctaContent.desc = ctaBlock.desc || "";
  ctaContent.button_text = ctaBlock.button_text || "";
  ctaContent.button_link = ctaBlock.button_link || "/register";
  if (Array.isArray(ctaBlock.features) && ctaBlock.features.length > 0) {
    ctaFeatures.value = ctaBlock.features;
  }
};

const fetchHomeBlocks = async () => {
  try {
    const res = await getCmsBlocks({ page: "home", lang: "zh-CN" });
    const items = res.data?.items || [];
    applyHomeBlocks(items);
  } catch (error) {
    // fallback to defaults
  }
};

// Lifecycle
onMounted(async () => {
  await nextTick();

  await fetchHomeBlocks();

  // Initialize particles
  initParticles();
  animateParticles();

  // Start typewriter effect
  startTypewriter();

  // Start stat animation with delay
  setTimeout(animateStats, 500);

  // Initialize scroll animations
  setTimeout(initScrollAnimations, 100);

  // Mouse move listener
  window.addEventListener("mousemove", handleMouseMove);

  // Window resize listener
  window.addEventListener("resize", initParticles);
});

onUnmounted(() => {
  if (typewriterTimeout) clearTimeout(typewriterTimeout);
  if (animationId) cancelAnimationFrame(animationId);

  window.removeEventListener("mousemove", handleMouseMove);
  window.removeEventListener("resize", initParticles);
});
</script>

<style>
@import url("https://fonts.googleapis.com/css2?family=Outfit:wght@400;500;600;700;800&family=Work+Sans:wght@300;400;500;600&display=swap");

/* CSS Variables */
:root {
  --color-bg: #0a0e17;
  --color-bg-alt: #111827;
  --color-primary: #0ea5e9;
  --color-primary-light: #38bdf8;
  --color-accent: #f97316;
  --color-text: #f1f5f9;
  --color-text-muted: #94a3b8;
  --color-border: #1e293b;
  --color-success: #10b981;
  --font-heading: "Outfit", sans-serif;
  --font-body: "Work Sans", sans-serif;
}

/* Base */
.home-page {
  min-height: 100vh;
  font-family: var(--font-body);
  background: var(--color-bg);
  color: var(--color-text);
  position: relative;
  overflow-x: hidden;
}

/* Particle Canvas */
.particle-canvas {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  pointer-events: none;
  z-index: 0;
}

/* Mouse Glow */
.mouse-glow {
  position: fixed;
  width: 400px;
  height: 400px;
  background: radial-gradient(
    circle,
    rgba(14, 165, 233, 0.08) 0%,
    transparent 70%
  );
  pointer-events: none;
  z-index: 1;
  transition: transform 0.1s ease-out;
}

/* Hero Section */
.hero-section {
  position: relative;
  min-height: 100vh;
  display: flex;
  align-items: center;
  overflow: hidden;
  background:
    radial-gradient(
      ellipse 80% 50% at 50% -20%,
      rgba(14, 165, 233, 0.15),
      transparent
    ),
    radial-gradient(
      ellipse 60% 40% at 80% 60%,
      rgba(249, 115, 22, 0.1),
      transparent
    );
}

.hero-grid {
  position: absolute;
  inset: 0;
  overflow: hidden;
  z-index: 0;
}

.grid-line {
  position: absolute;
  top: 0;
  bottom: 0;
  width: 1px;
  background: linear-gradient(
    to bottom,
    transparent,
    rgba(14, 165, 233, 0.1) 20%,
    rgba(14, 165, 233, 0.1) 80%,
    transparent
  );
  animation: gridPulse 4s ease-in-out infinite;
}

.grid-line:nth-child(odd) {
  animation-delay: 2s;
}

.grid-line.horizontal {
  top: auto;
  left: 0;
  right: 0;
  height: 1px;
  width: auto;
  background: linear-gradient(
    to right,
    transparent,
    rgba(14, 165, 233, 0.1) 20%,
    rgba(14, 165, 233, 0.1) 80%,
    transparent
  );
}

@keyframes gridPulse {
  0%,
  100% {
    opacity: 0.3;
  }
  50% {
    opacity: 0.8;
  }
}

.hero-glow {
  position: absolute;
  border-radius: 50%;
  filter: blur(100px);
  opacity: 0.5;
  pointer-events: none;
  z-index: 0;
}

.glow-1 {
  width: 600px;
  height: 600px;
  background: var(--color-primary);
  top: -200px;
  right: -200px;
  opacity: 0.15;
  animation: glowFloat 20s ease-in-out infinite;
}

.glow-2 {
  width: 400px;
  height: 400px;
  background: var(--color-accent);
  bottom: -100px;
  left: -100px;
  opacity: 0.1;
  animation: glowFloat 15s ease-in-out infinite reverse;
}

@keyframes glowFloat {
  0%,
  100% {
    transform: translate(0, 0) scale(1);
  }
  25% {
    transform: translate(50px, -50px) scale(1.1);
  }
  50% {
    transform: translate(-30px, 30px) scale(0.9);
  }
  75% {
    transform: translate(-50px, -30px) scale(1.05);
  }
}

.hero-container {
  position: relative;
  max-width: 1280px;
  margin: 0 auto;
  padding: 80px 24px;
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 64px;
  align-items: center;
  z-index: 2;
}

.hero-content {
  animation: fadeInUp 0.8s ease-out;
}

.hero-badge {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  background: rgba(14, 165, 233, 0.1);
  border: 1px solid rgba(14, 165, 233, 0.2);
  border-radius: 100px;
  font-size: 14px;
  font-weight: 500;
  color: var(--color-primary-light);
  margin-bottom: 24px;
  animation: badgePulse 3s ease-in-out infinite;
}

@keyframes badgePulse {
  0%,
  100% {
    box-shadow: 0 0 0 0 rgba(14, 165, 233, 0.3);
  }
  50% {
    box-shadow: 0 0 20px 5px rgba(14, 165, 233, 0.1);
  }
}

.badge-dot {
  width: 6px;
  height: 6px;
  background: var(--color-success);
  border-radius: 50%;
  animation: pulse 2s infinite;
}

@keyframes pulse {
  0%,
  100% {
    opacity: 1;
    transform: scale(1);
  }
  50% {
    opacity: 0.5;
    transform: scale(1.2);
  }
}

.hero-title {
  font-family: var(--font-heading);
  font-size: 64px;
  font-weight: 800;
  line-height: 1.1;
  margin: 0 0 24px;
  letter-spacing: -0.02em;
  min-height: 1.2em;
  display: flex;
  flex-direction: column;
}

.title-line {
  display: block;
}

.typewriter-row {
  display: flex;
  align-items: baseline;
  min-height: 1.1em;
}

.typewriter {
  display: inline-block;
  min-width: 1ch;
}

.title-gradient {
  background: linear-gradient(
    135deg,
    var(--color-primary-light) 0%,
    var(--color-accent) 100%
  );
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.cursor {
  display: inline-block;
  animation: blink 1s infinite;
  color: var(--color-primary-light);
  -webkit-text-fill-color: var(--color-primary-light);
}

@keyframes blink {
  0%,
  50% {
    opacity: 1;
  }
  51%,
  100% {
    opacity: 0;
  }
}

.hero-subtitle {
  font-size: 18px;
  line-height: 1.7;
  color: var(--color-text-muted);
  margin: 0 0 40px;
  max-width: 500px;
}

.hero-actions {
  display: flex;
  gap: 16px;
  margin-bottom: 48px;
}

.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 14px 28px;
  border-radius: 12px;
  font-family: var(--font-body);
  font-size: 15px;
  font-weight: 600;
  text-decoration: none;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  cursor: pointer;
  position: relative;
  overflow: hidden;
}

.btn::before {
  content: "";
  position: absolute;
  inset: 0;
  background: linear-gradient(
    45deg,
    transparent,
    rgba(255, 255, 255, 0.1),
    transparent
  );
  transform: translateX(-100%);
  transition: transform 0.6s;
}

.btn:hover::before {
  transform: translateX(100%);
}

.btn-primary {
  background: linear-gradient(135deg, var(--color-primary) 0%, #0284c7 100%);
  color: white;
  box-shadow: 0 4px 20px rgba(14, 165, 233, 0.3);
}

.btn-primary:hover {
  transform: translateY(-2px);
  box-shadow: 0 8px 30px rgba(14, 165, 233, 0.4);
}

.btn-secondary {
  background: transparent;
  color: var(--color-text);
  border: 1px solid var(--color-border);
}

.btn-secondary:hover {
  background: rgba(255, 255, 255, 0.05);
  border-color: var(--color-text-muted);
}

.btn-large {
  padding: 18px 36px;
  font-size: 17px;
}

.hero-stats {
  display: flex;
  gap: 48px;
}

.stat-item {
  animation: fadeInUp 0.8s ease-out 0.3s both;
}

.stat-value {
  font-family: var(--font-heading);
  font-size: 32px;
  font-weight: 700;
  color: var(--color-text);
  line-height: 1;
  margin-bottom: 4px;
}

.stat-label {
  font-size: 14px;
  color: var(--color-text-muted);
}

/* Hero Visual */
.hero-visual {
  position: relative;
  height: 500px;
  animation: fadeIn 1s ease-out 0.2s both;
}

.server-rack {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  width: 200px;
  background: var(--color-bg-alt);
  border: 1px solid var(--color-border);
  border-radius: 16px;
  padding: 20px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.5);
  animation: rackFloat 6s ease-in-out infinite;
}

@keyframes rackFloat {
  0%,
  100% {
    transform: translate(-50%, -50%) translateY(0);
  }
  50% {
    transform: translate(-50%, -50%) translateY(-15px);
  }
}

.rack-unit {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px;
  background: rgba(255, 255, 255, 0.02);
  border-radius: 8px;
  margin-bottom: 12px;
}

.rack-unit:last-child {
  margin-bottom: 0;
}

.unit-lights {
  display: flex;
  gap: 8px;
}

.light {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  animation: blink 2s infinite;
}

.light.green {
  background: var(--color-success);
  box-shadow: 0 0 10px var(--color-success);
}
.light.blue {
  background: var(--color-primary);
  box-shadow: 0 0 10px var(--color-primary);
}
.light.orange {
  background: var(--color-accent);
  box-shadow: 0 0 10px var(--color-accent);
}

@keyframes blink {
  0%,
  100% {
    opacity: 1;
  }
  50% {
    opacity: 0.4;
  }
}

.unit-label {
  font-size: 10px;
  color: var(--color-text-muted);
  font-family: var(--font-heading);
  font-weight: 600;
}

.floating-card {
  position: absolute;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px 20px;
  background: var(--color-bg-alt);
  border: 1px solid var(--color-border);
  border-radius: 16px;
  box-shadow: 0 10px 40px rgba(0, 0, 0, 0.3);
  animation: float 6s ease-in-out infinite;
  transition: transform 0.1s ease-out;
  transform-style: preserve-3d;
}

.card-1 {
  top: 15%;
  right: 0;
  animation-delay: 0s;
}

.card-2 {
  top: 45%;
  right: -40px;
  animation-delay: 2s;
}

.card-3 {
  bottom: 15%;
  right: 20px;
  animation-delay: 4s;
}

@keyframes float {
  0%,
  100% {
    transform: translateY(0) rotate(0deg);
  }
  50% {
    transform: translateY(-20px) rotate(1deg);
  }
}

.card-icon {
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(14, 165, 233, 0.1);
  border-radius: 10px;
  color: var(--color-primary-light);
  transform: translateZ(20px);
}

.card-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text);
  line-height: 1.3;
  transform: translateZ(10px);
}

.card-desc {
  font-size: 12px;
  color: var(--color-text-muted);
  transform: translateZ(10px);
}

/* Features Section */
.features-section {
  padding: 120px 24px;
  background: var(--color-bg-alt);
  position: relative;
  z-index: 2;
}

.features-section::before {
  content: "";
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 1px;
  background: linear-gradient(
    to right,
    transparent,
    var(--color-border) 50%,
    transparent
  );
}

.features-container {
  max-width: 1280px;
  margin: 0 auto;
}

.section-header {
  text-align: center;
  margin-bottom: 64px;
}

.section-badge {
  display: inline-block;
  padding: 6px 16px;
  background: rgba(14, 165, 233, 0.1);
  border: 1px solid rgba(14, 165, 233, 0.2);
  border-radius: 100px;
  font-size: 13px;
  font-weight: 600;
  color: var(--color-primary-light);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  margin-bottom: 16px;
}

.section-title {
  font-family: var(--font-heading);
  font-size: 42px;
  font-weight: 700;
  color: var(--color-text);
  margin: 0 0 16px;
  letter-spacing: -0.02em;
}

.section-desc {
  font-size: 18px;
  color: var(--color-text-muted);
  max-width: 600px;
  margin: 0 auto;
}

.features-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 24px;
}

.feature-card {
  position: relative;
  padding: 32px;
  background: var(--color-bg);
  border: 1px solid var(--color-border);
  border-radius: 20px;
  overflow: hidden;
  transition: all 0.4s cubic-bezier(0.4, 0, 0.2, 1);
}

.scroll-animate {
  opacity: 0;
  transform: translateY(40px);
  transition: all 0.8s cubic-bezier(0.4, 0, 0.2, 1);
}

.scroll-animate-active {
  opacity: 1;
  transform: translateY(0);
}

.feature-card:hover {
  transform: translateY(-8px);
  border-color: rgba(14, 165, 233, 0.3);
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
}

.feature-bg {
  position: absolute;
  inset: 0;
  background: radial-gradient(
    circle 300px at var(--mouse-x, 50%) var(--mouse-y, 50%),
    rgba(14, 165, 233, 0.08),
    transparent 40%
  );
  opacity: 0;
  transition: opacity 0.3s;
  pointer-events: none;
}

.feature-card:hover .feature-bg {
  opacity: 1;
}

.feature-glow {
  position: absolute;
  width: 200px;
  height: 200px;
  background: radial-gradient(
    circle,
    rgba(14, 165, 233, 0.15) 0%,
    transparent 70%
  );
  border-radius: 50%;
  pointer-events: none;
  transform: translate(-50%, -50%);
  opacity: 0;
  transition: opacity 0.3s;
}

.feature-card:hover .feature-glow {
  opacity: 1;
}

.feature-icon {
  position: relative;
  width: 56px;
  height: 56px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(
    135deg,
    rgba(14, 165, 233, 0.15) 0%,
    rgba(14, 165, 233, 0.05) 100%
  );
  border-radius: 14px;
  color: var(--color-primary-light);
  margin-bottom: 20px;
  z-index: 1;
}

.feature-title {
  font-family: var(--font-heading);
  font-size: 20px;
  font-weight: 600;
  color: var(--color-text);
  margin: 0 0 12px;
  position: relative;
  z-index: 1;
}

.feature-desc {
  font-size: 15px;
  line-height: 1.6;
  color: var(--color-text-muted);
  margin: 0 0 20px;
  position: relative;
  z-index: 1;
}

.feature-link {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 14px;
  font-weight: 600;
  color: var(--color-primary-light);
  text-decoration: none;
  transition: gap 0.2s;
  position: relative;
  z-index: 1;
}

.feature-link:hover {
  gap: 10px;
}

/* Products Section */
.products-section {
  padding: 120px 24px;
  background: var(--color-bg);
  position: relative;
  z-index: 2;
}

.products-container {
  max-width: 1280px;
  margin: 0 auto;
}

.products-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 32px;
}

.product-card {
  position: relative;
  background: var(--color-bg-alt);
  border: 1px solid var(--color-border);
  border-radius: 24px;
  overflow: hidden;
  transition: all 0.4s cubic-bezier(0.4, 0, 0.2, 1);
}

.product-card:hover {
  transform: translateY(-4px) scale(1.02);
  border-color: rgba(14, 165, 233, 0.3);
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.4);
}

.product-image {
  position: relative;
  height: 180px;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
}

.product-bg {
  position: absolute;
  inset: 0;
  opacity: 0.8;
  transition: transform 0.6s;
}

.product-card:hover .product-bg {
  transform: scale(1.1);
}

.product-bg.bg-1 {
  background: linear-gradient(
    135deg,
    rgba(14, 165, 233, 0.2) 0%,
    rgba(14, 165, 233, 0.05) 100%
  );
}

.product-bg.bg-2 {
  background: linear-gradient(
    135deg,
    rgba(249, 115, 22, 0.2) 0%,
    rgba(249, 115, 22, 0.05) 100%
  );
}

.product-bg.bg-3 {
  background: linear-gradient(
    135deg,
    rgba(16, 185, 129, 0.2) 0%,
    rgba(16, 185, 129, 0.05) 100%
  );
}

.product-icon-wrapper {
  position: relative;
  z-index: 1;
  transition: transform 0.4s;
}

.product-card:hover .product-icon-wrapper {
  transform: scale(1.1) rotate(5deg);
}

.product-image svg {
  width: 80px;
  height: 80px;
  color: var(--color-text);
}

.product-content {
  padding: 28px;
}

.product-tag {
  display: inline-block;
  padding: 4px 12px;
  background: rgba(14, 165, 233, 0.15);
  border-radius: 6px;
  font-size: 12px;
  font-weight: 600;
  color: var(--color-primary-light);
  margin-bottom: 12px;
}

.product-title {
  font-family: var(--font-heading);
  font-size: 22px;
  font-weight: 600;
  color: var(--color-text);
  margin: 0 0 8px;
}

.product-desc {
  font-size: 14px;
  color: var(--color-text-muted);
  margin: 0 0 20px;
}

.product-price {
  display: flex;
  align-items: baseline;
  gap: 4px;
  margin-bottom: 20px;
}

.price-symbol {
  font-size: 18px;
  color: var(--color-text-muted);
}

.price-value {
  font-family: var(--font-heading);
  font-size: 36px;
  font-weight: 700;
  color: var(--color-text);
}

.price-unit {
  font-size: 14px;
  color: var(--color-text-muted);
}

.product-btn {
  display: block;
  width: 100%;
  padding: 14px;
  background: transparent;
  border: 1px solid var(--color-border);
  border-radius: 12px;
  font-size: 15px;
  font-weight: 600;
  color: var(--color-text);
  text-align: center;
  text-decoration: none;
  transition: all 0.3s;
  position: relative;
  overflow: hidden;
}

.product-btn::before {
  content: "";
  position: absolute;
  inset: 0;
  background: var(--color-primary);
  transform: translateX(-100%);
  transition: transform 0.3s;
}

.product-btn:hover::before {
  transform: translateX(0);
}

.product-btn span {
  position: relative;
  z-index: 1;
}

.product-btn:hover {
  border-color: var(--color-primary);
  color: white;
}

/* CTA Section */
.cta-section {
  padding: 120px 24px;
  background: var(--color-bg-alt);
  position: relative;
  overflow: hidden;
  z-index: 2;
}

.cta-container {
  position: relative;
  max-width: 1000px;
  margin: 0 auto;
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 64px;
  align-items: center;
  z-index: 1;
}

.cta-content {
  animation: fadeInUp 0.8s ease-out;
}

.cta-title {
  font-family: var(--font-heading);
  font-size: 40px;
  font-weight: 700;
  color: var(--color-text);
  margin: 0 0 16px;
  line-height: 1.2;
  letter-spacing: -0.02em;
}

.cta-desc {
  font-size: 17px;
  color: var(--color-text-muted);
  line-height: 1.7;
  margin: 0 0 32px;
}

.cta-actions {
  margin-bottom: 32px;
}

.cta-features {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.cta-feature {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 14px;
  color: var(--color-text-muted);
  opacity: 0;
  animation: fadeInRight 0.5s ease-out var(--delay) forwards;
}

@keyframes fadeInRight {
  from {
    opacity: 0;
    transform: translateX(-20px);
  }
  to {
    opacity: 1;
    transform: translateX(0);
  }
}

.cta-feature svg {
  width: 20px;
  height: 20px;
  color: var(--color-success);
  flex-shrink: 0;
}

.cta-visual {
  position: relative;
  height: 400px;
}

.cta-orb {
  position: absolute;
  border-radius: 50%;
  filter: blur(60px);
  animation: floatOrb 8s ease-in-out infinite;
}

.orb-1 {
  width: 200px;
  height: 200px;
  background: var(--color-primary);
  top: 20%;
  left: 20%;
  opacity: 0.3;
  animation-delay: 0s;
}

.orb-2 {
  width: 150px;
  height: 150px;
  background: var(--color-accent);
  top: 50%;
  right: 20%;
  opacity: 0.25;
  animation-delay: 2s;
}

.orb-3 {
  width: 100px;
  height: 100px;
  background: var(--color-success);
  bottom: 20%;
  left: 40%;
  opacity: 0.2;
  animation-delay: 4s;
}

@keyframes floatOrb {
  0%,
  100% {
    transform: translate(0, 0);
  }
  33% {
    transform: translate(30px, -30px);
  }
  66% {
    transform: translate(-20px, 20px);
  }
}

/* Animations */
@keyframes fadeIn {
  from {
    opacity: 0;
  }
  to {
    opacity: 1;
  }
}

@keyframes fadeInUp {
  from {
    opacity: 0;
    transform: translateY(30px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

/* Responsive */
@media (max-width: 1024px) {
  .features-grid,
  .products-grid {
    grid-template-columns: repeat(2, 1fr);
  }

  .hero-container {
    grid-template-columns: 1fr;
    text-align: center;
  }

  .hero-actions {
    justify-content: center;
  }

  .hero-stats {
    justify-content: center;
  }

  .hero-visual {
    display: none;
  }

  .cta-container {
    grid-template-columns: 1fr;
    text-align: center;
  }

  .cta-features {
    align-items: center;
  }
}

@media (max-width: 768px) {
  .hero-title {
    font-size: 42px;
  }

  .section-title {
    font-size: 32px;
  }

  .features-grid,
  .products-grid {
    grid-template-columns: 1fr;
  }

  .hero-stats {
    flex-direction: column;
    gap: 24px;
  }

  .hero-actions {
    flex-direction: column;
  }

  .cta-title {
    font-size: 32px;
  }

  .particle-canvas {
    display: none;
  }

  .mouse-glow {
    display: none;
  }
}
</style>
