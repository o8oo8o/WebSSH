<template>
  <el-tab-pane label="Web登录审计" name="loginAudit">
    <!-- ========================================= -->
    <el-card>
      <el-row>
        <el-col :span="4" style="height: 24px;">
          <el-form-item label="登录状态">
            <el-radio-group v-model="data.is_success">
              <el-radio value="N">失败</el-radio>
              <el-radio value="Y">成功</el-radio>
            </el-radio-group>
          </el-form-item>
        </el-col>
        &nbsp;&nbsp;&nbsp;&nbsp;
        <el-col :span="8" style="height: 24px;">
          <el-form-item label="时间范围">
            <el-date-picker v-model="time_range" type="datetimerange" range-separator="To" start-placeholder="起始时间"
              end-placeholder="结束时间" />
          </el-form-item>
        </el-col>
        &nbsp;&nbsp;&nbsp;&nbsp;
        <el-col :span="4" style="height: 24px;">
          <el-form-item label="用户名">
            <el-input v-model="data.name" />
          </el-form-item>
        </el-col>
        &nbsp;&nbsp;&nbsp;&nbsp;
        <el-col :span="4" style="text-align: right;height: 24px;">
          <el-form-item label="客户端IP">
            <el-input v-model="data.client_ip" />
          </el-form-item>
        </el-col>
        <el-col :span="2" style="text-align: right;height: 24px;">
          <el-button type="primary" @click="searchLoginAudit">搜索</el-button>
        </el-col>
      </el-row>
      <el-row style="margin-top: 20px">
        <el-table :data="data.login_audit_list" style="width: 100%" :show-overflow-tooltip="true">
          <el-table-column sortable prop="id" label="ID" width="150"></el-table-column>
          <el-table-column sortable prop="name" label="用户名"></el-table-column>
          <el-table-column sortable prop="pwd" label="密码"></el-table-column>
          <el-table-column sortable prop="client_ip" label="客户端"></el-table-column>
          <el-table-column sortable prop="user_agent" label="userAgent"></el-table-column>
          <el-table-column sortable prop="err_msg" label="错误信息"></el-table-column>
          <el-table-column sortable prop="is_success" label="状态">
            <template #default="scope">
              <div v-if="scope.row.is_success === 'Y'">
                <el-tag type="success">成功</el-tag>
              </div>
              <div v-else>
                <el-tag type="danger">失败</el-tag>
              </div>
            </template>
          </el-table-column>
          <el-table-column sortable prop="occur_at" label="发生时间"></el-table-column>
        </el-table>

      </el-row>
      <el-row style="margin-top: 20px">
        <el-pagination v-model:current-page="currentPage" v-model:page-size="pageSize" :page-sizes="[10, 20, 50, 100]"
          :background="true" layout="total, sizes, prev, pager, next, jumper" :total="total"
          @size-change="searchLoginAudit" @current-change="searchLoginAudit" />
      </el-row>
    </el-card>
  </el-tab-pane>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from "vue";
import { ElMessage, dayjs } from "element-plus";
import axios from "axios";

const currentPage = ref(1);
const pageSize = ref(10);
const total = ref(0);

interface LoginAudit {
  id: number;
  name: string;
  pwd: string;
  client_ip: string;
  user_agent: string;
  err_msg: string;
  is_success: "Y" | "N";
  occur_at: string;
}


interface ResponseData {
  code: number;
  msg: string;
  count: number;
  data?: any
}


const time_range = ref<[Date, Date]>([
  new Date(2020, 1, 1, 0, 0, 0),
  new Date(2049, 12, 31, 0, 0, 0),
])

let data = reactive({
  login_audit_list: Array<LoginAudit>(),
  is_success: "N",
  client_ip: "",
  name: ""
});


/**
 * 获取数据
 */
function searchLoginAudit() {

  let body = {
    occur_begin: dayjs(time_range.value[0]).format("YYYY-MM-DD HH:mm:ss"),
    occur_end: dayjs(time_range.value[1]).format("YYYY-MM-DD HH:mm:ss"),
    name: data.name,
    is_success: data.is_success,
    client_ip: data.client_ip,
    offset: (currentPage.value * pageSize.value) - pageSize.value,
    limit: pageSize.value,
  };

  axios.post<ResponseData>(`/api/login_audit`, body)
    .then((ret) => {
      if (ret.data.code === 0) {
        data.login_audit_list = ret.data.data;
        total.value = ret.data.count;
      } else {
        ElMessage.error("获取登录审计信息错误");
      }
    })
}


onMounted(() => {
  searchLoginAudit();
})
</script>


<style scoped></style>