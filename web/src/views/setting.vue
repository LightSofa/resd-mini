<template>
  <div class="h-full relative p-5 overflow-y-auto [&::-webkit-scrollbar]:hidden" :key="renderKey">
    <NTabs type="line" animated v-model:value="activeTab">
      <NTabPane name="basic" :tab="t('setting.basic_setting')">
        <NForm
            :model="formValue"
            size="medium"
            label-placement="left"
            label-width="auto"
            require-mark-placement="right-hanging"
            class="w-[700px]"
        >
          <NFormItem :label="t('setting.save_dir')" path="SaveDirectory">
            <NInput :value="formValue.SaveDirectory" :placeholder="t('setting.save_dir')"/>
            <NButton strong secondary type="primary" @click="selectDir" class="ml-1">{{ t('common.select') }}</NButton>
          </NFormItem>

          <NFormItem :label="t('setting.filename_rules')" path="FilenameLen">
            <NInputNumber v-model:value="formValue.FilenameLen" :min="0" :max="9999" placeholder="0"/>
            <NSwitch v-model:value="formValue.FilenameTime" class="ml-1"></NSwitch>
            <NTooltip trigger="hover">
              <template #trigger>
                <NIcon size="18" class="ml-1 text-gray-500">
                  <HelpCircleOutline/>
                </NIcon>
              </template>
              {{ t("setting.filename_rules_tip") }}
            </NTooltip>
          </NFormItem>

          <NFormItem :label="t('setting.quality')" path="Quality">
            <NSelect v-model:value="formValue.Quality" :options="options"/>
            <NTooltip trigger="hover">
              <template #trigger>
                <NIcon size="18" class="ml-1 text-gray-500">
                  <HelpCircleOutline/>
                </NIcon>
              </template>
              {{ t("setting.quality_tip") }}
            </NTooltip>
          </NFormItem>

          <NFormItem :label="t('setting.auto_proxy')" path="AutoProxy">
            <NSwitch v-model:value="formValue.AutoProxy"/>
            <NTooltip trigger="hover">
              <template #trigger>
                <NIcon size="18" class="ml-1 text-gray-500">
                  <HelpCircleOutline/>
                </NIcon>
              </template>
              {{ t("setting.auto_proxy_tip") }}
            </NTooltip>
          </NFormItem>

          <NFormItem :label="t('setting.full_intercept')" path="WxAction">
            <NSwitch v-model:value="formValue.WxAction"/>
            <NTooltip trigger="hover">
              <template #trigger>
                <NIcon size="18" class="ml-1 text-gray-500">
                  <HelpCircleOutline/>
                </NIcon>
              </template>
              {{ t("setting.full_intercept_tip") }}
            </NTooltip>
          </NFormItem>

          <NFormItem :label="t('setting.insert_tail')" path="InsertTail">
            <NSwitch v-model:value="formValue.InsertTail"/>
            <NTooltip trigger="hover">
              <template #trigger>
                <NIcon size="18" class="ml-1 text-gray-500">
                  <HelpCircleOutline/>
                </NIcon>
              </template>
              {{ t("setting.insert_tail_tip") }}
            </NTooltip>
          </NFormItem>
        </NForm>
      </NTabPane>

      <NTabPane name="advanced" :tab="t('setting.advanced_setting')">
        <NForm
            :model="formValue"
            size="medium"
            label-placement="left"
            label-width="auto"
            require-mark-placement="right-hanging"
            class="w-[700px]"
        >
          <NFormItem label="Host" path="Host" :validation-status="hostValidationFeedback==='' ? undefined : 'error'" :feedback="hostValidationFeedback">
            <NInput v-model:value="formValue.Host" placeholder="127.0.0.1"/>
            <NTooltip trigger="hover">
              <template #trigger>
                <NIcon size="18" class="ml-1 text-gray-500">
                  <HelpCircleOutline/>
                </NIcon>
              </template>
              {{ t("setting.restart_tip") }}
            </NTooltip>
          </NFormItem>

          <NFormItem label="Port" path="Port" :validation-status="portValidationFeedback==='' ? undefined : 'error'" :feedback="portValidationFeedback">
            <NInput v-model:value="formValue.Port" placeholder="8899"/>
            <NTooltip trigger="hover">
              <template #trigger>
                <NIcon size="18" class="ml-1 text-gray-500">
                  <HelpCircleOutline/>
                </NIcon>
              </template>
              {{ t("setting.restart_tip") }}
            </NTooltip>
          </NFormItem>

          <NFormItem :label="t('setting.upstream_proxy')" path="UpstreamProxy">
            <NInput v-model:value="formValue.UpstreamProxy" placeholder="http://127.0.0.1:7890"/>
            <NSwitch v-model:value="formValue.OpenProxy" class="ml-1"/>
            <NTooltip trigger="hover">
              <template #trigger>
                <NIcon size="18" class="ml-1 text-gray-500">
                  <HelpCircleOutline/>
                </NIcon>
              </template>
              {{ t("setting.upstream_proxy_tip") }}
            </NTooltip>
          </NFormItem>

          <NFormItem :label="t('setting.download_proxy')" path="DownloadProxy">
            <NSwitch v-model:value="formValue.DownloadProxy"/>
            <NTooltip trigger="hover">
              <template #trigger>
                <NIcon size="18" class="ml-1 text-gray-500">
                  <HelpCircleOutline/>
                </NIcon>
              </template>
              {{ t("setting.download_proxy_tip") }}
            </NTooltip>
          </NFormItem>

          <NFormItem :label="t('setting.connections')" path="TaskNumber">
            <NInputNumber v-model:value="formValue.TaskNumber" :min="2" :max="64"/>
            <NTooltip trigger="hover">
              <template #trigger>
                <NIcon size="18" class="ml-1 text-gray-500">
                  <HelpCircleOutline/>
                </NIcon>
              </template>
              {{ t("setting.connections_tip") }}
            </NTooltip>
          </NFormItem>

          <NFormItem :label="t('setting.down_number')" path="DownNumber">
            <NInputNumber v-model:value="formValue.DownNumber" :min="1" :max="10"/>
            <NTooltip trigger="hover">
              <template #trigger>
                <NIcon size="18" class="ml-1 text-gray-500">
                  <HelpCircleOutline/>
                </NIcon>
              </template>
              {{ t("setting.down_number_tip") }}
            </NTooltip>
          </NFormItem>

          <NFormItem label="UserAgent" path="UserAgent">
            <NInput v-model:value="formValue.UserAgent" placeholder="UserAgent"/>
            <NTooltip trigger="hover">
              <template #trigger>
                <NIcon size="18" class="ml-1 text-gray-500">
                  <HelpCircleOutline/>
                </NIcon>
              </template>
              {{ t("setting.user_agent_tip") }}
            </NTooltip>
          </NFormItem>

          <NFormItem label="Headers" path="Headers">
            <NInput v-model:value="formValue.UseHeaders" placeholder="User-Agent,Referer,Authorization,Cookie"/>
            <NTooltip trigger="hover">
              <template #trigger>
                <NIcon size="18" class="ml-1 text-gray-500">
                  <HelpCircleOutline/>
                </NIcon>
              </template>
              {{ t("setting.use_headers_tip") }}
            </NTooltip>
          </NFormItem>

          <NFormItem :label="t('setting.domain_rule')" path="DomainRule">
            <NInput
                v-model:value="formValue.Rule"
                type="textarea"
                rows="5"
                :placeholder="t('setting.domain_rule_tip')"
            />
            <NTooltip trigger="hover">
              <template #trigger>
                <NIcon size="18" class="ml-1 text-gray-500">
                  <HelpCircleOutline/>
                </NIcon>
              </template>
              {{ t("setting.domain_rule_tip") }}
            </NTooltip>
          </NFormItem>
          <NFormItem :label="t('setting.mime_map')" path="MimeMap">
            <NInput
                v-model:value="MimeMap"
                type="textarea"
                rows="11"
                placeholder='{"video/mp4": { "Type": "video","Suffix": ".mp4"}}'
            />
            <NTooltip trigger="hover">
              <template #trigger>
                <NIcon size="18" class="ml-1 text-gray-500">
                  <HelpCircleOutline/>
                </NIcon>
              </template>
              {{ t("setting.mime_map_tip") }}
            </NTooltip>
          </NFormItem>

          <NFormItem :label="t('setting.action_rule_type')">
            <NInput v-model:value="actionRuleInput" :placeholder="t('setting.action_rule_type_placeholder')"/>
            <NButton strong secondary type="primary" @click="upsertActionRule" class="ml-1">{{ t('setting.action_rule_apply') }}</NButton>
            <NButton strong secondary type="error" @click="removeActionRule" class="ml-1" v-if="selectedActionRuleKey">{{ t('setting.action_rule_remove') }}</NButton>
          </NFormItem>

          <NFormItem :label="t('setting.action_rule_current')">
            <NSelect
                style="width: 100%;"
                v-model:value="selectedActionRuleKey"
                :options="actionRuleOptions"
                :placeholder="t('setting.action_rule_current_placeholder')"
            />
          </NFormItem>

          <template v-if="selectedActionRuleKey">
            <NFormItem :label="t('setting.action_rule_enabled')">
              <NSwitch v-model:value="currentActionRule.Enabled"/>
            </NFormItem>

            <NFormItem :label="t('setting.action_rule_command')">
              <NInput
                  v-model:value="currentActionRule.Command"
                  type="textarea"
                  :rows="4"
                  :placeholder='`m3u8d-cli download --M3u8Url \"%url%\" --FileName \"%filename%\" --SaveDir \"%defaultDir%\"`'
              />
            </NFormItem>

            <NFormItem :label="t('setting.action_rule_placeholders')">
              <NButton quaternary class="mr-1" @click="insertPlaceholder('%url%')">%url%</NButton>
              <NButton quaternary class="mr-1" @click="insertPlaceholder('%filename%')">%filename%</NButton>
              <NButton quaternary class="mr-1" @click="insertPlaceholder('%defaultDir%')">%defaultDir%</NButton>
              <NButton quaternary class="mr-1" @click="insertPlaceholder('%suffix%')">%suffix%</NButton>
              <NButton quaternary class="mr-1" @click="insertPlaceholder('%classify%')">%classify%</NButton>
              <NButton quaternary class="mr-1" @click="insertPlaceholder('%domain%')">%domain%</NButton>
            </NFormItem>

            <NFormItem :label="t('setting.action_rule_timeout')">
              <NInputNumber v-model:value="currentActionRule.TimeoutSec" :min="5" :max="7200"/>
            </NFormItem>

            <NFormItem :label="t('setting.action_rule_async')">
              <NSwitch v-model:value="currentActionRule.RunAsync"/>
            </NFormItem>

            <NFormItem :label="t('setting.action_rule_success_tip')">
              <NInput v-model:value="currentActionRule.SuccessTip" :placeholder="t('setting.action_rule_success_tip_placeholder')"/>
            </NFormItem>

            <NFormItem :label="t('setting.action_rule_fail_tip')">
              <NInput v-model:value="currentActionRule.FailTip" :placeholder="t('setting.action_rule_fail_tip_placeholder')"/>
            </NFormItem>
          </template>
        </NForm>
      </NTabPane>
    </NTabs>
  </div>
</template>

<script lang="ts" setup>
import {HelpCircleOutline} from "@vicons/ionicons5"
import {ref, watch} from "vue"
import {useIndexStore} from "@/stores"
import type {appType} from "@/types/app"
import appApi from "@/api/app"
import {computed, onMounted} from "vue"
import {useI18n} from 'vue-i18n'
import {isValidHost, isValidPort} from '@/func'
import {NButton, NIcon} from "naive-ui"
import {useRoute} from "vue-router"

const {t} = useI18n()
const store = useIndexStore()
const route = useRoute()

const options = computed(() =>
    t("setting.quality_value")
        .split(",")
        .map((value, index) => ({ value: index, label: value }))
)

const formValue = ref<appType.Config>(Object.assign({}, store.globalConfig))
const activeTab = ref("basic")

const MimeMap = ref(formValue.value.MimeMap ? JSON.stringify(formValue.value.MimeMap, null, 2) : "")
const renderKey = ref(999)
const actionRuleInput = ref("")
const selectedActionRuleKey = ref("")

const hostValidationFeedback = ref("")
const portValidationFeedback = ref("")
const createDefaultActionRule = (): appType.ActionRule => ({
  Enabled: true,
  Command: "",
  TimeoutSec: 120,
  RunAsync: true,
  SuccessTip: "",
  FailTip: "",
})

const actionRuleOptions = computed(() => {
  const rules = formValue.value.ActionRules || {}
  return Object.keys(rules).map((k) => ({
    label: k,
    value: k
  }))
})

const currentActionRule = computed(() => {
  if (!selectedActionRuleKey.value) {
    return createDefaultActionRule()
  }
  if (!formValue.value.ActionRules) {
    formValue.value.ActionRules = {}
  }
  if (!formValue.value.ActionRules[selectedActionRuleKey.value]) {
    formValue.value.ActionRules[selectedActionRuleKey.value] = createDefaultActionRule()
  }
  return formValue.value.ActionRules[selectedActionRuleKey.value]
})

const normalizeRuleKey = (raw: string) => {
  const key = (raw || "").trim().toLowerCase()
  if (!key) {
    return ""
  }
  return key
}

const upsertActionRule = () => {
  const key = normalizeRuleKey(actionRuleInput.value)
  if (!key) {
    window?.$message?.error(t("setting.action_rule_type_empty"))
    return
  }
  if (!formValue.value.ActionRules) {
    formValue.value.ActionRules = {}
  }
  if (!formValue.value.ActionRules[key]) {
    formValue.value.ActionRules[key] = createDefaultActionRule()
  }
  selectedActionRuleKey.value = key
  actionRuleInput.value = key
}

const removeActionRule = () => {
  if (!selectedActionRuleKey.value || !formValue.value.ActionRules) {
    return
  }
  delete formValue.value.ActionRules[selectedActionRuleKey.value]
  selectedActionRuleKey.value = ""
  actionRuleInput.value = ""
}

const insertPlaceholder = (placeholder: string) => {
  const key = selectedActionRuleKey.value
  if (!key) {
    return
  }
  const current = formValue.value.ActionRules[key]?.Command || ""
  formValue.value.ActionRules[key].Command = `${current}${current ? " " : ""}${placeholder}`
}

watch(formValue.value, () => {
  formValue.value.Port = formValue.value.Port.trim()
  formValue.value.Host = formValue.value.Host.trim()

  if (!isValidHost(formValue.value.Host)) {
    hostValidationFeedback.value = t("setting.host_format_error")
    return
  } else {
    hostValidationFeedback.value = ''
  }

  if (!isValidPort(parseInt(formValue.value.Port))) {
    portValidationFeedback.value = t("setting.port_format_error")
    return
  } else {
    portValidationFeedback.value = ''
  }
  store.setConfig(formValue.value)
}, {deep: true})

watch(selectedActionRuleKey, () => {
  actionRuleInput.value = selectedActionRuleKey.value
})

watch(() => route.query, () => {
  if (route.query.section !== "action-rules") {
    return
  }
  activeTab.value = "advanced"
  const type = normalizeRuleKey(String(route.query.type || ""))
  if (!type) {
    return
  }
  actionRuleInput.value = type
  upsertActionRule()
}, {immediate: true})

watch(MimeMap, () => {
  store.setConfig({
    MimeMap: JSON.parse(MimeMap.value)
  })
})

watch(() => {
  return store.globalConfig.Theme
}, () => {
  formValue.value.Theme = store.globalConfig.Theme
})

watch(() => store.globalConfig.Locale, () => {
  formValue.value.Locale = store.globalConfig.Locale
  renderKey.value++
})

onMounted(() => {
  const keys = Object.keys(formValue.value.ActionRules || {})
  if (keys.length > 0 && !selectedActionRuleKey.value) {
    selectedActionRuleKey.value = keys[0]
  }
})

const selectDir = () => {
  appApi.openDirectoryDialog().then((res: any) => {
    if (res.code === 1) {
      formValue.value.SaveDirectory = res.data.folder
    }
  }).catch((err: any) => {
    window?.$message?.error(err)
  })
}
</script>
<style lang="scss">
.n-tabs-nav--top{
  @apply sticky top-0 z-10;
  background-color: var(--n-color);
}
</style>
