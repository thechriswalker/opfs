/** @jsx React.DOM */
var React = require("react");
/*
  Chrome is the common UI for the app. e.g. the menus and titlebar.
  It could also render modals on demand. It should be last in the Layout,
  so it can easily render on top of everything else. It's going to be using
  absolute positioning anyway.

  I think it is going to be the base for the majority of the functionality that
  is not simply viewing a grid, or a slideshow. (e.g. search ui, add tag/album
  forms/dialogs, etc...)
*/

module.exports = React.createClass({
  displayName: "Chrome",
  render: function(){
    var pageName = this.props.name || "";

    return <nav className="titlebar navbar navbar-default">
      <div className="container-fluid">
        <div className="navbar-header">
          <a className="navbar-brand" href="#"><i className="fa fa-lg fa-camera main-logo"></i></a>
          <div className="navbar-brand">{"opfs://"}<span className="breadcrumbs">{pageName}</span></div>
        </div>
        <ul className="nav navbar-nav navbar-right">
          <li><a href="/search"><i className="fa fa-lg fa-search"></i></a></li>
          <li><a href="/tags"><i className="fa fa-lg fa-tags"></i></a></li>
          <li><a href="/albums"><i className="fa fa-lg fa-folder"></i></a></li>
          <li><a href="/"><i className="fa fa-lg fa-clock-o"></i></a></li>
        </ul>
      </div>
    </nav>;
  }
});