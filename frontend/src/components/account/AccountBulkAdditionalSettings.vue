<template>
  <section class="space-y-4 border-t-2 border-[color:var(--bh-ink)] pt-5" data-testid="bulk-additional-settings">
    <h3 class="text-xl font-extrabold text-bh-blue dark:text-blue-300">{{ zh ? '更多账号配置' : 'More account settings' }}</h3>
    <p class="text-sm">{{ zh ? '相同值回填，不同值留空；只修改勾选项。空值不会自动清除，请勾选“清空”。' : 'Shared values are shown; differing values stay blank. Only selected fields change. Choose Clear to erase a value.' }}</p>
    <div class="grid gap-4 sm:grid-cols-2">
      <div v-for="field in visible" :key="field.section+field.key" :class="field.kind==='json' && 'sm:col-span-2'">
        <label class="mb-2 flex items-center gap-2 font-bold"><input v-model="enabled[field.key]" type="checkbox" :disabled="locked" />{{ zh ? field.label : field.en }}</label>
        <Select v-if="field.kind==='boolean' || field.options" v-model="values[field.key]" :options="options(field)" :placeholder="' '" :disabled="locked || !enabled[field.key] || clear[field.key]" />
        <textarea v-else-if="field.kind==='json'" v-model="values[field.key]" rows="3" class="input w-full font-mono" :disabled="locked || !enabled[field.key] || clear[field.key]" />
        <input v-else v-model="values[field.key]" :type="field.kind==='secret'?'password':field.kind==='datetime'?'datetime-local':field.kind==='number'?'number':'text'" :min="field.min" :max="field.max" :step="field.kind==='number'?'any':undefined" autocomplete="off" class="input w-full" :disabled="locked || !enabled[field.key] || clear[field.key]" />
        <label v-if="field.kind!=='boolean' && field.key!=='name'" class="mt-1 flex items-center gap-2 text-sm"><input v-model="clear[field.key]" type="checkbox" :disabled="locked || !enabled[field.key]" />{{ zh ? '清空' : 'Clear' }}</label>
      </div>
    </div>
  </section>
</template>
<script setup lang="ts">
import { computed, reactive, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Select from '@/components/common/Select.vue'
import { bulkAdditionalFields, commonAccountValue, type BulkAdditionalField } from './bulkAdditionalFields'
import type { Account, AccountPlatform, AccountType } from '@/types'
const props=defineProps<{accounts:Account[];platforms:AccountPlatform[];types:AccountType[];locked?:boolean}>()
const {locale}=useI18n();const zh=computed(()=>!locale?.value?.startsWith('en'))
const enabled=reactive<Record<string,boolean>>({}),clear=reactive<Record<string,boolean>>({}),values=reactive<Record<string,string>>({})
watch(()=>props.accounts.map(a=>a.id).join(','),()=>{for(const key of Object.keys(enabled))delete enabled[key];for(const key of Object.keys(clear))delete clear[key]})
const visible=computed(()=>bulkAdditionalFields.filter(f=>(!f.platforms||props.platforms.length>0&&props.platforms.every(p=>f.platforms!.includes(p)))&&(!f.types||props.types.length>0&&props.types.every(ty=>f.types!.includes(ty)))))
const options=(f:BulkAdditionalField)=>f.kind==='boolean'?[{value:'true',label:zh.value?'开启':'On'},{value:'false',label:zh.value?'关闭':'Off'}]:(f.options||[]).map(value=>({value,label:value}))
watch(()=>props.accounts,()=>{for(const f of bulkAdditionalFields){if(enabled[f.key])continue;const v=f.kind==='secret'?'':commonAccountValue(props.accounts,f.section,f.key);values[f.key]=v===''?'':f.kind==='datetime'?new Date(typeof v==='number'?v*1000:String(v)).toLocaleString('sv-SE').replace(' ','T').slice(0,16):typeof v==='object'?JSON.stringify(v,null,2):String(v)}},{immediate:true})
const hasChanges=computed(()=>visible.value.some(f=>enabled[f.key]))
// 数组规则和对象配置保持原持久化形状，不能把“清空数组”写成空对象。
const arrayFields = new Set(['pool_mode_retry_status_codes', 'temp_unschedulable_rules', 'protocol_rules'])
function patch(){
 const out:Record<string,unknown>={},credentials:Record<string,unknown>={},extra:Record<string,unknown>={}
 for(const f of visible.value){if(!enabled[f.key])continue;const text=(values[f.key]||'').trim();let v:unknown
  if(clear[f.key])v=f.key==='expires_at'?0:f.kind==='json'?(arrayFields.has(f.key)?[]:{}):f.section==='extra'?null:''
  else{if(!text)throw new Error((zh.value?f.label:f.en)+(zh.value?'：请填写值':' requires a value'));v=text
   if(f.kind==='boolean')v=text==='true'
   if(f.kind==='number'){v=Number(text);if(!Number.isFinite(v)||f.min!==undefined&&Number(v)<f.min||f.max!==undefined&&Number(v)>f.max)throw new Error(f.label)}
   if(f.kind==='json'){
    v=JSON.parse(text)
    if(arrayFields.has(f.key)?!Array.isArray(v):!v||typeof v!=='object'||Array.isArray(v))throw new Error((zh.value?f.label:f.en)+(zh.value?'：JSON 类型不正确':': Invalid JSON type'))
   }
   if(f.kind==='datetime'){v=Math.floor(new Date(text).getTime()/1000);if(!Number.isFinite(v))throw new Error(f.label)}
  }
  if(f.section==='account')out[f.key]=v;else if(f.section==='credentials')credentials[f.key]=v;else extra[f.key]=v
 }
 if(Object.keys(credentials).length)out.credentials=credentials;if(Object.keys(extra).length)out.extra=extra;return out
}
defineExpose({patch,hasChanges})
</script>
