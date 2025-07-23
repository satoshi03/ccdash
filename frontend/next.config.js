/** @type {import('next').NextConfig} */
const nextConfig = {
  output: 'standalone',
  experimental: {
    serverComponentsExternalPackages: []
  },
  // Disable strict mode for production builds to avoid double rendering issues
  reactStrictMode: false,
  // Optimize for production
  swcMinify: true,
  // Configure static optimization
  trailingSlash: false,
  // Configure image optimization
  images: {
    unoptimized: true
  }
}

module.exports = nextConfig