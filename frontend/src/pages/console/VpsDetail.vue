<template>
  <div class="vps-detail-page">
    <!-- Unified Header Section -->
    <div class="detail-header">
      <div class="header-main">
        <div class="instance-info">
          <div class="instance-title">
            <DesktopOutlined class="title-icon" />
            <h1 class="title-text">{{ detail?.name || '加载中...' }}</h1>
            <VpsStatusTag :status="resolvedStatus || ''" class="title-status" />
          </div>
          <div class="instance-meta">
            <span class="meta-item">
              <span class="meta-label">ID</span>
              <span class="meta-value">{{ id }}</span>
            </span>
            <a-divider type="vertical" />
            <span class="meta-item">
              <DatabaseOutlined class="meta-icon" />
              <span class="meta-value">{{ specObj.cpu }}核</span>
            </span>
            <a-divider type="vertical" />
            <span class="meta-item">
              <ThunderboltOutlined class="meta-icon" />
              <span class="meta-value">{{ specObj.memory_gb }}GB</span>
            </span>
            <a-divider type="vertical" />
            <span class="meta-item">
              <CloudServerOutlined class="meta-icon" />
              <span class="meta-value">{{ specObj.disk_gb }}GB</span>
            </span>
            <a-divider type="vertical" />
            <span class="meta-item">
              <CloudUploadOutlined class="meta-icon" />
              <span class="meta-value">{{ specObj.bandwidth_mbps || '-' }}Mbps</span>
            </span>
          </div>
        </div>

        <div class="header-actions">
          <a-button @click="openPanel" type="primary">
            <template #icon><ApiOutlined /></template>
            控制面板
          </a-button>
          <a-button @click="openRenew">
            <template #icon><SyncOutlined /></template>
            续费
          </a-button>
          <a-button v-if="emergencyRenewEligible" @click="submitEmergencyRenew" danger>
            <template #icon><SyncOutlined /></template>
            紧急续费
          </a-button>
          <a-button @click="openVnc">
            <template #icon><CodeOutlined /></template>
            VNC
          </a-button>
          <a-button @click="refresh" :loading="loading">
            <template #icon><ReloadOutlined /></template>
          </a-button>
        </div>
      </div>
    </div>

    <!-- Main Tabs -->
    <div v-if="unsupportedFeatureHints.length" class="capability-notice">
      <a-alert type="info" show-icon>
        <template #message>部分功能按实例能力已自动隐藏</template>
        <template #description>
          <div class="capability-tags">
            <a-tag v-for="item in unsupportedFeatureHints" :key="item.key" color="default">
              {{ item.label }}：{{ item.reason }}
            </a-tag>
          </div>
        </template>
      </a-alert>
    </div>
    <a-tabs v-model:activeKey="activeTab" class="ecs-tabs">
      <a-tab-pane key="overview">
        <template #tab>
          <DashboardOutlined class="tab-icon" />
          <span>总览</span>
        </template>

        <div class="overview-grid">
          <!-- Instance Info Card -->
          <a-card class="overview-card instance-card" :bordered="false" :loading="loading">
            <template #title>
              <div class="card-title">
                <DesktopOutlined />
                <span>实例信息</span>
              </div>
            </template>

            <div class="instance-info-wrapper">
              <!-- Info List -->
              <div class="info-list">
                <div class="info-list-item">
                  <div class="info-list-icon">
                    <ThunderboltOutlined />
                  </div>
                  <div class="info-list-content">
                    <div class="info-list-label">实例状态</div>
                    <div class="info-list-value">
                      <VpsStatusTag :status="resolvedStatus || ''" />
                    </div>
                  </div>
                </div>
                <div class="info-list-item">
                  <div class="info-list-icon">
                    <GlobalOutlined />
                  </div>
                  <div class="info-list-content">
                    <div class="info-list-label">远程IP</div>
                    <div class="info-list-value">
                      <span>{{ access.remote_ip || '-' }}</span>
                      <a-button
                        v-if="access.remote_ip"
                        type="text"
                        size="small"
                        @click="copyRemoteIp"
                        title="复制IP"
                      >
                        <CopyOutlined />
                      </a-button>
                    </div>
                  </div>
                </div>
                <div class="info-list-item">
                  <div class="info-list-icon">
                    <HourglassOutlined />
                  </div>
                  <div class="info-list-content">
                    <div class="info-list-label">剩余天数</div>
                    <div class="info-list-value" :class="{ 'value-warning': remainingDays <= 7 }">
                      {{ remainingDays }}
                    </div>
                  </div>
                </div>
                <div class="info-list-item">
                  <div class="info-list-icon">
                    <KeyOutlined />
                  </div>
                  <div class="info-list-content">
                    <div class="info-list-label">系统密码</div>
                    <div class="info-list-value">
                      <span class="masked" v-if="!showPassword">••••••••</span>
                      <span class="password-text" v-else>{{ access.os_password || '-' }}</span>
                      <a-button type="text" size="small" @click="showPassword = !showPassword">
                        <EyeOutlined v-if="!showPassword" />
                        <EyeInvisibleOutlined v-else />
                      </a-button>
                      <a-button type="link" size="small" @click="openResetOsPassword">修改</a-button>
                    </div>
                  </div>
                </div>
                <div class="info-list-item">
                  <div class="info-list-icon">
                    <GlobalOutlined />
                  </div>
                  <div class="info-list-content">
                    <div class="info-list-label">区域</div>
                    <div class="info-list-value">{{ detail?.region || '-' }}</div>
                  </div>
                </div>
                <div class="info-list-item">
                  <div class="info-list-icon">
                    <CodeOutlined />
                  </div>
                  <div class="info-list-content">
                    <div class="info-list-label">操作系统</div>
                    <div class="info-list-value">
                      {{ systemLabel }}
                      <a-button type="link" size="small" @click="openReinstall">重装</a-button>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </a-card>

          <!-- Monitor Card -->
          <a-card class="overview-card monitor-card" :bordered="false" :loading="loading">
            <template #title>
              <div class="card-title">
                <LineChartOutlined />
                <span>实例监控</span>
              </div>
            </template>

            <div class="monitor-list">
              <div class="monitor-item">
                <div class="monitor-label">
                  <DatabaseOutlined class="monitor-icon" />
                  <span>CPU</span>
                </div>
                <div class="monitor-value-group">
                  <span class="monitor-value" :class="getCpuClass(currentCpu)">{{ currentCpu }}%</span>
                  <span class="monitor-spec">{{ specObj.cpu }}核</span>
                </div>
                <div class="monitor-bar">
                  <div class="monitor-bar-fill" :style="{ width: currentCpu + '%', background: getCpuColor(currentCpu) }"></div>
                </div>
              </div>
              <div class="monitor-item">
                <div class="monitor-label">
                  <ThunderboltOutlined class="monitor-icon" />
                  <span>内存</span>
                </div>
                <div class="monitor-value-group">
                  <span class="monitor-value" :class="getMemoryClass(currentMemory)">{{ currentMemory }}%</span>
                  <span class="monitor-spec">{{ specObj.memory_gb }}GB</span>
                </div>
                <div class="monitor-bar">
                  <div class="monitor-bar-fill" :style="{ width: currentMemory + '%', background: getMemoryColor(currentMemory) }"></div>
                </div>
              </div>
              <div class="monitor-item network-item">
                <div class="monitor-label">
                  <CloudUploadOutlined class="monitor-icon" />
                  <span>网络</span>
                </div>
                <div class="network-stats">
                  <div class="network-stat">
                    <span class="network-label-text">↓</span>
                    <span class="network-value-text">{{ currentTrafficIn }}</span>
                    <span class="network-unit-text">Mbps</span>
                  </div>
                  <div class="network-stat">
                    <span class="network-label-text">↑</span>
                    <span class="network-value-text">{{ currentTrafficOut }}</span>
                    <span class="network-unit-text">Mbps</span>
                  </div>
                </div>
              </div>
            </div>
          </a-card>

          <!-- Time & Price Card -->
          <a-card class="overview-card time-card" :bordered="false">
            <template #title>
              <div class="card-title">
                <CalendarOutlined />
                <span>时间与价格</span>
              </div>
            </template>

            <div class="time-price-body">
              <div class="info-grid">
              <div class="info-item">
                <span class="info-label">创建时间</span>
                <span class="info-value">{{ formatLocalTime(detail?.created_at) }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">到期时间</span>
                <span class="info-value" :class="{ 'value-warning': isExpiringSoon }">
                  {{ formatLocalTime(detail?.expire_at) }}
                </span>
              </div>
              <div class="info-item">
                <span class="info-label">剩余天数</span>
                <span class="info-value" :class="{ 'value-warning': remainingDays <= 7 }">
                  {{ remainingDays }}
                </span>
              </div>
              <div class="info-item">
                <span class="info-label">当前价格</span>
                <span class="info-value price-value">
                  ¥{{ Number(detail?.monthly_price || 0).toFixed(2) }}
                  <span class="price-unit">/月</span>
                </span>
              </div>
              </div>

              <a-divider style="margin: 16px 0;" />

              <div class="action-buttons">
                <a-button type="primary" size="large" @click="openRenew">
                  <SyncOutlined />
                  续费
                </a-button>
                <a-button v-if="emergencyRenewEligible" type="primary" size="large" danger @click="submitEmergencyRenew">
                  <SyncOutlined />
                  紧急续费
                </a-button>
              <a-button size="large" @click="openResize" v-if="resizeEnabled" :disabled="isExpired">
                <VerticalAlignTopOutlined />
                升降配
              </a-button>
              <a-button v-if="refundEnabled" size="large" danger @click="openRefund">
                <DeleteOutlined />
                退款
              </a-button>
              </div>
            </div>
          </a-card>

          <!-- Account Card -->
          <a-card class="overview-card account-card" :bordered="false" :loading="loading">
            <template #title>
              <div class="card-title">
                <SafetyOutlined />
                <span>连接信息</span>
              </div>
            </template>

            <div class="account-rows">
              <div class="account-row-item">
                <div class="account-row-label">操作系统</div>
                <div class="account-row-value">{{ systemLabel }}</div>
              </div>
              <div class="account-row-item">
                <div class="account-row-label">远程地址</div>
                <div class="account-row-value">
                  {{ access.remote_ip || '-' }}
                  <a-button type="link" size="small" @click="copyText(access.remote_ip, '远程地址')">
                    <CopyOutlined />
                  </a-button>
                </div>
              </div>
              <div class="account-row-item">
                <div class="account-row-label">系统用户</div>
                <div class="account-row-value">{{ isWindowsOS ? 'Administrator' : 'root' }}</div>
              </div>
              <div class="account-row-item">
                <div class="account-row-label">系统密码</div>
                <div class="account-row-value">
                  <span class="password-text" :class="{ masked: !showOsPassword }">
                    {{ showOsPassword ? (access.os_password || '-') : '••••••••' }}
                  </span>
                  <a-button type="link" size="small" @click="showOsPassword = !showOsPassword">
                    <component :is="showOsPassword ? EyeInvisibleOutlined : EyeOutlined" />
                  </a-button>
                  <a-button type="link" size="small" @click="openResetOsPassword">修改</a-button>
                </div>
              </div>
              <div class="account-row-item">
                <div class="account-row-label">面板用户</div>
                <div class="account-row-value">{{ detail?.name || '-' }}</div>
              </div>
              <div class="account-row-item">
                <div class="account-row-label">面板密码</div>
                <div class="account-row-value">
                  <span class="password-text" :class="{ masked: !showPanelPassword }">
                    {{ showPanelPassword ? (access.panel_password || '-') : '••••••••' }}
                  </span>
                  <a-button type="link" size="small" @click="showPanelPassword = !showPanelPassword">
                    <component :is="showPanelPassword ? EyeInvisibleOutlined : EyeOutlined" />
                  </a-button>
                </div>
              </div>
            </div>
          </a-card>
        </div>
      </a-tab-pane>

      <a-tab-pane key="monitor">
        <template #tab>
          <span>实时监控</span>
        </template>

        <div class="monitor-layout">
          <a-card class="monitor-panel" :bordered="false" :loading="loading">
            <template #title>
              <div class="card-title">
                <SafetyOutlined />
                <span>系统表现</span>
              </div>
            </template>

            <div class="monitor-summary">
              <div class="summary-list">
                <div class="summary-row">
                  <span>实例状态</span>
                  <span class="summary-value">
                    <span class="status-dot" :class="statusClass"></span>
                    <VpsStatusTag :status="resolvedStatus || ''" />
                  </span>
                </div>
                <div class="summary-row">
                  <span>操作系统</span>
                  <span class="summary-value">{{ systemLabel }}</span>
                </div>
                <div class="summary-row">
                  <span>到期时间</span>
                  <span class="summary-value" :class="{ warning: isExpiringSoon }">
                    {{ detail?.expire_at || '-' }}
                  </span>
                </div>
              </div>
              <div class="gauge-wrap">
                <div class="gauge">
                  <div class="gauge-mask"></div>
                  <div class="gauge-needle" :style="{ transform: `rotate(${perfNeedleDeg}deg)` }"></div>
                </div>
                <div class="gauge-label" :class="perfLabelClass">{{ perfLabel }}</div>
                <div class="gauge-sub">系统表现</div>
              </div>
            </div>

          </a-card>

          <a-card class="monitor-panel" :bordered="false" :loading="loading">
            <template #title>
              <div class="card-title">
                <DashboardOutlined />
                <span>CPU</span>
              </div>
            </template>
            <LineChart :data="monitor.cpu" :color="'#1677ff'" height="180" />
          </a-card>

          <a-card class="monitor-panel" :bordered="false" :loading="loading">
            <template #title>
              <div class="card-title">
                <CloudUploadOutlined />
                <span>IO</span>
              </div>
            </template>
            <LineChart :data="monitor.trafficOut" :color="'#fa8c16'" height="160" />
          </a-card>

          <a-card class="monitor-panel" :bordered="false" :loading="loading">
            <template #title>
              <div class="card-title">
                <ApiOutlined />
                <span>网络</span>
              </div>
            </template>
            <LineChart :data="monitor.trafficIn" :color="'#52c41a'" height="160" />
          </a-card>

          <a-card class="monitor-panel" :bordered="false" :loading="loading">
            <template #title>
              <div class="card-title">
                <DatabaseOutlined />
                <span>内存</span>
              </div>
            </template>
            <LineChart :data="monitor.memory" :color="'#722ed1'" height="160" />
          </a-card>
        </div>
      </a-tab-pane><a-tab-pane v-if="showFirewallTab" key="firewall">
        <template #tab>
          <span>防火墙</span>
        </template>
        <a-card class="security-card" :bordered="false">
          <template #title>
            <div class="card-title">
              <SafetyOutlined />
              <span>防火墙</span>
            </div>
          </template>
          <div class="tab-actions">
            <a-button type="primary" size="small" @click="openFirewallModal">
              <PlusOutlined />
              添加规则
            </a-button>
          </div>
          <a-table
            :data-source="firewallRules"
            :columns="firewallColumns"
            row-key="id"
            size="small"
            :scroll="{ x: 900 }"
            :loading="firewallLoading"
            :pagination="false"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'action'">
                <a-button type="link" size="small" danger @click="removeFirewallRule(record)">删除</a-button>
              </template>
            </template>
          </a-table>
        </a-card>
      </a-tab-pane>

      <a-tab-pane v-if="showPortTab" key="port">
        <template #tab>
          <span>端口映射</span>
        </template>
        <a-card class="security-card" :bordered="false">
          <template #title>
            <div class="card-title">
              <ApiOutlined />
              <span>端口映射</span>
            </div>
          </template>
          <div class="tab-actions">
            <a-button type="primary" size="small" @click="openPortModal">
              <PlusOutlined />
              添加映射
            </a-button>
          </div>
          <a-table
            :data-source="portMappings"
            :columns="portColumns"
            row-key="id"
            size="small"
            :scroll="{ x: 900 }"
            :loading="portLoading"
            :pagination="false"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'external'">
                <span>{{ formatPortExternal(record) }}</span>
              </template>
              <template v-if="column.key === 'action'">
                <a-tag v-if="Number(record?.sys) === 2" color="blue">系统</a-tag>
                <a-button
                  v-else
                  type="link"
                  size="small"
                  danger
                  :disabled="isProtectedPortMapping(record)"
                  @click="removePortMapping(record)"
                >
                  删除
                </a-button>
              </template>
            </template>
          </a-table>
        </a-card>
      </a-tab-pane>

      <a-tab-pane v-if="showSnapshotTab" key="snapshot">
        <template #tab>
          <span>快照</span>
        </template>
        <a-card class="security-card" :bordered="false">
          <template #title>
            <div class="card-title">
              <CameraOutlined />
              <span>快照</span>
            </div>
          </template>
          <div class="tab-actions">
            <a-button type="primary" size="small" @click="createSnapshot">
              <PlusOutlined />
              创建快照
            </a-button>
          </div>
          <a-table
            :data-source="snapshots"
            :columns="snapshotColumns"
            row-key="id"
            size="small"
            :scroll="{ x: 900 }"
            :loading="snapshotLoading"
            :pagination="false"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'state_label'">
                <a-badge :status="record.state_badge || 'default'" :text="record.state_label || '未知'" />
              </template>
              <template v-if="column.key === 'action'">
                <a-button type="link" size="small" @click="restoreSnapshot(record)">恢复</a-button>
                <a-button type="link" size="small" danger @click="deleteSnapshot(record)">删除</a-button>
              </template>
            </template>
          </a-table>
        </a-card>
      </a-tab-pane>

      <a-tab-pane v-if="showBackupTab" key="backup">
        <template #tab>
          <span>备份</span>
        </template>
        <a-card class="security-card" :bordered="false">
          <template #title>
            <div class="card-title">
              <SaveOutlined />
              <span>备份</span>
            </div>
          </template>
          <div class="tab-actions">
            <a-button type="primary" size="small" @click="createBackup">
              <PlusOutlined />
              创建备份
            </a-button>
          </div>
          <a-table
            :data-source="backups"
            :columns="backupColumns"
            row-key="id"
            size="small"
            :scroll="{ x: 900 }"
            :loading="backupLoading"
            :pagination="false"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'state_label'">
                <a-badge :status="record.state_badge || 'default'" :text="record.state_label || '未知'" />
              </template>
              <template v-if="column.key === 'action'">
                <a-button type="link" size="small" @click="restoreBackup(record)">恢复</a-button>
                <a-button type="link" size="small" danger @click="deleteBackup(record)">删除</a-button>
              </template>
            </template>
          </a-table>
        </a-card>
      </a-tab-pane>
    </a-tabs>
<!-- Renew Modal -->
    <a-modal
      v-model:open="renewOpen"
      title="续费实例"
      :width="480"
      @ok="submitRenew"
      :confirm-loading="renewing"
    >
      <a-form layout="vertical">
        <a-form-item label="续费周期">
          <a-select v-model:value="renewForm.cycleId" placeholder="选择周期">
            <a-select-option v-for="cycle in billingCycles" :key="cycle.id" :value="cycle.id">
              {{ cycle.name }} ({{ cycle.months }}个月)
              <template v-if="cycle.multiplier > 1">
                <a-tag color="orange" size="small">{{ cycle.multiplier }}倍</a-tag>
              </template>
            </a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="购买数量">
          <a-input-number v-model:value="renewForm.cycleQty" :min="1" :max="24" style="width: 100%">
            <template #addonAfter>个周期</template>
          </a-input-number>
        </a-form-item>
        <a-alert
          type="info"
          show-icon
          :message="`续费时长：${renewMonths} 个月`"
          :description="`系统将按月费 ￥${Number(detail?.monthly_price || 0).toFixed(2)} × ${renewMonths} 个月计算续费金额`"
        />
      </a-form>
    </a-modal>

    <!-- Resize Modal -->
    <a-modal
      v-model:open="resizeOpen"
      title="升降配置"
      :width="640"
      :footer="null"
      @cancel="closeResize"
    >
      <a-form layout="vertical">
        <a-form-item label="当前套餐">
          <a-input :value="`${currentPackage?.name || detail?.package_id || '-'} (￥${Number(detail?.monthly_price || 0).toFixed(2)}/月)`" disabled />
        </a-form-item>

        <a-form-item label="目标套餐">
          <a-select v-model:value="resizeForm.target_package_id" placeholder="选择套餐" :disabled="!resizeEnabled" @change="onPackageChange">
              <a-select-option v-for="pkg in packageOptions" :key="pkg.id" :value="pkg.id" :disabled="isResizePackageDisabled(pkg)">
                {{ formatResizePackageLabel(pkg) }} (￥{{ Number(pkg.monthly_price || 0).toFixed(2) }}/月)
              </a-select-option>
            </a-select>
        </a-form-item>

        <a-divider orientation="left">附加项管理</a-divider>

        <a-form-item>
          <a-checkbox v-model:checked="resizeForm.reset_addons" @change="onResetAddonsChange">
            重置所有附加项到下限
          </a-checkbox>
        </a-form-item>

        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="CPU 核心">
              <a-input-number
                v-model:value="resizeForm.add_cores"
                :min="addonMin.add_cores"
                :max="addonMax.add_cores"
                :step="addonStep.add_cores"
                :disabled="resizeForm.reset_addons || addonDisabled.add_cores"
                style="width: 100%"
              >
                <template #addonAfter>核</template>
              </a-input-number>
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="内存">
              <a-input-number
                v-model:value="resizeForm.add_mem_gb"
                :min="addonMin.add_mem_gb"
                :max="addonMax.add_mem_gb"
                :step="addonStep.add_mem_gb"
                :disabled="resizeForm.reset_addons || addonDisabled.add_mem_gb"
                style="width: 100%"
              >
                <template #addonAfter>GB</template>
              </a-input-number>
            </a-form-item>
          </a-col>
        </a-row>

        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="系统盘">
              <a-input-number
                v-model:value="resizeForm.add_disk_gb"
                :min="addonMin.add_disk_gb"
                :max="addonMax.add_disk_gb"
                :step="addonStep.add_disk_gb"
                :disabled="resizeForm.reset_addons || addonDisabled.add_disk_gb"
                style="width: 100%"
              >
                <template #addonAfter>GB</template>
              </a-input-number>
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="带宽">
              <a-input-number
                v-model:value="resizeForm.add_bw_mbps"
                :min="addonMin.add_bw_mbps"
                :max="addonMax.add_bw_mbps"
                :step="addonStep.add_bw_mbps"
                :disabled="resizeForm.reset_addons || addonDisabled.add_bw_mbps"
                style="width: 100%"
              >
                <template #addonAfter>Mbps</template>
              </a-input-number>
            </a-form-item>
          </a-col>
        </a-row>

        <a-divider orientation="left">执行时间</a-divider>

        <a-form-item label="执行方式">
          <a-radio-group v-model:value="resizeForm.schedule_mode" button-style="solid" style="width: 100%">
            <a-radio-button value="now" style="width: 50%; text-align: center">立即执行</a-radio-button>
            <a-radio-button value="scheduled" style="width: 50%; text-align: center">定时执行</a-radio-button>
          </a-radio-group>
        </a-form-item>

        <a-form-item v-if="resizeForm.schedule_mode === 'scheduled'" label="执行时间">
          <a-date-picker
            v-model:value="resizeForm.scheduled_at"
            show-time
            format="YYYY-MM-DD HH:mm:ss"
            :disabled-date="disabledDate"
            :disabled-time="disabledTime"
            style="width: 100%"
            placeholder="选择执行时间"
          />
        </a-form-item>

        <a-alert
          v-if="resizeQuote"
          type="success"
          show-icon
          style="margin-bottom: 16px"
        >
          <template #message>
            本周期需支付：￥{{ resizeQuoteAmount.toFixed(2) }}
          </template>
        </a-alert>
        <a-alert
          v-else-if="resizeQuoteError"
          type="error"
          show-icon
          style="margin-bottom: 16px"
          :message="resizeQuoteError"
        />

        <a-form-item>
          <a-button type="primary" block @click="submitResize" :loading="resizing" :disabled="!resizeEnabled || isExpired || isSameTargetSelection || !resizeForm.target_package_id">
            提交升降配
          </a-button>
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- Reinstall Modal -->
    <a-modal
      v-model:open="reinstallOpen"
      title="重装系统"
      :width="480"
      :confirm-loading="reinstalling"
      @ok="submitReinstall"
    >
      <a-alert
        message="重装提示"
        description="重装将清空当前系统盘数据，请确保已备份重要数据。"
        type="warning"
        show-icon
        style="margin-bottom: 16px"
      />
      <a-form layout="vertical">
        <a-form-item label="系统镜像" required>
          <a-select v-model:value="reinstallForm.template_id" placeholder="选择系统镜像">
            <a-select-option v-for="img in systemImageOptions" :key="img.id" :value="img.id">
              {{ img.name }}<span v-if="img.type"> ({{ img.type }})</span>
            </a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="新系统密码" required>
          <a-space style="width: 100%">
            <a-input-password
              v-model:value="reinstallForm.password"
              placeholder="请输入新系统密码"
              style="flex: 1"
              :maxlength="INPUT_LIMITS.PASSWORD"
            />
            <a-button :icon="h(ReloadOutlined)" @click="generateReinstallPassword">随机</a-button>
          </a-space>
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- Reset Password Modal -->
    <a-modal
      v-model:open="resetOsPasswordOpen"
      title="重置面板密码"
      :width="480"
      :confirm-loading="resetOsPasswordLoading"
      @ok="submitResetOsPassword"
    >
      <a-form layout="vertical">
        <a-form-item label="新面板密码" required>
          <a-space style="width: 100%">
            <a-input-password
              v-model:value="resetOsPasswordForm.password"
              placeholder="请输入新面板密码"
              style="flex: 1"
              :maxlength="INPUT_LIMITS.PASSWORD"
            />
            <a-button :icon="h(ReloadOutlined)" @click="generateOsPassword">随机</a-button>
          </a-space>
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- Firewall Modal -->
    <a-modal
      v-model:open="firewallOpen"
      title="添加防火墙规则"
      :width="480"
      @ok="submitFirewallRule"
    >
      <a-form layout="vertical">
        <a-form-item label="方向" required>
          <a-select v-model:value="firewallForm.direction">
            <a-select-option value="In">入站</a-select-option>
            <a-select-option value="Out">出站</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="协议" required>
          <a-select v-model:value="firewallForm.protocol">
            <a-select-option value="tcp">TCP</a-select-option>
            <a-select-option value="udp">UDP</a-select-option>
            <a-select-option value="all">全部</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="动作" required>
          <a-select v-model:value="firewallForm.method">
            <a-select-option value="allowed">允许</a-select-option>
            <a-select-option value="denied">拒绝</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="端口" required>
          <a-input v-model:value="firewallForm.port" placeholder="例如: 22 或 80-90" />
        </a-form-item>
        <a-form-item label="IP 地址" required>
          <a-input v-model:value="firewallForm.ip" placeholder="0.0.0.0" />
        </a-form-item>
        <a-form-item label="优先级">
          <a-input-number v-model:value="firewallForm.priority" :min="1" :max="65535" style="width: 100%" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- Port Mapping Modal -->
    <a-modal
      v-model:open="portOpen"
      title="添加端口映射"
      :width="480"
      :footer="null"
    >
      <a-form layout="vertical">
        <a-form-item label="名称">
          <a-input v-model:value="portForm.name" placeholder="可选" :maxlength="INPUT_LIMITS.PORT_MAPPING_NAME" />
        </a-form-item>
        <a-form-item label="外部端口">
          <a-input
            v-model:value="portForm.sport"
            placeholder="留空自动分配"
            @input="onPortSportInput"
          />
          <div v-if="portCandidates.length" class="port-candidates">
            <span class="port-candidates-label">可用端口：</span>
            <a-space wrap>
              <a-tag
                v-for="item in portCandidates"
                :key="item"
                @click="selectPortCandidate(item)"
              >
                {{ item }}
              </a-tag>
            </a-space>
          </div>
        </a-form-item>
        <a-form-item label="内部端口" required>
          <a-input v-model:value="portForm.dport" placeholder="VPS 内部端口" />
        </a-form-item>
      </a-form>
      <div class="modal-actions">
        <a-space>
          <a-button @click="portOpen = false">取消</a-button>
          <a-button type="primary" :loading="portLoading" @click="submitPortMapping">保存</a-button>
        </a-space>
      </div>
    </a-modal>

    <!-- Refund Modal -->
    <a-modal
      v-model:open="refundOpen"
      title="申请退款"
      :width="480"
      :confirm-loading="refunding"
      @ok="submitRefund"
    >
      <a-alert
        message="退款须知"
        description="退款申请提交后，系统将进行审核。审核通过后，实例将被释放，数据将无法恢复。请谨慎操作。"
        type="warning"
        show-icon
        style="margin-bottom: 16px"
      />
      <a-form layout="vertical">
        <a-form-item label="退款原因" required>
          <a-textarea
            v-model:value="refundReason"
            rows="4"
            placeholder="请详细描述退款原因，有助于我们改进服务"
            :maxlength="INPUT_LIMITS.REFUND_REASON"
            show-count
          />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup>
import { computed, onMounted, onBeforeUnmount, reactive, ref, h, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { useVpsStore } from "@/stores/vps";
import { useCatalogStore } from "@/stores/catalog";
import { useAuthStore } from "@/stores/auth";
import { useSiteStore } from "@/stores/site";
import { INPUT_LIMITS } from "@/constants/inputLimits";
import {
  createVpsRenewOrder,
  emergencyRenewVps,
  createVpsResizeOrder,
  quoteVpsResizeOrder,
  getVpsMonitor,
  rebootVps,
  startVps,
  shutdownVps,
  requestVpsRefund,
  listSystemImages,
  resetVpsOS,
  resetVpsOsPassword,
  getVpsSnapshots,
  createVpsSnapshot,
  deleteVpsSnapshot,
  restoreVpsSnapshot,
  getVpsBackups,
  createVpsBackup,
  deleteVpsBackup,
  restoreVpsBackup,
  getVpsFirewallRules,
  addVpsFirewallRule,
  deleteVpsFirewallRule,
  getVpsPortMappings,
  getVpsPortCandidates,
  addVpsPortMapping,
  deleteVpsPortMapping
} from "@/services/user";
import { message, Modal } from "ant-design-vue";
import LineChart from "@/components/Charts/LineChart.vue";
import VpsStatusTag from "@/components/VpsStatusTag.vue";
import dayjs from "dayjs";

// Icons
import {
  DesktopOutlined,
  ControlOutlined,
  DownOutlined,
  ApiOutlined,
  CodeOutlined,
  ReloadOutlined,
  CalendarOutlined,
  CloudServerOutlined,
  GlobalOutlined,
  DatabaseOutlined,
  PlayCircleOutlined,
  PauseCircleOutlined,
  CloseCircleOutlined,
  PoweroffOutlined,
  SyncOutlined,
  VerticalAlignTopOutlined,
  DownloadOutlined,
  KeyOutlined,
  DeleteOutlined,
  SafetyOutlined,
  CopyOutlined,
  EyeOutlined,
  EyeInvisibleOutlined,
  LineChartOutlined,
  DashboardOutlined,
  CloudUploadOutlined,
  PlusOutlined,
  CameraOutlined,
  SaveOutlined,
  InfoCircleOutlined,
  ThunderboltOutlined,
  ClockCircleOutlined,
  HourglassOutlined
} from "@ant-design/icons-vue";

const route = useRoute();
const router = useRouter();
const store = useVpsStore();
const catalog = useCatalogStore();
const auth = useAuthStore();
const site = useSiteStore();
const id = route.params.id;

const loading = ref(true);
const activeTab = ref("overview");
const showOsPassword = ref(false);
const showPanelPassword = ref(false);
const showPassword = ref(false);
const refundOpen = ref(false);
const refundReason = ref("");
const refunding = ref(false);
const reinstallOpen = ref(false);
const reinstalling = ref(false);
const reinstallImages = ref([]);
const reinstallForm = reactive({
  template_id: null,
  password: ""
});
const resetOsPasswordOpen = ref(false);
const resetOsPasswordLoading = ref(false);
const resetOsPasswordForm = reactive({
  password: ""
});

const firewallRules = ref([]);
const firewallLoading = ref(false);
const firewallOpen = ref(false);
const firewallForm = reactive({
  direction: "In",
  protocol: "tcp",
  method: "allowed",
  port: "",
  ip: "",
  priority: 100
});

const portMappings = ref([]);
const portLoading = ref(false);
const portOpen = ref(false);
const portForm = reactive({
  name: "",
  sport: "",
  dport: ""
});
const portCandidates = ref([]);
const portCandidatesLoading = ref(false);
let portCandidateTimer;

const snapshots = ref([]);
const snapshotLoading = ref(false);
const backups = ref([]);
const backupLoading = ref(false);
const renewOpen = ref(false);
const renewing = ref(false);
const renewForm = reactive({ cycleId: null, cycleQty: 1 });
const resizeOpen = ref(false);
const resizing = ref(false);
const resizeQuote = ref(null);
const resizeQuoteLoading = ref(false);
const resizeQuoteError = ref("");
const resizeForm = reactive({
  add_cores: 0,
  add_mem_gb: 0,
  add_disk_gb: 0,
  add_bw_mbps: 0,
  target_package_id: null,
  reset_addons: false,
  schedule_mode: "now",
  scheduled_at: null
});

const detail = computed(() => {
  if (!store.current) return null;
  return {
    id: store.current.id ?? store.current.ID,
    name: store.current.name ?? store.current.Name,
    status: store.current.status ?? store.current.Status,
    automation_state: store.current.automation_state ?? store.current.AutomationState ?? null,
    region: store.current.region ?? store.current.Region,
    expire_at: store.current.expire_at ?? store.current.ExpireAt,
    created_at: store.current.created_at ?? store.current.CreatedAt,
    spec: store.current.spec ?? store.current.Spec ?? store.current.spec_json ?? store.current.SpecJSON,
      access_info: store.current.access_info ?? store.current.AccessInfo ?? store.current.access_info_json ?? store.current.AccessInfoJSON,
      monthly_price: store.current.monthly_price ?? store.current.MonthlyPrice ?? 0,
      capabilities: store.current.capabilities ?? store.current.Capabilities ?? null,
      last_emergency_renew_at: store.current.last_emergency_renew_at ?? store.current.LastEmergencyRenewAt ?? null,
      system_id: store.current.system_id ?? store.current.SystemID ?? 0,
      package_id: store.current.package_id ?? store.current.PackageID ?? 0
    };
  });

const normalizeAutomationFeature = (value) => {
  const v = String(value || "").trim().toLowerCase();
  switch (v) {
    case "upgrade":
    case "downgrade":
      return "resize";
    case "refund_request":
      return "refund";
    default:
      return v;
  }
};

const automationFeatureSet = computed(() => {
  const raw = detail.value?.capabilities?.automation?.features;
  if (!Array.isArray(raw)) return null;
  return new Set(
    raw
      .map((item) => normalizeAutomationFeature(item))
      .filter(Boolean)
  );
});

const supportsAutomationFeature = (...features) => {
  if (!automationFeatureSet.value) return true;
  return features.some((feature) => automationFeatureSet.value.has(normalizeAutomationFeature(feature)));
};

const showFirewallTab = computed(() => supportsAutomationFeature("firewall"));
const showPortTab = computed(() => supportsAutomationFeature("port_mapping"));
const showSnapshotTab = computed(() => supportsAutomationFeature("snapshot"));
const showBackupTab = computed(() => supportsAutomationFeature("backup"));
const unsupportedFeatureHints = computed(() => {
  const rawFeatures = detail.value?.capabilities?.automation?.features;
  if (!Array.isArray(rawFeatures)) return [];
  const reasons = detail.value?.capabilities?.automation?.not_supported_reasons || {};
  const defs = [
    { key: "firewall", label: "防火墙", visible: showFirewallTab.value },
    { key: "port_mapping", label: "端口映射", visible: showPortTab.value },
    { key: "snapshot", label: "快照", visible: showSnapshotTab.value },
    { key: "backup", label: "备份", visible: showBackupTab.value },
    { key: "resize", label: "升降配", visible: supportsAutomationFeature("resize") },
    { key: "refund", label: "退款", visible: supportsAutomationFeature("refund") }
  ];
  return defs
    .filter((item) => !item.visible)
    .map((item) => ({
      key: item.key,
      label: item.label,
      reason: String(reasons[item.key] || "当前实例不支持该能力")
    }));
});

const availableTabs = computed(() => {
  const tabs = ["overview", "monitor"];
  if (showFirewallTab.value) tabs.push("firewall");
  if (showPortTab.value) tabs.push("port");
  if (showSnapshotTab.value) tabs.push("snapshot");
  if (showBackupTab.value) tabs.push("backup");
  return tabs;
});

const parseJson = (input) => {
  if (!input) return {};
  if (typeof input === "string") {
    try {
      return JSON.parse(input);
    } catch {
      return {};
    }
  }
  return input;
};

const specObj = computed(() => {
  const spec = detail.value?.spec;
  const fallback = detail.value || {};
  if (!spec) {
    return {
      cpu: fallback.cpu ?? fallback.CPU ?? 0,
      memory_gb: fallback.memory_gb ?? fallback.MemoryGB ?? 0,
      disk_gb: fallback.disk_gb ?? fallback.DiskGB ?? 0,
      bandwidth_mbps: fallback.bandwidth_mbps ?? fallback.BandwidthMB ?? null
    };
  }
  const obj = typeof spec === "string" ? parseJson(spec) : spec;
  return {
    cpu: obj.cpu ?? obj.cores ?? obj.CPU ?? obj.Cores ?? fallback.cpu ?? fallback.CPU ?? 0,
    memory_gb: obj.memory_gb ?? obj.mem_gb ?? obj.MemoryGB ?? fallback.memory_gb ?? fallback.MemoryGB ?? 0,
    disk_gb: obj.disk_gb ?? obj.DiskGB ?? fallback.disk_gb ?? fallback.DiskGB ?? 0,
    bandwidth_mbps:
      obj.bandwidth_mbps ?? obj.BandwidthMB ?? obj.bandwidth ?? fallback.bandwidth_mbps ?? fallback.BandwidthMB ?? null
  };
});

const systemLabel = computed(() => {
  const spec = parseJson(detail.value?.spec);
  const fromSpec = spec?.system_name || spec?.os_name || spec?.os || spec?.image_name || "";
  if (fromSpec) return String(fromSpec);
  const systemId = detail.value?.system_id;
  if (systemId) {
    const idText = String(systemId);
    const img =
      catalog.systemImages.find((item) => String(item.id) === idText) ||
      catalog.systemImages.find((item) => String(item.image_id) === idText);
    if (img?.name) return img.name;
    return `系统 ID ${systemId}`;
  }
  return "-";
});

const isWindowsOS = computed(() => String(systemLabel.value || "").toLowerCase().includes("windows"));

const currentAddons = computed(() => {
  const spec = parseJson(detail.value?.spec);
  return {
    add_cores: Number(spec?.add_cores ?? spec?.AddCores ?? 0),
    add_mem_gb: Number(spec?.add_mem_gb ?? spec?.AddMemGB ?? 0),
    add_disk_gb: Number(spec?.add_disk_gb ?? spec?.AddDiskGB ?? 0),
    add_bw_mbps: Number(spec?.add_bw_mbps ?? spec?.AddBWMbps ?? 0)
  };
});

const currentPackage = computed(() => {
  const pkgId = detail.value?.package_id;
  if (!pkgId) return null;
  return catalog.packages.find((pkg) => String(pkg.id) === String(pkgId)) || null;
});

const currentPlanGroup = computed(() => {
  if (!currentPackage.value) return null;
  const groupId = currentPackage.value.plan_group_id ?? currentPackage.value.planGroupId ?? currentPackage.value.PlanGroupID;
  return catalog.planGroups.find((g) => String(g.id) === String(groupId)) || null;
});

const packageOptions = computed(() => {
  if (!currentPlanGroup.value) return [];
  const groupId = currentPlanGroup.value.id ?? currentPlanGroup.value.ID;
  return catalog.packages
    .filter((pkg) => {
      const pkgGroup = pkg.plan_group_id ?? pkg.planGroupId ?? pkg.PlanGroupID;
      if (String(pkgGroup) !== String(groupId)) return false;
      if (pkg.active === false || pkg.visible === false) return false;
      return true;
    })
    .sort((a, b) => Number(a.monthly_price || 0) - Number(b.monthly_price || 0));
});

const resizeEnabled = computed(() => {
  if (site.settings?.resize_enabled === false) return false;
  return supportsAutomationFeature("resize", "upgrade", "downgrade");
});
const refundEnabled = computed(() => supportsAutomationFeature("refund", "refund_request"));
const currentDiskGB = computed(() => Number(specObj.value.disk_gb || 0));

const getResizeTargetPackage = () => {
  if (!resizeForm.target_package_id) return currentPackage.value;
  return packageOptions.value.find((pkg) => String(pkg.id) === String(resizeForm.target_package_id)) || currentPackage.value;
};

const formatResizePackageLabel = (pkg) => {
  const cpu = Number(pkg?.cores ?? pkg?.cpu ?? pkg?.CPU ?? pkg?.Cores ?? 0);
  const mem = Number(pkg?.memory_gb ?? pkg?.mem_gb ?? pkg?.MemoryGB ?? 0);
  return `${pkg?.name || "-"}（${cpu}核${mem}G）`;
};

const normalizeAddonRule = (minRaw, maxRaw, stepRaw, fallbackMax) => {
  const min = Number(minRaw ?? 0);
  const max = Number(maxRaw ?? 0);
  const step = Math.max(1, Number(stepRaw ?? 1));
  if (min === -1 || max === -1) {
    return { disabled: true, min: 0, max: 0, step: 1 };
  }
  const effectiveMin = min > 0 ? min : 0;
  const effectiveMax = max > 0 ? max : fallbackMax;
  return {
    disabled: false,
    min: effectiveMin,
    max: Math.max(effectiveMin, effectiveMax),
    step
  };
};

const buildResizeAddonRule = (targetPkg) => {
  const group = currentPlanGroup.value || {};
  const coreRule = normalizeAddonRule(group.add_core_min, group.add_core_max, group.add_core_step, 64);
  const memRule = normalizeAddonRule(group.add_mem_min, group.add_mem_max, group.add_mem_step, 256);
  const bwRule = normalizeAddonRule(group.add_bw_min, group.add_bw_max, group.add_bw_step, 1000);
  const diskRuleBase = normalizeAddonRule(group.add_disk_min, group.add_disk_max, group.add_disk_step, 2000);

  let diskMin = diskRuleBase.min;
  let diskImpossible = false;
  if (!diskRuleBase.disabled && targetPkg) {
    const pkgDisk = Number(targetPkg.disk_gb ?? targetPkg.DiskGB ?? 0);
    const required = Math.max(0, currentDiskGB.value - pkgDisk);
    diskMin = Math.max(diskMin, required);
  } else if (diskRuleBase.disabled && targetPkg) {
    const pkgDisk = Number(targetPkg.disk_gb ?? targetPkg.DiskGB ?? 0);
    if (pkgDisk < currentDiskGB.value) {
      diskImpossible = true;
    }
  }
  const diskRule = {
    disabled: diskRuleBase.disabled,
    min: diskRuleBase.disabled ? 0 : diskMin,
    max: diskRuleBase.max,
    step: diskRuleBase.step,
    impossible: diskImpossible
  };
  if (!diskRule.disabled && diskRule.min > diskRule.max) {
    diskRule.impossible = true;
    diskRule.max = diskRule.min;
  }
  return {
    add_cores: coreRule,
    add_mem_gb: memRule,
    add_disk_gb: diskRule,
    add_bw_mbps: bwRule
  };
};

const resizeAddonRule = computed(() => buildResizeAddonRule(getResizeTargetPackage()));

const isResizePackageDisabled = (pkg) => {
  if (!pkg) return true;
  const rule = buildResizeAddonRule(pkg);
  return rule.add_disk_gb.impossible;
};

const normalizePackageSpec = (pkg) => ({
  cpu: Number(pkg?.cores ?? pkg?.cpu ?? pkg?.CPU ?? pkg?.Cores ?? 0),
  memory_gb: Number(pkg?.memory_gb ?? pkg?.mem_gb ?? pkg?.MemoryGB ?? 0),
  disk_gb: Number(pkg?.disk_gb ?? pkg?.DiskGB ?? 0),
  bandwidth_mbps: Number(pkg?.bandwidth_mbps ?? pkg?.BandwidthMB ?? pkg?.bandwidth ?? 0)
});

const currentSpecForCompare = computed(() => {
  const fallback = {
    cpu: Number(specObj.value.cpu || 0),
    memory_gb: Number(specObj.value.memory_gb || 0),
    disk_gb: Number(specObj.value.disk_gb || 0),
    bandwidth_mbps: Number(specObj.value.bandwidth_mbps || 0)
  };
  if (!currentPackage.value) return fallback;
  const fromPackage = normalizePackageSpec(currentPackage.value);
  const hasSpec = Object.values(fromPackage).some((val) => Number(val) > 0);
  return hasSpec ? fromPackage : fallback;
});

const isSamePackageOption = (pkg) => {
  if (!pkg) return false;
  if (currentPackage.value?.id && String(pkg.id) === String(currentPackage.value.id)) return true;
  const pkgProduct = pkg.product_id ?? pkg.ProductID;
  const currentProduct = currentPackage.value?.product_id ?? currentPackage.value?.ProductID;
  if (pkgProduct && currentProduct && String(pkgProduct) === String(currentProduct)) return true;
  const pkgSpec = normalizePackageSpec(pkg);
  const currentSpec = currentSpecForCompare.value;
  return (
    pkgSpec.cpu === currentSpec.cpu &&
    pkgSpec.memory_gb === currentSpec.memory_gb &&
    pkgSpec.disk_gb === currentSpec.disk_gb &&
    pkgSpec.bandwidth_mbps === currentSpec.bandwidth_mbps
  );
};

  const isSameAddonsSelection = computed(() => {
    const current = currentAddons.value;
    return (
      Number(resizeForm.add_cores || 0) === Number(current.add_cores || 0) &&
      Number(resizeForm.add_mem_gb || 0) === Number(current.add_mem_gb || 0) &&
      Number(resizeForm.add_disk_gb || 0) === Number(current.add_disk_gb || 0) &&
      Number(resizeForm.add_bw_mbps || 0) === Number(current.add_bw_mbps || 0)
    );
  });

  const isSameTargetSelection = computed(() => {
    if (!resizeForm.target_package_id) return false;
    const target = packageOptions.value.find((pkg) => String(pkg.id) === String(resizeForm.target_package_id));
    return target ? isSamePackageOption(target) && isSameAddonsSelection.value : false;
  });

const resizeQuoteAmount = computed(() => {
  const raw = resizeQuote.value?.charge_amount ?? resizeQuote.value?.chargeAmount ?? 0;
  return Number(raw || 0);
});

const addonMin = computed(() => ({
  add_cores: resizeAddonRule.value.add_cores.min,
  add_mem_gb: resizeAddonRule.value.add_mem_gb.min,
  add_disk_gb: resizeAddonRule.value.add_disk_gb.min,
  add_bw_mbps: resizeAddonRule.value.add_bw_mbps.min
}));

const addonMax = computed(() => {
  return {
    add_cores: resizeAddonRule.value.add_cores.max,
    add_mem_gb: resizeAddonRule.value.add_mem_gb.max,
    add_disk_gb: resizeAddonRule.value.add_disk_gb.max,
    add_bw_mbps: resizeAddonRule.value.add_bw_mbps.max
  };
});

const addonStep = computed(() => {
  return {
    add_cores: resizeAddonRule.value.add_cores.step,
    add_mem_gb: resizeAddonRule.value.add_mem_gb.step,
    add_disk_gb: resizeAddonRule.value.add_disk_gb.step,
    add_bw_mbps: resizeAddonRule.value.add_bw_mbps.step
  };
});

const addonDisabled = computed(() => ({
  add_cores: resizeAddonRule.value.add_cores.disabled,
  add_mem_gb: resizeAddonRule.value.add_mem_gb.disabled,
  add_disk_gb: resizeAddonRule.value.add_disk_gb.disabled,
  add_bw_mbps: resizeAddonRule.value.add_bw_mbps.disabled
}));

const billingCycles = computed(() =>
  catalog.billingCycles.length
    ? catalog.billingCycles.filter((cycle) => cycle.active !== false)
    : [{ id: 1, name: "按月", months: 1, multiplier: 1 }]
);

const systemImageOptions = computed(() =>
  reinstallImages.value
    .map((item) => ({
      id: item.image_id ?? item.ImageID ?? null,
      name: item.name ?? item.Name ?? "",
      type: item.type ?? item.Type ?? ""
    }))
    .filter((item) => item.id)
);

const selectedRenewCycle = computed(() => billingCycles.value.find((c) => c.id === renewForm.cycleId));

const renewMonths = computed(() => {
  const months = Number(selectedRenewCycle.value?.months || 1);
  const qty = Number(renewForm.cycleQty || 1);
  return months * qty;
});

const access = computed(() => {
  const info = parseJson(detail.value?.access_info);
  return {
    remote_ip: info.remote_ip || info.ip || info.public_ip || info.ipv4 || info.Ip || "",
    remote_port: info.remote_port || info.port || info.ssh_port || info.Port || "",
    os_password: info.os_password || info.password || info.pass || info.Password || "",
    panel_password: info.panel_password || info.panelPassword || "",
    vnc_password: info.vnc_password || info.vnc || ""
  };
});

const hasInlinePort = computed(() => {
  const raw = String(access.value.remote_ip || "");
  return raw.includes(":") && !raw.startsWith("[");
});

const portForCommand = computed(() => {
  if (access.value.remote_port) return String(access.value.remote_port);
  if (hasInlinePort.value) return "";
  return isWindowsOS.value ? "3389" : "22";
});

const portForDisplay = computed(() => {
  if (access.value.remote_port) return String(access.value.remote_port);
  const host = String(access.value.remote_ip || "");
  if (hasInlinePort.value) {
    const parts = host.split(":");
    return parts[parts.length - 1] || "";
  }
  return isWindowsOS.value ? "3389" : "22";
});

const connectCommandLabel = computed(() => (isWindowsOS.value ? "RDP 连接命令" : "SSH 连接命令"));
const connectCommand = computed(() => {
  const host = access.value.remote_ip || "x.x.x.x";
  if (isWindowsOS.value) {
    const target = hasInlinePort.value ? host : (portForCommand.value ? `${host}:${portForCommand.value}` : host);
    return `mstsc.exe /v:${target}`;
  }
  if (hasInlinePort.value && !portForCommand.value) {
    return `ssh root@${host}`;
  }
  if (portForCommand.value) {
    return `ssh root@${host} -p ${portForCommand.value}`;
  }
  return `ssh root@${host}`;
});

const destroyInDays = computed(() => {
  const value = detail.value?.destroy_in_days;
  if (value === undefined || value === null) return null;
  return Number(value);
});

const isExpiringSoon = computed(() => {
  if (!detail.value?.expire_at) return false;
  const now = dayjs();
  const expire = dayjs(detail.value.expire_at);
  const days = expire.diff(now, 'day');
  return days <= 7 && days >= 0;
});

const isExpired = computed(() => {
  if (!detail.value?.expire_at) return false;
  const expire = dayjs(detail.value.expire_at);
  if (!expire.isValid()) return false;
  return !expire.isAfter(dayjs());
});

const emergencyRenewPolicy = computed(() => {
  const settings = site.settings || {};
  const enabledValue = settings.emergency_renew_enabled;
  const enabled =
    enabledValue === undefined ||
    enabledValue === null ||
    (typeof enabledValue === "string"
      ? enabledValue.toLowerCase() !== "false" && enabledValue !== "0"
      : enabledValue !== false);
  let windowDays = Number.parseInt(settings.emergency_renew_window_days ?? "7", 10);
  let intervalHours = Number.parseInt(settings.emergency_renew_interval_hours ?? "720", 10);
  if (!Number.isFinite(windowDays)) windowDays = 7;
  if (!Number.isFinite(intervalHours)) intervalHours = 720;
  if (windowDays < 0) windowDays = 0;
  if (intervalHours <= 0) intervalHours = 24;
  return {
    enabled,
    windowDays,
    intervalHours
  };
});

const emergencyRenewEligible = computed(() => {
  if (!emergencyRenewPolicy.value.enabled) return false;
  if (!detail.value?.expire_at) return false;
  const now = dayjs();
  const expire = dayjs(detail.value.expire_at);
  if (expire.isBefore(now)) return false;
  const windowDays = emergencyRenewPolicy.value.windowDays;
  if (windowDays > 0) {
    const windowStart = expire.subtract(windowDays, "day");
    if (now.isBefore(windowStart)) return false;
  }
  if (detail.value?.last_emergency_renew_at) {
    const lastAt = dayjs(detail.value.last_emergency_renew_at);
    const intervalHours = emergencyRenewPolicy.value.intervalHours;
    if (intervalHours > 0 && now.diff(lastAt, "hour", true) < intervalHours) return false;
  }
  return true;
});

const remainingDays = computed(() => {
  if (!detail.value?.expire_at) return '-';
  const now = dayjs();
  const expire = dayjs(detail.value.expire_at);
  const days = expire.diff(now, 'day');
  if (days < 0) return '已过期';
  return days === 0 ? '今天到期' : `${days} 天`;
});

const statusMap = {
  running: { text: "运行中", class: "running" },
  stopped: { text: "关机", class: "stopped" },
  error: { text: "异常", class: "error" },
  pending: { text: "创建中", class: "pending" },
  provisioning: { text: "创建中", class: "pending" },
  reinstalling: { text: "重装中", class: "reinstalling" },
  reinstall_failed: { text: "重装失败", class: "error" },
  locked: { text: "锁定", class: "locked" },
  expired_locked: { text: "已到期", class: "locked" },
  deleting: { text: "删除中", class: "pending" },
  failed: { text: "创建失败", class: "error" }
};

const statusFromAutomation = (state) => {
  switch (Number(state)) {
    case 1:
    case 13:
      return 'provisioning';
    case 2:
      return 'running';
    case 3:
      return 'stopped';
    case 4:
      return 'reinstalling';
    case 5:
      return 'reinstall_failed';
    case 10:
      return 'locked';
    case 11:
      return 'failed';
    case 12:
      return 'deleting';
    default:
      return '';
  }
};

const resolvedStatus = computed(() => {
  const raw = detail.value?.status?.toLowerCase() || "";
  const autoState = detail.value?.automation_state;
  const baseStatus = autoState !== null && autoState !== undefined ? statusFromAutomation(autoState) : raw;
  if (isExpired.value && (baseStatus === "locked" || baseStatus === "expired_locked")) {
    return "expired_locked";
  }
  return baseStatus;
});

const statusClass = computed(() => {
  const status = resolvedStatus.value;
  return statusMap[status]?.class || 'unknown';
});

const getStatusIcon = () => {
  const status = resolvedStatus.value;
  switch (status) {
    case 'running': return PlayCircleOutlined;
    case 'stopped': return PauseCircleOutlined;
    case 'error': return CloseCircleOutlined;
    case 'pending':
    case 'provisioning':
    case 'reinstalling': return SyncOutlined;
    case 'locked': return SafetyOutlined;
    default: return DesktopOutlined;
  }
};

const canStart = computed(() => {
  const status = resolvedStatus.value;
  return status === 'stopped' || status === 'error';
});

const canShutdown = computed(() => {
  const status = resolvedStatus.value;
  return status === 'running';
});

const canReboot = computed(() => {
  const status = resolvedStatus.value;
  return status === 'running';
});

const monitor = reactive({
  cpu: { labels: [], values: [] },
  memory: { labels: [], values: [] },
  trafficIn: { labels: [], values: [] },
  trafficOut: { labels: [], values: [] }
});

const currentCpu = computed(() => monitor.cpu.values[monitor.cpu.values.length - 1] || 0);
const currentMemory = computed(() => monitor.memory.values[monitor.memory.values.length - 1] || 0);
const perfScore = computed(() => {
  const cpu = Number(currentCpu.value || 0);
  const mem = Number(currentMemory.value || 0);
  const score = 100 - (cpu + mem) / 2;
  return Math.max(0, Math.min(100, Math.round(score)));
});
const perfNeedleDeg = computed(() => -90 + (perfScore.value / 100) * 180);
const perfLabel = computed(() => {
  if (perfScore.value >= 80) return "优";
  if (perfScore.value >= 60) return "良";
  if (perfScore.value >= 40) return "中";
  return "差";
});
const perfLabelClass = computed(() => ({
  good: perfScore.value >= 80,
  ok: perfScore.value >= 60 && perfScore.value < 80,
  mid: perfScore.value >= 40 && perfScore.value < 60,
  bad: perfScore.value < 40
}));
const currentTrafficIn = computed(() => monitor.trafficIn.values[monitor.trafficIn.values.length - 1] || 0);
const currentTrafficOut = computed(() => monitor.trafficOut.values[monitor.trafficOut.values.length - 1] || 0);

const getCpuColor = (val) => {
  if (val >= 90) return '#ff4d4f';
  if (val >= 70) return '#faad14';
  if (val >= 50) return '#1677ff';
  return '#52c41a';
};

const getCpuClass = (val) => {
  if (val >= 90) return 'metric-critical';
  if (val >= 70) return 'metric-warning';
  if (val >= 50) return 'metric-normal';
  return 'metric-good';
};

const getMemoryColor = (val) => {
  if (val >= 90) return '#ff4d4f';
  if (val >= 75) return '#faad14';
  if (val >= 50) return '#1677ff';
  return '#52c41a';
};

const getMemoryClass = (val) => {
  if (val >= 90) return 'metric-critical';
  if (val >= 75) return 'metric-warning';
  if (val >= 50) return 'metric-normal';
  return 'metric-good';
};

const pushPoint = (series, value) => {
  const label = new Date().toLocaleTimeString();
  series.labels.push(label);
  series.values.push(Number(value || 0));
  if (series.labels.length > 20) {
    series.labels.shift();
    series.values.shift();
  }
};

let timer;
const firewallColumns = [
  { title: "方向", dataIndex: "direction", key: "direction", width: 80 },
  { title: "协议", dataIndex: "protocol", key: "protocol", width: 80 },
  { title: "端口", dataIndex: "port", key: "port" },
  { title: "IP", dataIndex: "ip", key: "ip" },
  { title: "动作", dataIndex: "method", key: "method", width: 80 },
  { title: "操作", key: "action", width: 80 }
];

const portColumns = [
  { title: "名称", dataIndex: "name", key: "name" },
  { title: "外部地址", dataIndex: "external", key: "external" },
  { title: "目标端口", dataIndex: "dport", key: "dport" },
  { title: "操作", key: "action", width: 80 }
];

const snapshotColumns = [
  { title: "名称", dataIndex: "name", key: "name" },
  { title: "状态", dataIndex: "state_label", key: "state_label", width: 120 },
  { title: "创建时间", dataIndex: "created_at", key: "created_at" },
  { title: "操作", key: "action", width: 150 }
];

const backupColumns = [
  { title: "名称", dataIndex: "name", key: "name" },
  { title: "状态", dataIndex: "state_label", key: "state_label", width: 120 },
  { title: "创建时间", dataIndex: "created_at", key: "created_at" },
  { title: "操作", key: "action", width: 150 }
];

const snapshotBackupStateMeta = (state) => {
  switch (Number(state)) {
    case 1:
      return { label: "创建中", badge: "processing" };
    case 2:
      return { label: "创建成功", badge: "success" };
    case 3:
      return { label: "创建失败", badge: "error" };
    case 4:
      return { label: "恢复中", badge: "processing" };
    case 5:
      return { label: "删除中", badge: "warning" };
    default:
      return { label: "未知", badge: "default" };
  }
};

const normalizeFirewallRule = (item) => ({
  id: item.id ?? item.ID ?? item.rule_id ?? item.RuleID ?? item.firewall_id ?? item.FirewallID,
  direction: item.direction ?? item.Direction ?? "",
  protocol: item.protocol ?? item.Protocol ?? "",
  port: item.port ?? item.Port ?? item.start_port ?? item.StartPort ?? "",
  ip: item.ip ?? item.IP ?? item.start_ip ?? item.StartIP ?? "",
  method: item.method ?? item.Method ?? "",
  priority: item.priority ?? item.Priority ?? 0,
  raw: item
});

const normalizePortMapping = (item) => ({
  id: item.id ?? item.ID ?? item.port_id ?? item.PortID,
  name: item.name ?? item.Name ?? item.remark ?? "",
  sport: item.sport ?? item.Sport ?? item.source_port ?? item.SourcePort ?? "",
  dport: item.dport ?? item.Dport ?? item.target_port ?? item.TargetPort ?? "",
  api_url: item.api_url ?? item.apiUrl ?? item.ApiUrl ?? "",
  sys: item.sys ?? item.Sys ?? 0,
  raw: item
});

const isProtectedPortMapping = (record) => {
  const rawName = String(record?.name ?? "").trim();
  if (!rawName) return false;
  const normalizedName = rawName.toLowerCase();
  return normalizedName === "ssh" || rawName === "远程桌面";
};

const formatPortExternal = (record) => {
  const host = record?.api_url || "";
  const port = record?.sport || "";
  if (!host && !port) return "-";
  if (!host) return String(port);
  if (!port) return host;
  return `${host}:${port}`;
};

const normalizeSnapshotItem = (item) => {
  const id = item.id ?? item.ID ?? item.snapshot_id ?? item.snapshotId ?? item.sid ?? item.SID ?? item.virtuals_id ?? item.virtualsId;
  const state = item.state ?? item.State ?? 0;
  const stateMeta = snapshotBackupStateMeta(state);
  return {
    id,
    name: item.name ?? item.Name ?? (id ? `snapshot-${id}` : "snapshot"),
    state,
    state_label: stateMeta.label,
    state_badge: stateMeta.badge,
    created_at: item.created_at ?? item.create_time ?? item.createdAt ?? item.createTime ?? "",
    raw: item
  };
};

const normalizeBackupItem = (item) => {
  const id = item.id ?? item.ID ?? item.backup_id ?? item.backupId ?? item.bid ?? item.BID ?? item.virtuals_id ?? item.virtualsId;
  const state = item.state ?? item.State ?? 0;
  const stateMeta = snapshotBackupStateMeta(state);
  return {
    id,
    name: item.name ?? item.Name ?? (id ? `backup-${id}` : "backup"),
    state,
    state_label: stateMeta.label,
    state_badge: stateMeta.badge,
    created_at: item.created_at ?? item.create_time ?? item.createdAt ?? item.createTime ?? "",
    raw: item
  };
};

const fetchFirewallRules = async () => {
  firewallLoading.value = true;
  try {
    const res = await getVpsFirewallRules(id);
    const items = res.data?.data ?? res.data ?? [];
    firewallRules.value = Array.isArray(items) ? items.map(normalizeFirewallRule) : [];
  } catch (err) {
    message.error(err?.response?.data?.error || "加载防火墙规则失败");
  } finally {
    firewallLoading.value = false;
  }
};

const fetchPortMappings = async () => {
  portLoading.value = true;
  try {
    const res = await getVpsPortMappings(id);
    const items = res.data?.data ?? res.data ?? [];
    portMappings.value = Array.isArray(items) ? items.map(normalizePortMapping) : [];
  } catch (err) {
    message.error(err?.response?.data?.error || "加载端口映射失败");
  } finally {
    portLoading.value = false;
  }
};

const fetchPortCandidates = async (keywords) => {
  portCandidatesLoading.value = true;
  try {
    const res = await getVpsPortCandidates(id, { keywords });
    const items = res.data?.data ?? res.data ?? [];
    portCandidates.value = Array.isArray(items) ? items : [];
  } catch (err) {
    portCandidates.value = [];
  } finally {
    portCandidatesLoading.value = false;
  }
};

const schedulePortCandidates = (keywords) => {
  if (portCandidateTimer) {
    clearTimeout(portCandidateTimer);
  }
  portCandidateTimer = setTimeout(() => {
    fetchPortCandidates(keywords);
  }, 300);
};

const selectPortCandidate = (value) => {
  portForm.sport = String(value);
};

const onPortSportInput = () => {
  schedulePortCandidates(portForm.sport);
};

const fetchSnapshots = async () => {
  snapshotLoading.value = true;
  try {
    const res = await getVpsSnapshots(id);
    const items = res.data?.data ?? res.data ?? [];
    snapshots.value = Array.isArray(items) ? items.map(normalizeSnapshotItem) : [];
  } catch (err) {
    message.error(err?.response?.data?.error || "加载快照失败");
  } finally {
    snapshotLoading.value = false;
  }
};

const fetchBackups = async () => {
  backupLoading.value = true;
  try {
    const res = await getVpsBackups(id);
    const items = res.data?.data ?? res.data ?? [];
    backups.value = Array.isArray(items) ? items.map(normalizeBackupItem) : [];
  } catch (err) {
    message.error(err?.response?.data?.error || "加载备份失败");
  } finally {
    backupLoading.value = false;
  }
};

const loadSecurityData = async () => {
  await Promise.all([fetchFirewallRules(), fetchPortMappings(), fetchSnapshots(), fetchBackups()]);
};

watch(
  availableTabs,
  (tabs) => {
    if (!tabs.includes(activeTab.value)) {
      activeTab.value = tabs[0] || "overview";
    }
  },
  { immediate: true }
);

watch(activeTab, (val) => {
  if (["firewall", "port", "snapshot", "backup"].includes(val)) {
    loadSecurityData();
  }
});

const openFirewallModal = () => {
  firewallForm.direction = "In";
  firewallForm.protocol = "tcp";
  firewallForm.method = "allowed";
  firewallForm.port = "";
  firewallForm.ip = "0.0.0.0";
  firewallForm.priority = 100;
  firewallOpen.value = true;
};

const submitFirewallRule = async () => {
  if (!firewallForm.port) {
    message.error("请输入端口");
    return;
  }
  if (!firewallForm.ip) {
    message.error("请输入IP地址");
    return;
  }
  try {
    await addVpsFirewallRule(id, { ...firewallForm });
    firewallOpen.value = false;
    await fetchFirewallRules();
    message.success("防火墙规则已添加");
  } catch (err) {
    message.error(err?.response?.data?.error || "添加防火墙规则失败");
    await fetchFirewallRules();
  }
};

const removeFirewallRule = (record) => {
  const ruleId = record?.id || record?.ID;
  if (!ruleId) return;
  Modal.confirm({
    title: "删除防火墙规则",
    content: "确定要删除此规则吗？",
    onOk: async () => {
      try {
        await deleteVpsFirewallRule(id, ruleId);
        await fetchFirewallRules();
        message.success("防火墙规则已删除");
      } catch (err) {
        message.error("删除失败");
      }
    }
  });
};

const openPortModal = () => {
  portForm.name = "";
  portForm.sport = "";
  portForm.dport = "";
  portCandidates.value = [];
  fetchPortCandidates("");
  portOpen.value = true;
};

const submitPortMapping = async () => {
  if (!portForm.dport) {
    message.error("请输入目标端口");
    return;
  }
  if (String(portForm.name || "").length > INPUT_LIMITS.PORT_MAPPING_NAME) {
    message.error(`端口映射名称长度不能超过 ${INPUT_LIMITS.PORT_MAPPING_NAME} 个字符`);
    return;
  }
  portLoading.value = true;
  try {
    await addVpsPortMapping(id, {
      name: portForm.name,
      sport: portForm.sport?.trim(),
      dport: Number(portForm.dport)
    });
    portOpen.value = false;
    await fetchPortMappings();
    message.success("端口映射已添加");
  } catch (err) {
    message.error(err?.response?.data?.error || "添加端口映射失败");
  } finally {
    portLoading.value = false;
  }
};

const removePortMapping = (record) => {
  const mappingId = record?.id || record?.ID;
  if (!mappingId) return;
  Modal.confirm({
    title: "删除端口映射",
    content: "确定要删除此映射吗？",
    onOk: async () => {
      try {
        await deleteVpsPortMapping(id, mappingId);
        await fetchPortMappings();
        message.success("端口映射已删除");
      } catch (err) {
        message.error("删除失败");
      }
    }
  });
};

const createSnapshot = async () => {
  try {
    await createVpsSnapshot(id);
    await fetchSnapshots();
    message.success("快照已创建");
  } catch (err) {
    message.error(err?.response?.data?.error || "创建快照失败");
  }
};

const deleteSnapshot = (record) => {
  const snapId = record?.id || record?.ID;
  if (!snapId) return;
  Modal.confirm({
    title: "删除快照",
    content: "确定要删除此快照吗？",
    onOk: async () => {
      try {
        await deleteVpsSnapshot(id, snapId);
        await fetchSnapshots();
        message.success("快照已删除");
      } catch (err) {
        message.error("删除失败");
      }
    }
  });
};

const restoreSnapshot = (record) => {
  const snapId = record?.id || record?.ID;
  if (!snapId) return;
  Modal.confirm({
    title: "恢复快照",
    content: "确定要恢复到此快照吗？此操作不可撤销。",
    onOk: async () => {
      try {
        await restoreVpsSnapshot(id, snapId);
        message.success("快照恢复已开始");
      } catch (err) {
        message.error(err?.response?.data?.error || "恢复快照失败");
      }
    }
  });
};

const createBackup = async () => {
  try {
    await createVpsBackup(id);
    await fetchBackups();
    message.success("备份已创建");
  } catch (err) {
    message.error(err?.response?.data?.error || "创建备份失败");
  }
};

const deleteBackup = (record) => {
  const backupId = record?.id || record?.ID;
  if (!backupId) return;
  Modal.confirm({
    title: "删除备份",
    content: "确定要删除此备份吗？",
    onOk: async () => {
      try {
        await deleteVpsBackup(id, backupId);
        await fetchBackups();
        message.success("备份已删除");
      } catch (err) {
        message.error("删除失败");
      }
    }
  });
};

const restoreBackup = (record) => {
  const backupId = record?.id || record?.ID;
  if (!backupId) return;
  Modal.confirm({
    title: "恢复备份",
    content: "确定要恢复到此备份吗？此操作不可撤销。",
    onOk: async () => {
      try {
        await restoreVpsBackup(id, backupId);
        message.success("备份恢复已开始");
      } catch (err) {
        message.error(err?.response?.data?.error || "恢复备份失败");
      }
    }
  });
};

const fetchMonitor = async () => {
  try {
    const res = await getVpsMonitor(id);
    const data = res.data || {};
    if (store.current && (data.status || data.automation_state !== undefined || data.access_info || data.spec)) {
      store.current = {
        ...store.current,
        status: data.status ?? store.current.status,
        automation_state: data.automation_state ?? store.current.automation_state,
        access_info: data.access_info ?? store.current.access_info,
        spec: data.spec ?? store.current.spec
      };
    }
    pushPoint(monitor.cpu, data.cpu);
    pushPoint(monitor.memory, data.memory);
    const trafficIn = Number(data.bytes_in ?? data.in_bytes ?? data.rx_bytes ?? data.in ?? 0);
    const trafficOut = Number(data.bytes_out ?? data.out_bytes ?? data.tx_bytes ?? data.out ?? 0);
    pushPoint(monitor.trafficIn, Math.round(trafficIn / 1024));
    pushPoint(monitor.trafficOut, Math.round(trafficOut / 1024));
  } catch (err) {
    console.error('Failed to fetch monitor:', err);
  }
};

const base = import.meta.env.VITE_API_BASE || "";

const handleMoreAction = ({ key }) => {
  switch (key) {
    case 'reinstall': openReinstall(); break;
    case 'resetPassword': openResetOsPassword(); break;
    case 'refund': openRefund(); break;
  }
};

const openPanel = () => {
  const token = auth.token;
  const query = token ? `?token=${encodeURIComponent(token)}` : "";
  window.open(`${base}/api/v1/vps/${id}/panel${query}`, "_blank");
};

const openVnc = () => {
  const token = auth.token;
  const query = token ? `?token=${encodeURIComponent(token)}` : "";
  window.open(`${base}/api/v1/vps/${id}/vnc${query}`, "_blank");
};

const copyText = async (text, name) => {
  await navigator.clipboard.writeText(text);
  message.success(`已复制${name}`);
};

const copyRemoteIp = async () => {
  await copyText(access.value.remote_ip, '远程IP');
};

const formatLocalTime = (time) => {
  if (!time) return '-';
  return dayjs(time).format('YYYY-MM-DD HH:mm');
};

const copySshCommand = async () => {
  const cmd = connectCommand.value;
  await navigator.clipboard.writeText(cmd);
  message.success(`已复制 ${connectCommandLabel.value}`);
};

const openResetOsPassword = () => {
  resetOsPasswordForm.password = access.value.os_password || "";
  resetOsPasswordOpen.value = true;
};

const generateOsPassword = () => {
  const lower = "abcdefghjkmnpqrstuvwxyz";
  const upper = "ABCDEFGHJKMNPQRSTUVWXYZ";
  const digits = "23456789";
  const symbols = "!@#$%^&*";
  const all = `${lower}${upper}${digits}${symbols}`;
  const pick = (src) => src[Math.floor(Math.random() * src.length)];
  const chars = [pick(lower), pick(upper), pick(digits), pick(symbols)];
  while (chars.length < 12) {
    chars.push(pick(all));
  }
  resetOsPasswordForm.password = chars.join("");
};

const submitResetOsPassword = async () => {
  if (!resetOsPasswordForm.password) {
    message.error("请输入新面板密码");
    return;
  }
  if (String(resetOsPasswordForm.password || "").length > INPUT_LIMITS.PASSWORD) {
    message.error(`密码长度不能超过 ${INPUT_LIMITS.PASSWORD} 个字符`);
    return;
  }
  resetOsPasswordLoading.value = true;
  try {
    await resetVpsOsPassword(id, { password: resetOsPasswordForm.password });
    const info = parseJson(detail.value?.access_info);
    info.os_password = resetOsPasswordForm.password;
    if (store.current) {
      store.current = {
        ...store.current,
        access_info: JSON.stringify(info)
      };
    }
    message.success("面板密码已更新");
    resetOsPasswordOpen.value = false;
  } catch (err) {
    message.error(err?.response?.data?.error || "操作失败");
  } finally {
    resetOsPasswordLoading.value = false;
  }
};

const openReinstall = () => {
  const open = async () => {
    let lineId = Number(detail.value?.line_id || 0);
    if (!lineId) {
      const packageId = Number(detail.value?.package_id || 0);
      if (packageId) {
        const pkg = catalog.packages.find((item) => Number(item.id || 0) === packageId);
        const planGroupId = Number(pkg?.plan_group_id || 0);
        if (planGroupId) {
          const plan = catalog.planGroups.find((item) => Number(item.id || 0) === planGroupId);
          lineId = Number(plan?.line_id || 0);
        }
      }
    }

    try {
      const params = lineId > 0 ? { line_id: lineId } : undefined;
      const res = await listSystemImages(params);
      reinstallImages.value = res?.data?.items || [];
    } catch (err) {
      message.error(err?.response?.data?.error || err?.response?.data?.message || "获取镜像失败");
      return;
    }

    if (!systemImageOptions.value.length) {
      message.error("当前线路暂无可用镜像");
      return;
    }
    reinstallForm.template_id = systemImageOptions.value[0]?.id || null;
    reinstallForm.password = access.value.os_password || "";
    reinstallOpen.value = true;
  };

  open();
};

const generateReinstallPassword = () => {
  const lower = "abcdefghjkmnpqrstuvwxyz";
  const upper = "ABCDEFGHJKMNPQRSTUVWXYZ";
  const digits = "23456789";
  const symbols = "!@#$%^&*";
  const all = `${lower}${upper}${digits}${symbols}`;
  const pick = (src) => src[Math.floor(Math.random() * src.length)];
  const chars = [pick(lower), pick(upper), pick(digits), pick(symbols)];
  while (chars.length < 12) {
    chars.push(pick(all));
  }
  for (let i = chars.length - 1; i > 0; i -= 1) {
    const j = Math.floor(Math.random() * (i + 1));
    [chars[i], chars[j]] = [chars[j], chars[i]];
  }
  reinstallForm.password = chars.join("");
};

const submitReinstall = async () => {
  if (!reinstallForm.template_id) {
    message.error("请选择系统镜像");
    return;
  }
  if (!reinstallForm.password) {
    message.error("请输入新系统密码");
    return;
  }
  if (String(reinstallForm.password || "").length > INPUT_LIMITS.PASSWORD) {
    message.error(`密码长度不能超过 ${INPUT_LIMITS.PASSWORD} 个字符`);
    return;
  }
  reinstalling.value = true;
  try {
    await resetVpsOS(id, {
      template_id: reinstallForm.template_id,
      password: reinstallForm.password
    });
    message.success("已放入重装队列");
    reinstallOpen.value = false;
  } catch (err) {
    const errorText = err?.response?.data?.error || err?.response?.data?.message || "操作失败";
    message.error(errorText);
  } finally {
    reinstalling.value = false;
  }
};

const openRenew = () => {
  renewForm.cycleQty = 1;
  renewForm.cycleId = billingCycles.value[0]?.id ?? null;
  renewOpen.value = true;
};

const submitEmergencyRenew = () => {
  if (!detail.value?.id) return;
  Modal.confirm({
    title: "紧急续费确认",
    content: "紧急续费将按系统策略续费固定天数，确认继续？",
    okText: "确认",
    cancelText: "取消",
    async onOk() {
      try {
        await emergencyRenewVps(detail.value.id);
        message.success("紧急续费已提交");
        await refresh();
      } catch (err) {
        message.error(err.response?.data?.error || "紧急续费失败");
      }
    }
  });
};

const submitRenew = async () => {
  renewing.value = true;
  try {
    await createVpsRenewOrder(id, { duration_months: renewMonths.value });
    message.success("已生成续费订单");
    renewOpen.value = false;
  } catch (err) {
    const status = err?.response?.status;
    const errorText = err?.response?.data?.error || err?.response?.data?.message || '操作失败';
    if (status === 409) {
      Modal.confirm({
        title: "已有待处理续费订单",
        content: errorText,
        okText: "去订单列表处理",
        cancelText: "我知道了",
        onOk: () => router.push("/console/orders")
      });
    } else {
      message.error(errorText);
    }
  } finally {
    renewing.value = false;
  }
};

const openResize = () => {
  if (isExpired.value) {
    message.warning("已到期实例不支持升降配");
    return;
  }
  resizeForm.add_cores = currentAddons.value.add_cores;
  resizeForm.add_mem_gb = currentAddons.value.add_mem_gb;
  resizeForm.add_disk_gb = currentAddons.value.add_disk_gb;
  resizeForm.add_bw_mbps = currentAddons.value.add_bw_mbps;
  resizeForm.target_package_id = currentPackage.value?.id ?? null;
  resizeForm.reset_addons = false;
  resizeForm.schedule_mode = "now";
  resizeForm.scheduled_at = null;
  normalizeResizeAddons();
  resizeQuote.value = null;
  resizeQuoteError.value = "";
  resizeOpen.value = true;
  scheduleResizeQuote();
};

const closeResize = () => {
  if (resizeQuoteTimer) {
    clearTimeout(resizeQuoteTimer);
    resizeQuoteTimer = null;
  }
  resizeOpen.value = false;
};

const clampAddonValue = (value, min, max, step) => {
  const safeStep = Math.max(1, Number(step || 1));
  const safeMin = Number(min || 0);
  const safeMax = Math.max(safeMin, Number(max || safeMin));
  let next = Number(value || 0);
  if (!Number.isFinite(next)) next = safeMin;
  next = Math.max(safeMin, Math.min(safeMax, next));
  next = safeMin + Math.round((next - safeMin) / safeStep) * safeStep;
  if (next > safeMax) next = safeMax;
  if (next < safeMin) next = safeMin;
  return next;
};

const resetResizeAddonsToMin = () => {
  resizeForm.add_cores = addonMin.value.add_cores;
  resizeForm.add_mem_gb = addonMin.value.add_mem_gb;
  resizeForm.add_disk_gb = addonMin.value.add_disk_gb;
  resizeForm.add_bw_mbps = addonMin.value.add_bw_mbps;
};

const normalizeResizeAddons = () => {
  resizeForm.add_cores = clampAddonValue(resizeForm.add_cores, addonMin.value.add_cores, addonMax.value.add_cores, addonStep.value.add_cores);
  resizeForm.add_mem_gb = clampAddonValue(resizeForm.add_mem_gb, addonMin.value.add_mem_gb, addonMax.value.add_mem_gb, addonStep.value.add_mem_gb);
  resizeForm.add_disk_gb = clampAddonValue(resizeForm.add_disk_gb, addonMin.value.add_disk_gb, addonMax.value.add_disk_gb, addonStep.value.add_disk_gb);
  resizeForm.add_bw_mbps = clampAddonValue(resizeForm.add_bw_mbps, addonMin.value.add_bw_mbps, addonMax.value.add_bw_mbps, addonStep.value.add_bw_mbps);
};

const buildResizeQuotePayload = () => {
  const minSpec = {
    add_cores: addonMin.value.add_cores,
    add_mem_gb: addonMin.value.add_mem_gb,
    add_disk_gb: addonMin.value.add_disk_gb,
    add_bw_mbps: addonMin.value.add_bw_mbps
  };
  const spec = resizeForm.reset_addons
    ? minSpec
    : {
        add_cores: resizeForm.add_cores,
        add_mem_gb: resizeForm.add_mem_gb,
        add_disk_gb: resizeForm.add_disk_gb,
        add_bw_mbps: resizeForm.add_bw_mbps
      };
  return {
    target_package_id: resizeForm.target_package_id,
    reset_addons: resizeForm.reset_addons,
    spec
  };
};

const onPackageChange = () => {
  // 套餐改变后立即重置到新下限，避免出现非法缩容配置。
  resetResizeAddonsToMin();
  resizeForm.reset_addons = false;
  resizeForm.schedule_mode = "now";
  resizeForm.scheduled_at = null;
};

const onResetAddonsChange = (e) => {
  if (e.target.checked) {
    resetResizeAddonsToMin();
  }
};

const disabledDate = (current) => {
  // 禁用今天之前的日期
  return current && current < dayjs().startOf('day');
};

const disabledTime = (current) => {
  if (!current || !dayjs(current).isSame(dayjs(), 'day')) {
    return {};
  }
  // 如果是今天，禁用当前时间之前的小时和分钟
  const now = dayjs();
  return {
    disabledHours: () => {
      const hours = [];
      for (let i = 0; i < now.hour(); i++) {
        hours.push(i);
      }
      return hours;
    },
    disabledMinutes: (selectedHour) => {
      if (selectedHour !== now.hour()) return [];
      const minutes = [];
      for (let i = 0; i < now.minute(); i++) {
        minutes.push(i);
      }
      return minutes;
    },
    disabledSeconds: () => []
  };
};

const fetchResizeQuote = async () => {
  if (!resizeEnabled.value) {
    resizeQuote.value = null;
    resizeQuoteError.value = "升降配功能已关闭";
    return;
  }
  if (!resizeForm.target_package_id || isSameTargetSelection.value) {
    resizeQuote.value = null;
    resizeQuoteError.value = "";
    return;
  }
  const targetPkg = packageOptions.value.find((pkg) => String(pkg.id) === String(resizeForm.target_package_id));
  if (isResizePackageDisabled(targetPkg)) {
    resizeQuote.value = null;
    resizeQuoteError.value = "目标套餐无法满足当前磁盘容量";
    return;
  }
  resizeQuoteLoading.value = true;
  resizeQuoteError.value = "";
  try {
    const res = await quoteVpsResizeOrder(id, buildResizeQuotePayload());
    resizeQuote.value = res.data?.quote ?? res.data;
  } catch (err) {
    const status = err?.response?.status;
    const errorText = err?.response?.data?.error || err?.response?.data?.message || "操作失败";
    resizeQuote.value = null;
    if (status === 409) {
      resizeQuoteError.value = "已有进行中的升降配任务/订单";
      return;
    }
    resizeQuoteError.value = errorText;
  } finally {
    resizeQuoteLoading.value = false;
  }
};

let resizeQuoteTimer;
const scheduleResizeQuote = () => {
  if (resizeQuoteTimer) {
    clearTimeout(resizeQuoteTimer);
  }
  resizeQuoteTimer = setTimeout(() => {
    void fetchResizeQuote();
  }, 300);
};

const submitResize = async () => {
  if (isExpired.value) {
    message.warning("已到期实例不支持升降配");
    return;
  }
  if (!resizeEnabled.value) {
    message.error("升降配功能已关闭");
    return;
  }
  if (!resizeForm.target_package_id) {
    message.warning("请选择目标套餐");
    return;
  }
  const targetPkg = packageOptions.value.find((pkg) => String(pkg.id) === String(resizeForm.target_package_id));
  if (isResizePackageDisabled(targetPkg)) {
    message.warning("目标套餐无法满足当前磁盘容量，无法切换");
    return;
  }
  if (isSameTargetSelection.value) {
    message.warning("不能选择当前套餐");
    return;
  }
  if (resizeForm.schedule_mode === 'scheduled' && !resizeForm.scheduled_at) {
    message.warning("请选择执行时间");
    return;
  }
  resizing.value = true;
  try {
    const payload = buildResizeQuotePayload();
    // 如果选择定时执行，添加 scheduled_at 参数
    if (resizeForm.schedule_mode === 'scheduled' && resizeForm.scheduled_at) {
      payload.scheduled_at = resizeForm.scheduled_at.format('YYYY-MM-DD HH:mm:ss');
    }
    const res = await createVpsResizeOrder(id, payload);
    const order = res.data?.order ?? res.data;
    const successMsg = resizeForm.schedule_mode === 'scheduled'
      ? "已生成升降配订单，将在指定时间执行"
      : "已生成升降配订单";
    message.success(successMsg);
    resizeOpen.value = false;
    if (order?.id) {
      router.push(`/console/orders/${order.id}`);
    }
  } catch (err) {
    const status = err?.response?.status;
    const errorText = err?.response?.data?.error || err?.response?.data?.message || "操作失败";
    if (status === 409) {
      const data = err?.response?.data || {};
      const orderId = data.order?.id || data.order_id || data.orderId;
      Modal.confirm({
        title: "已有进行中的升降配任务/订单",
        content: errorText,
        okText: orderId ? "去订单详情" : "去订单列表",
        cancelText: "我知道了",
        onOk: () => {
          if (orderId) {
            router.push(`/console/orders/${orderId}`);
          } else {
            router.push("/console/orders");
          }
        }
      });
      return;
    }
    message.error(errorText);
  } finally {
    resizing.value = false;
  }
};

watch(
  () => [
    addonMin.value.add_cores,
    addonMin.value.add_mem_gb,
    addonMin.value.add_disk_gb,
    addonMin.value.add_bw_mbps,
    addonMax.value.add_cores,
    addonMax.value.add_mem_gb,
    addonMax.value.add_disk_gb,
    addonMax.value.add_bw_mbps,
    addonStep.value.add_cores,
    addonStep.value.add_mem_gb,
    addonStep.value.add_disk_gb,
    addonStep.value.add_bw_mbps
  ],
  () => {
    if (!resizeOpen.value) return;
    normalizeResizeAddons();
  }
);

watch(
  () => [
    resizeForm.target_package_id,
    resizeForm.add_cores,
    resizeForm.add_mem_gb,
    resizeForm.add_disk_gb,
    resizeForm.add_bw_mbps,
    resizeForm.reset_addons
  ],
  () => {
    if (!resizeOpen.value) return;
    scheduleResizeQuote();
  }
);

const reboot = async () => {
  try {
    await rebootVps(id);
    message.success("已触发重启");
  } catch (err) {
    message.error(err.response?.data?.message || '操作失败');
  }
};

const start = async () => {
  try {
    await startVps(id);
    message.success("已触发开机");
  } catch (err) {
    message.error(err.response?.data?.message || '操作失败');
  }
};

const shutdown = async () => {
  try {
    await shutdownVps(id);
    message.success("已触发关机");
  } catch (err) {
    message.error(err.response?.data?.message || '操作失败');
  }
};

const openRefund = () => {
  if (!refundEnabled.value) {
    message.error("当前实例不支持退款");
    return;
  }
  refundReason.value = "";
  refundOpen.value = true;
};

const submitRefund = async () => {
  if (refunding.value) return;
  if (!refundReason.value.trim()) {
    message.error("请填写退款原因");
    return;
  }
  if (String(refundReason.value || "").length > INPUT_LIMITS.REFUND_REASON) {
    message.error(`退款原因长度不能超过 ${INPUT_LIMITS.REFUND_REASON} 个字符`);
    return;
  }
  refunding.value = true;
  try {
    const res = await requestVpsRefund(id, { reason: refundReason.value });
    const orderId = res?.data?.order?.id ?? res?.data?.order?.ID;
    if (orderId) {
      message.success("已提交退款申请，订单ID: " + orderId + "，请到余额/账户查看");
    } else {
      message.success("已提交退款申请");
    }
    refundOpen.value = false;
  } catch (err) {
    message.error(err?.response?.data?.error || err?.response?.data?.message || "提交失败");
  } finally {
    refunding.value = false;
  }
};

const refresh = async () => {
  loading.value = true;
  try {
    await store.refresh(id);
    await fetchMonitor();
    message.success("已刷新");
  } catch (err) {
    message.error(err.response?.data?.message || '刷新失败');
  } finally {
    loading.value = false;
  }
};

onMounted(async () => {
  loading.value = true;
  try {
    if (!catalog.systemImages.length || !catalog.packages.length || !catalog.planGroups.length) {
      await catalog.fetchCatalog();
    }
    await site.fetchSettings();
    await store.fetchDetail(id);
    await fetchMonitor();
    timer = setInterval(fetchMonitor, 10000);
  } finally {
    loading.value = false;
  }
});

onBeforeUnmount(() => {
  if (timer) clearInterval(timer);
  if (resizeQuoteTimer) clearTimeout(resizeQuoteTimer);
  if (portCandidateTimer) clearTimeout(portCandidateTimer);
});
</script>

<style scoped>
.vps-detail-page {
  padding: 24px;
  background: #f0f2f5;
  min-height: calc(100vh - 64px);
  color: rgba(0, 0, 0, 0.88);
}

/* ========== Detail Header ========== */
.detail-header {
  background: #ffffff;
  border-radius: 8px 8px 0 0;
  margin-bottom: 0;
  box-shadow: 0 1px 2px 0 rgba(0, 0, 0, 0.03), 0 1px 6px -1px rgba(0, 0, 0, 0.02);
  overflow: hidden;
}

.header-breadcrumb {
  padding: 12px 24px;
  border-bottom: 1px solid #f0f0f0;
  background: #fafafa;
}

.header-breadcrumb :deep(.ant-breadcrumb) {
  font-size: 14px;
}

.header-breadcrumb :deep(.ant-breadcrumb-link) {
  color: rgba(0, 0, 0, 0.65);
}

.header-breadcrumb :deep(.ant-breadcrumb-link:hover) {
  color: #1677ff;
}

.header-main {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  padding: 20px 24px;
  gap: 24px;
}

.instance-info {
  flex: 1;
  min-width: 0;
}

.instance-title {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
}

.title-icon {
  font-size: 24px;
  color: #1677ff;
  flex-shrink: 0;
}

.title-text {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  color: rgba(0, 0, 0, 0.85);
  line-height: 1.4;
}

.title-status {
  margin-left: auto;
  flex-shrink: 0;
}

.instance-meta {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  font-size: 14px;
}

.instance-meta :deep(.ant-divider-vertical) {
  margin: 0 4px;
  border-color: #f0f0f0;
}

.meta-item {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 2px 0;
}

.meta-label {
  color: rgba(0, 0, 0, 0.45);
  font-size: 13px;
  margin-right: 2px;
}

.meta-value {
  color: rgba(0, 0, 0, 0.85);
  font-family: 'SFMono-Regular', Consolas, monospace;
  font-weight: 500;
  background: #fafafa;
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 13px;
}

.meta-icon {
  font-size: 14px;
  color: rgba(0, 0, 0, 0.45);
}

.header-actions {
  display: flex;
  gap: 8px;
  flex-shrink: 0;
}

.header-actions .ant-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

/* ========== Tabs ========== */
.ecs-tabs {
  margin-top: 0;
}

.ecs-tabs :deep(.ant-tabs-nav) {
  margin: 0 0 16px 0;
  background: #ffffff;
  border-radius: 0 0 8px 8px;
  padding: 0 20px;
  border-top: 1px solid #f0f0f0;
  box-shadow: 0 1px 2px 0 rgba(0, 0, 0, 0.03), 0 1px 6px -1px rgba(0, 0, 0, 0.02);
}

.ecs-tabs :deep(.ant-tabs-nav::before) {
  display: none;
}

.ecs-tabs :deep(.ant-tabs-tab) {
  padding: 14px 20px;
  font-weight: 500;
  color: rgba(0, 0, 0, 0.65);
  font-size: 14px;
  position: relative;
  transition: all 0.2s;
}

.ecs-tabs :deep(.ant-tabs-tab:hover) {
  color: #1677ff;
}

.ecs-tabs :deep(.ant-tabs-tab-active) {
  color: #1677ff;
  font-weight: 600;
}

.ecs-tabs :deep(.ant-tabs-tab-active .ant-tabs-tab-btn) {
  color: #1677ff;
}

.ecs-tabs :deep(.ant-tabs-ink-bar) {
  background: linear-gradient(90deg, #1677ff 0%, #4096ff 100%);
  height: 3px;
  border-radius: 3px 3px 0 0;
}

.ecs-tabs :deep(.ant-tabs-content) {
  padding-top: 0;
}

.ecs-tabs :deep(.ant-tabs-tab-btn) {
  outline: none;
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.tab-icon {
  font-size: 15px;
}

/* ========== Overview Grid ========== */
.overview-grid {
  display: grid;
  grid-template-columns: repeat(12, minmax(0, 1fr));
  gap: 20px;
}

.overview-card {
  border-radius: 12px;
  border: 1px solid rgba(0, 0, 0, 0.08);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04), 0 4px 12px rgba(0, 0, 0, 0.02);
  overflow: hidden;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  background: #ffffff;
}

.overview-card:hover {
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.08);
  transform: translateY(-2px);
}

.instance-card { grid-column: span 8; }
.monitor-card { grid-column: span 4; }
.account-card { grid-column: span 4; }
.time-card { grid-column: span 8; }

.overview-card :deep(.ant-card-head) {
  border-bottom: 1px solid rgba(0, 0, 0, 0.06);
  padding: 18px 24px;
  background: linear-gradient(180deg, #fafbfc 0%, #ffffff 100%);
}

.overview-card :deep(.ant-card-body) {
  padding: 24px;
}

.card-title {
  display: flex;
  align-items: center;
  gap: 10px;
  font-weight: 600;
  font-size: 16px;
  color: rgba(0, 0, 0, 0.88);
}

.card-title .anticon {
  font-size: 18px;
  color: #1677ff;
}

.expire-tag {
  margin-left: auto;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  animation: expirePulse 2s ease-in-out infinite;
}

@keyframes expirePulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.7; }
}

.live-tag {
  margin-left: auto;
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.live-dot {
  width: 6px;
  height: 6px;
  background: #52c41a;
  border-radius: 50%;
  animation: livePulse 1.5s ease-in-out infinite;
}

@keyframes livePulse {
  0%, 100% { opacity: 1; transform: scale(1); }
  50% { opacity: 0.5; transform: scale(1.2); }
}

/* ========== Instance Card ========== */
.instance-info-wrapper {
  padding: 0;
}

/* Info List */
.info-list {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 8px;
  margin-bottom: 24px;
}

.info-list-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 12px;
  border-radius: 6px;
  transition: background 0.2s;
}

.info-list-item:hover {
  background: #fafafa;
}

.info-list-icon {
  width: 32px;
  height: 32px;
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 16px;
  background: #f5f5f5;
  color: rgba(0, 0, 0, 0.45);
  flex-shrink: 0;
}

.info-list-content {
  flex: 1;
  min-width: 0;
}

.info-list-label {
  font-size: 12px;
  color: rgba(0, 0, 0, 0.45);
  margin-bottom: 2px;
}

.info-list-value {
  font-size: 14px;
  color: rgba(0, 0, 0, 0.88);
  font-weight: 500;
  display: flex;
  align-items: center;
  gap: 8px;
}

.info-list-value.value-warning {
  color: #fa8c16;
}

.info-list-value :deep(.ant-btn-link) {
  padding: 0;
  height: auto;
  font-size: 13px;
}

.masked {
  font-family: 'SFMono-Regular', Consolas, monospace;
  letter-spacing: 2px;
}

/* Specs Section */
.text-warning {
  color: #faad14;
}

.masked {
  font-family: 'SFMono-Regular', Consolas, monospace;
  letter-spacing: 2px;
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #52c41a;
  display: inline-block;
}

/* ========== Monitor Card ========== */
.monitor-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.monitor-item {
  display: grid;
  grid-template-columns: 80px 120px 1fr;
  align-items: center;
  gap: 16px;
  padding: 12px 0;
  border-bottom: 1px solid #f0f0f0;
}

.monitor-item:last-child {
  border-bottom: none;
  padding-bottom: 0;
}

.monitor-item:first-child {
  padding-top: 0;
}

.monitor-label {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  color: rgba(0, 0, 0, 0.65);
  font-weight: 500;
}

.monitor-icon {
  font-size: 16px;
  color: rgba(0, 0, 0, 0.45);
}

.monitor-value-group {
  display: flex;
  align-items: baseline;
  gap: 8px;
}

.monitor-value {
  font-size: 20px;
  font-weight: 600;
  color: rgba(0, 0, 0, 0.88);
  font-family: 'SF Mono', 'Monaco', 'Consolas', monospace;
}

.monitor-value.metric-good { color: #52c41a; }
.monitor-value.metric-normal { color: #1677ff; }
.monitor-value.metric-warning { color: #faad14; }
.monitor-value.metric-critical { color: #ff4d4f; }

.monitor-spec {
  font-size: 13px;
  color: rgba(0, 0, 0, 0.45);
}

.monitor-bar {
  height: 6px;
  background: #f0f0f0;
  border-radius: 3px;
  overflow: hidden;
  position: relative;
}

.monitor-bar-fill {
  height: 100%;
  border-radius: 3px;
  transition: width 0.3s ease;
}

.network-item {
  grid-template-columns: 80px 1fr;
}

.network-stats {
  display: flex;
  gap: 32px;
}

.network-stat {
  display: flex;
  align-items: baseline;
  gap: 6px;
}

.network-label-text {
  font-size: 14px;
  color: rgba(0, 0, 0, 0.45);
}

.network-value-text {
  font-size: 16px;
  font-weight: 600;
  color: rgba(0, 0, 0, 0.88);
  font-family: 'SF Mono', 'Monaco', 'Consolas', monospace;
}

.network-unit-text {
  font-size: 12px;
  color: rgba(0, 0, 0, 0.45);
}

/* ========== Time & Price Card ========== */
.time-price-body {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.info-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16px;
}

.info-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.info-label {
  font-size: 13px;
  color: rgba(0, 0, 0, 0.45);
}

.info-value {
  font-size: 15px;
  color: rgba(0, 0, 0, 0.88);
  font-weight: 500;
}

.info-value.value-warning {
  color: #faad14;
}

.price-value {
  font-size: 18px;
  font-weight: 600;
  color: #1677ff;
  font-family: 'SF Mono', 'Monaco', 'Consolas', monospace;
}

.price-unit {
  font-size: 13px;
  color: rgba(0, 0, 0, 0.45);
  font-weight: 400;
  margin-left: 4px;
}

.action-buttons {
  display: flex;
  gap: 12px;
}

.action-buttons .ant-btn {
  flex: 1;
  height: 40px;
  font-size: 15px;
}

/* ========== Connection Card ========== */
.account-rows {
  display: flex;
  flex-direction: column;
  gap: 0;
}

.account-row-item {
  display: grid;
  grid-template-columns: 90px 1fr;
  align-items: center;
  gap: 16px;
  padding: 12px 0;
  border-bottom: 1px solid #f0f0f0;
}

.account-row-item:last-child {
  border-bottom: none;
  padding-bottom: 0;
}

.account-row-item:first-child {
  padding-top: 0;
}

.account-row-label {
  font-size: 14px;
  color: rgba(0, 0, 0, 0.55);
  font-weight: 500;
}

.account-row-value {
  font-size: 14px;
  color: rgba(0, 0, 0, 0.88);
  font-weight: 500;
  display: flex;
  align-items: center;
  gap: 8px;
}

.account-row-value .password-text {
  font-family: 'SF Mono', 'Monaco', 'Consolas', monospace;
}

.account-row-value .password-text.masked {
  letter-spacing: 2px;
}

/* ========== Monitor Layout ========== */
.monitor-layout {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 20px;
}

.monitor-layout .monitor-panel {
  border-radius: 12px;
  border: 1px solid rgba(0, 0, 0, 0.08);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04), 0 4px 12px rgba(0, 0, 0, 0.02);
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

.monitor-layout .monitor-panel:hover {
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.08);
  transform: translateY(-2px);
}

.monitor-layout .monitor-panel:first-child {
  grid-column: span 2;
}

.monitor-layout .monitor-panel :deep(.ant-card-head) {
  border-bottom: 1px solid rgba(0, 0, 0, 0.06);
  padding: 18px 24px;
  background: linear-gradient(180deg, #fafbfc 0%, #ffffff 100%);
}

.monitor-layout .monitor-panel :deep(.ant-card-body) {
  padding: 24px;
}

.monitor-summary {
  display: grid;
  grid-template-columns: 1.1fr 0.9fr;
  gap: 24px;
  align-items: center;
}

.summary-list {
  display: grid;
  gap: 14px;
  font-size: 14px;
  color: rgba(0, 0, 0, 0.65);
}

.summary-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-bottom: 10px;
  border-bottom: 1px solid rgba(0, 0, 0, 0.06);
}

.summary-row:last-child {
  border-bottom: none;
  padding-bottom: 0;
}

.summary-value {
  color: rgba(0, 0, 0, 0.88);
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-weight: 500;
}

.summary-value.warning {
  color: #faad14;
}

.gauge-wrap {
  display: grid;
  justify-items: center;
  gap: 10px;
}

.gauge {
  width: 160px;
  height: 80px;
  position: relative;
  background: conic-gradient(
    from 270deg,
    #52c41a 0deg,
    #52c41a 90deg,
    #fadb14 90deg,
    #fadb14 140deg,
    #faad14 140deg,
    #faad14 180deg,
    transparent 180deg
  );
  border-radius: 160px 160px 0 0;
  overflow: hidden;
  box-shadow: inset 0 -2px 8px rgba(0, 0, 0, 0.08);
}

.gauge-mask {
  position: absolute;
  inset: 14px 14px 0 14px;
  background: #ffffff;
  border-radius: 160px 160px 0 0;
  box-shadow: 0 -2px 8px rgba(0, 0, 0, 0.04);
}

.gauge-needle {
  position: absolute;
  width: 60px;
  height: 2px;
  background: linear-gradient(90deg, rgba(0, 0, 0, 0.1) 0%, rgba(0, 0, 0, 0.3) 100%);
  bottom: 12px;
  left: 50%;
  transform-origin: left center;
  transition: transform 0.5s cubic-bezier(0.4, 0, 0.2, 1);
  border-radius: 1px;
}

.gauge-label {
  font-size: 24px;
  font-weight: 600;
  line-height: 1;
}

.gauge-label.good { color: #52c41a; }
.gauge-label.ok { color: #faad14; }
.gauge-label.mid { color: #fa8c16; }
.gauge-label.bad { color: #ff4d4f; }

.gauge-sub {
  font-size: 12px;
  color: rgba(0, 0, 0, 0.45);
  font-weight: 500;
}

@media (max-width: 1200px) {
  .monitor-layout {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .monitor-layout .monitor-panel:first-child {
    grid-column: span 2;
  }
}

@media (max-width: 768px) {
  .monitor-layout {
    grid-template-columns: 1fr;
  }

  .monitor-layout .monitor-panel:first-child {
    grid-column: span 1;
  }

  .monitor-summary {
    grid-template-columns: 1fr;
  }
}

/* ========== Security Cards ========== */
.capability-notice {
  margin-bottom: 12px;
}

.capability-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.tab-actions {
  margin-bottom: 16px;
}

.security-card :deep(.ant-card-head) {
  border-bottom: 1px solid rgba(0, 0, 0, 0.06);
  padding: 18px 24px;
  background: linear-gradient(180deg, #fafbfc 0%, #ffffff 100%);
}

.security-card :deep(.ant-card-body) {
  padding: 24px;
}

/* ========== Port Candidates ========== */
.port-candidates {
  margin-top: 10px;
}

.port-candidates-label {
  margin-bottom: 8px;
  font-size: 13px;
  color: rgba(0, 0, 0, 0.45);
  font-weight: 500;
}

.port-candidates :deep(.ant-tag) {
  cursor: pointer;
  margin-bottom: 6px;
  transition: all 0.2s;
  border-radius: 6px;
}

.port-candidates :deep(.ant-tag:hover) {
  transform: translateY(-1px);
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.1);
}

/* ========== Modal Actions ========== */
.modal-actions {
  display: flex;
  justify-content: flex-end;
  margin-top: 24px;
  padding-top: 20px;
  border-top: 1px solid rgba(0, 0, 0, 0.06);
}

/* ========== Misc ========== */
.masked {
  -webkit-text-security: disc;
  text-security: disc;
  font-family: 'SFMono-Regular', Consolas, monospace;
  letter-spacing: 2px;
}

/* ========== Dark Mode ========== */
:global(html.console-dark) .vps-detail-page {
  background: #0f1419;
  color: #f1f5f9;
}

:global(html.console-dark) .detail-header,
:global(html.console-dark) .ecs-tabs :deep(.ant-tabs-nav),
:global(html.console-dark) .overview-card,
:global(html.console-dark) .monitor-layout .monitor-panel {
  background: #1e2433;
  border-color: #2d3748;
  box-shadow: none;
}

:global(html.console-dark) .header-breadcrumb,
:global(html.console-dark) .meta-value,
:global(html.console-dark) .info-list-item:hover,
:global(html.console-dark) .info-list-icon,
:global(html.console-dark) .monitor-bar,
:global(html.console-dark) .gauge-mask {
  background: #161b28;
}

:global(html.console-dark) .header-breadcrumb,
:global(html.console-dark) .ecs-tabs :deep(.ant-tabs-nav),
:global(html.console-dark) .monitor-item,
:global(html.console-dark) .account-row-item {
  border-color: #2d3748;
}

:global(html.console-dark) .overview-card :deep(.ant-card-head),
:global(html.console-dark) .monitor-layout .monitor-panel :deep(.ant-card-head),
:global(html.console-dark) .security-card :deep(.ant-card-head) {
  background: linear-gradient(180deg, #1f2636 0%, #1b2232 100%);
  border-bottom-color: #2d3748;
}

:global(html.console-dark) .title-text,
:global(html.console-dark) .card-title,
:global(html.console-dark) .info-value,
:global(html.console-dark) .info-list-value,
:global(html.console-dark) .monitor-value,
:global(html.console-dark) .network-value-text,
:global(html.console-dark) .summary-value,
:global(html.console-dark) .account-row-value,
:global(html.console-dark) .meta-value {
  color: #f1f5f9;
}

:global(html.console-dark) .meta-label,
:global(html.console-dark) .meta-icon,
:global(html.console-dark) .info-label,
:global(html.console-dark) .info-list-label,
:global(html.console-dark) .monitor-label,
:global(html.console-dark) .monitor-icon,
:global(html.console-dark) .monitor-spec,
:global(html.console-dark) .network-label-text,
:global(html.console-dark) .network-unit-text,
:global(html.console-dark) .price-unit,
:global(html.console-dark) .account-row-label,
:global(html.console-dark) .summary-list,
:global(html.console-dark) .gauge-sub,
:global(html.console-dark) .port-candidates-label {
  color: #94a3b8;
}

:global(html.console-dark) .ecs-tabs :deep(.ant-tabs-tab) {
  color: #9fb0c7;
}

:global(html.console-dark) .ecs-tabs :deep(.ant-tabs-tab:hover),
:global(html.console-dark) .ecs-tabs :deep(.ant-tabs-tab-active),
:global(html.console-dark) .ecs-tabs :deep(.ant-tabs-tab-active .ant-tabs-tab-btn) {
  color: #60a5fa;
}

/* ========== Responsive ========== */
@media (max-width: 1200px) {
  .overview-grid { grid-template-columns: repeat(6, minmax(0, 1fr)); }
  .instance-card,
  .time-card { grid-column: span 6; }
  .monitor-card,
  .account-card { grid-column: span 6; }

  .info-list {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 768px) {
  .vps-detail-page {
    padding: 16px;
  }

  .overview-grid { grid-template-columns: 1fr; }
  .instance-card,
  .monitor-card,
  .account-card,
  .time-card { grid-column: span 1; }

  .header-main {
    flex-direction: column;
    padding: 16px;
  }

  .header-actions {
    width: 100%;
    justify-content: flex-start;
  }

  .instance-meta {
    font-size: 13px;
  }

  .info-list {
    grid-template-columns: 1fr;
  }

  .monitor-item {
    grid-template-columns: 1fr;
    gap: 8px;
  }

  .network-stats {
    gap: 24px;
  }

  .info-grid {
    grid-template-columns: 1fr;
  }

  .action-buttons {
    flex-direction: column;
  }

  .account-row-item {
    grid-template-columns: 1fr;
    gap: 4px;
  }
}

@media (max-width: 480px) {
  .header-breadcrumb {
    padding: 10px 16px;
  }

  .instance-title {
    flex-wrap: wrap;
  }

  .title-icon {
    font-size: 20px;
  }

  .title-text {
    font-size: 18px;
  }

  .title-status {
    margin-left: 0;
  }

  .instance-meta {
    gap: 4px;
  }

  .instance-meta :deep(.ant-divider-vertical) {
    display: none;
  }

  .header-actions {
    flex-wrap: wrap;
  }
}
</style>
