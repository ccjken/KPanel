<script setup lang="ts">
import { computed, ref } from 'vue'
import MarkdownIt from 'markdown-it'
import DOMPurify from 'dompurify'
import { useI18n } from '@/i18n'
import { copyText } from '@/lib/clipboard'

const props=defineProps<{content:string}>()
const i18n=useI18n()
const copyFailed=ref(false)
const parser=new MarkdownIt({html:false,linkify:true,typographer:false,highlight:()=>''})
const fence=parser.renderer.rules.fence
if(fence)parser.renderer.rules.fence=(tokens,index,options,env,self)=>`<div class="ai-code-block"><button type="button" class="ai-code-copy" data-code-copy aria-label="复制代码">复制</button>${fence(tokens,index,options,env,self)}</div>`
const rendered=computed(()=>DOMPurify.sanitize(parser.render(props.content),{USE_PROFILES:{html:true},FORBID_TAGS:['style','iframe','form'],FORBID_ATTR:['style']}))
async function copyCode(event:MouseEvent){
  const target=(event.target as HTMLElement).closest<HTMLButtonElement>('[data-code-copy]');if(!target)return
  const code=target.parentElement?.querySelector('code')?.textContent||'';if(!code)return
  copyFailed.value=false;target.textContent='复制'
  if(!await copyText(code)){copyFailed.value=true;return}
  target.textContent='已复制';window.setTimeout(()=>{target.textContent='复制'},1200)
}
</script>
<template><div class="ai-markdown" @click="copyCode" v-html="rendered" /><p v-if="copyFailed" role="alert" class="ai-inline-error">{{i18n.t('ai.error.copy')}}</p></template>
