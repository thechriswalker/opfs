"use strict";
var Navigator = require("react-simple-router").Navigator;

module.exports = {
  isSupported: function(window){
    // @TODO implement something sensisble here.
    return !!window.window && Navigator.historySupported();
  }
};