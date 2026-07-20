/** @type {import('next').NextConfig} */
const apiBaseURL = process.env.NEXT_PUBLIC_API_BASE || 'http://localhost:8080'

const nextConfig = {
  async rewrites() {
    return [
      {
        source: '/api/:path*',
        destination: `${apiBaseURL}/api/:path*`,
      },
    ]
  },
}

module.exports = nextConfig
