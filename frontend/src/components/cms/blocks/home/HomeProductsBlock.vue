<template>
  <section class="products-section">
    <div class="products-container">
      <div class="section-header scroll-animate">
        <InlineEdit
          v-if="isVisualMode"
          field-path="products.badge"
          v-model="localBadge"
          edit-type="text"
          label="模块徽标"
        />
        <div v-else class="section-badge">
          {{ content.badge || $t("home.products.badge") || "产品系列" }}
        </div>

        <InlineEdit
          v-if="isVisualMode"
          field-path="products.title"
          v-model="localTitle"
          edit-type="text"
          label="模块标题"
        />
        <h2 v-else class="section-title">
          {{ content.title || $t("home.products.title") || "满足各种规模需求" }}
        </h2>
      </div>

      <div class="products-grid">
        <template v-if="isVisualMode">
          <div
            v-for="(product, index) in editableProducts"
            :key="`product-${index}`"
            class="product-card scroll-animate"
            :style="{ '--delay': `${index * 0.15}s` }"
          >
            <div class="product-image">
              <div class="product-bg" :class="`bg-${index + 1}`"></div>
              <div class="product-icon-wrapper">
                <component :is="product.icon" v-if="product.icon" />
                <svg
                  v-else
                  width="48"
                  height="48"
                  viewBox="0 0 24 24"
                  fill="none"
                >
                  <path
                    d="M20 7L12 3L4 7M20 7L12 11M20 7V17L12 21M4 7L12 11M4 7V17L12 21M12 11V21"
                    stroke="currentColor"
                    stroke-width="2"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                  />
                </svg>
              </div>
            </div>
            <div class="product-content">
              <InlineEdit
                :field-path="`products.items.${index}.tag`"
                v-model="product.tag"
                edit-type="text"
                label="产品标签"
                :is-array-item="true"
                :can-add="index === editableProducts.length - 1"
                :can-remove="editableProducts.length > 1"
                @add-item="addProductItem"
                @remove-item="() => removeProductItem(index)"
              />
              <InlineEdit
                :field-path="`products.items.${index}.title`"
                v-model="product.title"
                edit-type="text"
                label="产品标题"
              />
              <InlineEdit
                :field-path="`products.items.${index}.description`"
                v-model="product.description"
                edit-type="textarea"
                label="产品描述"
                :rows="2"
              />
              <div class="product-price">
                <span class="price-symbol">￥</span>
                <InlineEdit
                  :field-path="`products.items.${index}.price`"
                  v-model="product.price"
                  edit-type="text"
                  label="价格"
                />
                <span class="price-unit">/月起</span>
              </div>
            </div>
          </div>

          <div v-if="editableProducts.length === 0" class="empty-products">
            <a-button type="dashed" @click="addProductItem">+ 添加产品</a-button>
          </div>
        </template>

        <template v-else>
          <div
            class="product-card scroll-animate"
            v-for="(product, index) in products"
            :key="index"
            :style="{ '--delay': `${index * 0.15}s` }"
          >
            <div class="product-image">
              <div class="product-bg" :class="`bg-${index + 1}`"></div>
              <div class="product-icon-wrapper">
                <component :is="product.icon" v-if="product.icon" />
              </div>
            </div>
            <div class="product-content">
              <div class="product-tag">{{ product.tag }}</div>
              <h3 class="product-title">{{ product.title }}</h3>
              <p class="product-desc">{{ product.description }}</p>
              <div class="product-price">
                <span class="price-symbol">￥</span>
                <span class="price-value">{{ product.price }}</span>
                <span class="price-unit">/月起</span>
              </div>
              <router-link to="/products" class="product-btn">
                {{ $t("home.products.view") || "查看详情" }}
              </router-link>
            </div>
          </div>
        </template>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { inject, computed } from "vue";
import { Button as AButton } from "ant-design-vue";
import InlineEdit from "@/components/InlineEdit.vue";

interface ProductItem {
  icon?: any;
  tag?: string;
  title?: string;
  description?: string;
  price?: string;
}

interface ProductsContent {
  badge?: string;
  title?: string;
}

const props = defineProps<{
  content: ProductsContent;
  products: ProductItem[];
}>();

const cmsEditContext: any = inject("cmsEditContext", null);
const isVisualMode = cmsEditContext?.editMode === "visual";

const localBadge = computed({
  get: () => props.content.badge || "",
  set: (val: string) => cmsEditContext?.updateField("products.badge", val),
});

const localTitle = computed({
  get: () => props.content.title || "",
  set: (val: string) => cmsEditContext?.updateField("products.title", val),
});

const editableProducts = computed({
  get: () => props.products || [],
  set: (_val) => {},
});

const addProductItem = () => {
  cmsEditContext?.addArrayItem("products.items", {
    tag: "新产品",
    title: "产品名称",
    description: "产品描述",
    price: "99",
  });
};

const removeProductItem = (index: number) => {
  cmsEditContext?.removeArrayItem("products.items", index);
};
</script>

<style scoped>
.empty-products {
  grid-column: 1 / -1;
  display: flex;
  justify-content: center;
  align-items: center;
  padding: 40px;
}

.section-header :deep(.editable-region) {
  display: block;
  margin-bottom: 8px;
}

.product-content :deep(.editable-region) {
  display: block;
  margin-bottom: 8px;
}

.product-price :deep(.editable-region) {
  display: inline-block;
}
</style>

