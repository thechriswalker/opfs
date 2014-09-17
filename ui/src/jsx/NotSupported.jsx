/** @jsx React.DOM */
var React = require("react");

module.exports = React.createClass({
  displayName: "NotSupported",
  render: function(){
    return <div className="not-supported alert alert-danger" role="alert">
      <p><strong><i className="fa fa-frown-o fa-3x"></i> Sorry</strong> I don't think this is going to work in your browser.</p>
    </div>;
  }
});