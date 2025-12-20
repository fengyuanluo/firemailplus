import type { NextConfig } from 'next';

const nextConfig: NextConfig = {
  // 启用standalone模式以优化Docker镜像大小
  output: 'standalone',

  // 优化配置
  experimental: {
    // 启用服务器组件优化
    serverComponentsExternalPackages: [],
  },

  // 压缩配置
  compress: true,

  // 静态文件优化
  assetPrefix: process.env.NODE_ENV === 'production' ? '' : '',

  // 图片优化
  images: {
    unoptimized: false,
  },

  async rewrites() {
    // 将前端 /api/* 代理到后端 3001 端口，便于单容器部署时同域调用
    return [
      {
        source: '/api/:path*',
        destination: 'http://localhost:3001/api/:path*',
      },
    ];
  },
};

export default nextConfig;
