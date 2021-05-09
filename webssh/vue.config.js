
module.exports = {
    
    devServer: {
        host: '0.0.0.0',
        port: 8080,
        proxy: {
            '/api': {
                target: 'http://127.0.0.1:8899',
                ws: true,
                secure: false,
                changeOrigin: true,
                pathRewrite: {
                    '^/api': '/api',//重写,
                }
            },
        }
    },
    publicPath: process.env.NODE_ENV === 'production' ? '/' : '/'
};

//============================================
// 