var React = require("react"),
    api = require("../../lib/api_urls"),
    PhotoGridPage = require("../PhotoGridPage");

module.exports = React.createClass({
  displayName: "SearchPage",
  render: function(){
    //get the search query from the "q" query param
    var search = this.props.hub.get("url").query.q;

    //convert back to an object.
    try{
      search = JSON.parse(search);
    } catch(e){
      search = false;
    }

    return PhotoGridPage({
      hub: this.props.hub,
      name: "search",
      search: search && api.search(search)
    });
  }
});