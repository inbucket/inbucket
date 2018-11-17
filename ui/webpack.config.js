const HtmlWebpackPlugin = require('html-webpack-plugin')
const webpack = require('webpack')

module.exports = {
  mode: 'development',
  output: {
    filename: 'static/[name].js',
    publicPath: '/',
  },
  module: {
    rules: [
      {
        test: /\.elm$/,
        exclude: [/elm-stuff/, /node_modules/],
        use: [
          { loader: 'elm-hot-webpack-loader' },
          {
            loader: 'elm-webpack-loader',
            options: {
              debug: true
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
    new webpack.HotModuleReplacementPlugin(),
  ],
  devServer: {
    inline: true,
    hot: true,
    stats: { colors: true },
    proxy: [{
      context: ['/api', '/debug', '/serve'],
      target: 'http://localhost:9000',
      ws: true,
    }],
  },
};
