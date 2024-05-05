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

let wakeLock: WakeLockSentinel;

/**
 * 屏幕保持唤醒状态
 */
const requestScreenWakeLock = async () => {
  try {
    wakeLock = await navigator.wakeLock.request("screen");
    // console.log("屏幕保持唤醒状态成功");
  } catch (error) {
    let err = error as Error;
    console.error(`错误：${err.name}, 消息：${err.message}`);
  }
};

onMounted(() => {
  if (isWakeLockSupported) {
    requestScreenWakeLock().then((ret)=>{});
  }
})

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
  // if (isWakeLockSupported && wakeLock) {
  //   // 屏幕唤醒锁已释放
  //   wakeLock.release().then(() => {
  //     wakeLock = null as unknown as WakeLockSentinel;
  //     // console.log("屏幕唤醒锁已释放");
  //   });
  // }
});

</script>

<style>
.el-header,
.el-footer {
  background-color: #2b75d6;
}

/* tab 标签和终端之间的空白处理 */
#app>section>div>div>div.el-tabs__header.is-top {
  margin-bottom: 0px;
  margin-top: 0px;
  height: 30px;
  border-bottom: none;
}
</style>