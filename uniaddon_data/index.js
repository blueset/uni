var uni = require('node-gyp-build')(__dirname)
// JS call API
module.exports = { 
  exec : uni.exec
};
