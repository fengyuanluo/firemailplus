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
};

export default nextConfig;
