const HtmlWebpackPlugin = require('html-webpack-plugin')
const webpack = require('webpack')

module.exports = (env, argv) => {
  const production = argv.mode === 'production'
  const config = {
    output: {
      filename: 'static/[name].[hash:8].js',
      publicPath: '/',
    },
    module: {
      rules: [
        {
          test: /\.js$/,
          exclude: [/elm-stuff/, /node_modules/],
          loader: 'babel-loader',
          query: {
            presets: [
              '@babel/preset-env',
            ],
          },
        },
        {
          test: /\.elm$/,
          exclude: [/elm-stuff/, /node_modules/],
          use: [
            { loader: 'elm-hot-webpack-loader' },
            {
              loader: 'elm-webpack-loader',
              options: {
                debug: !production,
                optimize: production,
              },
            },
          ],
        },
        {
          test: /\.css$/,
          exclude: [/node_modules/],
          loader: ['style-loader', 'css-loader'],
        },
      ]
    },
    plugins: [
      new HtmlWebpackPlugin({
        template: 'public/index.html',
        favicon: 'public/favicon.png',
      }),
    ],
    devServer: {
      inline: true,
      historyApiFallback: true,
      stats: { colors: true },
      overlay: true,
      open: true,
      proxy: [{
        context: ['/api', '/debug', '/serve'],
        target: 'http://localhost:9000',
        ws: true,
      }],
      watchOptions: {
        ignored: /node_modules/,
      },
    },
  }
  if (argv.hot) {
    config.plugins.push(new webpack.HotModuleReplacementPlugin())
  }
  return config
}
