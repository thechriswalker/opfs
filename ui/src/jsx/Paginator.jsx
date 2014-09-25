/** @jsx React.DOM */
var React = require("react");

module.exports = React.createClass({
  displayName: "Paginator",
  render: function(){
    //a next button if next available. loader if loading, nothing if nothing.
    var next = this.props.hub.get("paging.next");
    if(next){
      return <div className="paginator"><button className="btn" onClick={this.triggerNext}>want more?</button></div>;
    }
    return null;
  },
  triggerNext: function(){
    this.props.hub.dispatch("next");
  }
});