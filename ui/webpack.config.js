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
          include: [/\/src/, /\/node_modules\/@fortawesome\/fontawesome-free\/css/],
          test: /\.css$/,
          loader: ['style-loader', 'css-loader'],
        },
        {
          include: [/\/node_modules\/@fortawesome\/fontawesome-free\/webfonts/],
          test: /\.(eot|svg|ttf|woff|woff2)$/,
          loader: 'file-loader',
          options: {
            name: 'static/[name].[hash:8].[ext]',
          },
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
