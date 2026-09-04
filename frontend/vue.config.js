const { defineConfig } = require('@vue/cli-service')
module.exports = defineConfig({
  transpileDependencies: true,
  chainWebpack: (config) => {
    config.plugin('html').tap((args) => {
      args[0].title = 'Calciotto'
      return args
    })
  },
  devServer: {
    // webpack-dev-server's default 'auto' rejects a request whose Host header
    // it doesn't recognize (a DNS-rebinding protection) — which blocks
    // reaching the dev server by its LAN IP (http://192.168.1.7:4000) from
    // another device, even though the container already listens on 0.0.0.0.
    // 'all' is fine here: this only ever runs against the stubbed/local dev
    // backend, never production.
    allowedHosts: 'all'
  }
})
