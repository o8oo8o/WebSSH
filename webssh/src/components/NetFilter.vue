<template>
  <el-tab-pane label="访问控制" name="netFilter">
    <!-- ========================================= -->
    <el-card>
      <el-row>
        <el-col :span="8" style="text-align: right;height: 24px;">
          <el-form-item label="搜索:">
            <el-input v-model="searchNetFilterName" style="max-width: 400px" placeholder="请输入策略名称"
              class="input-with-select">
              <template #prepend>
                <el-select v-model="searchNetFilterType" placeholder="选择策略" style="width: 115px">
                  <el-option label="全部" value="ALL" />
                  <el-option label="允许" value="Y" />
                  <el-option label="拒绝" value="N" />
                </el-select>
              </template>
            </el-input>
          </el-form-item>
        </el-col>
        <el-col :span="2"></el-col>
        <!-- <el-col :span="6"></el-col> -->
        <el-col :span="6" style="height: 24px;">
          <el-form-item label="默认策略">
            <el-radio-group v-model="data.policy_conf">
              <el-radio value="N">拒绝</el-radio>
              <el-radio value="Y">允许</el-radio>
            </el-radio-group>
          </el-form-item>
        </el-col>
        <el-col :span="2" style="height: 24px;">
          <el-form-item label="">
            <el-popconfirm confirmButtonText="确定" cancelButtonText="取消" icon="el-icon-info" iconColor="red"
              title="确定修改策略,修改后可能导致不能连接?" @confirm="updatePolicyConf">
              <template #reference>
                <el-button type="danger">保存策略</el-button>
              </template>
            </el-popconfirm>
          </el-form-item>
        </el-col>
      </el-row>
      <el-row style="margin-top: 20px">
        <el-table :data="filterHostTable" style="width: 100%" :show-overflow-tooltip="true">
          <el-table-column fixed sortable prop="name" label="名称" width="200"></el-table-column>
          <el-table-column sortable prop="cidr" label="CIDR"></el-table-column>
          <el-table-column sortable prop="net_policy" label="策略">
            <template #default="scope">
              <div v-if="scope.row.net_policy === 'Y'">
                <el-tag type="success">允许</el-tag>
              </div>
              <div v-else>
                <el-tag type="danger">拒绝</el-tag>
              </div>
            </template>
          </el-table-column>
          <el-table-column sortable prop="policy_no" label="策略编号" width="120"></el-table-column>
          <el-table-column sortable prop="expiry_at" label="过期时间" width="180"></el-table-column>
          <el-table-column fixed="right" label="操作">
            <template #header>
              <el-button type="primary" @click="addNetFilter">新增</el-button>
              <el-button type="primary" @click="getNetFilterList(0, 10000)">刷新</el-button>
            </template>
            <template #default="scope">
              <el-button type="success" @click="editNetFilter(scope.row)">编辑</el-button>
              <el-popconfirm confirmButtonText="删除" cancelButtonText="取消" icon="el-icon-info" iconColor="red"
                title="确定删除吗" @confirm="deleteNetFilterById(scope.row.id)">
                <template #reference>
                  <el-button type="danger">删除</el-button>
                </template>
              </el-popconfirm>

            </template>
          </el-table-column>
        </el-table>
      </el-row>
    </el-card>
    <!-- ========================================= -->
    <el-dialog title="策略配置" v-model="data.net_filter_dialog_visible" width="60%">
      <el-form label-width="80px">
        <el-form-item label="名称" prop="name">
          <el-input minlength="1" maxlength="60" v-model.trim="netFilter.name" show-word-limit
            placeholder="请输入名称"></el-input>
        </el-form-item>
        <el-form-item label="CIDR" prop="cidr">
          <el-input v-model.trim="netFilter.cidr" minlength="1" maxlength="90" show-word-limit
            placeholder="请输入CIDR,举例:192.168.1.0/24 或 fe80::acbc:1234:5678::/64"></el-input>
        </el-form-item>
        <el-form-item label="策略">
          <el-radio-group v-model="netFilter.net_policy">
            <el-radio value="Y">允许</el-radio>
            <el-radio value="N">拒绝</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="策略编号">
          <el-input-number v-model="netFilter.policy_no" :min="1" :max="65535" />
        </el-form-item>
        <el-form-item label="过期时间">
          <el-date-picker v-model="netFilter.expiry_at" type="datetime" placeholder="选择策略过期时间" />
        </el-form-item>
      </el-form>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="data.net_filter_dialog_visible = false">取消</el-button>
          <el-button type="success" @click="saveNetFilter">保存</el-button>
        </span>
      </template>
    </el-dialog>
    <!-- ========================================= -->
  </el-tab-pane>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { ElMessage, dayjs } from "element-plus";
import axios from "axios";


enum Mode {
  "create" = 0,
  "update" = 1,
}

/**
 * 网络策略
 */

interface NetFilter {
  id: number;
  name: string;
  cidr: string;
  net_policy: "Y" | "N";
  policy_no: number;
  expiry_at: Date;
}

interface NetFilterInfo extends NetFilter {
  created_at?: string;
  updated_at?: string;
}

interface ResponseData {
  code: number;
  msg: string;
  data?: any
}

let data = reactive({
  mode: Mode.create,
  net_filter_dialog_visible: false,
  net_filter_list: Array<NetFilter>(),
  policy_conf: "N",
});

let netFilter = reactive<NetFilter>({
  id: 0,
  name: "",
  cidr: "",
  net_policy: "N",
  policy_no: 32768,
  expiry_at: new Date(2050, 0, 0, 0, 0, 0)
});

/**
 * 搜索
 */
const searchNetFilterName = ref("")
const searchNetFilterType = ref("ALL")
const filterHostTable = computed(() =>
  data.net_filter_list.filter(
    (i) => {
      if (searchNetFilterType.value === "ALL") {
        return !searchNetFilterName.value || i.name.toLowerCase().includes(searchNetFilterName.value.toLowerCase());
      }
      return i.net_policy === searchNetFilterType.value && i.name.toLowerCase().includes(searchNetFilterName.value.toLowerCase());
    }
  )
)

/**
 * 获取列表
 */
function getNetFilterList(offset: number = 0, limit: number = 10000) {
  axios.get<ResponseData>(`/api/net_filter?offset=${offset}&limit=${limit}`)
    .then((ret) => {
      if (ret.data.code === 0) {
        data.net_filter_list = ret.data.data;
      } else {
        ElMessage.error("获取策略列表错误");
      }
    })
}

/**
 * 添加策略
 */
function addNetFilter() {
  netFilter.id = 0;
  netFilter.name = "";
  netFilter.cidr = "";
  netFilter.net_policy = "Y";
  netFilter.policy_no = 32768;
  netFilter.expiry_at = new Date(2050, 0, 0, 0, 0, 0);
  data.net_filter_dialog_visible = true;

}

/**
 * 编辑策略
 * @param n 
 */
function editNetFilter(n: NetFilter) {
  netFilter.id = n.id;
  netFilter.name = n.name;
  netFilter.cidr = n.cidr;
  netFilter.net_policy = n.net_policy;
  netFilter.policy_no = n.policy_no;
  netFilter.expiry_at = n.expiry_at;
  data.net_filter_dialog_visible = true;
}

/**
 * 获取请求体
 */
function getRequestBody(): any {
  return {
    id: netFilter.id,
    name: netFilter.name,
    cidr: netFilter.cidr,
    net_policy: netFilter.net_policy,
    policy_no: netFilter.policy_no,
    expiry_at: dayjs(netFilter.expiry_at).format("YYYY-MM-DD HH:mm:ss"),
  }
}

/**
 * 根据ID更新策略
 */
function updateNetFilterById() {
  axios.put<ResponseData>(`/api/net_filter`, getRequestBody())
    .then((ret) => {
      if (ret.data.code === 0) {
        data.net_filter_list = ret.data.data;
      } else {
        ElMessage.error("更新策略错误,请检查输入");
      }
    })
}

/**
 * 根据ID删除策略
 */
function deleteNetFilterById(id: number) {
  axios.delete<ResponseData>(`/api/net_filter/${id}`)
    .then((ret) => {
      if (ret.data.code === 0) {
        data.net_filter_list = ret.data.data;
      } else {
        ElMessage.error("删除策略错误");
      }
    })
}

/**
 * 创建策略
 */
function createNetFilter() {
  axios.post<ResponseData>(`/api/net_filter`, getRequestBody())
    .then((ret) => {
      if (ret.data.code === 0) {
        data.net_filter_list = ret.data.data;
      } else {
        ElMessage.error("创建策略错误,请检查输入");
      }
    })
}

/**
 * 保存配置
 */
function saveNetFilter() {
  if (netFilter.id === 0) {
    createNetFilter();
  } else {
    updateNetFilterById();
  }
  netFilter.id = 0;
  netFilter.name = "";
  netFilter.cidr = "";
  netFilter.net_policy = "Y";
  netFilter.policy_no = 32768;
  netFilter.expiry_at = new Date(2050, 0, 0, 0, 0, 0);
  data.net_filter_dialog_visible = false;
}

/**
 * 获取策略配置
 */
function getPolicyConf() {
  axios.get<ResponseData>(`/api/policy_conf/1`)
    .then((ret) => {
      if (ret.data.code === 0) {
        data.policy_conf = ret.data.data.net_policy;
      } else {
        ElMessage.error("获取策略配置错误");
      }
    })
}

/**
 * 更新策略
 */
function updatePolicyConf() {
  axios.put<ResponseData>(`/api/policy_conf`, { id: 1, net_policy: data.policy_conf })
    .then((ret) => {
      if (ret.data.code === 0) {
        data.policy_conf = ret.data.data.net_policy;
        ElMessage.success("更新策略成功");
      } else {
        ElMessage.error("更新策略失败");
      }
    })
}

onMounted(() => {
  getPolicyConf();
  getNetFilterList();
});

</script>


<style scoped></style>