/** @type {import('next').NextConfig} */
const nextConfig = {
  output: 'standalone',
  serverExternalPackages: [],
  // Disable strict mode for production builds to avoid double rendering issues
  reactStrictMode: false,
  // Configure static optimization
  trailingSlash: false,
  // Configure image optimization
  images: {
    unoptimized: true
  }
}

module.exports = nextConfig