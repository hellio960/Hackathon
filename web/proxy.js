const http = require('http');
const httpProxy = require('http-proxy-middleware');
const express = require('express');
const path = require('path');

const app = express();

// 静态文件服务
app.use(express.static(__dirname));

// API代理
const apiProxy = httpProxy.createProxyMiddleware({
    target: 'http://127.0.0.1:8100',
    changeOrigin: true,
    pathRewrite: {
        '^/api': '' // 将 /api 前缀重写为空
    },
    onProxyRes: function (proxyRes, req, res) {
        // 添加CORS头部
        proxyRes.headers['Access-Control-Allow-Origin'] = '*';
        proxyRes.headers['Access-Control-Allow-Methods'] = 'GET,PUT,POST,DELETE,OPTIONS';
        proxyRes.headers['Access-Control-Allow-Headers'] = 'Content-Type, Authorization, Content-Length, X-Requested-With';
    }
});

app.use('/api', apiProxy);

// 处理OPTIONS请求
app.options('*', (req, res) => {
    res.header('Access-Control-Allow-Origin', '*');
    res.header('Access-Control-Allow-Methods', 'GET,PUT,POST,DELETE,OPTIONS');
    res.header('Access-Control-Allow-Headers', 'Content-Type, Authorization, Content-Length, X-Requested-With');
    res.sendStatus(200);
});

const PORT = 3000;
app.listen(PORT, () => {
    console.log(`代理服务器运行在 http://localhost:${PORT}`);
    console.log(`前端页面: http://localhost:${PORT}/release-list.html`);
    console.log(`API代理: http://localhost:${PORT}/api/* -> http://127.0.0.1:8100/*`);
});