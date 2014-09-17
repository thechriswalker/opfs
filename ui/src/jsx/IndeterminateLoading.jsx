/** @jsx React.DOM */
var React = require("react");

module.exports = React.createClass({
  displayName: "IndeterminateLoading",
  render: function(){
    return <div className="jumbo">
      <span className="fa-stack">
        <i className="fa fa-circle-o-notch fa-spin fa-stack-2x fa-fw"></i>
        <i className="fa fa-camera fa-stack-1x fa-fw"></i>
      </span>
    </div>;
  }
});