/** @jsx React.DOM */
var React = require("react");

module.exports = React.createClass({
  displayName: "NotFound",
  render: function(){
    return <div>
      <div className="jumbo">
        <i className="fa fa-3x fa-exclamation-triangle"></i>
      </div>
      <div className="jumbo-subtext">
        <p>Not Found: <kbd>{this.props.page}</kbd></p>
        <p>{"Sorry, the page you requested doesn't exist."}</p>
      </div>
    </div>;
  }
});