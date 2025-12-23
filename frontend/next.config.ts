import type { NextConfig } from 'next';

const nextConfig: NextConfig = {
  // 启用standalone模式以优化Docker镜像大小
  output: 'standalone',

  // 压缩配置
  compress: true,

  // 静态文件优化
  assetPrefix: process.env.NODE_ENV === 'production' ? '' : '',

  // 图片优化
  images: {
    unoptimized: false,
  },

  async rewrites() {
    // 将前端 /api/* 代理到后端 8080 端口，便于单容器部署时同域调用
    return [
      {
        source: '/api/:path*',
        destination: 'http://localhost:8080/api/:path*',
      },
    ];
  },

  // 构建阶段忽略 ESLint 报告（大量 legacy any 需要后续逐步治理）
  eslint: {
    ignoreDuringBuilds: true,
  },

  // Next.js 15 将 experimental.serverComponentsExternalPackages 改为顶层 serverExternalPackages
  serverExternalPackages: [],
};

export default nextConfig;
