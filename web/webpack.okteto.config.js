var path = require("path");

var srcPath = path.join(__dirname, "src");
var distPath = path.join(__dirname, "dist");

module.exports = {
  mode: 'development',

  entry: [
    "./src/index.jsx",
  ],

  module: {
    rules: [
      {
        test: /\.jsx?$/,
        use: 'ts-loader',
        include: srcPath,
        enforce: "pre",
      },
      {
        test: /\.jsx?$/,
        use: "source-map-loader",
        include: srcPath,
        enforce: "pre",
      },
    ],
  },

  optimization: {
    moduleIds: "named",
  },

  output: {
    path: distPath,
    publicPath: "/",
    filename: "testgrid.js",
  },

  devtool: "eval-source-map",

  devServer: {
    port: 8080,
    host: "0.0.0.0",
    hot: "only",
    historyApiFallback: {
      verbose: true,
      rewrites: [
        {
          from: /^\/run\/.*$/,
          to: function() {
            return '/index.html';
          }
        }
      ]
    },
    allowedHosts: "all",
  },
};
