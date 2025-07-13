<template>
  <el-container class="manage-container">
    <el-main>
      <div>
        <el-tabs v-model="data.active_name" type="card">
          <Connect></Connect>
          <Account></Account>
          <NetFilter></NetFilter>
          <LoginAudit></LoginAudit>
          <SshdUser></SshdUser>
          <SshdCert></SshdCert>
        </el-tabs>
      </div>
    </el-main>
  </el-container>
</template>

<script setup lang="ts">
import { reactive } from "vue";
import { useRouter } from "vue-router";
import { useGlobalStore } from "@/stores/store";
import Account from "@/components/Account.vue";
import Connect from "@/components/Connect.vue";
import NetFilter from "@/components/NetFilter.vue";
import LoginAudit from "@/components/LoginAudit.vue";
import SshdUser from "@/components/SshdUser.vue";
import SshdCert from "@/components/SshdCert.vue";

let router = useRouter();
let globalStore = useGlobalStore();

let data = reactive({
  active_name: "connectInfo",
});

/**
 * 回到主页
 */
function goHome() {
  router.push({ name: "Home" })
}

/**
 * 退出登陆
 */
function logout() {
  globalStore.logout();
  router.push({ "name": "Login" });
}

</script>

<style scoped>
.manage-container {
  background-color: #fff;
}
</style>
