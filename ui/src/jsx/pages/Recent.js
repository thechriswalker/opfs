var React = require("react"),
    api = require("../../lib/api_urls"),
    PhotoGridPage = require("../PhotoGridPage");

module.exports = React.createClass({
  displayName: "RecentPage",
  render: function(){
    return PhotoGridPage({
      hub: this.props.hub,
      name: "recent",
      search: api.recent()
    });
  }
});