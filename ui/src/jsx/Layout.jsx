/** @jsx React.DOM */
var React = require("react"),
    Chrome = require("./Chrome"),
    NotFound = require("./NotFound"),
    NotSupported = require("./NotSupported");

module.exports = React.createClass({
  displayName: "Layout",
  render: function(){
    var hub = this.props.hub,
        name = this.props.children ? this.props.name : "404";

    return <div className="container-fluid">
      {this.props.children || <NotFound page={hub.get("url.path")} /> }
      <Chrome hub={hub} name={name} />
      {hub.get("browserWarning") && <NotSupported />}
    </div>;
  }
});