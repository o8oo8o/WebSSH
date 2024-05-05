<template>
  <el-container>
    <el-header style="
        text-align: left;
        height: 24px;
        padding-left: 0px;
        padding-right: 0px;
      ">
      <el-row>
        <el-col :span="12">
          <el-button-group>
            <el-button type="primary" @click="goHome">回到主页</el-button>
          </el-button-group>
        </el-col>

        <el-col :span="12" style="text-align: right">
          <el-button-group>
            <el-button type="danger" @click="logout">退出</el-button>
          </el-button-group>
        </el-col>
      </el-row>
    </el-header>
    <el-main>
      <div>
        <el-tabs v-model="data.active_name" type="card">
          <Account></Account>
          <NetFilter></NetFilter>
          <LoginAudit></LoginAudit>
          <Connect></Connect>
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

let router = useRouter();
let globalStore = useGlobalStore();

let data = reactive({
  active_name: "accoutManage",
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