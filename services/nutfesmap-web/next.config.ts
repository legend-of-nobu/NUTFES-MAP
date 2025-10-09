/** @type {import('next').NextConfig} */
const nextConfig = {
  eslint: {
    // ESLintエラーで build を止めない
    ignoreDuringBuilds: true,
  },
  typescript: {
    // TypeScriptエラーで build を止めない
    ignoreBuildErrors: true,
  },
};

module.exports = nextConfig;
