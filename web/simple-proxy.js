const http = require('http');
const fs = require('fs');
const path = require('path');
const url = require('url');

// MIME类型映射
const mimeTypes = {
    '.html': 'text/html',
    '.js': 'text/javascript',
    '.css': 'text/css',
    '.json': 'application/json',
    '.png': 'image/png',
    '.jpg': 'image/jpg',
    '.gif': 'image/gif',
    '.svg': 'image/svg+xml',
    '.wav': 'audio/wav',
    '.mp4': 'video/mp4',
    '.woff': 'application/font-woff',
    '.ttf': 'application/font-ttf',
    '.eot': 'application/vnd.ms-fontobject',
    '.otf': 'application/font-otf',
    '.wasm': 'application/wasm'
};

function getContentType(filePath) {
    const ext = path.extname(filePath).toLowerCase();
    return mimeTypes[ext] || 'application/octet-stream';
}

function serveStaticFile(res, filePath) {
    fs.readFile(filePath, (err, content) => {
        if (err) {
            if (err.code === 'ENOENT') {
                res.writeHead(404, { 'Content-Type': 'text/plain' });
                res.end('File not found');
            } else {
                res.writeHead(500, { 'Content-Type': 'text/plain' });
                res.end('Server error');
            }
        } else {
            res.writeHead(200, { 
                'Content-Type': getContentType(filePath),
                'Access-Control-Allow-Origin': '*',
                'Access-Control-Allow-Methods': 'GET, POST, PUT, DELETE, OPTIONS',
                'Access-Control-Allow-Headers': 'Content-Type, Authorization'
            });
            res.end(content);
        }
    });
}

// 代理请求到后端API
function proxyRequest(req, res) {
    const targetUrl = `http://127.0.0.1:8100${req.url.replace('/api', '')}`;
    
    console.log(`代理请求: ${req.method} ${req.url}`);
    console.log(`转发到: 127.0.0.1:8100${req.url.replace('/api', '')}`);
    
    // 收集请求体数据
    let body = '';
    req.on('data', chunk => {
        body += chunk.toString();
    });
    
    req.on('end', () => {
        console.log(`请求体: ${body}`);
        
        const options = {
            hostname: '127.0.0.1',
            port: 8100,
            path: req.url.replace('/api', ''),
            method: req.method,
            headers: {
                ...req.headers,
                host: '127.0.0.1:8100'
            }
        };

        const proxyReq = http.request(options, (proxyRes) => {
            console.log(`后端响应状态: ${proxyRes.statusCode}`);
            
            // 设置CORS头
            res.setHeader('Access-Control-Allow-Origin', '*');
            res.setHeader('Access-Control-Allow-Methods', 'GET, POST, PUT, DELETE, OPTIONS');
            res.setHeader('Access-Control-Allow-Headers', 'Content-Type, Authorization');
            
            // 转发响应头
            Object.keys(proxyRes.headers).forEach(key => {
                res.setHeader(key, proxyRes.headers[key]);
            });
            
            res.statusCode = proxyRes.statusCode;
            
            // 收集响应体数据
            let responseBody = '';
            proxyRes.on('data', chunk => {
                responseBody += chunk.toString();
            });
            
            proxyRes.on('end', () => {
                console.log(`响应体: ${responseBody}`);
                res.end(responseBody);
            });
        });

        proxyReq.on('error', (err) => {
            console.error('代理请求错误:', err);
            res.statusCode = 500;
            res.end('代理请求失败');
        });

        if (body) {
            proxyReq.write(body);
        }
        proxyReq.end();
    });
}

const server = http.createServer((req, res) => {
    const parsedUrl = url.parse(req.url, true);
    const pathname = parsedUrl.pathname;

    // 处理OPTIONS请求
    if (req.method === 'OPTIONS') {
        res.writeHead(200, {
            'Access-Control-Allow-Origin': '*',
            'Access-Control-Allow-Methods': 'GET, POST, PUT, DELETE, OPTIONS',
            'Access-Control-Allow-Headers': 'Content-Type, Authorization'
        });
        res.end();
        return;
    }

    // API代理
    if (pathname.startsWith('/api/')) {
        proxyRequest(req, res);
        return;
    }

    // 静态文件服务
    let filePath = path.join(__dirname, pathname === '/' ? 'release-list.html' : pathname);
    
    // 检查文件是否存在
    fs.access(filePath, fs.constants.F_OK, (err) => {
        if (err) {
            // 如果文件不存在，尝试添加.html扩展名
            if (!path.extname(filePath)) {
                filePath += '.html';
            }
        }
        serveStaticFile(res, filePath);
    });
});

const PORT = 3000;
server.listen(PORT, () => {
    console.log(`代理服务器运行在 http://localhost:${PORT}`);
    console.log(`前端页面: http://localhost:${PORT}/release-list.html`);
    console.log(`API代理: http://localhost:${PORT}/api/* -> http://127.0.0.1:8100/*`);
});