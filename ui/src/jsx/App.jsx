/** @jsx React.DOM */
var React = require("react"),
    Router = require("react-simple-router").Component,
    SearchPage = require("./pages/Search"),
    TagsPage = require("./pages/Tags"),
    RecentPage = require("./pages/Recent"),
    Layout = require("./Layout");

var App = module.exports = React.createClass({
  displayName: "App",
  statics: {
    routes: [
      { pattern: /^\/(index\.html|)$/, handler: RecentPage },
      { pattern: /^\/search\/?$/, handler: SearchPage },
      { pattern: /^\/tags\/?$/, handler: TagsPage }
    ]
  },
  render: function(){
    var hub = this.props.hub, path = hub.get("url.path");
    return Router({path:path, routes:App.routes, notFound:Layout, hub:hub});
  }
});