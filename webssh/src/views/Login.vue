<template>
  <el-container>
    <el-header>
      <h1 style="text-align: center;">欢迎登陆</h1>
    </el-header>
    <el-main>
      <el-row>
        <el-col :span="8"></el-col>
        <el-col :span="8">
          <div>
            <el-divider />
            <el-form>
              <div>
                <div>
                  <el-form-item>
                    <el-card style="width: 100%;">
                      <el-input v-model="data.name" trim minlength="1" maxlength="64" show-word-limit clearable
                        placeholder="请输入用户名">
                        <template #prepend>账号</template>
                      </el-input>
                    </el-card>
                  </el-form-item>

                  <el-form-item>
                    <el-card style="width: 100%;">
                      <el-input v-model="data.pwd" trim type="password" minlength="3" maxlength="64" show-word-limit
                        show-password clearable placeholder="请输入密码">
                        <template #prepend>密码</template>
                      </el-input>
                    </el-card>
                  </el-form-item>

                  <el-form-item>
                    <el-card style="width: 100%;">
                      <div style="text-align: center">
                        <el-button @click="login"
                          type="primary">登&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;陆</el-button>
                      </div>
                    </el-card>
                  </el-form-item>
                </div>
              </div>
            </el-form>
          </div>
        </el-col>
      </el-row>
    </el-main>
  </el-container>
</template>

<script setup lang="ts">
import { reactive } from "vue";
import { useRouter } from "vue-router";
import { ElMessage } from "element-plus";
import axios from "axios";
import { useGlobalStore } from "@/stores/store";

let router = useRouter();
let globalStore = useGlobalStore();

let data = reactive({
  name: "",
  pwd: "",
})

interface ResponseData {
  code: number;
  msg: string;
  token: string;
  is_admin: "Y" | "N";
  is_root: "Y" | "N";
  user_name: string;
  user_desc: string;
  user_expiry_at: string;
}

/**
 * 登陆系统
 */
function login() {
  if (data.name.trim().length < 2) {
    ElMessage.error("用户名至少两个字符")
    return
  }

  if (data.pwd.trim().length < 2) {
    ElMessage.error("密码至少两个字符")
    return
  }

  axios.post<ResponseData>("/api/login", data).then((ret) => {
    if (ret.data.code === 0) {
      ElMessage.success("登陆成功");
      localStorage.setItem("token", ret.data.token);
      localStorage.setItem("auth", "yes");
      let res = ret.data;
      globalStore.login(res.is_admin, res.is_root, res.user_name, res.user_desc, res.user_expiry_at);
      router.push({ name: "Home" });
    } else {
      ElMessage.error("登陆失败");
    }
  }).catch(() => {
    ElMessage.error("登陆失败");
  })
}

</script>
