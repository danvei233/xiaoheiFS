<template>
  <div class="cart-page">
    <div class="cart-page__inner">
      <header class="page-heading">
        <div>
          <h1>购物车</h1>
          <p>{{ cart.itemCount }} 件商品</p>
        </div>
        <ElButton type="primary" :icon="ShoppingBag" @click="router.push({ name: 'PublicBuy' })">
          继续选购
        </ElButton>
      </header>

      <div class="cart-layout">
        <ElCard class="cart-table-card art-table-card">
          <ArtTableHeader :loading="loading || cart.merging" @refresh="refreshData">
            <template #left>
              <div class="table-heading">
                <span>购物车明细</span>
                <ElTag v-if="cart.isGuest" type="info" size="small">游客购物车</ElTag>
              </div>
            </template>
          </ArtTableHeader>

          <div v-if="!loading && !cart.merging && data.length === 0" class="cart-empty">
            <ElEmpty description="购物车是空的">
              <p>添加商品到购物车开始下单</p>
              <ElButton type="primary" @click="router.push({ name: 'PublicBuy' })">
                立即选购
              </ElButton>
            </ElEmpty>
          </div>

          <ArtTable
            v-else
            row-key="id"
            :loading="loading || cart.merging"
            :data="data"
            :columns="columns"
          >
            <template #product="{ row }">
              <div class="product-cell">
                <ArtSvgIcon icon="ri:server-line" />
                <div>
                  <strong>{{ row.packageName }}</strong>
                  <span>{{ row.systemName }}</span>
                </div>
              </div>
            </template>

            <template #specification="{ row }">
              <span class="specification">{{ row.specification }}</span>
            </template>

            <template #quantity="{ row }">
              <ElInputNumber
                :model-value="row.qty"
                :min="1"
                :max="99"
                size="small"
                @change="(value) => updateQuantity(row, Number(value || 1))"
              />
            </template>

            <template #amount="{ row }">
              <strong class="amount">{{ formatMoney(row.amount) }}</strong>
            </template>

            <template #operation="{ row }">
              <ElTooltip content="移除商品" placement="top">
                <ArtButtonTable type="delete" @click="removeItem(row)" />
              </ElTooltip>
            </template>
          </ArtTable>
        </ElCard>

        <aside class="summary-panel">
          <ElCard class="summary-card art-card-xs" shadow="never">
            <template #header><span class="summary-card__title">订单摘要</span></template>

            <div class="summary-row">
              <span>商品数量</span>
              <strong>{{ cart.itemCount }} 件</strong>
            </div>
            <div class="summary-row">
              <span>商品合计</span>
              <strong>{{ formatMoney(totalAmount) }}</strong>
            </div>

            <label class="coupon-field">
              <span>优惠码</span>
              <div class="coupon-row">
                <ElInput v-model="couponCode" placeholder="可选" clearable />
                <ElButton :loading="couponLoading" @click="applyCoupon">验证</ElButton>
              </div>
            </label>

            <div v-if="couponPreview" class="discount-row">
              <span>优惠</span>
              <strong>-{{ formatMoney(couponPreview.discount || 0) }}</strong>
            </div>

            <div class="payable-row">
              <span>应付金额</span>
              <strong>{{ formatMoney(payableTotal) }}</strong>
            </div>

            <ElButton
              type="primary"
              size="large"
              :loading="submitting || cart.merging"
              :disabled="cart.itemCount === 0"
              class="checkout-button"
              @click="submitOrder"
            >
              去结算
            </ElButton>
          </ElCard>
        </aside>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { ShoppingBag } from '@element-plus/icons-vue'
  import { useTable } from '@/hooks/core/useTable'
  import { useAuthStore } from '@/stores/auth'
  import { useCartStore, type CartStoreItem } from '@/stores/cart'
  import { useCatalogStore } from '@/stores/catalog'
  import { createOrderFromCartWithCoupon, previewCoupon } from '@/services/user'
  import { IdempotentRequest } from '@/utils/idempotentRequest'
  import type { CouponPreviewResponse, Package, SystemImage } from '@/services/types'

  defineOptions({ name: 'PublicCart' })

  interface CartRow extends CartStoreItem {
    packageName: string
    systemName: string
    specification: string
  }

  const router = useRouter()
  const auth = useAuthStore()
  const cart = useCartStore()
  const catalog = useCatalogStore()
  const submitting = ref(false)
  const couponCode = ref('')
  const couponLoading = ref(false)
  const couponPreview = ref<CouponPreviewResponse | null>(null)
  const orderRequest = new IdempotentRequest<{ coupon_code?: string } | undefined>('order')
  const findPackage = (packageId: number) =>
    (catalog.packages as Package[]).find((item) => String(item.id) === String(packageId))
  const findSystem = (systemId?: number) =>
    (catalog.systemImages as SystemImage[]).find((item) => String(item.id) === String(systemId))

  const formatSpecification = (item: CartStoreItem) => {
    const pkg = findPackage(item.package_id)
    const parts = [
      `CPU ${Number(pkg?.cores || 0) + Number(item.spec.add_cores || 0)} 核`,
      `内存 ${Number(pkg?.memory_gb || 0) + Number(item.spec.add_mem_gb || 0)} GB`,
      `磁盘 ${Number(pkg?.disk_gb || 0) + Number(item.spec.add_disk_gb || 0)} GB`,
      `带宽 ${Number(pkg?.bandwidth_mbps || 0) + Number(item.spec.add_bw_mbps || 0)} Mbps`
    ]
    if (item.spec.duration_months) parts.push(`时长 ${item.spec.duration_months} 个月`)
    return parts.join(' / ')
  }

  const enrichItem = (item: CartStoreItem): CartRow => ({
    ...item,
    packageName: findPackage(item.package_id)?.name || `套餐 #${item.package_id}`,
    systemName:
      findSystem(item.system_id)?.name || (item.system_id ? `系统 #${item.system_id}` : '默认系统'),
    specification: formatSpecification(item)
  })

  const fetchCartRows = async () => {
    await cart.fetchCart()
    const rows = cart.items.map(enrichItem)
    return {
      records: rows,
      current: 1,
      size: Math.max(1, rows.length),
      total: rows.length
    }
  }

  const { columns, data, loading, refreshData } = useTable<typeof fetchCartRows>({
    core: {
      apiFn: fetchCartRows,
      apiParams: { current: 1, size: 100 },
      columnsFactory: () => [
        { prop: 'product', label: '商品', minWidth: 220, useSlot: true },
        { prop: 'specification', label: '配置', minWidth: 320, useSlot: true },
        { prop: 'quantity', label: '数量', width: 140, align: 'center', useSlot: true },
        { prop: 'amount', label: '金额', width: 140, align: 'right', useSlot: true },
        {
          prop: 'operation',
          label: '操作',
          width: 80,
          fixed: 'right',
          align: 'right',
          useSlot: true
        }
      ]
    }
  })

  const totalAmount = computed(() => cart.items.reduce((total, item) => total + item.amount, 0))
  const payableTotal = computed(() => {
    const total = Number(couponPreview.value?.final_total ?? totalAmount.value)
    return Math.max(0, total)
  })

  const formatMoney = (amount: number) =>
    new Intl.NumberFormat('zh-CN', {
      style: 'currency',
      currency: 'CNY',
      minimumFractionDigits: 2
    }).format(Number(amount || 0))

  const updateQuantity = async (row: CartRow, qty: number) => {
    await cart.updateItem(row.id, { spec: row.spec, qty })
    couponPreview.value = null
    await refreshData()
  }

  const removeItem = async (row: CartRow) => {
    try {
      await ElMessageBox.confirm('确定要移除这个商品吗？', '提示', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      })
      await cart.removeItem(row.id)
      couponPreview.value = null
      await refreshData()
    } catch (error) {
      if (error !== 'cancel' && error !== 'close') ElMessage.error('移除商品失败')
    }
  }

  const applyCoupon = async () => {
    const code = couponCode.value.trim()
    if (!code) {
      ElMessage.warning('请先输入优惠码')
      return
    }
    if (!auth.token) {
      ElMessage.info('登录后可验证优惠码')
      return
    }
    couponLoading.value = true
    try {
      const response = await previewCoupon({ coupon_code: code })
      couponPreview.value = response.data ?? null
      ElMessage.success('优惠码已应用')
    } catch {
      couponPreview.value = null
    } finally {
      couponLoading.value = false
    }
  }

  const submitOrder = async () => {
    if (!auth.token) {
      await router.push({ name: 'Login', query: { redirect: '/cart' } })
      return
    }

    submitting.value = true
    let request: ReturnType<typeof orderRequest.prepare> | undefined
    try {
      const mergeResult = await cart.mergeGuestCart()
      if (mergeResult.failed > 0) {
        ElMessage.warning('仍有商品未同步到服务端，请处理后再结算')
        return
      }
      const coupon = couponCode.value.trim()
      request = orderRequest.prepare(() => (coupon ? { coupon_code: coupon } : undefined))
      const response = await createOrderFromCartWithCoupon(request.payload, request.key)
      orderRequest.resolve()
      const payload = response.data as Record<string, any>
      const orderId = payload.order?.id ?? payload.order?.ID ?? payload.id ?? payload.ID
      ElMessage.success('订单已创建')
      await cart.fetchCart()
      if (orderId) await router.push({ name: 'ConsoleOrderDetail', params: { id: orderId } })
    } catch (error) {
      if (request) orderRequest.reject(error)
    } finally {
      submitting.value = false
    }
  }

  watch(couponCode, () => {
    couponPreview.value = null
  })

  onMounted(async () => {
    try {
      if (!catalog.packages.length) await catalog.fetchCatalog()
    } catch {
      ElMessage.warning('商品目录加载失败，部分商品信息可能不完整')
    }
  })
</script>

<style lang="scss" scoped>
  .cart-page {
    min-height: calc(100vh - 64px);
    padding: 28px 20px 48px;
    background: var(--default-bg-color);

    &__inner {
      max-width: 1240px;
      margin: 0 auto;
    }
  }

  .page-heading {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    margin-bottom: 18px;

    h1 {
      margin: 0;
      color: var(--art-gray-900);
      font-size: 28px;
      letter-spacing: 0;
    }

    p {
      margin: 7px 0 0;
      color: var(--art-gray-600);
    }
  }

  .cart-layout {
    display: grid;
    grid-template-columns: minmax(0, 1fr) minmax(280px, 330px);
    gap: 18px;
    align-items: start;
    margin-top: 12px;
  }

  .cart-table-card {
    min-height: 520px;
    margin-top: 0;
  }

  .cart-empty {
    min-height: 430px;

    p {
      margin: -8px 0 16px;
      color: var(--art-gray-600);
    }
  }

  .table-heading {
    display: flex;
    gap: 8px;
    align-items: center;
    color: var(--art-gray-900);
    font-weight: 600;
  }

  .product-cell {
    display: flex;
    gap: 10px;
    align-items: center;

    > .art-svg-icon {
      flex: 0 0 auto;
      color: var(--theme-color);
      font-size: 22px;
    }

    div {
      display: grid;
      min-width: 0;
      gap: 2px;
    }

    strong {
      overflow: hidden;
      color: var(--art-gray-900);
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    span {
      color: var(--art-gray-600);
      font-size: 12px;
    }
  }

  .specification {
    color: var(--art-gray-700);
    line-height: 1.65;
  }

  .amount {
    color: var(--theme-color);
  }

  .summary-panel {
    position: sticky;
    top: 82px;
  }

  .summary-card__title {
    color: var(--art-gray-900);
    font-weight: 600;
  }

  .summary-row,
  .discount-row,
  .payable-row {
    display: flex;
    justify-content: space-between;
    gap: 12px;
    padding: 10px 0;
    color: var(--art-gray-700);
  }

  .discount-row {
    color: var(--el-color-success);
  }

  .coupon-field {
    display: grid;
    gap: 8px;
    margin: 10px 0;
    color: var(--art-gray-700);
    font-size: 13px;
  }

  .coupon-row {
    display: grid;
    grid-template-columns: minmax(0, 1fr) auto;
    gap: 8px;
  }

  .payable-row {
    align-items: baseline;
    margin-top: 6px;
    padding-top: 16px;
    border-top: 1px solid var(--default-border);

    strong {
      color: var(--theme-color);
      font-size: 21px;
    }
  }

  .checkout-button {
    width: 100%;
    margin-top: 14px;
  }

  @media (width <= 960px) {
    .cart-layout {
      grid-template-columns: 1fr;
    }

    .summary-panel {
      position: static;
    }
  }

  @media (width <= 640px) {
    .cart-page {
      padding: 20px 12px 36px;
    }

    .page-heading {
      align-items: stretch;
      flex-direction: column;
      gap: 14px;

      h1 {
        font-size: 24px;
      }
    }
  }
</style>
