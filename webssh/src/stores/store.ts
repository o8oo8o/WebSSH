import { ref } from "vue"
import { defineStore } from "pinia"

// 系统初始化状态存储
export const useGlobalStore = defineStore(
  "global",
  () => {
    const isInit = ref<boolean>(false);
    const isLogin = ref<boolean>(false);
    const isAdmin = ref<string>("N");
    const isRoot = ref<string>("N");
    const userName = ref<string>("");
    const userDesc = ref<string>("");
    const userExpiryAt = ref<string>("");


    function reset() {
      isInit.value = false;
      isLogin.value = false;
      isAdmin.value = "N";
      isRoot.value = "N";
      userName.value = "";
      userDesc.value = "";
      userExpiryAt.value = "";
    }

    function login(
      is_admin: string,
      is_root: string,
      user_name: string,
      user_desc: string,
      user_expiry_at: string) {
      isLogin.value = true;
      isAdmin.value = is_admin;
      isRoot.value = is_root;
      userName.value = user_name;
      userDesc.value = user_desc;
      userExpiryAt.value = user_expiry_at;
    }

    function logout() {
      isLogin.value = false;
      isAdmin.value = "N";
      isRoot.value = "N";
      userName.value = "";
      userDesc.value = "";
      userExpiryAt.value = "";
      localStorage.clear();
    }

    return { isInit, isLogin, isAdmin, isRoot, userName, userDesc,userExpiryAt, login, logout, reset }
  },
  // 持久化到浏览器
  {
    persist: true,
  }
)
