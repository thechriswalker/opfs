var React = require("react"),
    api = require("../../lib/api_urls"),
    PhotoGridPage = require("../PhotoGridPage");

module.exports = React.createClass({
  displayName: "TagsPage",
  render: function(){
    return PhotoGridPage({
      hub: this.props.hub,
      name: "tags",
      search: api.tags()
    });
  }
});