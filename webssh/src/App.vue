<template>
  <router-view />
</template>

<script setup lang="ts">

import { onBeforeMount, onBeforeUnmount, onMounted } from "vue";
import { useRouter } from "vue-router";
import axios from "axios";
import { useGlobalStore } from "./stores/store";


let router = useRouter();
let globalStore = useGlobalStore();

// 检查Wake Lock功能的支持情况
const isWakeLockSupported = "wakeLock" in navigator;

let wakeLock: WakeLockSentinel | null = null;

/**
 * 屏幕保持唤醒状态
 */
const requestScreenWakeLock = async () => {
  if (!isWakeLockSupported) return;
  
  try {
    wakeLock = await navigator.wakeLock.request("screen");
    // console.log("屏幕保持唤醒状态成功");
  } catch (error) {
    // 忽略 NotAllowedError，这是正常的行为
    if ((error as Error).name !== 'NotAllowedError') {
      console.error(`WakeLock 错误：${(error as Error).message}`);
    }
  }
};

/**
 * 处理页面可见性变化
 */
const handleVisibilityChange = async () => {
  if (document.visibilityState === 'visible') {
    // 页面变为可见时重新请求 wake lock
    await requestScreenWakeLock();
  }
};

onMounted(() => {
  // 添加页面可见性变化监听
  document.addEventListener('visibilitychange', handleVisibilityChange);
  
  // 初始请求 wake lock
  if (document.visibilityState === 'visible') {
    requestScreenWakeLock();
  }
});

/**
 * 检查系统是否已经初始化及运行模式
 */
onBeforeMount(async () => {
  await axios.get<{ code: number; msg: string; data: { is_init: boolean } }>("/api/sys/is_init")
    .then((res) => {
      if (res.data.code === 0) {
        if (!res.data.data.is_init) {
          // 如果系统没有进行过初始化操作,进入初始化页面
          globalStore.isInit = false;
          globalStore.logout();
          localStorage.clear();
          router.push({ "name": "SysInit" })
          return;
        }
      } 
    }).catch((err) => {
      console.log("获取系统初始化状态异常:" + err);
    });
});

onBeforeUnmount(() => {
  // 移除页面可见性变化监听
  document.removeEventListener('visibilitychange', handleVisibilityChange);
  
  // 释放 wake lock
  if (wakeLock) {
    wakeLock.release().then(() => {
      wakeLock = null;
    }).catch((error) => {
      console.error(`释放 WakeLock 错误：${error}`);
    });
  }
});

</script>

<style>
.el-header,
.el-footer {
  background-color: #2b75d6;
}

</style>