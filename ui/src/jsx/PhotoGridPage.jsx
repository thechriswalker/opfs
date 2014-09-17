/** @jsx React.DOM */
var React = require("react"),
    Layout = require("./Layout"),
    IndeterminateLoading = require("./IndeterminateLoading"),
    PhotoGrid = require("./PhotoGrid"),
    Slides = require("./Slides"),
    Paginator = require("./Paginator");

module.exports = React.createClass({
  displayName: "PhotoGridPage",
  render: function(){
    var hub = this.props.hub,
        name = this.props.name,
        items = hub.get("items");

    if(!Array.isArray(items)){
      //OK, we don't have anything. indeterminate loading.
      return <Layout hub={hub} name={name}>
        <IndeterminateLoading />
      </Layout>;
    }

    //all is good or a fixed error.
    return <Layout hub={hub} name={name}>
      <Slides hub={hub} />
      <PhotoGrid hub={hub} items={items} />
      <Paginator hub={hub} />
    </Layout>;
  },
  loadInitial: function(){
    if(this.props.search){
      this.props.hub.dispatch("search", this.props.search);
    }
  },
  componentDidMount: function(){
    this.loadInitial();
  },
  componentWillUpdate: function(nextProps){
    if(this.props.search !== nextProps.search){
      this.loadInitial();
    }
  }
});