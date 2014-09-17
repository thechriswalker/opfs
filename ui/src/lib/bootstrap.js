"use strict";
/* jshint browser:true */
var React = require("react"),
    Navigator = require("react-simple-router").Navigator,
    App = require("../jsx/App"),
    Hub = require("./Hub"),
    url = require("fast-url-parser"),
    qs = require("querystring"),
    support = require("./browser-support"),
    logic = require("./logic");

//we only want the side effect of require'ing the Stores.
require("./ItemStore");
require("./SelectionStore");

function parseURL(given){
  var parsed = url.parse(given);
  return {
    full: given,
    path: parsed.pathname,
    query: parsed.query?qs.parse(parsed.query):{},
    fragment: parsed.hash && parsed.hash.substring(1), //remove the "#"
  };
}


module.exports = function(initialState, options){

  if(typeof initialState.state.url === "string"){
    initialState.state.url = parseURL(initialState.state.url);
  }

  var hub = new Hub(),
    props = { hub: hub };

  //inflate the hub with the initial data.
  hub.inflate(initialState);

  //are we doing a static render?
  if(options.stringify){
    var suffix = options.renderStatic ? "StaticMarkup" : "String";
    return React["renderComponentTo"+suffix](App(props));
  }

  //add logic to the hub
  logic.Init(hub);

  //create the app.
  var el = options.element || global.document.createElement("DIV"),
      app = React.renderComponent(App(props), el);

  //re-render whenever data changes in state.
  props.hub.register(function(){
    app.forceUpdate();
  });

  //start the Navigator, capturing links
  Navigator.onNavigate(function(){
    //this is after the url HAS changed in the address bar.
    //we dispatch here to allow the logic to do what it needs to
    hub.dispatch("route", parseURL(window.location.href));
  });

  //we assume we are in a browser (or close enough) environment.
  //let's do some feature detection to determine whether this app
  //will work...
  if(!support.isSupported(global)){
    //we set this, to render a permanent browser warning.
    hub.set("browserWarning", true);
  }

  //now we call route with our current route. This cause the app to re-render,
  //and add a slideshow if it needs, if nothing changes, nothing changes.
  hub.dispatch("route", initialState.state.url);

  //return this so we can introspect it.
  return {app: app, hub: hub};
};