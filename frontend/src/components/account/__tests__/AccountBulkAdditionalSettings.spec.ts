import { mount, flushPromises } from '@vue/test-utils'
import { describe,it,expect,vi } from 'vitest'
import Fields from '../AccountBulkAdditionalSettings.vue'
import Select from '@/components/common/Select.vue'
import type { Account } from '@/types'
vi.mock('vue-i18n',()=>({useI18n:()=>({t:(k:string)=>k,locale:{value:'zh'}})}))
describe('批量新增配置覆盖',()=>{
 it('覆盖双平台Vertex和平台专用选项，不回填秘密',async()=>{
  const vertex=mount(Fields,{props:{accounts:[{id:1,platform:'anthropic',type:'service_account',credentials:{project_id:'project',location:'us-east5',service_account_json:'synthetic'}}] as Account[],platforms:['anthropic'],types:['service_account']}})
  await flushPromises()
  expect(vertex.text()).toContain('Vertex 区域')
  expect(vertex.text()).toContain('服务账号 JSON')
  expect((vertex.get('input[type="password"]').element as HTMLInputElement).value).toBe('')
  vertex.unmount()
  const qoder=mount(Fields,{props:{accounts:[{id:2,platform:'qoder',type:'cosy'}] as Account[],platforms:['qoder'],types:['cosy']}})
  await flushPromises();expect(qoder.text()).toContain('Qoder 刷新令牌');qoder.unmount()
 })
 it('相同值回填，不同值留空，只提交勾选及明确清空',async()=>{
  const accounts=[{id:1,name:'A',notes:'same',platform:'openai',type:'apikey',credentials:{pool_mode:true},extra:{quota_limit:10}},{id:2,name:'B',notes:'same',platform:'openai',type:'apikey',credentials:{pool_mode:false},extra:{quota_limit:10}}] as Account[]
  const w=mount(Fields,{props:{accounts,platforms:['openai'],types:['apikey']}});await flushPromises()
  const vm=w.vm as unknown as {patch:()=>Record<string,unknown>}
  expect(vm.patch()).toEqual({})
  const blocks=w.findAll('.grid > div');const notes=blocks.find(b=>b.text().includes('备注'))!
  expect((notes.get('input[type="text"]').element as HTMLInputElement).value).toBe('same')
  const pool=blocks.find(b=>b.text().includes('上游号池模式'))!
  expect(pool.findComponent(Select).props('modelValue')).toBe('')
  await notes.findAll('input[type="checkbox"]')[0].setValue(true)
  expect(vm.patch()).toEqual({notes:'same'})
  await notes.findAll('input[type="checkbox"]')[1].setValue(true)
  expect(vm.patch()).toEqual({notes:''});w.unmount()
 })
})
