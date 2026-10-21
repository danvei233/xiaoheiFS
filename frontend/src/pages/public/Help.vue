<template>
  <div class="help-page">
    <!-- Animated Background -->
    <canvas ref="bgCanvas" class="help-background"></canvas>

    <!-- Blocks-driven rendering (help page) -->
    <template v-for="block in resolvedBlocks" :key="block.type">
      <HelpHeroBlock
        v-if="block.type === 'help_hero'"
        :content="block.content"
        v-model:searchQuery="searchQuery"
      />
      <HelpActionsBlock
        v-else-if="block.type === 'help_actions'"
        :content="block.content"
        :is-authenticated="authStore.isAuthenticated"
      />
      <HelpFaqBlock
        v-else-if="block.type === 'help_faq'"
        :content="block.content"
        :search-query="searchQuery"
        @clear-search="searchQuery = ''"
      />
      <HelpContactBlock v-else-if="block.type === 'help_contact'" :content="block.content" />
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from "vue";
import { useAuthStore } from "@/stores/auth";
import { useSiteStore } from "@/stores/site";
import { getCmsBlocks } from "@/services/user";
import HelpHeroBlock from "@/components/cms/blocks/help/HelpHeroBlock.vue";
import HelpActionsBlock from "@/components/cms/blocks/help/HelpActionsBlock.vue";
import HelpFaqBlock from "@/components/cms/blocks/help/HelpFaqBlock.vue";
import HelpContactBlock from "@/components/cms/blocks/help/HelpContactBlock.vue";

type CMSBlock = {
  type: string;
  sort_order?: number;
  visible?: boolean;
  content_json?: string;
};

const authStore = useAuthStore();
const site = useSiteStore();

const searchQuery = ref("");
const bgCanvas = ref<HTMLCanvasElement>();

const blocks = ref<CMSBlock[]>([]);

const parseContentJson = (raw?: string) => {
  if (!raw) return {};
  try {
    return JSON.parse(raw);
  } catch {
    return {};
  }
};

const defaultBlocks = [
  { type: "help_hero", sort_order: 1, visible: true, content: {} },
  { type: "help_actions", sort_order: 2, visible: true, content: {} },
  { type: "help_faq", sort_order: 3, visible: true, content: {} },
  { type: "help_contact", sort_order: 4, visible: true, content: {} },
];

const resolvedBlocks = computed(() => {
  const items = (blocks.value || [])
    .filter((b) => b && b.visible !== false)
    .slice()
    .sort((a, b) => (a.sort_order || 0) - (b.sort_order || 0));

  const present = new Set(items.map((b) => b.type));
  const merged = items.concat(defaultBlocks.filter((d) => !present.has(d.type)));

  return merged.map((b) => ({
    type: b.type,
    sort_order: b.sort_order,
    visible: b.visible,
    content: (b as any).content ?? parseContentJson((b as any).content_json),
  }));
});

const fetchBlocks = async () => {
  try {
    const lang = site.currentLang || "zh-CN";
    const res = await getCmsBlocks({ page: "help", lang });
    blocks.value = res.data?.items || [];
  } catch {
    blocks.value = [];
  }
};

// Canvas animation
let animationId: number
const particles: Array<{
  x: number
  y: number
  vx: number
  vy: number
  radius: number
  opacity: number
}> = []

const initCanvas = () => {
  const canvas = bgCanvas.value
  if (!canvas) return

  const ctx = canvas.getContext('2d')
  if (!ctx) return

  const resize = () => {
    canvas.width = window.innerWidth
    canvas.height = window.innerHeight * 0.6
  }

  resize()
  window.addEventListener('resize', resize)

  // Create particles
  for (let i = 0; i < 50; i++) {
    particles.push({
      x: Math.random() * canvas.width,
      y: Math.random() * canvas.height,
      vx: (Math.random() - 0.5) * 0.5,
      vy: (Math.random() - 0.5) * 0.5,
      radius: Math.random() * 2,
      opacity: Math.random() * 0.5
    })
  }

  const animate = () => {
    ctx.clearRect(0, 0, canvas.width, canvas.height)

    // Update and draw particles
    particles.forEach((particle, i) => {
      particle.x += particle.vx
      particle.y += particle.vy

      if (particle.x < 0 || particle.x > canvas.width) particle.vx *= -1
      if (particle.y < 0 || particle.y > canvas.height) particle.vy *= -1

      ctx.beginPath()
      ctx.arc(particle.x, particle.y, particle.radius, 0, Math.PI * 2)
      ctx.fillStyle = `rgba(14, 165, 233, ${particle.opacity})`
      ctx.fill()

      // Draw connections
      particles.slice(i + 1).forEach(other => {
        const dx = particle.x - other.x
        const dy = particle.y - other.y
        const distance = Math.sqrt(dx * dx + dy * dy)

        if (distance < 120) {
          ctx.beginPath()
          ctx.moveTo(particle.x, particle.y)
          ctx.lineTo(other.x, other.y)
          ctx.strokeStyle = `rgba(14, 165, 233, ${0.1 * (1 - distance / 120)})`
          ctx.stroke()
        }
      })
    })

    animationId = requestAnimationFrame(animate)
  }

  animate()
}

onMounted(() => {
  initCanvas()
  fetchBlocks()
})

onUnmounted(() => {
  if (animationId) {
    cancelAnimationFrame(animationId)
  }
})
</script>

<style scoped>
.help-page {
  min-height: 100vh;
  background: var(--color-bg);
  position: relative;
  overflow: hidden;
}

.help-background {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 60vh;
  pointer-events: none;
  z-index: 0;
}
</style>
