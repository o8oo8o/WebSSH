<template>
  <div class="login-container">
    <div class="login-box">
      <div class="login-header">
        <h1>欢迎登录</h1>
        <p>Web Terminal Management System</p>
      </div>

      <div class="login-form">
        <div class="form-item">
          <el-input v-model="data.name" placeholder="请输入用户名" :prefix-icon="User" trim minlength="1" maxlength="64"
            show-word-limit clearable />
        </div>

        <div class="form-item">
          <el-input v-model="data.pwd" placeholder="请输入密码" :prefix-icon="Lock" type="password" trim minlength="3"
            maxlength="64" show-word-limit show-password clearable />
        </div>

        <div class="form-item">
          <el-button type="primary" class="login-button" @click="login">
            登录
          </el-button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive } from "vue";
import { useRouter } from "vue-router";
import { ElMessage } from "element-plus";
import { User, Lock } from '@element-plus/icons-vue';
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

<style scoped>
.login-container {
  min-height: 100vh;
  height: 100vh;
  background: linear-gradient(135deg, #1a365d 0%, #2d3748 100%);
  display: flex;
  justify-content: center;
  align-items: flex-start;
  padding: 20px;
  overflow: hidden;
  box-sizing: border-box;
}

.login-box {
  width: 100%;
  max-width: 420px;
  background: rgba(255, 255, 255, 0.98);
  border-radius: 20px;
  box-shadow: 0 10px 40px rgba(0, 0, 0, 0.2);
  padding: 40px 30px;
  backdrop-filter: blur(10px);
  margin-top: 10vh;
}

.login-header {
  text-align: center;
  margin-bottom: 25px;
}

.login-header h1 {
  color: #2d3748;
  font-size: clamp(1.8em, 4vw, 2.2em);
  font-weight: 600;
  margin: 0 0 10px 0;
}

.login-header p {
  color: #718096;
  font-size: clamp(0.9em, 2vw, 1.1em);
  margin: 0;
}

.login-form {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.form-item {
  width: 100%;
}

:deep(.el-input__wrapper) {
  background-color: #f8fafc;
  border: 2px solid transparent;
  border-radius: 12px;
  transition: all 0.3s ease;
  box-shadow: none !important;
  padding: 0 15px;
  height: 45px;
}

:deep(.el-input__wrapper:hover),
:deep(.el-input__wrapper.is-focus) {
  border-color: #4c51bf;
  background-color: #ffffff;
}

:deep(.el-input__inner) {
  height: 45px;
  font-size: 16px;
  color: #2d3748;
}

:deep(.el-input__prefix) {
  font-size: 1.2em;
  margin-right: 10px;
  color: #718096;
}

:deep(.el-input__inner::placeholder) {
  color: #a0aec0;
  opacity: 1;
  /* 修复Firefox中placeholder颜色过浅的问题 */
}

:deep(.el-input__inner::-webkit-input-placeholder) {
  color: #a0aec0;
}

:deep(.el-input__inner::-moz-placeholder) {
  color: #a0aec0;
}

:deep(.el-input__inner:-ms-input-placeholder) {
  color: #a0aec0;
}

.login-button {
  width: 100%;
  height: 45px;
  font-size: 16px;
  font-weight: 500;
  letter-spacing: 2px;
  background: -webkit-gradient(linear, left top, right top, from(#4c51bf), to(#667eea));
  background: -o-linear-gradient(left, #4c51bf, #667eea);
  background: linear-gradient(to right, #4c51bf, #667eea);
  border: none;
  border-radius: 12px;
  -webkit-transition: all 0.3s ease;
  -o-transition: all 0.3s ease;
  transition: all 0.3s ease;
  cursor: pointer;
  -webkit-appearance: none;
  -moz-appearance: none;
  appearance: none;
}

.login-button:hover {
  -webkit-transform: translateY(-2px);
  -ms-transform: translateY(-2px);
  transform: translateY(-2px);
  -webkit-box-shadow: 0 6px 20px rgba(76, 81, 191, 0.3);
  box-shadow: 0 6px 20px rgba(76, 81, 191, 0.3);
}

.login-button:active {
  -webkit-transform: translateY(0);
  -ms-transform: translateY(0);
  transform: translateY(0);
}

/* 响应式布局 */
@media screen and (max-width: 768px) {
  .login-container {
    padding: 15px;
  }

  .login-box {
    padding: 25px 20px;
    width: 90%;
    margin-top: 8vh;
  }

  :deep(.el-input__wrapper) {
    height: 40px;
  }

  .login-button {
    height: 40px;
    font-size: 15px;
  }
}

/* 平板设备适配 */
@media screen and (min-width: 768px) and (max-width: 1024px) {
  .login-box {
    max-width: 480px;
    padding: 35px 25px;
  }
}

/* 横屏手机适配 */
@media screen and (max-height: 480px) {
  .login-container {
    align-items: flex-start;
    padding: 10px;
  }

  .login-box {
    padding: 20px 15px;
    margin-top: 5vh;
  }

  .login-header {
    margin-bottom: 20px;
  }

  .login-form {
    gap: 15px;
  }
}

/* 确保在小屏幕设备上内容不会溢出 */
@media screen and (max-height: 600px) {
  .login-box {
    margin-top: 5vh;
    padding: 20px 15px;
  }
}

/* 暗色模式支持 */
@media (prefers-color-scheme: dark) {
  .login-box {
    background: rgba(255, 255, 255, 0.95);
  }

  :deep(.el-input__wrapper) {
    background-color: rgba(248, 250, 252, 0.95);
  }

  .login-header h1 {
    color: #1a202c;
  }

  .login-header p {
    color: #4a5568;
  }
}

/* 兼容性补充 */
@supports not (gap: 25px) {
  .login-form {
    margin: -12.5px 0;
  }

  .login-form>* {
    margin: 12.5px 0;
  }
}

@supports not (backdrop-filter: blur(10px)) {
  .login-box {
    background: rgba(255, 255, 255, 0.98);
  }
}
</style>
