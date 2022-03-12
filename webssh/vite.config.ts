import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import AutoImport from "unplugin-auto-import/vite";
import Components from "unplugin-vue-components/vite";
import { ElementPlusResolver } from "unplugin-vue-components/resolvers";

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [
    vue(),
    AutoImport({
      resolvers: [ElementPlusResolver()],
    }),
    Components({
      resolvers: [ElementPlusResolver()],
    }),
  ],

  // 公共路径默认为 / ,我们指定为当前项目根目录./
  base: "./",

  server: {
    // 监听地址
    host: "0.0.0.0",

    // 监听端口
    port: 3000,

    //boolean | string 设置服务启动时是否自动打开浏览器,当此值为字符串时,会被当作 URL 的路径名
    open: true,

    //为开发服务器配置 CORS，配置为允许跨域
    cors: true,

    // 配置代理
    proxy: {
      "/api": {
        // 代理目标地址
        target: "http://127.0.0.1:8899",

        // websocket 
        ws: true,

        // 支持https
        secure: false,

        // 是否允许不同源
        changeOrigin: true,

      },
    },
  },
  
  build:{
    // 指定打包路径，默认为项目根目录下的 dist 目录
    outDir: 'dist',

      terserOptions: {
          compress: {

              // 防止 Infinity 被压缩成 1/0
              keep_infinity: true,

              // 生产环境去除 console
              drop_console: true,

              // 生产环境去除 debugger
              drop_debugger: true   
          },
      },

      // chunk 大小警告的限制（以 kbs 为单位）
      chunkSizeWarningLimit: 1500
  }
});
