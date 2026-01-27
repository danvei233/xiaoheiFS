<template>
  <div class="cms-blocks-page">
    <div class="page-header">
      <h1 class="page-title">CMS块管理</h1>
      <a-button type="primary" @click="openCreateModal">
        <template #icon><PlusOutlined /></template> 新建块
      </a-button>
    </div>
    <a-card :bordered="false">
      <ProTable
        :columns="columns"
        :data-source="blocks"
        :loading="loading"
        :pagination="pagination"
        @change="handleTableChange"
        row-key="id"
      >
        <template #toolbar>
          <a-select
            v-model:value="filters.page"
            placeholder="页面筛选"
            style="width: 150px"
            allow-clear
            @change="handleFilterChange"
          >
            <a-select-option value="">全部页面</a-select-option>
            <a-select-option value="home">首页</a-select-option>
            <a-select-option value="products">产品页</a-select-option>
            <a-select-option value="docs">文档页</a-select-option>
            <a-select-option value="announcements">公告页</a-select-option>
            <a-select-option value="activities">活动页</a-select-option>
            <a-select-option value="tutorials">教程页</a-select-option>
            <a-select-option value="help">帮助页</a-select-option>
            <a-select-option value="footer">Footer</a-select-option>
          </a-select>
          <a-select
            v-model:value="filters.type"
            placeholder="类型筛选"
            style="width: 150px"
            allow-clear
            @change="handleFilterChange"
          >
            <a-select-option value="">全部类型</a-select-option>
            <a-select-option value="hero">Hero</a-select-option>
            <a-select-option value="features">Features</a-select-option>
            <a-select-option value="cta">CTA</a-select-option>
            <a-select-option value="products">Products</a-select-option>
            <a-select-option value="calculator">Calculator</a-select-option>
            <a-select-option value="pricing">Pricing</a-select-option>
            <a-select-option value="comparison">Comparison</a-select-option>
            <a-select-option value="footer">Footer</a-select-option>
            <a-select-option value="posts">Posts</a-select-option>
            <a-select-option value="resources">Resources</a-select-option>
            <a-select-option value="help_hero">Help: Hero</a-select-option>
            <a-select-option value="help_actions">Help: Actions</a-select-option>
            <a-select-option value="help_faq">Help: FAQ</a-select-option>
            <a-select-option value="help_contact">Help: Contact</a-select-option>
            <a-select-option value="custom_html">自定义HTML</a-select-option>
          </a-select></template
        >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'visible'">
            <a-switch
              :checked="record.visible"
              @change="(checked: boolean) => handleToggle(record, checked)"
            />
          </template>
          <template v-else-if="column.key === 'actions'">
            <a-space>
              <a-button type="link" size="small" @click="openEditModal(record)">
                编辑
              </a-button>
              <a-popconfirm
                title="确定要删除此块吗？"
                @confirm="handleDelete(record)"
              >
                <a-button type="link" danger size="small"> 删除 </a-button>
              </a-popconfirm>
            </a-space>
          </template>
        </template>
      </ProTable>
    </a-card>
    <!-- Create/Edit Modal -->
    <a-modal
      v-model:open="modalVisible"
      :title="isEditing ? '编辑块' : '新建块'"
      @ok="handleSubmit"
      :confirm-loading="submitting"
      width="1200px"
    >
      <a-row :gutter="16" class="editor-layout">
        <a-col :span="9" class="editor-form">
          <a-form :model="form" layout="vertical">
            <a-row :gutter="16">
              <a-col :span="12">
                <a-form-item label="页面">
                  <a-select v-model:value="form.page">
                    <a-select-option value="home">首页</a-select-option>
                    <a-select-option value="products">产品页</a-select-option>
                    <a-select-option value="docs">文档页</a-select-option>
                    <a-select-option value="announcements">公告页</a-select-option>
                    <a-select-option value="activities">活动页</a-select-option>
                    <a-select-option value="tutorials">教程页</a-select-option>
                    <a-select-option value="help">帮助页</a-select-option>
                    <a-select-option value="footer">Footer</a-select-option>
                  </a-select>
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item label="类型">
                  <a-select v-model:value="form.type">
                    <template v-if="form.page === 'help'">
                      <a-select-option value="help_hero">Help: Hero</a-select-option>
                      <a-select-option value="help_actions">Help: Actions</a-select-option>
                      <a-select-option value="help_faq">Help: FAQ</a-select-option>
                      <a-select-option value="help_contact">Help: Contact</a-select-option>
                      <a-select-option value="custom_html">自定义HTML</a-select-option>
                    </template>
                    <template v-else>
                      <a-select-option value="hero">Hero Banner</a-select-option>
                      <a-select-option value="features"
                        >Feature Cards</a-select-option
                      >
                      <a-select-option value="cta">CTA Banner</a-select-option>
                      <a-select-option value="products">Products</a-select-option>
                      <a-select-option value="calculator"
                        >Calculator</a-select-option
                      >
                      <a-select-option value="pricing">Pricing</a-select-option>
                      <a-select-option value="comparison"
                        >Comparison</a-select-option
                      >
                      <a-select-option value="footer">Footer</a-select-option>
                      <a-select-option value="posts">Posts</a-select-option>
                      <a-select-option value="resources">Resources</a-select-option>
                      <a-select-option value="custom_html">自定义HTML</a-select-option>
                    </template>
                  </a-select>
                </a-form-item>
              </a-col>
            </a-row>
            <a-form-item label="标题">
              <a-input v-model:value="form.title" />
            </a-form-item>
            <a-form-item label="副标题">
              <a-input v-model:value="form.subtitle" />
            </a-form-item>
            <a-form-item
              label="内容JSON"
              v-if="!structuredTypes.includes(form.type)"
            >
              <a-textarea
                v-model:value="form.content_json"
                :rows="6"
                placeholder='{"items": [...]}'
              />
            </a-form-item>
            <template v-if="form.page === 'home' && form.type === 'hero'">
              <a-form-item label="徽标文案">
                <a-input v-model:value="formContent.hero.badge" />
              </a-form-item>
              <a-form-item label="标题第一行">
                <a-input v-model:value="formContent.hero.title1" />
              </a-form-item>
              <a-form-item label="副标题">
                <a-textarea
                  v-model:value="formContent.hero.subtitle"
                  :rows="2"
                />
              </a-form-item>
              <a-row :gutter="16">
                <a-col :span="12">
                  <a-form-item label="主按钮文案">
                    <a-input
                      v-model:value="formContent.hero.primary_button_text"
                    />
                  </a-form-item>
                </a-col>
                <a-col :span="12">
                  <a-form-item label="主按钮链接">
                    <a-input
                      v-model:value="formContent.hero.primary_button_link"
                    />
                  </a-form-item>
                </a-col>
              </a-row>
              <a-row :gutter="16">
                <a-col :span="12">
                  <a-form-item label="次按钮文案">
                    <a-input
                      v-model:value="formContent.hero.secondary_button_text"
                    />
                  </a-form-item>
                </a-col>
                <a-col :span="12">
                  <a-form-item label="次按钮链接">
                    <a-input
                      v-model:value="formContent.hero.secondary_button_link"
                    />
                  </a-form-item>
                </a-col>
              </a-row>
              <a-form-item label="打字机词组">
                <div class="form-list">
                  <div
                    class="form-list-item"
                    v-for="(word, index) in formContent.hero.typewriter_words"
                    :key="index"
                  >
                    <a-input
                      v-model:value="formContent.hero.typewriter_words[index]"
                    />
                    <a-button
                      type="link"
                      danger
                      @click="
                        removeListItem(formContent.hero.typewriter_words, index)
                      "
                      >删除</a-button
                    >
                  </div>
                  <a-button
                    type="dashed"
                    @click="addListItem(formContent.hero.typewriter_words, '')"
                    >新增词组</a-button
                  >
                </div>
              </a-form-item>
              <a-form-item label="浮动卡片">
                <div class="form-list">
                  <div
                    class="form-list-item"
                    v-for="(card, index) in formContent.hero.cards"
                    :key="index"
                  >
                    <a-input v-model:value="card.title" placeholder="标题" />
                    <a-input v-model:value="card.desc" placeholder="描述" />
                    <a-button
                      type="link"
                      danger
                      @click="removeListItem(formContent.hero.cards, index)"
                      >删除</a-button
                    >
                  </div>
                  <a-button
                    type="dashed"
                    @click="
                      addListItem(formContent.hero.cards, {
                        title: '',
                        desc: '',
                      })
                    "
                    >新增卡片</a-button
                  >
                </div>
              </a-form-item>
              <a-form-item label="统计数据（Hero 内）">
                <div class="form-list">
                  <div
                    class="form-list-item"
                    v-for="(stat, index) in formContent.hero.stats"
                    :key="index"
                  >
                    <a-input-number
                      v-model:value="stat.value"
                      :min="0"
                      style="width: 120px"
                    />
                    <a-input
                      v-model:value="stat.suffix"
                      placeholder="后缀"
                      style="width: 80px"
                    />
                    <a-input v-model:value="stat.label" placeholder="标签" />
                    <a-button
                      type="link"
                      danger
                      @click="removeListItem(formContent.hero.stats, index)"
                      >删除</a-button
                    >
                  </div>
                  <a-button
                    type="dashed"
                    @click="
                      addListItem(formContent.hero.stats, {
                        value: 0,
                        suffix: '',
                        label: '',
                      })
                    "
                    >新增统计</a-button
                  >
                </div>
              </a-form-item>
            </template>
            <template v-if="form.type === 'stats'">
              <a-form-item label="统计数据">
                <div class="form-list">
                  <div
                    class="form-list-item"
                    v-for="(stat, index) in formContent.stats.items"
                    :key="index"
                  >
                    <a-input-number
                      v-model:value="stat.value"
                      :min="0"
                      style="width: 120px"
                    />
                    <a-input
                      v-model:value="stat.suffix"
                      placeholder="后缀"
                      style="width: 80px"
                    />
                    <a-input v-model:value="stat.label" placeholder="标签" />
                    <a-button
                      type="link"
                      danger
                      @click="removeListItem(formContent.stats.items, index)"
                      >删除</a-button
                    >
                  </div>
                  <a-button
                    type="dashed"
                    @click="
                      addListItem(formContent.stats.items, {
                        value: 0,
                        suffix: '',
                        label: '',
                      })
                    "
                    >新增统计</a-button
                  >
                </div>
              </a-form-item>
            </template>
            <template v-if="form.page === 'home' && form.type === 'features'">
              <a-form-item label="模块徽标">
                <a-input v-model:value="formContent.features.badge" />
              </a-form-item>
              <a-form-item label="模块标题">
                <a-input v-model:value="formContent.features.title" />
              </a-form-item>
              <a-form-item label="模块描述">
                <a-textarea
                  v-model:value="formContent.features.desc"
                  :rows="2"
                />
              </a-form-item>
              <a-form-item label="特性卡片">
                <div class="form-list">
                  <div
                    class="form-list-item"
                    v-for="(item, index) in formContent.features.items"
                    :key="index"
                  >
                    <a-input v-model:value="item.title" placeholder="标题" />
                    <a-input
                      v-model:value="item.description"
                      placeholder="描述"
                    />
                    <a-button
                      type="link"
                      danger
                      @click="removeListItem(formContent.features.items, index)"
                      >删除</a-button
                    >
                  </div>
                  <a-button
                    type="dashed"
                    @click="
                      addListItem(formContent.features.items, {
                        title: '',
                        description: '',
                      })
                    "
                    >新增卡片</a-button
                  >
                </div>
              </a-form-item>
            </template>
            <template v-if="form.page === 'home' && form.type === 'products'">
              <a-form-item label="模块徽标">
                <a-input v-model:value="formContent.products.badge" />
              </a-form-item>
              <a-form-item label="模块标题">
                <a-input v-model:value="formContent.products.title" />
              </a-form-item>
              <a-form-item label="产品卡片">
                <div class="form-list">
                  <div
                    class="form-list-item"
                    v-for="(item, index) in formContent.products.items"
                    :key="index"
                  >
                    <a-input
                      v-model:value="item.tag"
                      placeholder="标签"
                      style="width: 100px"
                    />
                    <a-input v-model:value="item.title" placeholder="标题" />
                    <a-input
                      v-model:value="item.description"
                      placeholder="描述"
                    />
                    <a-input
                      v-model:value="item.price"
                      placeholder="价格"
                      style="width: 120px"
                    />
                    <a-button
                      type="link"
                      danger
                      @click="removeListItem(formContent.products.items, index)"
                      >删除</a-button
                    >
                  </div>
                  <a-button
                    type="dashed"
                    @click="
                      addListItem(formContent.products.items, {
                        tag: '',
                        title: '',
                        description: '',
                        price: '',
                      })
                    "
                    >新增产品</a-button
                  >
                </div>
              </a-form-item>
            </template>
            <template v-if="form.page === 'home' && form.type === 'cta'">
              <a-form-item label="标题">
                <a-input v-model:value="formContent.cta.title" />
              </a-form-item>
              <a-form-item label="描述">
                <a-textarea v-model:value="formContent.cta.desc" :rows="2" />
              </a-form-item>
              <a-row :gutter="16">
                <a-col :span="12">
                  <a-form-item label="按钮文案">
                    <a-input v-model:value="formContent.cta.button_text" />
                  </a-form-item>
                </a-col>
                <a-col :span="12">
                  <a-form-item label="按钮链接">
                    <a-input v-model:value="formContent.cta.button_link" />
                  </a-form-item>
                </a-col>
              </a-row>
              <a-form-item label="亮点列表">
                <div class="form-list">
                  <div
                    class="form-list-item"
                    v-for="(item, index) in formContent.cta.features"
                    :key="index"
                  >
                    <a-input v-model:value="formContent.cta.features[index]" />
                    <a-button
                      type="link"
                      danger
                      @click="removeListItem(formContent.cta.features, index)"
                      >删除</a-button
                    >
                  </div>
                  <a-button
                    type="dashed"
                    @click="addListItem(formContent.cta.features, '')"
                    >新增亮点</a-button
                  >
                </div>
              </a-form-item>
            </template>
            <template v-if="form.page === 'products' && form.type === 'hero'">
              <a-form-item label="徽标文案">
                <a-input v-model:value="formContent.productsHero.badge" />
              </a-form-item>
              <a-form-item label="标题">
                <a-input v-model:value="formContent.productsHero.title" />
              </a-form-item>
              <a-form-item label="副标题">
                <a-textarea
                  v-model:value="formContent.productsHero.subtitle"
                  :rows="2"
                />
              </a-form-item>
              <a-form-item label="卖点列表">
                <div class="form-list">
                  <div
                    class="form-list-item"
                    v-for="(item, index) in formContent.productsHero.features"
                    :key="index"
                  >
                    <a-input
                      v-model:value="formContent.productsHero.features[index]"
                    />
                    <a-button
                      type="link"
                      danger
                      @click="
                        removeListItem(formContent.productsHero.features, index)
                      "
                      >删除</a-button
                    >
                  </div>
                  <a-button
                    type="dashed"
                    @click="addListItem(formContent.productsHero.features, '')"
                    >新增卖点</a-button
                  >
                </div>
              </a-form-item>
            </template>
            <template
              v-if="
                (form.page === 'docs' ||
                  form.page === 'announcements' ||
                  form.page === 'activities' ||
                  form.page === 'tutorials') &&
                form.type === 'hero'
              "
            >
              <a-form-item label="标题">
                <a-input v-model:value="formContent.docsHero.title" />
              </a-form-item>
              <a-form-item label="副标题">
                <a-textarea v-model:value="formContent.docsHero.subtitle" :rows="2" />
              </a-form-item>
            </template>
            <template
              v-if="
                (form.page === 'docs' ||
                  form.page === 'announcements' ||
                  form.page === 'activities' ||
                  form.page === 'tutorials') &&
                form.type === 'resources'
              "
            >
              <a-form-item label="模块标题">
                <a-input v-model:value="formContent.docsResources.title" />
              </a-form-item>
              <a-form-item label="资源卡片">
                <div class="form-list">
                  <div
                    class="form-list-item"
                    v-for="(item, index) in formContent.docsResources.items"
                    :key="index"
                  >
                    <a-select v-model:value="item.icon_key" style="width: 120px">
                      <a-select-option value="book">book</a-select-option>
                      <a-select-option value="video">video</a-select-option>
                      <a-select-option value="code">code</a-select-option>
                      <a-select-option value="chat">chat</a-select-option>
                    </a-select>
                    <a-input v-model:value="item.title" placeholder="标题" />
                    <a-input v-model:value="item.description" placeholder="描述" />
                    <a-input v-model:value="item.url" placeholder="链接" />
                    <a-button
                      type="link"
                      danger
                      @click="removeListItem(formContent.docsResources.items, index)"
                      >删除</a-button
                    >
                  </div>
                  <a-button
                    type="dashed"
                    @click="
                      addListItem(formContent.docsResources.items, {
                        icon_key: 'book',
                        title: '',
                        description: '',
                        url: '#',
                      })
                    "
                    >新增资源</a-button
                  >
                </div>
              </a-form-item>
            </template>

            <template v-if="form.page === 'help' && form.type === 'help_hero'">
              <a-form-item label="徽标文案">
                <a-input v-model:value="formContent.helpHero.badge" />
              </a-form-item>
              <a-form-item label="标题（前半）">
                <a-input v-model:value="formContent.helpHero.title_main" />
              </a-form-item>
              <a-form-item label="标题（渐变）">
                <a-input v-model:value="formContent.helpHero.title_gradient" />
              </a-form-item>
              <a-form-item label="副标题">
                <a-textarea v-model:value="formContent.helpHero.subtitle" :rows="2" />
              </a-form-item>
              <a-form-item label="搜索占位符">
                <a-input v-model:value="formContent.helpHero.search_placeholder" />
              </a-form-item>
              <a-form-item label="顶部统计">
                <div class="form-list">
                  <div class="form-list-item" v-for="(item, index) in formContent.helpHero.quick_stats" :key="index">
                    <a-input v-model:value="item.value" placeholder="值" style="width: 140px" />
                    <a-input v-model:value="item.label" placeholder="标签" />
                    <a-button type="link" danger @click="removeListItem(formContent.helpHero.quick_stats, index)"
                      >删除</a-button
                    >
                  </div>
                  <a-button
                    type="dashed"
                    @click="addListItem(formContent.helpHero.quick_stats, { value: '', label: '' })"
                    >新增统计</a-button
                  >
                </div>
              </a-form-item>
            </template>

            <template v-if="form.page === 'help' && form.type === 'help_actions'">
              <a-form-item label="快捷卡片">
                <div class="form-list">
                  <div class="form-list-item" v-for="(item, index) in formContent.helpActions.cards" :key="index">
                    <a-select v-model:value="item.key" style="width: 140px">
                      <a-select-option value="docs">docs</a-select-option>
                      <a-select-option value="tickets">tickets</a-select-option>
                      <a-select-option value="announcements">announcements</a-select-option>
                      <a-select-option value="contact">contact</a-select-option>
                    </a-select>
                    <a-input v-model:value="item.title" placeholder="标题" />
                    <a-input v-model:value="item.description" placeholder="描述" />
                    <a-input v-model:value="item.url" placeholder="链接/URL" />
                    <a-input v-if="item.key === 'tickets'" v-model:value="item.guest_url" placeholder="未登录跳转" />
                    <a-button type="link" danger @click="removeListItem(formContent.helpActions.cards, index)"
                      >删除</a-button
                    >
                  </div>
                  <a-button
                    type="dashed"
                    @click="
                      addListItem(formContent.helpActions.cards, {
                        key: 'docs',
                        title: '',
                        description: '',
                        url: '/docs',
                      })
                    "
                    >新增卡片</a-button
                  >
                </div>
              </a-form-item>
            </template>

            <template v-if="form.page === 'help' && form.type === 'help_faq'">
              <a-form-item label="模块标题">
                <a-input v-model:value="formContent.helpFaq.title" />
              </a-form-item>
              <a-form-item label="模块副标题">
                <a-input v-model:value="formContent.helpFaq.subtitle" />
              </a-form-item>
              <a-form-item label="分类Tabs">
                <div class="form-list">
                  <div class="form-list-item" v-for="(item, index) in formContent.helpFaq.categories" :key="index">
                    <a-select v-model:value="item.key" style="width: 140px">
                      <a-select-option value="all">all</a-select-option>
                      <a-select-option value="account">account</a-select-option>
                      <a-select-option value="payment">payment</a-select-option>
                      <a-select-option value="vps">vps</a-select-option>
                      <a-select-option value="billing">billing</a-select-option>
                    </a-select>
                    <a-input v-model:value="item.label" placeholder="显示名称" />
                    <a-button type="link" danger @click="removeListItem(formContent.helpFaq.categories, index)"
                      >删除</a-button
                    >
                  </div>
                  <a-button
                    type="dashed"
                    @click="addListItem(formContent.helpFaq.categories, { key: 'all', label: '' })"
                    >新增分类</a-button
                  >
                </div>
              </a-form-item>
              <a-form-item label="FAQ 列表">
                <div class="form-list">
                  <div class="form-list-item" v-for="(item, index) in formContent.helpFaq.faqs" :key="index">
                    <a-select v-model:value="item.category" style="width: 140px">
                      <a-select-option value="account">account</a-select-option>
                      <a-select-option value="payment">payment</a-select-option>
                      <a-select-option value="vps">vps</a-select-option>
                      <a-select-option value="billing">billing</a-select-option>
                    </a-select>
                    <a-input v-model:value="item.question" placeholder="问题" />
                    <a-textarea v-model:value="item.answer" :rows="2" placeholder="答案" />
                    <a-button type="link" danger @click="removeListItem(formContent.helpFaq.faqs, index)"
                      >删除</a-button
                    >
                  </div>
                  <a-button
                    type="dashed"
                    @click="addListItem(formContent.helpFaq.faqs, { category: 'account', question: '', answer: '' })"
                    >新增 FAQ</a-button
                  >
                </div>
              </a-form-item>
            </template>

            <template v-if="form.page === 'help' && form.type === 'help_contact'">
              <a-form-item label="标题">
                <a-input v-model:value="formContent.helpContact.title" />
              </a-form-item>
              <a-form-item label="说明">
                <a-textarea v-model:value="formContent.helpContact.description" :rows="2" />
              </a-form-item>
              <a-form-item label="联系渠道">
                <div class="form-list">
                  <div class="form-list-item" v-for="(item, index) in formContent.helpContact.channels" :key="index">
                    <a-select v-model:value="item.key" style="width: 140px">
                      <a-select-option value="chat">chat</a-select-option>
                      <a-select-option value="mail">mail</a-select-option>
                      <a-select-option value="tickets">tickets</a-select-option>
                    </a-select>
                    <a-input v-model:value="item.title" placeholder="标题" />
                    <a-input v-model:value="item.subtitle" placeholder="副标题" />
                    <a-button type="link" danger @click="removeListItem(formContent.helpContact.channels, index)"
                      >删除</a-button
                    >
                  </div>
                  <a-button
                    type="dashed"
                    @click="addListItem(formContent.helpContact.channels, { key: 'chat', title: '', subtitle: '' })"
                    >新增渠道</a-button
                  >
                </div>
              </a-form-item>
              <a-form-item label="CTA 标题">
                <a-input v-model:value="formContent.helpContact.cta_title" />
              </a-form-item>
              <a-form-item label="CTA 文案">
                <a-input v-model:value="formContent.helpContact.cta_desc" />
              </a-form-item>
              <a-form-item label="CTA 按钮">
                <a-row :gutter="16">
                  <a-col :span="12">
                    <a-input v-model:value="formContent.helpContact.cta_button_text" placeholder="按钮文案" />
                  </a-col>
                  <a-col :span="12">
                    <a-input v-model:value="formContent.helpContact.cta_url" placeholder="按钮链接" />
                  </a-col>
                </a-row>
              </a-form-item>
            </template>
            <template
              v-if="form.page === 'products' && form.type === 'calculator'"
            >
              <a-form-item label="标题">
                <a-input v-model:value="formContent.productsCalculator.title" />
              </a-form-item>
              <a-form-item label="说明">
                <a-textarea
                  v-model:value="formContent.productsCalculator.desc"
                  :rows="2"
                />
              </a-form-item>
              <a-form-item label="场景列表">
                <div class="form-list">
                  <div
                    class="form-list-item"
                    v-for="(item, index) in formContent.productsCalculator
                      .scenarios"
                    :key="index"
                  >
                    <a-input
                      v-model:value="item.icon"
                      placeholder="图标"
                      style="width: 90px"
                    />
                    <a-input v-model:value="item.name" placeholder="场景名称" />
                    <a-input
                      v-model:value="item.recommended"
                      placeholder="推荐配置"
                    />
                    <a-input-number
                      v-model:value="item.plan"
                      :min="0"
                      style="width: 80px"
                    />
                    <a-button
                      type="link"
                      danger
                      @click="
                        removeListItem(
                          formContent.productsCalculator.scenarios,
                          index,
                        )
                      "
                      >删除</a-button
                    >
                  </div>
                  <a-button
                    type="dashed"
                    @click="
                      addListItem(formContent.productsCalculator.scenarios, {
                        icon: '',
                        name: '',
                        recommended: '',
                        plan: 0,
                      })
                    "
                    >新增场景</a-button
                  >
                </div>
              </a-form-item>
            </template>
            <template
              v-if="form.page === 'products' && form.type === 'pricing'"
            >
              <a-form-item label="产品卡片">
                <div class="form-list">
                  <div
                    class="form-list-item column-group"
                    v-for="(item, index) in formContent.productsPricing
                      .products"
                    :key="index"
                  >
                    <a-input
                      v-model:value="item.icon"
                      placeholder="icon"
                      style="width: 120px"
                    />
                    <a-input v-model:value="item.name" placeholder="名称" />
                    <a-input
                      v-model:value="item.description"
                      placeholder="描述"
                    />
                    <a-input
                      v-model:value="item.price"
                      placeholder="价格"
                      style="width: 120px"
                    />
                    <a-input v-model:value="item.cta" placeholder="按钮文案" />
                    <a-switch v-model:checked="item.recommended" />
                    <div class="nested-list">
                      <div
                        class="form-list-item"
                        v-for="(res, resIndex) in item.resources"
                        :key="resIndex"
                      >
                        <a-input
                          v-model:value="res.label"
                          placeholder="资源名"
                          style="width: 90px"
                        />
                        <a-input v-model:value="res.value" placeholder="值" />
                        <a-input-number
                          v-model:value="res.percent"
                          :min="0"
                          :max="100"
                          style="width: 80px"
                        />
                        <a-button
                          type="link"
                          danger
                          @click="removeListItem(item.resources, resIndex)"
                          >删除</a-button
                        >
                      </div>
                      <a-button
                        type="dashed"
                        @click="
                          addListItem(item.resources, {
                            label: '',
                            value: '',
                            percent: 0,
                          })
                        "
                        >新增资源</a-button
                      >
                    </div>
                    <div class="nested-list">
                      <div
                        class="form-list-item"
                        v-for="(feature, fIndex) in item.features"
                        :key="fIndex"
                      >
                        <a-input v-model:value="item.features[fIndex]" />
                        <a-button
                          type="link"
                          danger
                          @click="removeListItem(item.features, fIndex)"
                          >删除</a-button
                        >
                      </div>
                      <a-button
                        type="dashed"
                        @click="addListItem(item.features, '')"
                        >新增特性</a-button
                      >
                    </div>
                    <a-button
                      type="link"
                      danger
                      @click="
                        removeListItem(
                          formContent.productsPricing.products,
                          index,
                        )
                      "
                      >删除产品</a-button
                    >
                  </div>
                  <a-button
                    type="dashed"
                    @click="
                      addListItem(formContent.productsPricing.products, {
                        icon: 'cloud',
                        name: '',
                        description: '',
                        price: '',
                        recommended: false,
                        cta: '',
                        resources: [],
                        features: [],
                      })
                    "
                    >新增产品</a-button
                  >
                </div>
              </a-form-item>
            </template>
            <template
              v-if="form.page === 'products' && form.type === 'comparison'"
            >
              <a-form-item label="标题">
                <a-input v-model:value="formContent.productsComparison.title" />
              </a-form-item>
              <a-form-item label="对比表">
                <div class="form-list">
                  <div
                    class="form-list-item column-group"
                    v-for="(row, index) in formContent.productsComparison.rows"
                    :key="index"
                  >
                    <a-input v-model:value="row.feature" placeholder="配置项" />
                    <div class="nested-list">
                      <div
                        class="form-list-item"
                        v-for="(val, vIndex) in row.values"
                        :key="vIndex"
                      >
                        <a-input v-model:value="row.values[vIndex]" />
                        <a-button
                          type="link"
                          danger
                          @click="removeListItem(row.values, vIndex)"
                          >删除</a-button
                        >
                      </div>
                      <a-button
                        type="dashed"
                        @click="addListItem(row.values, '')"
                        >新增值</a-button
                      >
                    </div>
                    <a-button
                      type="link"
                      danger
                      @click="
                        removeListItem(
                          formContent.productsComparison.rows,
                          index,
                        )
                      "
                      >删除行</a-button
                    >
                  </div>
                  <a-button
                    type="dashed"
                    @click="
                      addListItem(formContent.productsComparison.rows, {
                        feature: '',
                        values: [],
                      })
                    "
                    >新增行</a-button
                  >
                </div>
              </a-form-item>
            </template>
            <template v-if="form.page === 'products' && form.type === 'cta'">
              <a-form-item label="标题">
                <a-input v-model:value="formContent.productsCta.title" />
              </a-form-item>
              <a-form-item label="描述">
                <a-textarea
                  v-model:value="formContent.productsCta.desc"
                  :rows="2"
                />
              </a-form-item>
              <a-row :gutter="16">
                <a-col :span="12">
                  <a-form-item label="按钮文案">
                    <a-input
                      v-model:value="formContent.productsCta.contact_text"
                    />
                  </a-form-item>
                </a-col>
                <a-col :span="12">
                  <a-form-item label="按钮链接">
                    <a-input
                      v-model:value="formContent.productsCta.contact_link"
                    />
                  </a-form-item>
                </a-col>
              </a-row>
              <a-form-item label="邮箱">
                <a-input v-model:value="formContent.productsCta.email" />
              </a-form-item>
            </template>
            <template v-if="form.type === 'footer'">
              <a-form-item label="Footer 描述">
                <a-textarea
                  v-model:value="formContent.footer.description"
                  :rows="2"
                />
              </a-form-item>
              <a-form-item label="社交链接">
                <div class="form-list">
                  <div
                    class="form-list-item"
                    v-for="(item, index) in formContent.footer.social_links"
                    :key="index"
                  >
                    <a-input
                      v-model:value="item.key"
                      placeholder="key"
                      style="width: 120px"
                    />
                    <a-input v-model:value="item.url" placeholder="URL" />
                    <a-button
                      type="link"
                      danger
                      @click="
                        removeListItem(formContent.footer.social_links, index)
                      "
                      >删除</a-button
                    >
                  </div>
                  <a-button
                    type="dashed"
                    @click="
                      addListItem(formContent.footer.social_links, {
                        key: '',
                        url: '',
                      })
                    "
                    >新增社交链接</a-button
                  >
                </div>
              </a-form-item>
              <a-form-item label="栏目与链接">
                <div class="form-list">
                  <div
                    class="form-list-item column-group"
                    v-for="(section, index) in formContent.footer.sections"
                    :key="index"
                  >
                    <a-input
                      v-model:value="section.title"
                      placeholder="栏目标题"
                    />
                    <div class="nested-list">
                      <div
                        class="form-list-item"
                        v-for="(link, linkIndex) in section.links"
                        :key="linkIndex"
                      >
                        <a-input
                          v-model:value="link.label"
                          placeholder="链接文案"
                        />
                        <a-input
                          v-model:value="link.url"
                          placeholder="链接地址"
                        />
                        <a-button
                          type="link"
                          danger
                          @click="removeListItem(section.links, linkIndex)"
                          >删除</a-button
                        >
                      </div>
                      <a-button
                        type="dashed"
                        @click="
                          addListItem(section.links, { label: '', url: '' })
                        "
                        >新增链接</a-button
                      >
                    </div>
                    <a-button
                      type="link"
                      danger
                      @click="
                        removeListItem(formContent.footer.sections, index)
                      "
                      >删除栏目</a-button
                    >
                  </div>
                  <a-button
                    type="dashed"
                    @click="
                      addListItem(formContent.footer.sections, {
                        title: '',
                        links: [],
                      })
                    "
                    >新增栏目</a-button
                  >
                </div>
              </a-form-item>
              <a-form-item label="底部徽标">
                <div class="form-list">
                  <div
                    class="form-list-item"
                    v-for="(badge, index) in formContent.footer.badges"
                    :key="index"
                  >
                    <a-input v-model:value="formContent.footer.badges[index]" />
                    <a-button
                      type="link"
                      danger
                      @click="removeListItem(formContent.footer.badges, index)"
                      >删除</a-button
                    >
                  </div>
                  <a-button
                    type="dashed"
                    @click="addListItem(formContent.footer.badges, '')"
                    >新增徽标</a-button
                  >
                </div>
              </a-form-item>
            </template>
            <a-form-item label="自定义HTML" v-if="form.type === 'custom_html'">
              <a-textarea v-model:value="form.custom_html" :rows="6" />
            </a-form-item>
            <a-row :gutter="16">
              <a-col :span="8">
                <a-form-item label="语言">
                  <a-select v-model:value="form.lang">
                    <a-select-option value="zh-CN">简体中文</a-select-option>
                    <a-select-option value="en-US">English</a-select-option>
                  </a-select>
                </a-form-item>
              </a-col>
              <a-col :span="8">
                <a-form-item label="排序">
                  <a-input-number
                    v-model:value="form.sort_order"
                    :min="0"
                    style="width: 100%"
                  />
                </a-form-item>
              </a-col>
              <a-col :span="8">
                <a-form-item label="可见">
                  <a-switch v-model:checked="form.visible" />
                </a-form-item>
              </a-col>
            </a-row>
          </a-form>
        </a-col>
        <a-col :span="15" class="editor-preview">
          <div class="preview-panel">
            <div class="preview-header">
              <span>Preview</span>
              <div class="preview-controls">
                <a-switch
                  v-model:checked="previewZoomEnabled"
                  checked-children="720p"
                  un-checked-children="Fit"
                  size="small"
                />
                <a-slider
                  v-model:value="previewScalePercent"
                  :min="30"
                  :max="100"
                  :step="5"
                  :disabled="!previewZoomEnabled"
                  style="width: 140px"
                />
              </div>
            </div>
            <div class="preview-body" ref="previewBodyRef">
              <div class="preview-viewport" ref="previewViewportRef">
                <div
                  class="preview-canvas"
                  :class="{ 'is-zoom': previewZoomEnabled }"
                  :style="previewCanvasStyle"
                >
                  <div class="preview-content-wrapper">
                <template v-if="form.type === 'custom_html'">
                  <div
                    class="preview-html"
                    v-html="form.custom_html || '<p>暂无内容</p>'"
                  ></div>
                </template>
                <template
                  v-else-if="form.page === 'home' && form.type === 'hero'"
                >
                  <div class="home-page preview-surface preview-hero-surface">
                    <HomeHeroBlock
                      :hero-content="previewHomeHeroContent"
                      :typewriter-text="previewHomeTypewriterText"
                      :stats="previewHomeStats"
                      :animated-stats="previewHomeAnimatedStats"
                      :hero-cards="previewHomeCards"
                    />
                  </div>
                </template>
                <template
                  v-else-if="form.page === 'home' && form.type === 'features'"
                >
                  <div class="home-page preview-surface">
                    <HomeFeaturesBlock
                      :content="previewHomeFeaturesContent"
                      :features="previewHomeFeatures"
                    />
                  </div>
                </template>
                <template
                  v-else-if="form.page === 'home' && form.type === 'products'"
                >
                  <div class="home-page preview-surface">
                    <HomeProductsBlock
                      :content="previewHomeProductsContent"
                      :products="previewHomeProducts"
                    />
                  </div>
                </template>
                <template
                  v-else-if="form.page === 'home' && form.type === 'cta'"
                >
                  <div class="home-page preview-surface">
                    <HomeCtaBlock
                      :content="previewHomeCtaContent"
                      :features="previewHomeCtaFeatures"
                    />
                  </div>
                </template>
                <template
                  v-else-if="form.page === 'products' && form.type === 'hero'"
                >
                  <div class="products-page preview-surface">
                    <ProductsHeroBlock :content="previewProductsHeroContent" />
                  </div>
                </template>
                <template
                  v-else-if="
                    form.page === 'products' && form.type === 'calculator'
                  "
                >
                  <div class="products-page preview-surface">
                    <ProductsCalculatorBlock
                      :content="previewProductsCalculatorContent"
                      :scenarios="previewProductsScenarios"
                      :selected-scenario="previewSelectedScenario"
                      :on-select="previewSelectScenario"
                    />
                  </div>
                </template>
                <template
                  v-else-if="form.page === 'products' && form.type === 'pricing'"
                >
                  <div class="products-page preview-surface">
                    <ProductsPricingBlock
                      :products="previewProductsPricing"
                      :selected-plan="previewSelectedPlan"
                      :on-select="previewSelectPlan"
                      :on-hover="previewHoverPlan"
                    />
                  </div>
                </template>
                <template
                  v-else-if="
                    form.page === 'products' && form.type === 'comparison'
                  "
                >
                  <div class="products-page preview-surface">
                    <ProductsComparisonBlock
                      :content="previewProductsComparisonContent"
                      :products="previewProductsComparisonProducts"
                      :rows="previewProductsComparisonRows"
                    />
                  </div>
                </template>
                <template v-else-if="form.page === 'products' && form.type === 'cta'">
                  <div class="products-page preview-surface">
                    <ProductsCtaBlock :content="previewProductsCtaContent" />
                  </div>
                </template>
                <template v-else-if="form.page === 'help' && form.type === 'help_hero'">
                  <div class="help-page preview-surface">
                    <HelpHeroBlock :content="previewHelpHeroContent" v-model:searchQuery="previewHelpSearchQuery" />
                  </div>
                </template>
                <template v-else-if="form.page === 'help' && form.type === 'help_actions'">
                  <div class="help-page preview-surface">
                    <HelpActionsBlock :content="previewHelpActionsContent" :is-authenticated="true" />
                  </div>
                </template>
                <template v-else-if="form.page === 'help' && form.type === 'help_faq'">
                  <div class="help-page preview-surface">
                    <HelpFaqBlock :content="previewHelpFaqContent" :search-query="previewHelpSearchQuery" @clear-search="previewHelpSearchQuery = ''" />
                  </div>
                </template>
                <template v-else-if="form.page === 'help' && form.type === 'help_contact'">
                  <div class="help-page preview-surface">
                    <HelpContactBlock :content="previewHelpContactContent" />
                  </div>
                </template>
                <template v-else-if="form.type === 'footer'">
                  <div class="public-layout preview-surface">
                    <FooterBlock
                      :site-name="previewSiteName"
                      :logo-url="previewLogoUrl"
                      :content="previewFooterContent"
                      :sections="previewFooterSections"
                      :badges="previewFooterBadges"
                    />
                  </div>
                </template>
                <template v-else>
                  <pre class="preview-json">{{
                    JSON.stringify(activeContent, null, 2)
                  }}</pre>
                </template>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </a-col>
      </a-row>
    </a-modal>
  </div>
</template>
<script setup lang="ts">
import {
  ref,
  reactive,
  onMounted,
  onBeforeUnmount,
  computed,
  watch,
  nextTick,
} from "vue";
import { message } from "ant-design-vue";
import { PlusOutlined } from "@ant-design/icons-vue";
import ProTable from "@/components/ProTable.vue";
import HomeHeroBlock from "@/components/cms/blocks/home/HomeHeroBlock.vue";
import HomeFeaturesBlock from "@/components/cms/blocks/home/HomeFeaturesBlock.vue";
import HomeProductsBlock from "@/components/cms/blocks/home/HomeProductsBlock.vue";
import HomeCtaBlock from "@/components/cms/blocks/home/HomeCtaBlock.vue";
import ProductsHeroBlock from "@/components/cms/blocks/products/ProductsHeroBlock.vue";
import ProductsCalculatorBlock from "@/components/cms/blocks/products/ProductsCalculatorBlock.vue";
import ProductsPricingBlock from "@/components/cms/blocks/products/ProductsPricingBlock.vue";
import ProductsComparisonBlock from "@/components/cms/blocks/products/ProductsComparisonBlock.vue";
import ProductsCtaBlock from "@/components/cms/blocks/products/ProductsCtaBlock.vue";
import HelpHeroBlock from "@/components/cms/blocks/help/HelpHeroBlock.vue";
import HelpActionsBlock from "@/components/cms/blocks/help/HelpActionsBlock.vue";
import HelpFaqBlock from "@/components/cms/blocks/help/HelpFaqBlock.vue";
import HelpContactBlock from "@/components/cms/blocks/help/HelpContactBlock.vue";
import FooterBlock from "@/components/cms/blocks/FooterBlock.vue";
import {
  ThunderIcon,
  ShieldIcon,
  GlobeIcon,
  ServerIcon,
  DatabaseIcon,
  SettingsIcon,
  CloudIcon as HomeCloudIcon,
  CubeIcon,
  CodeIcon,
} from "@/components/cms/blocks/home/icons";
import {
  CloudIcon,
  RocketIcon,
  BoltIcon,
  BuildingIcon,
} from "@/components/cms/blocks/products/icons";
import "@/pages/public/Home.vue";
import "@/pages/public/Products.vue";
import "@/layouts/PublicLayout.vue";
import {
  listCmsBlocks,
  createCmsBlock,
  updateCmsBlock,
  deleteCmsBlock,
} from "@/services/admin";
const loading = ref(false);
const submitting = ref(false);
const modalVisible = ref(false);
const isEditing = ref(false);
const previewZoomEnabled = ref(true);
const previewScalePercent = ref(70);
const previewBodyRef = ref<HTMLElement | null>(null);
const previewViewportRef = ref<HTMLElement | null>(null);
const previewBaseScale = ref(1);
let previewResizeObserver: ResizeObserver | null = null;

const updatePreviewScale = () => {
  if (!previewBodyRef.value) return;
  const width = previewBodyRef.value.clientWidth || 0;
  const height = previewBodyRef.value.clientHeight || 0;
  if (!width || !height) return;
  const base = Math.min(width / 16, height / 9);
  const viewportWidth = Math.floor(base * 16);
  const viewportHeight = Math.floor(base * 9);
  if (previewViewportRef.value) {
    previewViewportRef.value.style.width = `${viewportWidth}px`;
    previewViewportRef.value.style.height = `${viewportHeight}px`;
  }
  const scale = Math.min(viewportWidth / PREVIEW_W, viewportHeight / PREVIEW_H);
  previewBaseScale.value = Math.max(0.1, scale);
};

const PREVIEW_W = 1280;
const PREVIEW_H = 720;
const previewCanvasStyle = computed(() => {
  if (!previewZoomEnabled.value) return {};
  const scale = previewBaseScale.value * (previewScalePercent.value / 100);
  return {
    width: `${PREVIEW_W}px`,
    height: `${PREVIEW_H}px`,
    // Use zoom so layout size matches visual size (transform would get clipped by the viewport).
    zoom: scale as any,
  };
});
const blocks = ref<any[]>([]);

watch(
  modalVisible,
  async (open) => {
    if (!open) return;
    await nextTick();
    updatePreviewScale();
    if (!previewResizeObserver) {
      previewResizeObserver = new ResizeObserver(updatePreviewScale);
    }
    if (previewBodyRef.value) {
      previewResizeObserver.observe(previewBodyRef.value);
    }
  },
  { immediate: false },
);


const filters = reactive({ page: "", type: "" });
const pagination = reactive({ current: 1, pageSize: 20, total: 0 });
const form = reactive({
  id: undefined,
  page: "home",
  type: "hero",
  title: "",
  subtitle: "",
  content_json: "",
  custom_html: "",
  lang: "zh-CN",
  visible: true,
  sort_order: 0,
});

// Avoid mismatched page/type combos that fall back to raw JSON preview.
watch(
  () => form.page,
  (page) => {
    const isHelpType =
      typeof form.type === "string" && form.type.startsWith("help_");
    if (page === "help") {
      if (form.type !== "custom_html" && !isHelpType) {
        form.type = "help_hero";
      }
      return;
    }
    if (isHelpType) {
      form.type = "hero";
    }
  },
);
const structuredTypes = [
  "hero",
  "stats",
  "features",
  "products",
  "cta",
  "posts",
  "resources",
  "calculator",
  "pricing",
  "comparison",
  "footer",
  "help_hero",
  "help_actions",
  "help_faq",
  "help_contact",
];
const formContent = reactive({
  hero: {
    badge: "下一代云计算平台",
    title1: "构建未来",
    subtitle:
      "企业级云服务器，全球节点覆盖，99.99% SLA 保障。秒级部署，弹性扩展，为您的业务保驾护航。",
    primary_button_text: "立即开始",
    primary_button_link: "/register",
    secondary_button_text: "浏览产品",
    secondary_button_link: "/products",
    typewriter_words: ["云端智能", "无限可能", "卓越性能", "安全可靠"],
    cards: [
      { title: "极速部署", desc: "60秒开机" },
      { title: "全球网络", desc: "覆盖150+国家" },
      { title: "多层防护", desc: "DDoS防御" },
    ],
    stats: [
      { value: 99.99, suffix: "%", label: "可用性" },
      { value: 50, suffix: "+", label: "全球节点" },
      { value: 100, suffix: "K+", label: "企业用户" },
    ],
  },
  stats: {
    items: [
      { value: 99.99, suffix: "%", label: "可用性" },
      { value: 50, suffix: "+", label: "全球节点" },
      { value: 100, suffix: "K+", label: "企业用户" },
    ],
  },
  features: {
    badge: "核心优势",
    title: "为什么选择我们的云服务",
    desc: "我们提供企业级基础设施，助力您的业务快速增长",
    items: [
      {
        title: "极致性能",
        description: "采用最新一代CPU和NVMe SSD，提供卓越计算性能和I/O吞吐量",
      },
      {
        title: "安全可靠",
        description: "多层安全防护体系，DDoS防护、WAF、SSL证书全方位保障",
      },
      {
        title: "全球覆盖",
        description: "50+ 数据中心遍布全球，BGP多线接入，智能调度最优线路",
      },
      {
        title: "弹性伸缩",
        description: "秒级扩容缩容，按需付费，资源利用率最大化",
      },
      {
        title: "数据保护",
        description: "多重备份机制，快照回滚，异地容灾，数据安全无忧",
      },
      {
        title: "简单易用",
        description: "可视化控制台，一键部署应用，API 丰富的自动化运维",
      },
    ],
  },
  products: {
    badge: "产品系列",
    title: "满足各种规模需求",
    items: [
      {
        tag: "入门首选",
        title: "云服务器",
        description: "适合个人开发者、小型项目",
        price: "29",
      },
      {
        tag: "企业推荐",
        title: "弹性计算",
        description: "适合中型企业、Web 应用",
        price: "99",
      },
      {
        tag: "性能旗舰",
        title: "GPU 实例",
        description: "适合 AI 训练、渲染任务",
        price: "399",
      },
    ],
  },
  cta: {
    title: "准备好开启您的云端之旅了吗？",
    desc: "立即注册，新用户享受免费试用额度，体验企业级云服务",
    button_text: "免费开始使用",
    button_link: "/register",
    features: ["无需绑定信用卡", "随时取消", "24/7 技术支持"],
  },
  productsHero: {
    badge: "灵活配置，按需选择",
    title: "选择最适合您的云服务方案",
    subtitle:
      "从入门到企业级，我们提供全面的云服务器解决方案。所有套餐均包含99.99% SLA保障。",
    features: ["秒级部署", "弹性扩容", "99.99% SLA", "24/7 支持"],
  },
  productsCalculator: {
    title: "智能推荐",
    desc: "选择您的使用场景，我们将为您推荐最佳配置",
    scenarios: [
      { icon: "📝", name: "个人博客", recommended: "基础型 - 1核1G", plan: 0 },
      { icon: "🛒", name: "小型电商", recommended: "标准型 - 2核4G", plan: 1 },
      {
        icon: "🎮",
        name: "游戏服务器",
        recommended: "高性能型 - 4核8G",
        plan: 2,
      },
      { icon: "🏢", name: "企业应用", recommended: "企业型 - 8核16G", plan: 3 },
    ],
  },
  productsPricing: {
    products: [
      {
        icon: "cloud",
        name: "基础型",
        description: "适合个人博客、小型网站",
        price: "29",
        recommended: false,
        cta: "立即选购",
        resources: [
          { label: "CPU", value: "1 核", percent: 25 },
          { label: "内存", value: "1 GB", percent: 12.5 },
          { label: "存储", value: "20 GB", percent: 20 },
          { label: "带宽", value: "1 Mbps", percent: 10 },
        ],
        features: [
          "默认 1核1G 配置",
          "20GB SSD 高速云盘",
          "1Mbps 带宽",
          "Linux 操作系统",
          "免费备案服务",
          "99.99% 可用性",
          "7天无理由退款",
          "24/7 工单支持",
        ],
      },
      {
        icon: "rocket",
        name: "标准型",
        description: "适合中小企业、Web 应用",
        price: "59",
        recommended: false,
        cta: "立即选购",
        resources: [
          { label: "CPU", value: "2 核", percent: 50 },
          { label: "内存", value: "4 GB", percent: 50 },
          { label: "存储", value: "40 GB", percent: 40 },
          { label: "带宽", value: "3 Mbps", percent: 30 },
        ],
        features: [
          "默认 2核4G 配置",
          "40GB SSD 高速云盘",
          "3Mbps 带宽",
          "Linux/Windows 系统",
          "免费自动备份",
          "负载均衡支持",
          "DDoS 防护",
          "优先技术支持",
        ],
      },
      {
        icon: "bolt",
        name: "高性能型",
        description: "适合计算密集型应用",
        price: "129",
        recommended: true,
        cta: "立即选购",
        resources: [
          { label: "CPU", value: "4 核", percent: 100 },
          { label: "内存", value: "8 GB", percent: 100 },
          { label: "存储", value: "80 GB", percent: 80 },
          { label: "带宽", value: "5 Mbps", percent: 50 },
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
          "SLA 保障",
        ],
      },
      {
        icon: "building",
        name: "企业型",
        description: "适合大型企业、关键业务",
        price: "299",
        recommended: false,
        cta: "联系销售",
        resources: [
          { label: "CPU", value: "8 核", percent: 100 },
          { label: "内存", value: "16 GB", percent: 100 },
          { label: "存储", value: "160 GB", percent: 100 },
          { label: "带宽", value: "10 Mbps", percent: 100 },
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
          "99.995% SLA",
        ],
      },
    ],
  },
  productsComparison: {
    title: "详细配置对比",
    rows: [
      { feature: "CPU", values: ["1 核", "2 核", "4 核", "8 核"] },
      { feature: "内存", values: ["1 GB", "4 GB", "8 GB", "16 GB"] },
      {
        feature: "存储",
        values: ["20 GB SSD", "40 GB SSD", "80 GB SSD", "160 GB SSD"],
      },
      { feature: "带宽", values: ["1 Mbps", "3 Mbps", "5 Mbps", "10 Mbps"] },
      {
        feature: "操作系统",
        values: ["Linux", "Linux/Windows", "任意系统", "任意系统"],
      },
      { feature: "流量限制", values: ["不限", "不限", "不限", "不限"] },
      {
        feature: "备份数量",
        values: ["手动", "每天1次", "每天1次", "实时备份"],
      },
      { feature: "DDoS防护", values: ["基础", "基础", "高级", "企业级"] },
      { feature: "技术支持", values: ["工单", "工单", "优先", "专属经理"] },
      { feature: "SLA保障", values: ["99.99%", "99.99%", "99.99%", "99.995%"] },
    ],
  },
  productsCta: {
    title: "需要定制方案？",
    desc: "联系我们的销售团队，为您量身定制企业级云解决方案",
    contact_text: "联系销售",
    contact_link: "/console/tickets",
    email: "sales@example.com",
  },
  docsHero: {
    title: "文档中心",
    subtitle: "官方文档与最佳实践",
  },
  docsResources: {
    title: "相关资源",
    items: [
      { icon_key: "book", title: "API 文档", description: "完整的 API 参考手册和示例代码", url: "#" },
      { icon_key: "video", title: "视频教程", description: "手把手教您使用各项功能", url: "#" },
      { icon_key: "code", title: "代码示例", description: "常用场景的代码片段和最佳实践", url: "#" },
      { icon_key: "chat", title: "社区支持", description: "加入讨论，获取帮助与经验分享", url: "#" },
    ],
  },
  helpHero: {
    badge: "帮助中心",
    title_main: "我们能为您",
    title_gradient: "做些什么？",
    subtitle: "快速找到您需要的答案，或联系我们的专业团队获取支持",
    search_placeholder: "搜索问题、关键词...",
    quick_stats: [
      { value: "100+", label: "常见问题" },
      { value: "24/7", label: "在线支持" },
      { value: "<5m", label: "平均响应" },
      { value: "99.9%", label: "满意度" },
    ],
  },
  helpActions: {
    cards: [
      { key: "docs", title: "文档中心", description: "详细的产品文档和使用指南", url: "/docs" },
      { key: "tickets", title: "提交工单", description: "获取一对一的技术支持", url: "/console/tickets", guest_url: "/auth/login" },
      { key: "announcements", title: "最新公告", description: "系统更新与重要通知", url: "/announcements" },
      { key: "contact", title: "邮件支持", description: "support@example.com", url: "mailto:support@example.com" },
    ],
  },
  helpFaq: {
    title: "常见问题",
    subtitle: "快速找到您关心的问题答案",
    categories: [
      { key: "all", label: "全部" },
      { key: "account", label: "账号相关" },
      { key: "payment", label: "支付问题" },
      { key: "vps", label: "VPS使用" },
      { key: "billing", label: "账单退款" },
    ],
    faqs: [
      { category: "account", question: "如何注册账号？", answer: "点击页面右上角的\"注册\"按钮，填写用户名、邮箱和密码即可完成注册。" },
      { category: "payment", question: "支持哪些支付方式？", answer: "我们支持支付宝、微信支付、银行卡等多种支付方式。" },
      { category: "vps", question: "VPS多久可以开通？", answer: "订单支付成功后通常在1-5分钟内完成开通。" },
      { category: "billing", question: "退款政策是什么？", answer: "我们提供7天无理由退款服务（具体以服务条款为准）。" },
    ],
  },
  helpContact: {
    title: "还有问题？",
    description: "我们的专业支持团队随时准备为您提供帮助",
    channels: [
      { key: "chat", title: "在线客服", subtitle: "工作日 9:00 - 18:00" },
      { key: "mail", title: "邮件支持", subtitle: "24小时内回复" },
      { key: "tickets", title: "工单系统", subtitle: "技术问题优先处理" },
    ],
    cta_title: "立即开始使用",
    cta_desc: "注册账号，享受专业的云服务",
    cta_button_text: "免费注册",
    cta_url: "/auth/register",
  },
  footer: {
    description:
      "专业的云服务提供商，为企业提供可靠、安全、高性能的云计算解决方案",
    social_links: [
      { key: "github", url: "#" },
      { key: "twitter", url: "#" },
      { key: "discord", url: "#" },
    ],
    sections: [
      {
        title: "产品服务",
        links: [
          { label: "云服务器", url: "/products" },
          { label: "对象存储", url: "/products" },
          { label: "云数据库", url: "/products" },
          { label: "CDN加速", url: "/products" },
        ],
      },
      {
        title: "资源中心",
        links: [
          { label: "开发文档", url: "/docs" },
          { label: "帮助中心", url: "/help" },
          { label: "产品公告", url: "/announcements" },
          { label: "教程指南", url: "/tutorials" },
        ],
      },
      {
        title: "客户支持",
        links: [
          { label: "帮助中心", url: "/help" },
          { label: "提交工单", url: "/console/tickets" },
          { label: "联系我们", url: "#" },
          { label: "服务状态", url: "#" },
        ],
      },
      {
        title: "关于我们",
        links: [
          { label: "关于我们", url: "#" },
          { label: "加入我们", url: "#" },
          { label: "隐私政策", url: "#" },
          { label: "服务条款", url: "#" },
        ],
      },
    ],
    badges: ["99.99% Uptime", "SOC2 Certified"],
  },
});
const defaultHeroCards = JSON.parse(JSON.stringify(formContent.hero.cards));

const isHeroCardList = (cards: any) =>
  Array.isArray(cards) &&
  cards.some(
    (item) =>
      item && (typeof item.title === "string" || typeof item.desc === "string"),
  );
const safeParse = (raw: string) => {
  if (!raw) return {};
  try {
    return JSON.parse(raw);
  } catch (error) {
    return {};
  }
};

const resolveHeroStatsFromBlock = (record: any) => {
  if (!record?.content_json) return [];
  const parsed = safeParse(record.content_json || "");
  if (Array.isArray(parsed.items)) return parsed.items;
  if (Array.isArray(parsed.stats)) return parsed.stats;
  return [];
};
const applyContent = (type: string, content: any) => {
  if (!content || typeof content !== "object") return;
  if (type === "hero" && form.page === "products")
    Object.assign(formContent.productsHero, content);
  if (type === "hero" && form.page === "home") {
    const next = { ...content };
    if (!isHeroCardList(next.cards)) {
      delete next.cards;
    }
    Object.assign(formContent.hero, next);
  }
  if (type === "stats" && Array.isArray(content.items))
    formContent.stats.items = content.items;
  if (type === "features") Object.assign(formContent.features, content);
  if (type === "products") Object.assign(formContent.products, content);
  if (type === "cta" && form.page === "products")
    Object.assign(formContent.productsCta, content);
  if (type === "cta" && form.page === "home")
    Object.assign(formContent.cta, content);
  if (type === "calculator")
    Object.assign(formContent.productsCalculator, content);
  if (type === "pricing") Object.assign(formContent.productsPricing, content);
  if (type === "comparison")
    Object.assign(formContent.productsComparison, content);
  if (
    type === "hero" &&
    (form.page === "docs" ||
      form.page === "announcements" ||
      form.page === "activities" ||
      form.page === "tutorials")
  )
    Object.assign(formContent.docsHero, content);
  if (type === "resources")
    Object.assign(formContent.docsResources, content);
  if (type === "footer") Object.assign(formContent.footer, content);
  if (type === "help_hero") Object.assign(formContent.helpHero, content);
  if (type === "help_actions") Object.assign(formContent.helpActions, content);
  if (type === "help_faq") Object.assign(formContent.helpFaq, content);
  if (type === "help_contact") Object.assign(formContent.helpContact, content);
};
const buildContentJson = (type: string) => {
  if (type === "hero" && form.page === "products")
    return formContent.productsHero;
  if (
    type === "hero" &&
    (form.page === "docs" ||
      form.page === "announcements" ||
      form.page === "activities" ||
      form.page === "tutorials")
  )
    return formContent.docsHero;
  if (type === "hero") return formContent.hero;
  if (type === "stats") return formContent.stats;
  if (type === "features") return formContent.features;
  if (type === "products") return formContent.products;
  if (type === "cta" && form.page === "products")
    return formContent.productsCta;
  if (type === "cta") return formContent.cta;
  if (
    type === "resources" &&
    (form.page === "docs" ||
      form.page === "announcements" ||
      form.page === "activities" ||
      form.page === "tutorials")
  )
    return formContent.docsResources;
  if (type === "posts") return {};
  if (type === "calculator") return formContent.productsCalculator;
  if (type === "pricing") return formContent.productsPricing;
  if (type === "comparison") return formContent.productsComparison;
  if (type === "footer") return formContent.footer;
  if (type === "help_hero") return formContent.helpHero;
  if (type === "help_actions") return formContent.helpActions;
  if (type === "help_faq") return formContent.helpFaq;
  if (type === "help_contact") return formContent.helpContact;
  return {};
};
const addListItem = (list: any[], value: any) => {
  list.push(value);
};
const removeListItem = (list: any[], index: number) => {
  list.splice(index, 1);
};
const defaultContent = JSON.parse(JSON.stringify(formContent));
const resetContent = () => {
  Object.assign(formContent.hero, defaultContent.hero);
  formContent.stats.items = [...defaultContent.stats.items];
  Object.assign(formContent.features, defaultContent.features);
  Object.assign(formContent.products, defaultContent.products);
  Object.assign(formContent.cta, defaultContent.cta);
  Object.assign(formContent.productsHero, defaultContent.productsHero);
  Object.assign(
    formContent.productsCalculator,
    defaultContent.productsCalculator,
  );
  Object.assign(formContent.productsPricing, defaultContent.productsPricing);
  Object.assign(
    formContent.productsComparison,
    defaultContent.productsComparison,
  );
  Object.assign(formContent.productsCta, defaultContent.productsCta);
  Object.assign(formContent.docsHero, defaultContent.docsHero);
  Object.assign(formContent.docsResources, defaultContent.docsResources);
  Object.assign(formContent.footer, defaultContent.footer);
  Object.assign(formContent.helpHero, defaultContent.helpHero);
  Object.assign(formContent.helpActions, defaultContent.helpActions);
  Object.assign(formContent.helpFaq, defaultContent.helpFaq);
  Object.assign(formContent.helpContact, defaultContent.helpContact);
};
const activeContent = computed(() => {
  if (form.type === "hero" && form.page === "products")
    return formContent.productsHero;
  if (
    form.type === "hero" &&
    (form.page === "docs" ||
      form.page === "announcements" ||
      form.page === "activities" ||
      form.page === "tutorials")
  )
    return formContent.docsHero;
  if (form.type === "hero") return formContent.hero;
  if (form.type === "stats") return formContent.stats;
  if (form.type === "features") return formContent.features;
  if (form.type === "products") return formContent.products;
  if (form.type === "cta" && form.page === "products")
    return formContent.productsCta;
  if (form.type === "cta") return formContent.cta;
  if (
    form.type === "resources" &&
    (form.page === "docs" ||
      form.page === "announcements" ||
      form.page === "activities" ||
      form.page === "tutorials")
  )
    return formContent.docsResources;
  if (form.type === "posts") return {};
  if (form.type === "calculator") return formContent.productsCalculator;
  if (form.type === "pricing") return formContent.productsPricing;
  if (form.type === "comparison") return formContent.productsComparison;
  if (form.type === "footer") return formContent.footer;
  if (form.type === "help_hero") return formContent.helpHero;
  if (form.type === "help_actions") return formContent.helpActions;
  if (form.type === "help_faq") return formContent.helpFaq;
  if (form.type === "help_contact") return formContent.helpContact;
  return {};
});
const previewSiteName = "Cloud Service";
const previewLogoUrl = "";
const homeFeatureIconMap: Record<string, any> = {
  thunder: ThunderIcon,
  shield: ShieldIcon,
  globe: GlobeIcon,
  server: ServerIcon,
  database: DatabaseIcon,
  settings: SettingsIcon,
};
const homeProductIconMap: Record<string, any> = {
  cloud: HomeCloudIcon,
  cube: CubeIcon,
  code: CodeIcon,
};
const productsIconMap: Record<string, any> = {
  cloud: CloudIcon,
  rocket: RocketIcon,
  bolt: BoltIcon,
  building: BuildingIcon,
};

// Help preview uses a standalone search query so the hero search can drive FAQ filtering.
const previewHelpSearchQuery = ref("");
const previewHelpHeroContent = computed(() =>
  form.type === "help_hero" ? activeContent.value : formContent.helpHero,
);
const previewHelpActionsContent = computed(() =>
  form.type === "help_actions" ? activeContent.value : formContent.helpActions,
);
const previewHelpFaqContent = computed(() =>
  form.type === "help_faq" ? activeContent.value : formContent.helpFaq,
);
const previewHelpContactContent = computed(() =>
  form.type === "help_contact" ? activeContent.value : formContent.helpContact,
);
const previewHomeHeroContent = computed(() => ({
  badge: activeContent.value.badge || "",
  title1: activeContent.value.title1 || "",
  subtitle: activeContent.value.subtitle || "",
  primary_button_text: activeContent.value.primary_button_text || "",
  primary_button_link: activeContent.value.primary_button_link || "/register",
  secondary_button_text: activeContent.value.secondary_button_text || "",
  secondary_button_link:
    activeContent.value.secondary_button_link || "/products",
}));
const previewHomeTypewriterText = computed(() => {
  const words = Array.isArray(activeContent.value.typewriter_words)
    ? activeContent.value.typewriter_words
    : [];
  return words[0] || "";
});
const previewHomeStats = computed(() =>
  Array.isArray(activeContent.value.stats) ? activeContent.value.stats : [],
);
const previewHomeAnimatedStats = computed(() =>
  previewHomeStats.value.map((stat: any) => {
    const value = stat?.value ?? "";
    return typeof value === "number" ? value.toString() : String(value);
  }),
);
const previewHomeCards = computed(() => {
  const cards = Array.isArray(activeContent.value.cards)
    ? activeContent.value.cards
    : [];
  if (isHeroCardList(cards)) return cards;
  return defaultHeroCards;
});
const previewHomeFeaturesContent = computed(() => ({
  badge: activeContent.value.badge || "",
  title: activeContent.value.title || "",
  desc: activeContent.value.desc || "",
}));
const previewHomeFeatures = computed(() => {
  const items = Array.isArray(activeContent.value.items)
    ? activeContent.value.items
    : [];
  return items.map((item: any, index: number) => ({
    icon:
      homeFeatureIconMap[item.icon] ||
      Object.values(homeFeatureIconMap)[index] ||
      ThunderIcon,
    title: item.title || "",
    description: item.description || "",
  }));
});
const previewHomeProductsContent = computed(() => ({
  badge: activeContent.value.badge || "",
  title: activeContent.value.title || "",
}));
const previewHomeProducts = computed(() => {
  const items = Array.isArray(activeContent.value.items)
    ? activeContent.value.items
    : [];
  return items.map((item: any, index: number) => ({
    icon:
      homeProductIconMap[item.icon] ||
      Object.values(homeProductIconMap)[index] ||
      HomeCloudIcon,
    tag: item.tag || "",
    title: item.title || "",
    description: item.description || "",
    price: item.price || "",
  }));
});
const previewHomeCtaContent = computed(() => ({
  title: activeContent.value.title || "",
  desc: activeContent.value.desc || "",
  button_text: activeContent.value.button_text || "",
  button_link: activeContent.value.button_link || "/register",
}));
const previewHomeCtaFeatures = computed(() =>
  Array.isArray(activeContent.value.features)
    ? activeContent.value.features
    : [],
);
const previewProductsHeroFeatures = computed(() => {
  const list = Array.isArray(formContent.productsHero.features)
    ? formContent.productsHero.features
    : [];
  if (list.length > 0) return list;
  return Array.isArray(defaultContent.productsHero.features)
    ? defaultContent.productsHero.features
    : [];
});
const previewProductsHeroContent = computed(() => ({
  badge: formContent.productsHero.badge || "",
  title: formContent.productsHero.title || "",
  subtitle: formContent.productsHero.subtitle || "",
  features: previewProductsHeroFeatures.value,
}));
const previewProductsCalculatorContent = computed(() => ({
  title: formContent.productsCalculator.title || "",
  desc: formContent.productsCalculator.desc || "",
}));
const previewProductsScenarios = computed(() =>
  Array.isArray(formContent.productsCalculator.scenarios)
    ? formContent.productsCalculator.scenarios
    : [],
);
const previewSelectedScenario = ref<number | null>(0);
const previewSelectScenario = (index: number) => {
  previewSelectedScenario.value = index;
};
watch(
  previewProductsScenarios,
  (list) => {
    if (!list.length) {
      previewSelectedScenario.value = null;
      return;
    }
    if (
      previewSelectedScenario.value === null ||
      previewSelectedScenario.value >= list.length
    ) {
      previewSelectedScenario.value = 0;
    }
  },
  { immediate: true },
);
const previewProductsPricing = computed(() => {
  const items = Array.isArray(formContent.productsPricing.products)
    ? formContent.productsPricing.products
    : [];
  return items.map((item: any, index: number) => ({
    icon:
      productsIconMap[item.icon] ||
      Object.values(productsIconMap)[index] ||
      CloudIcon,
    name: item.name || "",
    description: item.description || "",
    price: item.price || "",
    recommended: !!item.recommended,
    cta: item.cta || "",
    resources: Array.isArray(item.resources) ? item.resources : [],
    features: Array.isArray(item.features) ? item.features : [],
  }));
});
const previewSelectedPlan = ref(0);
const previewSelectPlan = (index: number) => {
  previewSelectedPlan.value = index;
};
const previewHoverPlan = (_index: number) => {};
const previewProductsComparisonContent = computed(() => ({
  title: formContent.productsComparison.title || "",
}));
const previewProductsComparisonRows = computed(() =>
  Array.isArray(formContent.productsComparison.rows)
    ? formContent.productsComparison.rows
    : [],
);
const previewProductsComparisonProducts = computed(() => {
  const items = Array.isArray(formContent.productsPricing.products)
    ? formContent.productsPricing.products
    : [];
  return items.map((item: any) => ({ name: item.name || "" }));
});
const previewProductsCtaContent = computed(() => ({
  title: formContent.productsCta.title || "",
  desc: formContent.productsCta.desc || "",
  contact_text: formContent.productsCta.contact_text || "",
  contact_link: formContent.productsCta.contact_link || "/console/tickets",
  email: formContent.productsCta.email || "",
}));
const previewFooterContent = computed(() => ({
  description: activeContent.value.description || "",
  social_links: Array.isArray(activeContent.value.social_links)
    ? activeContent.value.social_links
    : [],
}));
const previewFooterSections = computed(() =>
  Array.isArray(activeContent.value.sections)
    ? activeContent.value.sections
    : [],
);
const previewFooterBadges = computed(() =>
  Array.isArray(activeContent.value.badges) ? activeContent.value.badges : [],
);
const defaultBlocksByPage: Record<string, Record<string, any>> = {
  home: {
    hero: defaultContent.hero,
    features: defaultContent.features,
    products: defaultContent.products,
    cta: defaultContent.cta,
  },
  products: {
    hero: defaultContent.productsHero,
    calculator: defaultContent.productsCalculator,
    pricing: defaultContent.productsPricing,
    comparison: defaultContent.productsComparison,
    cta: defaultContent.productsCta,
  },
  docs: {
    hero: defaultContent.docsHero,
    posts: {},
    resources: defaultContent.docsResources,
  },
  announcements: {
    hero: defaultContent.docsHero,
    posts: {},
    resources: defaultContent.docsResources,
  },
  activities: {
    hero: defaultContent.docsHero,
    posts: {},
    resources: defaultContent.docsResources,
  },
  tutorials: {
    hero: defaultContent.docsHero,
    posts: {},
    resources: defaultContent.docsResources,
  },
  help: {
    help_hero: defaultContent.helpHero,
    help_actions: defaultContent.helpActions,
    help_faq: defaultContent.helpFaq,
    help_contact: defaultContent.helpContact,
  },
  footer: { footer: defaultContent.footer },
};
const ensureDefaultBlocks = async () => {
  try {
    const pages = Object.keys(defaultBlocksByPage);
    for (const pageKey of pages) {
      const res = await listCmsBlocks({
        page: pageKey,
        lang: "zh-CN",
        limit: 200,
        offset: 0,
      });
      const existing = new Set(
        (res.data?.items || []).map((item: any) => item.type),
      );
      const defaults = defaultBlocksByPage[pageKey];
      for (const [type, content] of Object.entries(defaults)) {
        if (existing.has(type)) continue;
        await createCmsBlock({
          page: pageKey,
          type,
          title: `${pageKey}-${type}`,
          subtitle: "",
          content_json: JSON.stringify(content),
          custom_html: "",
          lang: "zh-CN",
          visible: true,
          sort_order: 0,
        });
      }
    }
  } catch (error) {
    /* Defaults are a convenience; ignore failures. */
  }
};
const columns = [
  { title: "ID", dataIndex: "id", key: "id", width: 80 },
  { title: "页面", dataIndex: "page", key: "page", width: 100 },
  { title: "类型", dataIndex: "type", key: "type", width: 120 },
  { title: "标题", dataIndex: "title", key: "title", ellipsis: true },
  { title: "语言", dataIndex: "lang", key: "lang", width: 80 },
  { title: "排序", dataIndex: "sort_order", key: "sort_order", width: 80 },
  { title: "可见", dataIndex: "visible", key: "visible", width: 80 },
  { title: "操作", key: "actions", width: 150, fixed: "right" },
];
const fetchBlocks = async () => {
  loading.value = true;
  try {
    const params: any = {
      limit: pagination.pageSize,
      offset: (pagination.current - 1) * pagination.pageSize,
    };
    if (filters.page) params.page = filters.page;
    if (filters.type) params.type = filters.type;
    const res = await listCmsBlocks(params);
    blocks.value = res.data?.items || [];
    pagination.total = res.data?.total || 0;
  } finally {
    loading.value = false;
  }
};
const handleTableChange = (pag: any) => {
  pagination.current = pag.current;
  fetchBlocks();
};
const handleFilterChange = () => {
  pagination.current = 1;
  fetchBlocks();
};
const handleToggle = async (record: any, checked: boolean) => {
  try {
    await updateCmsBlock(record.id, { visible: checked });
    record.visible = checked;
    message.success("操作成功");
  } catch (error: any) {
    message.error(error.response?.data?.error || "操作失败");
  }
};
const openCreateModal = () => {
  isEditing.value = false;
  Object.assign(form, {
    id: undefined,
    page: "home",
    type: "hero",
    title: "",
    subtitle: "",
    content_json: "",
    custom_html: "",
    lang: "zh-CN",
    visible: true,
    sort_order: 0,
  });
  resetContent();
  modalVisible.value = true;
};
const openEditModal = (record: any) => {
  isEditing.value = true;
  Object.assign(form, record);
  if (structuredTypes.includes(record.type)) {
    resetContent();
    const parsed = safeParse(record.content_json || "");
    applyContent(record.type, parsed);
  }
  if (record.page === "home" && record.type === "hero") {
    const statsBlock = blocks.value.find((item) => item?.type === "stats");
    if (statsBlock) {
      const statsItems = resolveHeroStatsFromBlock(statsBlock);
      if (statsItems.length) {
        formContent.hero.stats = statsItems;
      }
    }
  }
  modalVisible.value = true;
};
const handleSubmit = async () => {
  if (!form.title) {
    message.error("请填写标题");
    return;
  }
  submitting.value = true;
  try {
    const payload = {
      page: form.page,
      type: form.type,
      title: form.title,
      subtitle: form.subtitle,
      content_json: JSON.stringify(buildContentJson(form.type)),
      custom_html: form.custom_html,
      lang: form.lang,
      visible: form.visible,
      sort_order: form.sort_order,
    };
    if (isEditing.value && form.id) {
      await updateCmsBlock(form.id, payload);
    } else {
      await createCmsBlock(payload);
    }
    message.success(isEditing.value ? "更新成功" : "创建成功");
    modalVisible.value = false;
    fetchBlocks();
  } catch (error: any) {
    message.error(error.response?.data?.error || "操作失败");
  } finally {
    submitting.value = false;
  }
};

const handleDelete = async (record: any) => {
  try {
    await deleteCmsBlock(record.id);
    message.success("删除成功");
    fetchBlocks();
  } catch (error: any) {
    message.error(error.response?.data?.error || "删除失败");
  }
};
onMounted(async () => {
  await ensureDefaultBlocks();
  fetchBlocks();
  if (!previewResizeObserver) {
    previewResizeObserver = new ResizeObserver(updatePreviewScale);
  }
  if (previewBodyRef.value) {
    previewResizeObserver.observe(previewBodyRef.value);
  }
});
onBeforeUnmount(() => {
  previewResizeObserver?.disconnect();
  previewResizeObserver = null;
});
</script>
<style scoped>
.cms-blocks-page {
  padding: 24px;
}
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}
.page-title {
  font-size: 20px;
  font-weight: 600;
  margin: 0;
}
.form-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.form-list-item {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  padding: 4px 8px;
  border-radius: 8px;
  border: 1px solid rgba(0, 0, 0, 0.08);
  background: #fff;
}
.form-list-item :deep(.ant-input),
.form-list-item :deep(.ant-input-number),
.form-list-item :deep(.ant-select) {
  min-width: 120px;
  flex: 1 1 160px;
}
.form-list-item :deep(.ant-btn) {
  margin-left: auto;
}
.form-list-item.column-group {
  align-items: flex-start;
  border: 1px solid rgba(0, 0, 0, 0.08);
  border-radius: 12px;
  padding: 12px;
  background: #fafafa;
}
.nested-list {
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-top: 8px;
}
.editor-layout {
  height: 640px;
  min-height: 640px;
}
.editor-toolbar {
  display: flex;
  justify-content: flex-end;
  margin-bottom: 12px;
}
.editor-form :deep(.ant-form-item) {
  margin-bottom: 12px;
}
.editor-form :deep(.ant-form-item-label > label) {
  height: 24px;
}
.editor-form .form-list {
  gap: 8px;
}
.editor-form .form-list-item {
  gap: 6px;
}
.editor-form {
  height: 100%;
  overflow: auto;
  padding-right: 6px;
}
.editor-form :deep(.ant-form) {
  padding-bottom: 24px;
}
.editor-preview {
  height: 100%;
}
.preview-panel {
  border: 1px solid rgba(0, 0, 0, 0.06);
  border-radius: 8px;
  height: 100%;
  display: flex;
  flex-direction: column;
  background: #fafafa;
}
.preview-header {
  padding: 12px 16px;
  font-weight: 600;
  border-bottom: 1px solid rgba(0, 0, 0, 0.06);
  background: #fff;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.preview-controls {
  display: flex;
  align-items: center;
  gap: 12px;
}
.preview-body {
  padding: 16px;
  overflow: auto;
  height: 100%;
  position: relative;
  display: flex;
  align-items: flex-start;
  justify-content: center;
  background: #f0f2f5;
}
.preview-viewport {
  display: flex;
  justify-content: center;
  align-items: flex-start;
  background: #f0f2f5;
}
.preview-canvas {
  width: 100%;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.12);
  overflow: auto;
}
.preview-canvas.is-zoom {
  width: 1280px;
  height: 720px;
}

/* 预览内容包装器 */
.preview-content-wrapper {
  width: 100%;
  background: white;
  min-height: 100%;
}
.preview-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.preview-item {
  padding: 8px 12px;
  border: 1px solid rgba(0, 0, 0, 0.06);
  border-radius: 6px;
  background: #fff;
}
.preview-hero h3 {
  margin: 8px 0;
}
.preview-badge {
  display: inline-block;
  padding: 2px 8px;
  background: rgba(24, 144, 255, 0.1);
  color: #1677ff;
  border-radius: 12px;
  font-size: 12px;
}
.preview-actions {
  display: flex;
  gap: 8px;
  margin-top: 8px;
  flex-wrap: wrap;
}
.preview-json {
  background: #fff;
  padding: 16px;
  border-radius: 6px;
  border: 1px solid rgba(0, 0, 0, 0.06);
  font-size: 12px;
  overflow-x: auto;
  white-space: pre-wrap;
}
.preview-html {
  background: #fff;
  padding: 12px;
  border-radius: 6px;
  border: 1px solid rgba(0, 0, 0, 0.06);
}
.preview-surface {
  min-height: auto;
  border-radius: 8px;
  overflow: hidden;
}
.preview-hero-surface {
  overflow: visible;
  padding-right: 140px;
  padding-bottom: 24px;
  position: relative;
}
.preview-surface.home-page,
.preview-surface.products-page,
.preview-surface.public-layout {
  min-height: auto;
}

/* In admin preview we don't run Home.vue's IntersectionObserver, so force scroll-animate visible. */
.preview-panel :deep(.scroll-animate) {
  opacity: 1 !important;
  transform: none !important;
  transition: none !important;
}
</style>
